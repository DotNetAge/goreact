package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DotNetAge/goreact/core"
)

// REPLTool implements a tool for executing Go code snippets.
type REPLTool struct{}

// NewREPLTool 创建 REPL 工具
func NewREPLTool() core.FuncTool {
	return &REPLTool{}
}

func (t *REPLTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "repl",
		Description: "Execute a Go code snippet and return the output. Useful for verifying logic or data structures.",
		Parameters: []core.Parameter{
			{
				Name:        "code",
				Type:        "string",
				Description: "The Go code to execute (should be a complete main package).",
				Required:    true,
			},
		},
	}
}

func (t *REPLTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	code, ok := params["code"].(string)
	if !ok {
		return nil, fmt.Errorf("missing code parameter")
	}

	tmpDir, err := os.MkdirTemp("", "repl-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "go", "run", tmpFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}
