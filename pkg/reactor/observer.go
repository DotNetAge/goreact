package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
)

// BaseObserver provides base observer functionality
type BaseObserver struct {
	memory   *memory.Memory
	llmClient interface {
		Chat(ctx context.Context, messages []interface{}) (interface{}, error)
	}
	config *common.ObserverConfig
}

// NewBaseObserver creates a new BaseObserver
func NewBaseObserver(config *common.ObserverConfig) *BaseObserver {
	if config == nil {
		config = &common.ObserverConfig{
			EnableInsightExtraction:   true,
			EnableRelevanceAssessment: true,
			EnableMemoryUpdate:        true,
			MaxInsightsPerObservation: common.DefaultMaxInsightsPerObservation,
			RelevanceThreshold:        common.DefaultRelevanceThreshold,
			PersistRawResult:          false,
			MaxResultSize:             common.DefaultMaxResultSize,
		}
	}
	return &BaseObserver{config: config}
}

// SetMemory sets the memory instance
func (o *BaseObserver) SetMemory(mem *memory.Memory) {
	o.memory = mem
}

// Observe processes an action result
func (o *BaseObserver) Observe(ctx context.Context, result *core.ActionResult, state *core.State) (*core.Observation, error) {
	// Process result
	content, err := o.Process(result.Result)
	if err != nil {
		content = fmt.Sprintf("Failed to process result: %v", err)
	}
	
	// Create observation
	observation := core.NewObservation(content, o.getSource(result), result.Success)
	
	// Extract insights
	if o.config.EnableInsightExtraction {
		insights := o.extractInsights(ctx, result)
		observation.WithInsights(insights)
	}
	
	// Assess relevance
	if o.config.EnableRelevanceAssessment {
		relevance := o.assessRelevance(ctx, result, state)
		observation.WithRelevance(relevance)
	}
	
	// Handle error
	if !result.Success {
		observation.WithError(result.Error)
	}
	
	// Update memory
	if o.config.EnableMemoryUpdate {
		if err := o.UpdateMemory(ctx, observation, state); err != nil {
			// Log error but don't fail the observation
			observation.Metadata["memory_update_error"] = err.Error()
		}
	}
	
	// Update trajectory
	if state.Trajectory != nil {
		state.Trajectory.AddStep(state.GetLastThought(), state.GetLastAction(), observation)
	}
	
	return observation, nil
}

// Process processes the result into a string
func (o *BaseObserver) Process(result any) (string, error) {
	if result == nil {
		return "No result", nil
	}
	
	switch v := result.(type) {
	case string:
		return v, nil
	case map[string]any:
		// Convert map to readable string
		var parts []string
		for key, val := range v {
			parts = append(parts, fmt.Sprintf("%s: %v", key, val))
		}
		return strings.Join(parts, "\n"), nil
	case []any:
		// Convert slice to readable string
		var parts []string
		for i, val := range v {
			parts = append(parts, fmt.Sprintf("[%d] %v", i, val))
		}
		return strings.Join(parts, "\n"), nil
	default:
		return fmt.Sprintf("%v", result), nil
	}
}

// extractInsights extracts insights from the result
func (o *BaseObserver) extractInsights(ctx context.Context, result *core.ActionResult) []string {
	insights := []string{}
	
	if result == nil {
		return insights
	}
	
	// Basic insights based on result
	if result.Success {
		insights = append(insights, "Action completed successfully")
		
		// Extract insights from result data
		if resultMap, ok := result.Result.(map[string]any); ok {
			for key, val := range resultMap {
				if strings.Contains(key, "insight") || strings.Contains(key, "finding") {
					insights = append(insights, fmt.Sprintf("%v", val))
				}
			}
		}
	} else {
		insights = append(insights, fmt.Sprintf("Action failed: %s", result.Error))
	}
	
	// Limit insights
	if len(insights) > o.config.MaxInsightsPerObservation {
		insights = insights[:o.config.MaxInsightsPerObservation]
	}
	
	return insights
}

