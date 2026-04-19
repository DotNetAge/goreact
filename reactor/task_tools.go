package reactor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

const (
	// DefaultSubAgentTimeout is the default timeout for a subagent execution.
	DefaultSubAgentTimeout = 5 * time.Minute
)

// TaskCreateTool allows the agent to spawn a subagent to handle a specific subtask.
type TaskCreateTool struct {
	reactor    *Reactor
	parentTask *string // nullable; set by the reactor when running as a subagent
}

// SetParentTaskID sets the parent task ID for task hierarchy tracking.
func (t *TaskCreateTool) SetParentTaskID(id string) {
	t.parentTask = &id
}

func NewTaskCreateTool(r *Reactor) *TaskCreateTool {
	return &TaskCreateTool{reactor: r}
}

func (t *TaskCreateTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "task_create",
		Description: "Spawn a subagent to handle a complex subtask asynchronously. The subagent runs in a background goroutine with independent timeout control. Use 'task_list' or 'task_result' to check progress and retrieve results.",
		Parameters: []core.Parameter{
			{
				Name:        "description",
				Type:        "string",
				Description: "A short description of the task.",
				Required:    true,
			},
			{
				Name:        "prompt",
				Type:        "string",
				Description: "Detailed instructions for the subagent. Be specific about what output is expected.",
				Required:    true,
			},
			{
				Name:        "timeout_seconds",
				Type:        "integer",
				Description: "Optional timeout in seconds for the subagent (default: 300).",
				Required:    false,
			},
		},
	}
}

func (t *TaskCreateTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	description, ok1 := params["description"].(string)
	prompt, ok2 := params["prompt"].(string)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("missing required parameters: description, prompt")
	}

	// Determine parent ID: prefer explicit parent, fall back to tool's parentTask
	parentID := ""
	if t.parentTask != nil && *t.parentTask != "" {
		parentID = *t.parentTask
	}

	// Register task in TaskManager with parent ID for hierarchy tracking
	task, err := t.reactor.taskManager.CreateTask(parentID, description, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create task record: %w", err)
	}
	_ = t.reactor.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusInProgress, "", "")

	fmt.Printf("[Subagent] Starting task %s: %s\n", task.ID, description)

	// Create a sub-reactor with the same config, sharing the EventBus
	subReactor := NewReactor(t.reactor.config, WithEventBus(t.reactor.eventBus))
	subReactor.toolRegistry = t.reactor.toolRegistry
	subReactor.skillRegistry = t.reactor.skillRegistry
	subReactor.taskManager = t.reactor.taskManager
	subReactor.llmClient = t.reactor.llmClient

	// Set parent task ID on the subagent's TaskCreateTool so nested tasks get proper hierarchy
	if tool, exists := subReactor.toolRegistry.Get("task_create"); exists {
		if tct, ok := tool.(*TaskCreateTool); ok {
			tct.SetParentTaskID(task.ID)
		}
	}

	// Determine timeout
	timeout := DefaultSubAgentTimeout
	if raw, ok := params["timeout_seconds"]; ok {
		switch v := raw.(type) {
		case float64:
			timeout = time.Duration(v) * time.Second
		case int:
			timeout = time.Duration(v) * time.Second
		}
	}

	// Create an independent context with timeout isolation
	subCtx, subCancel := context.WithTimeout(context.Background(), timeout)

	// Register pending task on the reactor's task manager (Issue #2: instance-bound)
	resultCh := make(chan *RunResult, 1)
	t.reactor.registerPendingTask(task.ID, resultCh)

	// Emit subtask spawned event
	if t.reactor.eventBus != nil {
		t.reactor.eventBus.Emit(core.NewReactEvent(
			"", "main", "", core.SubtaskSpawned,
			core.SubtaskInfo{TaskID: task.ID, Description: description, Timeout: timeout.String()},
		))
	}

	// Launch subagent asynchronously in a goroutine
	go func() {
		defer subCancel()
		result, runErr := subReactor.Run(subCtx, prompt, nil)
		if runErr != nil {
			_ = t.reactor.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusFailed, "", runErr.Error())
			if t.reactor.eventBus != nil {
				t.reactor.eventBus.Emit(core.NewReactEvent(
					"", task.ID, parentID, core.SubtaskCompleted,
					core.SubtaskResult{TaskID: task.ID, Success: false, Error: runErr.Error()},
				))
			}
			resultCh <- &RunResult{Answer: fmt.Sprintf("subagent failed: %v", runErr)}
		} else {
			_ = t.reactor.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusCompleted, result.Answer, "")
			if t.reactor.eventBus != nil {
				t.reactor.eventBus.Emit(core.NewReactEvent(
					"", task.ID, parentID, core.SubtaskCompleted,
					core.SubtaskResult{TaskID: task.ID, Success: true, Answer: result.Answer},
				))
			}
			resultCh <- result
		}
		close(resultCh)
	}()

	return fmt.Sprintf("Subtask %q (ID: %s) spawned and running asynchronously (timeout: %v).\nUse 'task_result' with ID %q to retrieve the result when ready.", description, task.ID, timeout, task.ID), nil
}

