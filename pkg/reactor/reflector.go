package reactor

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
)

// ReflectorConfig represents reflector configuration
type ReflectorConfig struct {
	MaxReflectionLength int
	EnableAutoRetry      bool
	MaxRetries           int
	MinReflectionScore   float64
}

// DefaultReflectorConfig returns the default reflector config
func DefaultReflectorConfig() *ReflectorConfig {
	return &ReflectorConfig{
		MaxReflectionLength: 1000,
		EnableAutoRetry:     true,
		MaxRetries:          3,
		MinReflectionScore:  0.7,
	}
}

// BaseReflector provides base reflector functionality
type BaseReflector struct {
	llmClient     any // Would be LLM client
	memory        any // Would be Memory
	config        *ReflectorConfig
	reflections   []*core.Reflection
}

// NewBaseReflector creates a new BaseReflector
func NewBaseReflector(config *ReflectorConfig) *BaseReflector {
	if config == nil {
		config = DefaultReflectorConfig()
	}
	return &BaseReflector{
		config:      config,
		reflections: []*core.Reflection{},
	}
}

// WithLLMClient sets the LLM client
func (r *BaseReflector) WithLLMClient(client any) *BaseReflector {
	r.llmClient = client
	return r
}

// WithMemory sets the memory
func (r *BaseReflector) WithMemory(memory any) *BaseReflector {
	r.memory = memory
	return r
}

// Reflect performs reflection on a failed execution
func (r *BaseReflector) Reflect(ctx context.Context, state *core.State) (*core.Reflection, error) {
	// Get the trajectory
	trajectory := state.Trajectory
	if trajectory == nil {
		// Build trajectory from state
		builder := core.NewTrajectoryBuilder(state)
		trajectory = builder.Build()
	}
	
	// Get failure context
	failureContext := trajectory.GetFailureContext()
	if len(failureContext) == 0 {
		// No explicit failure point, analyze entire trajectory
		failureContext = trajectory.Steps
	}
	
	// Analyze failure
	failureReason := r.analyzeFailure(failureContext, state)
	
	// Create reflection
	reflection := core.NewReflection(
		state.SessionName,
		trajectory.Name,
		failureReason,
	)
	
	// Generate analysis
	analysis := r.generateAnalysis(failureContext, state)
	reflection.WithAnalysis(analysis)
	
	// Generate heuristic
	heuristic := r.generateHeuristic(failureContext, state)
	reflection.WithHeuristic(heuristic)
	
	// Generate suggestions
	suggestions := r.generateSuggestions(failureContext, state)
	reflection.WithSuggestions(suggestions)
	
	// Calculate score
	score := r.calculateScore(failureContext, state)
	reflection.WithScore(score)
	
	// Store reflection
	r.reflections = append(r.reflections, reflection)
	
	return reflection, nil
}

// ReflectFromTrajectory performs reflection from a trajectory
func (r *BaseReflector) ReflectFromTrajectory(ctx context.Context, trajectory *core.Trajectory, state *core.State) (*core.Reflection, error) {
	if trajectory == nil {
		return nil, fmt.Errorf("trajectory is nil")
	}
	
	// Temporarily set trajectory for analysis
	originalTrajectory := state.Trajectory
	state.Trajectory = trajectory
	defer func() { state.Trajectory = originalTrajectory }()
	
	return r.Reflect(ctx, state)
}

// analyzeFailure analyzes the failure
func (r *BaseReflector) analyzeFailure(context []*core.TrajectoryStep, state *core.State) string {
	// Would use LLM to analyze
	// Simplified implementation
	
	for i, step := range context {
		if step.Observation != nil && !step.Observation.Success {
			return fmt.Sprintf("Execution failed at step %d: %s", 
				state.CurrentStep - len(context) + i,
				step.Observation.Error)
		}
	}
	
	return fmt.Sprintf("Execution failed at step %d", state.CurrentStep)
}

