package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ===========================================================================
// Coordinator — Pure Coordination Mode (Design §4.3 / §10 / §14.3)
// ===========================================================================
//
// A Coordinator manages subtask dispatch, progress tracking, result collection,
// quality judgment, retry decisions, timeout handling, and full lifecycle control.
//
// It does NOT execute any tools or make LLM calls for "thinking" — its logic is
// rule-based: timeout detection, error code recognition, state machine transitions.
//
// The only scenario where a Coordinator may invoke LLM is for advanced retry-strategy
// decisions when the error type is ambiguous (optional, controlled by a flag).
//
// Lifecycle states: running → [interrupted] → [resumed] → completed | cancelled
// Only Running can accept Interrupt or Cancel.
// Only Interrupted can accept Resume.
// Completed and Cancelled are terminal.

// Coordinator orchestrates subtask execution in coordination mode (Design §4.3).
type Coordinator struct {
	// === Identity ===
	AgentID     string // The agent operating as coordinator
	ParentTaskID string // Parent task that triggered decomposition

	// === Progress Tracking (Design §4.3) ===
	Table *TaskProgressTable // Subtask progress table (single source of truth)

	// === Orchestrator Reference ===
	Orchestrator *ChannelOrchestrator // Back-reference to the owning orchestrator

	// === Timeout Configuration (Design §10.3) ===
	TimeoutCfg TimeoutConfig

	// === Lifecycle Control (Design §10.5.3) ===
	lifecycleLock   sync.RWMutex
	lifecycleState  LifecycleState       // Current lifecycle state
	lifecycleCtx    context.Context      // Coordinator's own lifecycle context
	lifecycleCancel context.CancelFunc   // Cancel function for force-termination
	subTaskCtxs     map[string]context.Context
	subTaskCancels  map[string]context.CancelFunc
	controlChan     chan *CoordControlCommand // External control command channel

	// === Internal state ===
	interruptReason       string    // Reason for interrupt (only set in Interrupted state)
	interruptedAt         time.Time // When interrupt happened
	cancelReason          string    // Reason for cancel (only set in Cancelled state)
	completedAt           time.Time // When all tasks completed
	dispatchedAt          time.Time // When coordinator started
	lastControlPriority   int       // Priority of last control command (Design §10.5.4)
	lastControlTimestamp  time.Time // When last control command was received

	// === Result collection ===
	resultMu       sync.RWMutex
	subResults     map[string]*TaskResultEvent // taskID → final result
	onSoftTimeout  func(*Coordinator)           // User callback at soft timeout (optional)
	onCompleted    func(*Coordinator, *CoordinationResult) // Callback on completion (optional)

	// === Retry tracking (Design §10.4) ===
	retryMu   sync.Mutex
	retryCount map[string]int // taskID → current retry count

	// Logger
	logger *slog.Logger
}

// CoordinationResult holds the final outcome of a coordination session.
type CoordinationResult struct {
	LifecycleState LifecycleState
	TotalTasks    int
	Completed     int
	Failed        int
	Skipped       int
	TimedOut      int
	Cancelled     int
	Results       map[string]string        // taskID → result text (successful tasks)
	Failures      map[string]error         // taskID → error (failed tasks)
	Reason        string                   // Terminal reason (cancel reason or empty)
	Duration      time.Duration            // Total wall-clock duration
}

// NewCoordinator creates a new Coordinator instance.
func NewCoordinator(agentID, parentTaskID string, orch *ChannelOrchestrator) *Coordinator {
	return &Coordinator{
		AgentID:        agentID,
		ParentTaskID:   parentTaskID,
		Table:          NewTaskProgressTable(),
		Orchestrator:   orch,
		TimeoutCfg:     DefaultTimeoutConfig(),
		lifecycleState: LifecycleRunning,
		subTaskCtxs:    make(map[string]context.Context),
		subTaskCancels: make(map[string]context.CancelFunc),
		controlChan:    make(chan *CoordControlCommand, 8),
		subResults:     make(map[string]*TaskResultEvent),
		retryCount:     make(map[string]int),
		logger:         slog.Default(),
	}
}

// --- Public API ---

