package steps

import (
	"context"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/terminator"
)

// Ensure checkFinishStep implements gochat's pipeline.Step.
var _ pipeline.Step[*core.PipelineContext] = (*checkFinishStep)(nil)

// checkFinishStep wraps the Terminator interface into a pipeline.Step.
// Unlike the others, its purpose is to validate state boundaries and decide
// whether to continue the loop.
type checkFinishStep struct {
	t terminator.Terminator
}

// CheckFinish creates a step that asserts boundaries, limits, and success criteria.
// If it decides the loop should end, it mutates state.IsFinished to true.
func CheckFinish(t terminator.Terminator) *checkFinishStep {
	return &checkFinishStep{t: t}
}

func (s *checkFinishStep) Name() string {
	return "check_finish"
}

func (s *checkFinishStep) Execute(ctx context.Context, state *core.PipelineContext) error {
	// Let the rules/strategies inside Terminator decide.
	stop, err := s.t.CheckTermination(state)
	if err != nil {
		return err // A critical failure in the rule engine itself.
	}

	if stop {
		// Stop was returned: lock state and mark as finished.
		// Note that the FinishReason is usually set by the Terminator internally.
		state.IsFinished = true
	}
	return nil
}
