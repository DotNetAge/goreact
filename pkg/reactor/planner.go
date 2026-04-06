package reactor

import (
	"context"
	"fmt"
	"strings"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
)

// PlannerConfig represents planner configuration
type PlannerConfig struct {
	MaxPlanSteps    int
	EnableReplan    bool
	ReplanThreshold float64
}

// DefaultPlannerConfig returns the default planner config
func DefaultPlannerConfig() *PlannerConfig {
	return &PlannerConfig{
		MaxPlanSteps:    10,
		EnableReplan:    true,
		ReplanThreshold: 0.5,
	}
}

// BasePlanner provides base planner functionality
type BasePlanner struct {
	llmClient   core.Client
	memory      *memory.Memory
	config      *PlannerConfig
	planHistory []*goreactcore.Plan
}

// NewBasePlanner creates a new BasePlanner
func NewBasePlanner(config *PlannerConfig) *BasePlanner {
	if config == nil {
		config = DefaultPlannerConfig()
	}
	return &BasePlanner{
		config:      config,
		planHistory: []*goreactcore.Plan{},
	}
}

// WithLLMClient sets the LLM client
func (p *BasePlanner) WithLLMClient(client core.Client) *BasePlanner {
	p.llmClient = client
	return p
}

// WithMemory sets the memory
func (p *BasePlanner) WithMemory(mem *memory.Memory) *BasePlanner {
	p.memory = mem
	return p
}

// Plan creates an execution plan
func (p *BasePlanner) Plan(ctx context.Context, input string, state *goreactcore.State) (*goreactcore.Plan, error) {
	// Try to retrieve similar plans from memory
	var similarPlans []*goreactcore.Plan
	var reflections []*goreactcore.Reflection
	
	if p.memory != nil {
		similarPlans = p.getSimilarPlansFromMemory(ctx, input, 3)
		reflections = p.getReflectionsFromMemory(ctx, input, 5)
	}
	
	// Use LLM for planning if available
	if p.llmClient != nil {
		return p.planWithLLM(ctx, input, state, similarPlans, reflections)
	}
	
	// Fallback: Create plan without LLM
	return p.createDefaultPlan(input, state, similarPlans)
}

// planWithLLM creates a plan using LLM
func (p *BasePlanner) planWithLLM(ctx context.Context, input string, state *goreactcore.State, similarPlans []*goreactcore.Plan, reflections []*goreactcore.Reflection) (*goreactcore.Plan, error) {
	// Build planning prompt
	prompt := p.buildPlanPrompt(input, state, similarPlans, reflections)
	
	// Call LLM
	resp, err := p.llmClient.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Parse response into plan
	plan, err := p.parsePlanResponse(resp.Content, state)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}
	
	// Validate plan
	if err := p.Validate(plan); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}
	
	// Store in history
	p.planHistory = append(p.planHistory, plan)
	
	// Store plan in memory
	if p.memory != nil {
		planNode := goreactcore.NewPlanNode(state.SessionName, plan.Goal)
		planNode.Steps = plan.Steps
		planNode.Status = plan.Status
		if err := p.memory.Plans().Add(ctx, planNode); err != nil {
			// Log error but don't fail planning
		}
	}
	
	return plan, nil
}

