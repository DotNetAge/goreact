package tools

import (
	"context"
	"os"
	"testing"

	"github.com/DotNetAge/goreact/core"
)

var ctx = context.Background()

func TestCalculator_toFloat64(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		val, ok := ToFloat64(float64(3.14))
		if !ok || val != 3.14 {
			t.Errorf("Expected 3.14, got %f, ok=%v", val, ok)
		}
	})

	t.Run("float32", func(t *testing.T) {
		val, ok := ToFloat64(float32(2.71))
		if !ok || val < 2.70 || val > 2.72 {
			t.Errorf("Expected ~2.71, got %f, ok=%v", val, ok)
		}
	})

	t.Run("int", func(t *testing.T) {
		val, ok := ToFloat64(int(42))
		if !ok || val != 42 {
			t.Errorf("Expected 42, got %f, ok=%v", val, ok)
		}
	})

	t.Run("int64", func(t *testing.T) {
		val, ok := ToFloat64(int64(123456789))
		if !ok || val != 123456789 {
			t.Errorf("Expected 123456789, got %f, ok=%v", val, ok)
		}
	})

	t.Run("int32", func(t *testing.T) {
		val, ok := ToFloat64(int32(-10))
		if !ok || val != -10 {
			t.Errorf("Expected -10, got %f, ok=%v", val, ok)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		_, ok := ToFloat64("string")
		if ok {
			t.Error("Expected false for string")
		}

		_, ok = ToFloat64(nil)
		if ok {
			t.Error("Expected false for nil")
		}

		_, ok = ToFloat64(struct{}{})
		if ok {
			t.Error("Expected false for struct")
		}
	})
}

func TestEdit(t *testing.T) {
	edit := NewEdit()

	t.Run("basic edit", func(t *testing.T) {
		testFile := "/tmp/goreact_test_edit.txt"
		os.WriteFile(testFile, []byte("Hello World"), 0644)

		result, err := edit.Execute(ctx, map[string]any{
			"path": testFile,
			"edits": []any{
				map[string]any{"old_text": "World", "new_text": "Go"},
			},
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
		if resultMap["edits_applied"] != 1 {
			t.Errorf("Expected 1 edit, got %v", resultMap["edits_applied"])
		}
		os.Remove(testFile)
	})

	t.Run("missing path", func(t *testing.T) {
		_, err := edit.Execute(ctx, map[string]any{
			"edits": []any{map[string]any{"old_text": "a", "new_text": "b"}},
		})
		if err == nil {
			t.Error("Expected error for missing path")
		}
	})

	t.Run("missing edits", func(t *testing.T) {
		_, err := edit.Execute(ctx, map[string]any{"path": "/tmp/test.txt"})
		if err == nil {
			t.Error("Expected error for missing edits")
		}
	})

	t.Run("text not found", func(t *testing.T) {
		testFile := "/tmp/goreact_test_edit2.txt"
		os.WriteFile(testFile, []byte("Hello World"), 0644)

		_, err := edit.Execute(ctx, map[string]any{
			"path": testFile,
			"edits": []any{
				map[string]any{"old_text": "NotFound", "new_text": "X"},
			},
		})
		if err == nil {
			t.Error("Expected error when text not found")
		}
		os.Remove(testFile)
	})

	t.Run("invalid edit format", func(t *testing.T) {
		testFile := "/tmp/goreact_test_edit3.txt"
		os.WriteFile(testFile, []byte("Hello"), 0644)

		_, err := edit.Execute(ctx, map[string]any{
			"path": testFile,
			"edits": []any{
				map[string]any{"wrong_key": "value"},
			},
		})
		if err == nil {
			t.Error("Expected error for invalid edit format")
		}
		os.Remove(testFile)
	})

	t.Run("Name and Description", func(t *testing.T) {
		if edit.Info().Name != "edit" {
			t.Errorf("Expected 'edit', got %q", edit.Info().Name)
		}
		if edit.Info().Description == "" {
			t.Error("Expected non-empty description")
		}
		if edit.Info().SecurityLevel != core.LevelSensitive {
			t.Errorf("Expected LevelSensitive, got %v", edit.Info().SecurityLevel)
		}
	})
}

func TestTruncateString(t *testing.T) {
	t.Run("short string", func(t *testing.T) {
		result := truncateString("short", 10)
		if result != "short" {
			t.Errorf("Expected 'short', got %q", result)
		}
	})

	t.Run("long string", func(t *testing.T) {
		result := truncateString("this is a long string", 10)
		if result != "this is a ... (truncated)" {
			t.Errorf("Expected truncated string, got %q", result)
		}
	})

	t.Run("exact length", func(t *testing.T) {
		result := truncateString("abc", 3)
		if result != "abc" {
			t.Errorf("Expected 'abc', got %q", result)
		}
	})
}
