// Package reactor provides the ReAct engine implementation for the goreact framework.
package reactor

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
)

// LLMClient is the LLM client interface from gochat
type LLMClient = core.Client

// Engine represents the ReAct engine interface
type Engine interface {
	// Execute executes the ReAct loop
	Execute(ctx context.Context, input string, opts ...Option) (*Result, error)
	
	// ExecuteStream executes the ReAct loop with streaming
	ExecuteStream(ctx context.Context, input string, opts ...Option) (<-chan any, error)
	
	// State returns the current state
	State() *goreactcore.State
	
	// Pause pauses the execution
	Pause() error
	
	// Resume resumes the execution
	Resume(ctx context.Context, state *goreactcore.State, answer string) (*Result, error)
	
	// ResumeStream resumes the execution with streaming
	ResumeStream(ctx context.Context, state *goreactcore.State, answer string) (<-chan any, error)
	
	// Stop stops the execution
	Stop() error
}

// Result represents the result of engine execution
type Result struct {
	// Answer is the final answer
	Answer string `json:"answer"`

	// Confidence is the confidence score of the answer (0.0 to 1.0)
	Confidence float64 `json:"confidence"`

	// Status is the execution status
	Status goreactcommon.Status `json:"status"`

	// SessionName is the session identifier
	SessionName string `json:"session_name"`

	// State is the final state
	State *goreactcore.State `json:"state"`

	// Trajectory is the execution trajectory
	Trajectory *goreactcore.Trajectory `json:"trajectory"`

	// Reflections is the list of reflections generated during execution
	Reflections []*goreactcore.Reflection `json:"reflections"`

	// TokenUsage is the token usage
	TokenUsage *goreactcommon.TokenUsage `json:"token_usage"`

	// Duration is the execution duration
	Duration time.Duration `json:"duration"`

	// Error is the error message
	Error string `json:"error,omitempty"`

	// PendingQuestion is the pending question if paused
	PendingQuestion *goreactcore.PendingQuestionNode `json:"pending_question,omitempty"`
}

// Option is a function that configures the engine
type Option func(*Options)

// Options contains engine options
type Options struct {
	SessionName string
	MaxSteps    int
	MaxRetries  int
	Timeout     time.Duration
	Context     map[string]any
}

// Reactor is the main ReAct engine implementation
type Reactor struct {
	// Components
	planner    Planner
	thinker    Thinker
	actor      Actor
	observer   Observer
	reflector  Reflector
	terminator Terminator

	// LLM Client
	llmClient LLMClient

	// Memory
	memory *memory.Memory

	// State
	state *goreactcore.State

	// Configuration
	config *Config

	// Control
	paused  bool
	stopped bool
}

// Config represents reactor configuration
type Config struct {
	MaxSteps         int
	MaxRetries       int
	Timeout          time.Duration
	EnablePlan       bool
	EnableReflection bool
	EnableEvolution  bool
	PauseOnToolAuth  bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSteps:         goreactcommon.DefaultMaxSteps,
		MaxRetries:       goreactcommon.DefaultMaxRetries,
		Timeout:          goreactcommon.DefaultTimeout,
		EnablePlan:       true,
		EnableReflection: true,
		EnableEvolution:  true,
		PauseOnToolAuth:  true,
	}
}

// NewReactor creates a new Reactor
func NewReactor(opts ...Option) *Reactor {
	config := DefaultConfig()
	r := &Reactor{
		config: config,
	}

	for _, opt := range opts {
		opt(&Options{})
	}

	return r
}

// WithPlanner sets the planner
func (r *Reactor) WithPlanner(planner Planner) *Reactor {
	r.planner = planner
	return r
}

// WithThinker sets the thinker
func (r *Reactor) WithThinker(thinker Thinker) *Reactor {
	r.thinker = thinker
	return r
}

// WithActor sets the actor
func (r *Reactor) WithActor(actor Actor) *Reactor {
	r.actor = actor
	return r
}

// WithObserver sets the observer
func (r *Reactor) WithObserver(observer Observer) *Reactor {
	r.observer = observer
	return r
}

