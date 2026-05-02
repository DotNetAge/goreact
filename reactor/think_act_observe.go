package reactor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// Think executes a single thinking phase with full-schema tools.
// No L1 routing — the LLM decides tool vs answer in one call.
// The System Prompt and Instructions remain stable across rounds;
// direction is steered via tool result footers.
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
	// Pre-Think: restore any previously offloaded results from disk
	r.restoreOffloadedResults(ctx)

	// Use cached LLM tool definitions — rebuilt only when RegisterTool is called
	llmTools := r.getLLMTools()

	// Build system prompt sections using the centralized Prompt
	var sections []gochatcore.Message
	if r.prompt != nil {
		sections = r.prompt.ToSectionedMessages()
	}

	callInput := CallInput{
		SystemPromptSections: sections,
		UserMessage:          ctx.Input,
		History:              ctx.ConversationHistory,
		Tools:                llmTools,
	}

	var contentBuf strings.Builder
	result := r.llmCaller.CallStream(ctx.Ctx(), callInput, func(chunk string) {
		contentBuf.WriteString(chunk)
		ctx.EmitEvent(core.ThinkingDelta, chunk)
	})

	content := contentBuf.String()
	if content == "" {
		content = result.Content
	}

	var thought *Thought
	if len(result.ToolCalls) > 0 {
		thought = nativeToolCallsToThought(result.ToolCalls)
	} else {
		var parseErr error
		thought, parseErr = ParseThinkResponse(content)
		if parseErr != nil {
			return int(result.TokenUsage.InputTokens), fmt.Errorf("think parse failed: %w", parseErr)
		}
	}

	ctx.LastThought = thought
	return int(result.TokenUsage.InputTokens), nil
}

// nativeToolCallsToThought converts native gochat ToolCalls to the Thought format used by
// the Act phase. This bridges native function calling (non-streaming) with the
// Thought-based execution pipeline.
//
// Native tool calls are converted to DecisionAct with the ToolCalls map populated.
// The tool name → parameter map structure matches what executeToolCalls expects.
func nativeToolCallsToThought(tcs []gochatcore.ToolCall) *Thought {
	if len(tcs) == 0 {
		return nil
	}

	toolCalls := make(map[string]map[string]any, len(tcs))
	for _, tc := range tcs {
		var params map[string]any
		if tc.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Arguments), &params); err != nil {
				params = map[string]any{"raw_args": tc.Arguments}
			}
		}
		toolCalls[tc.Name] = params
	}

	return &Thought{
		Decision:  DecisionAct,
		ToolCalls: toolCalls,
		Reasoning: "LLM returned native tool calls",
	}
}

// Act executes the decision from the Think phase.
func (r *Reactor) Act(ctx *ReactContext) error {
	thought := ctx.LastThought
	if thought == nil {
		return fmt.Errorf("act called without a thought")
	}

	start := time.Now()

	switch thought.Decision {
	case DecisionAnswer:
		ctx.LastAction = &Action{
			Type:   ActionTypeAnswer,
			Result: coalesce(thought.FinalAnswer, thought.Reasoning),
		}
		return nil

	case DecisionClarify:
		q := thought.ClarificationQuestion
		if q == "" {
			q = "Could you provide more details so I can better assist you?"
		}
		ctx.LastAction = &Action{Type: ActionTypeClarify, Result: q}
		return nil

	case DecisionAct:
		return r.executeToolCalls(ctx, thought, start)

	default:
		ctx.LastAction = &Action{
			Type:   ActionTypeAnswer,
			Result: coalesce(thought.FinalAnswer, thought.Reasoning),
		}
		return nil
	}
}

// executeToolCalls executes tool calls in two phases:
//
//  1. Sync tools (IsAsync=false) execute SERIALLY, one at a time.
//     Each must complete before the next starts. Results are collected in order.
//
//  2. Async tools (IsAsync=true) execute in PARALLEL, launched in goroutines.
//     Each returns immediately with {task_id, status: "running"}.
//
// This ensures deterministic behavior for sync tools (e.g., read_file, grep)
// while allowing long-running async tools (e.g., web_search, bash) to run concurrently.
func (r *Reactor) executeToolCalls(ctx *ReactContext, thought *Thought, start time.Time) error {
	type toolCall struct {
		name   string
		params map[string]any
	}
	var calls []toolCall

	if len(thought.ToolCalls) > 0 {
		for name, params := range thought.ToolCalls {
			calls = append(calls, toolCall{name, params})
		}
	} else if thought.ActionTarget != "" {
		calls = append(calls, toolCall{thought.ActionTarget, thought.ActionParams})
	}

	if len(calls) == 0 {
		ctx.LastAction = &Action{
			Type:   ActionTypeAnswer,
			Result: coalesce(thought.FinalAnswer, "Sorry, I cannot determine which tool to use for your request."),
		}
		return nil
	}

	// Separate sync and async tool calls
	var syncCalls, asyncCalls []toolCall
	taskIDcounter := 0
	for _, c := range calls {
		isAsync := false
		if tool, ok := r.toolRegistry.Get(c.name); ok {
			isAsync = tool.Info().IsAsync
		}
		if isAsync {
			asyncCalls = append(asyncCalls, c)
		} else {
			syncCalls = append(syncCalls, c)
		}
	}

	var action Action
	action.Timestamp = start
	action.Type = ActionTypeToolCall
	var results []string

	// Phase 1: Execute sync tools SERIALLY (one at a time)
	for _, c := range syncCalls {
		res, err := r.toolExecutor.Execute(ctx.Ctx(), c.name, c.params)
		if err != nil {
			action.Error = err
			action.ErrorMsg = err.Error()
			results = append(results, fmt.Sprintf("[%s] error: %s", c.name, err.Error()))
		} else if res.Interaction != nil {
			answer, interactErr := r.interactionHandler.HandleInteraction(ctx.Ctx(), res.Interaction)
			if interactErr != nil {
				results = append(results, fmt.Sprintf("[%s] interaction error: %s", c.name, interactErr.Error()))
			} else {
				results = append(results, fmt.Sprintf("[%s] %s", c.name, answer))
			}
			if res.Duration > action.Duration {
				action.Duration = res.Duration
			}
		} else {
			results = append(results, fmt.Sprintf("[%s] %s", c.name, res.Result))
			if res.Duration > action.Duration {
				action.Duration = res.Duration
			}
		}
	}

	// Phase 2: Execute async tools in PARALLEL (all launched at once)
	if len(asyncCalls) > 0 {
		type asyncResult struct {
			name string
			id   string
		}
		asyncCh := make(chan asyncResult, len(asyncCalls))

		for _, c := range asyncCalls {
			c := c // capture
			go func() {
				go r.toolExecutor.Execute(ctx.Ctx(), c.name, c.params)
				taskIDcounter++
				taskID := fmt.Sprintf("async-%s-%d", c.name, taskIDcounter)
				asyncCh <- asyncResult{name: c.name, id: taskID}
			}()
		}

		for i := 0; i < len(asyncCalls); i++ {
			ar := <-asyncCh
			results = append(results, fmt.Sprintf("[%s] %s", ar.name,
				fmt.Sprintf(`{"task_id": "%s", "status": "running"}`, ar.id)))
		}
	}

	action.Result = strings.Join(results, "\n")
	ctx.LastAction = &action
	return nil
}

func coalesce(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
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
