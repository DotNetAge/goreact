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
// Merged Gate: Responsibility + Atomicity Check (Step A+B, Design §5.1)
// =========================================================================//

// gatePromptData contains all inputs needed for the merged gate prompt.
type gatePromptData struct {
	description    string // agent description (from SystemPrompt)
	userInput      string // user's original input
	intentSummary  string // intent classification summary
	intentTopic    string // intent topic
	capabilities   string // list of available skills/capabilities
}

// combinedGateResponse is the JSON structure parsed from the merged LLM call.
type combinedGateResponse struct {
	IsMatch               bool                `json:"is_match"`
	Confidence            float64             `json:"confidence"`
	MatchReasoning        string              `json:"match_reasoning"`
	RequiresDecomposition bool                `json:"requires_decomposition"`
	DecompReasoning       string              `json:"decomp_reasoning"`
	SubTasks              []TaskDecomposition `json:"sub_tasks,omitempty"`
}

// keywordMatch checks if any significant word from desc appears in input.
func (r *Reactor) keywordMatch(desc, input string) bool {
	descWords := tokenize(desc)
	inputLower := strings.ToLower(input)
	matchCount := 0
	for _, w := range descWords {
		if len(w) <= 2 {
			continue
		}
		if strings.Contains(inputLower, strings.ToLower(w)) {
			matchCount++
		}
	}
	return matchCount >= 2 || (len(descWords) > 0 && float64(matchCount)/float64(len(descWords)) > 0.3)
}

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

// checkResponsibilityAndAtomicity performs a single merged LLM call to determine
// both whether this agent should handle the task AND whether it needs decomposition.
//
// Before the LLM call, a keyword-match fast path is tried. If keywords match,
// the agent is known to be responsible and only the atomicity check runs.
//
// Returns ResponsibilityCheck, AtomicityCheck, and optionally decomposition sub-tasks.
func (r *Reactor) checkResponsibilityAndAtomicity(ctx *ReactContext, l1Result *l1RoutingResult) (*ResponsibilityCheck, *AtomicityCheck, error) {
	description := r.config.SystemPrompt
	if len(description) > 1024 {
		description = description[:1024]
	}
	if description == "" {
		description = "general-purpose assistant"
	}

	// Fast path: keyword match → known responsibility, only check atomicity
	if r.keywordMatch(description, ctx.Input) {
		atomicity := r.checkAtomicity(ctx, l1Result)
		return &ResponsibilityCheck{
			IsMatch:    true,
			Confidence: 0.8,
			Reasoning:  "keyword match in description",
		}, atomicity, nil
	}

	// Merged LLM path: build combined prompt
	data := gatePromptData{
		description: description,
		userInput:   ctx.Input,
		capabilities: r.buildCapabilityList(),
	}
	if ctx.Intent != nil {
		data.intentSummary = ctx.Intent.Summary
		data.intentTopic = ctx.Intent.Topic
	} else {
		data.intentSummary = ctx.Input
		data.intentTopic = ""
	}

	prompt := buildCombinedGatePrompt(data)

	resp, err := r.callLLMForGate(ctx, prompt)
	if err != nil {
		logger.Warn("combined gate LLM failed, assuming match + atomic", "error", err)
		return &ResponsibilityCheck{
			IsMatch:    true,
			Confidence: 0.5,
			Reasoning:  "llm-failed-assume-match",
		}, &AtomicityCheck{
			IsAtomic:  true,
			Reasoning: "llm-failed-assume-atomic",
		}, nil
	}

	var result combinedGateResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		logger.Warn("combined gate parse failed, assuming match + atomic", "error", err)
		return &ResponsibilityCheck{
			IsMatch:    true,
			Confidence: 0.5,
			Reasoning:  "parse-failed-assume-match",
		}, &AtomicityCheck{
			IsAtomic:  true,
			Reasoning: "parse-failed-assume-atomic",
		}, nil
	}

	respCheck := &ResponsibilityCheck{
		IsMatch:    result.IsMatch,
		Confidence: result.Confidence,
		Reasoning:  result.MatchReasoning,
	}

	// Decomposition is only relevant when agent IS responsible
	atomicity := &AtomicityCheck{
		IsAtomic:  !result.IsMatch || !result.RequiresDecomposition,
		SubTasks:  result.SubTasks,
		Reasoning: result.DecompReasoning,
	}

	return respCheck, atomicity, nil
}

