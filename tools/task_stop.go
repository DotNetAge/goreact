package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/core"
)

type TaskStopTool struct{}

func NewTaskStopTool() *TaskStopTool {
	return &TaskStopTool{}
}

func (t *TaskStopTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "TaskStop",
		Description: "Stop a running or pending task. The task's status will be set to 'stopped' and no further work will be done.",
		Prompt: `Stop a running or pending task by task_id.

Use this to:
- Terminate a task that is no longer needed
- Cancel a task that has been running too long
- Clean up before starting a different approach

The task's status will change to "stopped". A stopped task cannot be restarted.

Required parameter:
- task_id: the unique identifier of the task to stop

Returns:
- success: whether the task was stopped
- message: status message
- task_id: the stopped task's ID`,
		Tags: []string{"task", "stop", "cancel", "orchestration"},
		Parameters: []core.Parameter{
			{Name: "task_id", Type: "string", Description: "The unique identifier of the task to stop.", Required: true},
		},
	}
}

func (t *TaskStopTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	taskID, _ := params["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	tc := core.GetToolContext(ctx)
	if tc == nil || tc.SessionID == "" {
		return nil, fmt.Errorf("TaskStop requires ToolContext with SessionID")
	}

	task, err := GetTask(ctx, tc.SessionID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task %q not found", taskID)
	}

	if task.Status == TaskCompleted || task.Status == TaskFailed || task.Status == TaskStopped {
		return map[string]any{
			"success": false,
			"message": fmt.Sprintf("Task %q is already %s, cannot stop", taskID, task.Status),
			"task_id": taskID,
		}, nil
	}

	task.Status = TaskStopped
	now := time.Now()
	task.CompletedAt = &now

	if err := UpdateTask(ctx, tc.SessionID, task); err != nil {
		return nil, fmt.Errorf("failed to stop task: %w", err)
	}

	if tc.EmitEvent != nil {
		tc.EmitEvent(core.ReactEvent{
			AgentID: "main",
			Type:    core.SubtaskCompleted,
			Data:    map[string]any{"task_id": taskID, "success": false, "stopped": true},
		})
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Task %q stopped successfully", taskID),
		"task_id": taskID,
	}, nil
}
