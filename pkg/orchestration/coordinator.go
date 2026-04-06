package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Execution Coordinator Implementation
// =============================================================================

// ExecutionCoordinator implements Coordinator interface
type ExecutionCoordinator struct {
	pool        ConcurrencyPool
	retryHandler RetryHandler
	config      *CoordinatorConfig
}

// CoordinatorConfig represents coordinator configuration
type CoordinatorConfig struct {
	MaxConcurrent int           `json:"max_concurrent"`
	Timeout       time.Duration `json:"timeout"`
	RetryCount    int           `json:"retry_count"`
	FailFast      bool          `json:"fail_fast"`
}

// DefaultCoordinatorConfig returns default coordinator config
func DefaultCoordinatorConfig() *CoordinatorConfig {
	return &CoordinatorConfig{
		MaxConcurrent: 5,
		Timeout:       5 * time.Minute,
		RetryCount:    3,
		FailFast:      false,
	}
}

// NewExecutionCoordinator creates a new execution coordinator
func NewExecutionCoordinator(config *CoordinatorConfig) *ExecutionCoordinator {
	if config == nil {
		config = DefaultCoordinatorConfig()
	}
	return &ExecutionCoordinator{
		pool:         NewDefaultConcurrencyPool(config.MaxConcurrent),
		retryHandler: NewRetryHandler(&ErrorHandlerConfig{MaxRetries: config.RetryCount}),
		config:       config,
	}
}

// Execute executes the plan, automatically selecting execution mode
func (c *ExecutionCoordinator) Execute(ctx context.Context, plan *OrchestrationPlan, agents map[string]Agent) ([]*SubResult, error) {
	if len(plan.ExecutionOrder) == 0 {
		return nil, nil
	}

	// Determine execution mode based on plan structure
	if len(plan.ExecutionOrder) == 1 {
		// Single layer - parallel execution
		return c.ExecuteParallel(ctx, plan.SubTasks, agents)
	}

	// Multiple layers - wave execution (hybrid)
	return c.executeWaves(ctx, plan, agents)
}

// ExecuteParallel executes sub-tasks in parallel
func (c *ExecutionCoordinator) ExecuteParallel(ctx context.Context, subTasks []*SubTask, agents map[string]Agent) ([]*SubResult, error) {
	if len(subTasks) == 0 {
		return nil, nil
	}

	results := make([]*SubResult, len(subTasks))
	var errors []error
	var errorsMu sync.Mutex

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.config.MaxConcurrent)

	for i, st := range subTasks {
		wg.Add(1)
		go func(idx int, subTask *SubTask) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[idx] = &SubResult{
					SubTaskName: subTask.Name,
					Success:     false,
					Error:       ctx.Err(),
				}
				return
			}

			// Execute with retry
			result := c.executeWithRetry(ctx, subTask, agents)
			results[idx] = result

			if !result.Success && c.config.FailFast {
				errorsMu.Lock()
				errors = append(errors, result.Error)
				errorsMu.Unlock()
			}
		}(i, st)
	}

	wg.Wait()

	if len(errors) > 0 && c.config.FailFast {
		return results, errors[0]
	}

	return results, nil
}

// ExecuteSequential executes sub-tasks sequentially
func (c *ExecutionCoordinator) ExecuteSequential(ctx context.Context, subTasks []*SubTask, agents map[string]Agent) ([]*SubResult, error) {
	if len(subTasks) == 0 {
		return nil, nil
	}

	results := make([]*SubResult, len(subTasks))
	contextData := make(map[string]any)

	for i, st := range subTasks {
		// Pass previous results as context
		if st.Input == nil {
			st.Input = make(map[string]any)
		}
		for k, v := range contextData {
			st.Input[k] = v
		}

		result := c.executeWithRetry(ctx, st, agents)
		results[i] = result

		if !result.Success {
			if c.config.FailFast {
				return results, result.Error
			}
			continue
		}

		// Store result for next task
		for k, v := range result.Output {
			contextData[k] = v
		}
	}

	return results, nil
}

// executeWaves executes tasks in waves (layers)
func (c *ExecutionCoordinator) executeWaves(ctx context.Context, plan *OrchestrationPlan, agents map[string]Agent) ([]*SubResult, error) {
	allResults := make(map[string]*SubResult)
	
	// Create sub-task map
	taskMap := make(map[string]*SubTask)
	for _, st := range plan.SubTasks {
		taskMap[st.Name] = st
	}

	// Execute each wave
	for waveIdx, wave := range plan.ExecutionOrder {
		// Check context
		select {
		case <-ctx.Done():
			return mapToSlice(allResults), ctx.Err()
		default:
		}

		// Get sub-tasks for this wave
		waveTasks := make([]*SubTask, 0, len(wave))
		for _, taskName := range wave {
			if st, exists := taskMap[taskName]; exists {
				// Inject dependency results
				st = c.injectDependencyResults(st, allResults)
				waveTasks = append(waveTasks, st)
			}
		}

		// Execute wave in parallel
		waveResults, err := c.ExecuteParallel(ctx, waveTasks, agents)
		if err != nil {
			return mapToSlice(allResults), err
		}

		// Store results
		for _, r := range waveResults {
			allResults[r.SubTaskName] = r
		}

		// Check for failures
		for _, r := range waveResults {
			if !r.Success && c.config.FailFast {
				return mapToSlice(allResults), r.Error
			}
		}

		// Update progress (would emit event)
		_ = waveIdx // wave index for progress tracking
	}

	return mapToSlice(allResults), nil
}

