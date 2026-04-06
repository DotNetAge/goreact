// Package orchestration provides multi-agent coordination for the goreact framework.
package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Orchestrator Implementation
// =============================================================================

// orchestratorImpl implements Orchestrator interface
type orchestratorImpl struct {
	mu          sync.RWMutex
	registry    map[string]Agent
	planner     TaskPlanner
	selector    Selector
	coordinator Coordinator
	aggregator  Aggregator
	stateMgr    *StateManager
	pauseResume *PauseResumeManager
	monitor     *ExecutionMonitor
	config      *Config
	status      *OrchestrationState
}

// Config represents orchestrator configuration
type Config struct {
	MaxConcurrentAgents int                  `json:"max_concurrent_agents"`
	Timeout             time.Duration        `json:"timeout"`
	EnableCaching       bool                 `json:"enable_caching"`
	Concurrency         *ConcurrencyConfig   `json:"concurrency"`
	Planner             *PlannerConfig       `json:"planner"`
	Selector            *SelectorConfig      `json:"selector"`
	Coordinator         *CoordinatorConfig   `json:"coordinator"`
	Aggregator          *AggregatorConfig    `json:"aggregator"`
}

// DefaultConfig returns default orchestrator config
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrentAgents: 5,
		Timeout:             5 * time.Minute,
		EnableCaching:       true,
		Concurrency:         DefaultConcurrencyConfig(),
		Planner:             DefaultPlannerConfig(),
		Selector:            DefaultSelectorConfig(),
		Coordinator:         DefaultCoordinatorConfig(),
		Aggregator:          DefaultAggregatorConfig(),
	}
}

// NewOrchestrator creates a new Orchestrator
func NewOrchestrator(config *Config) Orchestrator {
	if config == nil {
		config = DefaultConfig()
	}

	planner := NewTaskPlanner(config.Planner)
	selector := NewAgentSelector(config.Selector)
	coordinator := NewExecutionCoordinator(config.Coordinator)
	aggregator := NewResultAggregator(config.Aggregator)
	stateStorage := NewMemoryStateStorage()
	stateMgr := NewStateManager(stateStorage)
	
	return &orchestratorImpl{
		registry:    make(map[string]Agent),
		planner:     planner,
		selector:    selector,
		coordinator: coordinator,
		aggregator:  aggregator,
		stateMgr:    stateMgr,
		pauseResume: NewPauseResumeManager(stateMgr),
		monitor:     NewExecutionMonitor(),
		config:      config,
	}
}

// RegisterAgent registers an agent
func (o *orchestratorImpl) RegisterAgent(name string, agent Agent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.registry[name] = agent
}

// UnregisterAgent unregisters an agent
func (o *orchestratorImpl) UnregisterAgent(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.registry, name)
}

// GetAgent retrieves an agent
func (o *orchestratorImpl) GetAgent(name string) (Agent, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	agent, exists := o.registry[name]
	return agent, exists
}