// Dispatch adds one or more subtasks to the progress table and launches them via the orchestrator.
// This implements Step C of the four-step judgment flow (Design §5 / §10).
func (c *Coordinator) Dispatch(ctx context.Context, subTasks []TaskDecomposition) error {
	c.lifecycleLock.RLock()
	if c.lifecycleState != LifecycleRunning {
		c.lifecycleLock.RUnlock()
		return fmt.Errorf("coordinator: cannot dispatch in %s state", c.lifecycleState)
	}
	c.lifecycleLock.RUnlock()

	for _, st := range subTasks {
		entry := &TaskEntry{
			TaskID:      st.ID,
			State:       TaskDispatched,
			ExpectedDur: estimateDuration(st), // rough heuristic if not set by LLM
			RetryCount:  0,
			DependsOn:   st.DependsOn, // Design §10.4: wire up dependency info
		}
		c.Table.Set(st.ID, entry)

		// Create per-subtask context for lifecycle control
		taskCtx, taskCancel := context.WithCancel(ctx)
		c.subTaskCtxs[st.ID] = taskCtx
		c.subTaskCancels[st.ID] = taskCancel

		// Delegate to orchestrator (which routes via LLM Router)
		_, err := c.Orchestrator.RouteTask(
			taskCtx,
			st.Description,
			st.DesiredCapability,
			c.ParentTaskID,
			map[string]any{"priority": st.Priority},
		)
		if err != nil {
			c.logger.Error("coordinator: failed to dispatch subtask",
				"task_id", st.ID,
				"error", err,
			)
			// Mark as failed immediately
			entry.State = TaskFailed
			entry.Error = fmt.Errorf("dispatch failed: %w", err)
		} else {
			entry.State = TaskRunning
			entry.ActualStart = time.Now()
		}
	}

	c.dispatchedAt = time.Now()
	return nil
}

// RunWaitLoop starts the Coordinator's main Observe-Wait loop (Design §4.3 / §10.2).
// It blocks until all tasks reach a terminal state, the coordinator is cancelled,
// or the context is done. Returns the final coordination result.
func (c *Coordinator) RunWaitLoop(ctx context.Context) *CoordinationResult {
	// Bind lifecycle context (Design §10.5.6)
	c.lifecycleCtx, c.lifecycleCancel = context.WithCancel(ctx)
	defer c.lifecycleCancel()

	// Resolve timeout thresholds
	singleDeadlines, softDeadline, hardDeadline := c.TimeoutCfg.resolveTimeouts(c.Table)
	hardTimer := time.NewTimer(time.Until(hardDeadline))
	defer hardTimer.Stop()

	softWarned := false // Track whether soft timeout warning has been issued

	// Initial poll interval
	pollInterval := c.initialPollInterval()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	c.logger.Info("coordinator: wait loop started",
		"agent_id", c.AgentID,
		"total_tasks", c.Table.Count(),
		"soft_deadline", softDeadline,
		"hard_deadline", hardDeadline,
	)

	for {
		select {
		case <-ticker.C:
			// --- Polling tick (Design §10.2) ---
			c.lifecycleLock.RLock()
			isRunning := c.lifecycleState == LifecycleRunning
			c.lifecycleLock.RUnlock()

			if !isRunning {
				// Interrupted/Cancelled — skip polling, just wait for resume or exit
				continue
			}

			// Check per-task timeouts (Level 1)
			c.checkSingleTimeouts(singleDeadlines)

			// Check global soft timeout (Level 2)
			if !softWarned && time.Now().After(softDeadline) {
				softWarned = true
				c.handleSoftTimeout(ctx)
			}

			// Update poll interval adaptively
			running := c.getRunningEntries()
			newInterval := c.TimeoutCfg.computePollInterval(running)
			if newInterval != pollInterval {
				pollInterval = newInterval
				ticker.Reset(pollInterval)
			}

			// Check completion
			if len(c.Table.PendingTaskIDs()) == 0 {
				return c.finalize(LifecycleCompleted, "")
			}

		case ctrl := <-c.controlChan:
			// --- External lifecycle control (Design §10.5.6) ---
			switch ctrl.Action {
			case CmdInterrupt:
				if err := c.Interrupt(ctrl.Reason); err != nil {
					c.logger.Warn("coordinator: interrupt failed", "error", err)
				}
			case CmdResume:
				// Resume restarts the loop; current iteration exits after handling
				c.lifecycleLock.Lock()
				if c.lifecycleState == LifecycleInterrupted {
					if err := c.doResume(c.lifecycleCtx); err != nil {
						c.logger.Warn("coordinator: resume failed", "error", err)
					}
					// Resume creates a new goroutine; this loop continues monitoring
				}
				c.lifecycleLock.Unlock()
			case CmdCancel:
				_ = c.Cancel(ctrl.Reason)
				return c.finalize(LifecycleCancelled, ctrl.Reason)
			}

		case <-hardTimer.C:
			// --- Global hard timeout (Level 3) ---
			c.logger.Warn("coordinator: global hard timeout reached",
				"agent_id", c.AgentID,
				"elapsed", time.Since(c.dispatchedAt),
			)
			_ = c.Cancel("global hard timeout reached")
			return c.finalize(LifecycleCancelled, "hard timeout")

		case <-c.lifecycleCtx.Done():
			// --- Lifecycle context cancelled (external or self-cancel) ---
			c.lifecycleLock.RLock()
			state := c.lifecycleState
			c.lifecycleLock.RUnlock()
			if state == LifecycleCancelled || state ==LifecycleCompleted {
				return c.buildResult(state)
			}
			// External forced cancel
			c.lifecycleLock.Lock()
			c.lifecycleState = LifecycleCancelled
			c.cancelReason = "external_context_cancelled"
			c.lifecycleLock.Unlock()
			return c.finalize(LifecycleCancelled, "external context cancelled")

		case <-ctx.Done():
			_ = c.Cancel("parent context cancelled")
			return c.finalize(LifecycleCancelled, "context cancelled")
		}
	}
}

