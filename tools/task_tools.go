package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

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
			{Name: "name", Type: "string", Description: "Agent name within the team (e.g., 'researcher'). Must be unique within the team.", Required: true},
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

// SubAgentListTool lists all SubAgent tasks.
type SubAgentListTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *SubAgentListTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewSubAgentListTool creates a new SubAgentListTool.
func NewSubAgentListTool() *SubAgentListTool {
	return &SubAgentListTool{}
}

func (t *SubAgentListTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "subagent_list",
		Description: "List all spawned SubAgents and their statuses. For team-mode agents, prefer 'team_status'.",
		IsReadOnly:  true,
		Parameters: []core.Parameter{
			{Name: "parent_id", Type: "string", Description: "Optional: filter by parent task ID.", Required: false},
		},
	}
}

func (t *SubAgentListTool) Execute(ctx context.Context, params map[string]any) (any, error) {
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
		return "", fmt.Errorf("failed to list subagent tasks: %w", err)
	}

	var agentTasks []*core.Task
	for _, task := range tasks {
		if task.Metadata != nil && task.Metadata["subagent_name"] != nil {
			agentTasks = append(agentTasks, task)
		}
	}

	if len(agentTasks) == 0 {
		return "No SubAgent tasks have been spawned yet.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d SubAgent task(s):\n", len(agentTasks))
	for _, task := range agentTasks {
		agentName, _ := task.Metadata["subagent_name"].(string)
		agentModel, _ := task.Metadata["subagent_model"].(string)
		teamID, _ := task.Metadata["subagent_team_id"].(string)
		fmt.Fprintf(&sb, "  - %s | Agent: %s | Model: %s | Team: %s | Status: %s\n",
			task.ID, agentName, agentModel, teamID, task.Status)
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
