package observer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ray/goreact/pkg/core"
)

// mockObserver is a mock implementation of the Observer interface
type mockObserver struct {
	observedData string
	err          error
}

func (m *mockObserver) Observe(ctx *core.PipelineContext) error {
	if m.err != nil {
		return m.err
	}

	lastTrace := ctx.LastTrace()
	if lastTrace == nil || lastTrace.Action == nil {
		return nil
	}

	raw, ok := ctx.Get("raw_output")
	if !ok {
		return nil
	}

	rawStr, ok := raw.(string)
	if !ok {
		return nil
	}

	// Process and attach observation
	cleanData := strings.ToUpper(rawStr)
	lastTrace.Observation = &core.Observation{
		Data:      cleanData,
		Raw:       rawStr,
		IsSuccess: true,
	}

	m.observedData = cleanData
	ctx.Set("raw_output", nil)
	return nil
}

func TestMockObserver_Observe(t *testing.T) {
	t.Run("successful observation", func(t *testing.T) {
		observer := &mockObserver{}

		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{
			Step:    1,
			Thought: "Testing",
			Action: &core.Action{
				Name:  "TestTool",
				Input: map[string]any{"key": "value"},
			},
		})
		ctx.Set("raw_output", "test result")

		err := observer.Observe(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		trace := ctx.LastTrace()
		if trace.Observation == nil {
			t.Fatal("Expected observation to be set")
		}
		if trace.Observation.Data != "TEST RESULT" {
			t.Errorf("Expected data 'TEST RESULT', got '%s'", trace.Observation.Data)
		}
		if trace.Observation.Raw != "test result" {
			t.Errorf("Expected raw 'test result', got '%s'", trace.Observation.Raw)
		}
		if !trace.Observation.IsSuccess {
			t.Error("Expected IsSuccess to be true")
		}

		// Verify raw_output was cleared (set to nil)
		rawOutput, ok := ctx.Get("raw_output")
		if ok {
			t.Error("Expected raw_output to be cleared")
		}
		if rawOutput != nil {
			t.Errorf("Expected raw_output to be nil, got %v", rawOutput)
		}
	})

	t.Run("no action to observe", func(t *testing.T) {
		observer := &mockObserver{}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := observer.Observe(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("error handling", func(t *testing.T) {
		observer := &mockObserver{
			err: assertError("observer failed"),
		}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")

		err := observer.Observe(ctx)
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
