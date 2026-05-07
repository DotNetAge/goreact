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
	Status        string   `json:"status"`         // pending, in_progress, completed, cancelled
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
var todoStores sync.Map // map[string]*sessionTodoStore

type sessionTodoStore struct {
	mu      sync.RWMutex
	items   []TodoItem
	counter int
}

func (s *sessionTodoStore) nextID() string {
	s.counter++
	return fmt.Sprintf("todo_%d", s.counter)
}

func getSessionStore(ctx context.Context) *sessionTodoStore {
	sessionID := ExtractSessionID(ctx)
	if sessionID == "" {
		sessionID = "__default__"
	}
	store, _ := todoStores.LoadOrStore(sessionID, &sessionTodoStore{
		items: make([]TodoItem, 0),
	})
	return store.(*sessionTodoStore)
}

// TodoWriteTool implements a tool for managing a task list.
type TodoWriteTool struct{}

// NewTodoWriteTool creates a todo write tool.
func NewTodoWriteTool() core.FuncTool {
	return &TodoWriteTool{}
}

const prompt = `Track progress on any multi-step task by maintaining a visible checklist. Works for any kind of work — coding, research, analysis, writing, planning, debugging, or anything that requires multiple steps.

When to use:
- Break any complex task into visible steps: a user request that involves 3+ distinct actions, has branching logic, or requires sequential verification.
- Show progress to the user: they can see what you've done, what's next, and what's blocked.
- Manage parallel work streams: use dependencies to express ordering constraints between independent tasks.
- Keep yourself organized: when switching between tasks, the todo list helps you remember where you left off.

Usage:
- Create todos proactively when starting any multi-step task — before you begin, outline what you'll do.
- When you START working on a task - mark it as in_progress (use merge=true).
- IMMEDIATELY after COMPLETING a task - mark it as completed (use merge=true).
- Status must be one of: pending, in_progress, completed, cancelled.
- Use priority (lower = higher priority) and dependencies to define execution order.
- Use tool_call and tool_params to pre-configure which tool to use for each todo item.`

func (t *TodoWriteTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "TodoWrite",
		Description: "Track progress on any multi-step task with a visible checklist. Break down work, show progress, and manage dependencies.",
		Prompt:      prompt,
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

	store := getSessionStore(ctx)
	store.mu.Lock()
	defer store.mu.Unlock()

	if merge {
		for i := range newItems {
			if newItems[i].ID == "" {
				newItems[i].ID = store.nextID()
			}
			if newItems[i].CreatedAt == 0 {
				newItems[i].CreatedAt = now
			}
			newItems[i].UpdatedAt = now

			found := false
			for j, existing := range store.items {
				if existing.ID == newItems[i].ID {
					store.items[j] = newItems[i]
					found = true
					break
				}
			}
			if !found {
				store.items = append(store.items, newItems[i])
			}
		}
	} else {
		for i := range newItems {
			if newItems[i].ID == "" {
				newItems[i].ID = store.nextID()
			}
			if newItems[i].CreatedAt == 0 {
				newItems[i].CreatedAt = now
			}
			newItems[i].UpdatedAt = now
		}
		store.items = newItems
	}

	return map[string]any{
		"success": true,
		"count":   len(store.items),
		"summary": formatTodoSummary(store.items),
		"message": fmt.Sprintf("Todo list updated. %d item(s) total.", len(store.items)),
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
		Name:        "TodoRead",
		Description: "Read the current todo list with optional status filter.",
		Prompt: `View the current todo list, sorted by priority. Use this to check progress, see what's pending, or find tasks matching a specific status.

Usage:
- Call with no parameters to see everything.
- Add status="in_progress" to see what you're currently working on.
- Add status="pending" to see what's next.
- Use this before TodoWrite to see current state before making changes.`,
		Tags:        []string{"task", "plan", "read", "status"},
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
	store := getSessionStore(ctx)
	store.mu.RLock()
	defer store.mu.RUnlock()

	statusFilter, _ := params["status"].(string)

	var filtered []TodoItem
	for _, item := range store.items {
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
Pending todos with tool_call and tool_params configured are turned into
actionable steps, executed in priority order with dependencies respected.

Usage:
- Call after setting up todos with TodoWrite.
- Review the plan, then execute each step.
- Mark completed todos via TodoWrite with merge=true.`

func (t *TodoExecuteTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "TodoExecute",
		Description: "Analyze pending todos and produce an execution plan in priority order.",
		Prompt:      todoExecuteDescription,
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
	store := getSessionStore(ctx)
	store.mu.RLock()
	defer store.mu.RUnlock()

	// Separate pending items and completed dependency set
	completedSet := make(map[string]bool)
	for _, item := range store.items {
		if item.Status == "completed" {
			completedSet[item.ID] = true
		}
	}

	// Filter pending items whose dependencies are all met
	var ready []TodoItem
	var blocked []TodoItem
	for _, item := range store.items {
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
	fmt.Fprintf(&sb, "Execution Plan: %d ready, %d blocked, %d total\n", len(ready), len(blocked), len(store.items))
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
