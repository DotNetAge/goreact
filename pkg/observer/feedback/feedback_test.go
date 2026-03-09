package feedback

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

func TestSmartGeneratorSuccess(t *testing.T) {
	gen := NewSmartGenerator()
	ctx := core.NewContext()

	fb := gen.Generate(&types.ExecutionResult{
		Success:  true,
		Output:   42,
		Metadata: map[string]any{"tool_name": "calculator"},
	}, ctx)

	if fb == "" {
		t.Error("feedback should not be empty")
	}
	if !strings.Contains(fb, "42") {
		t.Errorf("feedback should contain result: %s", fb)
	}
}

func TestSmartGeneratorFailure(t *testing.T) {
	gen := NewSmartGenerator()
	ctx := core.NewContext()

	fb := gen.Generate(&types.ExecutionResult{
		Success:  false,
		Error:    fmt.Errorf("connection refused"),
		Metadata: map[string]any{"tool_name": "http"},
	}, ctx)

	if !strings.Contains(fb, "connection refused") {
		t.Errorf("feedback should mention error: %s", fb)
	}
}

func TestSmartGeneratorHTTP404(t *testing.T) {
	gen := NewSmartGenerator()
	ctx := core.NewContext()

	fb := gen.Generate(&types.ExecutionResult{
		Success:  true,
		Output:   `{"status": 404, "body": "Not Found"}`,
		Metadata: map[string]any{"tool_name": "http"},
	}, ctx)

	if !strings.Contains(fb, "404") {
		t.Errorf("feedback should mention 404: %s", fb)
	}
}

func TestSmartGeneratorEmptyResult(t *testing.T) {
	gen := NewSmartGenerator()
	ctx := core.NewContext()

	fb := gen.Generate(&types.ExecutionResult{
		Success:  true,
		Output:   "[]",
		Metadata: map[string]any{"tool_name": "search"},
	}, ctx)

	if !strings.Contains(strings.ToLower(fb), "empty") {
		t.Errorf("feedback should mention empty result: %s", fb)
	}
}

func TestSmartGeneratorNoMetadata(t *testing.T) {
	gen := NewSmartGenerator()
	ctx := core.NewContext()

	fb := gen.Generate(&types.ExecutionResult{
		Success: true,
		Output:  "hello",
	}, ctx)

	if fb == "" {
		t.Error("feedback should not be empty even without metadata")
	}
}
