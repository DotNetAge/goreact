// Package reactor implements GoReAct's T-A-O execution engine.
//
// This file (coordination.go) adds the dual-mode framework (Executor/Coordinator)
// and lifecycle control capabilities as defined in the "Role-based Multi-Agent
// Orchestration Model" design document.
package reactor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ===========================================================================
// Agent Mode — Executor / Coordinator (mutually exclusive)
// ===========================================================================

// AgentMode represents the current operational mode of an agent.
type AgentMode string

const (
	// ModeExecutor is the default mode: runs T-A-O loop to complete atomic tasks.
	ModeExecutor AgentMode = "executor"

	// ModeCoordinator is the coordination mode: collects/monitors/rates sub-task results.
	// Entered after WBS decomposition, exits when all sub-tasks complete.
	ModeCoordinator AgentMode = "coordinator"
)

// String returns a readable representation of the mode.
func (m AgentMode) String() string {
	return string(m)
}

// IsExecutor returns true if the mode is executor.
func (m AgentMode) IsExecutor() bool { return m == ModeExecutor }

// IsCoordinator returns true if the mode is coordinator.
func (m AgentMode) IsCoordinator() bool { return m == ModeCoordinator }

// ===========================================================================
// Lifecycle State — Coordinator lifecycle management
// ===========================================================================

// LifecycleState represents the current state of a Coordinator's lifecycle.
type LifecycleState string

const (
	LifecycleRunning    LifecycleState = "running"     // Actively coordinating
	LifecycleInterrupted LifecycleState = "interrupted" // Paused by external command
	LifecycleCancelled  LifecycleState = "cancelled"   // Terminated (terminal)
	LifecycleCompleted  LifecycleState = "completed"   // All tasks done (terminal)
)

// IsTerminal returns true if the lifecycle state cannot transition further.
func (s LifecycleState) IsTerminal() bool {
	return s == LifecycleCancelled || s == LifecycleCompleted
}

// CanTransitionTo checks if a transition to target state is valid per the state machine.
func (from LifecycleState) CanTransitionTo(to LifecycleState) bool {
	switch from {
	case LifecycleRunning:
		return to == LifecycleInterrupted || to == LifecycleCancelled || to == LifecycleCompleted
	case LifecycleInterrupted:
		return to == LifecycleRunning /*resume*/ || to == LifecycleCancelled
	default:
		return false // Terminal states don't transition
	}
}

// ===========================================================================
// Task Progress Table — Coordinator's core data structure
// ===========================================================================

// TaskStatus represents the status of a single sub-task in progress tracking.
type TaskStatus string

const (
	TaskDispatched  TaskStatus = "dispatched"   // Sent to orchestrator
	TaskAssigned    TaskStatus = "assigned"     // Orchestrator assigned an executor
	TaskRunning     TaskStatus = "running"      // Executor is working on it
	TaskSucceeded   TaskStatus = "succeeded"    // Completed successfully
	TaskFailed      TaskStatus = "failed"       // Execution failed
	TaskTimedOut    TaskStatus = "timed_out"    // Exceeded time limit
	TaskCancelled   TaskStatus = "cancelled"    // Cancelled by lifecycle control
	TaskRetryPending TaskStatus = "retry_pending" // Awaiting retry
)

// TaskEntry tracks a single sub-task's status within a Coordinator's scope.
type TaskEntry struct {
	TaskID          string        // Sub-task unique ID
	Title           string        // Human-readable title
	Description     string        // Detailed task description
	Priority        int           // Priority (lower = higher)
	Status          TaskStatus    // Current status
	Result          *TaskResultHolder // Final result (nil until completed)
	Error           error         // Error if failed
	RetryCount      int           // Number of retries attempted
	MaxRetries      int           // Maximum allowed retries
	DispatchedAt    *time.Time    // When dispatched
	StartedAt       *time.Time    // When execution started
	CompletedAt     *time.Time    // When finished
	PausedAt        *time.Time    // When paused (for interrupt/resume)
	Duration        time.Duration // Total wall-clock duration
}

