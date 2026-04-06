package reactor

import (
	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
)

// TerminatorConfig represents terminator configuration
type TerminatorConfig struct {
	MaxSteps            int
	MaxRetries          int
	StuckThreshold      int
	EnableStuckDetection bool
}

// DefaultTerminatorConfig returns the default terminator config
func DefaultTerminatorConfig() *TerminatorConfig {
	return &TerminatorConfig{
		MaxSteps:            common.DefaultMaxSteps,
		MaxRetries:          common.DefaultMaxRetries,
		StuckThreshold:      3,
		EnableStuckDetection: true,
	}
}

// BaseTerminator provides base terminator functionality
type BaseTerminator struct {
	config    *TerminatorConfig
	evaluators []core.Evaluator
}

// NewBaseTerminator creates a new BaseTerminator
func NewBaseTerminator(config *TerminatorConfig) *BaseTerminator {
	if config == nil {
		config = DefaultTerminatorConfig()
	}
	return &BaseTerminator{
		config:     config,
		evaluators: []core.Evaluator{},
	}
}

// WithEvaluator adds an evaluator
func (t *BaseTerminator) WithEvaluator(evaluator core.Evaluator) *BaseTerminator {
	t.evaluators = append(t.evaluators, evaluator)
	return t
}

// ShouldTerminate checks if execution should terminate
func (t *BaseTerminator) ShouldTerminate(state *core.State) bool {
	// Check max steps
	if state.CurrentStep >= t.config.MaxSteps {
		return true
	}
	
	// Check max retries
	if state.RetryCount >= t.config.MaxRetries {
		return true
	}
	
	// Check status
	if state.Status == common.StatusPaused || state.Status == common.StatusCanceled {
		return true
	}
	
	// Check for final answer
	if len(state.Thoughts) > 0 {
		lastThought := state.Thoughts[len(state.Thoughts)-1]
		if lastThought.IsAnswer() {
			return true
		}
	}
	
	// Check for stuck
	if t.config.EnableStuckDetection && t.IsStuck(state) {
		return true
	}
	
	return false
}

// Reason returns the termination reason
func (t *BaseTerminator) Reason(state *core.State) string {
	if state.CurrentStep >= t.config.MaxSteps {
		return "Max steps reached"
	}
	if state.RetryCount >= t.config.MaxRetries {
		return "Max retries exceeded"
	}
	if state.Status == common.StatusPaused {
		return "Execution paused by user"
	}
	if state.Status == common.StatusCanceled {
		return "Execution canceled"
	}
	if t.IsStuck(state) {
		return "Stuck in loop detected"
	}
	if len(state.Thoughts) > 0 {
		lastThought := state.Thoughts[len(state.Thoughts)-1]
		if lastThought.IsAnswer() {
			return "Goal achieved"
		}
	}
	return ""
}

// TerminationReason returns the termination reason as enum
func (t *BaseTerminator) TerminationReason(state *core.State) common.TerminationReason {
	if state.CurrentStep >= t.config.MaxSteps {
		return common.TerminationReasonMaxSteps
	}
	if state.RetryCount >= t.config.MaxRetries {
		return common.TerminationReasonMaxRetries
	}
	if state.Status == common.StatusPaused {
		return common.TerminationReasonUserInterrupted
	}
	if state.Status == common.StatusCanceled {
		return common.TerminationReasonUserInterrupted
	}
	if t.IsStuck(state) {
		return common.TerminationReasonStuckDetected
	}
	if len(state.Thoughts) > 0 {
		lastThought := state.Thoughts[len(state.Thoughts)-1]
		if lastThought.IsAnswer() {
			return common.TerminationReasonGoalAchieved
		}
	}
	return common.TerminationReasonErrorOccurred
}

// IsStuck detects if the execution is stuck in a loop
func (t *BaseTerminator) IsStuck(state *core.State) bool {
	if !t.config.EnableStuckDetection {
		return false
	}
	
	// Need at least stuckThreshold * 2 observations to detect patterns
	if len(state.Observations) < t.config.StuckThreshold*2 {
		return false
	}
	
	// Check for repeated identical actions
	actionCounts := make(map[string]int)
	start := len(state.Actions) - t.config.StuckThreshold*2
	if start < 0 {
		start = 0
	}
	
	for i := start; i < len(state.Actions); i++ {
		if state.Actions[i] == nil {
			continue
		}
		key := string(state.Actions[i].Type) + ":" + state.Actions[i].Target
		actionCounts[key]++
		if actionCounts[key] >= t.config.StuckThreshold {
			return true
		}
	}
	
	// Check for repeated errors
	errorCount := 0
	start = len(state.Observations) - t.config.StuckThreshold
	if start < 0 {
		start = 0
	}
	
	for i := start; i < len(state.Observations); i++ {
		if state.Observations[i] != nil && !state.Observations[i].Success {
			errorCount++
		}
	}
	
	return errorCount >= t.config.StuckThreshold
}

// Evaluate evaluates the final state
func (t *BaseTerminator) Evaluate(state *core.State) *core.EvaluationResult {
	// Check if we have evaluators
	if len(t.evaluators) > 0 {
		// Run all evaluators and aggregate results
		combinedResult := core.NewEvaluationResult(true, "combined evaluation")
		totalScore := 0.0
		
		for _, evaluator := range t.evaluators {
			result := evaluator.Evaluate(state)
			totalScore += result.Score
			if !result.Success {
				combinedResult.Success = false
			}
			for _, s := range result.Suggestions {
				combinedResult.AddSuggestion(s)
			}
			for k, v := range result.Metrics {
				combinedResult.WithMetric(k, v)
			}
		}
		
		combinedResult.Score = totalScore / float64(len(t.evaluators))
		combinedResult.TerminationReason = t.TerminationReason(state)
		return combinedResult
	}
	
	// Default evaluation
	result := core.NewEvaluationResult(
		state.Status == common.StatusCompleted,
		t.Reason(state),
	)
	result.TerminationReason = t.TerminationReason(state)
	
	// Calculate score based on trajectory
	if state.Trajectory != nil {
		result.Score = state.Trajectory.GetSuccessRate()
		result.WithMetric("success_rate", result.Score)
		result.WithMetric("step_count", float64(state.Trajectory.GetStepCount()))
	}
	
	return result
}
