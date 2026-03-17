package presets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

func successResult(toolName string, output any) *types.ExecutionResult {
	return &types.ExecutionResult{
		Success:  true,
		Output:   output,
		Metadata: map[string]any{"tool_name": toolName},
	}
}

func failResult(toolName string, err error) *types.ExecutionResult {
	return &types.ExecutionResult{
		Success:  false,
		Error:    err,
		Metadata: map[string]any{"tool_name": toolName},
	}
}

// === SmartObserver ===

func TestSmartObserverSuccess(t *testing.T) {
	o := NewSmartObserver()
	fb, err := o.Observe(successResult("calculator", 42), core.NewContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fb.ShouldContinue {
		t.Error("should continue")
	}
	if !strings.Contains(fb.Message, "42") {
		t.Errorf("message should contain result: %s", fb.Message)
	}
}

func TestSmartObserverFailure(t *testing.T) {
	o := NewSmartObserver()
	fb, _ := o.Observe(failResult("http", fmt.Errorf("connection refused")), core.NewContext())
	if !strings.Contains(fb.Message, "connection refused") {
		t.Errorf("message should contain error: %s", fb.Message)
	}
}

func TestSmartObserverLoopDetection(t *testing.T) {
	o := NewSmartObserver()
	ctx := core.NewContext()

	params := map[string]any{"url": "https://api.example.com"}
	result := &types.ExecutionResult{
		Success:  false,
		Error:    fmt.Errorf("timeout"),
		Metadata: map[string]any{"tool_name": "http", "parameters": params},
	}

	o.Observe(result, ctx)
	o.Observe(result, ctx)
	fb, _ := o.Observe(result, ctx)

	if fb.Metadata["loop_detected"] != true {
		t.Error("should detect loop after 3 repeats")
	}
	if !strings.Contains(fb.Message, "Loop detected") {
		t.Errorf("message should mention loop: %s", fb.Message)
	}
}

func TestSmartObserverUpdatesHistory(t *testing.T) {
	o := NewSmartObserver()
	ctx := core.NewContext()

	o.Observe(successResult("calculator", 42), ctx)

	history, ok := ctx.Get("history")
	if !ok {
		t.Fatal("history should be set in context")
	}
	if !strings.Contains(history.(string), "42") {
		t.Errorf("history should contain result: %s", history)
	}
}

// === StrictObserver ===

func TestStrictObserverValidResult(t *testing.T) {
	o := NewStrictObserver()
	fb, _ := o.Observe(successResult("calculator", 42), core.NewContext())
	if strings.Contains(fb.Message, "Validation issues") {
		t.Errorf("valid result should not have issues: %s", fb.Message)
	}
}

func TestStrictObserverHTTPError(t *testing.T) {
	o := NewStrictObserver()
	fb, _ := o.Observe(
		successResult("http", `{"status": 500, "body": "Internal Server Error"}`),
		core.NewContext(),
	)
	if !strings.Contains(fb.Message, "Validation issues") {
		t.Errorf("HTTP 500 should have validation issues: %s", fb.Message)
	}
	if fb.Metadata["validation_failed"] != true {
		t.Error("should mark validation_failed")
	}
}

func TestStrictObserverErrorPattern(t *testing.T) {
	o := NewStrictObserver()
	fb, _ := o.Observe(
		successResult("http", `{"error": "invalid API key"}`),
		core.NewContext(),
	)
	if !strings.Contains(fb.Message, "Validation issues") {
		t.Errorf("error pattern should trigger validation: %s", fb.Message)
	}
}

func TestStrictObserverLoopDetection(t *testing.T) {
	o := NewStrictObserver()
	ctx := core.NewContext()

	params := map[string]any{"cmd": "rm -rf /"}
	result := &types.ExecutionResult{
		Success:  false,
		Error:    fmt.Errorf("permission denied"),
		Metadata: map[string]any{"tool_name": "bash", "parameters": params},
	}

	o.Observe(result, ctx)
	fb, _ := o.Observe(result, ctx) // StrictObserver maxRepeats=2

	if fb.Metadata["loop_detected"] != true {
		t.Error("should detect loop after 2 repeats (strict mode)")
	}
}

// === VerboseObserver ===

type mockLogger struct {
	messages []string
}

func (l *mockLogger) Info(msg string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf("%s %v", msg, args))
}

func TestVerboseObserverLogs(t *testing.T) {
	logger := &mockLogger{}
	o := NewVerboseObserver(logger)

	o.Observe(successResult("calculator", 42), core.NewContext())

	if len(logger.messages) == 0 {
		t.Error("should have logged")
	}
	if !strings.Contains(logger.messages[0], "calculator") {
		t.Errorf("log should contain tool name: %s", logger.messages[0])
	}
}

// === ProductionObserver ===

func TestProductionObserver(t *testing.T) {
	o := NewProductionObserver()
	fb, err := o.Observe(successResult("calculator", 42), core.NewContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fb.ShouldContinue {
		t.Error("should continue")
	}
}
