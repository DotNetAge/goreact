package orchestration

import (
	"fmt"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ===========================================================================
// Event Types — Communication Protocol (Design §7.1)
// ===========================================================================
// All Agent ↔ Orchestrator communication uses standardized Go channel events.
// Events are categorized into:
//   - Upstream (Agent → Orchestrator): task dispatch requests, status queries, score reports
//   - Downstream (Orchestrator → Agent): task assignments, result forwarding, timeout warnings

// ========== Upstream Events (Agent → Orchestrator) ==========

// TaskDispatchEvent is sent by an Agent (in Coordinator mode) to the Orchestrator
// when it needs to delegate a subtask. This triggers the LLM Router pipeline (Design §6.4).
type TaskDispatchEvent struct {
	RequestID        string    // Unique request ID
	SourceAgentID    string    // Requesting agent's ID
	ParentTaskID     string    // Parent task ID (if any)
	TaskID           string    // Sub-task ID
	TaskDescription  string    // Detailed task description for the executor
	DesiredCapability string  // Expected capability hint (can be empty = orchestrator infers)
	Priority         int       // Priority (lower = higher priority)
	Timestamp        time.Time
}

// QueryStatusEvent is sent by a Coordinator to query progress of specific subtasks (Design §10.2).
type QueryStatusEvent struct {
	RequestID     string
	SourceAgentID string
	TaskIDs       []string // Task IDs to query
	Timestamp     time.Time
}

// AgentScoreEvent is sent by a Coordinator to record performance scores for executed agents (Design §8.3).
type AgentScoreEvent struct {
	TargetAgentID string
	TaskID        string
	Score         int  // 0-3 score (see Design §8.1)
	Success       bool // Binary success flag
	Timestamp     time.Time
}

// ========== Downstream Events (Orchestrator → Agent) ==========

// TaskAssignedEvent confirms task assignment to the target agent (Design §6.4).
type TaskAssignedEvent struct {
	RequestID     string
	TaskID        string
	TargetAgentID string // Selected executor agent ID
	Timestamp     time.Time
}

// TaskResultEvent carries execution results from executor back through the orchestrator (Design §7.1).
type TaskResultEvent struct {
	TaskID        string
	TargetAgentID string // Executor agent ID
	Result        string // Execution result content
	Error         error  // Error if failed (nil = success)
	Duration      time.Duration
	Timestamp     time.Time
}

// TimeoutWarningEvent is sent when approaching soft timeout threshold (Design §10.3).
type TimeoutWarningEvent struct {
	TaskID    string
	Elapsed   time.Duration
	Remaining time.Duration
	Timestamp time.Time
}

// ========== Lifecycle Control Events (Design §10.5) ==========

// ControlCommand is an alias to core.ControlCommand (defined in control.go).
// Use CmdInterrupt, CmdResume, CmdCancel constants for action values.
//
// Lifecycle control event sent to a Coordinator (Design §10.5.1).
// Extended with Requester, Timestamp and Priority for orchestration-layer tracking.
type CoordControlCommand struct {
	core.ControlCommand     // Embedded core control command
	Reason     string       // Human-readable reason
	Requester  string       // "user" | "system" | "parent_coordinator" | "orchestrator"
	Timestamp  time.Time    // When the command was issued
	Priority   int          // Priority: 4=orchestrator, 3=parent, 2=self, 1=user (Design §10.5.4)
}

// CommandPriority defines priority levels for control commands (Design §10.5.4).
// Higher priority commands override lower priority commands.
const (
	PriorityUser         = 1 // User-initiated commands (lowest)
	PrioritySelf         = 2 // Self-initiated (e.g., internal timeout)
	PriorityParentCoord  = 3 // Parent coordinator
	PriorityOrchestrator = 4 // Orchestrator (highest)
)

// CoordLifecycleEvent notifies the orchestrator of Coordinator state changes (Design §10.5.2).
type CoordLifecycleEvent struct {
	CoordAgentID string
	ParentTaskID string
	Action       string // "interrupted" | "resumed" | "cancelled" | "completed"
	Reason       string
	PausedTasks  []string // Paused task IDs on interrupt
	Timestamp    time.Time
}

// ResumeTaskEvent is sent by the orchestrator to an agent to resume execution after interrupt (Design §10.5.2).
type ResumeTaskEvent struct {
	TaskID    string
	Timestamp time.Time
}

// TaskPausedEvent is an agent's acknowledgment of pause, reporting saved intermediate state (Design §10.5.5).
type TaskPausedEvent struct {
	TaskID      string
	AgentID     string
	SavedState  map[string]interface{} // Intermediate state snapshot (optional)
	ProgressPct float64               // Estimated progress percentage (0.0-1.0)
	Timestamp   time.Time
}

// ========== WBS Decomposition Types (Design §11 / §5) ==========

// ResponsibilityCheckResult holds the result of Step A: responsibility judgment (Design §5.1).
type ResponsibilityCheckResult struct {
	IsMatch    bool
	Confidence float64
	Reasoning  string
}

// AtomicityCheckResult holds the result of Step B: atomicity / WBS decomposition (Design §5.2).
type AtomicityCheckResult struct {
	IsAtomic bool
	SubTasks []TaskDecomposition
	Reasoning string
}

// TaskDecomposition represents one atomic subtask from WBS decomposition (Design §5.2 / §11).
type TaskDecomposition struct {
	ID                string   // Globally unique sub-task ID
	Title             string   // Brief title
	Description       string   // Detailed description for the executor agent
	Priority          int      // Priority (lower = higher priority)
	DependsOn         []string // Dependency sub-task IDs
	DesiredCapability string   // Optional capability hint (empty = orchestrator infers)
}

// ========== Coordinator Types (Design §10) ==========

// TaskState represents a subtask's lifecycle state (Design §9.2 / §14.1).
type TaskState string

const (
	TaskDispatched   TaskState = "dispatched"    // Dispatched, waiting for acceptance
	TaskSubExecuting TaskState = "sub_executing" // Sub-agent executing the task (Design §9.1)
	TaskRunning      TaskState = "running"       // Currently executing
	TaskPaused       TaskState = "paused"        // Paused (received Interrupt)
	TaskCompleted    TaskState = "completed"     // Successfully finished
	TaskFailed       TaskState = "failed"        // Execution failed
	TaskTimeout      TaskState = "timeout"       // Timed out
	TaskCancelled    TaskState = "cancelled"     // Cancelled (received Cancel)
	TaskSkipped      TaskState = "skipped"       // Skipped (dependency failed)
)

// LifecycleState represents a Coordinator's lifecycle state (Design §10.5.2 / §14.3).
type LifecycleState string

const (
	LifecycleRunning    LifecycleState = "running"      // Normal operation
	LifecycleInterrupted LifecycleState = "interrupted"  // Paused by interrupt
	LifecycleCancelled  LifecycleState = "cancelled"     // Terminated
	LifecycleCompleted  LifecycleState = "completed"     // All tasks done
)

// OrchestratorState defines the lifecycle states of the Orchestrator (Design §9.3 / P2-4).
type OrchestratorState string

const (
	OrchestratorInitializing OrchestratorState = "initializing" // Loading agents, setting up
	OrchestratorRunning      OrchestratorState = "running"      // Processing messages normally
	OrchestratorDraining     OrchestratorState = "draining"     // Rejecting new work, finishing pending
	OrchestratorStopped      OrchestratorState = "stopped"      // Fully stopped
)

// IsTerminal returns true if the orchestrator state is a terminal state.
func (s OrchestratorState) IsTerminal() bool {
	return s == OrchestratorStopped
}

// TaskEntry tracks a single subtask's progress within a Coordinator (Design §10.2 / §14.1).
type TaskEntry struct {
	TaskID          string
	State           TaskState
	ExpectedDur     time.Duration // LLM-estimated duration
	ActualStart     time.Time     // Actual start time
	Result          string        // Execution result
	Error           error         // Execution error
	RetryCount      int           // Number of retries
	Score           int           // Post-completion score (0-3)

	// Dependency tracking (Design §10.4)
	DependsOn []string // Upstream task IDs this task depends on

	// Lifecycle control fields (Design §10.5)
	PausedAt      *time.Time // Pause timestamp (only in paused state)
	SavedSnapshot []byte     // Agent-reported intermediate state snapshot
	ProgressPct   float64    // Estimated progress percentage (0.0-1.0)
}

// TaskProgressTable is the Coordinator's primary state data structure (Design §4.3 / §14.1).
// It maps task IDs to their progress entries and provides concurrent-safe access.
type TaskProgressTable struct {
	mu      sync.RWMutex
	entries map[string]*TaskEntry
}

// NewTaskProgressTable creates an empty task progress table.
func NewTaskProgressTable() *TaskProgressTable {
	return &TaskProgressTable{
		entries: make(map[string]*TaskEntry),
	}
}

// Get retrieves a task entry by ID. Returns nil if not found.
func (t *TaskProgressTable) Get(taskID string) *TaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.entries[taskID]
}

