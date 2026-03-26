package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
)

type mockActor struct {
	err error
}

func (m *mockActor) Act(ctx *core.PipelineContext) error {
	return m.err
}

type mockThinker struct {
	finish bool
	err    error
}

func (m *mockThinker) Think(ctx *core.PipelineContext) error {
	if m.finish {
		ctx.IsFinished = true
	}
	return m.err
}

type mockObserver struct {
	err error
}

func (m *mockObserver) Observe(ctx *core.PipelineContext) error {
	return m.err
}

type mockTerminator struct {
	stop bool
	err  error
}

func (m *mockTerminator) CheckTermination(ctx *core.PipelineContext) (bool, error) {
	return m.stop, m.err
}

func TestActorStep(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		actor := &mockActor{}
		step := Actor(actor)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := step.Execute(ctx, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("with error in state", func(t *testing.T) {
		actor := &mockActor{}
		step := Actor(actor)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.Error = errors.New("state error")

		err := step.Execute(ctx, ctx)
		if err == nil {
			t.Error("Expected error")
		}
		if err != ctx.Error {
			t.Errorf("Expected %v, got %v", ctx.Error, err)
		}
	})
}

func TestActorStep_Name(t *testing.T) {
	actor := &mockActor{}
	step := Actor(actor)

	if step.Name() != "actor" {
		t.Errorf("Expected 'actor', got %q", step.Name())
	}
}

func TestThinkerStep(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		thinker := &mockThinker{}
		step := Thinker(thinker)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := step.Execute(ctx, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("marks finished", func(t *testing.T) {
		thinker := &mockThinker{finish: true}
		step := Thinker(thinker)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := step.Execute(ctx, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true")
		}
	})
}

func TestThinkerStep_Name(t *testing.T) {
	thinker := &mockThinker{}
	step := Thinker(thinker)

	if step.Name() != "thinker" {
		t.Errorf("Expected 'thinker', got %q", step.Name())
	}
}

func TestObserverStep(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		observer := &mockObserver{}
		step := Observer(observer)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := step.Execute(ctx, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestObserverStep_Name(t *testing.T) {
	observer := &mockObserver{}
	step := Observer(observer)

	if step.Name() != "observer" {
		t.Errorf("Expected 'observer', got %q", step.Name())
	}
}

func TestCheckFinishStep(t *testing.T) {
	t.Run("continue", func(t *testing.T) {
		terminator := &mockTerminator{stop: false}
		step := CheckFinish(terminator)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := step.Execute(ctx, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ctx.IsFinished {
			t.Error("Expected IsFinished to be false")
		}
	})

	t.Run("stop", func(t *testing.T) {
		terminator := &mockTerminator{stop: true}
		step := CheckFinish(terminator)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := step.Execute(ctx, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true")
		}
	})

	t.Run("terminator error", func(t *testing.T) {
		expectedErr := errors.New("terminator error")
		terminator := &mockTerminator{err: expectedErr}
		step := CheckFinish(terminator)
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := step.Execute(ctx, ctx)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestCheckFinishStep_Name(t *testing.T) {
	terminator := &mockTerminator{}
	step := CheckFinish(terminator)

	if step.Name() != "check_finish" {
		t.Errorf("Expected 'check_finish', got %q", step.Name())
	}
}