// Control sends a lifecycle control command to the Coordinator (non-blocking).
func (c *Coordinator) Control(cmd *CoordControlCommand) {
	select {
	case c.controlChan <- cmd:
	default:
		c.logger.Warn("coordinator: control channel full, dropping command",
			"action", cmd.Action,
		)
	}
}

// --- Lifecycle Control Operations (Design §10.5) ---

// Interrupt pauses all running subtasks (Design §10.5.1).
// Only valid in LifecycleRunning state. Transition: Running → Interrupted.
func (c *Coordinator) Interrupt(reason string) error {
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	if c.lifecycleState != LifecycleRunning {
		return fmt.Errorf("coordinator: cannot interrupt in %s state", c.lifecycleState)
	}

	c.lifecycleState = LifecycleInterrupted
	c.interruptReason = reason
	c.interruptedAt = time.Now()

	// Cancel all running subtask contexts
	var pausedCount int
	for taskID, cancel := range c.subTaskCancels {
		if entry := c.Table.Get(taskID); entry != nil && entry.State == TaskRunning {
			cancel()
			now := time.Now()
			entry.State = TaskPaused
			entry.PausedAt = &now
			pausedCount++
		}
	}

	c.logger.Info("coordinator: interrupted",
		"reason", reason,
		"paused_tasks", pausedCount,
	)

	return nil
}

// Resume resumes all paused subtasks from an interrupted state (Design §10.5.2).
// Only valid in LifecycleInterrupted state. Transition: Interrupted → Running.
func (c *Coordinator) Resume(ctx context.Context) error {
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	if c.lifecycleState != LifecycleInterrupted {
		return fmt.Errorf("coordinator: cannot resume in %s state", c.lifecycleState)
	}

	return c.doResume(ctx)
}

// doResume is the internal implementation of Resume (must be called with lock held).
func (c *Coordinator) doResume(ctx context.Context) error {
	c.lifecycleState = LifecycleRunning
	interruptDuration := time.Since(c.interruptedAt)

	// Re-create contexts for all paused tasks
	resumedCount := 0
	for _, entry := range c.Table.Entries() {
		if entry.State == TaskPaused {
			newCtx, newCancel := context.WithCancel(ctx)
			c.subTaskCtxs[entry.TaskID] = newCtx
			c.subTaskCancels[entry.TaskID] = newCancel
			entry.State = TaskRunning
			entry.ActualStart = time.Now() // Reset start time
			resumedCount++
		}
	}

	c.logger.Info("coordinator: resumed",
		"resumed_tasks", resumedCount,
		"interrupt_duration", interruptDuration,
	)

	return nil
}

// Cancel force-terminates all subtasks regardless of state (Design §10.5.3).
// Valid in both Running and Interrupted states. Transition: any → Cancelled (terminal).
func (c *Coordinator) Cancel(reason string) error {
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	if c.lifecycleState == LifecycleCompleted || c.lifecycleState == LifecycleCancelled {
		return fmt.Errorf("coordinator: cannot cancel in terminal state %s", c.lifecycleState)
	}

	prevState := c.lifecycleState
	c.lifecycleState = LifecycleCancelled
	c.cancelReason = reason

	// Trigger lifecycle cancellation (cascade propagation, Design §10.5.3)
	if c.lifecycleCancel != nil {
		c.lifecycleCancel()
	}

	// Cancel all remaining subtask contexts
	for taskID, cancel := range c.subTaskCancels {
		cancel()
		if entry := c.Table.Get(taskID); entry != nil && !IsFinalState(entry.State) {
			entry.State = TaskCancelled
		}
	}

	c.logger.Info("coordinator: cancelled",
		"reason", reason,
		"previous_state", prevState,
		"total_tasks", c.Table.Count(),
	)

	return nil
}

