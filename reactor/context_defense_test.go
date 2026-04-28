package reactor

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/DotNetAge/goreact/core"
)

func TestToolResultPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	smallResult := "hello world"
	persisted := core.PersistToDisk("test_tool", smallResult, tmpDir, 100, 50)
	if persisted != nil {
		t.Error("small result should not be persisted")
	}

	largeResult := string(make([]byte, 500))
	for i := range largeResult {
		largeResult = largeResult[:i] + "x" + largeResult[i+1:]
	}
	persisted = core.PersistToDisk("test_tool", largeResult, tmpDir, 100, 50)
	if persisted == nil {
		t.Fatal("large result should be persisted")
	}
	if persisted.FullSize != 500 {
		t.Errorf("expected FullSize=500, got %d", persisted.FullSize)
	}
	if persisted.FilePath == "" {
		t.Error("persisted result should have a file path")
	}

	fullContent, err := os.ReadFile(persisted.FilePath)
	if err != nil {
		t.Fatalf("failed to read persisted result: %v", err)
	}
	if string(fullContent) != largeResult {
		t.Error("persisted content does not match original")
	}

	tag := core.PersistedResultTag(persisted)
	if tag == "" {
		t.Error("tag should not be empty")
	}
}

func TestMicroCompact(t *testing.T) {
	estimateFn := func(s string) int {
		return len(s) / 3
	}

	messages := []core.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: string(make([]byte, 6000))}, // ~2000 tokens
		{Role: "user", Content: "tell me more"},
		{Role: "assistant", Content: string(make([]byte, 6000))}, // ~2000 tokens
		{Role: "user", Content: "thanks"},
	}

	// Target: 1500 tokens, should compact the large messages
	result := core.MicroCompact(messages, estimateFn, 1500)
	if len(result) == 0 {
		t.Error("compact should not remove all messages")
	}
	if len(result) > len(messages) {
		t.Error("compact should not add messages")
	}

	// Last message should be preserved
	if result[len(result)-1].Content != "thanks" {
		t.Errorf("last message should be preserved, got: %q", result[len(result)-1].Content)
	}
}

func TestTrimJSONResult(t *testing.T) {
	// Small JSON should pass through
	small := `{"key": "value"}`
	trimmed := core.TrimJSONResult(small, 1000)
	if trimmed != small {
		t.Error("small JSON should not be trimmed")
	}

	// Large JSON array should be trimmed
	items := make([]string, 100)
	for i := range items {
		items[i] = fmt.Sprintf(`"item_%d_with_long_padding_to_make_it_bigger"`, i)
	}
	largeJSON := `{"results": [` + joinStrings(items, ",") + `]}`
	trimmed = core.TrimJSONResult(largeJSON, 500)
	if len(trimmed) > 600 { // allow some overhead for the truncation notice
		t.Errorf("trimmed result should be much smaller, got %d chars", len(trimmed))
	}
}

func joinStrings(items []string, sep string) string {
	result := ""
	for i, s := range items {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func TestExecuteToolWithPersistence(t *testing.T) {
	registry := NewToolRegistry()

	executor := core.NewToolExecutor(
		registry,
		core.WithMaxPersistChars(100),
	)

	largeTool := &mockLargeResultTool{size: 500}
	_ = registry.Register(largeTool)

	execResult, err := executor.Execute(nil, "mock_large", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := execResult.Result
	originalSize := 500
	if len(result) > 1000 {
		t.Errorf("result should be truncated/persisted (got %d chars), not the full %d chars", len(result), originalSize)
	}
	// Should contain the persisted marker
	if len(result) == originalSize {
		t.Error("result should be persisted/truncated, not the full original content")
	}
}

type mockLargeResultTool struct {
	size int
}

func (t *mockLargeResultTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "mock_large",
		Description: "returns a large result",
	}
}

func (t *mockLargeResultTool) Execute(_ context.Context, _ map[string]any) (any, error) {
	return string(make([]byte, t.size)), nil
}
