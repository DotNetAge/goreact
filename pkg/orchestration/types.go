package orchestration

import (
	"time"
)

// =============================================================================
// Task Types
// =============================================================================

// Task represents a task to be orchestrated
type Task struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Input       map[string]any `json:"input"`
	Context     map[string]any `json:"context"`
	Priority    int            `json:"priority"`
	Timeout     time.Duration  `json:"timeout"`
	CreatedAt   time.Time      `json:"created_at"`
}

// SubTask represents a decomposed sub-task
type SubTask struct {
	Name                string         `json:"name"`
	ParentName          string         `json:"parent_name"`
	Description         string         `json:"description"`
	RequiredCapabilities []string      `json:"required_capabilities"`
	Dependencies        []string       `json:"dependencies"`
	Priority            int            `json:"priority"`
	Timeout             time.Duration  `json:"timeout"`
	Input               map[string]any `json:"input"`
}

// =============================================================================
// Plan Types
// =============================================================================

// OrchestrationPlan represents a complete execution plan
type OrchestrationPlan struct {
	Name              string        `json:"name"`
	TaskName          string        `json:"task_name"`
	SubTasks          []*SubTask    `json:"sub_tasks"`
	DependencyGraph   *Graph        `json:"dependency_graph"`
	ExecutionOrder    [][]string    `json:"execution_order"` // Layers of tasks
	EstimatedDuration time.Duration `json:"estimated_duration"`
}

// Graph represents a dependency graph
type Graph struct {
	Nodes []string `json:"nodes"`
	Edges []*Edge  `json:"edges"`
}

// Edge represents a dependency edge in the graph
type Edge struct {
	From string         `json:"from"`
	To   string         `json:"to"`
	Type DependencyType `json:"type"`
}

// DependencyType defines the type of dependency between tasks
type DependencyType string

const (
	DependencySequential    DependencyType = "sequential"
	DependencyData          DependencyType = "data_dependency"
	DependencyResource      DependencyType = "resource_dependency"
	DependencyConditional   DependencyType = "conditional"
)

// =============================================================================
// Agent Selection Types
// =============================================================================

// Capabilities describes an agent's capabilities
type Capabilities struct {
	Skills       []string `json:"skills"`
	Tools        []string `json:"tools"`
	Domains      []string `json:"domains"`
	Languages    []string `json:"languages"`
	MaxComplexity int     `json:"max_complexity"`
}

// LoadInfo represents agent load information
type LoadInfo struct {
	AgentName        string        `json:"agent_name"`
	ActiveTasks      int           `json:"active_tasks"`
	QueueLength      int           `json:"queue_length"`
	CPUPercent       float64       `json:"cpu_percent"`
	MemoryPercent    float64       `json:"memory_percent"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	LastUpdateTime   time.Time     `json:"last_update_time"`
}

// AgentMatch represents an agent match result
type AgentMatch struct {
	AgentName        string  `json:"agent_name"`
	CapabilityScore  float64 `json:"capability_score"`
	LoadScore        float64 `json:"load_score"`
	HistoryScore     float64 `json:"history_score"`
	TotalScore       float64 `json:"total_score"`
}

// =============================================================================
// Result Types
// =============================================================================

// SubResult represents the result of a sub-task execution
type SubResult struct {
	SubTaskName string         `json:"sub_task_name"`
	AgentName   string         `json:"agent_name"`
	Success     bool           `json:"success"`
	Output      map[string]any `json:"output"`
	Error       error          `json:"error,omitempty"`
	Duration    time.Duration  `json:"duration"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
}

// Result represents the final orchestration result
type Result struct {
	TaskName      string         `json:"task_name"`
	SubResults    []*SubResult   `json:"sub_results"`
	FinalOutput   map[string]any `json:"final_output"`
	Duration      time.Duration  `json:"duration"`
	Success       bool           `json:"success"`
	Error         error          `json:"error,omitempty"`
}

