package core

import (
	"fmt"

	"github.com/ray/goreact/pkg/types"
)

// Observer 观察模块接口
type Observer interface {
	Observe(result *types.ExecutionResult, context *Context) (*types.Feedback, error)
}

// DefaultObserver 默认观察模块实现
type DefaultObserver struct{}

// NewDefaultObserver 创建默认观察模块
func NewDefaultObserver() *DefaultObserver {
	return &DefaultObserver{}
}

// Observe 观察执行结果
func (o *DefaultObserver) Observe(result *types.ExecutionResult, context *Context) (*types.Feedback, error) {
	feedback := &types.Feedback{
		Metadata: make(map[string]interface{}),
	}

	if result.Success {
		feedback.ShouldContinue = true
		feedback.Message = fmt.Sprintf("Tool executed successfully. Result: %v", result.Output)
	} else {
		feedback.ShouldContinue = true
		feedback.Message = fmt.Sprintf("Tool execution failed: %v. Please try a different approach.", result.Error)
	}

	// 更新历史记录
	o.updateHistory(context, feedback.Message)

	return feedback, nil
}

// updateHistory 更新历史记录
func (o *DefaultObserver) updateHistory(context *Context, message string) {
	history := ""
	if h, ok := context.Get("history"); ok {
		if historyStr, ok := h.(string); ok {
			history = historyStr
		}
	}

	history += message + "\n"
	context.Set("history", history)
}
