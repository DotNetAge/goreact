package condition

import (
	"testing"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

func TestMaxIteration(t *testing.T) {
	cond := MaxIteration(5)
	ctx := core.NewContext()

	// 未达到最大迭代
	state := &types.LoopState{Iteration: 3}
	stop, reason := cond.ShouldStop(state, ctx)
	if stop {
		t.Errorf("Expected not to stop at iteration 3, but got stop=true")
	}

	// 达到最大迭代
	state.Iteration = 5
	stop, reason = cond.ShouldStop(state, ctx)
	if !stop {
		t.Errorf("Expected to stop at iteration 5, but got stop=false")
	}
	if reason != "reached maximum iterations" {
		t.Errorf("Expected reason 'reached maximum iterations', got '%s'", reason)
	}

	// 超过最大迭代
	state.Iteration = 6
	stop, _ = cond.ShouldStop(state, ctx)
	if !stop {
		t.Errorf("Expected to stop at iteration 6, but got stop=false")
	}
}

func TestTimeout(t *testing.T) {
	cond := Timeout(100 * time.Millisecond)
	ctx := core.NewContext()
	state := &types.LoopState{Iteration: 1}

	// 未超时
	stop, _ := cond.ShouldStop(state, ctx)
	if stop {
		t.Errorf("Expected not to stop immediately, but got stop=true")
	}

	// 等待超时
	time.Sleep(150 * time.Millisecond)
	stop, reason := cond.ShouldStop(state, ctx)
	if !stop {
		t.Errorf("Expected to stop after timeout, but got stop=false")
	}
	if reason != "timeout exceeded" {
		t.Errorf("Expected reason 'timeout exceeded', got '%s'", reason)
	}
}

func TestTaskComplete(t *testing.T) {
	cond := TaskComplete()
	ctx := core.NewContext()

	// 任务未完成
	state := &types.LoopState{
		Iteration: 1,
		LastThought: &types.Thought{
			Reasoning:    "thinking",
			ShouldFinish: false,
		},
	}
	stop, _ := cond.ShouldStop(state, ctx)
	if stop {
		t.Errorf("Expected not to stop when task not complete, but got stop=true")
	}

	// 任务完成
	state.LastThought.ShouldFinish = true
	state.LastThought.FinalAnswer = "42"
	stop, reason := cond.ShouldStop(state, ctx)
	if !stop {
		t.Errorf("Expected to stop when task complete, but got stop=false")
	}
	if reason != "task completed" {
		t.Errorf("Expected reason 'task completed', got '%s'", reason)
	}

	// 没有 LastThought
	state.LastThought = nil
	stop, _ = cond.ShouldStop(state, ctx)
	if stop {
		t.Errorf("Expected not to stop when no LastThought, but got stop=true")
	}
}

func TestCompositeCondition(t *testing.T) {
	cond := NewComposite(
		MaxIteration(10),
		Timeout(1*time.Second),
		TaskComplete(),
	)
	ctx := core.NewContext()

	// 所有条件都不满足
	state := &types.LoopState{Iteration: 5}
	stop, _ := cond.ShouldStop(state, ctx)
	if stop {
		t.Errorf("Expected not to stop when no condition met, but got stop=true")
	}

	// 满足最大迭代条件
	state.Iteration = 10
	stop, reason := cond.ShouldStop(state, ctx)
	if !stop {
		t.Errorf("Expected to stop at max iteration, but got stop=false")
	}
	if reason != "reached maximum iterations" {
		t.Errorf("Expected reason 'reached maximum iterations', got '%s'", reason)
	}

	// 满足任务完成条件
	state = &types.LoopState{
		Iteration: 3,
		LastThought: &types.Thought{
			ShouldFinish: true,
			FinalAnswer:  "done",
		},
	}
	stop, reason = cond.ShouldStop(state, ctx)
	if !stop {
		t.Errorf("Expected to stop when task complete, but got stop=false")
	}
	if reason != "task completed" {
		t.Errorf("Expected reason 'task completed', got '%s'", reason)
	}
}

func TestCompositeConditionOrder(t *testing.T) {
	// 测试条件检查顺序（第一个满足的条件会返回）
	cond := NewComposite(
		MaxIteration(5),
		TaskComplete(),
	)
	ctx := core.NewContext()

	// 同时满足两个条件，应该返回第一个
	state := &types.LoopState{
		Iteration: 5,
		LastThought: &types.Thought{
			ShouldFinish: true,
			FinalAnswer:  "done",
		},
	}
	stop, reason := cond.ShouldStop(state, ctx)
	if !stop {
		t.Errorf("Expected to stop, but got stop=false")
	}
	if reason != "reached maximum iterations" {
		t.Errorf("Expected first condition reason, got '%s'", reason)
	}
}
