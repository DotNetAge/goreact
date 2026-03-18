package thinker_test

import (
	"context"
	"testing"

	"github.com/ray/goreact/pkg/core"
)

// mockThinker is a mock implementation of the Thinker interface
type mockThinker struct {
	shouldFinish bool
	thought      string
	err          error
}

func (m *mockThinker) Think(ctx *core.PipelineContext) error {
	if m.err != nil {
		return m.err
	}

	trace := &core.Trace{
		Step:    ctx.CurrentStep,
		Thought: m.thought,
	}

	if m.shouldFinish {
		ctx.IsFinished = true
		ctx.FinalResult = "Task completed"
		ctx.FinishReason = "TestCompleted"
	} else {
		trace.Action = &core.Action{
			Name:  "TestTool",
			Input: map[string]any{"test": "value"},
		}
	}

	ctx.AppendTrace(trace)
	return nil
}

func TestMockThinker_Think(t *testing.T) {
	t.Run("successful thought with action", func(t *testing.T) {
		thinker := &mockThinker{
			shouldFinish: false,
			thought:      "I need to test this",
		}

		ctx := core.NewPipelineContext(
			context.Background(),
			"test-session",
			"What is 2+2?",
		)

		err := thinker.Think(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify trace was appended
		if len(ctx.Traces) != 1 {
			t.Fatalf("Expected 1 trace, got %d", len(ctx.Traces))
		}

		trace := ctx.Traces[0]
		if trace.Thought != "I need to test this" {
			t.Errorf("Expected thought 'I need to test this', got '%s'", trace.Thought)
		}
		if trace.Action == nil {
			t.Error("Expected action to be set")
		}
		if ctx.IsFinished {
			t.Error("Expected context not to be finished")
		}
	})

	t.Run("finish condition", func(t *testing.T) {
		thinker := &mockThinker{
			shouldFinish: true,
			thought:      "Task is done",
		}

		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		err := thinker.Think(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !ctx.IsFinished {
			t.Error("Expected context to be finished")
		}
		if ctx.FinalResult != "Task completed" {
			t.Errorf("Expected final result 'Task completed', got '%s'", ctx.FinalResult)
		}
	})

	t.Run("error handling", func(t *testing.T) {
		thinker := &mockThinker{
			err: assertError("thinker failed"),
		}

		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		err := thinker.Think(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
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