// TaskResultHolder wraps the result content from a completed sub-task.
type TaskResultHolder struct {
	Content  string        // Result text
	AgentID  string        // Executor agent ID
	Duration time.Duration // How long execution took
	Score    int           // Quality score (0-3)
}

// IsTerminal checks if this task entry has reached a terminal state.
func (e *TaskEntry) IsTerminal() bool {
	switch e.Status {
	case TaskSucceeded, TaskFailed, TaskTimedOut, TaskCancelled:
		return true
	default:
		return false
	}
}

// IsCompletedSuccessfully returns true if the task succeeded.
func (e *TaskEntry) IsCompletedSuccessfully() bool { return e.Status == TaskSucceeded }

// CanRetry returns true if the task can be retried.
func (e *TaskEntry) CanRetry() bool {
	return e.Status == TaskFailed && e.RetryCount < e.MaxRetries
}

// TaskProgressTable is the Coordinator's primary state structure for tracking sub-tasks.
// It provides thread-safe access to all sub-task entries and supports lifecycle queries.
type TaskProgressTable struct {
	mu       sync.RWMutex
	entries  map[string]*TaskEntry // key: task ID
	parentID string                // Parent task ID
	order    []string              // Insertion order for deterministic iteration
}

// NewTaskProgressTable creates a new progress table for a given parent task.
func NewTaskProgressTable(parentTaskID string) *TaskProgressTable {
	return &TaskProgressTable{
		entries:  make(map[string]*TaskEntry),
		parentID: parentTaskID,
		order:    make([]string, 0),
	}
}

// Add registers a new sub-task with initial DISPATCHED status.
func (t *TaskProgressTable) Add(entry *TaskEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.entries[entry.TaskID]; !exists {
		t.order = append(t.order, entry.TaskID)
	}
	t.entries[entry.TaskID] = entry
}

// Get retrieves a task entry by ID (returns copy).
func (t *TaskProgressTable) Get(taskID string) *TaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	e, ok := t.entries[taskID]
	if !ok {
		return nil
	}
	cp := *e
	return &cp
}

// UpdateStatus changes a task's status and optionally sets related fields.
func (t *TaskProgressTable) UpdateStatus(taskID string, status TaskStatus, opts ...TaskUpdateOption) {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[taskID]
	if !ok {
		return
	}

	e.Status = status
	now := time.Now()
	for _, opt := range opts {
		opt(e, now)
	}
}

// TaskUpdateOption is a functional option for UpdateStatus.
type TaskUpdateOption func(*TaskEntry, time.Time)

// WithResult sets the result content on update.
func WithResult(result *TaskResultHolder) TaskUpdateOption {
	return func(e *TaskEntry, _ time.Time) { e.Result = result }
}

// WithError sets the error on update.
func WithError(err error) TaskUpdateOption {
	return func(e *TaskEntry, _ time.Time) { e.Error = err }
}

// WithTimestamps sets StartedAt or CompletedAt based on status.
func WithTimestamps() TaskUpdateOption {
	return func(e *TaskEntry, now time.Time) {
		switch e.Status {
		case TaskRunning:
			e.StartedAt = &now
		case TaskSucceeded, TaskFailed, TaskTimedOut, TaskCancelled:
			e.CompletedAt = &now
			if e.StartedAt != nil {
				e.Duration = now.Sub(*e.StartedAt)
			}
		}
	}
}

// WithIncrementRetry increments retry count.
func WithIncrementRetry() TaskUpdateOption {
	return func(e *TaskEntry, _ time.Time) { e.RetryCount++ }
}

// ListAll returns all entries in insertion order.
func (t *TaskProgressTable) ListAll() []*TaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*TaskEntry, 0, len(t.order))
	for _, id := range t.order {
		if e, ok := t.entries[id]; ok {
			cp := *e
			result = append(result, &cp)
		}
	}
	return result
}

