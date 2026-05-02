package reactor

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
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

// ---------------------------------------------------------------------------
// L3 Progressive Disclosure: Reference File Resolution
// ---------------------------------------------------------------------------

// maxReferenceFileSize is the maximum file size (in bytes) for inline reference
// injection into the LLM context. Files exceeding this limit are marked as TODO.
const maxReferenceFileSize = 64 * 1024 // 64KB

// textExtensions lists file extensions that qualify as text/reference files
// compliant with the Skill specification (Markdown-based).
var textExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
	".mdown":    true,
	".mkd":      true,
}

// referenceFile represents a discovered reference file with its classification.
type referenceFile struct {
	RelPath string // relative path under references/ (e.g., "guide.md")
	AbsPath string // absolute filesystem path
	Size    int64  // file size in bytes
	Ext     string // lowercase file extension (including dot)
}

// ResolvedReferences holds Level 3 resolved reference file contents ready for
// injection into the LLM context during Phase 2 (L2) planning.
//
// Two output formats:
//   - Content: <references>filename\n[file contents]\n...</references>
//     for text/markdown files within size limits
//   - Links:   <reference-links>path, size bytes</reference-links>
//     for binary or oversized files (metadata only)
type ResolvedReferences struct {
	Content      string // XML block with inline text file contents
	Links        string // XML block with binary/oversized file metadata
	FilesLoaded  int    // count of successfully loaded text files
	FilesSkipped int    // count of skipped (oversized/binary/non-compliant) files
}

