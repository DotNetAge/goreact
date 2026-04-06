package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
	"github.com/DotNetAge/goreact/pkg/reactor"
	"github.com/DotNetAge/goreact/pkg/skill"
	"github.com/DotNetAge/goreact/pkg/tool"
)

// Executor implements the Agent interface with full functionality
type Executor struct {
	*BaseAgent
	llmClient  core.Client
	reactor    reactor.Engine
	memory     *memory.Memory
	skillReg   *skill.Registry
	toolReg    *tool.Registry
}

// NewExecutor creates a new Agent executor
func NewExecutor(config *Config, llm core.Client, opts ...Option) *Executor {
	if config == nil {
		config = DefaultConfig()
	}
	
	e := &Executor{
		BaseAgent:  NewBaseAgent(config),
		llmClient:  llm,
		memory:     nil, // Memory should be set via WithMemory option
		skillReg:   skill.NewRegistry(),
		toolReg:    tool.NewRegistry(),
	}
	
	for _, opt := range opts {
		opt(e)
	}
	
	return e
}

// Option is a function that configures the executor
type Option func(*Executor)

// WithMemory sets the memory
func WithMemory(mem *memory.Memory) Option {
	return func(e *Executor) {
		e.memory = mem
	}
}

// WithReactor sets the reactor
func WithReactor(r reactor.Engine) Option {
	return func(e *Executor) {
		e.reactor = r
	}
}

// WithSkillRegistry sets the skill registry
func WithSkillRegistry(r *skill.Registry) Option {
	return func(e *Executor) {
		e.skillReg = r
	}
}

// WithToolRegistry sets the tool registry
func WithToolRegistry(r *tool.Registry) Option {
	return func(e *Executor) {
		e.toolReg = r
	}
}

// Ask executes a question
func (e *Executor) Ask(ctx context.Context, question string, files ...string) (*Result, error) {
	startTime := time.Now()
	
	// Create or get session
	sessionName := e.generateSessionName()
	
	// Initialize reactor if not set
	if e.reactor == nil {
		e.reactor = reactor.NewReactor(
			reactor.WithSessionName(sessionName),
			reactor.WithMaxSteps(e.config.MaxSteps),
			reactor.WithMaxRetries(e.config.MaxRetries),
		)
	}
	
	// Build input with files
	input := question
	if len(files) > 0 {
		input = fmt.Sprintf("%s\n\nAttached files: %v", question, files)
	}
	
	// Execute
	reactorResult, err := e.reactor.Execute(ctx, input,
		reactor.WithSessionName(sessionName),
		reactor.WithMaxSteps(e.config.MaxSteps),
	)
	if err != nil {
		return &Result{
			Status:   goreactcommon.StatusFailed,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, err
	}
	
	// Convert result
	result := &Result{
		Answer:      reactorResult.Answer,
		Status:      reactorResult.Status,
		SessionName: sessionName,
		Trajectory:  reactorResult.State.Trajectory,
		TokenUsage:  reactorResult.TokenUsage,
		Duration:    time.Since(startTime),
	}
	
	if reactorResult.PendingQuestion != nil {
		result.PendingQuestion = &PendingQuestion{
			ID:            reactorResult.PendingQuestion.Name,
			Type:          goreactcommon.QuestionType(reactorResult.PendingQuestion.Type),
			Question:      reactorResult.PendingQuestion.Question,
			Options:       reactorResult.PendingQuestion.Options,
			DefaultAnswer: reactorResult.PendingQuestion.DefaultAnswer,
		}
	}
	
	return result, nil
}

// Resume resumes a paused session
func (e *Executor) Resume(ctx context.Context, sessionName string, answer string) (*Result, error) {
	startTime := time.Now()
	
	// Get frozen session from memory
	_, err := e.memory.FrozenSessions().Get(ctx, sessionName)
	if err != nil {
		return &Result{
			Status:   goreactcommon.StatusFailed,
			Error:    fmt.Sprintf("session not found: %s", sessionName),
			Duration: time.Since(startTime),
		}, err
	}
	
	// Deserialize state from frozen session
	// In a real implementation, would deserialize StateData
	var frozenState *goreactcore.State
	// frozenState = deserializeState(frozenSession.StateData)
	
	// Resume with answer
	reactorResult, err := e.reactor.Resume(ctx, frozenState, answer)
	if err != nil {
		return &Result{
			Status:   goreactcommon.StatusFailed,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, err
	}
	
	// Convert result
	result := &Result{
		Answer:      reactorResult.Answer,
		Status:      reactorResult.Status,
		SessionName: sessionName,
		TokenUsage:  reactorResult.TokenUsage,
		Duration:    time.Since(startTime),
	}
	
	if reactorResult.State != nil {
		result.Trajectory = reactorResult.State.Trajectory
	}
	
	return result, nil
}

// AskStream executes a question with streaming response
func (e *Executor) AskStream(ctx context.Context, question string, files ...string) (<-chan any, error) {
	// Create channel for streaming results
	ch := make(chan any, 100)
	
	go func() {
		defer close(ch)
		
		result, err := e.Ask(ctx, question, files...)
		if err != nil {
			ch <- err
			return
		}
		ch <- result
	}()
	
	return ch, nil
}

// ResumeStream resumes a paused session with streaming response
func (e *Executor) ResumeStream(ctx context.Context, sessionName string, answer string) (<-chan any, error) {
	ch := make(chan any, 100)
	
	go func() {
		defer close(ch)
		
		result, err := e.Resume(ctx, sessionName, answer)
		if err != nil {
			ch <- err
			return
		}
		ch <- result
	}()
	
	return ch, nil
}

// generateSessionName generates a unique session name
func (e *Executor) generateSessionName() string {
	return fmt.Sprintf("%s-%d", e.config.Name, time.Now().UnixNano())
}

// GetMemory returns the memory instance
func (e *Executor) GetMemory() *memory.Memory {
	return e.memory
}

// GetSkillRegistry returns the skill registry
func (e *Executor) GetSkillRegistry() *skill.Registry {
	return e.skillReg
}

// GetToolRegistry returns the tool registry
func (e *Executor) GetToolRegistry() *tool.Registry {
	return e.toolReg
}

// RegisterSkill registers a skill
func (e *Executor) RegisterSkill(s *skill.Skill) error {
	return e.skillReg.Register(s)
}

// RegisterTool registers a tool
func (e *Executor) RegisterTool(t tool.Tool) error {
	return e.toolReg.Register(t)
}
