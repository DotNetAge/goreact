package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// Echo 回显工具
type Echo struct {
	info *core.ToolInfo
}

// NewEcho 创建回显工具
func NewEcho() core.FuncTool {
	return &Echo{
		info: &core.ToolInfo{
			Name:          "echo",
			Description:   "Echoes back the input message. Useful for testing and debugging. Params: {message: string}",
			SecurityLevel: core.LevelSafe,
		},
	}
}

func (e *Echo) Info() *core.ToolInfo {
	return e.info
}

func (e *Echo) Execute(ctx context.Context, params map[string]any) (any, error) {
	message, ok := params["message"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message' parameter")
	}

	// 简单回显消息
	return "Echo: " + message, nil
}
