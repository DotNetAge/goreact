package builtin

import (
	"fmt"
	"github.com/ray/goreact/pkg/tool"
)

// Echo 回显工具（用于测试）
type Echo struct{}

// NewEcho 创建回显工具
func NewEcho() tool.Tool {
	return &Echo{}
}

// Name 返回工具名称
func (e *Echo) Name() string {
	return "echo"
}

// Description 返回工具描述
func (e *Echo) Description() string {
	return "Echoes back the input message. Params: {message: string}"
}

// Execute 执行回显
func (e *Echo) Execute(params map[string]interface{}) (interface{}, error) {
	message, ok := params["message"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message' parameter")
	}
	return fmt.Sprintf("Echo: %s", message), nil
}
