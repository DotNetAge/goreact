package reactor

import (
	"context"
	"fmt"
	"github.com/DotNetAge/goreact/core"
)

// TaskCreateTool allows the agent to spawn a subagent to handle a specific subtask.
type TaskCreateTool struct {
	reactor *defaultReactor
}

func NewTaskCreateTool(r *defaultReactor) *TaskCreateTool {
	return &TaskCreateTool{reactor: r}
}

func (t *TaskCreateTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "task_create",
		Description: "Spawn a subagent to handle a complex subtask autonomously. Use this for independent research or complex multi-step sub-problems.",
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
		},
	}
}

func (t *TaskCreateTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	description, ok1 := params["description"].(string)
	prompt, ok2 := params["prompt"].(string)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("missing required parameters: description, prompt")
	}

	// Register task in TaskManager
	task, err := t.reactor.taskManager.CreateTask("", description, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create task record: %w", err)
	}
	_ = t.reactor.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusInProgress, "", "")

	fmt.Printf("[Subagent] Starting task %s: %s\n", task.ID, description)

	// Create a sub-reactor with the same config
	subReactor := NewReactor(t.reactor.config)
	// Subagent inherits the registries and task manager
	subReactor.toolRegistry = t.reactor.toolRegistry
	subReactor.skillRegistry = t.reactor.skillRegistry
	subReactor.taskManager = t.reactor.taskManager

	// Run the subagent loop
	result, err := subReactor.Run(ctx, prompt, nil)
	if err != nil {
		_ = t.reactor.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusFailed, "", err.Error())
		return "", fmt.Errorf("subagent failed: %w", err)
	}

	_ = t.reactor.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusCompleted, result.Answer, "")
	return fmt.Sprintf("Subtask %q (ID: %s) completed.\nResult: %s", description, task.ID, result.Answer), nil
}
