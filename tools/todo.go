package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// TodoItem represents a single todo item with priority and execution metadata.
type TodoItem struct {
	ID            string   `json:"id"`
	Status        string   `json:"status"`        // pending, in_progress, completed, cancelled
	Content       string   `json:"content"`        // Task description
	Priority      int      `json:"priority"`       // Lower number = higher priority (default: 0)
	Dependencies  []string `json:"dependencies"`   // IDs of items that must complete first
	ToolCall      string   `json:"tool_call"`      // Suggested tool to execute (e.g. "task_create")
	ToolParams    string   `json:"tool_params"`    // JSON string of tool parameters
	AssignedAgent string   `json:"assigned_agent"` // Agent ID if delegated
	CreatedAt     int64    `json:"created_at"`
	UpdatedAt     int64    `json:"updated_at"`
}

// todoStore is a package-level in-memory store for todo items, safe for concurrent access.
var (
	todoStore   []TodoItem
	todoStoreMu sync.RWMutex
	todoCounter int
)

func nextTodoID() string {
	todoCounter++
	return fmt.Sprintf("todo_%d", todoCounter)
}

// TodoWriteTool implements a tool for managing a task list.
type TodoWriteTool struct{}

// NewTodoWriteTool creates a todo write tool.
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
- NEVER INCLUDE THESE IN TODOS: linting; testing; searching or examining the codebase.
- Use priority (lower = higher priority) and dependencies to define execution order.
- Use tool_call and tool_params to pre-configure how each todo should be executed.`

func (t *TodoWriteTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "todo_write",
		Description: todoDescription,
		Parameters: []core.Parameter{
			{
				Name:        "todos",
				Type:        "string",
				Description: "JSON string of todo items array. Each item: {id, status, content, priority, dependencies, tool_call, tool_params}",
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
	todosStr, ok := params["todos"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'todos' parameter: expected JSON string")
	}

	var newItems []TodoItem
	if err := json.Unmarshal([]byte(todosStr), &newItems); err != nil {
		return nil, fmt.Errorf("failed to parse todos JSON: %w", err)
	}

	merge, _ := params["merge"].(bool)
	now := time.Now().Unix()

	todoStoreMu.Lock()
	defer todoStoreMu.Unlock()

	if merge {
		for i := range newItems {
			if newItems[i].ID == "" {
				newItems[i].ID = nextTodoID()
			}
			if newItems[i].CreatedAt == 0 {
				newItems[i].CreatedAt = now
			}
			newItems[i].UpdatedAt = now

			found := false
			for j, existing := range todoStore {
				if existing.ID == newItems[i].ID {
					todoStore[j] = newItems[i]
					found = true
					break
				}
			}
			if !found {
				todoStore = append(todoStore, newItems[i])
			}
		}
	} else {
		for i := range newItems {
			if newItems[i].ID == "" {
				newItems[i].ID = nextTodoID()
			}
			if newItems[i].CreatedAt == 0 {
				newItems[i].CreatedAt = now
			}
			newItems[i].UpdatedAt = now
		}
		todoStore = newItems
	}

	return map[string]any{
		"success": true,
		"count":   len(todoStore),
		"summary": formatTodoSummary(todoStore),
		"message": fmt.Sprintf("Todo list updated. %d item(s) total.", len(todoStore)),
	}, nil
}

// todoReadTool allows reading the current todo list.
type todoReadTool struct{}

// NewTodoReadTool creates a tool for reading the current todo list.
func NewTodoReadTool() core.FuncTool {
	return &todoReadTool{}
}

func (t *todoReadTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "todo_read",
		Description: "Read the current todo list. Returns all items and their statuses, sorted by priority.",
		Parameters: []core.Parameter{
			{
				Name:        "status",
				Type:        "string",
				Description: "Optional filter by status: pending, in_progress, completed, cancelled.",
				Required:    false,
			},
		},
	}
}

func (t *todoReadTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	todoStoreMu.RLock()
	defer todoStoreMu.RUnlock()

	statusFilter, _ := params["status"].(string)

	var filtered []TodoItem
	for _, item := range todoStore {
		if statusFilter != "" && item.Status != statusFilter {
			continue
		}
		filtered = append(filtered, item)
	}

	if len(filtered) == 0 {
		return map[string]any{
			"success": true,
			"count":   0,
			"items":   []TodoItem{},
			"summary": "No todo items match the filter.",
		}, nil
	}

	return map[string]any{
		"success": true,
		"count":   len(filtered),
		"items":   filtered,
		"summary": formatTodoSummary(filtered),
	}, nil
}

// TodoExecuteTool analyzes pending todos and returns a plan for executing them
// in priority/dependency order, optionally generating task_create calls.
type TodoExecuteTool struct{}

// NewTodoExecuteTool creates a tool for executing todos in order.
func NewTodoExecuteTool() core.FuncTool {
	return &TodoExecuteTool{}
}

const todoExecuteDescription = `Analyze the current todo list and return an execution plan.
For each pending todo that has tool_call and tool_params configured, it generates
the corresponding tool invocation plan. Todos are executed in priority order,
respecting dependencies.

