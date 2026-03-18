package core

import (
	"context"
	"time"

	chatcore "github.com/DotNetAge/gochat/pkg/core"
)

// PipelineContext carries the state, memory, and observability hooks for a single ReAct execution cycle.
type PipelineContext struct {
	context.Context

	SessionID   string
	Input       string
	CurrentStep int
	IsFinished  bool
	FinalResult string
	Traces      []*Trace
	Attachments []chatcore.Attachment // User or Tool supplied multimedia/files

	// Configuration & Constraints
	MaxSteps int

	// Observability & Telemetry
	Logger      Logger
	Metrics     Metrics
	TotalTokens *TokenUsage
	StartTime   time.Time
	Error       error

	FinishReason    string
	OnThoughtStream func(string) // Optional hook for UI streaming updates

	// State storage for intermediate data (e.g., raw_output)
	state map[string]any
}

// ContextOption is a functional option for configuring a PipelineContext.
type ContextOption func(*PipelineContext)

// WithAttachments injects user-uploaded files or multimedia into the context
func WithAttachments(attachments ...chatcore.Attachment) ContextOption {
	return func(ctx *PipelineContext) {
		ctx.Attachments = append(ctx.Attachments, attachments...)
	}
}

// WithLogger injects a logger into the context.
func WithLogger(l Logger) ContextOption {
	return func(c *PipelineContext) {
		c.Logger = l
	}
}

// WithMetrics injects a metrics collector.
func WithMetrics(m Metrics) ContextOption {
	return func(c *PipelineContext) {
		c.Metrics = m
	}
}

// WithMaxSteps sets the maximum number of reasoning steps to prevent infinite loops.
func WithMaxSteps(max int) ContextOption {
	return func(c *PipelineContext) {
		c.MaxSteps = max
	}
}

// WithThoughtStream registers a hook for streaming the LLM's thought process in real-time.
func WithThoughtStream(hook func(string)) ContextOption {
	return func(c *PipelineContext) {
		c.OnThoughtStream = hook
	}
}

// NewPipelineContext initializes a fresh context for a new task.
func NewPipelineContext(ctx context.Context, sessionID, input string, opts ...ContextOption) *PipelineContext {
	pctx := &PipelineContext{
		Context:     ctx,
		SessionID:   sessionID,
		Input:       input,
		CurrentStep: 1,
		IsFinished:  false,
		MaxSteps:    10, // Default constraint
		Logger:      DefaultLogger(),
		Metrics:     DefaultMetrics(),
		TotalTokens: &TokenUsage{},
		StartTime:   time.Now(),
		Traces:      make([]*Trace, 0),
		Attachments: make([]chatcore.Attachment, 0),
		state:       make(map[string]any),
	}

	for _, opt := range opts {
		opt(pctx)
	}

	return pctx
}

// AppendTrace adds a new thought-action-observation cycle to the context's memory.
func (c *PipelineContext) AppendTrace(t *Trace) {
	c.Traces = append(c.Traces, t)
	c.CurrentStep++
}

// ToLLMMessages (deprecated) - Serialization is now the responsibility of Thinker implementations.
func (c *PipelineContext) ToLLMMessages() []chatcore.Message {
	return nil
}

// LastTrace returns the most recently appended Trace or nil if empty.
func (c *PipelineContext) LastTrace() *Trace {
	if len(c.Traces) == 0 {
		return nil
	}
	return c.Traces[len(c.Traces)-1]
}

// Get is a helper to fetch scoped execution variables.
func (c *PipelineContext) Get(key string) (any, bool) {
	if c.state == nil {
		return nil, false
	}
	val, ok := c.state[key]
	return val, ok
}

// Set is a helper to store scoped execution variables.
func (c *PipelineContext) Set(key string, value any) {
	if c.state == nil {
		c.state = make(map[string]any)
	}
	if value == nil {
		delete(c.state, key)
	} else {
		c.state[key] = value
	}
}