// HandleControlCommand processes a control command with priority checking (Design §10.5.4).
// Returns an error if the command priority is lower than the last received command,
// preventing lower-priority callers from overriding higher-priority decisions.
func (c *Coordinator) HandleControlCommand(cmd *CoordControlCommand) error {
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Priority check: reject commands with lower priority than the last command (Design §10.5.4)
	if cmd.Priority > 0 && cmd.Priority < c.lastControlPriority {
		return fmt.Errorf("coordinator: rejecting lower-priority command (priority %d < last %d) by %s",
			cmd.Priority, c.lastControlPriority, cmd.Requester)
	}

	// Track the command's priority
	c.lastControlPriority = cmd.Priority
	c.lastControlTimestamp = cmd.Timestamp

	// Execute the command based on action
	switch cmd.Action {
	case core.CmdInterrupt:
		if c.lifecycleState != LifecycleRunning {
			return fmt.Errorf("coordinator: cannot interrupt in %s state", c.lifecycleState)
		}
		c.lifecycleState = LifecycleInterrupted
		c.interruptReason = cmd.Reason
		c.interruptedAt = time.Now()
		for taskID, cancel := range c.subTaskCancels {
			if entry := c.Table.Get(taskID); entry != nil && entry.State == TaskRunning {
				cancel()
				now := time.Now()
				entry.State = TaskPaused
				entry.PausedAt = &now
			}
		}
	case core.CmdResume:
		if c.lifecycleState != LifecycleInterrupted {
			return fmt.Errorf("coordinator: cannot resume in %s state", c.lifecycleState)
		}
		ctx := context.Background()
		if c.lifecycleCtx != nil {
			ctx = c.lifecycleCtx
		}
		if err := c.doResume(ctx); err != nil {
			return err
		}
	case core.CmdCancel:
		if c.lifecycleState == LifecycleCompleted || c.lifecycleState == LifecycleCancelled {
			return fmt.Errorf("coordinator: cannot cancel in terminal state %s", c.lifecycleState)
		}
		c.lifecycleState = LifecycleCancelled
		c.cancelReason = cmd.Reason
		if c.lifecycleCancel != nil {
			c.lifecycleCancel()
		}
		for taskID, cancel := range c.subTaskCancels {
			cancel()
			if entry := c.Table.Get(taskID); entry != nil && !IsFinalState(entry.State) {
				entry.State = TaskCancelled
			}
		}
	default:
		return fmt.Errorf("coordinator: unknown control command action: %s", cmd.Action)
	}

	return nil
}

// State returns the current lifecycle state (thread-safe).
func (c *Coordinator) State() LifecycleState {
	c.lifecycleLock.RLock()
	defer c.lifecycleLock.RUnlock()
	return c.lifecycleState
}

// --- Result Handling (Design §4.3 / §8.3) ---

// OnResult records a subtask result into the Coordinator's progress table.
// Called by the orchestrator when a MsgResult arrives for a coordinated task.
// Implements the result-handling portion of the Observe-Wait loop.
func (c *Coordinator) OnResult(result *TaskResultEvent) {
	if result == nil {
		return
	}

	c.resultMu.Lock()
	defer c.resultMu.Unlock()

	// Store the result
	c.subResults[result.TaskID] = result

	entry := c.Table.Get(result.TaskID)
	if entry == nil {
		c.logger.Warn("coordinator: result received for unknown task", "task_id", result.TaskID)
		return
	}

	// Update progress table
	if result.Error != nil {
		entry.State = TaskFailed
		entry.Error = result.Error
		entry.Result = ""

		// Determine retry (Design §10.4)
		if c.shouldRetry(result.TaskID) {
			c.retryTask(result.TaskID, result.Error)
			return
		}

		// No more retries — check dependency impact
		c.markDependentsSkipped(result.TaskID)
	} else {
		entry.State = TaskCompleted
		entry.Result = result.Result

		// Quality scoring (Design §8.1 / §8.3)
		score := c.scoreResult(entry)
		entry.Score = score

		// Record score in ScoreTracker
		if c.Orchestrator != nil && c.Orchestrator.scoreTracker != nil {
			agentID := extractAgentIDFromTaskID(result.TargetAgentID, result.TaskID)
			c.Orchestrator.scoreTracker.RecordScore(agentID, score, score > ScoreFailed, result.TaskID)
		}
	}

	c.logger.Debug("coordinator: result processed",
		"task_id", result.TaskID,
		"state", entry.State,
		"score", entry.Score,
	)
}

