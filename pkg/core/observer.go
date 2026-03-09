package core

import (
	"fmt"
	"strings"

	"github.com/ray/goreact/pkg/types"
)

const (
	DefaultMaxHistorySize = 10000 // 最大历史字符数
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
	// 添加 nil 检查
	if result == nil {
		return nil, fmt.Errorf("execution result cannot be nil")
	}
	if context == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

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

// updateHistory 更新历史记录（限制大小防止无界增长）
func (o *DefaultObserver) updateHistory(context *Context, message string) {
	history := ""
	if h, ok := context.Get("history"); ok {
		if historyStr, ok := h.(string); ok {
			history = historyStr
		}
	}

	history += message + "\n"

	// 如果历史超过限制，保留最新的部分
	if len(history) > DefaultMaxHistorySize {
		history = history[len(history)-DefaultMaxHistorySize:]
		// 找到第一个换行符，从完整行开始
		if idx := strings.Index(history, "\n"); idx != -1 {
			history = history[idx+1:]
		}
	}

	context.Set("history", history)
}
