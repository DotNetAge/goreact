package builtin

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/pkg/tools"
)

// Echo 回显工具
type Echo struct{}

// NewEcho 创建回显工具
func NewEcho() tools.Tool {
	return &Echo{}
}

// Name 返回工具名称
func (e *Echo) Name() string {
	return "echo"
}

// Description 返回工具描述
func (e *Echo) Description() string {
	return "Echoes back the input message. Useful for testing and debugging. Params: {message: string}"
}

// Execute 执行回显操作
// SecurityLevel returns the tool's security risk level
func (t *Echo) SecurityLevel() tools.SecurityLevel {
	return tools.LevelSafe // Default, needs manual update for risky tools
}

func (e *Echo) Execute(ctx context.Context, params map[string]any) (any, error) {
	message, ok := params["message"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message' parameter")
	}

	// 简单回显消息
	return "Echo: " + message, nil
}
