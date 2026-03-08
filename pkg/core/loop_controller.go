package core

import (
	"fmt"

	"github.com/ray/goreact/pkg/types"
)

// LoopController 循环控制器接口
type LoopController interface {
	Control(state *types.LoopState) *types.LoopAction
}

// DefaultLoopController 默认循环控制器实现
type DefaultLoopController struct {
	maxIterations int
}

// NewDefaultLoopController 创建默认循环控制器
func NewDefaultLoopController(maxIterations int) *DefaultLoopController {
	return &DefaultLoopController{
		maxIterations: maxIterations,
	}
}

// Control 控制循环
func (c *DefaultLoopController) Control(state *types.LoopState) *types.LoopAction {
	// 检查是否达到最大迭代次数
	if state.Iteration >= c.maxIterations {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         fmt.Sprintf("Reached maximum iterations (%d)", c.maxIterations),
		}
	}

	// 检查是否应该结束（基于思考结果）
	if state.LastThought != nil && state.LastThought.ShouldFinish {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "Task completed",
		}
	}

	// 检查反馈是否建议停止
	if state.LastFeedback != nil && !state.LastFeedback.ShouldContinue {
		return &types.LoopAction{
			ShouldContinue: false,
			Reason:         "Feedback suggests stopping",
		}
	}

	// 继续循环
	return &types.LoopAction{
		ShouldContinue: true,
		Reason:         "Continue processing",
	}
}