// Orchestrate executes the orchestration task and returns the final result
func (o *orchestratorImpl) Orchestrate(ctx context.Context, task *Task) (*Result, error) {
	// Start monitoring
	o.monitor.Start(ctx)
	defer o.monitor.Stop()
	
	startTime := time.Now()
	sessionName := fmt.Sprintf("session-%s-%d", task.ID, startTime.Unix())
	
	// Initialize state
	state := o.stateMgr.CreateState(sessionName, nil)
	o.status = state
	
	// Phase 1: Planning
	o.monitor.Record("planning_started", task)
	state.ExecutionPhase = PhasePlanning
	
	plan, err := o.planner.Plan(task)
	if err != nil {
		state.ExecutionPhase = PhaseFailed
		return nil, NewOrchestrationError(ErrorPlanningFailed, "failed to plan task", err)
	}
	
	state.Plan = plan
	state.ExecutionPhase = PhaseSelecting
	o.monitor.Record("planning_completed", plan)
	
	// Phase 2: Agent Selection
	o.monitor.Record("selection_started", plan)
	
	agents := make(map[string]Agent)
	for _, subTask := range plan.SubTasks {
		agent, err := o.selector.Select(subTask, o.getAgentList())
		if err != nil {
			state.ExecutionPhase = PhaseFailed
			return nil, NewOrchestrationError(ErrorAgentSelectionFailed,
				fmt.Sprintf("failed to select agent for sub-task %s", subTask.Name), err)
		}
		agents[subTask.Name] = agent
		
		// Initialize agent state
		state.AgentStates[agent.Name()] = &AgentState{
			AgentName:   agent.Name(),
			SubTaskName: subTask.Name,
			Status:      AgentStatusPending,
			StartTime:   time.Now(),
		}
	}
	
	state.ExecutionPhase = PhaseExecuting
	o.monitor.Record("selection_completed", agents)
	
	// Phase 3: Execution
	o.monitor.Record("execution_started", plan)
	
	subResults, err := o.coordinator.Execute(ctx, plan, agents)
	if err != nil {
		state.ExecutionPhase = PhaseFailed
		return nil, NewOrchestrationError(ErrorExecutionFailed, "execution failed", err)
	}
	
	o.monitor.Record("execution_completed", subResults)
	
	// Phase 4: Aggregation
	state.ExecutionPhase = PhaseAggregating
	o.monitor.Record("aggregation_started", subResults)
	
	result, err := o.aggregator.Aggregate(subResults)
	if err != nil {
		state.ExecutionPhase = PhaseFailed
		return nil, NewOrchestrationError(ErrorExecutionFailed, "failed to aggregate results", err)
	}
	
	result.TaskName = task.Name
	result.Duration = time.Since(startTime)
	
	state.ExecutionPhase = PhaseCompleted
	o.monitor.Record("aggregation_completed", result)
	
	// Record metrics
	o.monitor.metrics.RecordMetric("execution_time", float64(result.Duration), nil)
	o.monitor.metrics.IncrementCounter("total_tasks", 1)
	if result.Success {
		o.monitor.metrics.IncrementCounter("completed_tasks", 1)
	} else {
		o.monitor.metrics.IncrementCounter("failed_tasks", 1)
	}
	
	return result, nil
}

// Status returns the current orchestration status
func (o *orchestratorImpl) Status() *OrchestrationState {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

// Pause pauses the orchestration execution
func (o *orchestratorImpl) Pause() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if o.status == nil {
		return fmt.Errorf("no active orchestration")
	}
	
	if o.status.ExecutionPhase != PhaseExecuting {
		return fmt.Errorf("can only pause during executing phase")
	}
	
	o.status.ExecutionPhase = PhaseSuspended
	return nil
}

// Resume resumes the orchestration with user answers
func (o *orchestratorImpl) Resume(ctx context.Context, sessionName string, answers map[string]string) (*Result, error) {
	answerMap := &AnswerMap{
		SessionName: sessionName,
		Answers:     answers,
		Timestamp:   time.Now(),
	}
	
	resumeResult, err := o.pauseResume.Resume(ctx, &ResumeRequest{
		SessionName: sessionName,
		AnswerMap:   answerMap,
		ResumeAll:   true,
	})
	if err != nil {
		return nil, err
	}
	
	// Continue execution with resumed agents
	// This would need to continue from where it left off
	_ = resumeResult // Placeholder for continued execution
	
	return &Result{Success: true}, nil
}

// Stop stops the orchestration execution
func (o *orchestratorImpl) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if o.status == nil {
		return nil
	}
	
	o.status.ExecutionPhase = PhaseFailed
	return nil
}

// getAgentList returns the list of registered agents
func (o *orchestratorImpl) getAgentList() []Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	agents := make([]Agent, 0, len(o.registry))
	for _, agent := range o.registry {
		agents = append(agents, agent)
	}
	return agents
}

