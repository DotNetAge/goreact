package steps

import (
	"context"

	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/core"
)

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
	// The state passed is exactly our PipelineContext (thanks to generics).
	return s.a.Act(state)
}
