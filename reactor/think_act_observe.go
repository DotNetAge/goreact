package reactor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// l1RoutingResult holds the parsed L1 routing decision.
type l1RoutingResult struct {
	Path          string   // "tool" | "skill" | "answer" | "delegate"
	Target        string   // selected skill name or primary tool name, or agent name (for "delegate")
	SelectedTools []string // tool names (for "tool" path)
	Answer        string   // final answer text (for "answer" path)
	Reasoning     string
}

// Think asks the LLM to decide the next action based on the current context.
// Implements PROGRESSIVE DISCLOSURE two-phase thinking (L1 -> L2):
//
//	Phase 1 (L1 Routing):  All tools loaded as MINIMAL schema (~50 tokens/tool)
//	                        + all skills metadata in SystemPrompt.
//	                        LLM routes to: "tool" | "skill" | "answer"
//	                        Outputs selected target(s) by name.
//	Phase 2 (L2 Planning):  Selected tools upgraded to FULL schema.
//	                        If skill routed: skill L2 instructions injected.
//	                        LLM generates actual action with proper parameters.
//
// Verified by Experiment 1: Qwen accepts empty-properties NativeTools.
// Verified by Experiment 2: LLM can semantically match capabilities to tool names.
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
	estimateFn := r.tokenEstimator.Estimate
	totalTokens := 0

	// --- Load ALL tools (L1 minimal + L2 full ready) ---
	allToolInfos := core.ToToolInfos(r.toolRegistry.All())
	minimalLLMTools := ToolInfosToMinimalLLMTools(allToolInfos)

	// --- Load applicable skills ---
	skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)
	skillsSection := BuildSkillsSystemPrompt(skills)

	// --- Build agents section for progressive disclosure (delegate routing) ---
	var agentsSection string
	if r.orchestrator != nil {
		agentsSection = r.buildAgentsSection()
	}

	var memoryRecords []core.MemoryRecord
	if r.memory != nil {
		records, err := r.memory.Retrieve(
			ctx.Ctx(), ctx.Input,
			core.WithMemoryTypes(core.MemoryTypeLongTerm, core.MemoryTypeUser, core.MemoryTypeExperience),
			core.WithMemoryLimit(3),
		)
		if err != nil {
			ctx.EmitEvent(core.Error, fmt.Sprintf("memory retrieval failed (non-fatal): %v", err))
		} else {
			memoryRecords = records
		}
	}

	accountTokens := func(content string) {
		if r.contextWindow != nil && content != "" {
			r.contextWindow.AddTokens(int64(estimateFn(content)))
		}
	}

	// --- Pre-L1 accounting (FIX P1-#4: complete all layers) ---
	accountTokens(skillsSection)
	accountTokens(agentsSection)         // Agent metadata for progressive disclosure
	accountTokens(r.config.SystemPrompt) // FIX: was missing — base identity prompt
	minimalTokens := EstimateTokensForTools(minimalLLMTools, estimateFn)
	if minimalTokens > 0 && r.contextWindow != nil {
		r.contextWindow.AddTokens(minimalTokens)
	}

	// ====================================================================
	// PHASE 1 (L1 Routing): Minimal tools + skills + agents -> route decision
	// ====================================================================
	var agentNamesForL1 []string
	if r.orchestrator != nil {
		agentNamesForL1 = r.orchestrator.ListAgents()
	}
	l1Prompt := buildL1RoutingPrompt(skills, agentNamesForL1)
	accountTokens(l1Prompt)  // FIX: was missing — phase instruction
	accountTokens(ctx.Input) // FIX: was missing — user message

	// FIX: Account history tokens for ContextWindow (replicates buildLLMBuilder trim logic)
	maxTurns := r.maxHistoryTurns()
	historyForL1 := ctx.ConversationHistory
	if maxTurns > 0 && len(historyForL1) > maxTurns {
		historyForL1 = historyForL1[len(historyForL1)-maxTurns:]
	}
	for _, msg := range historyForL1 {
		accountTokens(msg.Content)
	}

	l1Content, l1Tokens, err := r.callLLMStream(
		ctx, l1Prompt, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), minimalLLMTools, skillsSection,
	)
	if err != nil {
		return totalTokens + l1Tokens, fmt.Errorf("think L1 routing failed: %w", err)
	}
	totalTokens += l1Tokens

	// FIX(P1-#4): Add estimated input tokens for L1 call to ContextWindow and return value
	l1InputTokens := r.estimateInputTokens(l1Prompt, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), minimalLLMTools, skillsSection)
	totalTokens += l1InputTokens

	l1Result, err := parseL1RoutingResponse(l1Content)
	if err != nil {
		return totalTokens, fmt.Errorf("think L1 parse failed: %w", err)
	}
	logger.Info("L1 Route", "path", l1Result.Path, "target", l1Result.Target,
		"tools", l1Result.SelectedTools, "reasoning", truncate(l1Result.Reasoning, 200))

	var actCtx *ActivatedSkillContext
	var l1SelectedToolNames []string

	switch l1Result.Path {
	case "answer":
		ctx.LastThought = &Thought{
			Decision:    DecisionAnswer,
			Reasoning:   l1Result.Reasoning,
			FinalAnswer: l1Result.Answer,
			IsFinal:     true,
		}
		return totalTokens, nil

	case "delegate":
		// L1 delegate route: set decision directly, skip L2
		ctx.LastThought = &Thought{
			Decision:       DecisionDelegate,
			Reasoning:      l1Result.Reasoning,
			DelegateTarget: l1Result.Target,
			DelegatePrompt: l1Result.Answer, // reuse Answer field for task prompt
		}
		if ctx.LastThought.DelegatePrompt == "" {
			ctx.LastThought.DelegatePrompt = ctx.Input
		}
		return totalTokens, nil

	case "tool":
		l1SelectedToolNames = l1Result.SelectedTools
		if len(l1SelectedToolNames) == 0 && l1Result.Target != "" {
			l1SelectedToolNames = []string{l1Result.Target}
		}

	case "skill":
		if l1Result.Target == "" {
			l1Result.Path = "tool"
			break
		}
		actCtx, err = r.ActivateSkill(l1Result.Target, allToolInfos)
		if err != nil {
			logger.Info("Skill activation failed, falling back to direct tool mode", "error", err)
			actCtx = nil
			l1Result.Path = "tool"
		} else if actCtx != nil {
			accountTokens(actCtx.Instructions)
		}

	default:
		ctx.LastThought = &Thought{
			Decision:    DecisionAnswer,
			Reasoning:   l1Result.Reasoning,
			FinalAnswer: "I'm unsure how to proceed. Could you clarify your request?",
			IsFinal:     true,
		}
		return totalTokens, nil
	}

	// ====================================================================
	// PHASE 2 (L2 Planning): Full-schema tools + optional skill instructions
	// ====================================================================
	var l2FullSchemaTools []gochatcore.Tool
	if len(l1SelectedToolNames) > 0 && actCtx == nil {
		l2FullSchemaTools = UpgradeToolsToFullSchema(l1SelectedToolNames, allToolInfos)
		if len(l2FullSchemaTools) == 0 {
			logger.Info("L1-selected tools not found in registry, falling back to all full-schema tools")
			l2FullSchemaTools = ToolInfosToLLMTools(allToolInfos)
		}
	} else {
		l2FullSchemaTools = ToolInfosToLLMTools(allToolInfos)
	}

	// FIX(P1-#4/#7): Account incremental tool schema cost (L2 full minus L2 minimal overlap).
	// Use EstimateTokensForTools delta to avoid double-counting the base tool identity cost.
	fullSchemaTokens := EstimateTokensForTools(l2FullSchemaTools, estimateFn)
	if fullSchemaTokens > minimalTokens && r.contextWindow != nil {
		r.contextWindow.AddTokens(fullSchemaTokens - minimalTokens) // Only incremental cost
	} else if fullSchemaTokens > 0 && r.contextWindow != nil {
		r.contextWindow.AddTokens(fullSchemaTokens)
	}

	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, memoryRecords, actCtx, r.intentRegistry, agentsSection)
	accountTokens(instructions)

	// FIX(P1-#4): Re-account per-call inputs for L2 (sent again as separate API call)
	accountTokens(ctx.Input) // User message sent again
	for _, msg := range historyForL1 {
		accountTokens(msg.Content) // History sent again
	}

	r.checkSlide(ctx.Ctx())

	content, tokens, err := r.callLLMStream(ctx, instructions, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), l2FullSchemaTools, skillsSection)
	if err != nil {
		return totalTokens + tokens, fmt.Errorf("think L2 planning failed: %w", err)
	}
	totalTokens += tokens

	// FIX(P1-#4): Add estimated input tokens for L2 call
	l2InputTokens := r.estimateInputTokens(instructions, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), l2FullSchemaTools, skillsSection)
	totalTokens += l2InputTokens

	thought, err := ParseThinkResponse(content)
	if err != nil {
		return totalTokens + tokens, fmt.Errorf("think L2 parse failed: %w", err)
	}

	if actCtx != nil && thought.SelectedSkill == "" {
		thought.SelectedSkill = actCtx.Skill.Name
	}

	ctx.LastThought = thought
	return totalTokens, nil
}

