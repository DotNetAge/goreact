package reactor

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
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
	llmClient     any // Would be LLM client
	memory        any // Would be Memory
	config        *PlannerConfig
	planHistory   []*core.Plan
}

// NewBasePlanner creates a new BasePlanner
func NewBasePlanner(config *PlannerConfig) *BasePlanner {
	if config == nil {
		config = DefaultPlannerConfig()
	}
	return &BasePlanner{
		config:      config,
		planHistory: []*core.Plan{},
	}
}

// WithLLMClient sets the LLM client
func (p *BasePlanner) WithLLMClient(client any) *BasePlanner {
	p.llmClient = client
	return p
}

// WithMemory sets the memory
func (p *BasePlanner) WithMemory(memory any) *BasePlanner {
	p.memory = memory
	return p
}

// Plan creates an execution plan
func (p *BasePlanner) Plan(ctx context.Context, input string, state *core.State) (*core.Plan, error) {
	// Create new plan
	plan := core.NewPlan(state.SessionName, input)
	
	// Would use LLM to generate plan steps
	// Simplified implementation - add basic steps
	
	plan.AddStep("analyze", "Analyze the input and understand the task")
	plan.AddStep("plan", "Create a detailed execution plan")
	plan.AddStep("execute", "Execute the planned actions")
	plan.AddStep("verify", "Verify the results")
	plan.AddStep("respond", "Generate the final response")
	
	// Limit steps
	if len(plan.Steps) > p.config.MaxPlanSteps {
		plan.Steps = plan.Steps[:p.config.MaxPlanSteps]
	}
	
	plan.Status = common.PlanStatusRunning
	
	// Store in history
	p.planHistory = append(p.planHistory, plan)
	
	return plan, nil
}

// Replan creates a new plan based on current state
func (p *BasePlanner) Replan(ctx context.Context, state *core.State) (*core.Plan, error) {
	if !p.config.EnableReplan {
		return state.Plan, nil
	}
	
	if state.Plan == nil {
		return p.Plan(ctx, state.Input, state)
	}
	
	// Would use LLM to revise the plan
	// Simplified implementation - mark current plan as revised
	
	plan := state.Plan
	plan.Status = common.PlanStatusRevised
	
	// Add recovery step
	plan.AddStep("recover", "Recover from the failure and adjust approach")
	
	// Store in history
	p.planHistory = append(p.planHistory, plan)
	
	return plan, nil
}

// Validate validates a plan
func (p *BasePlanner) Validate(plan *core.Plan) error {
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
func (p *BasePlanner) validateStepDependencies(stepIndex int, plan *core.Plan, visited map[int]bool) error {
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
func (p *BasePlanner) NeedReplan(state *core.State) bool {
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
func (p *BasePlanner) GetPlanHistory() []*core.Plan {
	return p.planHistory
}

// GetSimilarPlans retrieves similar plans from history
func (p *BasePlanner) GetSimilarPlans(goal string, limit int) []*core.Plan {
	// Would use semantic similarity to find similar plans
	// Simplified implementation - return recent plans
	if limit <= 0 || limit > len(p.planHistory) {
		limit = len(p.planHistory)
	}
	
	start := len(p.planHistory) - limit
	if start < 0 {
		start = 0
	}
	
	return p.planHistory[start:]
}
