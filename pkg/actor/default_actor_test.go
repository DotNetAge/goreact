package actor_test

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
)

// mockActor is a mock implementation of the Actor interface
type mockActor struct {
	executedAction *core.Action
	err            error
}

func (m *mockActor) Act(ctx *core.PipelineContext) error {
	if m.err != nil {
		return m.err
	}

	lastTrace := ctx.LastTrace()
	if lastTrace == nil || lastTrace.Action == nil {
		return nil
	}

	m.executedAction = lastTrace.Action

	// Simulate tool execution by storing result in context
	ctx.Set("raw_output", "mock result")
	return nil
}

func TestMockActor_Act(t *testing.T) {
	t.Run("successful action execution", func(t *testing.T) {
		actor := &mockActor{}

		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{
			Step:    1,
			Thought: "Testing",
			Action: &core.Action{
				Name:  "TestTool",
				Input: map[string]any{"key": "value"},
			},
		})

		err := actor.Act(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if actor.executedAction == nil {
			t.Fatal("Expected action to be executed")
		}
		if actor.executedAction.Name != "TestTool" {
			t.Errorf("Expected action name 'TestTool', got '%s'", actor.executedAction.Name)
		}

		// Verify result was stored
		rawOutput, ok := ctx.Get("raw_output")
		if !ok {
			t.Error("Expected raw_output to be set")
		}
		if rawOutput != "mock result" {
			t.Errorf("Expected 'mock result', got '%v'", rawOutput)
		}
	})

	t.Run("no action to execute", func(t *testing.T) {
		actor := &mockActor{}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := actor.Act(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if actor.executedAction != nil {
			t.Error("Expected no action to be executed")
		}
	})

	t.Run("error handling", func(t *testing.T) {
		actor := &mockActor{
			err: assertError("actor failed"),
		}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := actor.Act(ctx)
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