// AgentResult represents the result from an agent
type AgentResult struct {
	AgentName string    `json:"agent_name"`
	Success   bool      `json:"success"`
	Result    string    `json:"result"`
	Error     string    `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// =============================================================================
// State Types
// =============================================================================

// ExecutionPhase represents the current phase of orchestration
type ExecutionPhase string

const (
	PhaseIdle       ExecutionPhase = "idle"        // Initial state
	PhasePlanning   ExecutionPhase = "planning"
	PhaseSelecting  ExecutionPhase = "selecting"
	PhaseExecuting  ExecutionPhase = "executing"
	PhaseAggregating ExecutionPhase = "aggregating"
	PhaseSuspended  ExecutionPhase = "suspended"
	PhaseRetrying   ExecutionPhase = "retrying"    // Retry after failure
	PhaseCompleted  ExecutionPhase = "completed"
	PhaseFailed     ExecutionPhase = "failed"
)

// AgentStatus represents the status of an agent in orchestration
type AgentStatus string

const (
	AgentStatusPending   AgentStatus = "pending"
	AgentStatusRunning   AgentStatus = "running"
	AgentStatusSuspended AgentStatus = "suspended"
	AgentStatusCompleted AgentStatus = "completed"
	AgentStatusFailed    AgentStatus = "failed"
	AgentStatusBlocked   AgentStatus = "blocked"
)

// AgentState represents the state of an agent in orchestration
type AgentState struct {
	AgentName      string          `json:"agent_name"`
	SubTaskName    string          `json:"sub_task_name"`
	Status         AgentStatus     `json:"status"`
	FrozenState    []byte          `json:"frozen_state,omitempty"`
	PendingQuestion *PendingQuestion `json:"pending_question,omitempty"`
	Result         *SubResult      `json:"result,omitempty"`
	StartTime      time.Time       `json:"start_time"`
	EndTime        time.Time       `json:"end_time,omitempty"`
}

// OrchestrationState represents the complete state of an orchestration
type OrchestrationState struct {
	SessionName      string                     `json:"session_name"`
	Plan             *OrchestrationPlan         `json:"plan"`
	AgentStates      map[string]*AgentState     `json:"agent_states"`
	ExecutionPhase   ExecutionPhase             `json:"execution_phase"`
	PendingQuestions []*PendingQuestion         `json:"pending_questions"`
	CompletedSubTasks []string                  `json:"completed_sub_tasks"`
	FailedSubTasks   []string                   `json:"failed_sub_tasks"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

// PendingQuestion represents a question pending user response
type PendingQuestion struct {
	ID           string    `json:"id"`
	AgentName    string    `json:"agent_name"`
	SubTaskName  string    `json:"sub_task_name"`
	Question     string    `json:"question"`
	QuestionType string    `json:"question_type"`
	Options      []string  `json:"options,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// =============================================================================
// Pause-Resume Types
// =============================================================================

// SnapshotLevel defines the granularity of snapshots
type SnapshotLevel string

const (
	SnapshotLevelOrchestration SnapshotLevel = "orchestration"
	SnapshotLevelAgent         SnapshotLevel = "agent"
)

// OrchestrationSnapshot represents a snapshot of the orchestration state
type OrchestrationSnapshot struct {
	SessionName      string                 `json:"session_name"`
	Plan             *OrchestrationPlan     `json:"plan"`
	AgentStates      map[string]*AgentState `json:"agent_states"`
	ExecutionPhase   ExecutionPhase         `json:"execution_phase"`
	PendingQuestions []*PendingQuestion     `json:"pending_questions"`
	CreatedAt        time.Time              `json:"created_at"`
	Checksum         string                 `json:"checksum"`
}

// AgentSnapshot represents a snapshot of a single agent's state
type AgentSnapshot struct {
	AgentName          string           `json:"agent_name"`
	SessionName        string           `json:"session_name"`
	FrozenState        []byte           `json:"frozen_state"`
	PendingQuestion    *PendingQuestion `json:"pending_question,omitempty"`
	LastSuccessfulStep int              `json:"last_successful_step"`
	CreatedAt          time.Time        `json:"created_at"`
	Checksum           string           `json:"checksum"`
}

// SignedSnapshot represents a signed snapshot with Ed25519 signature
type SignedSnapshot struct {
	Content   []byte `json:"content"`    // Snapshot content (JSON serialized)
	Signature []byte `json:"signature"`  // Ed25519 signature
	Algorithm string `json:"algorithm"`  // Signature algorithm: Ed25519
	KeyID     string `json:"key_id"`     // Key identifier
	Timestamp int64  `json:"timestamp"`  // Signature timestamp
}

// AnswerMap maps question IDs to answers
type AnswerMap struct {
	SessionName string            `json:"session_name"`
	Answers     map[string]string `json:"answers"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ResumeRequest represents a resume request
type ResumeRequest struct {
	SessionName string     `json:"session_name"`
	AnswerMap   *AnswerMap `json:"answer_map"`
	ResumeAll   bool       `json:"resume_all"`
}

// ResumeResult represents the result of a resume operation
type ResumeResult struct {
	SessionName   string          `json:"session_name"`
	ResumedAgents []string        `json:"resumed_agents"`
	StillPending  []string        `json:"still_pending"`
	Errors        map[string]error `json:"errors,omitempty"`
}

// ResumeStrategy defines the resume strategy
type ResumeStrategy int

const (
	ResumeAll       ResumeStrategy = iota // Resume all suspended agents
	ResumeSelected                        // Resume only selected agents
	ResumeSequential                      // Resume agents one by one in order
)

// =============================================================================
// Error Types
// =============================================================================

// ErrorType defines orchestration error types
type ErrorType string

const (
	ErrorPlanningFailed       ErrorType = "planning_failed"
	ErrorAgentSelectionFailed ErrorType = "agent_selection_failed"
	ErrorExecutionFailed      ErrorType = "execution_failed"
	ErrorTimeout              ErrorType = "timeout"
	ErrorResourceExhausted    ErrorType = "resource_exhausted"
	ErrorDependencyViolation  ErrorType = "dependency_violation"
)

// =============================================================================
// Concurrency Types
// =============================================================================

// ConcurrencyConfig represents concurrency configuration
type ConcurrencyConfig struct {
	MaxConcurrent     int           `json:"max_concurrent"`      // Max concurrent agents, default 5
	RateLimitPerAgent int           `json:"rate_limit_per_agent"` // Requests per agent per second, default 10
	TokenBucketSize   int           `json:"token_bucket_size"`   // Token bucket size, default 100
	Timeout           time.Duration `json:"timeout"`             // Single task timeout, default 5 minutes
	RetryCount        int           `json:"retry_count"`         // Retry count, default 3
	InitialBackoff    time.Duration `json:"initial_backoff"`     // Initial backoff, default 1s
	MaxBackoff        time.Duration `json:"max_backoff"`         // Max backoff, default 30s
	FailFast          bool          `json:"fail_fast"`           // Fail on first error
}

// DefaultConcurrencyConfig returns default concurrency configuration
func DefaultConcurrencyConfig() *ConcurrencyConfig {
	return &ConcurrencyConfig{
		MaxConcurrent:     5,
		RateLimitPerAgent: 10,
		TokenBucketSize:   100,
		Timeout:           5 * time.Minute,
		RetryCount:        3,
		InitialBackoff:    time.Second,
		MaxBackoff:        30 * time.Second,
		FailFast:          false,
	}
}

// =============================================================================
// Aggregation Types
// =============================================================================

// MergeStrategy defines result merge strategy
type MergeStrategy string

const (
	MergeStrategyConcat      MergeStrategy = "concat"       // Simple concatenation
	MergeStrategyStructured  MergeStrategy = "structured"   // Structured merge
	MergeStrategyLLM         MergeStrategy = "llm"          // LLM-based merge
	MergeStrategyVoting      MergeStrategy = "voting"       // Voting-based merge
)

// ValidationRule represents a validation rule
type ValidationRule struct {
	Name     string                   `json:"name"`
	Check    func(*SubResult) error   `json:"-"`
	Severity ValidationSeverity       `json:"severity"`
}

// ValidationSeverity defines validation severity
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

// =============================================================================
// Observability Types
// =============================================================================

// Metrics represents orchestration metrics
type Metrics struct {
	OrchestrationLatency  time.Duration `json:"orchestration_latency"`
	ExecutionTime         time.Duration `json:"execution_time"`
	SubTaskSuccessRate    float64       `json:"sub_task_success_rate"`
	AgentUtilization      float64       `json:"agent_utilization"`
	ParallelismEfficiency float64       `json:"parallelism_efficiency"`
	ActiveAgents          int           `json:"active_agents"`
	QueuedTasks           int           `json:"queued_tasks"`
	AvgExecutionTime      time.Duration `json:"avg_execution_time"`
	ConcurrencyUtilization float64      `json:"concurrency_utilization"`
	RateLimitHits         int           `json:"rate_limit_hits"`
}

// AlertLevel defines alert severity
type AlertLevel string

const (
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert represents an orchestration alert
type Alert struct {
	Name        string      `json:"name"`
	Condition   string      `json:"condition"`
	Level       AlertLevel  `json:"level"`
	Message     string      `json:"message"`
	Timestamp   time.Time   `json:"timestamp"`
}

// =============================================================================
// Decomposition Types
// =============================================================================

// DecompositionStrategy defines task decomposition strategy
type DecompositionStrategy string

const (
	DecompositionRule     DecompositionStrategy = "rule"     // Rule-based decomposition
	DecompositionLLM      DecompositionStrategy = "llm"      // LLM-based decomposition
	DecompositionHybrid   DecompositionStrategy = "hybrid"   // Hybrid decomposition
)

// DecompositionRuleSpec represents a rule-based decomposition specification
type DecompositionRuleSpec struct {
	TaskType    string   `json:"task_type"`
	SubTaskDefs []SubTaskDef `json:"sub_task_defs"`
}

// SubTaskDef defines a sub-task template
type SubTaskDef struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	Dependencies []string `json:"dependencies"`
}
