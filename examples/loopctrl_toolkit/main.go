package main

import (
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/terminator/condition"
	"github.com/ray/goreact/pkg/terminator/cost"
	looppresets "github.com/ray/goreact/pkg/terminator/presets"
	"github.com/ray/goreact/pkg/terminator/stagnation"
	"github.com/ray/goreact/pkg/types"
)

func main() {
	fmt.Println("=== LoopController Toolkit 示例 ===")

	// ============================================================
	// 1. Composite Stop Conditions
	// ============================================================
	fmt.Println("\n--- 1. Composite Stop Conditions ---")

	cond := condition.NewComposite(
		condition.MaxIteration(5),
		condition.Timeout(10*time.Second),
	)

	// 正常继续
	state := &types.LoopState{Iteration: 2, Task: "test"}
	stop, reason := cond.ShouldStop(state, core.NewContext())
	fmt.Printf("迭代 2: stop=%v, reason=%s\n", stop, reason)

	// 达到最大迭代
	state.Iteration = 5
	stop, reason = cond.ShouldStop(state, core.NewContext())
	fmt.Printf("迭代 5: stop=%v, reason=%s\n", stop, reason)

	// 任务完成
	state = &types.LoopState{
		Iteration:   2,
		LastThought: &types.Thought{ShouldFinish: true, FinalAnswer: "42"},
	}
	cond2 := condition.NewComposite(
		condition.MaxIteration(10),
		condition.TaskComplete(),
	)
	stop, reason = cond2.ShouldStop(state, core.NewContext())
	fmt.Printf("任务完成: stop=%v, reason=%s\n", stop, reason)

	// ============================================================
	// 2. Stagnation Detector
	// ============================================================
	fmt.Println("\n--- 2. Stagnation Detector ---")

	detector := stagnation.NewDetector(
		stagnation.WithNoProgressLimit(3),
	)

	// 模拟连续无行动
	for i := 1; i <= 4; i++ {
		result := detector.Check(&types.LoopState{
			Iteration:   i,
			LastThought: &types.Thought{Reasoning: "Let me think..."},
			// 没有 Action = 无行动
		})
		fmt.Printf("迭代 %d: stagnant=%v", i, result.IsStagnant)
		if result.IsStagnant {
			fmt.Printf(", type=%s, suggestion=%s", result.Type, result.Suggestion)
		}
		fmt.Println()
	}

	// 模拟重复失败
	fmt.Println()
	detector2 := stagnation.NewDetector(stagnation.WithRepeatedFailureLimit(2))
	for i := 1; i <= 3; i++ {
		result := detector2.Check(&types.LoopState{
			Iteration: i,
			LastThought: &types.Thought{
				Action: &types.Action{ToolName: "http", Parameters: map[string]any{"url": "a"}},
			},
			LastResult: &types.ExecutionResult{Success: false, Error: fmt.Errorf("timeout")},
		})
		fmt.Printf("失败 %d: stagnant=%v", i, result.IsStagnant)
		if result.IsStagnant {
			fmt.Printf(", type=%s", result.Type)
		}
		fmt.Println()
	}

	// ============================================================
	// 3. Cost Tracker
	// ============================================================
	fmt.Println("\n--- 3. Cost Tracker ---")

	tracker := cost.NewTracker(cost.Pricing{
		InputTokenPrice:  0.01, // $0.01 / 1K tokens
		OutputTokenPrice: 0.03, // $0.03 / 1K tokens
	})

	tracker.RecordTokens(500, 200)
	tracker.RecordTokens(800, 300)
	tracker.RecordTokens(600, 250)

	fmt.Printf("总成本: $%.4f\n", tracker.TotalCost())
	fmt.Printf("总输入 tokens: %d\n", tracker.TotalInputTokens())
	fmt.Printf("总输出 tokens: %d\n", tracker.TotalOutputTokens())
	fmt.Printf("超过 $0.05 限制: %v\n", tracker.ExceedsLimit(0.05))
	fmt.Printf("超过 $0.01 限制: %v\n", tracker.ExceedsLimit(0.01))
	fmt.Println(tracker.Report())

	// ============================================================
	// 4. LoopController Presets
	// ============================================================
	fmt.Println("--- 4. LoopController Presets ---")

	// SmartController
	smart := looppresets.NewSmartController()
	action := smart.Control(&types.LoopState{Iteration: 2, Task: "test"})
	fmt.Printf("SmartController 迭代 2: continue=%v\n", action.ShouldContinue)

	action = smart.Control(&types.LoopState{Iteration: 20, Task: "test"})
	fmt.Printf("SmartController 迭代 20: continue=%v, reason=%s\n", action.ShouldContinue, action.Reason)

	// BudgetController
	budget := looppresets.NewBudgetController(0.10)
	action = budget.Control(&types.LoopState{Iteration: 1, Task: "test"})
	fmt.Printf("BudgetController 迭代 1: continue=%v\n", action.ShouldContinue)

	// TimedController
	timed := looppresets.NewTimedController(5 * time.Minute)
	action = timed.Control(&types.LoopState{Iteration: 1, Task: "test"})
	fmt.Printf("TimedController 迭代 1: continue=%v\n", action.ShouldContinue)

	fmt.Println("\n=== 示例完成 ===")
}