// Set updates or creates a task entry.
func (t *TaskProgressTable) Set(taskID string, entry *TaskEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries[taskID] = entry
}

// Delete removes a task entry.
func (t *TaskProgressTable) Delete(taskID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, taskID)
}

// Entries returns a snapshot of all entries (safe for concurrent use).
func (t *TaskProgressTable) Entries() []*TaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	snapshot := make([]*TaskEntry, 0, len(t.entries))
	for _, e := range t.entries {
		snapshot = append(snapshot, e)
	}
	return snapshot
}

// EntriesMap returns all entries as a map (caller must not hold across goroutines).
func (t *TaskProgressTable) EntriesMap() map[string]*TaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	m := make(map[string]*TaskEntry, len(t.entries))
	for k, v := range t.entries {
		m[k] = v
	}
	return m
}

// Count returns the number of tracked tasks.
func (t *TaskProgressTable) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

// PendingTaskIDs returns IDs of tasks that are not yet completed/failed/cancelled/skipped/timeout.
func (t *TaskProgressTable) PendingTaskIDs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var ids []string
	finalStates := map[TaskState]bool{
		TaskCompleted: true, TaskFailed: true, TaskCancelled: true,
		TaskSkipped: true, TaskTimeout: true,
	}
	for id, entry := range t.entries {
		if !finalStates[entry.State] {
			ids = append(ids, id)
		}
	}
	return ids
}

