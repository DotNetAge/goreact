// Package reactor implements GoReAct's T-A-O execution engine.
//
// This file implements the four-step responsibility gate (Design §5) that sits
// between Level 1 routing and Level 2 planning in the Think phase.
//
// The gate determines whether an agent should:
//   - Execute the task itself (Executor mode, existing path via Level 2)
//   - Delegate to the orchestrator (Coordinator mode, new path)
//   - Decompose into sub-tasks via WBS and coordinate their execution
//
// Gate is only active when EnableOrchestration=true in AgentConfig AND
// an orchestrator is wired into the Reactor. Otherwise the agent behaves
// identically to the pre-orchestration version.
package reactor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ===========================================================================
// Step A: Responsibility Check (§5.1)
// =========================================================================//

// checkResponsibility determines whether the current task falls within this
// agent's functional scope by comparing user intent against its Description.
//
// Two strategies are tried in order:
//  1. Keyword match (fast path): check if Description keywords appear in intent
//  2. LLM semantic match (default path): ask LLM to judge similarity
//
// Returns ResponsibilityCheck with IsMatch=true if agent should handle it.
func (r *Reactor) checkResponsibility(ctx *ReactContext, l1Result *l1RoutingResult) (*ResponsibilityCheck, error) {
	description := r.config.SystemPrompt
	if len(description) > 1024 {
		description = description[:1024]
	}
	if description == "" {
		description = "general-purpose assistant"
	}

	// Fast path: keyword overlap detection
	if r.keywordMatch(description, ctx.Input) {
		return &ResponsibilityCheck{
			IsMatch:    true,
			Confidence: 0.8,
			Reasoning:  "keyword match in description",
		}, nil
	}

	// Default path: LLM semantic judgment
	prompt := fmt.Sprintf(`You are checking whether a task matches an agent's capabilities.

## Agent Description
%s

## User Task
%s

## Intent Summary
%s

Respond with JSON only:
{"is_match": true/false, "confidence": 0.0-1.0, "reasoning": "brief explanation"}`,
		description, ctx.Input,
		func() string {
			if ctx.Intent != nil { return ctx.Intent.Summary }
			return ctx.Input
		}(),
	)

	resp, err := r.callLLMForGate(ctx, prompt)
	if err != nil {
		// Fallback: assume it IS our responsibility (safe default)
		logger.Warn("responsibility check LLM failed, assuming match", "error", err)
		return &ResponsibilityCheck{
			IsMatch:    true,
			Confidence: 0.5,
			Reasoning:  "llm-failed-assume-match",
		}, nil
	}

	var result ResponsibilityCheck
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return &ResponsibilityCheck{
			IsMatch:    true,
			Confidence: 0.5,
			Reasoning:  "parse-failed-assume-match",
		}, nil
	}

	return &result, nil
}

// keywordMatch checks if any significant word from desc appears in input.
func (r *Reactor) keywordMatch(desc, input string) bool {
	descWords := tokenize(desc)
	inputLower := strings.ToLower(input)
	matchCount := 0
	for _, w := range descWords {
		if len(w) <= 2 {
			continue // skip short words
		}
		if strings.Contains(inputLower, strings.ToLower(w)) {
			matchCount++
		}
	}
	// At least 2 keyword hits or >30% of description words
	return matchCount >= 2 || (len(descWords) > 0 && float64(matchCount)/float64(len(descWords)) > 0.3)
}

