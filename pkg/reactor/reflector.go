package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
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
	llmClient   core.Client
	memory      *memory.Memory
	config      *ReflectorConfig
	reflections []*goreactcore.Reflection
}

// NewBaseReflector creates a new BaseReflector
func NewBaseReflector(config *ReflectorConfig) *BaseReflector {
	if config == nil {
		config = DefaultReflectorConfig()
	}
	return &BaseReflector{
		config:      config,
		reflections: []*goreactcore.Reflection{},
	}
}

// WithLLMClient sets the LLM client
func (r *BaseReflector) WithLLMClient(client core.Client) *BaseReflector {
	r.llmClient = client
	return r
}

// WithMemory sets the memory
func (r *BaseReflector) WithMemory(mem *memory.Memory) *BaseReflector {
	r.memory = mem
	return r
}

// Reflect performs reflection on a failed execution
func (r *BaseReflector) Reflect(ctx context.Context, state *goreactcore.State) (*goreactcore.Reflection, error) {
	// Get the trajectory
	trajectory := state.Trajectory
	if trajectory == nil {
		// Build trajectory from state
		builder := goreactcore.NewTrajectoryBuilder(state)
		trajectory = builder.Build()
	}
	
	// Get failure context
	failureContext := trajectory.GetFailureContext()
	if len(failureContext) == 0 {
		// No explicit failure point, analyze entire trajectory
		failureContext = trajectory.Steps
	}
	
	// Use LLM for analysis if available
	if r.llmClient != nil {
		return r.reflectWithLLM(ctx, failureContext, state)
	}
	
	// Fallback: Analyze failure without LLM
	return r.analyzeFailureLocally(failureContext, state)
}

// reflectWithLLM performs reflection using LLM
func (r *BaseReflector) reflectWithLLM(ctx context.Context, context []*goreactcore.TrajectoryStep, state *goreactcore.State) (*goreactcore.Reflection, error) {
	// Build reflection prompt
	prompt := r.buildReflectionPrompt(context, state)
	
	// Call LLM
	resp, err := r.llmClient.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Parse response
	reflection, err := r.parseReflectionResponse(resp.Content, state)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reflection: %w", err)
	}
	
	// Store reflection
	r.reflections = append(r.reflections, reflection)
	
	return reflection, nil
}

// analyzeFailureLocally analyzes failure without LLM
func (r *BaseReflector) analyzeFailureLocally(context []*goreactcore.TrajectoryStep, state *goreactcore.State) (*goreactcore.Reflection, error) {
	// Analyze failure
	failureReason := r.analyzeFailure(context, state)
	
	// Create reflection
	reflection := goreactcore.NewReflection(
		state.SessionName,
		state.Trajectory.Name,
		failureReason,
	)
	
	// Generate analysis
	analysis := r.generateAnalysis(context, state)
	reflection.WithAnalysis(analysis)
	
	// Generate heuristic
	heuristic := r.generateHeuristic(context, state)
	reflection.WithHeuristic(heuristic)
	
	// Generate suggestions
	suggestions := r.generateSuggestions(context, state)
	reflection.WithSuggestions(suggestions)
	
	// Calculate score
	score := r.calculateScore(context, state)
	reflection.WithScore(score)
	
	// Store reflection
	r.reflections = append(r.reflections, reflection)
	
	return reflection, nil
}

// ReflectFromTrajectory performs reflection from a trajectory
func (r *BaseReflector) ReflectFromTrajectory(ctx context.Context, trajectory *goreactcore.Trajectory, state *goreactcore.State) (*goreactcore.Reflection, error) {
	if trajectory == nil {
		return nil, fmt.Errorf("trajectory is nil")
	}
	
	// Temporarily set trajectory for analysis
	originalTrajectory := state.Trajectory
	state.Trajectory = trajectory
	defer func() { state.Trajectory = originalTrajectory }()
	
	return r.Reflect(ctx, state)
}

// buildReflectionPrompt builds the prompt for LLM-based reflection
func (r *BaseReflector) buildReflectionPrompt(context []*goreactcore.TrajectoryStep, state *goreactcore.State) string {
	var sb strings.Builder
	
	sb.WriteString(`Analyze this failed execution and provide reflection.

## Execution Context
- Session: ` + state.SessionName + `
- Step: ` + fmt.Sprintf("%d", state.CurrentStep) + ` / ` + fmt.Sprintf("%d", state.MaxSteps) + `
- Input: ` + state.Input + `

## Trajectory Steps
`)
	
	for i, step := range context {
		sb.WriteString(fmt.Sprintf("\n### Step %d\n", i+1))
		if step.Thought != nil {
			sb.WriteString(fmt.Sprintf("**Thought**: %s\n", step.Thought.Content))
		}
		if step.Action != nil {
			sb.WriteString(fmt.Sprintf("**Action**: %s on %s\n", step.Action.Type, step.Action.Target))
			if len(step.Action.Params) > 0 {
				sb.WriteString("**Params**: ")
				for k, v := range step.Action.Params {
					sb.WriteString(fmt.Sprintf("%s=%v ", k, v))
				}
				sb.WriteString("\n")
			}
		}
		if step.Observation != nil {
			status := "success"
			if !step.Observation.Success {
				status = "failed"
			}
			sb.WriteString(fmt.Sprintf("**Observation** (%s): %s\n", status, step.Observation.Content))
			if step.Observation.Error != "" {
				sb.WriteString(fmt.Sprintf("**Error**: %s\n", step.Observation.Error))
			}
		}
	}
	
	sb.WriteString(`
## Analysis Required
1. Identify the root cause of failure
2. Analyze what went wrong
3. Generate a heuristic lesson
4. Provide actionable suggestions

## Output Format (JSON)
{
  "failure_reason": "description of what failed",
  "analysis": "detailed analysis of the failure",
  "heuristic": "a memorable lesson for future",
  "suggestions": ["suggestion1", "suggestion2"],
  "confidence": 0.0-1.0
}`)
	
	return sb.String()
}

