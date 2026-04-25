package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// --- SubAgent Tools (Async Independent Agents) ---

// SubAgentTool spawns an independent Agent with its own SystemPrompt and Model.
type SubAgentTool struct {
	accessor   ReactorAccessor
	parentTask *string
}

// SetAccessor sets the reactor accessor.
func (t *SubAgentTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// SetParentTaskID sets the parent task ID for hierarchy tracking.
func (t *SubAgentTool) SetParentTaskID(id string) {
	t.parentTask = &id
}

// NewSubAgentTool creates a new SubAgentTool.
func NewSubAgentTool() *SubAgentTool {
	return &SubAgentTool{}
}

func (t *SubAgentTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name: "subagent",
		Description: `Spawn an independent Agent (SubAgent) to handle a subtask. A SubAgent has its own SystemPrompt and Model, runs in its own goroutine (independent thread), and can communicate with other team members via send_message/receive_messages.

Team collaboration:
1. First create a team with 'team_create'.
2. Spawn SubAgents with 'team_name' to automatically join the team.
3. Agents communicate via 'send_message' (channel-based messaging).
4. Lead agent uses 'wait_team' to collect all results.
5. Lead agent uses 'team_delete' to clean up.`,
		Parameters: []core.Parameter{
			{Name: "name", Type: "string", Description: "Agent name within the team. Use the format @{role_name} (e.g., '@researcher', '@reviewer'). The @ prefix helps the LLM identify agent references in skills and prompts. Must be unique within the team.", Required: true},
			{Name: "description", Type: "string", Description: "Brief description of this agent's task.", Required: true},
			{Name: "prompt", Type: "string", Description: "The task instruction for this agent. Should be self-contained.", Required: true},
			{Name: "system_prompt", Type: "string", Description: "The agent's system prompt — defines its role, expertise, and behavior.", Required: false},
			{Name: "model", Type: "string", Description: "Override model (e.g., 'gpt-4o', 'claude-3-opus'). If empty, inherits parent's model.", Required: false},
			{Name: "team_name", Type: "string", Description: "Team ID to join. If set, the agent joins the team and gets send_message/receive_messages tools.", Required: false},
		},
	}
}

func (t *SubAgentTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	name, ok1 := params["name"].(string)
	description, ok2 := params["description"].(string)
	prompt, ok3 := params["prompt"].(string)
	if !ok1 || !ok2 || !ok3 {
		return "", fmt.Errorf("missing required parameters: name, description, prompt")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	systemPrompt, _ := params["system_prompt"].(string)
	model, _ := params["model"].(string)
	teamName, _ := params["team_name"].(string)

	parentID := ""
	if t.parentTask != nil && *t.parentTask != "" {
		parentID = *t.parentTask
	}

	tm := t.accessor.TaskManager()
	task, err := tm.CreateTask(parentID, description, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create subagent task record: %w", err)
	}

	task.Metadata = map[string]any{
		"subagent_name":    name,
		"subagent_prompt":  systemPrompt,
		"subagent_model":   model,
		"subagent_team_id": teamName,
	}
	_ = tm.UpdateTaskStatus(task.ID, core.TaskStatusInProgress, "", "")

	// Join team if specified
	bus := t.accessor.MessageBus()
	if teamName != "" && bus != nil {
		if err := bus.JoinTeam(teamName, name, task.ID); err != nil {
			return nil, fmt.Errorf("failed to join team %q: %w", teamName, err)
		}
	}

	// Emit event
	if emitter := t.accessor.EventEmitter(); emitter != nil {
		emitter(core.NewReactEvent("", "main", parentID, core.SubtaskSpawned,
			core.SubtaskInfo{TaskID: task.ID, Description: fmt.Sprintf("[%s] %s", name, description)}))
	}

	// Create result channel and register it
	resultCh := make(chan any, 1)
	t.accessor.RegisterPendingTask(task.ID, resultCh)

	// Launch async SubAgent execution in a goroutine
	t.accessor.RunSubAgent(ctx, task.ID, systemPrompt, prompt, model, resultCh)

	// Return immediately — the SubAgent runs in the background
	if teamName != "" {
		return fmt.Sprintf("SubAgent %q (ID: %s) queued for team %q.\nUse 'team_status' to monitor, 'wait_team' to collect all results.", name, task.ID, teamName), nil
	}
	return fmt.Sprintf("SubAgent %q (ID: %s) queued.\nUse 'subagent_result' with ID %q to retrieve the result.", name, task.ID, task.ID), nil
}

// SubAgentResultTool retrieves the result of an async subagent.
type SubAgentResultTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *SubAgentResultTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewSubAgentResultTool creates a new SubAgentResultTool.
func NewSubAgentResultTool() *SubAgentResultTool {
	return &SubAgentResultTool{}
}

func (t *SubAgentResultTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "subagent_result",
		Description: "Retrieve the result of a specific SubAgent by task ID. Blocks until the SubAgent completes or times out.",
		Parameters: []core.Parameter{
			{Name: "task_id", Type: "string", Description: "The SubAgent's task ID.", Required: true},
			{Name: "wait_seconds", Type: "integer", Description: "How long to wait in seconds (default: 60).", Required: false},
		},
	}
}

func (t *SubAgentResultTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	taskID, ok := params["task_id"].(string)
	if !ok || taskID == "" {
		return "", fmt.Errorf("missing required parameter: task_id")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	ch, exists := t.accessor.GetPendingTask(taskID)
	if !exists {
		task, err := t.accessor.TaskManager().GetTask(taskID)
		if err != nil {
			return "", fmt.Errorf("subagent task %q not found", taskID)
		}
		return fmt.Sprintf("SubAgent Task %q status: %s\nDescription: %s\nResult: %s",
			taskID, task.Status, task.Description, task.Output), nil
	}

	// Wait for result with timeout
	waitSeconds := 60
	if raw, ok := params["wait_seconds"]; ok {
		if v, ok := ToFloat64(raw); ok {
			waitSeconds = int(v)
		}
	}

	// Use a timer-based select on the channel
	select {
	case result, ok := <-ch:
		t.accessor.RemovePendingTask(taskID)
		if !ok {
			task, _ := t.accessor.TaskManager().GetTask(taskID)
			if task != nil {
				return fmt.Sprintf("SubAgent Task %q status: %s\nResult: %s", taskID, task.Status, task.Output), nil
			}
			return "", fmt.Errorf("subagent task %q channel closed unexpectedly", taskID)
		}
		if resultStr, ok := result.(string); ok {
			return fmt.Sprintf("SubAgent Task %q completed.\nAnswer: %s", taskID, resultStr), nil
		}
		return fmt.Sprintf("SubAgent Task %q completed.\nResult: %v", taskID, result), nil
	case <-time.After(time.Duration(waitSeconds) * time.Second):
		task, err := t.accessor.TaskManager().GetTask(taskID)
		status := "in_progress"
		if err == nil {
			status = string(task.Status)
		}
		return fmt.Sprintf("SubAgent Task %q is still running (status: %s). Try again later.", taskID, status), nil
	}
}
