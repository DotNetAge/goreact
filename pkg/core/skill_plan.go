package core

import (
	"time"
)

// SkillExecutionPlan represents a compiled skill execution plan
type SkillExecutionPlan struct {
	// Name is the plan name
	Name string `json:"name" yaml:"name"`
	
	// SkillName is the source skill name
	SkillName string `json:"skill_name" yaml:"skill_name"`
	
	// Steps are the execution steps
	Steps []*ExecutionStep `json:"steps" yaml:"steps"`
	
	// Parameters are the parameter specifications
	Parameters []*ParameterSpec `json:"parameters" yaml:"parameters"`
	
	// CompiledAt is the compilation timestamp
	CompiledAt time.Time `json:"compiled_at" yaml:"compiled_at"`
	
	// ExecutionCount is the number of times executed
	ExecutionCount int `json:"execution_count" yaml:"execution_count"`
	
	// SuccessRate is the success rate (0.0-1.0)
	SuccessRate float64 `json:"success_rate" yaml:"success_rate"`
}

// ExecutionStep represents a step in execution
type ExecutionStep struct {
	// Index is the step index
	Index int `json:"index" yaml:"index"`
	
	// ToolName is the tool to execute
	ToolName string `json:"tool_name" yaml:"tool_name"`
	
	// ParamsTemplate is the parameter template
	ParamsTemplate map[string]any `json:"params_template" yaml:"params_template"`
	
	// Condition is the execution condition
	Condition string `json:"condition" yaml:"condition"`
	
	// ExpectedOutcome is the expected outcome
	ExpectedOutcome string `json:"expected_outcome" yaml:"expected_outcome"`
	
	// OnFailure is the failure handling strategy
	OnFailure string `json:"on_failure" yaml:"on_failure"`
	
	// Timeout is the step timeout
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// ParameterSpec represents a parameter specification
type ParameterSpec struct {
	// Name is the parameter name
	Name string `json:"name" yaml:"name"`
	
	// Type is the parameter type
	Type string `json:"type" yaml:"type"`
	
	// Required indicates if the parameter is required
	Required bool `json:"required" yaml:"required"`
	
	// Default is the default value
	Default any `json:"default" yaml:"default"`
	
	// Description is the parameter description
	Description string `json:"description" yaml:"description"`
	
	// Validation is the validation pattern
	Validation string `json:"validation" yaml:"validation"`
}

// NewSkillExecutionPlan creates a new SkillExecutionPlan
func NewSkillExecutionPlan(skillName string) *SkillExecutionPlan {
	return &SkillExecutionPlan{
		Name:        "skill-plan-" + generateID(),
		SkillName:   skillName,
		Steps:       []*ExecutionStep{},
		Parameters:  []*ParameterSpec{},
		CompiledAt:  time.Now(),
		SuccessRate: 1.0,
	}
}

// AddStep adds an execution step
func (p *SkillExecutionPlan) AddStep(toolName string, params map[string]any) *ExecutionStep {
	step := &ExecutionStep{
		Index:         len(p.Steps),
		ToolName:      toolName,
		ParamsTemplate: params,
	}
	p.Steps = append(p.Steps, step)
	return step
}

// AddParameter adds a parameter specification
func (p *SkillExecutionPlan) AddParameter(name, typ string, required bool) *ParameterSpec {
	param := &ParameterSpec{
		Name:     name,
		Type:     typ,
		Required: required,
	}
	p.Parameters = append(p.Parameters, param)
	return param
}

// RecordExecution records an execution result
func (p *SkillExecutionPlan) RecordExecution(success bool) {
	p.ExecutionCount++
	
	// Update success rate with exponential moving average
	alpha := 0.1
	if success {
		p.SuccessRate = alpha*1.0 + (1-alpha)*p.SuccessRate
	} else {
		p.SuccessRate = alpha*0.0 + (1-alpha)*p.SuccessRate
	}
}

// GetStep returns a step by index
func (p *SkillExecutionPlan) GetStep(index int) *ExecutionStep {
	if index < 0 || index >= len(p.Steps) {
		return nil
	}
	return p.Steps[index]
}

// WithCondition sets the step condition
func (s *ExecutionStep) WithCondition(condition string) *ExecutionStep {
	s.Condition = condition
	return s
}

// WithExpectedOutcome sets the expected outcome
func (s *ExecutionStep) WithExpectedOutcome(outcome string) *ExecutionStep {
	s.ExpectedOutcome = outcome
	return s
}

// WithOnFailure sets the failure handling strategy
func (s *ExecutionStep) WithOnFailure(strategy string) *ExecutionStep {
	s.OnFailure = strategy
	return s
}

// WithTimeout sets the step timeout
func (s *ExecutionStep) WithTimeout(timeout time.Duration) *ExecutionStep {
	s.Timeout = timeout
	return s
}

// WithDefault sets the parameter default value
func (p *ParameterSpec) WithDefault(value any) *ParameterSpec {
	p.Default = value
	return p
}

// WithDescription sets the parameter description
func (p *ParameterSpec) WithDescription(desc string) *ParameterSpec {
	p.Description = desc
	return p
}

// WithValidation sets the validation pattern
func (p *ParameterSpec) WithValidation(pattern string) *ParameterSpec {
	p.Validation = pattern
	return p
}