// =============================================================================
// Legacy Compatibility
// =============================================================================

// Orchestrator is an alias for the interface (for backward compatibility)
type OrchestratorLegacy = Orchestrator

// NewLegacyOrchestrator creates a legacy orchestrator (for backward compatibility)
func NewLegacyOrchestrator(config *Config) Orchestrator {
	return NewOrchestrator(config)
}

// =============================================================================
// Simple Implementations for Legacy API
// =============================================================================

// SimpleOrchestrator is a simple orchestrator for backward compatibility
type SimpleOrchestrator struct {
	mu       sync.RWMutex
	agents   map[string]any
	selector Selector
	coordinator Coordinator
	aggregator  Aggregator
	config   *Config
}

// NewSimpleOrchestrator creates a simple orchestrator
func NewSimpleOrchestrator(config *Config) *SimpleOrchestrator {
	if config == nil {
		config = DefaultConfig()
	}
	return &SimpleOrchestrator{
		agents:   make(map[string]any),
		config:   config,
	}
}

// RegisterAgent registers an agent (legacy)
func (o *SimpleOrchestrator) RegisterAgent(name string, agent any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.agents[name] = agent
}

// UnregisterAgent unregisters an agent (legacy)
func (o *SimpleOrchestrator) UnregisterAgent(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.agents, name)
}

// GetAgent retrieves an agent (legacy)
func (o *SimpleOrchestrator) GetAgent(name string) (any, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	agent, exists := o.agents[name]
	return agent, exists
}

// Execute executes a task using coordinated agents (legacy)
func (o *SimpleOrchestrator) Execute(ctx context.Context, task *Task) (*OrchestrationResult, error) {
	startTime := time.Now()
	
	// Simplified execution
	agentResults := []*AgentResult{}
	
	return &OrchestrationResult{
		Task:         task,
		AgentResults: agentResults,
		FinalResult:  "Task completed",
		Duration:     time.Since(startTime),
	}, nil
}

// ExecuteWithCoordination executes with coordination (legacy)
func (o *SimpleOrchestrator) executeAgent(ctx context.Context, agent any, task *Task) *AgentResult {
	return &AgentResult{
		AgentName: "agent",
		Success:   true,
		Result:    "Result",
	}
}

// =============================================================================
// Execution Plan (Legacy)
// =============================================================================

// ExecutionPlan represents a coordinated execution plan (legacy)
type ExecutionPlan struct {
	TaskID   string
	Steps    []*ExecutionStep
	Status   string
}

// ExecutionStep represents a step in the execution plan (legacy)
type ExecutionStep struct {
	AgentName string
	Action    string
	Params    map[string]any
	Status    string
}

// OrchestrationResult represents the result of orchestration (legacy)
type OrchestrationResult struct {
	Task         *Task
	AgentResults []*AgentResult
	FinalResult  string
	Duration     time.Duration
}

// =============================================================================
// Helper Functions
// =============================================================================

// ValidateTask validates a task before execution
func ValidateTask(task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if task.Description == "" {
		return fmt.Errorf("task description is required")
	}
	return nil
}

// CreateTask creates a new task with defaults
func CreateTask(id, description string, input map[string]any) *Task {
	return &Task{
		ID:          id,
		Name:        id,
		Description: description,
		Input:       input,
		Context:     make(map[string]any),
		Priority:    0,
		Timeout:     5 * time.Minute,
		CreatedAt:   time.Now(),
	}
}

// CreateSubTask creates a new sub-task
func CreateSubTask(name, description string, capabilities []string) *SubTask {
	return &SubTask{
		Name:                 name,
		Description:          description,
		RequiredCapabilities: capabilities,
		Dependencies:         []string{},
		Input:                make(map[string]any),
	}
}

// GetOrchestrationError creates a standardized error
func GetOrchestrationError(code string, message string, err error) error {
	return fmt.Errorf("[%s] %s: %w", code, message, err)
}
