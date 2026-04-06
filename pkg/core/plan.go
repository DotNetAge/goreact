package core

import (
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Plan represents an execution plan
type Plan struct {
	// Name is the unique identifier
	Name string `json:"name" yaml:"name"`
	
	// SessionName is the session name
	SessionName string `json:"session_name" yaml:"session_name"`
	
	// Goal is the overall goal
	Goal string `json:"goal" yaml:"goal"`
	
	// Steps are the plan steps
	Steps []*PlanStep `json:"steps" yaml:"steps"`
	
	// CurrentStepIndex is the current step index
	CurrentStepIndex int `json:"current_step_index" yaml:"current_step_index"`
	
	// Status is the plan status
	Status common.PlanStatus `json:"status" yaml:"status"`
	
	// Success indicates if the plan succeeded
	Success bool `json:"success" yaml:"success"`
	
	// TaskType is the type of task
	TaskType string `json:"task_type" yaml:"task_type"`
	
	// CreatedAt is the creation timestamp
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	
	// UpdatedAt is the last update timestamp
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// PlanStep represents a step in a plan
type PlanStep struct {
	// Index is the step index
	Index int `json:"index" yaml:"index"`
	
	// Action is the action to perform
	Action string `json:"action" yaml:"action"`
	
	// Description is the step description
	Description string `json:"description" yaml:"description"`
	
	// Status is the step status
	Status common.StepStatus `json:"status" yaml:"status"`
	
	// Outcome is the step outcome
	Outcome string `json:"outcome" yaml:"outcome"`
	
	// ExpectedOutcome is the expected outcome
	ExpectedOutcome string `json:"expected_outcome" yaml:"expected_outcome"`
	
	// Tools are the tools to use
	Tools []string `json:"tools" yaml:"tools"`
	
	// Dependencies are the step dependencies
	Dependencies []int `json:"dependencies" yaml:"dependencies"`
}

// NewPlan creates a new Plan
func NewPlan(sessionName, goal string) *Plan {
	return &Plan{
		Name:             "plan-" + generateID(),
		SessionName:      sessionName,
		Goal:             goal,
		Steps:            []*PlanStep{},
		CurrentStepIndex: 0,
		Status:           common.PlanStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// AddStep adds a step to the plan
func (p *Plan) AddStep(action, description string) *PlanStep {
	step := &PlanStep{
		Index:       len(p.Steps),
		Action:      action,
		Description: description,
		Status:      common.StepStatusPending,
		Tools:       []string{},
		Dependencies: []int{},
	}
	p.Steps = append(p.Steps, step)
	p.UpdatedAt = time.Now()
	return step
}

// GetCurrentStep returns the current step
func (p *Plan) GetCurrentStep() *PlanStep {
	if p.CurrentStepIndex >= len(p.Steps) {
		return nil
	}
	return p.Steps[p.CurrentStepIndex]
}

// AdvanceStep advances to the next step
func (p *Plan) AdvanceStep() bool {
	if p.CurrentStepIndex >= len(p.Steps)-1 {
		return false
	}
	p.CurrentStepIndex++
	p.UpdatedAt = time.Now()
	return true
}

// CompleteStep marks the current step as completed
func (p *Plan) CompleteStep(outcome string) {
	step := p.GetCurrentStep()
	if step != nil {
		step.Status = common.StepStatusCompleted
		step.Outcome = outcome
		p.UpdatedAt = time.Now()
	}
}

// FailStep marks the current step as failed
func (p *Plan) FailStep(reason string) {
	step := p.GetCurrentStep()
	if step != nil {
		step.Status = common.StepStatusFailed
		step.Outcome = reason
		p.UpdatedAt = time.Now()
	}
}

// IsComplete checks if all steps are completed
func (p *Plan) IsComplete() bool {
	for _, step := range p.Steps {
		if step.Status != common.StepStatusCompleted && step.Status != common.StepStatusSkipped {
			return false
		}
	}
	return true
}

// MarkSuccess marks the plan as successful
func (p *Plan) MarkSuccess() {
	p.Status = common.PlanStatusCompleted
	p.Success = true
	p.UpdatedAt = time.Now()
}

// MarkFailed marks the plan as failed
func (p *Plan) MarkFailed() {
	p.Status = common.PlanStatusFailed
	p.Success = false
	p.UpdatedAt = time.Now()
}

// PlanNode represents a Plan node in the memory graph
type PlanNode struct {
	BaseNode
	SessionName string              `json:"session_name" yaml:"session_name"`
	Goal        string              `json:"goal" yaml:"goal"`
	Steps       []*PlanStep         `json:"steps" yaml:"steps"`
	Status      common.PlanStatus   `json:"status" yaml:"status"`
	Success     bool                `json:"success" yaml:"success"`
	TaskType    string              `json:"task_type" yaml:"task_type"`
}

// NewPlanNode creates a new PlanNode
func NewPlanNode(sessionName, goal string) *PlanNode {
	return &PlanNode{
		BaseNode: BaseNode{
			Name:        "plan-" + generateID(),
			NodeType:    common.NodeTypePlan,
			Description: goal,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		SessionName: sessionName,
		Goal:        goal,
		Steps:       []*PlanStep{},
		Status:      common.PlanStatusPending,
	}
}