// ListByStatus returns entries matching the given status.
func (t *TaskProgressTable) ListByStatus(status TaskStatus) []*TaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*TaskEntry
	for _, id := range t.order {
		if e, ok := t.entries[id]; ok && e.Status == status {
			cp := *e
			result = append(result, &cp)
		}
	}
	return result
}

// Count returns total number of tasks.
func (t *TaskProgressTable) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

// PendingCount returns count of non-terminal tasks.
func (t *TaskProgressTable) PendingCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	count := 0
	for _, id := range t.order {
		if e, ok := t.entries[id]; ok && !e.IsTerminal() {
			count++
		}
	}
	return count
}

// CompletedCount returns count of successfully completed tasks.
func (t *TaskProgressTable) CompletedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	count := 0
	for _, id := range t.order {
		if e, ok := t.entries[id]; ok && e.IsCompletedSuccessfully() {
			count++
		}
	}
	return count
}

// FailedCount returns count of failed/timed-out tasks.
func (t *TaskProgressTable) FailedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	count := 0
	for _, id := range t.order {
		if e, ok := t.entries[id]; ok {
			switch e.Status {
			case TaskFailed, TaskTimedOut, TaskCancelled:
				count++
			}
		}
	}
	return count
}

// AllCompleted returns true if every task has reached a terminal state.
func (t *TaskProgressTable) AllCompleted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, id := range t.order {
		if e, ok := t.entries[id]; ok && !e.IsTerminal() {
			return false
		}
	}
	return true
}

// ParentID returns the parent task ID.
func (t *TaskProgressTable) ParentID() string { return t.parentID }

// Summary returns a human-readable summary of the table state.
func (t *TaskProgressTable) Summary() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := len(t.entries)
	succeeded := 0
	failed := 0
	pending := 0

	for _, id := range t.order {
		if e, ok := t.entries[id]; ok {
			switch {
			case e.IsCompletedSuccessfully():
				succeeded++
			case e.IsTerminal():
				failed++
			default:
				pending++
			}
		}
	}

	return fmt.Sprintf("ProgressTable[parent=%s]: total=%d succeeded=%d failed=%d pending=%d",
		t.parentID, total, succeeded, failed, pending)
}

// ===========================================================================
// CoordState — Coordinator runtime state (attached to ReactContext)
// ===========================================================================

// CoordState holds the runtime state for an agent operating in Coordinator mode.
// This is only valid when ReactContext.Mode == ModeCoordinator.
type CoordState struct {
	// ====== Basic fields ======
	ParentTaskID string             // Parent task ID
	TaskProgress *TaskProgressTable // Progress tracking table
	SubTaskResults map[string]*core.TaskResultEvent // Received results keyed by task ID
	DispatchedAt time.Time          // When coordination started
	GlobalTimer  *time.Timer        // Overall timeout timer

	// ====== Lifecycle control fields ======
	mu              sync.Mutex       // Protects lifecycle state transitions
	LifecycleCtx    context.Context              // Coordinator lifecycle context
	LifecycleCancel context.CancelFunc           // Cancellation function
	SubTaskCtxs     map[string]context.Context    // Per-sub-task independent context
	SubTaskCancels  map[string]context.CancelFunc // Per-sub-task cancellation functions
	LifecycleState  LifecycleState               // Current lifecycle state
	ControlChan     chan *core.ControlCommand         // External control command channel
	InterruptReason string                       // Interrupt reason (only in Interrupted state)
	InterruptedAt   time.Time                    // Interrupt timestamp
	CancelReason    string                        // Cancel reason (only in Cancelled state)
}