// resolveL3References performs Level 3 progressive disclosure: scans the skill's
// references/ directory and resolves reference files into injectable content.
//
// This implements the third stage of Skill progressive disclosure (see design doc
// 渐进式披露设计方案.md §5 Skill 三层披露):
//
//	L1 Metadata   (~100 tokens)  → Name + Description → startup
//	L2 Instructions(<5000 tokens) → SKILL.md body    → activation
//	L3 Resources   (按需)         → references/*      → this function
//
// # Classification rules
//
//   - .md/.markdown/.txt files ≤ maxReferenceFileSize: read full content,
//     wrap in <references>...</references> for direct context injection
//   - Text files > maxReferenceFileSize: emit [TODO] marker with size info;
//     large file handling is deferred until a chunking/summarization strategy
//     is designed (would otherwise risk context window overflow)
//   - Binary files (detected via null-byte heuristic in first 512 bytes):
//     emit filename + size in <reference-links>...</reference-links>
//   - Non-markdown non-text extensions: silently skipped — they don't conform
//     to the Skill specification's reference format; future work may add a
//     file analyzer pipeline to convert arbitrary formats to Markdown
func (r *Reactor) resolveL3References(actCtx *ActivatedSkillContext) *ResolvedReferences {
	if actCtx == nil || actCtx.ResourceBasePath == "" {
		return nil
	}

	refDir := filepath.Join(actCtx.ResourceBasePath, "references")
	_, err := fs.Stat(core.OS, strings.TrimLeft(refDir, "/"))
	if err != nil {
		return nil // no references/ directory — L3 not applicable
	}

	entries, err := fs.ReadDir(core.OS, strings.TrimLeft(refDir, "/"))
	if err != nil || len(entries) == 0 {
		return nil
	}

	var refs []referenceFile
	for _, e := range entries {
		if e.IsDir() {
			continue // skip subdirectories
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		fi, statErr := e.Info()
		if statErr != nil {
			continue
		}
		refs = append(refs, referenceFile{
			RelPath: name,
			AbsPath: filepath.Join(refDir, name),
			Size:    fi.Size(),
			Ext:     ext,
		})
	}
	if len(refs) == 0 {
		return nil
	}

	result := &ResolvedReferences{}
	var contentSB strings.Builder
	var linksSB strings.Builder

	contentSB.WriteString("<references>\n")
	hasContent := false

	for _, rf := range refs {
		switch {
		case textExtensions[rf.Ext]:
			// --- Text/Markdown file ---
			if rf.Size > maxReferenceFileSize {
				// Oversized: TODO marker — would risk context window overflow
				fmt.Fprintf(&linksSB, "  - %s (%d bytes, oversized — TODO: chunking strategy pending)\n",
					rf.RelPath, rf.Size)
				result.FilesSkipped++
				continue
			}
			data, readErr := core.ReadFileFromFS(core.OS, rf.AbsPath)
			if readErr != nil {
				logger.Warn("L3: failed to read reference file", "path", rf.AbsPath, "error", readErr)
				result.FilesSkipped++
				continue
			}
			// Safety check: ensure no null bytes sneaked through extension check
			if containsNullBytes(data) {
				fmt.Fprintf(&linksSB, "  - %s (%d bytes, detected binary)\n", rf.RelPath, rf.Size)
				result.FilesSkipped++
				continue
			}
			fmt.Fprintf(&contentSB, "--- %s (%d bytes) ---\n%s\n", rf.RelPath, rf.Size, data)
			result.FilesLoaded++
			hasContent = true

		default:
			// --- Binary or non-compliant extension ---
			isBinary := false
			if f, openErr := core.OpenFromFS(core.OS, rf.AbsPath); openErr == nil {
				buf := make([]byte, 512)
				n, _ := f.Read(buf)
				f.Close()
				isBinary = containsNullBytes(buf[:n])
			}

			if isBinary {
				fmt.Fprintf(&linksSB, "  - %s (%d bytes, binary)\n", rf.RelPath, rf.Size)
			} else {
				// Non-markdown text file (.py, .json, .yaml etc.) — skip per spec.
				// Future work: add file analyzer pipeline for format conversion.
				logger.Debug("L3: skipping non-compliant reference file",
					"path", rf.RelPath, "ext", rf.Ext)
			}
			result.FilesSkipped++
		}
	}

	contentSB.WriteString("</references>\n")
	if hasContent {
		result.Content = contentSB.String()
	}
	if linksSB.Len() > 0 {
		var lb strings.Builder
		lb.WriteString("<reference-links>\n")
		lb.WriteString(linksSB.String())
		lb.WriteString("</reference-links>")
		result.Links = lb.String()
	}

	if result.FilesLoaded == 0 && result.FilesSkipped == 0 {
		return nil
	}

	logger.Info("L3 Progressive Disclosure resolved",
		"loaded", result.FilesLoaded, "skipped", result.FilesSkipped)
	return result
}

// containsNullBytes checks whether data contains any null bytes (\x00),
// which is a reliable heuristic for detecting binary (non-text) content.
func containsNullBytes(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// Think asks the LLM to decide the next action based on the current context.
// Implements PROGRESSIVE DISCLOSURE multi-phase thinking (L1 -> L2 -> L3):
//
//	Phase 1 (L1 Routing):  All tools loaded as MINIMAL schema (~50 tokens/tool)
//	                        + all skills metadata in SystemPrompt.
//	                        LLM routes to: "tool" | "skill" | "answer" | "delegate"
//	                        Outputs selected target(s) by name.
//	Phase 2 (L2 Planning):  Selected tools upgraded to FULL schema.
//	                        If skill routed: skill L2 instructions injected.
//	Phase 3 (L3 Resources): If skill activated, scan references/ directory and
//	                        resolve reference files (text inlined, binary as links).
//	                        LLM generates actual action with proper parameters.
//
// Verified by Experiment 1: Qwen accepts empty-properties NativeTools.
// Verified by Experiment 2: LLM can semantically match capabilities to tool names.
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
	estimateFn := r.tokenEstimator.Estimate
	totalTokens := 0

	allToolInfos := core.ToToolInfos(r.toolRegistry.All())
	minimalLLMTools := ToolInfosToMinimalLLMTools(allToolInfos)
	skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)
	skillsSection := BuildSkillsSystemPrompt(skills)

	memoryRecords := r.retrieveMemory(ctx)

	accountTokens := func(content string) {
		if r.contextWindow != nil && content != "" {
			r.contextWindow.AddTokens(int64(estimateFn(content)))
		}
	}

	// --- Pre-L1 token accounting ---
	accountTokens(skillsSection)
	accountTokens(r.config.SystemPrompt)
	minimalTokens := EstimateTokensForTools(minimalLLMTools, estimateFn)
	if minimalTokens > 0 && r.contextWindow != nil {
		r.contextWindow.AddTokens(minimalTokens)
	}

	// ====================================================================
	// PHASE 1 (L1 Routing): Minimal tools + skills -> route decision
	// ====================================================================
	l1Result, l1Tokens, historyForL1, err := r.thinkL1(ctx, skills, skillsSection, accountTokens)
	totalTokens += l1Tokens
	if err != nil {
		return totalTokens, err
	}

	// ====================================================================
	// DECISION SWITCH: Early exit for answer/delegate/default paths
	// ====================================================================
	var actCtx *ActivatedSkillContext
	var l1SelectedToolNames []string
	var l3Refs *ResolvedReferences

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
		ctx.LastThought = &Thought{
			Decision:       DecisionDelegate,
			Reasoning:      l1Result.Reasoning,
			DelegateTarget: l1Result.Target,
			DelegatePrompt: l1Result.Answer,
		}
		if ctx.LastThought.DelegatePrompt == "" {
			ctx.LastThought.DelegatePrompt = ctx.Input
		}
		return totalTokens, nil

	case "tool", "skill":
		// Responsibility Gate
		if r.orchestrator != nil {
			gateTokens, gateErr := r.executeResponsibilityGate(ctx, l1Result, totalTokens)
			totalTokens += gateTokens
			if gateErr != nil {
				logger.Warn("responsibility gate error, falling through to Level 2", "error", gateErr)
			} else {
				return totalTokens, nil
			}
		}

		// Skill activation + L3 resolution
		actCtx, l1Result, l3Refs = r.activateAndResolve(l1Result, allToolInfos, accountTokens)
		l1SelectedToolNames = r.resolveToolNames(l1Result)
		if l1Result.Path != "skill" && l1Result.Target != "" && l1Result.Path != "" {
			// actCtx was set by activateAndResolve, path may have changed to "tool"
		}
		l1SelectedToolNames = r.resolveToolNames(l1Result)

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
	l2Tokens, err := r.thinkL2(ctx, l1SelectedToolNames, actCtx, l3Refs, skillsSection, allToolInfos,
		minimalTokens, minimalLLMTools, memoryRecords, historyForL1, skills, accountTokens, estimateFn)
	totalTokens += l2Tokens
	if err != nil {
		return totalTokens, err
	}

	return totalTokens, nil
}