// tokenize splits text into words, removing punctuation.
func tokenize(text string) []string {
	var words []string
	var current strings.Builder
	for _, ch := range text {
		if isAlphaNumeric(ch) {
			current.WriteRune(ch)
		} else if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func isAlphaNumeric(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// ===========================================================================
// Step B: Atomicity / WBS Decomposition Check (§5.1)
// =========================================================================//

// checkAtomicity asks the LLM whether the current task is atomic (can be
// completed by a single agent in one T-A-O cycle) or requires WBS
// decomposition into multiple sub-tasks.
func (r *Reactor) checkAtomicity(ctx *ReactContext, l1Result *l1RoutingResult) (*AtomicityCheck, error) {
	// Build available capabilities list for context
	capabilities := r.buildCapabilityList()

	taskDesc := fmt.Sprintf("Task: %s\nIntent: %s", ctx.Input,
		func() string {
			if ctx.Intent != nil { return fmt.Sprintf("%s (topic: %s)", ctx.Intent.Summary, ctx.Intent.Topic) }
			return ctx.Input
		}(),
	)

	prompt := fmt.Sprintf(`You are analyzing whether a complex task needs to be decomposed into subtasks.

## Task Decomposition Judgment (Step B)

Before diving into execution, you need to determine if the current task needs to be decomposed into multiple subtasks.

### Judgment Criteria
A task SHOULD be decomposed if it meets ANY of the following conditions:
1. Requires 3 or more different Skill/Tool combinations to complete
2. Contains obvious serial or parallel steps (do A then B; or A and B can be done simultaneously)
3. Different steps may require different domain knowledge
4. Estimated total duration exceeds 60 seconds

### Decomposition Requirements (if decomposition is needed)
- Break the task into atomic subtasks, each completable by a single agent independently
- Specify dependencies between subtasks (DependsOn)
- Specify desired capability for each subtask (DesiredCapability), leave empty if uncertain
- Set reasonable priority (Priority, lower number = higher priority)
- Each subtask description must be detailed enough for another agent to complete independently based on the description alone

## Input Task
%s
%s

## Output Format (JSON only, no markdown, no code blocks)
{
  "requires_decomposition": true/false,
  "reasoning": "why decomposition is or isn't needed",
  "sub_tasks": [
    {
      "id": "task-001",
      "title": "brief title",
      "description": "detailed description including goals, inputs, expected outputs",
      "capability": "desired capability or empty",
      "priority": 1,
      "depends_on": []
    }
  ]
}`,
		taskDesc,
		func() string {
			if capabilities == "" { return "" }
			return "\n### Available Capabilities\n" + capabilities
		}(),
	)

	resp, err := r.callLLMForGate(ctx, prompt)
	if err != nil {
		// Fallback: treat as atomic (safer — avoids infinite decomposition loops)
		logger.Warn("atomicity check LLM failed, treating as atomic", "error", err)
		return &AtomicityCheck{
			IsAtomic:  true,
			Reasoning: "llm-failed-assume-atomic",
		}, nil
	}

	var result struct {
		RequiresDecomposition bool                `json:"requires_decomposition"`
		Reasoning             string              `json:"reasoning"`
		SubTasks              []TaskDecomposition `json:"sub_tasks"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return &AtomicityCheck{
			IsAtomic:  true,
			Reasoning: "parse-failed-assume-atomic",
		}, nil
	}

	return &AtomicityCheck{
		IsAtomic:  !result.RequiresDecomposition,
		SubTasks:  result.SubTasks,
		Reasoning: result.Reasoning,
	}, nil
}

// buildCapabilityList builds a summary of available skills/capabilities for the prompt.
func (r *Reactor) buildCapabilityList() string {
	skills := r.skillRegistry.ListSkills()
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, s := range skills {
		fmt.Fprintf(&sb, "- %s: %s\n", s.Name, s.Description)
	}
	return sb.String()
}

// ===========================================================================
// Step C: Dispatch to Orchestrator (§5.1)
// =========================================================================//

// dispatchAndCoordinate enters Coordinator mode: dispatches all sub-tasks to
// the orchestrator and returns a Coordinate decision that causes the T-A-O
// loop to switch into result-collection mode.
func (r *Reactor) dispatchAndCoordinate(ctx *ReactContext, l1Tokens int, subTasks []TaskDecomposition) (int, error) {
	if r.orchestrator == nil {
		// No orchestrator — fall back to executing ourselves
		logger.Warn("no orchestrator available, falling back to executor mode")
		return l1Tokens, nil
	}

	totalTokens := l1Tokens

	// Initialize CoordState
	parentID := fmt.Sprintf("parent-%d", time.Now().UnixNano()%1000000)
	cs := NewCoordState(parentID, 10*time.Minute) // Default 10 min global timeout
	ctx.Mode = ModeCoordinator
	ctx.CoordState = cs

	logger.Info("entering Coordinator mode", "parent_task", parentID, "subtasks", len(subTasks))

	// Register and dispatch each sub-task
	for _, st := range subTasks {
		entry := &TaskEntry{
			TaskID:       st.ID,
			Title:        st.Title,
			Description:  st.Description,
			Priority:     st.Priority,
			Status:       TaskDispatched,
			MaxRetries:   2,
			DispatchedAt: ptrTime(time.Now()),
		}
		cs.TaskProgress.Add(entry)
		cs.RegisterSubTask(st.ID)

		// Delegate to orchestrator
		delegatePrompt := st.Description
		if delegatePrompt == "" {
			delegatePrompt = st.Title
		}

		result, delErr := r.orchestrator.DelegateTo(
			ctx.Ctx(),
			"", // let orchestrator pick agent based on capability hint
			delegatePrompt,
			parentID,
			map[string]any{"desired_capability": st.DesiredCapability, "priority": st.Priority},
		)

		if delErr != nil {
			logger.Error("sub-task delegation failed", "task_id", st.ID, "error", delErr)
			cs.TaskProgress.UpdateStatus(st.ID, TaskFailed, WithError(delErr), WithTimestamps())
		} else {
			logger.Info("sub-task delegated", "task_id", st.ID, "delegate_task", result.TaskID)
			cs.TaskProgress.UpdateStatus(st.ID, TaskAssigned, WithTimestamps())

			// Start async waiter goroutine for this task
			go r.waitForSubTaskResult(ctx, st.ID, result.TaskID, cs)
		}

		totalTokens += 50 // rough estimate for bookkeeping
	}

	// Build coordinate thought
	ctx.LastThought = &Thought{
		Decision:       DecisionCoordinate,
		Reasoning:      fmt.Sprintf("WBS decomposed %d sub-tasks, entered Coordinator mode", len(subTasks)),
		IsFinal:        false,
		Timestamp:      time.Now(),
	}

	return totalTokens, nil
}

// waitForSubTaskResult waits asynchronously for a single sub-task result and
// updates the CoordState when it arrives.
func (r *Reactor) waitForSubTaskResult(ctx *ReactContext, subTaskID, orchTaskID string, cs *CoordState) {
	result, err := r.orchestrator.WaitForResult(ctx.Ctx(), orchTaskID)
	if err != nil {
		logger.Warn("sub-task wait failed", "task_id", subTaskID, "error", err)
		cs.TaskProgress.UpdateStatus(subTaskID, TaskFailed, WithError(err), WithTimestamps())
		return
	}

	// Update progress table with success
	holder := &TaskResultHolder{
		Content:  result.Output,
		Duration: 0, // TODO: extract from result if available
		Score:    2, // Default score (will be re-evaluated in Observe)
	}
	cs.TaskProgress.UpdateStatus(subTaskID, TaskSucceeded, WithResult(holder), WithTimestamps())

	// Store raw result
	cs.SubTaskResults[subTaskID] = &core.TaskResultEvent{
		TaskID:   subTaskID,
		Result:   result.Output,
		Duration: 0,
		Timestamp: time.Now(),
	}

	logger.Info("sub-task completed", "task_id", subTaskID)
}

// ===========================================================================
// Delegate-to-Orchestrator Path (Step A → Not My Responsibility)
// =========================================================================//

// delegateToOrchestrator handles the case where Step A determines the task is
// NOT this agent's responsibility. It delegates entirely to the orchestrator
// and sets up Coordinator mode for result collection.
func (r *Reactor) delegateToOrchestrator(ctx *ReactContext, l1Result *l1RoutingResult, respCheck *ResponsibilityCheck) (int, error) {
	if r.orchestrator == nil {
		// No orchestrator — fall through to normal execution with warning
		logger.Warn("no orchestrator but task not our responsibility, handling anyway")
		return 0, fmt.Errorf("no orchestrator available for delegation")
	}

	parentID := fmt.Sprintf("delegated-%d", time.Now().UnixNano()%1000000)
	cs := NewCoordState(parentID, 5*time.Minute)
	ctx.Mode = ModeCoordinator
	ctx.CoordState = cs

	// Create a single entry for the delegated task
	taskID := "delegated-task-1"
	entry := &TaskEntry{
		TaskID:       taskID,
		Title:        "Delegated task",
		Description:  ctx.Input,
		Priority:     1,
		Status:       TaskDispatched,
		MaxRetries:   2,
		DispatchedAt: ptrTime(time.Now()),
	}
	cs.TaskProgress.Add(entry)
	cs.RegisterSubTask(taskID)

	// Delegate to orchestrator
	delegatePrompt := ctx.Input
	if l1Result.Reasoning != "" {
		delegatePrompt = fmt.Sprintf("[Original reasoning: %s]\n\n%s", l1Result.Reasoning, ctx.Input)
	}

	result, delErr := r.orchestrator.DelegateTo(
		ctx.Ctx(),
		"", // let orchestrator pick
		delegatePrompt,
		parentID,
		nil,
	)

	if delErr != nil {
		logger.Error("full delegation failed", "error", delErr)
		cs.TaskProgress.UpdateStatus(taskID, TaskFailed, WithError(delErr), WithTimestamps())
		// Fall back to answer mode
		ctx.LastThought = &Thought{
			Decision:    DecisionAnswer,
			Reasoning:   fmt.Sprintf("Delegation failed (%s). Handling directly.", delErr),
			FinalAnswer: fmt.Sprintf("I encountered an issue delegating this task. Error: %v", delErr),
			IsFinal:     true,
		}
		return 50, nil
	}

	cs.TaskProgress.UpdateStatus(taskID, TaskAssigned, WithTimestamps())
	go r.waitForSubTaskResult(ctx, taskID, result.TaskID, cs)

	ctx.LastThought = &Thought{
		Decision:       DecisionCoordinate,
		Reasoning:      fmt.Sprintf("Not my responsibility (confidence %.2f). Delegated to orchestrator.", respCheck.Confidence),
		IsFinal:        false,
		DelegateTarget: result.TaskID,
		Timestamp:      time.Now(),
	}

	return 50, nil
}

// ===========================================================================
// Helper: LLM Call for Gate Decisions
// =========================================================================//

// callLLMForGate makes a lightweight LLM call for gate decisions (Step A/B).
// Uses a simpler prompt/response pattern than the full Think pipeline.
func (r *Reactor) callLLMForGate(ctx *ReactContext, systemPrompt string) (string, error) {
	userMsg := ctx.Input
	if userMsg == "" {
		userMsg = "Please respond."
	}

	// Use mock LLM if configured (for testing)
	if r.mockLLM != nil {
		resp, err := r.mockLLM(systemPrompt, userMsg, nil)
		if err != nil {
			return "", err
		}
		return resp.Content, nil
	}

	// Real LLM call — use same builder pattern as buildLLMBuilder
	builder := r.llmClient.
		Model(r.config.Model).
		Temperature(r.config.Temperature).
		MaxTokens(r.config.MaxTokens).
		SystemMessage(systemPrompt)

	response, err := builder.GetResponseFor(r.config.ClientType)
	if err != nil {
		return "", err
	}
	if response.Content == "" {
		return "", fmt.Errorf("empty LLM response")
	}
	return response.Content, nil
}

// ptrTime returns a pointer to t.
func ptrTime(t time.Time) *time.Time { return &t }

// ===========================================================================
// executeResponsibilityGate — Orchestrates the 4-step gate (Design §5)
// =========================================================================//

// executeResponsibilityGate runs the full four-step gate between L1 routing
// and L2 planning. Returns (additionalTokens, error). If the gate handles the
// decision (delegate or coordinate), ctx.LastThought is set and the caller
// should return immediately. If error is non-nil, caller should fall through.
func (r *Reactor) executeResponsibilityGate(ctx *ReactContext, l1Result *l1RoutingResult, tokensSoFar int) (int, error) {
	// Step A: Responsibility Check — Is this my job?
	respCheck, err := r.checkResponsibility(ctx, l1Result)
	if err != nil {
		return 0, fmt.Errorf("step A (responsibility): %w", err)
	}
	logger.Info("gate Step A: responsibility", "is_match", respCheck.IsMatch,
		"confidence", respCheck.Confidence)

	if !respCheck.IsMatch {
		// Not our job → delegate to orchestrator → Coordinator mode
		delegateTokens, delErr := r.delegateToOrchestrator(ctx, l1Result, respCheck)
		return delegateTokens, delErr
	}

	// Step B: Atomicity Check — Can I handle this as one atomic task?
	atomicity, err := r.checkAtomicity(ctx, l1Result)
	if err != nil {
		return 0, fmt.Errorf("step B (atomicity): %w", err)
	}
	logger.Info("gate Step B: atomicity", "is_atomic", atomicity.IsAtomic,
		"subtasks", len(atomicity.SubTasks))

	if atomicity.IsAtomic {
		// Atomic task → proceed to Level 2 (normal executor path)
		// Return 0 extra tokens; caller continues to L2
		return 0, nil
	}

	// Non-atomic task → WBS decomposition done → Step C/D: dispatch & coordinate
	dispatchTokens, dispErr := r.dispatchAndCoordinate(ctx, tokensSoFar, atomicity.SubTasks)
	return dispatchTokens, dispErr
}
