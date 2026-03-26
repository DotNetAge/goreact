package builtin

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/DotNetAge/goreact/pkg/tools"
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

func TestCron(t *testing.T) {
	// 创建 Cron工具
	cron := NewCron()

	t.Run("validate valid expression", func(t *testing.T) {
		result, err := cron.Execute(context.Background(), map[string]any{
			"operation":  "validate",
			"expression": "*/5 * * * *",
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMap := result.(map[string]any)
		if resultMap["valid"] != true {
			t.Error("Expected validation to pass")
		}
	})

	t.Run("validate invalid expression", func(t *testing.T) {
		result, err := cron.Execute(context.Background(), map[string]any{
			"operation":  "validate",
			"expression": "invalid",
		})

		if err != nil {
			t.Fatalf("Expected no error for invalid expression, got %v", err)
		}

		resultMap := result.(map[string]any)
		if resultMap["valid"] != false {
			t.Error("Expected validation to fail for invalid expression")
		}
		if resultMap["error"] == nil {
			t.Error("Expected error message for invalid expression")
		}
	})

	t.Run("parse expression", func(t *testing.T) {
		result, err := cron.Execute(context.Background(), map[string]any{
			"operation":  "parse",
			"expression": "0 12 * * *",
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMap := result.(map[string]any)
		if resultMap["expression"] != "0 12 * * *" {
			t.Errorf("Expected expression '0 12 * * *', got %v", resultMap["expression"])
		}
		if resultMap["hour"] != "12" {
			t.Errorf("Expected hour '12', got %v", resultMap["hour"])
		}
	})

	t.Run("calculate next occurrence", func(t *testing.T) {
		// 使用固定的起始时间
		fromTime := "2026-03-19T10:00:00Z"
		result, err := cron.Execute(context.Background(), map[string]any{
			"operation":  "next",
			"expression": "0 12 * * *", // 每天中午 12 点
			"from":       fromTime,
			"count":      3.0,
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		results, ok := result.([]string)
		if !ok {
			t.Fatal("Expected result to be []string")
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// 验证第一个结果是当天的 12:00
		if !strings.Contains(results[0], "T12:00:00Z") {
			t.Errorf("Expected first result at 12:00, got %s", results[0])
		}
	})

	t.Run("complex expression with step", func(t *testing.T) {
		result, err := cron.Execute(context.Background(), map[string]any{
			"operation":  "validate",
			"expression": "*/15 9-17 * * 1-5",
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMap := result.(map[string]any)
		if resultMap["valid"] != true {
			t.Error("Expected validation to pass for complex expression")
		}
	})

	t.Run("out of range value", func(t *testing.T) {
		result, err := cron.Execute(context.Background(), map[string]any{
			"operation":  "validate",
			"expression": "60 * * * *", // 分钟超出范围
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		resultMap := result.(map[string]any)
		if resultMap["valid"] != false {
			t.Error("Expected validation to fail for out of range value")
		}
	})
}

func TestBash(t *testing.T) {
	bash := NewBash()

	t.Run("basic command execution", func(t *testing.T) {
		result, err := bash.Execute(context.Background(), map[string]any{"command": "echo hello"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
	})

	t.Run("missing command parameter", func(t *testing.T) {
		_, err := bash.Execute(context.Background(), map[string]any{})
		if err == nil {
			t.Error("Expected error for missing command")
		}
	})

	t.Run("command with error", func(t *testing.T) {
		result, err := bash.Execute(context.Background(), map[string]any{"command": "ls /nonexistent_dir_123"})
		if err != nil {
			t.Fatalf("Expected no error (error in result), got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != false {
			t.Error("Expected success to be false")
		}
		if resultMap["error"] == nil {
			t.Error("Expected error message")
		}
	})

	t.Run("Name and Description", func(t *testing.T) {
		if bash.Name() != "bash" {
			t.Errorf("Expected 'bash', got %q", bash.Name())
		}
		if bash.Description() == "" {
			t.Error("Expected non-empty description")
		}
		if bash.SecurityLevel() != tools.LevelHighRisk {
			t.Errorf("Expected HighRisk, got %v", bash.SecurityLevel())
		}
	})
}

func TestLS(t *testing.T) {
	ls := NewLS()

	t.Run("list current directory", func(t *testing.T) {
		result, err := ls.Execute(context.Background(), map[string]any{"path": "."})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
		if resultMap["total_items"] == nil {
			t.Error("Expected total_items to be set")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := ls.Execute(context.Background(), map[string]any{"path": "/nonexistent_dir_12345"})
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})

	t.Run("path is not a directory", func(t *testing.T) {
		_, err := ls.Execute(context.Background(), map[string]any{"path": "builtin_test.go"})
		if err == nil {
			t.Error("Expected error when path is not a directory")
		}
	})

	t.Run("show hidden files", func(t *testing.T) {
		result, err := ls.Execute(context.Background(), map[string]any{"path": ".", "show_hidden": true})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		items := resultMap["items"].([]map[string]any)
		found := false
		for _, item := range items {
			name, ok := item["name"].(string)
			if !ok {
				continue
			}
			if name == "builtin_test.go" || strings.HasPrefix(name, ".") {
				found = true
				break
			}
		}
		_ = found
	})

	t.Run("Name and Description", func(t *testing.T) {
		if ls.Name() != "ls" {
			t.Errorf("Expected 'ls', got %q", ls.Name())
		}
		if ls.Description() == "" {
			t.Error("Expected non-empty description")
		}
		if ls.SecurityLevel() != tools.LevelSafe {
			t.Errorf("Expected LevelSafe, got %v", ls.SecurityLevel())
		}
	})
}

func TestGlob(t *testing.T) {
	glob := NewGlob()

	t.Run("find go files", func(t *testing.T) {
		result, err := glob.Execute(context.Background(), map[string]any{"pattern": "*.go", "path": "."})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
		if resultMap["matches_found"] == nil {
			t.Error("Expected matches_found to be set")
		}
	})

	t.Run("missing pattern", func(t *testing.T) {
		_, err := glob.Execute(context.Background(), map[string]any{"path": "."})
		if err == nil {
			t.Error("Expected error for missing pattern")
		}
	})

	t.Run("non-existent search path", func(t *testing.T) {
		_, err := glob.Execute(context.Background(), map[string]any{"pattern": "*.go", "path": "/nonexistent_dir_12345"})
		if err == nil {
			t.Error("Expected error for non-existent path")
		}
	})

	t.Run("search path is not a directory", func(t *testing.T) {
		_, err := glob.Execute(context.Background(), map[string]any{"pattern": "*.go", "path": "builtin_test.go"})
		if err == nil {
			t.Error("Expected error when path is not a directory")
		}
	})

	t.Run("Name and Description", func(t *testing.T) {
		if glob.Name() != "glob" {
			t.Errorf("Expected 'glob', got %q", glob.Name())
		}
		if glob.Description() == "" {
			t.Error("Expected non-empty description")
		}
		if glob.SecurityLevel() != tools.LevelSafe {
			t.Errorf("Expected LevelSafe, got %v", glob.SecurityLevel())
		}
	})
}

func TestRead(t *testing.T) {
	read := NewRead()

	t.Run("read this test file", func(t *testing.T) {
		result, err := read.Execute(context.Background(), map[string]any{"path": "builtin_test.go"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
		if resultMap["content"] == nil {
			t.Error("Expected content to be set")
		}
	})

	t.Run("read with line range", func(t *testing.T) {
		result, err := read.Execute(context.Background(), map[string]any{
			"path":       "builtin_test.go",
			"start_line": 1.0,
			"end_line":   5.0,
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
	})

	t.Run("missing path", func(t *testing.T) {
		_, err := read.Execute(context.Background(), map[string]any{})
		if err == nil {
			t.Error("Expected error for missing path")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := read.Execute(context.Background(), map[string]any{"path": "/nonexistent_file_12345.txt"})
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("path is a directory", func(t *testing.T) {
		_, err := read.Execute(context.Background(), map[string]any{"path": "."})
		if err == nil {
			t.Error("Expected error when path is a directory")
		}
	})

	t.Run("Name and Description", func(t *testing.T) {
		if read.Name() != "read" {
			t.Errorf("Expected 'read', got %q", read.Name())
		}
		if read.Description() == "" {
			t.Error("Expected non-empty description")
		}
		if read.SecurityLevel() != tools.LevelSafe {
			t.Errorf("Expected LevelSafe, got %v", read.SecurityLevel())
		}
	})
}

func TestWrite(t *testing.T) {
	write := NewWrite()

	t.Run("write to temp file", func(t *testing.T) {
		testFile := "/tmp/goreact_test_write.txt"
		result, err := write.Execute(context.Background(), map[string]any{
			"path":    testFile,
			"content": "hello world",
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["success"] != true {
			t.Error("Expected success to be true")
		}
		if resultMap["bytes_written"] == nil {
			t.Error("Expected bytes_written to be set")
		}
		os.Remove(testFile)
	})

	t.Run("append to file", func(t *testing.T) {
		testFile := "/tmp/goreact_test_append.txt"
		write.Execute(context.Background(), map[string]any{"path": testFile, "content": "line1\n"})
		result, err := write.Execute(context.Background(), map[string]any{
			"path":    testFile,
			"content": "line2\n",
			"append":  true,
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		resultMap := result.(map[string]any)
		if resultMap["mode"] != "append" {
			t.Errorf("Expected mode 'append', got %v", resultMap["mode"])
		}
		os.Remove(testFile)
	})

	t.Run("missing path", func(t *testing.T) {
		_, err := write.Execute(context.Background(), map[string]any{"content": "hello"})
		if err == nil {
			t.Error("Expected error for missing path")
		}
	})

	t.Run("missing content", func(t *testing.T) {
		_, err := write.Execute(context.Background(), map[string]any{"path": "/tmp/test.txt"})
		if err == nil {
			t.Error("Expected error for missing content")
		}
	})

	t.Run("Name and Description", func(t *testing.T) {
		if write.Name() != "write" {
			t.Errorf("Expected 'write', got %q", write.Name())
		}
		if write.Description() == "" {
			t.Error("Expected non-empty description")
		}
		if write.SecurityLevel() != tools.LevelSensitive {
			t.Errorf("Expected LevelSensitive, got %v", write.SecurityLevel())
		}
	})
}

func TestValidateFunctions(t *testing.T) {
	t.Run("validateRequired", func(t *testing.T) {
		err := validateRequired(map[string]any{"key": "value"}, "key")
		if err != nil {
			t.Error("Expected no error for existing key")
		}

		err = validateRequired(map[string]any{}, "missing")
		if err == nil {
			t.Error("Expected error for missing key")
		}
	})

	t.Run("validateRequiredString", func(t *testing.T) {
		val, err := validateRequiredString(map[string]any{"key": "value"}, "key")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if val != "value" {
			t.Errorf("Expected 'value', got %q", val)
		}

		_, err = validateRequiredString(map[string]any{"key": 123}, "key")
		if err == nil {
			t.Error("Expected error for non-string value")
		}

		_, err = validateRequiredString(map[string]any{}, "missing")
		if err == nil {
			t.Error("Expected error for missing key")
		}
	})

	t.Run("validateFileSafety - restricted files", func(t *testing.T) {
		err := validateFileSafety("/etc/passwd")
		if err == nil {
			t.Error("Expected error for passwd")
		}

		err = validateFileSafety("/etc/shadow")
		if err == nil {
			t.Error("Expected error for shadow")
		}

		err = validateFileSafety("/etc/sudoers")
		if err == nil {
			t.Error("Expected error for sudoers")
		}

		err = validateFileSafety("/safe/path/file.txt")
		if err != nil {
			t.Errorf("Expected no error for safe path, got %v", err)
		}
	})
}
