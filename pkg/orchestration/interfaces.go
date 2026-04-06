package orchestration

import (
	"context"
	"time"
)

// =============================================================================
// Orchestrator Interface
// =============================================================================

// Orchestrator is the main interface for multi-agent coordination
type Orchestrator interface {
	// Orchestrate executes the orchestration task and returns the final result
	Orchestrate(ctx context.Context, task *Task) (*Result, error)
	
	// Status returns the current orchestration status
	Status() *OrchestrationState
	
	// Pause pauses the orchestration execution
	Pause() error
	
	// Resume resumes the orchestration with user answers
	Resume(ctx context.Context, sessionName string, answers map[string]string) (*Result, error)
	
	// Stop stops the orchestration execution
	Stop() error
}

// =============================================================================
// Task Planner Interface
// =============================================================================

// TaskPlanner plans and decomposes tasks
type TaskPlanner interface {
	// Plan generates a complete execution plan
	Plan(task *Task) (*OrchestrationPlan, error)
	
	// Decompose decomposes a task into sub-tasks
	Decompose(task *Task) ([]*SubTask, error)
	
	// Optimize optimizes the execution plan
	Optimize(plan *OrchestrationPlan) (*OrchestrationPlan, error)
}

// =============================================================================
// Agent Selector Interface
// =============================================================================

// Selector selects agents for sub-tasks
type Selector interface {
	// Select selects the best agent for a single sub-task
	Select(subTask *SubTask, candidates []Agent) (Agent, error)
	
	// SelectBatch selects agents for multiple sub-tasks
	SelectBatch(subTasks []*SubTask, candidates []Agent) (map[string]Agent, error)
	
	// Capabilities returns the capabilities of an agent
	Capabilities(agent Agent) (*Capabilities, error)
}

// Agent represents an agent that can execute tasks
type Agent interface {
	// Name returns the agent's name
	Name() string
	
	// Execute executes a sub-task
	Execute(ctx context.Context, subTask *SubTask) (*SubResult, error)
	
	// Capabilities returns the agent's capabilities
	Capabilities() *Capabilities
	
	// Freeze freezes the agent's state
	Freeze() ([]byte, error)
	
	// Thaw restores the agent's state from frozen data
	Thaw(data []byte) error
	
	// Status returns the agent's current status
	Status() AgentStatus
}

// =============================================================================
// Execution Coordinator Interface
// =============================================================================

// Coordinator coordinates agent execution
type Coordinator interface {
	// Execute executes the plan, automatically selecting execution mode
	Execute(ctx context.Context, plan *OrchestrationPlan, agents map[string]Agent) ([]*SubResult, error)
	
	// ExecuteParallel executes sub-tasks in parallel
	ExecuteParallel(ctx context.Context, subTasks []*SubTask, agents map[string]Agent) ([]*SubResult, error)
	
	// ExecuteSequential executes sub-tasks sequentially
	ExecuteSequential(ctx context.Context, subTasks []*SubTask, agents map[string]Agent) ([]*SubResult, error)
}

// =============================================================================
// Result Aggregator Interface
// =============================================================================

// Aggregator aggregates agent results
type Aggregator interface {
	// Aggregate aggregates sub-task results into final result
	Aggregate(results []*SubResult) (*Result, error)
	
	// Merge merges multiple results into a string
	Merge(results []*SubResult) string
	
	// Validate validates the results
	Validate(results []*SubResult) error
}

// =============================================================================
// State Storage Interface
// =============================================================================

// StateStorage stores orchestration state
type StateStorage interface {
	// Store stores the orchestration state
	Store(ctx context.Context, state *OrchestrationState) error
	
	// Get retrieves the orchestration state
	Get(ctx context.Context, sessionName string) (*OrchestrationState, error)
	
	// UpdateAgentState updates a specific agent's state
	UpdateAgentState(ctx context.Context, sessionName string, agentName string, agentState *AgentState) error
	
	// Delete deletes the orchestration state
	Delete(ctx context.Context, sessionName string) error
}

// =============================================================================
// Capability Matcher Interface
// =============================================================================