// parseReflectionResponse parses the LLM response into a Reflection
func (r *BaseReflector) parseReflectionResponse(response string, state *goreactcore.State) (*goreactcore.Reflection, error) {
	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	jsonStr := response[jsonStart : jsonEnd+1]
	
	var parsed struct {
		FailureReason string   `json:"failure_reason"`
		Analysis      string   `json:"analysis"`
		Heuristic     string   `json:"heuristic"`
		Suggestions   []string `json:"suggestions"`
		Confidence    float64  `json:"confidence"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	reflection := goreactcore.NewReflection(
		state.SessionName,
		state.Trajectory.Name,
		parsed.FailureReason,
	)
	reflection.WithAnalysis(parsed.Analysis)
	reflection.WithHeuristic(parsed.Heuristic)
	reflection.WithSuggestions(parsed.Suggestions)
	reflection.WithScore(parsed.Confidence)
	
	return reflection, nil
}

// analyzeFailure analyzes the failure
func (r *BaseReflector) analyzeFailure(context []*goreactcore.TrajectoryStep, state *goreactcore.State) string {
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
func (r *BaseReflector) generateAnalysis(context []*goreactcore.TrajectoryStep, state *goreactcore.State) string {
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
func (r *BaseReflector) generateHeuristic(context []*goreactcore.TrajectoryStep, state *goreactcore.State) string {
	for _, step := range context {
		if step.Action != nil && step.Observation != nil && !step.Observation.Success {
			switch step.Action.Type {
			case goreactcommon.ActionTypeToolCall:
				return "Verify tool parameters and availability before execution"
			case goreactcommon.ActionTypeSkillInvoke:
				return "Ensure skill is properly configured and all dependencies are met"
			case goreactcommon.ActionTypeSubAgentDelegate:
				return "Verify sub-agent capabilities before delegation"
			}
		}
	}
	
	return "Always validate preconditions before executing actions"
}

// generateSuggestions generates actionable suggestions
func (r *BaseReflector) generateSuggestions(context []*goreactcore.TrajectoryStep, state *goreactcore.State) []string {
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
func (r *BaseReflector) calculateScore(context []*goreactcore.TrajectoryStep, state *goreactcore.State) float64 {
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
func (r *BaseReflector) GenerateHeuristic(reflection *goreactcore.Reflection) string {
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

// StoreReflection stores a reflection in memory
func (r *BaseReflector) StoreReflection(ctx context.Context, reflection *goreactcore.Reflection, state *goreactcore.State) error {
	if reflection == nil {
		return fmt.Errorf("reflection is nil")
	}
	
	// Store in local cache
	r.reflections = append(r.reflections, reflection)
	
	// Store in memory if available
	if r.memory != nil {
		reflectionNode := goreactcore.NewReflectionNode(state.SessionName, reflection.TrajectoryName, reflection.FailureReason)
		reflectionNode.Analysis = reflection.Analysis
		reflectionNode.Heuristic = reflection.Heuristic
		reflectionNode.Suggestions = reflection.Suggestions
		reflectionNode.Score = reflection.Score
		
		if err := r.memory.Reflections().Add(ctx, reflectionNode); err != nil {
			return fmt.Errorf("failed to store reflection in memory: %w", err)
		}
	}
	
	return nil
}

// RetrieveRelevantReflections retrieves relevant reflections for a query
func (r *BaseReflector) RetrieveRelevantReflections(ctx context.Context, query string, limit int) ([]*goreactcore.Reflection, error) {
	// Try to retrieve from memory first
	if r.memory != nil {
		reflectionNodes, err := r.memory.Reflections().List(ctx)
		if err == nil && len(reflectionNodes) > 0 {
			// Convert nodes to reflections
			reflections := make([]*goreactcore.Reflection, 0, limit)
			for i := len(reflectionNodes) - 1; i >= 0 && len(reflections) < limit; i-- {
				node := reflectionNodes[i]
				reflection := &goreactcore.Reflection{
					Name:          node.Name,
					TrajectoryName: node.TrajectoryName,
					FailureReason: node.FailureReason,
					Analysis:      node.Analysis,
					Heuristic:     node.Heuristic,
					Suggestions:   node.Suggestions,
					Score:         node.Score,
				}
				reflections = append(reflections, reflection)
			}
			return reflections, nil
		}
	}
	
	// Fallback to local cache
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
func (r *BaseReflector) GetReflections() []*goreactcore.Reflection {
	return r.reflections
}
