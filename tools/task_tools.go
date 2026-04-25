package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// --- Task Tools (Synchronous Inline Execution) ---
//
// TaskCreateTool creates and executes a subtask synchronously within the
// current reactor thread. This is the plan→execute workflow tool that
// pairs with todo_execute for sequential execution.

// TaskCreateTool executes a subtask synchronously inline.
type TaskCreateTool struct {
	accessor   ReactorAccessor
	parentTask *string
}

// SetAccessor sets the reactor accessor for task management.
func (t *TaskCreateTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// SetParentTaskID sets the parent task ID for hierarchy tracking.
func (t *TaskCreateTool) SetParentTaskID(id string) {
	t.parentTask = &id
}

// NewTaskCreateTool creates a new TaskCreateTool.
func NewTaskCreateTool() *TaskCreateTool {
	return &TaskCreateTool{}
}

func (t *TaskCreateTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name: "task_create",
		Description: `Create and execute a subtask synchronously within the current reactor thread. The task runs inline with the same system prompt and model, ensuring sequential execution order. Use this for step-by-step plan execution (e.g., after todo_execute produces a plan). Returns the task result immediately upon completion.

Key behaviors:
- Runs synchronously: blocks until the task finishes.
- Shares the same reactor context (system prompt, model, tools, event bus).
- Task output is injected into the current conversation for continuity.
- Use 'task_list' to view all tasks and their statuses.
- Use 'task_result' to retrieve the result of a previously completed task.`,
		Parameters: []core.Parameter{
			{Name: "description", Type: "string", Description: "A short description of the task.", Required: true},
			{Name: "prompt", Type: "string", Description: "Detailed instructions for the task. Be specific about what output is expected.", Required: true},
		},
	}
}

func (t *TaskCreateTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	description, ok1 := params["description"].(string)
	prompt, ok2 := params["prompt"].(string)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("missing required parameters: description, prompt")
	}

	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	parentID := ""
	if t.parentTask != nil && *t.parentTask != "" {
		parentID = *t.parentTask
	}

	tm := t.accessor.TaskManager()
	task, err := tm.CreateTask(parentID, description, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create task record: %w", err)
	}

	_ = tm.UpdateTaskStatus(task.ID, core.TaskStatusInProgress, "", "")

	// Emit event
	if emitter := t.accessor.EventEmitter(); emitter != nil {
		emitter(core.NewReactEvent("", "main", parentID, core.SubtaskSpawned,
			core.SubtaskInfo{TaskID: task.ID, Description: description}))
	}

	// Run synchronously inline using the accessor's RunInline method
	answer, runErr := t.accessor.RunInline(ctx, prompt)

	if runErr != nil {
		_ = tm.UpdateTaskStatus(task.ID, core.TaskStatusFailed, "", runErr.Error())
		if emitter := t.accessor.EventEmitter(); emitter != nil {
			emitter(core.NewReactEvent("", task.ID, parentID, core.SubtaskCompleted,
				core.SubtaskResult{TaskID: task.ID, Success: false, Error: runErr.Error()}))
		}
		return fmt.Sprintf("Task %q failed: %v", task.ID, runErr), runErr
	}

	_ = tm.UpdateTaskStatus(task.ID, core.TaskStatusCompleted, answer, "")

	if emitter := t.accessor.EventEmitter(); emitter != nil {
		emitter(core.NewReactEvent("", task.ID, parentID, core.SubtaskCompleted,
			core.SubtaskResult{TaskID: task.ID, Success: true, Answer: answer}))
	}

	return fmt.Sprintf("Task %q completed.\nAnswer: %s", task.ID, answer), nil
}

// --- Task Result Tool ---

// TaskResultTool retrieves the result of a previously completed task.
type TaskResultTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *TaskResultTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewTaskResultTool creates a new TaskResultTool.
func NewTaskResultTool() *TaskResultTool {
	return &TaskResultTool{}
}

func (t *TaskResultTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "task_result",
		Description: "Retrieve the result of a previously completed task by its ID.",
		Parameters: []core.Parameter{
			{Name: "task_id", Type: "string", Description: "The ID of the task to retrieve results for.", Required: true},
		},
	}
}

func (t *TaskResultTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	taskID, ok := params["task_id"].(string)
	if !ok || taskID == "" {
		return "", fmt.Errorf("missing required parameter: task_id")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	task, err := t.accessor.TaskManager().GetTask(taskID)
	if err != nil {
		return "", fmt.Errorf("task %q not found", taskID)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Task %q\n", taskID)
	fmt.Fprintf(&sb, "  Status:      %s\n", task.Status)
	fmt.Fprintf(&sb, "  Description: %s\n", task.Description)
	fmt.Fprintf(&sb, "  Parent:      %s\n", task.ParentID)
	if task.Output != "" {
		fmt.Fprintf(&sb, "  Output:      %s\n", task.Output)
	}
	if task.Error != "" {
		fmt.Fprintf(&sb, "  Error:       %s\n", task.Error)
	}
	return sb.String(), nil
}

// --- Task List Tool ---

// TaskListTool lists all tasks and their statuses.
type TaskListTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *TaskListTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewTaskListTool creates a new TaskListTool.
func NewTaskListTool() *TaskListTool {
	return &TaskListTool{}
}

func (t *TaskListTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "task_list",
		Description: "List all tasks and their current statuses. Use this to track plan execution progress after using task_create.",
		IsReadOnly:  true,
		Parameters: []core.Parameter{
			{Name: "parent_id", Type: "string", Description: "Optional: filter tasks by parent task ID.", Required: false},
		},
	}
}

func (t *TaskListTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	parentID, _ := params["parent_id"].(string)
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	tm := t.accessor.TaskManager()
	var tasks []*core.Task
	var err error
	if parentID != "" {
		tasks, err = tm.ListSubTasks(parentID)
	} else {
		tasks, err = tm.ListAllTasks()
	}
	if err != nil {
		return "", fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		if parentID != "" {
			return fmt.Sprintf("No subtasks found for parent %q.", parentID), nil
		}
		return "No tasks have been created yet.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d task(s):\n", len(tasks))
	for _, task := range tasks {
		fmt.Fprintf(&sb, "  - ID: %s | Status: %s | Parent: %s | Desc: %s\n",
			task.ID, task.Status, task.ParentID, task.Description)
		if task.Output != "" {
			output := task.Output
			if len(output) > 200 {
				output = output[:200] + "... [truncated]"
			}
			fmt.Fprintf(&sb, "    Output: %s\n", output)
		}
		if task.Error != "" {
			fmt.Fprintf(&sb, "    Error: %s\n", task.Error)
		}
	}
	return sb.String(), nil
}