// CapabilityMatcher matches capabilities between required and provided
type CapabilityMatcher interface {
	// Match returns a score for capability matching
	Match(required, provided *Capabilities) float64
}

// =============================================================================
// Load Balancer Interface
// =============================================================================

// LoadBalancer balances load among agents
type LoadBalancer interface {
	// Select selects an agent based on load
	Select(candidates []Agent, loads map[string]*LoadInfo) (Agent, error)
}

// =============================================================================
// Result Validator Interface
// =============================================================================

// ResultValidator validates sub-task results
type ResultValidator interface {
	// Validate validates the results
	Validate(results []*SubResult) error
	
	// AddRule adds a validation rule
	AddRule(rule *ValidationRule)
}

// =============================================================================
// Decomposition Strategy Interface
// =============================================================================

// Decomposer decomposes tasks into sub-tasks
type Decomposer interface {
	// Decompose decomposes a task into sub-tasks
	Decompose(task *Task) ([]*SubTask, error)
	
	// Strategy returns the decomposition strategy name
	Strategy() DecompositionStrategy
}

// =============================================================================
// Dependency Analyzer Interface
// =============================================================================

// DependencyAnalyzer analyzes dependencies between sub-tasks
type DependencyAnalyzer interface {
	// Analyze analyzes dependencies between sub-tasks
	Analyze(subTasks []*SubTask) (*Graph, error)
	
	// TopologicalSort performs topological sort on the dependency graph
	TopologicalSort(graph *Graph) ([][]string, error)
	
	// DetectCycle detects cycles in the dependency graph
	DetectCycle(graph *Graph) ([]string, error)
}

// =============================================================================
// Retry Handler Interface
// =============================================================================

// RetryHandler handles retries for failed operations
type RetryHandler interface {
	// Retry retries the operation with backoff
	Retry(ctx context.Context, op func() error) error
	
	// WithBackoff executes with exponential backoff
	WithBackoff(ctx context.Context, op func() error, initial, max time.Duration) error
}

// =============================================================================
// Metrics Collector Interface
// =============================================================================

// MetricsCollector collects orchestration metrics
type MetricsCollector interface {
	// RecordMetric records a metric
	RecordMetric(name string, value float64, tags map[string]string)
	
	// GetMetrics returns current metrics
	GetMetrics() *Metrics
	
	// Reset resets the metrics
	Reset()
}

// =============================================================================
// Alert Manager Interface
// =============================================================================

// AlertManager manages alerts
type AlertManager interface {
	// Check checks for alert conditions
	Check(metrics *Metrics) []*Alert
	
	// AddRule adds an alert rule
	AddRule(name string, condition string, level AlertLevel)
}

// =============================================================================
// Snapshot Manager Interface
// =============================================================================

// SnapshotManager manages snapshots
type SnapshotManager interface {
	// CreateSnapshot creates a snapshot at the specified level
	CreateSnapshot(state *OrchestrationState, level SnapshotLevel) ([]byte, error)
	
	// RestoreSnapshot restores state from a snapshot
	RestoreSnapshot(data []byte) (*OrchestrationState, error)
	
	// Sign signs a snapshot
	Sign(data []byte) (*SignedSnapshot, error)
	
	// Verify verifies a signed snapshot
	Verify(signed *SignedSnapshot) ([]byte, error)
}

// =============================================================================
// Health Checker Interface
// =============================================================================

// HealthChecker checks agent health
type HealthChecker interface {
	// Check checks the health of an agent
	Check(agent Agent) error
	
	// CheckAll checks the health of all agents
	CheckAll(agents []Agent) map[string]error
}

// =============================================================================
// Concurrency Pool Interface
// =============================================================================

// ConcurrencyPool manages concurrent execution
type ConcurrencyPool interface {
	// Submit submits a task for execution
	Submit(ctx context.Context, task func() error) error
	
	// WaitAll waits for all tasks to complete
	WaitAll() error
	
	// Close closes the pool
	Close()
	
	// Stats returns pool statistics
	Stats() PoolStats
}

// PoolStats represents pool statistics
type PoolStats struct {
	ActiveWorkers int
	QueuedTasks   int
	CompletedTasks int
	FailedTasks   int
}
