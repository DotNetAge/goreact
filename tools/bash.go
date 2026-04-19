package tools

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/DotNetAge/goreact/core"
)

// Bash Bash命令工具
type Bash struct {
	info *core.ToolInfo
}

// NewBash 创建Bash工具
func NewBash() core.FuncTool {
	return &Bash{
		&core.ToolInfo{
			Name:          "bash",
			Description:   "Bash命令执行工具，支持执行bash命令",
			SecurityLevel: core.LevelHighRisk,
		},
	}
}

func (b *Bash) Info() *core.ToolInfo {
	return b.info
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