// retrieveMemory fetches relevant memory records for the current context.
func (r *Reactor) retrieveMemory(ctx *ReactContext) []core.MemoryRecord {
	if r.memory == nil {
		return nil
	}
	records, err := r.memory.Retrieve(
		ctx.Ctx(), ctx.Input,
		core.WithMemoryTypes(core.MemoryTypeLongTerm, core.MemoryTypeUser, core.MemoryTypeExperience),
		core.WithMemoryLimit(3),
	)
	if err != nil {
		ctx.EmitEvent(core.Error, fmt.Sprintf("memory retrieval failed (non-fatal): %v", err))
		return nil
	}
	return records
}

// thinkL1 executes Phase 1 L1 routing and returns the parsed result.
func (r *Reactor) thinkL1(ctx *ReactContext, skills []*core.Skill, skillsSection string, accountTokens func(string)) (*l1RoutingResult, int, ConversationHistory, error) {
	totalTokens := 0

	l1Prompt := buildL1RoutingPrompt(skills)
	accountTokens(l1Prompt)
	accountTokens(ctx.Input)

	maxTurns := r.maxHistoryTurns()
	historyForL1 := ctx.ConversationHistory
	if maxTurns > 0 && len(historyForL1) > maxTurns {
		historyForL1 = historyForL1[len(historyForL1)-maxTurns:]
	}
	for _, msg := range historyForL1 {
		accountTokens(msg.Content)
	}

	l1Content, l1Tokens, err := r.callLLMStream(
		ctx, l1Prompt, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), nil, skillsSection,
	)
	if err != nil {
		return nil, totalTokens + l1Tokens, nil, fmt.Errorf("think L1 routing failed: %w", err)
	}
	totalTokens += l1Tokens

	l1InputTokens := r.estimateInputTokens(l1Prompt, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), nil, skillsSection)
	totalTokens += l1InputTokens

	l1Result, err := parseL1RoutingResponse(l1Content)
	if err != nil {
		return nil, totalTokens, nil, fmt.Errorf("think L1 parse failed: %w", err)
	}
	logger.Info("L1 Route", "path", l1Result.Path, "target", l1Result.Target,
		"tools", l1Result.SelectedTools, "reasoning", truncate(l1Result.Reasoning, 200))

	return l1Result, totalTokens, historyForL1, nil
}

