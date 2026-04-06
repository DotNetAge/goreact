package core

import (
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Action represents an action to be executed
type Action struct {
	// Type is the action type
	Type common.ActionType `json:"type" yaml:"type"`
	
	// Target is the target name (tool/skill/agent name)
	Target string `json:"target" yaml:"target"`
	
	// Params are the action parameters
	Params map[string]any `json:"params" yaml:"params"`
	
	// Reasoning is why this action was chosen
	Reasoning string `json:"reasoning" yaml:"reasoning"`
	
	// Timestamp is the action timestamp
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// ActionResult represents the result of an action execution
type ActionResult struct {
	// Success indicates if the execution was successful
	Success bool `json:"success" yaml:"success"`
	
	// Result is the execution result
	Result any `json:"result" yaml:"result"`
	
	// Error is the error message if execution failed
	Error string `json:"error" yaml:"error"`
	
	// Duration is the execution duration
	Duration time.Duration `json:"duration" yaml:"duration"`
	
	// Metadata contains additional metadata
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
	
	// ToolName is the executed tool name (if tool call)
	ToolName string `json:"tool_name" yaml:"tool_name"`
	
	// SkillName is the executed skill name (if skill invoke)
	SkillName string `json:"skill_name" yaml:"skill_name"`
	
	// SubAgentName is the delegated agent name (if delegation)
	SubAgentName string `json:"sub_agent_name" yaml:"sub_agent_name"`
}

// NewAction creates a new Action
func NewAction(actionType common.ActionType, target string, params map[string]any) *Action {
	return &Action{
		Type:      actionType,
		Target:    target,
		Params:    params,
		Timestamp: time.Now(),
	}
}

// WithReasoning sets the reasoning
func (a *Action) WithReasoning(reasoning string) *Action {
	a.Reasoning = reasoning
	return a
}

// IsToolCall checks if the action is a tool call
func (a *Action) IsToolCall() bool {
	return a.Type == common.ActionTypeToolCall
}

// IsSkillInvoke checks if the action is a skill invocation
func (a *Action) IsSkillInvoke() bool {
	return a.Type == common.ActionTypeSkillInvoke
}

// IsDelegation checks if the action is a sub-agent delegation
func (a *Action) IsDelegation() bool {
	return a.Type == common.ActionTypeSubAgentDelegate
}

// IsNoAction checks if no action is needed
func (a *Action) IsNoAction() bool {
	return a.Type == common.ActionTypeNoAction
}

// NewActionResult creates a new ActionResult
func NewActionResult(success bool, result any) *ActionResult {
	return &ActionResult{
		Success:  success,
		Result:   result,
		Metadata: make(map[string]any),
	}
}

// WithError sets the error
func (r *ActionResult) WithError(err string) *ActionResult {
	r.Error = err
	r.Success = false
	return r
}

// WithDuration sets the duration
func (r *ActionResult) WithDuration(d time.Duration) *ActionResult {
	r.Duration = d
	return r
}

// WithMetadata adds metadata
func (r *ActionResult) WithMetadata(key string, value any) *ActionResult {
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}
	r.Metadata[key] = value
	return r
}

// WithTool sets the tool name
func (r *ActionResult) WithTool(name string) *ActionResult {
	r.ToolName = name
	return r
}

// WithSkill sets the skill name
func (r *ActionResult) WithSkill(name string) *ActionResult {
	r.SkillName = name
	return r
}

// WithSubAgent sets the sub-agent name
func (r *ActionResult) WithSubAgent(name string) *ActionResult {
	r.SubAgentName = name
	return r
}

// ParamRule represents a validation rule for action parameters
type ParamRule struct {
	// Name is the parameter name
	Name string `json:"name" yaml:"name"`
	
	// Type is the parameter type
	Type string `json:"type" yaml:"type"`
	
	// Required indicates if the parameter is required
	Required bool `json:"required" yaml:"required"`
	
	// Min is the minimum value/length
	Min int `json:"min" yaml:"min"`
	
	// Max is the maximum value/length
	Max int `json:"max" yaml:"max"`
	
	// Pattern is a regex pattern for string validation
	Pattern string `json:"pattern" yaml:"pattern"`
	
	// Enum is the list of allowed values
	Enum []string `json:"enum" yaml:"enum"`
	
	// Description is the parameter description
	Description string `json:"description" yaml:"description"`
}

// Insight represents an insight extracted from an observation
type Insight struct {
	// Type is the insight type
	Type common.InsightType `json:"type" yaml:"type"`
	
	// Content is the insight content
	Content string `json:"content" yaml:"content"`
	
	// Confidence is the confidence level
	Confidence float64 `json:"confidence" yaml:"confidence"`
	
	// Source is the source of the insight
	Source string `json:"source" yaml:"source"`
}

// InsightRule represents a rule for insight extraction
type InsightRule struct {
	// Name is the rule name
	Name string `json:"name" yaml:"name"`
	
	// Type is the insight type
	Type common.InsightType `json:"type" yaml:"type"`
	
	// Pattern is the pattern to match
	Pattern string `json:"pattern" yaml:"pattern"`
	
	// Priority is the rule priority
	Priority int `json:"priority" yaml:"priority"`
	
	// Enabled indicates if the rule is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
}