// NewCoordState creates a new CoordState for coordinator mode.
func NewCoordState(parentTaskID string, overallTimeout time.Duration) *CoordState {
	ctx, cancel := context.WithCancel(context.Background())
	cs := &CoordState{
		ParentTaskID:    parentTaskID,
		TaskProgress:    NewTaskProgressTable(parentTaskID),
		SubTaskResults:  make(map[string]*core.TaskResultEvent),
		DispatchedAt:    time.Now(),
		LifecycleCtx:    ctx,
		LifecycleCancel: cancel,
		SubTaskCtxs:     make(map[string]context.Context),
		SubTaskCancels:  make(map[string]context.CancelFunc),
		LifecycleState:  LifecycleRunning,
		ControlChan:     make(chan *core.ControlCommand, 8),
	}

	if overallTimeout > 0 {
		cs.GlobalTimer = time.AfterFunc(overallTimeout, func() {
			select {
			case cs.ControlChan <- &core.ControlCommand{Action: core.CmdCancel, Reason: "global timeout exceeded", Timestamp: time.Now()}:
			default:
				// Channel full — already being cancelled
			}
		})
	}

	return cs
}

// Dispose cleans up all resources held by the CoordState.
// Must be called when coordination ends (success, cancellation, or error).
func (cs *CoordState) Dispose() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.LifecycleCancel != nil {
		cs.LifecycleCancel()
	}
	if cs.GlobalTimer != nil {
		cs.GlobalTimer.Stop()
	}

	// Safe close: only close if not already closed
	if cs.ControlChan != nil {
		select {
		case <-cs.ControlChan:
			// Already closed
		default:
			close(cs.ControlChan)
		}
		cs.ControlChan = nil
	}

	// Cancel all sub-task contexts
	for _, cancel := range cs.SubTaskCancels {
		if cancel != nil {
			cancel()
		}
	}
}

// ===========================================================================
// Responsibility Gate Data Structures
// ===========================================================================

// ResponsibilityCheck is the output of Step A (Responsibility Check) in Think phase.
type ResponsibilityCheck struct {
	IsMatch    bool    // True if the task matches this agent's capabilities
	Confidence float64 // Match confidence (0.0–1.0)
	Reasoning  string  // LLM's reasoning for the match decision
}

// AtomicityCheck is the output of Step B (Atomicity/WBS Check) in Think phase.
type AtomicityCheck struct {
	IsAtomic  bool                 // True if the task is atomic (no decomposition needed)
	SubTasks  []TaskDecomposition  // WBS decomposition results (if not atomic)
	Reasoning string               // LLM reasoning for the atomicity decision
}

// TaskDecomposition represents one atomic sub-task from WBS decomposition.
type TaskDecomposition struct {
	ID                 string            // Globally unique sub-task ID
	Title              string            // Sub-task title
	Description        string            // Detailed description (instruction for executor)
	Priority           int               // Priority (lower = first)
	DependsOn          []string          // Predecessor sub-task IDs (empty = no dependency)
	DesiredCapability  string            // Optional capability hint (empty = orchestrator infers)
}

// ===========================================================================
// Coordinator Lifecycle Control Methods
// ===========================================================================

// Coordinator provides lifecycle control methods for agents in coordinator mode.
// These methods are called on the ReactContext's CoordState.
//
// The lifecycle state machine:
//
//	Running ──→ Interrupted ──→ Running (via Resume)
//	  │              │
//	  ├──→ Completed │         ├──→ Cancelled (terminal)
//	  └──→ Cancelled ─────────┘ (terminal)

// Interrupt pauses the Coordinator, preserving all state for later resumption.
// Sub-tasks that are currently running are cancelled via their individual contexts.
func (cs *CoordState) Interrupt(reason string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.checkLifecycleTransition(LifecycleInterrupted, reason)

	if cs.LifecycleState != LifecycleRunning {
		return fmt.Errorf("coordinator: cannot interrupt from state %s", cs.LifecycleState)
	}

	// Cancel all active sub-task contexts
	cs.cancelAllSubTasks()

	cs.LifecycleState = LifecycleInterrupted
	cs.InterruptReason = reason
	cs.InterruptedAt = time.Now()

	return nil
}