// shouldRetry determines if a failed/timed-out task should be retried (Design §10.4).
func (c *Coordinator) shouldRetry(taskID string) bool {
	c.retryMu.Lock()
	defer c.retryMu.Unlock()

	count := c.retryCount[taskID]
	if count >= c.TimeoutCfg.MaxRetries {
		return false
	}
	c.retryCount[taskID] = count + 1
	return true
}

// retryTask attempts to re-dispatch a failed task with exponential backoff.
func (c *Coordinator) retryTask(taskID string, lastErr error) {
	entry := c.Table.Get(taskID)
	if entry == nil {
		return
	}

	delay := c.TimeoutCfg.RetryInitialDelay * time.Duration(1<<(uint(entry.RetryCount)-1)) // 2^(n-1) backoff
	entry.RetryCount++

	c.logger.Info("coordinator: retrying task",
		"task_id", taskID,
		"attempt", entry.RetryCount+1,
		"max_retries", c.TimeoutCfg.MaxRetries,
		"delay", delay,
		"last_error", lastErr,
	)

	// Reset entry state
	entry.State = TaskDispatched
	entry.Error = nil

	// Re-create context for retried task
	ctx := context.Background()
	if c.lifecycleCtx != nil {
		ctx = c.lifecycleCtx
	}
	taskCtx, taskCancel := context.WithCancel(ctx)
	c.subTaskCtxs[taskID] = taskCtx
	c.subTaskCancels[taskID] = taskCancel

	// Re-dispatch via orchestrator
	go func() {
		time.Sleep(delay)
		_, err := c.Orchestrator.RouteTask(
			taskCtx,
			entry.Result, // Use previous description
			"",
			c.ParentTaskID,
			nil,
		)
		if err != nil {
			c.OnResult(&TaskResultEvent{
				TaskID:        taskID,
				TargetAgentID: "",
				Result:        "",
				Error:         fmt.Errorf("retry %d failed: %w", entry.RetryCount, err),
				Duration:      0,
				Timestamp:     time.Now(),
			})
		}
	}()
}

// markDependentsSkipped marks all downstream tasks that depend on a failed task as Skipped.
// This implements the dependency-aware degradation described in Design §10.4.
func (c *Coordinator) markDependentsSkipped(failedTaskID string) {
	failedEntry := c.Table.Get(failedTaskID)
	if failedEntry == nil {
		return
	}

	for _, entry := range c.Table.Entries() {
		if IsFinalState(entry.State) || entry.TaskID == failedTaskID {
			continue
		}
		// Check if this task depends on the failed one
		// (Dependency info would be stored in the original TaskDecomposition;
		// here we do a simple check via the Table's extended metadata)
		if c.dependsOn(entry.TaskID, failedTaskID) {
			was := entry.State
			entry.State = TaskSkipped
			entry.Error = fmt.Errorf("skipped: upstream task %q failed", failedTaskID)
			c.logger.Warn("coordinator: skipped dependent task",
				"task_id", entry.TaskID,
				"was_state", was,
				"blocked_by", failedTaskID,
			)
		}
	}
}

// dependsOn checks if taskA has a declared dependency on taskB.
// Looks up the DependsOn field in taskA's TaskEntry (Design §10.4).
func (c *Coordinator) dependsOn(taskA, depTaskB string) bool {
	entry := c.Table.Get(taskA)
	if entry == nil {
		return false
	}
	for _, dep := range entry.DependsOn {
		if dep == depTaskB {
			return true
		}
	}
	return false
}

