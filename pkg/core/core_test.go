package core

import (
	"testing"

	"github.com/ray/goreact/pkg/llm/mock"
	"github.com/ray/goreact/pkg/tool"
	"github.com/ray/goreact/pkg/types"
)

// mockAgentManager 模拟AgentManager实现（已废弃）

func TestContext(t *testing.T) {
	// 创建上下文
	ctx := NewContext()

	// 测试设置和获取
	ctx.Set("key", "value")
	if value, ok := ctx.Get("key"); !ok || value != "value" {
		t.Errorf("Expected to get value, got %v (ok: %v)", value, ok)
	}

	// 测试不存在的键
	if _, ok := ctx.Get("non_existent"); ok {
		t.Error("Expected non_existent key to not exist")
	}
}

func TestLoopController(t *testing.T) {
	// 创建循环控制器
	controller := NewDefaultLoopController(3)

	// 测试循环控制
	state := &types.LoopState{
		Iteration: 1,
	}

	action := controller.Control(state)
	if !action.ShouldContinue {
		t.Error("Expected to continue for iteration 1")
	}

	state.Iteration = 3
	action = controller.Control(state)
	if action.ShouldContinue {
		t.Error("Expected to stop for iteration 3")
	}
}

func TestThinker(t *testing.T) {
	// 创建mock LLM客户端
	mockResponses := []string{
		"Thought: I need to test\nFinal Answer: Test response",
	}
	llmClient := mock.NewMockClient(mockResponses)

	// 创建工具管理器
	toolManager := tool.NewManager()

	// 创建思考器
	thinker := NewDefaultThinker(llmClient, toolManager.GetToolDescriptions())

	// 创建上下文
	ctx := NewContext()

	// 测试思考
	thought, err := thinker.Think("Test task", ctx)
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

func TestActor(t *testing.T) {
	// 创建工具管理器
	toolManager := tool.NewManager()

	// 创建行动者
	actor := NewDefaultActor(toolManager)

	// 创建上下文
	ctx := NewContext()

	// 创建动作
	action := &types.Action{
		ToolName:   "test_tool",
		Parameters: map[string]interface{}{"key": "value"},
	}

	// 测试行动
	result, err := actor.Act(action, ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.Success {
		t.Error("Expected action to fail for non-existent tool")
	}
}

func TestObserver(t *testing.T) {
	// 创建观察者
	observer := NewDefaultObserver()

	// 创建上下文
	ctx := NewContext()

	// 测试成功的执行结果
	successResult := &types.ExecutionResult{
		Success: true,
		Output:  "Success output",
	}

	feedback, err := observer.Observe(successResult, ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !feedback.ShouldContinue {
		t.Error("Expected to continue for success")
	}

	// 测试失败的执行结果
	failResult := &types.ExecutionResult{
		Success: false,
		Error:   nil,
	}

	feedback, err = observer.Observe(failResult, ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !feedback.ShouldContinue {
		t.Error("Expected to continue for failure")
	}
}