// WithReflector sets the reflector
func (r *Reactor) WithReflector(reflector Reflector) *Reactor {
	r.reflector = reflector
	return r
}

// WithTerminator sets the terminator
func (r *Reactor) WithTerminator(terminator Terminator) *Reactor {
	r.terminator = terminator
	return r
}

// WithMemory sets the memory
func (r *Reactor) WithMemory(mem *memory.Memory) *Reactor {
	r.memory = mem
	// Also set memory to observer if it has SetMemory method
	if observer, ok := r.observer.(*BaseObserver); ok {
		observer.SetMemory(mem)
	}
	return r
}

// Execute executes the ReAct loop
func (r *Reactor) Execute(ctx context.Context, input string, opts ...Option) (*Result, error) {
	startTime := time.Now()

	// Apply options
	options := &Options{
		MaxSteps:   r.config.MaxSteps,
		MaxRetries: r.config.MaxRetries,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Initialize state
	r.state = goreactcore.NewState(
		options.SessionName,
		input,
		options.MaxSteps,
		options.MaxRetries,
	)
	r.state.Status = goreactcommon.StatusRunning

	// Create timeout context
	if r.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	// Intent classification phase
	intent, err := r.thinker.ClassifyIntent(ctx, input, r.state)
	if err != nil {
		// If intent classification fails, continue with Task intent
		intent = &goreactcore.IntentResult{
			Type:       string(goreactcommon.IntentTask),
			Confidence: 0.5,
			Reasoning:  "intent classification failed, defaulting to task",
		}
	}

	// Route based on intent type
	switch intent.Type {
	case string(goreactcommon.IntentChat):
		// Generate chat response directly
		thought := goreactcore.NewThought(input, intent.Reasoning, "answer", intent.Confidence)
		thought.WithFinalAnswer(intent.Reasoning)
		return &Result{
			Answer:      intent.Reasoning,
			Confidence:  intent.Confidence,
			Status:      goreactcommon.StatusCompleted,
			SessionName: r.state.SessionName,
			State:       r.state,
			TokenUsage:  r.state.TokenUsage,
			Duration:    time.Since(startTime),
		}, nil

	case string(goreactcommon.IntentClarification):
		// Process clarification response
		if intent.ExtractedAnswer != "" && r.memory != nil {
			// Update memory with clarification answer
			// Create a memory item for the clarification answer
			answerItem := goreactcore.NewMemoryItemNode(
				r.state.SessionName,
				intent.ExtractedAnswer,
				goreactcommon.MemoryItemTypeFact,
			)
			answerItem.Source = goreactcommon.MemorySourceUser
			answerItem.EmphasisLevel = goreactcommon.EmphasisLevelImportant

			if _, err := r.memory.ShortTerms().Add(ctx, r.state.SessionName, answerItem); err != nil {
				// Log error but continue
				r.state.Context["clarification_store_error"] = err.Error()
			}
		}

		// Continue with task execution if there's a pending question
		if r.state.PendingQuestion == nil {
			// No pending question, treat as new task
			intent.Type = string(goreactcommon.IntentTask)
		} else {
			// Resume from pause with clarification answer
			r.state.ClearPendingQuestion()
		}

	case string(goreactcommon.IntentFollowUp):
		// Load context from related session
		if intent.RelatedSession != "" && r.memory != nil {
			// Load session history from memory
			sessionHistory, err := r.memory.Sessions().GetHistory(ctx, intent.RelatedSession)
			if err == nil && sessionHistory != nil {
				// Merge context into current state
				if r.state.Context == nil {
					r.state.Context = make(map[string]any)
				}
				r.state.Context["related_session"] = intent.RelatedSession
				r.state.Context["related_history"] = sessionHistory
			}
		}
		// Continue with task execution
		intent.Type = string(goreactcommon.IntentTask)

	case string(goreactcommon.IntentFeedback):
		// Process feedback
		if r.memory != nil {
			// Store feedback in memory
			feedbackItem := goreactcore.NewMemoryItemNode(
				r.state.SessionName,
				input,
				goreactcommon.MemoryItemTypeCorrection,
			)
			feedbackItem.Source = goreactcommon.MemorySourceUser
			feedbackItem.EmphasisLevel = goreactcommon.EmphasisLevelImportant

			if _, err := r.memory.ShortTerms().Add(ctx, r.state.SessionName, feedbackItem); err != nil {
				// Log error but continue
				r.state.Context["feedback_store_error"] = err.Error()
			}
		}
		// Continue with task execution
		intent.Type = string(goreactcommon.IntentTask)

	case string(goreactcommon.IntentTask):
		// Continue to Plan phase
	}

	// Plan phase (only for Task intent)
	if intent.Type == string(goreactcommon.IntentTask) && r.config.EnablePlan && r.planner != nil {
		plan, err := r.planner.Plan(ctx, input, r.state)
		if err != nil {
			return nil, fmt.Errorf("planning failed: %w", err)
		}
		r.state.Plan = plan
	}

	// ReAct loop
	for !r.state.IsComplete() && !r.paused && !r.stopped {
		select {
		case <-ctx.Done():
			r.state.SetStatus(goreactcommon.StatusCanceled)
			return &Result{
				Status:      goreactcommon.StatusCanceled,
				SessionName: r.state.SessionName,
				State:       r.state,
				Trajectory:  r.state.Trajectory,
				Reflections: r.state.Reflections,
				TokenUsage:  r.state.TokenUsage,
				Duration:    time.Since(startTime),
				Error:       ctx.Err().Error(),
			}, ctx.Err()
		default:
		}

		// Think phase
		thought, err := r.thinker.Think(ctx, r.state)
		if err != nil {
			return nil, fmt.Errorf("thinking failed: %w", err)
		}
		r.state.AddThought(thought)
		r.state.IncrementStep()

		// Check for final answer
		if thought.IsAnswer() {
			r.state.SetStatus(goreactcommon.StatusCompleted)
			r.state.Trajectory.MarkSuccess(thought.FinalAnswer)
			return &Result{
				Answer:      thought.FinalAnswer,
				Confidence:  thought.Confidence,
				Status:      goreactcommon.StatusCompleted,
				SessionName: r.state.SessionName,
				State:       r.state,
				Trajectory:  r.state.Trajectory,
				Reflections: r.state.Reflections,
				TokenUsage:  r.state.TokenUsage,
				Duration:    time.Since(startTime),
			}, nil
		}

		// Act phase
		action := thought.ToAction()
		if action == nil {
			continue
		}

		actionResult, err := r.actor.Act(ctx, action, r.state)
		if err != nil {
			r.state.AddAction(action)
			continue
		}
		r.state.AddAction(action)

		// Observe phase
		observation, err := r.observer.Observe(ctx, actionResult, r.state)
		if err != nil {
			return nil, fmt.Errorf("observation failed: %w", err)
		}
		r.state.AddObservation(observation)

		// Update trajectory
		r.state.Trajectory.AddStep(thought, action, observation)

		// Check for failure and reflection
		if !observation.Success && r.config.EnableReflection && r.reflector != nil {
			if r.state.CanRetry() {
				reflection, err := r.reflector.Reflect(ctx, r.state)
				if err == nil {
					r.state.AddReflection(reflection)
					r.state.IncrementRetry()
				}
			}
		}
	}

	// Check termination conditions
	if r.paused {
		return &Result{
			Status:          goreactcommon.StatusPaused,
			SessionName:     r.state.SessionName,
			State:           r.state,
			Trajectory:      r.state.Trajectory,
			Reflections:     r.state.Reflections,
			TokenUsage:      r.state.TokenUsage,
			Duration:        time.Since(startTime),
			PendingQuestion: r.state.PendingQuestion,
		}, nil
	}

	if r.stopped {
		return &Result{
			Status:      goreactcommon.StatusCanceled,
			SessionName: r.state.SessionName,
			State:       r.state,
			Trajectory:  r.state.Trajectory,
			Reflections: r.state.Reflections,
			TokenUsage:  r.state.TokenUsage,
			Duration:    time.Since(startTime),
		}, nil
	}

	// Max steps reached without answer
	return &Result{
		Status:      goreactcommon.StatusFailed,
		SessionName: r.state.SessionName,
		State:       r.state,
		Trajectory:  r.state.Trajectory,
		Reflections: r.state.Reflections,
		TokenUsage:  r.state.TokenUsage,
		Duration:    time.Since(startTime),
		Error:       "max steps reached without answer",
	}, nil
}

// ExecuteStream executes with streaming
func (r *Reactor) ExecuteStream(ctx context.Context, input string, opts ...Option) (<-chan any, error) {
	ch := make(chan any, 100)

	go func() {
		defer close(ch)
		result, err := r.Execute(ctx, input, opts...)
		if err != nil {
			ch <- err
		} else {
			ch <- result
		}
	}()

	return ch, nil
}

// State returns the current state
func (r *Reactor) State() *goreactcore.State {
	return r.state
}

// Pause pauses the execution
func (r *Reactor) Pause() error {
	r.paused = true
	return nil
}

// Resume resumes the execution
func (r *Reactor) Resume(ctx context.Context, state *goreactcore.State, answer string) (*Result, error) {
	r.state = state
	r.state.ClearPendingQuestion()
	r.paused = false
	return r.Execute(ctx, state.Input)
}

// ResumeStream resumes the execution with streaming
func (r *Reactor) ResumeStream(ctx context.Context, state *goreactcore.State, answer string) (<-chan any, error) {
	ch := make(chan any, 100)
	
	go func() {
		defer close(ch)
		result, err := r.Resume(ctx, state, answer)
		if err != nil {
			ch <- err
		} else {
			ch <- result
		}
	}()
	
	return ch, nil
}

// Stop stops the execution
func (r *Reactor) Stop() error {
	r.stopped = true
	return nil
}

// Planner interface
type Planner interface {
	Plan(ctx context.Context, input string, state *goreactcore.State) (*goreactcore.Plan, error)
	Replan(ctx context.Context, state *goreactcore.State) (*goreactcore.Plan, error)
}

// Thinker interface
type Thinker interface {
	Think(ctx context.Context, state *goreactcore.State) (*goreactcore.Thought, error)
	ClassifyIntent(ctx context.Context, input string, state *goreactcore.State) (*goreactcore.IntentResult, error)
}

// Actor interface
type Actor interface {
	Act(ctx context.Context, action *goreactcore.Action, state *goreactcore.State) (*goreactcore.ActionResult, error)
	Validate(action *goreactcore.Action) error
}

// Observer interface
type Observer interface {
	Observe(ctx context.Context, result *goreactcore.ActionResult, state *goreactcore.State) (*goreactcore.Observation, error)
	Process(result any) (string, error)
	UpdateMemory(ctx context.Context, observation *goreactcore.Observation, state *goreactcore.State) error
}

// Reflector interface
type Reflector interface {
	Reflect(ctx context.Context, state *goreactcore.State) (*goreactcore.Reflection, error)
	GenerateHeuristic(reflection *goreactcore.Reflection) string
	StoreReflection(reflection *goreactcore.Reflection, state *goreactcore.State) error
	RetrieveRelevantReflections(ctx context.Context, query string, limit int) ([]*goreactcore.Reflection, error)
}

// Terminator interface
type Terminator interface {
	ShouldTerminate(state *goreactcore.State) bool
	Reason(state *goreactcore.State) string
}

// WithSessionName sets the session name
func WithSessionName(name string) Option {
	return func(o *Options) {
		o.SessionName = name
	}
}

// WithMaxSteps sets the max steps
func WithMaxSteps(maxSteps int) Option {
	return func(o *Options) {
		o.MaxSteps = maxSteps
	}
}

// WithMaxRetries sets the max retries
func WithMaxRetries(maxRetries int) Option {
	return func(o *Options) {
		o.MaxRetries = maxRetries
	}
}

// WithTimeout sets the timeout
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// WithContext sets the context
func WithContext(context map[string]any) Option {
	return func(o *Options) {
		o.Context = context
	}
}
