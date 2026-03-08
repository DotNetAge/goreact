package core

import (
	"testing"

	"github.com/ray/goreact/pkg/tool"
	"github.com/ray/goreact/pkg/types"
)

func TestContext(t *testing.T) {
	// 创建上下文
	ctx := NewContext()

	// 测试设置和获取
	ctx.Set("key", "value")
	if value, ok := ctx.Get("key"); !ok || value != "value" {
		t.Errorf("Expected to get value, got %v (ok: %v)", value, ok)
	}

	// 测试获取不存在的键
	if _, ok := ctx.Get("nonexistent"); ok {
		t.Error("Expected key to not exist")
	}

	// 测试克隆
	cloned := ctx.Clone()
	if value, ok := cloned.Get("key"); !ok || value != "value" {
		t.Error("Expected cloned context to have the same data")
	}

	// 修改克隆的上下文不应影响原始上下文
	cloned.Set("key", "new_value")
	if value, ok := ctx.Get("key"); !ok || value != "value" {
		t.Error("Expected original context to remain unchanged")
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

	// 创建执行结果
	result := &types.ExecutionResult{
		Success: true,
		Output:  "Test output",
	}

	// 测试观察
	feedback, err := observer.Observe(result, ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !feedback.ShouldContinue {
		t.Error("Expected feedback to continue")
	}
}

func TestLoopController(t *testing.T) {
	// 创建循环控制器
	controller := NewDefaultLoopController(5)

	// 创建循环状态
	state := &types.LoopState{
		Iteration: 1,
		Task:      "Test task",
	}

	// 测试循环控制
	action := controller.Control(state)

	if !action.ShouldContinue {
		t.Error("Expected loop to continue")
	}

	// 测试达到最大迭代次数
	state.Iteration = 6
	action = controller.Control(state)

	if action.ShouldContinue {
		t.Error("Expected loop to stop after max iterations")
	}
}
