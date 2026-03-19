package builtin

import (
	"context"
	"os"
	"testing"
)

func TestCalculator(t *testing.T) {
	// 创建计算器工具
	calculator := NewCalculator()

	// 测试加法
	_, err := calculator.Execute(context.Background(), map[string]any{"operation": "add", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for add, got %v", err)
	}

	// 测试减法
	_, err = calculator.Execute(context.Background(), map[string]any{"operation": "subtract", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for subtract, got %v", err)
	}

	// 测试乘法
	_, err = calculator.Execute(context.Background(), map[string]any{"operation": "multiply", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for multiply, got %v", err)
	}

	// 测试除法
	_, err = calculator.Execute(context.Background(), map[string]any{"operation": "divide", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for divide, got %v", err)
	}

	// 测试除法错误（除数为 0）
	_, err = calculator.Execute(context.Background(), map[string]any{"operation": "divide", "a": 10, "b": 0})
	if err == nil {
		t.Error("Expected error for divide by zero")
	}

	// 测试未知操作
	_, err = calculator.Execute(context.Background(), map[string]any{"operation": "unknown", "a": 10, "b": 5})
	if err == nil {
		t.Error("Expected error for unknown operation")
	}
}

func TestEcho(t *testing.T) {
	// 创建echo工具
	echo := NewEcho()

	// 测试echo
	result, err := echo.Execute(context.Background(), map[string]any{"message": "Hello, World!"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "Echo: Hello, World!" {
		t.Errorf("Expected echo result 'Echo: Hello, World!', got '%v'", result)
	}
}

func TestDateTime(t *testing.T) {
	// 创建DateTime工具
	datetime := NewDateTime()

	// 测试获取当前时间
	result, err := datetime.Execute(context.Background(), map[string]any{"operation": "now"})
	if err != nil {
		t.Errorf("Expected no error for now, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for now")
	}

	// 测试格式化时间 - 使用 RFC3339 格式
	result, err = datetime.Execute(context.Background(), map[string]any{"operation": "format", "value": "2026-03-08T12:00:00Z", "format": "2006-01-02"})
	if err != nil {
		t.Errorf("Expected no error for format, got %v", err)
	}

	// 测试解析时间 - 使用 RFC3339 格式
	result, err = datetime.Execute(context.Background(), map[string]any{"operation": "parse", "value": "2026-03-08T12:00:00Z", "format": "2006-01-02"})
	if err != nil {
		t.Errorf("Expected no error for parse, got %v", err)
	}

	// 测试未知操作
	_, err = datetime.Execute(context.Background(), map[string]any{"operation": "unknown"})
	if err == nil {
		t.Error("Expected error for unknown operation")
	}
}

func TestGrep(t *testing.T) {
	// 创建 Grep 工具
	grep := NewGrep()

	// 测试在当前文件中查找模式
	result, err := grep.Execute(context.Background(), map[string]any{"pattern": "TestGrep", "path": "./builtin_test.go"})
	if err != nil {
		t.Errorf("Expected no error for grep, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for grep")
	}
}

func TestReplace(t *testing.T) {
	// 创建 Replace 工具
	replace := NewReplace()

	// 创建一个临时测试文件
	testFile := "/tmp/test_replace.txt"
	initialContent := "Hello World! Hello Go! Hello Testing!"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	t.Run("replace all occurrences", func(t *testing.T) {
		result, err := replace.Execute(context.Background(), map[string]any{
			"path":    testFile,
			"search":  "Hello",
			"replace": "Hi",
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
		if int(resultMap["replacements"].(int)) != 3 {
			t.Errorf("Expected 3 replacements, got %v", resultMap["replacements"])
		}

		// 验证文件内容
		content, _ := os.ReadFile(testFile)
		expected := "Hi World! Hi Go! Hi Testing!"
		if string(content) != expected {
			t.Errorf("Expected file content '%s', got '%s'", expected, string(content))
		}
	})

	t.Run("replace with limit", func(t *testing.T) {
		// 重置文件内容
		os.WriteFile(testFile, []byte(initialContent), 0644)

		result, err := replace.Execute(context.Background(), map[string]any{
			"path":    testFile,
			"search":  "Hello",
			"replace": "Hi",
			"limit":   2.0,
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMap := result.(map[string]any)
		if int(resultMap["replacements"].(int)) != 2 {
			t.Errorf("Expected 2 replacements, got %v", resultMap["replacements"])
		}

		// 验证文件内容 (只替换前 2 个)
		content, _ := os.ReadFile(testFile)
		expected := "Hi World! Hi Go! Hello Testing!"
		if string(content) != expected {
			t.Errorf("Expected file content '%s', got '%s'", expected, string(content))
		}
	})

	t.Run("text not found", func(t *testing.T) {
		result, err := replace.Execute(context.Background(), map[string]any{
			"path":    testFile,
			"search":  "NotFound",
			"replace": "Something",
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMap := result.(map[string]any)
		if resultMap["success"] != false {
			t.Error("Expected success to be false when text not found")
		}
		if int(resultMap["replacements"].(int)) != 0 {
			t.Errorf("Expected 0 replacements, got %v", resultMap["replacements"])
		}
	})
}
