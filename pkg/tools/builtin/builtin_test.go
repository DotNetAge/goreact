package builtin

import (
	"context"
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
	// 创建Grep工具
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
