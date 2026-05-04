package tools

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/DotNetAge/goreact/core"
)

type TaskCreateTool struct {
	spawn   SpawnFunc
	counter atomic.Int64
}

func NewTaskCreateTool(spawn SpawnFunc) *TaskCreateTool {
	return &TaskCreateTool{spawn: spawn}
}

func (t *TaskCreateTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "TaskCreate",
		Description: "Create a background task that runs an agent asynchronously. Returns immediately with a task_id. Use TaskGet to retrieve output or TaskList to see all tasks.",
		Prompt: `Create and start a background task. The task runs an agent asynchronously and returns immediately.

Use cases:
- Long-running agent tasks that you want to track
- Parallel agent execution — create multiple tasks and check their status later
- Tasks that need to be monitored or stopped later

The task_id returned can be used with:
- TaskGet: retrieve task output and status
- TaskList: see all tasks in the current session
- TaskUpdate: update task description or metadata
- TaskStop: stop a running task

Usage:
- Provide a clear, descriptive task_description
- Specify the agent_name to run the task (use FindAgent to discover available agents)
- Multiple TaskCreate calls in the same round run in parallel`,
		Tags:    []string{"task", "create", "async", "agent", "orchestration"},
		IsAsync: true,
		Parameters: []core.Parameter{
			{Name: "task_description", Type: "string", Description: "Clear description of what the task should accomplish.", Required: true},
			{Name: "agent_name", Type: "string", Description: "Name of the agent to execute the task. Use FindAgent to discover agents.", Required: true},
		},
	}
}

func (t *TaskCreateTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	taskDesc, _ := params["task_description"].(string)
	if taskDesc == "" {
		return nil, fmt.Errorf("task_description is required")
	}
	agentName, _ := params["agent_name"].(string)
	if agentName == "" {
		return nil, fmt.Errorf("agent_name is required")
	}

	tc := core.GetToolContext(ctx)
	if tc == nil || tc.EmitEvent == nil {
		return nil, fmt.Errorf("TaskCreate requires ToolContext with EventBus")
	}
	if t.spawn == nil {
		return nil, fmt.Errorf("TaskCreate: SpawnFunc not configured")
	}
	if tc.SessionID == "" {
		return nil, fmt.Errorf("TaskCreate: SessionID not set")
	}

	taskID := fmt.Sprintf("task-%d", t.counter.Add(1))

	task := &Task{
		ID:          taskID,
		Type:        TaskTypeAgent,
		Description: taskDesc,
		Status:      TaskPending,
		AgentName:   agentName,
		Prompt:      taskDesc,
	}

	if err := CreateTask(ctx, tc.SessionID, task); err != nil {
		return nil, fmt.Errorf("failed to create task record: %w", err)
	}

	tc.EmitEvent(core.ReactEvent{
		AgentID: "main",
		Type:    core.SubtaskSpawned,
		Data:    map[string]any{"task_id": taskID, "agent_name": agentName, "task": taskDesc},
	})

	go func() {
		now := time.Now()
		task.StartedAt = &now
		task.Status = TaskRunning
		_ = UpdateTask(ctx, tc.SessionID, task)

		result, err := t.spawn(ctx, agentName, taskDesc)
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		if err != nil {
			task.Status = TaskFailed
			task.Error = err.Error()
		} else {
			task.Status = TaskCompleted
			task.Result = result
		}
		_ = UpdateTask(ctx, tc.SessionID, task)

		if tc.ResultStore != nil {
			taskResult := &core.TaskResult{
				TaskID: taskID,
				Result: result,
				Done:   true,
			}
			if err != nil {
				taskResult.Error = err.Error()
			}
			tc.ResultStore.Store(taskID, taskResult)
		}

		tc.EmitEvent(core.ReactEvent{
			AgentID: agentName,
			Type:    core.SubtaskCompleted,
			Data:    map[string]any{"task_id": taskID, "success": err == nil},
		})
	}()

	return map[string]any{
		"task_id":          taskID,
		"status":           "running",
		"agent_name":       agentName,
		"task_description": taskDesc,
	}, nil
}
