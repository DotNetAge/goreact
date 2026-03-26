package terminator_test

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
)

// mockTerminator is a mock implementation of the Terminator interface
type mockTerminator struct {
	shouldStop bool
	err        error
}

func (m *mockTerminator) CheckTermination(ctx *core.PipelineContext) (bool, error) {
	if m.err != nil {
		return false, m.err
	}

	if m.shouldStop {
		ctx.IsFinished = true
		ctx.FinishReason = "TestStopped"
		return true, nil
	}

	// Check max steps
	if ctx.CurrentStep >= ctx.MaxSteps {
		ctx.IsFinished = true
		ctx.FinishReason = "MaxStepsReached"
		return true, nil
	}

	return false, nil
}

func TestMockTerminator_CheckTermination(t *testing.T) {
	t.Run("continue loop", func(t *testing.T) {
		terminator := &mockTerminator{
			shouldStop: false,
		}

		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.CurrentStep = 2
		ctx.MaxSteps = 10

		stop, err := terminator.CheckTermination(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if stop {
			t.Error("Expected to continue loop, got stop=true")
		}
		if ctx.IsFinished {
			t.Error("Expected context not to be finished")
		}
	})

	t.Run("stop by flag", func(t *testing.T) {
		terminator := &mockTerminator{
			shouldStop: true,
		}

		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		stop, err := terminator.CheckTermination(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !stop {
			t.Error("Expected stop=true, got false")
		}
		if !ctx.IsFinished {
			t.Error("Expected context to be finished")
		}
		if ctx.FinishReason != "TestStopped" {
			t.Errorf("Expected FinishReason 'TestStopped', got '%s'", ctx.FinishReason)
		}
	})

	t.Run("max steps reached", func(t *testing.T) {
		terminator := &mockTerminator{
			shouldStop: false,
		}

		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.CurrentStep = 10
		ctx.MaxSteps = 10

		stop, err := terminator.CheckTermination(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !stop {
			t.Error("Expected stop=true due to max steps")
		}
		if !ctx.IsFinished {
			t.Error("Expected context to be finished")
		}
		if ctx.FinishReason != "MaxStepsReached" {
			t.Errorf("Expected FinishReason 'MaxStepsReached', got '%s'", ctx.FinishReason)
		}
	})

	t.Run("error handling", func(t *testing.T) {
		terminator := &mockTerminator{
			err: assertError("terminator failed"),
		}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		stop, err := terminator.CheckTermination(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if stop {
			t.Error("Expected stop=false on error")
		}
	})
}

func assertError(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