// checkAtomicity runs atomicity-only check (used when keyword match bypasses merged gate).
func (r *Reactor) checkAtomicity(ctx *ReactContext, l1Result *l1RoutingResult) *AtomicityCheck {
	capabilities := r.buildCapabilityList()
	taskDesc := fmt.Sprintf("Task: %s\nIntent: %s", ctx.Input,
		func() string {
			if ctx.Intent != nil {
				return fmt.Sprintf("%s (topic: %s)", ctx.Intent.Summary, ctx.Intent.Topic)
			}
			return ctx.Input
		}(),
	)

	prompt := buildAtomicityOnlyPrompt(taskDesc, capabilities)

	resp, err := r.callLLMForGate(ctx, prompt)
	if err != nil {
		logger.Warn("atomicity check LLM failed, treating as atomic", "error", err)
		return &AtomicityCheck{IsAtomic: true, Reasoning: "llm-failed-assume-atomic"}
	}

	var result struct {
		RequiresDecomposition bool                `json:"requires_decomposition"`
		Reasoning             string              `json:"reasoning"`
		SubTasks              []TaskDecomposition `json:"sub_tasks"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return &AtomicityCheck{IsAtomic: true, Reasoning: "parse-failed-assume-atomic"}
	}

	return &AtomicityCheck{
		IsAtomic:  !result.RequiresDecomposition,
		SubTasks:  result.SubTasks,
		Reasoning: result.Reasoning,
	}
}

// buildCombinedGatePrompt constructs the merged prompt for responsibility + atomicity.
func buildCombinedGatePrompt(d gatePromptData) string {
	capSection := ""
	if d.capabilities != "" {
		capSection = "\n### Available Capabilities\n" + d.capabilities
	}

	return fmt.Sprintf(`You are determining whether a task matches an agent's capabilities and whether it needs decomposition.

## Agent Description
%s

## User Task
%s

## Intent Summary
%s
%s
%s

### Responsibility Judgment
A task IS your responsibility if its topic and intent match your agent description.
Respond with is_match=true if you are the right agent for this task.

### Decomposition Judgment (only relevant if is_match=true)
A task SHOULD be decomposed if it meets ANY of:
1. Requires 3+ different Skill/Tool combinations
2. Contains obvious serial or parallel steps
3. Different steps require different domain knowledge
4. Estimated total duration exceeds 60 seconds

### Decomposition Requirements (only if decomposed)
- Break into atomic subtasks, each completable independently
- Each subtask description must be self-contained
- Specify dependencies (depends_on) between subtasks
- Set reasonable priorities (lower = higher priority)
- Max 15 subtasks

## Output Format (JSON only, no markdown)
{
  "is_match": true/false,
  "confidence": 0.0-1.0,
  "match_reasoning": "why this task is/is not your responsibility",
  "requires_decomposition": false,
  "decomp_reasoning": "why decomposition is or isn't needed",
  "sub_tasks": []
}`,
		d.description, d.userInput, d.intentSummary,
		func() string {
			if d.intentTopic != "" { return "Topic: " + d.intentTopic }
			return ""
		}(),
		capSection,
	)
}

// buildAtomicityOnlyPrompt constructs the atomicity-only prompt (used after keyword match).
func buildAtomicityOnlyPrompt(taskDesc, capabilities string) string {
	capSection := ""
	if capabilities != "" {
		capSection = "\n### Available Capabilities\n" + capabilities
	}

	return fmt.Sprintf(`You are analyzing whether a complex task needs to be decomposed into subtasks.

## Task Decomposition Judgment

### Criteria
A task SHOULD be decomposed if it meets ANY of:
1. Requires 3+ different Skill/Tool combinations
2. Contains obvious serial or parallel steps
3. Different steps require different domain knowledge
4. Estimated total duration exceeds 60 seconds

### Requirements (if decomposed)
- Break into atomic subtasks, each completable independently
- Specify dependencies (depends_on)
- Specify desired capability (leave empty if uncertain)
- Set reasonable priority (lower = higher priority)
- Max 15 subtasks

## Input Task
%s
%s

## Output Format (JSON only, no markdown)
{
  "requires_decomposition": true/false,
  "reasoning": "why decomposition is or isn't needed",
  "sub_tasks": [
    {
      "id": "task-001",
      "title": "brief title",
      "description": "detailed description",
      "capability": "desired capability or empty",
      "priority": 1,
      "depends_on": []
    }
  ]
}`,
		taskDesc, capSection,
	)
}

// validateWBSQuality runs quality checks on the decomposed sub-tasks (P1-2 / Design §11.3).
// Checks: completeness, non-overlap, DAG cycle detection, and quantity control.
// Returns an error if any critical validation fails.
func validateWBSQuality(subTasks []TaskDecomposition) error {
	if len(subTasks) == 0 {
		return nil
	}

	// Check 1: Quantity control — 3-10 subtasks recommended range (Design §11.3)
	if len(subTasks) > 15 {
		return fmt.Errorf("wbs validation: too many subtasks (%d > 15), consider simplifying", len(subTasks))
	}

	// Check 2: No overlap — verify no duplicate task IDs or titles
	titleSet := make(map[string]bool)
	idSet := make(map[string]bool)
	for _, st := range subTasks {
		if idSet[st.ID] {
			return fmt.Errorf("wbs validation: duplicate task ID %q", st.ID)
		}
		idSet[st.ID] = true
		if titleSet[st.Title] {
			return fmt.Errorf("wbs validation: duplicate task title %q", st.Title)
		}
		titleSet[st.Title] = true
	}

	// Check 3: DAG cycle detection — ensure no circular dependencies
	if err := detectDAGCycle(subTasks); err != nil {
		return fmt.Errorf("wbs validation: circular dependency detected: %w", err)
	}

	// Check 4: Dependency validity — all depends_on references must exist
	for _, st := range subTasks {
		for _, dep := range st.DependsOn {
			if !idSet[dep] {
				return fmt.Errorf("wbs validation: task %q depends on unknown task %q", st.ID, dep)
			}
		}
	}

	return nil
}

// detectDAGCycle performs a topological sort to detect cycles in the dependency graph (P1-2).
// Returns an error if a cycle is found.
func detectDAGCycle(subTasks []TaskDecomposition) error {
	if len(subTasks) == 0 {
		return nil
	}

	// Build adjacency list and in-degree count
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	idSet := make(map[string]bool)

	for _, st := range subTasks {
		idSet[st.ID] = true
		if _, ok := inDegree[st.ID]; !ok {
			inDegree[st.ID] = 0
		}
	}

	for _, st := range subTasks {
		for _, dep := range st.DependsOn {
			graph[dep] = append(graph[dep], st.ID)
			inDegree[st.ID]++
		}
	}

	// Kahn's algorithm for topological sort
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	processed := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		processed++

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if processed != len(subTasks) {
		return fmt.Errorf("circular dependency among %d tasks", len(subTasks)-processed)
	}
	return nil
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

	// Compute duration from task lifecycle timestamps
	duration := computeTaskDuration(result)

	// Update progress table with success
	holder := &TaskResultHolder{
		Content:  result.Output,
		Duration: duration,
		Score:    2, // Default score (will be re-evaluated in Observe)
	}
	cs.TaskProgress.UpdateStatus(subTaskID, TaskSucceeded, WithResult(holder), WithTimestamps())

	// Store raw result
	cs.SubTaskResults[subTaskID] = &core.TaskResultEvent{
		TaskID:    subTaskID,
		Result:    result.Output,
		Duration:  duration,
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

// computeTaskDuration computes the elapsed time from task CreatedAt to UpdatedAt.
// Falls back to 0 if timestamps are not available.
func computeTaskDuration(task *core.Task) time.Duration {
	if task == nil {
		return 0
	}
	if task.UpdatedAt.IsZero() || task.CreatedAt.IsZero() {
		return 0
	}
	return task.UpdatedAt.Sub(task.CreatedAt)
}

// ===========================================================================
// executeResponsibilityGate — Orchestrates the merged gate (Step A+B, Design §5)
// =========================================================================//

// executeResponsibilityGate runs the merged responsibility+atomicity gate between
// L1 routing and L2 planning. Uses a SINGLE LLM call for both judgments.
//
// Flow:
//  1. Keyword match fast path (threshold: 2+ keyword hits or >30% desc overlap)
//  2. If fast path fails: single LLM call for responsibility + atomicity
//  3. Not our job → delegate to orchestrator
//  4. Atomic task → proceed to Level 2
//  5. Non-atomic task → validate WBS → dispatch & coordinate
func (r *Reactor) executeResponsibilityGate(ctx *ReactContext, l1Result *l1RoutingResult, tokensSoFar int) (int, error) {
	// Merged Step A+B: Single LLM call (with keyword fast path)
	respCheck, atomicity, err := r.checkResponsibilityAndAtomicity(ctx, l1Result)
	if err != nil {
		return 0, fmt.Errorf("merged gate (A+B): %w", err)
	}

	logger.Info("gate Step A: responsibility", "is_match", respCheck.IsMatch,
		"confidence", respCheck.Confidence)
	logger.Info("gate Step B: atomicity", "is_atomic", atomicity.IsAtomic,
		"subtasks", len(atomicity.SubTasks))

	if !respCheck.IsMatch {
		// Not our job → delegate to orchestrator → Coordinator mode
		delegateTokens, delErr := r.delegateToOrchestrator(ctx, l1Result, respCheck)
		return delegateTokens, delErr
	}

	if atomicity.IsAtomic {
		// Atomic task → proceed to Level 2 (normal executor path)
		return 0, nil
	}

	// WBS Quality Assurance — validate decomposition before dispatching
	if err := validateWBSQuality(atomicity.SubTasks); err != nil {
		logger.Warn("wbs quality validation failed, falling back to executor mode", "error", err)
		return 0, nil
	}

	// Non-atomic task → dispatch & coordinate
	return r.dispatchAndCoordinate(ctx, tokensSoFar, atomicity.SubTasks)
}