// buildL1RoutingPrompt constructs the Phase 1 routing prompt for L1 progressive disclosure.
// The model sees ALL tools with minimal schema, ALL skill metadata, AND available agents, then routes.
func buildL1RoutingPrompt(skills []*core.Skill, agentNames []string) string {
	hasSkills := len(skills) > 0
	hasAgents := len(agentNames) > 0

	base := "Analyze the user's input and determine the best way to respond.\n" +
		"\n" +
		"You see all available tools (with names and descriptions only) and any active capabilities below.\n"

	if hasAgents {
		base += "\n" + "Available agents for delegation: " + strings.Join(agentNames, ", ") + "\n"
		base += "When a task matches an agent's specialty, prefer delegating to it over using tools directly.\n"
	}

	base += "\n" +
		"Decide which path to take:\n" +
		"- **\"tool\"**: The user's request can be fulfilled by directly invoking one or more tools. Set 'selected_tools' to the tool name(s) you need.\n" +
		"- **\"skill\"**: The request requires specialized domain expertise. Choose the most applicable skill from <available_capabilities>. Set 'target' to the skill name.\n"

	if hasAgents {
		base += "- **\"delegate\"**: The task should be delegated to a specialized agent. Set 'target' to the exact agent name from the available agents list.\n"
	}

	base += "- **\"answer\"**: This is a simple question, chitchat, or knowledge query that needs no tools or agents. Provide the answer directly.\n" +
		"\n" +
		"Respond with JSON only:\n" +
		"{\"path\":\"tool|skill|answer|delegate\",\"target\":\"<skill_or_tool_or_agent_name>\",\"selected_tools\":[\"tool_name1\",...],\"answer\":\"<if answer path>\",\"reasoning\":\"<brief explanation>\"}\n" +
		"\n" +
		"CRITICAL rules:\n" +
		"- For \"tool\" path: you MUST populate selected_tools with specific tool names from the available set\n" +
		"- For \"skill\" path: set target to the EXACT skill name from available_capabilities\n"

	if hasAgents {
		base += "- For \"delegate\" path: set target to the EXACT agent name from the available agents list\n"
	}

	base += "- Select the MINIMUM set of tools sufficient for the task\n" +
		"- When unsure between tool and skill, prefer \"skill\" for complex multi-step tasks\n"

	if hasAgents {
		base += "- When an available agent's role clearly matches the task, prefer \"delegate\" over \"tool\"\n"
	}
	if hasSkills {
		return base + "\n\nAvailable capabilities (skills) are listed in the system prompt above — reference them by exact name."
	}
	return base
}