// generateAnalysis generates detailed analysis
func (r *BaseReflector) generateAnalysis(context []*core.TrajectoryStep, state *core.State) string {
	// Would use LLM to generate
	// Simplified implementation
	
	analysis := "Analysis of execution failure:\n"
	
	for i, step := range context {
		if step.Thought != nil {
			analysis += fmt.Sprintf("Step %d - Thought: %s\n", i, step.Thought.Content)
		}
		if step.Action != nil {
			analysis += fmt.Sprintf("Step %d - Action: %s on %s\n", i, step.Action.Type, step.Action.Target)
		}
		if step.Observation != nil {
			status := "success"
			if !step.Observation.Success {
				status = "failed"
			}
			analysis += fmt.Sprintf("Step %d - Observation: %s (%s)\n", i, step.Observation.Content, status)
		}
	}
	
	return analysis
}

// generateHeuristic generates a heuristic lesson
func (r *BaseReflector) generateHeuristic(context []*core.TrajectoryStep, state *core.State) string {
	// Would use LLM to generate
	// Simplified implementation based on failure patterns
	
	for _, step := range context {
		if step.Action != nil && step.Observation != nil && !step.Observation.Success {
			switch step.Action.Type {
			case common.ActionTypeToolCall:
				return "Verify tool parameters and availability before execution"
			case common.ActionTypeSkillInvoke:
				return "Ensure skill is properly configured and all dependencies are met"
			case common.ActionTypeSubAgentDelegate:
				return "Verify sub-agent capabilities before delegation"
			}
		}
	}
	
	return "Always validate preconditions before executing actions"
}

// generateSuggestions generates actionable suggestions
func (r *BaseReflector) generateSuggestions(context []*core.TrajectoryStep, state *core.State) []string {
	// Would use LLM to generate
	// Simplified implementation
	
	suggestions := []string{
		"Review the execution trajectory for potential improvements",
		"Consider alternative approaches for failed steps",
		"Add more validation before critical actions",
	}
	
	// Add context-specific suggestions
	for _, step := range context {
		if step.Action != nil && step.Observation != nil && !step.Observation.Success {
			suggestions = append(suggestions, 
				fmt.Sprintf("Investigate why action '%s' failed and add error handling", step.Action.Target))
		}
	}
	
	return suggestions
}

// calculateScore calculates the reflection quality score
func (r *BaseReflector) calculateScore(context []*core.TrajectoryStep, state *core.State) float64 {
	// Base score
	score := 0.5
	
	// Adjust based on available context
	if len(context) >= 3 {
		score += 0.1
	}
	
	// Adjust based on trajectory completeness
	if state.Trajectory != nil {
		successRate := state.Trajectory.GetSuccessRate()
		// Lower success rate means more valuable reflection
		if successRate < 0.5 {
			score += 0.2
		}
	}
	
	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// GenerateHeuristic generates a heuristic from a reflection
func (r *BaseReflector) GenerateHeuristic(reflection *core.Reflection) string {
	if reflection == nil {
		return ""
	}
	
	// Return the existing heuristic or generate a new one
	if reflection.Heuristic != "" {
		return reflection.Heuristic
	}
	
	return fmt.Sprintf("When encountering '%s', consider: %s", 
		reflection.FailureReason, 
		reflection.Analysis)
}

// StoreReflection stores a reflection (would integrate with memory)
func (r *BaseReflector) StoreReflection(reflection *core.Reflection, state *core.State) error {
	if reflection == nil {
		return fmt.Errorf("reflection is nil")
	}
	
	// Would store in memory
	// For now, just keep in local cache
	r.reflections = append(r.reflections, reflection)
	
	return nil
}

// RetrieveRelevantReflections retrieves relevant reflections for a query
func (r *BaseReflector) RetrieveRelevantReflections(ctx context.Context, query string, limit int) ([]*core.Reflection, error) {
	// Would use semantic search in memory
	// Simplified implementation - return recent reflections
	
	if limit <= 0 || limit > len(r.reflections) {
		limit = len(r.reflections)
	}
	
	start := len(r.reflections) - limit
	if start < 0 {
		start = 0
	}
	
	return r.reflections[start:], nil
}

// GetReflections returns all stored reflections
func (r *BaseReflector) GetReflections() []*core.Reflection {
	return r.reflections
}