// buildPlanPrompt builds the planning prompt
func (p *BasePlanner) buildPlanPrompt(input string, state *goreactcore.State, similarPlans []*goreactcore.Plan, reflections []*goreactcore.Reflection) string {
	var sb strings.Builder
	
	sb.WriteString(`Create an execution plan for the following task.

## Task
`)
	sb.WriteString(input)
	
	// Add context if available
	if state.Context != nil {
		sb.WriteString("\n## Context\n")
		for k, v := range state.Context {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
	}
	
	// Add similar plans as reference
	if len(similarPlans) > 0 {
		sb.WriteString("\n## Similar Plans (for reference)\n")
		for i, plan := range similarPlans {
			sb.WriteString(fmt.Sprintf("\n### Plan %d\n", i+1))
			sb.WriteString(fmt.Sprintf("Goal: %s\n", plan.Goal))
			sb.WriteString("Steps:\n")
			for j, step := range plan.Steps {
				sb.WriteString(fmt.Sprintf("%d. %s\n", j+1, step.Description))
			}
		}
	}
	
	// Add relevant reflections
	if len(reflections) > 0 {
		sb.WriteString("\n## Lessons Learned (avoid past mistakes)\n")
		for _, ref := range reflections {
			sb.WriteString(fmt.Sprintf("- %s\n", ref.Heuristic))
		}
	}
	
	sb.WriteString(`
## Planning Guidelines
1. Break down the task into logical steps
2. Each step should have a clear description
3. Consider dependencies between steps
4. Limit to ` + fmt.Sprintf("%d", p.config.MaxPlanSteps) + ` steps

## Output Format (JSON)
{
  "goal": "clear goal statement",
  "steps": [
    {"description": "step description", "expected_action": "action type"},
    ...
  ]
}`)
	
	return sb.String()
}

// parsePlanResponse parses the LLM response into a Plan
func (p *BasePlanner) parsePlanResponse(response string, state *goreactcore.State) (*goreactcore.Plan, error) {
	var parsed struct {
		Goal  string `json:"goal"`
		Steps []struct {
			Description    string `json:"description"`
			ExpectedAction string `json:"expected_action"`
		} `json:"steps"`
	}

	if err := goreactcommon.ParseJSONObject(response, &parsed); err != nil {
		return nil, err
	}
	
	// Create plan
	plan := goreactcore.NewPlan(state.SessionName, parsed.Goal)
	
	// Add steps
	for i, step := range parsed.Steps {
		planStep := &goreactcore.PlanStep{
			Index:       i,
			Description: step.Description,
			Action:      step.ExpectedAction,
			Status:      goreactcommon.StepStatusPending,
		}
		plan.Steps = append(plan.Steps, planStep)
	}
	
	plan.Status = goreactcommon.PlanStatusRunning
	
	return plan, nil
}

// createDefaultPlan creates a default plan without LLM
func (p *BasePlanner) createDefaultPlan(input string, state *goreactcore.State, similarPlans []*goreactcore.Plan) (*goreactcore.Plan, error) {
	plan := goreactcore.NewPlan(state.SessionName, input)
	
	// Check if we have similar plans to use as reference
	if len(similarPlans) > 0 && similarPlans[0].Status == goreactcommon.PlanStatusCompleted {
		// Use similar plan structure
		for _, step := range similarPlans[0].Steps {
			plan.Steps = append(plan.Steps, &goreactcore.PlanStep{
				Index:       step.Index,
				Description: step.Description,
				Status:      goreactcommon.StepStatusPending,
			})
		}
	} else {
		// Create default steps
		plan.AddStep("analyze", "Analyze the input and understand the task")
		plan.AddStep("plan", "Create a detailed execution plan")
		plan.AddStep("execute", "Execute the planned actions")
		plan.AddStep("verify", "Verify the results")
		plan.AddStep("respond", "Generate the final response")
	}
	
	// Limit steps
	if len(plan.Steps) > p.config.MaxPlanSteps {
		plan.Steps = plan.Steps[:p.config.MaxPlanSteps]
	}
	
	plan.Status = goreactcommon.PlanStatusRunning
	
	// Store in history
	p.planHistory = append(p.planHistory, plan)
	
	// Store plan in memory
	if p.memory != nil {
		planNode := goreactcore.NewPlanNode(state.SessionName, plan.Goal)
		planNode.Steps = plan.Steps
		planNode.Status = plan.Status
		
		if err := p.memory.Plans().Add(context.Background(), planNode); err != nil {
			// Log error but don't fail planning
		}
	}
	
	return plan, nil
}

// Replan creates a new plan based on current state
func (p *BasePlanner) Replan(ctx context.Context, state *goreactcore.State) (*goreactcore.Plan, error) {
	if !p.config.EnableReplan {
		return state.Plan, nil
	}
	
	if state.Plan == nil {
		return p.Plan(ctx, state.Input, state)
	}
	
	// Use LLM for replanning if available
	if p.llmClient != nil {
		return p.replanWithLLM(ctx, state)
	}
	
	// Fallback: Modify current plan
	plan := state.Plan
	plan.Status = goreactcommon.PlanStatusRevised
	
	// Add recovery step
	plan.AddStep("recover", "Recover from the failure and adjust approach")
	
	// Store in history
	p.planHistory = append(p.planHistory, plan)
	
	return plan, nil
}

// replanWithLLM creates a revised plan using LLM
func (p *BasePlanner) replanWithLLM(ctx context.Context, state *goreactcore.State) (*goreactcore.Plan, error) {
	// Build replanning prompt
	prompt := p.buildReplanPrompt(state)
	
	// Call LLM
	resp, err := p.llmClient.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Parse response
	plan, err := p.parsePlanResponse(resp.Content, state)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}
	
	plan.Status = goreactcommon.PlanStatusRevised
	
	// Store in history
	p.planHistory = append(p.planHistory, plan)
	
	return plan, nil
}

// buildReplanPrompt builds the replanning prompt
func (p *BasePlanner) buildReplanPrompt(state *goreactcore.State) string {
	var sb strings.Builder
	
	sb.WriteString(`Revise the execution plan based on the current state.

## Original Goal
`)
	sb.WriteString(state.Input)
	
	sb.WriteString("\n## Current Plan\n")
	if state.Plan != nil {
		for i, step := range state.Plan.Steps {
			status := "pending"
			if i < state.Plan.CurrentStepIndex {
				status = "completed"
			} else if i == state.Plan.CurrentStepIndex {
				status = "current"
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, status, step.Description))
		}
	}
	
	sb.WriteString("\n## Current Status\n")
	sb.WriteString(fmt.Sprintf("Step: %d / %d\n", state.CurrentStep, state.MaxSteps))
	
	// Add last observation
	if len(state.Observations) > 0 {
		lastObs := state.Observations[len(state.Observations)-1]
		sb.WriteString(fmt.Sprintf("\n## Last Observation\n%s\n", lastObs.Content))
		if !lastObs.Success {
			sb.WriteString(fmt.Sprintf("Error: %s\n", lastObs.Error))
		}
	}
	
	sb.WriteString(`
## Replanning Guidelines
1. Analyze what went wrong
2. Adjust the remaining steps
3. Add recovery steps if needed
4. Keep the goal unchanged

## Output Format (JSON)
{
  "goal": "same or refined goal",
  "steps": [
    {"description": "step description", "expected_action": "action type"},
    ...
  ]
}`)
	
	return sb.String()
}

