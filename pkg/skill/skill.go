// Package skill provides skill interfaces and implementations for the goreact framework.
package skill

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Skill represents a reusable capability unit
type Skill struct {
	// Name is the skill name (1-64 chars, lowercase letters, numbers, hyphens)
	Name string `json:"name" yaml:"name"`
	
	// Description is the skill description (1-1024 chars)
	// Describes what the skill does and when to use it
	Description string `json:"description" yaml:"description"`
	
	// Path is the skill directory path
	Path string `json:"path" yaml:"path"`
	
	// License is the skill license
	License string `json:"license" yaml:"license"`
	
	// Compatibility is the environment requirements
	Compatibility string `json:"compatibility" yaml:"compatibility"`
	
	// Agent is the agent that owns this skill
	Agent string `json:"agent" yaml:"agent"`
	
	// Intent is the intent pattern this skill handles
	Intent string `json:"intent" yaml:"intent"`
	
	// Template is the skill template (SKILL.md body content)
	Template string `json:"template" yaml:"template"`
	
	// Parameters are the skill parameters
	Parameters []Parameter `json:"parameters" yaml:"parameters"`
	
	// Steps are the execution steps
	Steps []ExecutionStep `json:"steps" yaml:"steps"`
	
	// AllowedTools are the tools this skill can use (space-separated in frontmatter)
	AllowedTools []string `json:"allowed_tools" yaml:"allowed_tools"`
	
	// Metadata contains additional metadata from frontmatter
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
	
	// ContentHash is the hash of SKILL.md content for cache invalidation
	ContentHash string `json:"content_hash" yaml:"content_hash"`
	
	// CreatedAt is the creation timestamp
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	
	// UpdatedAt is the last update timestamp
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// Parameter represents a skill parameter
type Parameter struct {
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
}

// ExecutionStep represents a step in skill execution
type ExecutionStep struct {
	// Index is the step index
	Index int `json:"index" yaml:"index"`
	
	// ToolName is the tool to use
	ToolName string `json:"tool_name" yaml:"tool_name"`
	
	// ParamsTemplate is the parameter template
	ParamsTemplate map[string]any `json:"params_template" yaml:"params_template"`
	
	// Condition is the condition for this step
	Condition string `json:"condition" yaml:"condition"`
	
	// ExpectedOutcome is the expected outcome
	ExpectedOutcome string `json:"expected_outcome" yaml:"expected_outcome"`
	
	// Description is the step description
	Description string `json:"description" yaml:"description"`
	
	// OnError is the error handling strategy (continue, stop, retry)
	OnError string `json:"on_error" yaml:"on_error"`
	
	// MaxRetries is the maximum number of retries when OnError is retry
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
	
	// RetryDelay is the delay between retries
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`
}

// NewSkill creates a new Skill
func NewSkill(name, description, agent string) *Skill {
	return &Skill{
		Name:         name,
		Description:  description,
		Agent:        agent,
		Parameters:   []Parameter{},
		Steps:        []ExecutionStep{},
		AllowedTools: []string{},
		Metadata:     make(map[string]any),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// WithIntent sets the intent
func (s *Skill) WithIntent(intent string) *Skill {
	s.Intent = intent
	return s
}

// WithTemplate sets the template
func (s *Skill) WithTemplate(template string) *Skill {
	s.Template = template
	return s
}

// WithParameter adds a parameter
func (s *Skill) WithParameter(param Parameter) *Skill {
	s.Parameters = append(s.Parameters, param)
	return s
}

// WithStep adds an execution step
func (s *Skill) WithStep(step ExecutionStep) *Skill {
	step.Index = len(s.Steps)
	s.Steps = append(s.Steps, step)
	return s
}

// WithAllowedTools sets the allowed tools
func (s *Skill) WithAllowedTools(tools []string) *Skill {
	s.AllowedTools = tools
	return s
}

// WithPath sets the skill path
func (s *Skill) WithPath(path string) *Skill {
	s.Path = path
	return s
}

// WithLicense sets the license
func (s *Skill) WithLicense(license string) *Skill {
	s.License = license
	return s
}

// WithCompatibility sets the compatibility
func (s *Skill) WithCompatibility(compat string) *Skill {
	s.Compatibility = compat
	return s
}

// WithContentHash sets the content hash
func (s *Skill) WithContentHash(hash string) *Skill {
	s.ContentHash = hash
	return s
}

// ComputeContentHash computes the content hash from template and steps
func (s *Skill) ComputeContentHash() string {
	// Simple hash computation - in production would use crypto/sha256
	hash := fmt.Sprintf("%s-%d-%d", s.Name, len(s.Steps), len(s.Parameters))
	return hash
}

// SkillExecutionPlan represents a compiled skill execution plan
type SkillExecutionPlan struct {
	// Name is the plan name
	Name string `json:"name" yaml:"name"`
	
	// SkillName is the skill name
	SkillName string `json:"skill_name" yaml:"skill_name"`
	
	// Steps are the execution steps
	Steps []ExecutionStep `json:"steps" yaml:"steps"`
	
	// Parameters are the parameter specifications
	Parameters []ParameterSpec `json:"parameters" yaml:"parameters"`
	
	// CompiledAt is the compilation timestamp
	CompiledAt time.Time `json:"compiled_at" yaml:"compiled_at"`
	
	// ExecutionCount is the number of times this plan has been executed
	ExecutionCount int `json:"execution_count" yaml:"execution_count"`
	
	// SuccessRate is the success rate
	SuccessRate float64 `json:"success_rate" yaml:"success_rate"`
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
}

// NewSkillExecutionPlan creates a new SkillExecutionPlan
func NewSkillExecutionPlan(skillName string) *SkillExecutionPlan {
	return &SkillExecutionPlan{
		Name:         "plan-" + skillName,
		SkillName:    skillName,
		Steps:        []ExecutionStep{},
		Parameters:   []ParameterSpec{},
		CompiledAt:   time.Now(),
		ExecutionCount: 0,
		SuccessRate:  0.0,
	}
}

// IncrementExecution increments the execution count
func (p *SkillExecutionPlan) IncrementExecution(success bool) {
	p.ExecutionCount++
	// Update success rate using exponential moving average
	if success {
		p.SuccessRate = p.SuccessRate*0.9 + 0.1
	} else {
		p.SuccessRate = p.SuccessRate * 0.9
	}
}

// GeneratedSkill represents a skill generated by evolution
type GeneratedSkill struct {
	// Name is the skill name
	Name string `json:"name" yaml:"name"`
	
	// Description is the skill description
	Description string `json:"description" yaml:"description"`
	
	// Content is the skill content (SKILL.md)
	Content string `json:"content" yaml:"content"`
	
	// FilePath is the file path where the skill is saved
	FilePath string `json:"file_path" yaml:"file_path"`
	
	// Parameters are the skill parameters
	Parameters []SkillParameter `json:"parameters" yaml:"parameters"`
	
	// Examples are usage examples
	Examples []string `json:"examples" yaml:"examples"`
	
	// CreatedAt is the creation timestamp
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	
	// SourceSession is the source session
	SourceSession string `json:"source_session" yaml:"source_session"`
	
	// Status is the generation status
	Status common.GeneratedStatus `json:"status" yaml:"status"`
}

// SkillParameter represents a parameter for generated skill
type SkillParameter struct {
	// Name is the parameter name
	Name string `json:"name" yaml:"name"`
	
	// Type is the parameter type
	Type string `json:"type" yaml:"type"`
	
	// Required indicates if the parameter is required
	Required bool `json:"required" yaml:"required"`
	
	// Default is the default value
	Default string `json:"default" yaml:"default"`
	
	// Description is the parameter description
	Description string `json:"description" yaml:"description"`
}

// SkillNode represents a Skill node in the memory graph
type SkillNode struct {
	Name         string            `json:"name" yaml:"name"`
	NodeType     string            `json:"node_type" yaml:"node_type"`
	Description  string            `json:"description" yaml:"description"`
	Agent        string            `json:"agent" yaml:"agent"`
	Intent       string            `json:"intent" yaml:"intent"`
	Template     string            `json:"template" yaml:"template"`
	Parameters   []Parameter       `json:"parameters" yaml:"parameters"`
	AllowedTools []string          `json:"allowed_tools" yaml:"allowed_tools"`
}

// SkillCompiler compiles skill templates into execution plans
type SkillCompiler interface {
	// Compile compiles a skill into an execution plan
	Compile(ctx context.Context, skill *Skill) (*SkillExecutionPlan, error)
	
	// ParseFrontmatter parses SKILL.md frontmatter
	ParseFrontmatter(content string) (*SkillFrontmatter, error)
	
	// ParseBody parses SKILL.md body into steps
	ParseBody(body string) ([]ExecutionStep, error)
}

// ListOption is a functional option for list operations
type ListOption func(*ListOptions)

// ListOptions contains options for list operations
type ListOptions struct {
	Agent   string
	Tags    []string
	Limit   int
	Offset  int
}

// WithAgent filters by agent
func WithAgent(agent string) ListOption {
	return func(opts *ListOptions) {
		opts.Agent = agent
	}
}

// WithTags filters by tags
func WithTags(tags []string) ListOption {
	return func(opts *ListOptions) {
		opts.Tags = tags
	}
}

// WithLimit sets the limit
func WithLimit(limit int) ListOption {
	return func(opts *ListOptions) {
		opts.Limit = limit
	}
}

// WithOffset sets the offset
func WithOffset(offset int) ListOption {
	return func(opts *ListOptions) {
		opts.Offset = offset
	}
}

// SkillAccessor provides access to skills through memory
type SkillAccessor interface {
	// Get retrieves a skill by name
	Get(ctx context.Context, name string) (*Skill, error)
	
	// List lists all skills with optional filters
	List(ctx context.Context, opts ...ListOption) ([]*Skill, error)
	
	// Search performs semantic search on skills
	Search(ctx context.Context, query string, topK int) ([]*Skill, error)
	
	// GetExecutionPlan retrieves a compiled execution plan
	GetExecutionPlan(ctx context.Context, skillName string) (*SkillExecutionPlan, error)
	
	// StoreExecutionPlan stores a compiled execution plan
	StoreExecutionPlan(ctx context.Context, plan *SkillExecutionPlan) error
	
	// DeleteExecutionPlan deletes an execution plan
	DeleteExecutionPlan(ctx context.Context, skillName string) error
	
	// UpdateExecutionStats updates execution statistics
	UpdateExecutionStats(ctx context.Context, skillName string, success bool, duration time.Duration) error
}

// SkillFrontmatter represents the frontmatter of a SKILL.md file
type SkillFrontmatter struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	License       string         `yaml:"license"`
	Compatibility string         `yaml:"compatibility"`
	AllowedTools  []string       `yaml:"allowed-tools"`
	Metadata      map[string]any `yaml:"metadata"`
}

// TemplateContext provides context for template rendering
type TemplateContext struct {
	// Session is the current session state
	Session *SessionState `json:"session"`
	
	// Steps are the previous step results
	Steps []*StepResult `json:"steps"`
	
	// Params are the call parameters
	Params map[string]any `json:"params"`
	
	// Runtime is the runtime context
	Runtime *RuntimeContext `json:"runtime"`
}

// SessionState represents the session state for templates
type SessionState struct {
	// Name is the session name
	Name string `json:"name"`
	
	// Input is the original input
	Input string `json:"input"`
	
	// CurrentStep is the current step number
	CurrentStep int `json:"current_step"`
	
	// Context is additional context
	Context map[string]any `json:"context"`
}

// StepResult represents the result of a step execution
type StepResult struct {
	// Index is the step index
	Index int `json:"index"`
	
	// ToolName is the tool that was executed
	ToolName string `json:"tool_name"`
	
	// Success indicates if the step succeeded
	Success bool `json:"success"`
	
	// Result is the step result
	Result any `json:"result"`
	
	// Error is the error message if failed
	Error string `json:"error"`
}

// RuntimeContext provides runtime context for templates
type RuntimeContext struct {
	// Timestamp is the current timestamp
	Timestamp time.Time `json:"timestamp"`
	
	// WorkingDir is the working directory
	WorkingDir string `json:"working_dir"`
	
	// EnvVars are environment variables
	EnvVars map[string]string `json:"env_vars"`
}

// SkillParser parses SKILL.md files
type SkillParser struct{}

// NewSkillParser creates a new SkillParser
func NewSkillParser() *SkillParser {
	return &SkillParser{}
}

// Parse parses a SKILL.md content into a Skill
func (p *SkillParser) Parse(content string, path string) (*Skill, error) {
	frontmatter, body, err := p.splitFrontmatter(content)
	if err != nil {
		return nil, err
	}
	
	skill := &Skill{
		Name:          frontmatter.Name,
		Description:   frontmatter.Description,
		Path:          path,
		License:       frontmatter.License,
		Compatibility: frontmatter.Compatibility,
		AllowedTools:  frontmatter.AllowedTools,
		Metadata:      frontmatter.Metadata,
		Template:      body,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	// Compute content hash
	skill.ContentHash = skill.ComputeContentHash()
	
	return skill, nil
}

// splitFrontmatter splits content into frontmatter and body
func (p *SkillParser) splitFrontmatter(content string) (*SkillFrontmatter, string, error) {
	// Simple implementation - would use proper YAML parser in production
	// Look for --- delimiters
	return &SkillFrontmatter{
		Name:        "unknown",
		Description: "",
	}, content, nil
}
