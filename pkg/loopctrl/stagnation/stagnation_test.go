package stagnation

import (
	"testing"

	"github.com/ray/goreact/pkg/types"
)

func TestDetectorNoAction(t *testing.T) {
	detector := NewDetector(WithNoProgressLimit(3))

	// 第 1 次无行动
	state := &types.LoopState{
		Iteration: 1,
		LastThought: &types.Thought{
			Reasoning: "thinking",
			Action:    nil, // 无行动
		},
	}
	result := detector.Check(state)
	if result.IsStagnant {
		t.Errorf("Expected not stagnant after 1 no-action, but got stagnant=true")
	}

	// 第 2 次无行动
	state.Iteration = 2
	result = detector.Check(state)
	if result.IsStagnant {
		t.Errorf("Expected not stagnant after 2 no-actions, but got stagnant=true")
	}

	// 第 3 次无行动 - 应该检测到停滞
	state.Iteration = 3
	result = detector.Check(state)
	if !result.IsStagnant {
		t.Errorf("Expected stagnant after 3 no-actions, but got stagnant=false")
	}
	if result.Type != "no_action" {
		t.Errorf("Expected type 'no_action', got '%s'", result.Type)
	}
	if result.Suggestion == "" {
		t.Errorf("Expected non-empty suggestion")
	}
}

func TestDetectorRepeatedFailure(t *testing.T) {
	detector := NewDetector(WithRepeatedFailureLimit(2))

	action := &types.Action{
		ToolName:   "http",
		Parameters: map[string]any{"url": "example.com"},
	}

	// 第 1 次失败
	state := &types.LoopState{
		Iteration:   1,
		LastThought: &types.Thought{Action: action},
		LastResult:  &types.ExecutionResult{Success: false},
	}
	result := detector.Check(state)
	if result.IsStagnant {
		t.Errorf("Expected not stagnant after 1 failure, but got stagnant=true")
	}

	// 第 2 次相同失败 - 应该检测到停滞
	state.Iteration = 2
	result = detector.Check(state)
	if !result.IsStagnant {
		t.Errorf("Expected stagnant after 2 repeated failures, but got stagnant=false")
	}
	if result.Type != "repeated_failure" {
		t.Errorf("Expected type 'repeated_failure', got '%s'", result.Type)
	}
}

func TestDetectorResetOnSuccess(t *testing.T) {
	detector := NewDetector(WithRepeatedFailureLimit(2))

	action := &types.Action{
		ToolName:   "http",
		Parameters: map[string]any{"url": "example.com"},
	}

	// 第 1 次失败
	state := &types.LoopState{
		Iteration:   1,
		LastThought: &types.Thought{Action: action},
		LastResult:  &types.ExecutionResult{Success: false},
	}
	detector.Check(state)

	// 成功 - 应该重置计数
	state.Iteration = 2
	state.LastResult = &types.ExecutionResult{Success: true}
	detector.Check(state)

	// 再次失败 - 不应该检测到停滞（因为已重置）
	state.Iteration = 3
	state.LastResult = &types.ExecutionResult{Success: false}
	result := detector.Check(state)
	if result.IsStagnant {
		t.Errorf("Expected not stagnant after reset, but got stagnant=true")
	}
}

func TestDetectorResetOnDifferentAction(t *testing.T) {
	detector := NewDetector(WithRepeatedFailureLimit(2))

	action1 := &types.Action{
		ToolName:   "http",
		Parameters: map[string]any{"url": "example.com"},
	}
	action2 := &types.Action{
		ToolName:   "calculator",
		Parameters: map[string]any{"op": "add"},
	}

	// 第 1 次失败（action1）
	state := &types.LoopState{
		Iteration:   1,
		LastThought: &types.Thought{Action: action1},
		LastResult:  &types.ExecutionResult{Success: false},
	}
	detector.Check(state)

	// 第 2 次失败（action2，不同的工具）- 应该重置计数
	state.Iteration = 2
	state.LastThought = &types.Thought{Action: action2}
	result := detector.Check(state)
	if result.IsStagnant {
		t.Errorf("Expected not stagnant with different action, but got stagnant=true")
	}
}

func TestDetectorNoActionResetOnAction(t *testing.T) {
	detector := NewDetector(WithNoProgressLimit(3))

	// 第 1 次无行动
	state := &types.LoopState{
		Iteration:   1,
		LastThought: &types.Thought{Action: nil},
	}
	detector.Check(state)

	// 第 2 次无行动
	state.Iteration = 2
	detector.Check(state)

	// 有行动 - 应该重置计数
	state.Iteration = 3
	state.LastThought = &types.Thought{
		Action: &types.Action{ToolName: "http"},
	}
	state.LastResult = &types.ExecutionResult{Success: true}
	detector.Check(state)

	// 再次无行动 - 不应该检测到停滞（因为已重置）
	state.Iteration = 4
	state.LastThought = &types.Thought{Action: nil}
	result := detector.Check(state)
	if result.IsStagnant {
		t.Errorf("Expected not stagnant after reset, but got stagnant=true")
	}
}

func TestDetectorReset(t *testing.T) {
	detector := NewDetector(WithNoProgressLimit(2))

	// 第 1 次无行动
	state := &types.LoopState{
		Iteration:   1,
		LastThought: &types.Thought{Action: nil},
	}
	detector.Check(state)

	// 第 2 次无行动 - 应该检测到停滞
	state.Iteration = 2
	result := detector.Check(state)
	if !result.IsStagnant {
		t.Errorf("Expected stagnant, but got stagnant=false")
	}

	// 重置检测器
	detector.Reset()

	// 再次无行动 - 不应该检测到停滞（因为已重置）
	state.Iteration = 3
	result = detector.Check(state)
	if result.IsStagnant {
		t.Errorf("Expected not stagnant after Reset(), but got stagnant=true")
	}
}

func TestDetectorCustomLimits(t *testing.T) {
	// 测试自定义限制
	detector := NewDetector(
		WithNoProgressLimit(5),
		WithRepeatedFailureLimit(3),
	)

	// 测试无行动限制
	state := &types.LoopState{
		Iteration:   1,
		LastThought: &types.Thought{Action: nil},
	}
	for i := 1; i < 5; i++ {
		state.Iteration = i
		result := detector.Check(state)
		if result.IsStagnant {
			t.Errorf("Expected not stagnant at iteration %d, but got stagnant=true", i)
		}
	}

	state.Iteration = 5
	result := detector.Check(state)
	if !result.IsStagnant {
		t.Errorf("Expected stagnant at iteration 5, but got stagnant=false")
	}
}
