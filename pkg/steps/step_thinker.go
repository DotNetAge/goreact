package steps

import (
	"context"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/thinker"
)

// Ensure thinkerStep implements gochat's pipeline.Step.
var _ pipeline.Step[*core.PipelineContext] = (*thinkerStep)(nil)

// thinkerStep wraps the Thinker interface into a pipeline.Step.
type thinkerStep struct {
	t thinker.Thinker
}

// Thinker creates a new pipeline step that reasons and plans the next move.
func Thinker(t thinker.Thinker) *thinkerStep {
	return &thinkerStep{t: t}
}

func (s *thinkerStep) Name() string {
	return "thinker"
}

func (s *thinkerStep) Execute(ctx context.Context, state *core.PipelineContext) error {
	// The Thinker itself might update state.IsFinished to true if it believes
	// the target intent is accomplished without needing an external action.
	return s.t.Think(state)
}
