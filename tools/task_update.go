package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

type TaskUpdateTool struct{}

func NewTaskUpdateTool() *TaskUpdateTool {
	return &TaskUpdateTool{}
}

func (t *TaskUpdateTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "TaskUpdate",
		Description: "Update a task's description or other metadata. Cannot change task status directly (use TaskStop to stop a task).",
		Prompt: `Update a task's metadata such as its description.

Use cases:
- Refine or clarify the task description after further analysis
- Add context to a task that was created too generically

You can update:
- description: the task's description/prompt

You cannot:
- Change task status directly (use TaskStop to stop)
- Change task type or agent assignment
- Update a task that doesn't exist

Required parameter:
- task_id: the unique identifier of the task to update

Optional parameter:
- description: new description for the task`,
		Tags: []string{"task", "update", "metadata", "orchestration"},
		Parameters: []core.Parameter{
			{Name: "task_id", Type: "string", Description: "The unique identifier of the task to update.", Required: true},
			{Name: "description", Type: "string", Description: "New description for the task.", Required: false},
		},
	}
}

func (t *TaskUpdateTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	taskID, _ := params["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	tc := core.GetToolContext(ctx)
	if tc == nil || tc.SessionID == "" {
		return nil, fmt.Errorf("TaskUpdate requires ToolContext with SessionID")
	}

	task, err := GetTask(ctx, tc.SessionID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task %q not found", taskID)
	}

	updated := false
	if desc, ok := params["description"].(string); ok && desc != "" {
		task.Description = desc
		task.Prompt = desc
		updated = true
	}

	if !updated {
		return map[string]any{
			"success": false,
			"message": "No update parameters provided",
		}, nil
	}

	if err := UpdateTask(ctx, tc.SessionID, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Task %q updated successfully", taskID),
		"task_id": taskID,
	}, nil
}