// --- Task Result Tool ---

// TaskResultTool allows the agent to retrieve the result of an async subagent task.
type TaskResultTool struct {
	reactor *Reactor
}

func NewTaskResultTool(r *Reactor) *TaskResultTool {
	return &TaskResultTool{reactor: r}
}

func (t *TaskResultTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "task_result",
		Description: "Retrieve the result of a previously spawned subagent task. Blocks until the task completes or times out. Returns the task status and output.",
		Parameters: []core.Parameter{
			{
				Name:        "task_id",
				Type:        "string",
				Description: "The ID of the task to retrieve results for.",
				Required:    true,
			},
			{
				Name:        "wait_seconds",
				Type:        "integer",
				Description: "How long to wait for the result in seconds (default: 60). If the task is not done, returns current status.",
				Required:    false,
			},
		},
	}
}

func (t *TaskResultTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	taskID, ok := params["task_id"].(string)
	if !ok || taskID == "" {
		return "", fmt.Errorf("missing required parameter: task_id")
	}

	waitTimeout := 60 * time.Second
	if raw, ok := params["wait_seconds"]; ok {
		switch v := raw.(type) {
		case float64:
			waitTimeout = time.Duration(v) * time.Second
		case int:
			waitTimeout = time.Duration(v) * time.Second
		}
	}

	// Check the pending channel from reactor instance (Issue #2: instance-bound)
	ch, exists := t.reactor.getPendingTask(taskID)

	if !exists {
		// No pending channel — task may have completed already; check TaskManager
		task, err := t.reactor.taskManager.GetTask(taskID)
		if err != nil {
			return "", fmt.Errorf("task %q not found", taskID)
		}
		return fmt.Sprintf("Task %q status: %s\nDescription: %s\nResult: %s",
			taskID, task.Status, task.Description, task.Output), nil
	}

	// Wait for the result with a timeout
	select {
	case result, ok := <-ch:
		t.reactor.removePendingTask(taskID)
		if !ok {
			task, _ := t.reactor.taskManager.GetTask(taskID)
			if task != nil {
				return fmt.Sprintf("Task %q status: %s\nDescription: %s\nResult: %s",
					taskID, task.Status, task.Description, task.Output), nil
			}
			return "", fmt.Errorf("task %q channel closed unexpectedly", taskID)
		}
		return fmt.Sprintf("Task %q completed.\nAnswer: %s", taskID, result.Answer), nil
	case <-time.After(waitTimeout):
		task, err := t.reactor.taskManager.GetTask(taskID)
		status := "in_progress"
		if err == nil {
			status = string(task.Status)
		}
		return fmt.Sprintf("Task %q is still running (status: %s). Try again later with 'task_result'.", taskID, status), nil
	}
}

// --- Task List Tool ---

// TaskListTool allows the agent to list all tasks and their statuses.
type TaskListTool struct {
	reactor *Reactor
}

func NewTaskListTool(r *Reactor) *TaskListTool {
	return &TaskListTool{reactor: r}
}

func (t *TaskListTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "task_list",
		Description: "List all spawned subagent tasks and their current statuses. Use this to monitor parallel task progress.",
		Parameters: []core.Parameter{
			{
				Name:        "parent_id",
				Type:        "string",
				Description: "Optional: filter tasks by parent task ID. Leave empty to list all tasks.",
				Required:    false,
			},
		},
	}
}

func (t *TaskListTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	parentID, _ := params["parent_id"].(string)

	var tasks []*core.Task
	var err error
	if parentID != "" {
		tasks, err = t.reactor.taskManager.ListSubTasks(parentID)
	} else {
		tasks, err = t.reactor.taskManager.ListAllTasks()
	}
	if err != nil {
		return "", fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		if parentID != "" {
			return fmt.Sprintf("No subtasks found for parent %q.", parentID), nil
		}
		return "No tasks have been spawned yet.", nil
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