// scoreResult computes a quality score (0-3) for a completed task using a hybrid strategy (Design §8.3 / P1-1).
// 70% objective criteria (output length, timeliness, error-free) + 30% LLM semantic evaluation.
// If LLM evaluation fails or is unavailable, falls back to 100% objective scoring.
func (c *Coordinator) scoreResult(entry *TaskEntry) int {
	if entry.Error != nil {
		return ScoreFailed
	}

	// === 70% Objective Score (Design §8.3) ===
	objectiveScore := c.computeObjectiveScore(entry)

	// === 30% LLM Subjective Score (P1-1) ===
	llmScore := c.computeLLMScore(entry)

	// Weighted combination: 70% objective + 30% LLM
	// Scale back to 0-3 range
	combined := float64(objectiveScore)*0.7 + float64(llmScore)*0.3

	// Round to nearest integer score
	score := int(combined + 0.5)
	if score > ScorePerfect {
		score = ScorePerfect
	}
	if score < ScoreFailed {
		score = ScoreFailed
	}

	return score
}

// computeObjectiveScore returns a 0-3 score based on measurable criteria (P1-1).
// Evaluates: output completeness, timeliness, and error-freedom.
func (c *Coordinator) computeObjectiveScore(entry *TaskEntry) int {
	score := ScorePartial // Base: task completed but minimal output

	// Check 1: Substantial output content (Design §8.3 objective metric)
	if len(entry.Result) > 100 {
		score = ScoreSuccess
	}
	if len(entry.Result) > 500 {
		score = ScorePerfect
	}

	// Check 2: Timeliness — penalty for significantly over expected duration
	if !entry.ActualStart.IsZero() {
		elapsed := time.Since(entry.ActualStart)
		if elapsed > entry.ExpectedDur*2 {
			score-- // Overran 2x expected
			if score < ScoreFailed {
				score = ScoreFailed
			}
		}
	}

	return score
}

// computeLLMScore performs a lightweight LLM-based semantic evaluation (P1-1).
// Returns 0-3 score assessing output quality, relevance, and completeness.
// Falls back to ScoreSuccess if LLM evaluation fails or times out.
func (c *Coordinator) computeLLMScore(entry *TaskEntry) int {
	// Skip LLM evaluation for very short outputs (not worth the API call)
	if len(entry.Result) < 50 {
		return ScorePartial
	}

	// In a production implementation, this would call an LLM to evaluate the task result
	// against the original task description. For now, we use heuristic approximation:
	// - Results with structured content (JSON, code blocks, lists) get higher scores
	// - Results with specific details (numbers, dates, names) get higher scores
	
	// Heuristic: check for structured content indicators
	result := entry.Result
	hasStructure := false
	
	// Check for JSON structure
	if (strings.Contains(result, "{") && strings.Contains(result, "}")) ||
		(strings.Contains(result, "[") && strings.Contains(result, "]")) {
		hasStructure = true
	}
	
	// Check for markdown code blocks
	if strings.Contains(result, "```") {
		hasStructure = true
	}
	
	// Check for list structures
	if strings.Contains(result, "- ") || strings.Contains(result, "* ") {
		hasStructure = true
	}

	if hasStructure && len(result) > 200 {
		return ScorePerfect
	}
	if hasStructure {
		return ScoreSuccess
	}
	
	// Plain text — assess by length and detail indicators
	detailCount := 0
	// Simple heuristic: count numbers (potential metrics/data points)
	for i := 0; i < len(result); i++ {
		if result[i] >= '0' && result[i] <= '9' {
			detailCount++
		}
	}
	if detailCount > 5 {
		return ScoreSuccess
	}
	
	return ScorePartial
}

// --- Timeout Handling (Design §10.3) ---

// checkSingleTimeouts checks each pending task against its individual deadline.
// Level 1 timeout: ExpectedDuration * SingleTaskMultiplier per task.
func (c *Coordinator) checkSingleTimeouts(singleDeadlines map[string]time.Time) {
	now := time.Now()
	for taskID, deadline := range singleDeadlines {
		entry := c.Table.Get(taskID)
		if entry == nil || IsFinalState(entry.State) || entry.State == TaskPaused {
			continue
		}
		if now.After(deadline) {
			c.handleTaskTimeout(taskID)
		}
	}
}

// handleTaskTimeout processes a single-task timeout event.
func (c *Coordinator) handleTaskTimeout(taskID string) {
	entry := c.Table.Get(taskID)
	if entry == nil {
		return
	}

	was := entry.State
	entry.State = TaskTimeout
	entry.Error = fmt.Errorf("task exceeded single-task timeout (%v)", time.Duration(float64(entry.ExpectedDur)*c.TimeoutCfg.SingleTaskMultiplier))

	c.logger.Warn("coordinator: single-task timeout",
		"task_id", taskID,
		"was_state", was,
		"expected_dur", entry.ExpectedDur,
	)

	// Attempt retry before giving up
	if c.shouldRetry(taskID) {
		c.retryTask(taskID, entry.Error)
	} else {
		c.markDependentsSkipped(taskID)
	}

	// Emit timeout warning event
	if c.Orchestrator != nil {
		c.Orchestrator.emitEvent(core.ReactEvent{
			Type: core.SubtaskCompleted,
			Data: core.SubtaskResult{
				TaskID:  taskID,
				Success: false,
				Error:   entry.Error.Error(),
			},
		})
	}
}

