package mock

import (
	"testing"
)

func TestMockClient(t *testing.T) {
	// 创建mock客户端
	responses := []string{
		"First response",
		"Second response",
	}
	client := NewMockClient(responses)

	// 测试生成
	response1, err := client.Generate("Prompt 1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response1 != "First response" {
		t.Errorf("Expected 'First response', got '%s'", response1)
	}

	// 测试第二次生成
	response2, err := client.Generate("Prompt 2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response2 != "Second response" {
		t.Errorf("Expected 'Second response', got '%s'", response2)
	}

	// 测试超出响应数量
	response3, err := client.Generate("Prompt 3")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response3 == "" {
		t.Error("Expected non-empty response for fallback")
	}
}
