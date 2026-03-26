package steps

import (
	"context"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/DotNetAge/goreact/pkg/actor"
	"github.com/DotNetAge/goreact/pkg/core"
)

// Ensure actorStep implements gochat's pipeline.Step.
var _ pipeline.Step[*core.PipelineContext] = (*actorStep)(nil)

// actorStep wraps the Actor interface into a pipeline.Step.
type actorStep struct {
	a actor.Actor
}

// Actor creates a new pipeline step that executes the provided Actor.
// It maps the cognitive decision (Thought+Action) into physical execution.
func Actor(a actor.Actor) *actorStep {
	return &actorStep{a: a}
}

func (s *actorStep) Name() string {
	return "actor"
}

func (s *actorStep) Execute(ctx context.Context, state *core.PipelineContext) error {
	// Check if a Hook (e.g., SecurityHook) has injected an error into the state
	// indicating that the operation was rejected or failed authorization.
	if state.Error != nil {
		// We return the error so that gochat pipeline stops the current execution cycle
		// and triggers OnStepError hooks.
		return state.Error
	}

	// The state passed is exactly our PipelineContext (thanks to generics).
	return s.a.Act(state)
}
