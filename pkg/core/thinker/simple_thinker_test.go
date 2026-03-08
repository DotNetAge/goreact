package thinker_test

import (
	"testing"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/llm/mock"
	"github.com/ray/goreact/pkg/tool"
)

func TestSimpleThinker(t *testing.T) {
	// 创建 LLM 客户端
	mockResponses := []string{
		"Thought: I need to test\nFinal Answer: Test response",
	}
	llmClient := mock.NewMockClient(mockResponses)

	// 创建工具管理器
	toolManager := tool.NewManager()

	// 创建思考器
	thinkerImpl := thinker.NewSimpleThinker(llmClient, toolManager.GetToolDescriptions())

	// 创建上下文
	ctx := core.NewContext()

	// 测试思考
	thought, err := thinkerImpl.Think("Test task", ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !thought.ShouldFinish {
		t.Error("Expected thought to finish")
	}

	if thought.FinalAnswer != "Test response" {
		t.Errorf("Expected final answer 'Test response', got '%s'", thought.FinalAnswer)
	}
}
