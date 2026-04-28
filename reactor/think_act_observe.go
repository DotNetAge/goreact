package reactor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
	gochatcore "github.com/DotNetAge/gochat/core"
)

// l1RoutingResult holds the parsed L1 routing decision.
type l1RoutingResult struct {
	Path          string   // "tool" | "skill" | "answer"
	Target        string   // selected skill name or primary tool name
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
	accountTokens(skillsSection)
	minimalTokens := EstimateTokensForTools(minimalLLMTools, estimateFn)
	if minimalTokens > 0 && r.contextWindow != nil {
		r.contextWindow.AddTokens(minimalTokens)
	}

	// ====================================================================
	// PHASE 1 (L1 Routing): Minimal tools + skills -> route decision
	// ====================================================================
	l1Prompt := buildL1RoutingPrompt(skills)

	l1Content, l1Tokens, err := r.callLLMStream(
		ctx, l1Prompt, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), minimalLLMTools, skillsSection,
	)
	if err != nil {
		return totalTokens + l1Tokens, fmt.Errorf("think L1 routing failed: %w", err)
	}
	totalTokens += l1Tokens

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
			Reasoning:    l1Result.Reasoning,
			FinalAnswer:  l1Result.Answer,
			IsFinal:      true,
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
			Reasoning:    l1Result.Reasoning,
			FinalAnswer:  "I'm unsure how to proceed. Could you clarify your request?",
			IsFinal:      true,
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

	fullSchemaTokens := EstimateTokensForTools(l2FullSchemaTools, estimateFn)
	if fullSchemaTokens > 0 && r.contextWindow != nil {
		r.contextWindow.AddTokens(fullSchemaTokens)
	}

	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, memoryRecords, actCtx, r.intentRegistry)
	accountTokens(instructions)
	r.checkSlide(ctx.Ctx())

	content, tokens, err := r.callLLMStream(ctx, instructions, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), l2FullSchemaTools, skillsSection)
	if err != nil {
		return totalTokens + tokens, fmt.Errorf("think L2 planning failed: %w", err)
	}
	totalTokens += tokens

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
// The model sees ALL tools with minimal schema and ALL skill metadata, then routes.
func buildL1RoutingPrompt(skills []*core.Skill) string {
	hasSkills := len(skills) > 0
	base := "Analyze the user's input and determine the best way to respond.\n" +
		"\n" +
		"You see all available tools (with names and descriptions only) and any active capabilities below.\n" +
		"\n" +
		"Decide which path to take:\n" +
		"- **\"tool\"**: The user's request can be fulfilled by directly invoking one or more tools. Set 'selected_tools' to the tool name(s) you need.\n" +
		"- **\"skill\"**: The request requires specialized domain expertise. Choose the most applicable skill from <available_capabilities>. Set 'target' to the skill name.\n" +
		"- **\"answer\"**: This is a simple question, chitchat, or knowledge query that needs no tools. Provide the answer directly.\n" +
		"\n" +
		"Respond with JSON only:\n" +
		"{\"path\":\"tool|skill|answer\",\"target\":\"<skill_or_tool_name>\",\"selected_tools\":[\"tool_name1\",...],\"answer\":\"<if answer path>\",\"reasoning\":\"<brief explanation>\"}\n" +
		"\n" +
		"CRITICAL rules:\n" +
		"- For \"tool\" path: you MUST populate selected_tools with specific tool names from the available set\n" +
		"- For \"skill\" path: set target to the EXACT skill name from available_capabilities  \n" +
		"- Select the MINIMUM set of tools sufficient for the task\n" +
		"- When unsure between tool and skill, prefer \"skill\" for complex multi-step tasks"
	if hasSkills {
		return base + "\n\nAvailable capabilities (skills) are listed in the system prompt above — reference them by exact name."
	}
	return base
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
