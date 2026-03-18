package steps

import (
	"context"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/observer"
)

// observerStep wraps the Observer interface into a pipeline.Step.
type observerStep struct {
	o observer.Observer
}

// Observer creates a new pipeline step that processes raw execution results.
// It parses, cleans, and translates them into semantic context.
func Observer(o observer.Observer) *observerStep {
	return &observerStep{o: o}
}

func (s *observerStep) Name() string {
	return "observer"
}

func (s *observerStep) Execute(ctx context.Context, state *core.PipelineContext) error {
	return s.o.Observe(state)
}
