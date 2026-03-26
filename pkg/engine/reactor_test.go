package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/core"
)

// ====================
// 1. Mock Components
// ====================

type mockThinker struct{}

func (m *mockThinker) Think(ctx *core.PipelineContext) error {
	// Let's pretend the Thinker needs 3 iterations to solve the problem.
	if ctx.CurrentStep <= 3 {
		// Needs more info
		trace := &core.Trace{
			Step:    ctx.CurrentStep,
			Thought: fmt.Sprintf("Step %d: I don't know the answer yet. I should search.", ctx.CurrentStep),
			Action: &core.Action{
				Name:  "SearchTool",
				Input: map[string]any{"query": "weather today"},
			},
		}
		ctx.AppendTrace(trace)
	} else {
		// Target achieved - don't append trace when finishing
		ctx.IsFinished = true
		ctx.FinalResult = "The weather today is sunny."
		ctx.FinishReason = "TaskCompleted"
	}
	return nil
}

type mockActor struct{}

func (m *mockActor) Act(ctx *core.PipelineContext) error {
	lastTrace := ctx.LastTrace()
	if lastTrace == nil || lastTrace.Action == nil {
		return nil
	}

	// Pretend to execute the Action
	if lastTrace.Action.Name == "SearchTool" {
		// Set some raw digital output in the shared state for the Observer to pick up
		ctx.Set("raw_output", `{"status":"ok", "data":{"weather":"sunny", "temp": 25}, "extra":"lots of noise"}`)
	}
	return nil
}

type mockObserver struct{}

func (m *mockObserver) Observe(ctx *core.PipelineContext) error {
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

	// 1. Pretend to denoise and extract the core info from JSON string
	cleanData := "sunny" // Extracted
	if strings.Contains(rawStr, "sunny") {
		cleanData = "Weather: sunny, Temp: 25"
	}

	// 2. Attach Observation back to the trace
	lastTrace.Observation = &core.Observation{
		Data:      cleanData,
		Raw:       rawStr,
		IsSuccess: true,
	}

	// Clear the temporary state queue
	ctx.Set("raw_output", nil)
	return nil
}

type mockTerminator struct{}

func (m *mockTerminator) CheckTermination(ctx *core.PipelineContext) (bool, error) {
	// Rule 1: Max steps safety boundary
	if ctx.CurrentStep >= ctx.MaxSteps {
		ctx.IsFinished = true
		ctx.FinishReason = "MaxStepsReached"
		return true, nil
	}

	// Rule 2: Goal accomplished gracefully
	if ctx.IsFinished {
		// The thinker has manually requested termination
		return true, nil
	}

	// Continue loop
	return false, nil
}

// ====================
// 2. Engine Unit Test
// ====================

func TestReactor_Run(t *testing.T) {
	// Initialize the Reactor with mock cognitive pipeline
	reactor := NewReactor(
		WithThinker(&mockThinker{}),
		WithActor(&mockActor{}),
		WithObserver(&mockObserver{}),
		WithTerminator(&mockTerminator{}),
	)

	// Run the engine
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	reactCtx, err := reactor.Run(ctx, "session-123", "What is the weather today?")

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !reactCtx.IsFinished {
		t.Fatalf("Expected context to be finished, but it is not.")
	}
	if reactCtx.FinishReason != "TaskCompleted" {
		t.Fatalf("Expected FinishReason 'TaskCompleted', got %s", reactCtx.FinishReason)
	}
	if reactCtx.FinalResult != "The weather today is sunny." {
		t.Fatalf("Expected FinalResult to be sunny, got %s", reactCtx.FinalResult)
	}

	// Ensure the loop ran exactly 3 times (3 traces appended)
	if reactCtx.CurrentStep != 4 {
		t.Fatalf("Expected CurrentStep to be 4 (next step after 3 completed), got %d", reactCtx.CurrentStep)
	}

	// Check the Scratchpad (Memory Traces)
	if len(reactCtx.Traces) != 3 {
		t.Fatalf("Expected 3 traces, got %d", len(reactCtx.Traces))
	}

	// Inspect the first trace (Thought + Action + Observation)
	trace1 := reactCtx.Traces[0]
	if trace1.Action.Name != "SearchTool" {
		t.Fatalf("Expected Action to be SearchTool, got %s", trace1.Action.Name)
	}
	if trace1.Observation == nil || trace1.Observation.Data != "Weather: sunny, Temp: 25" {
		t.Fatalf("Expected observation to be formatted correctly")
	}
}