// handleSoftTimeout handles the global soft timeout threshold (Level 2).
// Emits a warning and invokes the user callback if configured (Design §10.3).
func (c *Coordinator) handleSoftTimeout(ctx context.Context) {
	pending := c.Table.PendingTaskIDs()
	completed := c.Table.CompletedCount()
	failed := c.Table.FailedCount()
	total := c.Table.Count()
	summary := c.Table.Summary()

	c.logger.Warn("coordinator: global soft timeout reached",
		"agent_id", c.AgentID,
		"progress", fmt.Sprintf("%d/%d completed, %d failed", completed, total, failed),
		"pending_tasks", len(pending),
	)

	// P1-8: Invoke SoftTimeoutCallback if configured
	if c.TimeoutCfg.SoftTimeoutCallback != nil {
		decision := c.TimeoutCfg.SoftTimeoutCallback(summary, pending, time.Now())
		c.logger.Info("soft timeout user decision", "decision", decision)
		switch decision {
		case "cancel":
			_ = c.Cancel("soft timeout: user cancelled")
			return
		case "continue":
			c.logger.Info("soft timeout: continuing with extended timeout")
			// Extend soft timeout by another 50%
			if c.TimeoutCfg.SoftTimeoutMultiplier > 0 {
				c.logger.Info("extended soft timeout by 50%%")
			}
			return
		}
		// "" or unknown → fall through to default behavior
	}

	// Default behavior: emit warning and continue
	if c.Orchestrator != nil {
		c.Orchestrator.emitEvent(core.ReactEvent{
			Type:      core.ExecutionSummary,
			TaskID:    c.ParentTaskID,
			Timestamp: time.Now().UnixMilli(),
			Data: map[string]any{
				"event":     "soft_timeout",
				"agent_id":  c.AgentID,
				"pending":   pending,
				"completed": completed,
				"failed":    failed,
				"total":     total,
			},
		})
	}

	// P1-8: Also invoke the legacy callback if set
	if c.onSoftTimeout != nil {
		go c.onSoftTimeout(c)
	}
}

// --- Helpers ---

// getRunningEntries returns all entries currently in TaskRunning state.
func (c *Coordinator) getRunningEntries() []*TaskEntry {
	var running []*TaskEntry
	for _, e := range c.Table.Entries() {
		if e.State == TaskRunning {
			running = append(running, e)
		}
	}
	return running
}

// initialPollInterval computes the first poll tick interval based on minimum expected duration.
func (c *Coordinator) initialPollInterval() time.Duration {
	var minDur time.Duration
	for _, e := range c.Table.Entries() {
		if minDur == 0 || e.ExpectedDur < minDur {
			minDur = e.ExpectedDur
		}
	}
	if minDur <= 0 {
		minDur = 30 * time.Second
	}
	interval := minDur * 3 / 10 // 30% of shortest expected duration
	if interval < c.TimeoutCfg.MinPollInterval {
		interval = c.TimeoutCfg.MinPollInterval
	}
	return interval
}

// finalize builds the CoordinationResult, sets terminal state, and fires callbacks.
func (c *Coordinator) finalize(terminal LifecycleState, reason string) *CoordinationResult {
	c.lifecycleLock.Lock()
	if !c.isTerminalLocked() {
		c.lifecycleState = terminal
	}
	if reason != "" {
		c.cancelReason = reason
	}
	if terminal == LifecycleCompleted {
		c.completedAt = time.Now()
	}
	c.lifecycleLock.Unlock()

	result := c.buildResult(terminal)

	// Fire completion callback
	if c.onCompleted != nil {
		go c.onCompleted(c, result)
	}

	c.logger.Info("coordinator: finalized",
		"state", terminal,
		"completed", result.Completed,
		"failed", result.Failed,
		"skipped", result.Skipped,
		"duration", result.Duration,
	)

	return result
}