// Validate validates a plan
func (p *BasePlanner) Validate(plan *goreactcore.Plan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	
	if plan.Goal == "" {
		return fmt.Errorf("plan goal is empty")
	}
	
	if len(plan.Steps) == 0 {
		return fmt.Errorf("plan has no steps")
	}
	
	// Check for circular dependencies
	visited := make(map[int]bool)
	for _, step := range plan.Steps {
		if err := p.validateStepDependencies(step.Index, plan, visited); err != nil {
			return err
		}
	}
	
	return nil
}

// validateStepDependencies validates step dependencies for cycles
func (p *BasePlanner) validateStepDependencies(stepIndex int, plan *goreactcore.Plan, visited map[int]bool) error {
	if visited[stepIndex] {
		return fmt.Errorf("circular dependency detected at step %d", stepIndex)
	}
	
	visited[stepIndex] = true
	defer func() { visited[stepIndex] = false }()
	
	if stepIndex >= len(plan.Steps) {
		return nil
	}
	
	step := plan.Steps[stepIndex]
	for _, depIndex := range step.Dependencies {
		if err := p.validateStepDependencies(depIndex, plan, visited); err != nil {
			return err
		}
	}
	
	return nil
}

// NeedReplan checks if replanning is needed
func (p *BasePlanner) NeedReplan(state *goreactcore.State) bool {
	if !p.config.EnableReplan || state.Plan == nil {
		return false
	}
	
	// Check if current step is deviating from expected
	currentStep := state.Plan.GetCurrentStep()
	if currentStep == nil {
		return false
	}
	
	// Check if last observation deviates from expected outcome
	if len(state.Observations) > 0 {
		lastObs := state.Observations[len(state.Observations)-1]
		if !lastObs.Success {
			// Failure might need replanning
			return true
		}
		
		// Check relevance score
		if lastObs.Relevance < p.config.ReplanThreshold {
			return true
		}
	}
	
	return false
}

// GetPlanHistory returns the plan history
func (p *BasePlanner) GetPlanHistory() []*goreactcore.Plan {
	return p.planHistory
}

// GetSimilarPlans retrieves similar plans from history
func (p *BasePlanner) GetSimilarPlans(goal string, limit int) []*goreactcore.Plan {
	if limit <= 0 || limit > len(p.planHistory) {
		limit = len(p.planHistory)
	}
	
	start := len(p.planHistory) - limit
	if start < 0 {
		start = 0
	}
	
	return p.planHistory[start:]
}

// getSimilarPlansFromMemory retrieves similar plans from memory
func (p *BasePlanner) getSimilarPlansFromMemory(ctx context.Context, goal string, limit int) []*goreactcore.Plan {
	if p.memory == nil {
		return nil
	}
	
	planNodes, err := p.memory.Plans().List(ctx)
	if err != nil {
		return nil
	}
	
	result := make([]*goreactcore.Plan, 0, limit)
	for i := len(planNodes) - 1; i >= 0 && len(result) < limit; i-- {
		planNode := planNodes[i]
		// Convert PlanNode to Plan
		plan := &goreactcore.Plan{
			Name:            planNode.Name,
			SessionName:     planNode.SessionName,
			Goal:            planNode.Goal,
			Steps:           planNode.Steps,
			Status:          planNode.Status,
			CurrentStepIndex: 0,
		}
		result = append(result, plan)
	}
	
	return result
}

// getReflectionsFromMemory retrieves relevant reflections from memory
func (p *BasePlanner) getReflectionsFromMemory(ctx context.Context, goal string, limit int) []*goreactcore.Reflection {
	if p.memory == nil {
		return nil
	}
	
	reflectionNodes, err := p.memory.Reflections().List(ctx)
	if err != nil {
		return nil
	}
	
	result := make([]*goreactcore.Reflection, 0, limit)
	for i := len(reflectionNodes) - 1; i >= 0 && len(result) < limit; i-- {
		reflectionNode := reflectionNodes[i]
		// Convert ReflectionNode to Reflection
		reflection := &goreactcore.Reflection{
			Name:          reflectionNode.Name,
			TrajectoryName: reflectionNode.TrajectoryName,
			FailureReason: reflectionNode.FailureReason,
			Analysis:      reflectionNode.Analysis,
			Heuristic:     reflectionNode.Heuristic,
			Suggestions:   reflectionNode.Suggestions,
			Score:         reflectionNode.Score,
		}
		result = append(result, reflection)
	}
	
	return result
}