// Resume continues a previously interrupted Coordinator.
// Creates new contexts for sub-tasks that were interrupted (not cancelled ones).
func (cs *CoordState) Resume() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.checkLifecycleTransition(LifecycleRunning, "")

	if cs.LifecycleState != LifecycleInterrupted {
		return fmt.Errorf("coordinator: cannot resume from state %s", cs.LifecycleState)
	}

	// Recreate lifecycle context (old one was cancelled during interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	cs.LifecycleCtx = ctx
	cs.LifecycleCancel = cancel

	// Reset sub-task contexts for tasks that were running when interrupted
	cs.recreateInterruptedSubTaskContexts()

	cs.LifecycleState = LifecycleRunning
	cs.InterruptReason = ""
	cs.InterruptedAt = time.Time{}

	return nil
}

// Cancel irreversibly terminates the Coordinator. This is a terminal state.
func (cs *CoordState) Cancel(reason string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	targetState := LifecycleCancelled
	cs.checkLifecycleTransition(targetState, reason)

	if cs.LifecycleState.IsTerminal() {
		return fmt.Errorf("coordinator: cannot cancel from terminal state %s", cs.LifecycleState)
	}

	// Cancel everything
	if cs.LifecycleCancel != nil {
		cs.LifecycleCancel()
	}
	cs.cancelAllSubTasks()

	cs.LifecycleState = targetState
	cs.CancelReason = reason

	return nil
}

// MarkCompleted transitions the Coordinator to the completed (terminal) state.
// Called automatically when all sub-tasks finish.
func (cs *CoordState) MarkCompleted() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.LifecycleState.IsTerminal() {
		return
	}
	cs.LifecycleState = LifecycleCompleted
}

// RegisterSubTask registers a new sub-task with its own cancellable context.
func (cs *CoordState) RegisterSubTask(taskID string) context.Context {
	taskCtx, taskCancel := context.WithCancel(cs.LifecycleCtx)
	cs.SubTaskCtxs[taskID] = taskCtx
	cs.SubTaskCancels[taskID] = taskCancel
	return taskCtx
}

// UnregisterSubTask removes a completed/cancelled sub-task's context.
func (cs *CoordState) UnregisterSubTask(taskID string) {
	delete(cs.SubTaskCtxs, taskID)
	delete(cs.SubTaskCancels, taskID)
}

// ===========================================================================
// Internal helpers
// ===========================================================================

func (cs *CoordState) checkLifecycleTransition(target LifecycleState, reason string) {
	from := cs.LifecycleState
	if from == target {
		return
	}
	if !from.CanTransitionTo(target) {
		logger.Warn("invalid lifecycle state transition",
			"from", from,
			"to", target,
			"reason", reason,
		)
		return
	}
	logger.Info("coordinator lifecycle transition",
		"from", from,
		"to", target,
		"reason", reason,
	)
}

func (cs *CoordState) cancelAllSubTasks() {
	for taskID, cancel := range cs.SubTaskCancels {
		if cancel != nil {
			cancel()
		}
		// Mark cancelled tasks in progress table
		if cs.TaskProgress != nil {
			cs.TaskProgress.UpdateStatus(taskID, TaskCancelled, WithError(errors.New("coordinator interrupted")))
		}
	}
}

func (cs *CoordState) recreateInterruptedSubTaskContexts() {
	if cs.TaskProgress == nil {
		return
	}

	for _, entry := range cs.TaskProgress.ListAll() {
		// Only recreate contexts for tasks that were actually running (not yet terminal)
		if !entry.IsTerminal() && entry.Status != TaskDispatched && entry.Status != TaskAssigned {
			taskCtx, taskCancel := context.WithCancel(cs.LifecycleCtx)
			cs.SubTaskCtxs[entry.TaskID] = taskCtx
			cs.SubTaskCancels[entry.TaskID] = taskCancel
			// Reset status back to dispatched so it gets re-executed
			cs.TaskProgress.UpdateStatus(entry.TaskID, TaskDispatched)
		}
	}
}

var ErrInvalidLifecycleTransition = errors.New("invalid lifecycle state transition")
