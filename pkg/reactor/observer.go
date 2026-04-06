package reactor

import (
	"context"

	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
)

// BaseObserver provides base observer functionality
type BaseObserver struct {
	memory  any // Would be Memory
	config  *common.ObserverConfig
}

// NewBaseObserver creates a new BaseObserver
func NewBaseObserver(config *common.ObserverConfig) *BaseObserver {
	if config == nil {
		config = &common.ObserverConfig{
			EnableInsightExtraction:    true,
			EnableRelevanceAssessment:  true,
			EnableMemoryUpdate:         true,
			MaxInsightsPerObservation:  common.DefaultMaxInsightsPerObservation,
			RelevanceThreshold:         common.DefaultRelevanceThreshold,
			PersistRawResult:           false,
			MaxResultSize:              common.DefaultMaxResultSize,
		}
	}
	return &BaseObserver{config: config}
}

// Observe processes an action result
func (o *BaseObserver) Observe(ctx context.Context, result *core.ActionResult, state *core.State) (*core.Observation, error) {
	// Process result
	content, err := o.Process(result.Result)
	if err != nil {
		content = "Failed to process result"
	}
	
	// Create observation
	observation := core.NewObservation(content, o.getSource(result), result.Success)
	
	// Extract insights
	if o.config.EnableInsightExtraction {
		insights := o.extractInsights(result)
		observation.WithInsights(insights)
	}
	
	// Assess relevance
	if o.config.EnableRelevanceAssessment {
		relevance := o.assessRelevance(result, state)
		observation.WithRelevance(relevance)
	}
	
	// Handle error
	if !result.Success {
		observation.WithError(result.Error)
	}
	
	// Update memory
	if o.config.EnableMemoryUpdate {
		o.updateMemory(observation, state)
	}
	
	// Update trajectory
	if state.Trajectory != nil {
		state.Trajectory.AddStep(state.GetLastThought(), state.GetLastAction(), observation)
	}
	
	return observation, nil
}

// Process processes the result into a string
func (o *BaseObserver) Process(result any) (string, error) {
	// Would implement result processing
	// Simplified implementation
	return "Processed result", nil
}

// extractInsights extracts insights from the result
func (o *BaseObserver) extractInsights(result *core.ActionResult) []string {
	insights := []string{}
	
	// Would implement insight extraction
	// Simplified implementation
	
	if result.Success {
		insights = append(insights, "Action completed successfully")
	}
	
	return insights
}

// assessRelevance assesses the relevance to the current task
func (o *BaseObserver) assessRelevance(result *core.ActionResult, state *core.State) float64 {
	// Would implement relevance assessment
	// Simplified implementation
	return 0.7
}

// getSource gets the source of the action
func (o *BaseObserver) getSource(result *core.ActionResult) string {
	if result.ToolName != "" {
		return result.ToolName
	}
	if result.SkillName != "" {
		return result.SkillName
	}
	if result.SubAgentName != "" {
		return result.SubAgentName
	}
	return "unknown"
}

// updateMemory updates the memory with the observation
func (o *BaseObserver) updateMemory(observation *core.Observation, state *core.State) {
	// Would update memory
	// Simplified implementation
}
