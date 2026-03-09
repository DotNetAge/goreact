package presets

import (
	"fmt"
	"testing"
	"time"

	"github.com/ray/goreact/pkg/actor/debug"
	"github.com/ray/goreact/pkg/actor/schema"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/tool"
	"github.com/ray/goreact/pkg/types"
)

func setupToolManager() *tool.Manager {
	tm := tool.NewManager()
	tm.RegisterTools(
		schema.NewTool("echo", "Echo input", schema.Define(
			schema.Param("msg", schema.String, "Message").Required(),
		), func(p schema.ValidatedParams) (any, error) {
			return "echo: " + p.GetString("msg"), nil
		}),
		schema.NewTool("fail", "Always fails", schema.Define(
			schema.Param("msg", schema.String, "Message").Required(),
		), func(p schema.ValidatedParams) (any, error) {
			return nil, fmt.Errorf("intentional failure")
		}),
	)
	return tm
}

func newAction(toolName string, params map[string]any) *types.Action {
	return &types.Action{ToolName: toolName, Parameters: params}
}

// === ResilientActor ===

func TestResilientActorSuccess(t *testing.T) {
	actor := NewResilientActor(setupToolManager(),
		WithTimeout(5*time.Second),
		WithRetry(3, 10*time.Millisecond),
	)

	result, err := actor.Act(
		newAction("echo", map[string]any{"msg": "hello"}),
		core.NewContext(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Output != "echo: hello" {
		t.Errorf("expected 'echo: hello', got '%v'", result.Output)
	}
}

func TestResilientActorNilAction(t *testing.T) {
	actor := NewResilientActor(setupToolManager())

	_, err := actor.Act(nil, core.NewContext())
	if err == nil {
		t.Error("expected error for nil action")
	}
}

func TestResilientActorToolNotFound(t *testing.T) {
	actor := NewResilientActor(setupToolManager())

	result, err := actor.Act(
		newAction("nonexistent", map[string]any{}),
		core.NewContext(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for nonexistent tool")
	}
}

// === DebugActor ===

func TestDebugActorTracing(t *testing.T) {
	tracer := debug.NewExecutionTracer(true)
	profiler := debug.NewPerformanceProfiler()
	actor := NewDebugActor(setupToolManager(), tracer, profiler)

	actor.Act(
		newAction("echo", map[string]any{"msg": "test"}),
		core.NewContext(),
	)
	actor.Act(
		newAction("fail", map[string]any{"msg": "test"}),
		core.NewContext(),
	)

	records := tracer.GetRecords()
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ToolName != "echo" {
		t.Errorf("expected 'echo', got '%s'", records[0].ToolName)
	}
	if records[0].Error != nil {
		t.Error("first record should have no error")
	}
	if records[1].Error == nil {
		t.Error("second record should have error")
	}

	report := profiler.Report()
	if report == "" {
		t.Error("profiler report should not be empty")
	}
}

// === SafeActor ===

func TestSafeActorAllowed(t *testing.T) {
	actor := NewSafeActor(setupToolManager(),
		WithAllowedTools("echo"),
	)

	result, err := actor.Act(
		newAction("echo", map[string]any{"msg": "hello"}),
		core.NewContext(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success for allowed tool")
	}
}

func TestSafeActorDenied(t *testing.T) {
	actor := NewSafeActor(setupToolManager(),
		WithAllowedTools("echo"),
	)

	result, err := actor.Act(
		newAction("fail", map[string]any{"msg": "hello"}),
		core.NewContext(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for denied tool")
	}
	if result.Error == nil {
		t.Error("expected error message for denied tool")
	}
}

func TestSafeActorNoRestriction(t *testing.T) {
	// 不设白名单 = 全部允许
	actor := NewSafeActor(setupToolManager())

	result, err := actor.Act(
		newAction("echo", map[string]any{"msg": "hello"}),
		core.NewContext(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success when no restriction")
	}
}

// === ProductionActor ===

func TestProductionActorSuccess(t *testing.T) {
	actor := NewProductionActor(setupToolManager())

	result, err := actor.Act(
		newAction("echo", map[string]any{"msg": "hello"}),
		core.NewContext(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestProductionActorFailure(t *testing.T) {
	actor := NewProductionActor(setupToolManager())

	result, err := actor.Act(
		newAction("fail", map[string]any{"msg": "hello"}),
		core.NewContext(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
}

func TestProductionActorNilAction(t *testing.T) {
	actor := NewProductionActor(setupToolManager())

	_, err := actor.Act(nil, core.NewContext())
	if err == nil {
		t.Error("expected error for nil action")
	}
}

func TestProductionActorMetadata(t *testing.T) {
	actor := NewProductionActor(setupToolManager())

	result, _ := actor.Act(
		newAction("echo", map[string]any{"msg": "hello"}),
		core.NewContext(),
	)

	if result.Metadata == nil {
		t.Fatal("expected metadata")
	}
	if result.Metadata["tool_name"] != "echo" {
		t.Errorf("expected tool_name 'echo', got '%v'", result.Metadata["tool_name"])
	}
	if _, ok := result.Metadata["duration"]; !ok {
		t.Error("expected duration in metadata")
	}
}
