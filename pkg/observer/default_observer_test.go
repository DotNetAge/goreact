package observer

import (
	"context"
	"errors"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
)

func TestDefaultObserver_Name(t *testing.T) {
	o := &defaultObserver{}
	_ = o
}

func TestDefaultObserver_Observe(t *testing.T) {
	o := &defaultObserver{}

	t.Run("nil trace", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("trace with nil action", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{})
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("trace with action but no output or error", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("trace with action and raw error", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		ctx.Set("raw_error", errors.New("test error"))
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		lastTrace := ctx.LastTrace()
		if lastTrace.Observation == nil {
			t.Fatal("Expected observation to be set")
		}
		if lastTrace.Observation.IsSuccess {
			t.Error("Expected IsSuccess to be false")
		}
		if lastTrace.Observation.Error == nil {
			t.Error("Expected Error to be set")
		}
	})

	t.Run("trace with action and string output", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		ctx.Set("raw_output", "string result")
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		lastTrace := ctx.LastTrace()
		if lastTrace.Observation == nil {
			t.Fatal("Expected observation to be set")
		}
		if !lastTrace.Observation.IsSuccess {
			t.Error("Expected IsSuccess to be true")
		}
		if lastTrace.Observation.Data != "string result" {
			t.Errorf("Expected 'string result', got %q", lastTrace.Observation.Data)
		}
	})

	t.Run("trace with action and byte output", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		ctx.Set("raw_output", []byte("bytes result"))
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		lastTrace := ctx.LastTrace()
		if lastTrace.Observation == nil {
			t.Fatal("Expected observation to be set")
		}
		if lastTrace.Observation.Data != "bytes result" {
			t.Errorf("Expected 'bytes result', got %q", lastTrace.Observation.Data)
		}
	})

	t.Run("trace with action and struct output (JSON marshal success)", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		ctx.Set("raw_output", map[string]any{"key": "value"})
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		lastTrace := ctx.LastTrace()
		if lastTrace.Observation == nil {
			t.Fatal("Expected observation to be set")
		}
		if lastTrace.Observation.Data != `{"key":"value"}` {
			t.Errorf("Expected JSON output, got %q", lastTrace.Observation.Data)
		}
	})

	t.Run("trace with nil output (context deletes key)", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		ctx.Set("raw_output", nil)
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		lastTrace := ctx.LastTrace()
		if lastTrace.Observation != nil {
			t.Error("Expected observation to NOT be set when raw_output is nil")
		}
	})

	t.Run("trace with existing observation", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{
			Step:       1,
			Thought:    "test thought",
			Action:     &core.Action{Name: "test"},
			Observation: &core.Observation{Data: "already observed"},
		})
		ctx.Set("raw_output", "new output")
		err := o.Observe(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("cleans up raw_error from context", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		ctx.Set("raw_error", errors.New("test error"))
		o.Observe(ctx)

		_, hasRawError := ctx.Get("raw_error")
		if hasRawError {
			t.Error("Expected raw_error to be cleaned up")
		}
	})

	t.Run("cleans up raw_output from context", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test"}})
		ctx.Set("raw_output", "output")
		o.Observe(ctx)

		_, hasRawOutput := ctx.Get("raw_output")
		if hasRawOutput {
			t.Error("Expected raw_output to be cleaned up")
		}
	})
}

func TestDefaultObserver_ImplementsInterface(t *testing.T) {
	var _ Observer = (*defaultObserver)(nil)
}

func TestMockObserver(t *testing.T) {
	o := &mockObserver{err: errors.New("test error")}
	ctx := core.NewPipelineContext(context.Background(), "test", "input")

	err := o.Observe(ctx)
	if err == nil {
		t.Error("Expected error")
	}
}

type mockObserver struct {
	err error
}

func (m *mockObserver) Observe(ctx *core.PipelineContext) error {
	return m.err
}

func (m *mockObserver) Name() string {
	return "mock"
}