// activateAndResolve handles skill activation and L3 reference resolution when L1 routes to a skill.
func (r *Reactor) activateAndResolve(l1Result *l1RoutingResult, allToolInfos []core.ToolInfo, accountTokens func(string)) (*ActivatedSkillContext, *l1RoutingResult, *ResolvedReferences) {
	if l1Result.Path != "skill" || l1Result.Target == "" {
		return nil, l1Result, nil
	}

	actCtx, err := r.ActivateSkill(l1Result.Target, allToolInfos)
	if err != nil {
		logger.Info("Skill activation failed, falling back to direct tool mode", "error", err)
		return nil, l1Result, nil
	}

	if actCtx == nil {
		return nil, l1Result, nil
	}

	accountTokens(actCtx.Instructions)

	var l3Refs *ResolvedReferences
	if actCtx != nil {
		l3Refs = r.resolveL3References(actCtx)
		if l3Refs != nil {
			if l3Refs.Content != "" {
				accountTokens(l3Refs.Content)
			}
			if l3Refs.Links != "" {
				accountTokens(l3Refs.Links)
			}
		}
	}

	return actCtx, l1Result, l3Refs
}

// resolveToolNames extracts selected tool names from the L1 result.
func (r *Reactor) resolveToolNames(l1Result *l1RoutingResult) []string {
	if len(l1Result.SelectedTools) > 0 {
		return l1Result.SelectedTools
	}
	if l1Result.Target != "" {
		return []string{l1Result.Target}
	}
	return nil
}

// thinkL2 executes Phase 2 L2 planning with full-schema tools.
func (r *Reactor) thinkL2(ctx *ReactContext, l1SelectedToolNames []string, actCtx *ActivatedSkillContext,
	l3Refs *ResolvedReferences, skillsSection string, allToolInfos []core.ToolInfo,
	minimalTokens int64, minimalLLMTools []gochatcore.Tool, memoryRecords []core.MemoryRecord,
	historyForL1 ConversationHistory, skills []*core.Skill, accountTokens func(string), estimateFn func(string) int) (int, error) {

	totalTokens := 0

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
	if r.contextWindow != nil {
		if fullSchemaTokens > minimalTokens {
			r.contextWindow.AddTokens(fullSchemaTokens - minimalTokens)
		} else if fullSchemaTokens > 0 {
			r.contextWindow.AddTokens(fullSchemaTokens)
		}
	}

	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, memoryRecords, actCtx, r.intentRegistry, l3Refs)
	accountTokens(instructions)
	accountTokens(ctx.Input)
	for _, msg := range historyForL1 {
		accountTokens(msg.Content)
	}

	r.checkSlide(ctx.Ctx())

	content, tokens, err := r.callLLMStream(ctx, instructions, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), l2FullSchemaTools, skillsSection)
	if err != nil {
		return totalTokens + tokens, fmt.Errorf("think L2 planning failed: %w", err)
	}
	totalTokens += tokens

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
// The model sees ALL tools with minimal schema and ALL skill metadata, then routes.
// Agent delegation is NOT handled at L1 — routing decisions are delegated to the Orchestrator.
func buildL1RoutingPrompt(skills []*core.Skill) string {
	result, err := renderL1RoutingPrompt(l1RoutingPromptData{
		HasSkills: len(skills) > 0,
	})
	if err != nil {
		return fmt.Sprintf("l1 routing prompt render error: %v", err)
	}
	return result
}

