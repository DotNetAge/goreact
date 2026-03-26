package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/DotNetAge/goreact/pkg/actor"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/observer"
	"github.com/DotNetAge/goreact/pkg/steps"
	"github.com/DotNetAge/goreact/pkg/terminator"
	"github.com/DotNetAge/goreact/pkg/thinker"
)

// Reactor represents the core ReAct loop engine.
// It assembles the four fundamental cognitive components into a robust pipeline.
type Reactor struct {
	logger  core.Logger
	metrics core.Metrics

	thinker    thinker.Thinker
	actor      actor.Actor
	observer   observer.Observer
	terminator terminator.Terminator
	hooks      []pipeline.Hook[*core.PipelineContext]
}

// Option is a functional option for configuring a Reactor.
type Option func(*Reactor)

// WithEngineLogger injects a global logger into the reactor pipeline.
func WithEngineLogger(l core.Logger) Option {
	return func(r *Reactor) {
		r.logger = l
	}
}

// WithEngineMetrics injects a global metrics recorder into the reactor.
func WithEngineMetrics(m core.Metrics) Option {
	return func(r *Reactor) {
		r.metrics = m
	}
}

// WithPipelineHook injects a custom hook into the pipeline execution.
func WithPipelineHook(h pipeline.Hook[*core.PipelineContext]) Option {
	return func(r *Reactor) {
		r.hooks = append(r.hooks, h)
	}
}

// WithThinker injects a custom Thinker.
func WithThinker(t thinker.Thinker) Option {
	return func(r *Reactor) {
		r.thinker = t
	}
}

// WithActor injects a custom Actor.
func WithActor(a actor.Actor) Option {
	return func(r *Reactor) {
		r.actor = a
	}
}

// WithObserver injects a custom Observer.
func WithObserver(o observer.Observer) Option {
	return func(r *Reactor) {
		r.observer = o
	}
}

// WithTerminator injects a custom Terminator.
func WithTerminator(t terminator.Terminator) Option {
	return func(r *Reactor) {
		r.terminator = t
	}
}

// NewReactor constructs a new ReAct loop engine.
func NewReactor(opts ...Option) *Reactor {
	r := &Reactor{
		logger:  core.DefaultLogger(),
		metrics: core.DefaultMetrics(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes the full Think-Act-Observe-Check loop until the Terminator halts it.
func (r *Reactor) Run(ctx context.Context, sessionID, input string, customOpts ...core.ContextOption) (*core.PipelineContext, error) {
	// Inject default engine loggers and metrics into the context options.
	opts := []core.ContextOption{
		core.WithLogger(r.logger),
		core.WithMetrics(r.metrics),
	}
	opts = append(opts, customOpts...)

	// 1. Build the shared PipelineContext.
	reactCtx := core.NewPipelineContext(ctx, sessionID, input, opts...)

	reactCtx.Logger.Info("Agent Session Started", "session", sessionID, "input", input)
	start := time.Now()
	defer func() {
		reactCtx.Logger.Info("Agent Session Ended", "session", sessionID, "duration", time.Since(start), "finished", reactCtx.IsFinished)
	}()

	// 2. Validate dependencies.
	if r.thinker == nil || r.actor == nil || r.observer == nil || r.terminator == nil {
		return nil, fmt.Errorf("reactor initialization failed: all 4 components (Thinker, Actor, Observer, Terminator) must be configured")
	}

	// 3. Assemble the core ReAct cognitive pipeline
	p := pipeline.New[*core.PipelineContext]()

	p.AddSteps(
		steps.Thinker(r.thinker),
		steps.Actor(r.actor),
		steps.Observer(r.observer),
		steps.CheckFinish(r.terminator),
	)

	// Register all configured hooks
	for _, h := range r.hooks {
		p.AddHook(h)
	}

	// 4. The main loop
	for {
		// reactCtx.CurrentStep++

		// Execute one full cycle
		err := p.Execute(ctx, reactCtx)
		if err != nil {
			reactCtx.FinishReason = "PipelineFatalError"
			reactCtx.Error = err
			return reactCtx, err
		}

		if reactCtx.IsFinished {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	return reactCtx, nil
}
