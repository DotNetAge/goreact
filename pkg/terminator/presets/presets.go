package presets

import (
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/terminator/condition"
	"github.com/ray/goreact/pkg/terminator/cost"
	"github.com/ray/goreact/pkg/terminator/stagnation"
	"github.com/ray/goreact/pkg/types"
)

// SmartController 智能控制器
type SmartController struct {
	condition condition.StopCondition
	detector  *stagnation.Detector
}

// NewSmartController 创建智能控制器
func NewSmartController() *SmartController {
	return &SmartController{
		condition: condition.NewComposite(
			condition.MaxIteration(20),
			condition.TaskComplete(),
		),
		detector: stagnation.NewDetector(
			stagnation.WithNoProgressLimit(3),
			stagnation.WithRepeatedFailureLimit(2),
		),
	}
}

// Control 控制循环
func (c *SmartController) Control(state *types.LoopState) *types.LoopAction {
	// 检查停止条件
	if stop, reason := c.condition.ShouldStop(state, core.NewContext()); stop {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         reason,
		}
	}

	// 检查停滞
	result := c.detector.Check(state)
	if result.IsStagnant {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "stagnation detected: " + result.Type,
		}
	}

	return &types.LoopAction{ShouldContinue: true}
}

// BudgetController 预算控制器
type BudgetController struct {
	condition condition.StopCondition
	detector  *stagnation.Detector
	tracker   *cost.Tracker
	budget    float64
}

// NewBudgetController 创建预算控制器
func NewBudgetController(budget float64) *BudgetController {
	return &BudgetController{
		condition: condition.NewComposite(
			condition.MaxIteration(20),
			condition.TaskComplete(),
		),
		detector: stagnation.NewDetector(
			stagnation.WithNoProgressLimit(3),
			stagnation.WithRepeatedFailureLimit(2),
		),
		tracker: cost.NewTracker(cost.Pricing{
			InputTokenPrice:  0.01,
			OutputTokenPrice: 0.03,
		}),
		budget: budget,
	}
}

// Control 控制循环
func (c *BudgetController) Control(state *types.LoopState) *types.LoopAction {
	// 检查预算
	if c.tracker.ExceedsLimit(c.budget) {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "budget exceeded",
		}
	}

	// 检查停止条件
	if stop, reason := c.condition.ShouldStop(state, core.NewContext()); stop {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         reason,
		}
	}

	// 检查停滞
	result := c.detector.Check(state)
	if result.IsStagnant {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "stagnation detected: " + result.Type,
		}
	}

	return &types.LoopAction{ShouldContinue: true}
}

// RecordTokens 记录 token 使用量
func (c *BudgetController) RecordTokens(inputTokens, outputTokens int) {
	c.tracker.RecordTokens(inputTokens, outputTokens)
}

// TimedController 时间控制器
type TimedController struct {
	condition condition.StopCondition
	detector  *stagnation.Detector
}

// NewTimedController 创建时间控制器
func NewTimedController(timeout time.Duration) *TimedController {
	return &TimedController{
		condition: condition.NewComposite(
			condition.MaxIteration(20),
			condition.Timeout(timeout),
			condition.TaskComplete(),
		),
		detector: stagnation.NewDetector(
			stagnation.WithNoProgressLimit(3),
			stagnation.WithRepeatedFailureLimit(2),
		),
	}
}

// Control 控制循环
func (c *TimedController) Control(state *types.LoopState) *types.LoopAction {
	// 检查停止条件
	if stop, reason := c.condition.ShouldStop(state, core.NewContext()); stop {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         reason,
		}
	}

	// 检查停滞
	result := c.detector.Check(state)
	if result.IsStagnant {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "stagnation detected: " + result.Type,
		}
	}

	return &types.LoopAction{ShouldContinue: true}
}

// ProductionController 生产控制器（全部最佳实践）
type ProductionController struct {
	condition condition.StopCondition
	detector  *stagnation.Detector
	tracker   *cost.Tracker
	budget    float64
}

// NewProductionController 创建生产控制器
func NewProductionController() *ProductionController {
	return &ProductionController{
		condition: condition.NewComposite(
			condition.MaxIteration(30),
			condition.Timeout(10*time.Minute),
			condition.TaskComplete(),
		),
		detector: stagnation.NewDetector(
			stagnation.WithNoProgressLimit(3),
			stagnation.WithRepeatedFailureLimit(2),
		),
		tracker: cost.NewTracker(cost.Pricing{
			InputTokenPrice:  0.01,
			OutputTokenPrice: 0.03,
		}),
		budget: 1.00, // 默认 $1.00/任务
	}
}

// Control 控制循环
func (c *ProductionController) Control(state *types.LoopState) *types.LoopAction {
	// 检查预算
	if c.tracker.ExceedsLimit(c.budget) {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "budget exceeded",
		}
	}

	// 检查停止条件
	if stop, reason := c.condition.ShouldStop(state, core.NewContext()); stop {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         reason,
		}
	}

	// 检查停滞
	result := c.detector.Check(state)
	if result.IsStagnant {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "stagnation detected: " + result.Type,
		}
	}

	return &types.LoopAction{ShouldContinue: true}
}

// RecordTokens 记录 token 使用量
func (c *ProductionController) RecordTokens(inputTokens, outputTokens int) {
	c.tracker.RecordTokens(inputTokens, outputTokens)
}
