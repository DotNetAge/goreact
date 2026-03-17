package terminator

import (
	"fmt"

	"github.com/ray/goreact/pkg/types"
)

// DefaultTerminator 默认循环终结器实现
type DefaultTerminator struct {
	maxIterations int
}

// NewDefaultTerminator 创建默认循环终结器
func NewDefaultTerminator(maxIterations int) *DefaultTerminator {
	return &DefaultTerminator{
		maxIterations: maxIterations,
	}
}

// Control 控制循环
func (c *DefaultTerminator) Control(state *types.LoopState) *types.LoopAction {
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