// parseL1RoutingResponse extracts the L1 routing result from LLM response JSON.
func parseL1RoutingResponse(content string) (*l1RoutingResult, error) {
	content = core.StripMarkdownCodeBlock(content)
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
// Supports both Executor mode (tool calls / answers) and Coordinator mode
// (sub-task result collection via orchestrator).
func (r *Reactor) Act(ctx *ReactContext) error {
	thought := ctx.LastThought
	if thought == nil {
		return fmt.Errorf("act called without a thought")
	}

	start := time.Now()
	action := Action{
		Timestamp: start,
	}

	// ====== Coordinator Mode Branch ======
	// When Think produced DecisionCoordinate, we are in Coordinator mode.
	// Act does not execute tools — instead it triggers dispatch if not yet done,
	// or reports current coordination status.
	if thought.Decision == DecisionCoordinate && ctx.Mode == ModeCoordinator && ctx.CoordState != nil {
		action.Type = ActionTypeToolCall // Reuse tool_call type for coordination action
		action.Target = "coordinate"
		action.Result = ctx.CoordState.TaskProgress.Summary()

		ctx.LastAction = &action
		return nil
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
// In Executor mode: analyzes tool execution results (existing logic).
// In Coordinator mode: analyzes sub-task completion status, checks if all done.
func (r *Reactor) Observe(ctx *ReactContext) error {
	action := ctx.LastAction
	if action == nil {
		return fmt.Errorf("observe called without an action")
	}

	// ====== Coordinator Mode Branch ======
	if ctx.Mode == ModeCoordinator && ctx.CoordState != nil {
		return r.observeCoordinator(ctx)
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

// observeCoordinator handles Observe phase when in Coordinator mode.
// Checks sub-task progress, determines if all tasks are done, and produces
// a final answer or continues waiting.
func (r *Reactor) observeCoordinator(ctx *ReactContext) error {
	cs := ctx.CoordState
	if cs == nil || cs.TaskProgress == nil {
		return fmt.Errorf("coordinator mode but no CoordState")
	}

	tp := cs.TaskProgress
	total := tp.Count()
	completed := tp.CompletedCount()
	failed := tp.FailedCount()
	pending := tp.PendingCount()

	logger.Info("coordinator observe",
		"total", total, "completed", completed, "failed", failed, "pending", pending)

	var obs *Observation

	if tp.AllCompleted() {
		// All tasks done → produce summary answer
		summary := r.buildCoordinatorSummary(cs)
		cs.MarkCompleted()

		// Switch back to Executor mode for next cycle (if any)
		ctx.Mode = ModeExecutor

		obs = NewSuccessObservation(summary,
			fmt.Sprintf("all %d tasks done (%d succeeded, %d failed)", total, completed, failed))
		ctx.LastThought = &Thought{
			Decision:    DecisionAnswer,
			Reasoning:   fmt.Sprintf("Coordination complete. %d/%d tasks succeeded.", completed, total),
			FinalAnswer: summary,
			IsFinal:     true,
		}
	} else if pending > 0 {
		// Still waiting for results
		obs = NewSuccessObservation(tp.Summary(),
			fmt.Sprintf("coordinating: %d/%d complete, %d pending", completed+failed, total, pending))

		// Check lifecycle state — if cancelled/interrupted, stop
		if cs.LifecycleState.IsTerminal() {
			summary := r.buildCoordinatorSummary(cs)
			obs = NewSuccessObservation(summary, fmt.Sprintf("coordination ended: %s", cs.LifecycleState))
			ctx.LastThought = &Thought{
				Decision:    DecisionAnswer,
				FinalAnswer: summary,
				IsFinal:     true,
			}
		}
	} else {
		// All terminal states reached (mix of success/fail)
		summary := r.buildCoordinatorSummary(cs)
		cs.MarkCompleted()
		ctx.Mode = ModeExecutor

		obs = NewSuccessObservation(summary,
			fmt.Sprintf("coordination finished: %d succeeded, %d failed", completed, failed))
		ctx.LastThought = &Thought{
			Decision:    DecisionAnswer,
			Reasoning:   "All sub-tasks reached terminal state",
			FinalAnswer: summary,
			IsFinal:     true,
		}
	}

	ctx.LastObservation = obs
	return nil
}

// buildCoordinatorSummary builds a human-readable summary of all sub-task results.
func (r *Reactor) buildCoordinatorSummary(cs *CoordState) string {
	if cs.TaskProgress == nil {
		return "(no task progress)"
	}

	var sb strings.Builder
	entries := cs.TaskProgress.ListAll()

	sb.WriteString("## Task Coordination Summary\n\n")

	for _, e := range entries {
		statusIcon := map[TaskStatus]string{
			TaskSucceeded:    "[OK]",
			TaskFailed:       "[FAIL]",
			TaskTimedOut:     "[TIMEOUT]",
			TaskCancelled:    "[CANCELLED]",
			TaskRetryPending: "[RETRY]",
		}[e.Status]

		if statusIcon == "" {
			statusIcon = string(e.Status)
		}

		fmt.Fprintf(&sb, "%s **%s** (priority=%d)\n", statusIcon, e.Title, e.Priority)

		if e.Result != nil && e.Result.Content != "" {
			// Truncate very long results
			content := e.Result.Content
			if len(content) > 500 {
				content = content[:500] + "...(truncated)"
			}
			fmt.Fprintf(&sb, "  Result: %s\n", content)
		}
		if e.Error != nil {
			fmt.Fprintf(&sb, "  Error: %v\n", e.Error)
		}
		sb.WriteString("\n")
	}

	// Append aggregated stats
	s := cs.TaskProgress
	fmt.Fprintf(&sb, "---\nTotal: %d | Succeeded: %d | Failed: %d\n",
		s.Count(), s.CompletedCount(), s.FailedCount())

	return sb.String()
}