// AllCompleted returns true when all tasks are in terminal states.
func (t *TaskProgressTable) AllCompleted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, e := range t.entries {
		if !IsFinalState(e.State) {
			return false
		}
	}
	return len(t.entries) > 0
}

// Summary returns a one-line summary of task progress.
func (t *TaskProgressTable) Summary() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	completed, failed, running := 0, 0, 0
	for _, e := range t.entries {
		switch e.State {
		case TaskCompleted:
			completed++
		case TaskFailed, TaskTimeout, TaskCancelled, TaskSkipped:
			failed++
		case TaskRunning, TaskSubExecuting, TaskDispatched:
			running++
		}
	}
	return fmt.Sprintf("%d/%d completed, %d failed, %d running",
		completed, len(t.entries), failed, running)
}

// ListAll returns a snapshot of all entries as TaskEntry slices for reporting.
func (t *TaskProgressTable) ListAll() []TaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]TaskEntry, 0, len(t.entries))
	for _, e := range t.entries {
		result = append(result, *e)
	}
	return result
}

// CompletedCount returns the number of successfully completed tasks.
func (t *TaskProgressTable) CompletedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n := 0
	for _, e := range t.entries {
		if e.State == TaskCompleted {
			n++
		}
	}
	return n
}

// FailedCount returns the number of failed/timeout/cancelled/skipped tasks.
func (t *TaskProgressTable) FailedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n := 0
	failed := map[TaskState]bool{TaskFailed: true, TaskTimeout: true, TaskCancelled: true, TaskSkipped: true}
	for _, e := range t.entries {
		if failed[e.State] {
			n++
		}
	}
	return n
}

// --- Helper: convert core.AgentState to/from our types ---

// agentStateToCore maps orchestration AgentState values for RuntimeDirectory compatibility.
// This bridges the gap between the design doc's AgentState and core package's actual type.
func agentStateToRuntimeState(state string) core.AgentState {
	switch state {
	case "idle":
		return core.AgentStateIdle
	case "busy":
		return core.AgentStateBusy
	case "coordinating":
		return core.AgentStateCoordinating
	case "dormant":
		return core.AgentStateDormant
	default:
		return core.AgentStateError // Unknown state treated as error
	}
}

// IsFinalState checks if a TaskState is terminal (no further transitions possible).
func IsFinalState(s TaskState) bool {
	switch s {
	case TaskCompleted, TaskFailed, TaskTimeout, TaskCancelled, TaskSkipped:
		return true
	default:
		return false
	}
}
