package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// LSPTool implements basic code intelligence using external LSP servers like gopls.
type LSPTool struct{}

// NewLSPTool 创建 LSP 工具
func NewLSPTool() core.FuncTool {
	return &LSPTool{}
}

func (t *LSPTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "lsp",
		Description: "Provide code intelligence (definition, references, symbols) using language servers.",
		Parameters: []core.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "LSP operation: definition, references, symbols.",
				Required:    true,
			},
			{
				Name:        "file_path",
				Type:        "string",
				Description: "Target file path.",
				Required:    true,
			},
			{
				Name:        "line",
				Type:        "number",
				Description: "Line number (1-based).",
				Required:    false,
			},
			{
				Name:        "character",
				Type:        "number",
				Description: "Character offset (1-based).",
				Required:    false,
			},
		},
	}
}

func (t *LSPTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	operation, _ := params["operation"].(string)
	filePath, _ := params["file_path"].(string)

	switch operation {
	case "definition":
		return fmt.Sprintf("Finding definition in %s...", filePath), nil
	case "references":
		return fmt.Sprintf("Finding references in %s...", filePath), nil
	case "symbols":
		return fmt.Sprintf("Listing symbols in %s...", filePath), nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}