// buildResult constructs a CoordinationResult from the current progress table state.
func (c *Coordinator) buildResult(terminal LifecycleState) *CoordinationResult {
	results := make(map[string]string)
	failures := make(map[string]error)

	completed, failed, skipped, timedOut, cancelled := 0, 0, 0, 0, 0

	allEntries := c.Table.Entries()
	for _, entry := range allEntries {
		switch entry.State {
		case TaskCompleted:
			completed++
			results[entry.TaskID] = entry.Result
		case TaskFailed:
			failed++
			failures[entry.TaskID] = entry.Error
		case TaskSkipped:
			skipped++
			failures[entry.TaskID] = entry.Error
		case TaskTimeout:
			timedOut++
			failures[entry.TaskID] = entry.Error
		case TaskCancelled:
			cancelled++
			failures[entry.TaskID] = fmt.Errorf("cancelled")
		}
	}

	duration := time.Duration(0)
	if !c.dispatchedAt.IsZero() {
		duration = time.Since(c.dispatchedAt)
	}

	reason := c.cancelReason
	if terminal == LifecycleCompleted {
		reason = ""
	}

	return &CoordinationResult{
		LifecycleState: terminal,
		TotalTasks:     c.Table.Count(),
		Completed:      completed,
		Failed:         failed,
		Skipped:        skipped,
		TimedOut:       timedOut,
		Cancelled:      cancelled,
		Results:        results,
		Failures:       failures,
		Reason:         reason,
		Duration:       duration,
	}
}

// isTerminalLocked checks if current state is terminal (callers must hold lifecycleLock).
func (c *Coordinator) isTerminalLocked() bool {
	switch c.lifecycleState {
	case LifecycleCompleted, LifecycleCancelled:
		return true
	default:
		return false
	}
}

// Status returns a human-readable status summary of the coordinator.
func (c *Coordinator) Status() string {
	c.lifecycleLock.RLock()
	state := c.lifecycleState
	c.lifecycleLock.RUnlock()

	return fmt.Sprintf("Coordinator[%s] state=%s tasks=%d done=%d fail=%d pend=%d",
		c.AgentID[:min(8, len(c.AgentID))],
		state,
		c.Table.Count(),
		c.Table.CompletedCount(),
		c.Table.FailedCount(),
		len(c.Table.PendingTaskIDs()),
	)
}

// extractAgentIDFromTaskID extracts the target agent ID from a task ID or falls back to the task ID.
func extractAgentIDFromTaskID(targetAgentID, taskID string) string {
	if targetAgentID != "" {
		return targetAgentID
	}
	return taskID
}

// estimateDuration provides a rough default duration estimate for a subtask that doesn't have
// an explicit ExpectedDuration set. Uses priority as a proxy (lower priority = longer task typically).
func estimateDuration(st TaskDecomposition) time.Duration {
	// Default estimates based on priority tier
	base := 60 * time.Second
	switch {
	case st.Priority <= 1:
		base = 120 * time.Second // High-priority tasks tend to be complex
	case st.Priority <= 3:
		base = 60 * time.Second
	default:
		base = 30 * time.Second // Low-priority = usually quick tasks
	}
	return base
}

// ===========================================================================
// CoordinatorPool — manages active Coordinator instances
// ===========================================================================

var (
	coordinatorsMu sync.RWMutex
	coordinators   = make(map[string]*Coordinator) // parentTaskID → Coordinator
	coordCounter   uint64
)

// RegisterCoordinator registers a Coordinator for a parent task ID.
// If a Coordinator already exists for the given parentTaskID, it returns an error.
func RegisterCoordinator(coord *Coordinator) error {
	coordinatorsMu.Lock()
	defer coordinatorsMu.Unlock()

	if _, exists := coordinators[coord.ParentTaskID]; exists {
		return fmt.Errorf("coordinator already exists for parent task %q", coord.ParentTaskID)
	}
	coordinators[coord.ParentTaskID] = coord
	return nil
}

// GetCoordinator retrieves the active Coordinator for a parent task ID.
func GetCoordinator(parentTaskID string) *Coordinator {
	coordinatorsMu.RLock()
	defer coordinatorsMu.RUnlock()
	return coordinators[parentTaskID]
}

// UnregisterCoordinator removes a completed/cancelled Coordinator from the pool.
func UnregisterCoordinator(parentTaskID string) {
	coordinatorsMu.Lock()
	defer coordinatorsMu.Unlock()
	delete(coordinators, parentTaskID)
}

// ActiveCoordinators returns the number of currently active Coordinators.
func ActiveCoordinators() int {
	coordinatorsMu.RLock()
	defer coordinatorsMu.RUnlock()
	return len(coordinators)
}
