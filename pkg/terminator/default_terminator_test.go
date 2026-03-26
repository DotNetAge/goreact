package terminator

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
)

func TestDefaultTerminator_CheckTermination(t *testing.T) {
	t.Run("Rule 1 - already finished", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.IsFinished = true

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !stop {
			t.Error("Expected stop=true when already finished")
		}
	})

	t.Run("Rule 2 - max steps exceeded", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.MaxSteps = 10
		ctx.CurrentStep = 10

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !stop {
			t.Error("Expected stop=true when max steps exceeded")
		}
		if ctx.FinishReason != "MaxStepsExceeded" {
			t.Errorf("Expected FinishReason 'MaxStepsExceeded', got %q", ctx.FinishReason)
		}
		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be set")
		}
	})

	t.Run("Rule 2 - beyond max steps", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.MaxSteps = 5
		ctx.CurrentStep = 6

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !stop {
			t.Error("Expected stop=true when beyond max steps")
		}
	})

	t.Run("Rule 3 - no stagnation (less than 3 traces)", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.MaxSteps = 10
		ctx.CurrentStep = 1
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "tool1"}})

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stop {
			t.Error("Expected stop=false when no stagnation detected")
		}
	})

	t.Run("Rule 3 - no stagnation (different actions)", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.MaxSteps = 10
		ctx.CurrentStep = 3
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "tool1"}})
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "tool2"}})
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "tool3"}})

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stop {
			t.Error("Expected stop=false when actions differ")
		}
	})

	t.Run("Rule 3 - stagnation detected (same action 3 times)", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.MaxSteps = 10
		ctx.CurrentStep = 3
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "same_tool"}})
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "same_tool"}})
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "same_tool"}})

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stop {
			t.Error("Expected stop=false (stagnation is warning only)")
		}
	})

	t.Run("nil actions are handled", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.MaxSteps = 10
		ctx.CurrentStep = 3
		ctx.AppendTrace(&core.Trace{})
		ctx.AppendTrace(&core.Trace{})
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "tool"}})

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stop {
			t.Error("Expected stop=false when some actions are nil")
		}
	})

	t.Run("continues when not at max steps", func(t *testing.T) {
		term := Default()
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.MaxSteps = 10
		ctx.CurrentStep = 5

		stop, err := term.CheckTermination(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stop {
			t.Error("Expected stop=false when below max steps")
		}
	})
}

func TestTerminator_Interface(t *testing.T) {
	var _ Terminator = Default()
}