Usage:
- Call this tool after setting up todos with todo_write.
- Review the execution plan before proceeding.
- Execute each step in the plan using the specified tools.
- Mark completed todos by calling todo_write with merge=true.`

func (t *TodoExecuteTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "todo_execute",
		Description: todoExecuteDescription,
		Parameters: []core.Parameter{
			{
				Name:        "auto_execute",
				Type:        "boolean",
				Description: "If true, returns the execution steps. If false, just returns the plan.",
				Required:    false,
			},
		},
	}
}

func (t *TodoExecuteTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	todoStoreMu.RLock()
	defer todoStoreMu.RUnlock()

	// Separate pending items and completed dependency set
	completedSet := make(map[string]bool)
	for _, item := range todoStore {
		if item.Status == "completed" {
			completedSet[item.ID] = true
		}
	}

	// Filter pending items whose dependencies are all met
	var ready []TodoItem
	var blocked []TodoItem
	for _, item := range todoStore {
		if item.Status != "pending" {
			continue
		}
		allDepsMet := true
		for _, dep := range item.Dependencies {
			if !completedSet[dep] {
				allDepsMet = false
				break
			}
		}
		if allDepsMet {
			ready = append(ready, item)
		} else {
			blocked = append(blocked, item)
		}
	}

	// Sort ready items by priority (stable sort)
	for i := 0; i < len(ready)-1; i++ {
		for j := i + 1; j < len(ready); j++ {
			if ready[j].Priority < ready[i].Priority {
				ready[i], ready[j] = ready[j], ready[i]
			}
		}
	}

	// Build execution plan
	type ExecStep struct {
		TodoID     string `json:"todo_id"`
		Content    string `json:"content"`
		ToolCall   string `json:"tool_call,omitempty"`
		ToolParams string `json:"tool_params,omitempty"`
		Priority   int    `json:"priority"`
	}

	var steps []ExecStep
	for _, item := range ready {
		steps = append(steps, ExecStep{
			TodoID:     item.ID,
			Content:    item.Content,
			ToolCall:   item.ToolCall,
			ToolParams: item.ToolParams,
			Priority:   item.Priority,
		})
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Execution Plan: %d ready, %d blocked, %d total\n", len(ready), len(blocked), len(todoStore))
	sb.WriteString("--- Ready to Execute (by priority) ---\n")
	for i, step := range steps {
		fmt.Fprintf(&sb, "  Step %d: [%s] %s (priority=%d)\n", i+1, step.TodoID, step.Content, step.Priority)
		if step.ToolCall != "" {
			fmt.Fprintf(&sb, "    -> %s(%s)\n", step.ToolCall, step.ToolParams)
		}
	}
	if len(blocked) > 0 {
		sb.WriteString("--- Blocked (waiting for dependencies) ---\n")
		for _, item := range blocked {
			fmt.Fprintf(&sb, "  [%s] %s (deps: %v)\n", item.ID, item.Content, item.Dependencies)
		}
	}

	return map[string]any{
		"success":       true,
		"ready_count":   len(ready),
		"blocked_count": len(blocked),
		"steps":         steps,
		"summary":       sb.String(),
	}, nil
}

// formatTodoSummary builds a human-readable summary of todo items.
func formatTodoSummary(items []TodoItem) string {
	var sb strings.Builder
	for _, item := range items {
		prio := ""
		if item.Priority != 0 {
			prio = fmt.Sprintf(" (prio:%d)", item.Priority)
		}
		tool := ""
		if item.ToolCall != "" {
			tool = fmt.Sprintf(" [%s]", item.ToolCall)
		}
		fmt.Fprintf(&sb, "  [%s] %s:%s%s%s\n", item.Status, item.ID, item.Content, prio, tool)
	}
	return sb.String()
}
