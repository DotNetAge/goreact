package builtin

import (
	"context"
	"testing"
)

func TestCalculator(t *testing.T) {
	// 创建计算器工具
	calculator := NewCalculator()

	// 测试加法
	_, err := calculator.Execute(context.Background(), map[string]interface{}{"operation": "add", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for add, got %v", err)
	}

	// 测试减法
	_, err = calculator.Execute(context.Background(), map[string]interface{}{"operation": "subtract", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for subtract, got %v", err)
	}

	// 测试乘法
	_, err = calculator.Execute(context.Background(), map[string]interface{}{"operation": "multiply", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for multiply, got %v", err)
	}

	// 测试除法
	_, err = calculator.Execute(context.Background(), map[string]interface{}{"operation": "divide", "a": 10, "b": 5})
	if err != nil {
		t.Errorf("Expected no error for divide, got %v", err)
	}

	// 测试除法错误（除数为0）
	_, err = calculator.Execute(context.Background(), map[string]interface{}{"operation": "divide", "a": 10, "b": 0})
	if err == nil {
		t.Error("Expected error for divide by zero")
	}

	// 测试未知操作
	_, err = calculator.Execute(context.Background(), map[string]interface{}{"operation": "unknown", "a": 10, "b": 5})
	if err == nil {
		t.Error("Expected error for unknown operation")
	}
}

func TestEcho(t *testing.T) {
	// 创建echo工具
	echo := NewEcho()

	// 测试echo
	result, err := echo.Execute(context.Background(), map[string]interface{}{"message": "Hello, World!"})
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
	result, err := datetime.Execute(context.Background(), map[string]interface{}{"operation": "now"})
	if err != nil {
		t.Errorf("Expected no error for now, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for now")
	}

	// 测试格式化时间 - 使用RFC3339格式
	result, err = datetime.Execute(context.Background(), map[string]interface{}{"operation": "format", "value": "2026-03-08T12:00:00Z", "format": "2006-01-02"})
	if err != nil {
		t.Errorf("Expected no error for format, got %v", err)
	}

	// 测试解析时间 - 使用RFC3339格式
	result, err = datetime.Execute(context.Background(), map[string]interface{}{"operation": "parse", "value": "2026-03-08T12:00:00Z", "format": "2006-01-02"})
	if err != nil {
		t.Errorf("Expected no error for parse, got %v", err)
	}

	// 测试未知操作
	_, err = datetime.Execute(context.Background(), map[string]interface{}{"operation": "unknown"})
	if err == nil {
		t.Error("Expected error for unknown operation")
	}
}

func TestHTTP(t *testing.T) {
	// 创建HTTP工具
	httpTool := NewHTTP()

	// 测试GET请求（使用一个可靠的测试端点）
	result, err := httpTool.Execute(context.Background(), map[string]interface{}{"method": "GET", "url": "https://httpbin.org/get"})
	if err != nil {
		t.Errorf("Expected no error for GET, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for GET")
	}

	// 测试未知方法 - HTTP工具可能对未知方法不返回错误，而是使用默认方法
	result, err = httpTool.Execute(context.Background(), map[string]interface{}{"method": "UNKNOWN", "url": "https://httpbin.org/get"})
	if err != nil {
		t.Errorf("Expected no error for unknown method, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for unknown method")
	}
}

func TestCurl(t *testing.T) {
	// 创建Curl工具
	curl := NewCurl()

	// 测试GET请求（使用一个可靠的测试端点）
	result, err := curl.Execute(context.Background(), map[string]interface{}{"method": "GET", "url": "https://httpbin.org/get"})
	if err != nil {
		t.Errorf("Expected no error for GET, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for GET")
	}

	// 测试POST请求
	result, err = curl.Execute(context.Background(), map[string]interface{}{"method": "POST", "url": "https://httpbin.org/post", "body": `{"test": "value"}`})
	if err != nil {
		t.Errorf("Expected no error for POST, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for POST")
	}
}

func TestPort(t *testing.T) {
	// 创建Port工具
	port := NewPort()

	// 测试一个不太可能被占用的端口
	result, err := port.Execute(context.Background(), map[string]interface{}{"port": 9999.0, "address": "localhost"})
	if err != nil {
		t.Errorf("Expected no error for port check, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for port check")
	}
}

func TestGrep(t *testing.T) {
	// 创建Grep工具
	grep := NewGrep()

	// 测试在当前文件中查找模式
	result, err := grep.Execute(context.Background(), map[string]interface{}{"pattern": "TestGrep", "path": "./builtin_test.go"})
	if err != nil {
		t.Errorf("Expected no error for grep, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result for grep")
	}
}
