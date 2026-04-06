// Package agent provides agent interfaces and implementations for the goreact framework.
package agent

import (
	"context"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Agent represents an intelligent agent
type Agent interface {
	// Name returns the agent name
	Name() string
	
	// Domain returns the agent domain
	Domain() string
	
	// Description returns the agent description
	Description() string
	
	// Model returns the model name
	Model() string
	
	// Skills returns the skill names
	Skills() []string
	
	// PromptTemplate returns the prompt template
	PromptTemplate() string
	
	// Config returns the agent configuration
	Config() *Config
	
	// Ask executes a question
	Ask(ctx context.Context, question string, files ...string) (*Result, error)
	
	// Resume resumes a paused session
	Resume(ctx context.Context, sessionName string, answer string) (*Result, error)
	
	// AskStream executes a question with streaming response
	AskStream(ctx context.Context, question string, files ...string) (<-chan any, error)
}

// Config represents agent configuration
type Config struct {
	// Name is the agent name
	Name string `json:"name" yaml:"name"`
	
	// Domain is the agent domain
	Domain string `json:"domain" yaml:"domain"`
	
	// Description is the agent description
	Description string `json:"description" yaml:"description"`
	
	// Model is the model name
	Model string `json:"model" yaml:"model"`
	
	// PromptTemplate is the prompt template
	PromptTemplate string `json:"prompt_template" yaml:"prompt_template"`
	
	// MaxSteps is the maximum number of steps
	MaxSteps int `json:"max_steps" yaml:"max_steps"`
	
	// MaxRetries is the maximum number of retries
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
	
	// Timeout is the execution timeout
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	
	// Skills are the skill names
	Skills []string `json:"skills" yaml:"skills"`
	
	// Tools are the tool names
	Tools []string `json:"tools" yaml:"tools"`
	
	// Metadata contains additional metadata
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
}

// DefaultConfig returns the default agent configuration
func DefaultConfig() *Config {
	return &Config{
		Name:           common.DefaultAgentName,
		Domain:         common.DefaultDomain,
		Model:          common.DefaultModel,
		MaxSteps:       common.DefaultMaxSteps,
		MaxRetries:     common.DefaultMaxRetries,
		Timeout:        common.DefaultTimeout,
		Skills:         []string{},
		Tools:          []string{},
		Metadata:       make(map[string]any),
	}
}

// Result represents the result of an agent execution
type Result struct {
	// Answer is the final answer
	Answer string `json:"answer"`
	
	// Status is the execution status
	Status common.Status `json:"status"`
	
	// SessionName is the session name
	SessionName string `json:"session_name"`
	
	// Trajectory is the execution trajectory
	Trajectory any `json:"trajectory"`
	
	// Reflections are the reflections
	Reflections []any `json:"reflections"`
	
	// TokenUsage is the token usage
	TokenUsage *common.TokenUsage `json:"token_usage"`
	
	// Duration is the execution duration
	Duration time.Duration `json:"duration"`
	
	// Metadata contains additional metadata
	Metadata map[string]any `json:"metadata"`
	
	// Error is the error message
	Error string `json:"error,omitempty"`
	
	// PendingQuestion is the pending question if paused
	PendingQuestion *PendingQuestion `json:"pending_question,omitempty"`
}

// PendingQuestion represents a question that needs user response
type PendingQuestion struct {
	// ID is the question ID
	ID string `json:"id"`
	
	// Type is the question type
	Type common.QuestionType `json:"type"`
	
	// Question is the question text
	Question string `json:"question"`
	
	// Options are the available options
	Options []string `json:"options"`
	
	// DefaultAnswer is the default answer
	DefaultAnswer string `json:"default_answer"`
}

// BaseAgent provides a base implementation for agents
type BaseAgent struct {
	config *Config
}

// NewBaseAgent creates a new BaseAgent
func NewBaseAgent(config *Config) *BaseAgent {
	if config == nil {
		config = DefaultConfig()
	}
	return &BaseAgent{config: config}
}

// Name returns the agent name
func (a *BaseAgent) Name() string {
	return a.config.Name
}

// Domain returns the agent domain
func (a *BaseAgent) Domain() string {
	return a.config.Domain
}

// Description returns the agent description
func (a *BaseAgent) Description() string {
	return a.config.Description
}

// Model returns the model name
func (a *BaseAgent) Model() string {
	return a.config.Model
}

// Skills returns the skill names
func (a *BaseAgent) Skills() []string {
	return a.config.Skills
}

// PromptTemplate returns the prompt template
func (a *BaseAgent) PromptTemplate() string {
	return a.config.PromptTemplate
}

// Config returns the agent configuration
func (a *BaseAgent) Config() *Config {
	return a.config
}

// Input represents the input for an agent
type Input struct {
	// Question is the user question
	Question string `json:"question"`
	
	// Files are the input files
	Files []string `json:"files"`
	
	// Context is additional context
	Context map[string]any `json:"context"`
}

// NewInput creates a new Input
func NewInput(question string, files ...string) *Input {
	return &Input{
		Question: question,
		Files:    files,
		Context:  make(map[string]any),
	}
}

// WithContext adds context
func (i *Input) WithContext(key string, value any) *Input {
	if i.Context == nil {
		i.Context = make(map[string]any)
	}
	i.Context[key] = value
	return i
}
