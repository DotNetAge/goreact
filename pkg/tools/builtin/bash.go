package builtin

import (
	"context"
	"fmt"
	"github.com/ray/goreact/pkg/tools"
	"os/exec"
)

// Bash Bash命令工具
type Bash struct{}

// NewBash 创建Bash工具
func NewBash() *Bash {
	return &Bash{}
}

// Name 返回工具名称
func (b *Bash) Name() string {
	return "bash"
}

// Description 返回工具描述
func (b *Bash) Description() string {
	return "Bash命令执行工具，支持执行bash命令"
}

// Execute 执行bash命令
// SecurityLevel returns the tool's security risk level
func (t *Bash) SecurityLevel() tools.SecurityLevel {
	return tools.LevelHighRisk // Default, needs manual update for risky tools
}

func (b *Bash) Execute(ctx context.Context, params map[string]any) (any, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'command' parameter")
	}

	// 执行命令
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()

	result := map[string]any{
		"output":  string(output),
		"success": err == nil,
	}

	if err != nil {
		result["error"] = err.Error()
	}

	return result, nil
}