// assessRelevance assesses the relevance to the current task
func (o *BaseObserver) assessRelevance(ctx context.Context, result *core.ActionResult, state *core.State) float64 {
	if result == nil || state == nil {
		return 0.5
	}
	
	// Base relevance
	relevance := 0.5
	
	// Adjust based on success
	if result.Success {
		relevance += 0.2
	} else {
		relevance -= 0.1
	}
	
	// Adjust based on result content
	if result.Result != nil {
		relevance += 0.1
	}
	
	// Check if result relates to current plan step
	if state.Plan != nil {
		currentStep := state.Plan.GetCurrentStep()
		if currentStep != nil {
			// Check if action target matches expected action
			if result.ToolName != "" || result.SkillName != "" {
				relevance += 0.1
			}
		}
	}
	
	// Ensure relevance is within bounds
	if relevance > 1.0 {
		relevance = 1.0
	}
	if relevance < 0.0 {
		relevance = 0.0
	}
	
	return relevance
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

// UpdateMemory updates the memory with the observation
func (o *BaseObserver) UpdateMemory(ctx context.Context, observation *core.Observation, state *core.State) error {
	if o.memory == nil {
		return nil
	}
	
	// Create memory item node
	memoryItem := core.NewMemoryItemNode(
		state.SessionName,
		observation.Content,
		common.MemoryItemTypeObservation,
	)
	
	// Set additional properties
	memoryItem.Source = common.MemorySourceAction
	memoryItem.Importance = observation.Relevance
	
	// Add metadata
	memoryItem.Metadata["success"] = observation.Success
	memoryItem.Metadata["source"] = observation.Source
	if len(observation.Insights) > 0 {
		memoryItem.Metadata["insights"] = observation.Insights
	}
	if observation.Error != "" {
		memoryItem.Metadata["error"] = observation.Error
	}
	
	// Add to short-term memory (this also creates the session-observation relationship)
	_, err := o.memory.ShortTerms().Add(ctx, state.SessionName, memoryItem)
	if err != nil {
		return fmt.Errorf("failed to store observation in memory: %w", err)
	}
	
	return nil
}

// extractInsightsFromContent extracts insights from content using rules
func (o *BaseObserver) extractInsightsFromContent(content string) []string {
	insights := []string{}
	
	// Look for patterns that indicate insights
	patterns := []struct {
		keyword string
		prefix  string
	}{
		{"found", "Discovery: "},
		{"discovered", "Discovery: "},
		{"error", "Error: "},
		{"warning", "Warning: "},
		{"note", "Note: "},
		{"important", "Important: "},
	}
	
	contentLower := strings.ToLower(content)
	for _, pattern := range patterns {
		if strings.Contains(contentLower, pattern.keyword) {
			// Extract sentence containing the keyword
			sentences := strings.Split(content, ".")
			for _, sentence := range sentences {
				if strings.Contains(strings.ToLower(sentence), pattern.keyword) {
					insights = append(insights, pattern.prefix+strings.TrimSpace(sentence))
					break
				}
			}
		}
	}
	
	return insights
}

// calculateRelevanceScore calculates relevance score based on content analysis
func (o *BaseObserver) calculateRelevanceScore(content string, goal string) float64 {
	if content == "" || goal == "" {
		return 0.5
	}
	
	contentLower := strings.ToLower(content)
	goalLower := strings.ToLower(goal)
	
	// Tokenize
	contentWords := strings.Fields(contentLower)
	goalWords := strings.Fields(goalLower)
	
	// Count matching words
	matches := 0
	for _, goalWord := range goalWords {
		for _, contentWord := range contentWords {
			if goalWord == contentWord && len(goalWord) > 2 {
				matches++
				break
			}
		}
	}
	
	// Calculate score
	if len(goalWords) == 0 {
		return 0.5
	}
	
	score := float64(matches) / float64(len(goalWords))
	if score > 1.0 {
		score = 1.0
	}
	
	// Base score + match bonus
	return 0.3 + 0.7*score
}

// parseObservationFromJSON parses observation from JSON response
func (o *BaseObserver) parseObservationFromJSON(response string) (*core.Observation, error) {
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	jsonStr := response[jsonStart : jsonEnd+1]
	
	var parsed struct {
		Content   string   `json:"content"`
		Source    string   `json:"source"`
		Success   bool     `json:"success"`
		Insights  []string `json:"insights"`
		Relevance float64  `json:"relevance"`
		Error     string   `json:"error"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	observation := core.NewObservation(parsed.Content, parsed.Source, parsed.Success)
	observation.WithInsights(parsed.Insights)
	observation.WithRelevance(parsed.Relevance)
	if parsed.Error != "" {
		observation.WithError(parsed.Error)
	}
	
	return observation, nil
}