// executeWithRetry executes a sub-task with retry logic
func (c *ExecutionCoordinator) executeWithRetry(ctx context.Context, subTask *SubTask, agents map[string]Agent) *SubResult {
	agent, exists := agents[subTask.Name]
	if !exists {
		// Try to find agent by capability
		for _, a := range agents {
			if c.agentCanHandle(a, subTask) {
				agent = a
				break
			}
		}
	}

	if agent == nil {
		return &SubResult{
			SubTaskName: subTask.Name,
			Success:     false,
			Error:       fmt.Errorf("no agent available for sub-task: %s", subTask.Name),
		}
	}

	var lastErr error
	for retry := 0; retry <= c.config.RetryCount; retry++ {
		if retry > 0 {
			// Backoff
			select {
			case <-ctx.Done():
				return &SubResult{
					SubTaskName: subTask.Name,
					AgentName:   agent.Name(),
					Success:     false,
					Error:       ctx.Err(),
				}
			case <-time.After(time.Second * time.Duration(retry)):
			}
		}

		startTime := time.Now()
		result, err := agent.Execute(ctx, subTask)
		if result == nil {
			result = &SubResult{
				SubTaskName: subTask.Name,
				AgentName:   agent.Name(),
			}
		}

		if err != nil {
			lastErr = err
			result.Error = err
			result.Success = false
			
			// Check if error is retryable
			if orchErr, ok := err.(*OrchestrationError); ok && !orchErr.Recoverable {
				break
			}
			continue
		}

		result.StartTime = startTime
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		result.Success = true
		return result
	}

	return &SubResult{
		SubTaskName: subTask.Name,
		AgentName:   agent.Name(),
		Success:     false,
		Error:       lastErr,
	}
}

// agentCanHandle checks if agent can handle sub-task
func (c *ExecutionCoordinator) agentCanHandle(agent Agent, subTask *SubTask) bool {
	caps := agent.Capabilities()
	if caps == nil {
		return false
	}

	for _, reqCap := range subTask.RequiredCapabilities {
		found := false
		for _, skill := range caps.Skills {
			if skill == reqCap {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// injectDependencyResults injects results from dependencies into sub-task input
func (c *ExecutionCoordinator) injectDependencyResults(subTask *SubTask, results map[string]*SubResult) *SubTask {
	if len(subTask.Dependencies) == 0 {
		return subTask
	}

	// Create a copy to avoid modifying original
	st := &SubTask{
		Name:                 subTask.Name,
		ParentName:           subTask.ParentName,
		Description:          subTask.Description,
		RequiredCapabilities: subTask.RequiredCapabilities,
		Dependencies:         subTask.Dependencies,
		Priority:             subTask.Priority,
		Timeout:              subTask.Timeout,
		Input:                make(map[string]any),
	}

	// Copy original input
	for k, v := range subTask.Input {
		st.Input[k] = v
	}

	// Inject dependency results
	for _, dep := range subTask.Dependencies {
		if result, exists := results[dep]; exists && result.Success {
			for k, v := range result.Output {
				st.Input["dep_"+dep+"_"+k] = v
			}
		}
	}

	return st
}

// mapToSlice converts result map to slice
func mapToSlice(m map[string]*SubResult) []*SubResult {
	results := make([]*SubResult, 0, len(m))
	for _, r := range m {
		results = append(results, r)
	}
	return results
}

// =============================================================================
// Wave Execution Model
// =============================================================================

// WaveExecutor executes tasks in waves
type WaveExecutor struct {
	coordinator *ExecutionCoordinator
}

// NewWaveExecutor creates a new wave executor
func NewWaveExecutor(config *CoordinatorConfig) *WaveExecutor {
	return &WaveExecutor{
		coordinator: NewExecutionCoordinator(config),
	}
}

// ExecuteWave executes a single wave of tasks
func (e *WaveExecutor) ExecuteWave(ctx context.Context, tasks []*SubTask, agents map[string]Agent) ([]*SubResult, error) {
	return e.coordinator.ExecuteParallel(ctx, tasks, agents)
}

// =============================================================================
// Hybrid Execution Strategy
// =============================================================================

// HybridExecutor combines parallel and sequential execution
type HybridExecutor struct {
	parallelThreshold int
	coordinator       *ExecutionCoordinator
}

// NewHybridExecutor creates a new hybrid executor
func NewHybridExecutor(config *CoordinatorConfig) *HybridExecutor {
	return &HybridExecutor{
		parallelThreshold: 3,
		coordinator:       NewExecutionCoordinator(config),
	}
}

// Execute chooses execution mode based on task count
func (e *HybridExecutor) Execute(ctx context.Context, subTasks []*SubTask, agents map[string]Agent) ([]*SubResult, error) {
	if len(subTasks) <= e.parallelThreshold {
		return e.coordinator.ExecuteSequential(ctx, subTasks, agents)
	}
	return e.coordinator.ExecuteParallel(ctx, subTasks, agents)
}
