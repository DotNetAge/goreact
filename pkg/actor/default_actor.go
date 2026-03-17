package actor

import (
	"fmt"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/tools"
	"github.com/ray/goreact/pkg/types"
)

// DefaultActor 默认行动模块实现
type DefaultActor struct {
	toolManager *tools.Manager
}

// NewDefaultActor 创建默认行动模块
func NewDefaultActor(toolManager *tools.Manager) *DefaultActor {
	return &DefaultActor{
		toolManager: toolManager,
	}
}

// Act 执行动作
func (a *DefaultActor) Act(action *types.Action, context *core.Context) (*types.ExecutionResult, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}

	// 执行工具
	output, err := a.toolManager.ExecuteTool(action.ToolName, action.Parameters)

	result := &types.ExecutionResult{
		Success:  err == nil,
		Output:   output,
		Metadata: make(map[string]interface{}),
	}

	// 记录执行信息
	result.Metadata["tool_name"] = action.ToolName
	result.Metadata["parameters"] = action.Parameters

	// 优化错误消息
	if err != nil {
		result.Error = fmt.Errorf("tool execution failed: tool=%s, params=%v, error=%w", action.ToolName, action.Parameters, err)
	}

	return result, nil
}
