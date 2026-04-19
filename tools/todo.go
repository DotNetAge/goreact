package tools

import (
	"context"

	"github.com/DotNetAge/goreact/core"
)

// TodoWriteTool implements a tool for managing a task list.
type TodoWriteTool struct{}

// NewTodoWriteTool 创建待办事项写入工具
func NewTodoWriteTool() core.FuncTool {
	return &TodoWriteTool{}
}

const todoDescription = `Create and manage a structured task list for your current coding session. This helps track progress, organize complex tasks, and demonstrate thoroughness.

Usage:
- Use proactively for complex multi-step tasks (3+ distinct steps) or when the user provides multiple tasks.
- When you START working on a task - mark it as in_progress (use merge=true).
- IMMEDIATELY after COMPLETING a task - mark it as completed (use merge=true).
- Always include all properties (id, status, content) for clarity.
- Status must be one of: pending, in_progress, completed, cancelled.
- NEVER INCLUDE THESE IN TODOS: linting; testing; searching or examining the codebase.`

func (t *TodoWriteTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "todo_write",
		Description: todoDescription,
		Parameters: []core.Parameter{
			{
				Name:        "todos",
				Type:        "string",
				Description: "JSON string of todo items array. Each item: {id, status, content}",
				Required:    true,
			},
			{
				Name:        "merge",
				Type:        "boolean",
				Description: "Whether to merge with existing todos.",
				Required:    true,
			},
		},
	}
}

func (t *TodoWriteTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	// In a real implementation, this would update a persistent state.
	return "Todo list updated successfully.", nil
}