// buildAgentsSection constructs the <agents> progressive disclosure block
// from the Orchestrator's agent registry. Each agent is listed with its name,
// role, and description so the LLM can make informed delegate routing decisions.
func (r *Reactor) buildAgentsSection() string {
	agentNames := r.orchestrator.ListAgents()
	if len(agentNames) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, name := range agentNames {
		info := r.orchestrator.AgentInfo(name)
		if info == nil {
			continue
		}
		fmt.Fprintf(&sb, "- name: %q\n  role: %q\n", name, info.Role)
		if info.Description != "" {
			fmt.Fprintf(&sb, "  description: %s\n", info.Description)
		}
		if info.Model != "" {
			fmt.Fprintf(&sb, "  model: %s\n", info.Model)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// parseL1RoutingResponse extracts the L1 routing result from LLM response JSON.
func parseL1RoutingResponse(content string) (*l1RoutingResult, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		} else {
			content = stripJSONWrappers(content)
		}
	}
	var result l1RoutingResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse L1 routing JSON: %w\nraw: %s", err, content)
	}
	result.Path = strings.ToLower(strings.TrimSpace(result.Path))
	if result.Path == "" {
		result.Path = "answer"
	}
	return &result, nil
}

// Act executes the decision from the Think phase.
func (r *Reactor) Act(ctx *ReactContext) error {
	thought := ctx.LastThought
	if thought == nil {
		return fmt.Errorf("act called without a thought")
	}

	start := time.Now()
	action := Action{
		Timestamp: start,
	}

	switch thought.Decision {
	case DecisionAnswer:
		action.Type = ActionTypeAnswer
		action.Result = thought.FinalAnswer
		if action.Result == "" {
			action.Result = thought.Reasoning
		}

	case DecisionClarify:
		action.Type = ActionTypeClarify
		question := thought.ClarificationQuestion
		if question == "" {
			question = "Could you provide more details so I can better assist you?"
		}
		action.Result = question

	case DecisionDelegate:
		action.Type = ActionTypeToolCall
		action.Target = "subagent_task" // virtual tool name for delegation
		action.Params = map[string]any{
			"agent_name": thought.DelegateTarget,
			"prompt":     thought.DelegatePrompt,
		}
		// Execute delegation via orchestrator if available
		if r.orchestrator != nil && thought.DelegateTarget != "" {
			delegateResult, delegateErr := r.orchestrator.DelegateTo(
				ctx.Ctx(), thought.DelegateTarget, thought.DelegatePrompt, "", nil,
			)
			if delegateErr != nil {
				action.Error = delegateErr
				action.ErrorMsg = delegateErr.Error()
			} else {
				action.Result = fmt.Sprintf("Task delegated to agent %q (task_id: %s). Use subagent_result to retrieve the result.", thought.DelegateTarget, delegateResult.TaskID)
				action.Params["task_id"] = delegateResult.TaskID
			}
		} else if r.orchestrator == nil {
			action.Error = fmt.Errorf("no orchestrator configured for delegation")
			action.ErrorMsg = action.Error.Error()
		}

	case DecisionAct:
		action.Type = ActionTypeToolCall
		action.Target = thought.ActionTarget
		action.Params = thought.ActionParams

		if action.Target == "" {
			action.Type = ActionTypeAnswer
			action.Result = thought.FinalAnswer
			if action.Result == "" {
				action.Result = "Sorry, I cannot determine which tool to use for your request."
			}
			break
		}

		execResult, execErr := r.toolExecutor.Execute(ctx.Ctx(), action.Target, action.Params)
		if execErr != nil {
			action.Error = execErr
			action.ErrorMsg = execErr.Error()
		} else if execResult.Interaction != nil {
			answer, interactErr := r.interactionHandler.HandleInteraction(ctx.Ctx(), execResult.Interaction)
			if interactErr != nil {
				action.Error = interactErr
				action.ErrorMsg = interactErr.Error()
			} else {
				action.Result = answer
			}
		} else {
			action.Result = execResult.Result
		}
		if execResult != nil {
			action.Duration = execResult.Duration
		}

	default:
		action.Type = ActionTypeAnswer
		action.Result = thought.FinalAnswer
		if action.Result == "" {
			action.Result = thought.Reasoning
		}
	}

	ctx.LastAction = &action
	return nil
}

// Observe evaluates the result of the Act phase.
func (r *Reactor) Observe(ctx *ReactContext) error {
	action := ctx.LastAction
	if action == nil {
		return fmt.Errorf("observe called without an action")
	}

	var obs *Observation

	switch action.Type {
	case ActionTypeToolCall:
		if action.Error != nil {
			obs = NewErrorObservation(action.Error, false)
			obs.Insights = []string{fmt.Sprintf("Tool %q execution failed", action.Target)}
		} else {
			insights := analyzeActionResult(action.Result)
			obs = NewSuccessObservation(action.Result, insights...)
		}

	case ActionTypeAnswer:
		obs = NewSuccessObservation(action.Result, "direct answer generated")

	case ActionTypeClarify:
		obs = NewSuccessObservation(action.Result, "clarification question generated")

	default:
		obs = NewSuccessObservation(action.Result)
	}

	ctx.LastObservation = obs
	return nil
}
