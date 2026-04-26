package tools

import (
	"context"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// --- Mock ReactorAccessor for testing ---

type mockReactorAccessor struct {
	taskManager *core.InMemoryTaskManager
	messageBus  *core.AgentMessageBus
	pending     map[string]chan any
	runInlineFn func(ctx context.Context, prompt string) (string, error)
}

func newMockReactorAccessor() *mockReactorAccessor {
	return &mockReactorAccessor{
		taskManager: core.NewInMemoryTaskManager(),
		messageBus:  core.NewAgentMessageBus(),
		pending:     make(map[string]chan any),
	}
}

func (m *mockReactorAccessor) TaskManager() core.TaskManager       { return m.taskManager }
func (m *mockReactorAccessor) MessageBus() *core.AgentMessageBus   { return m.messageBus }
func (m *mockReactorAccessor) EventEmitter() func(core.ReactEvent) { return nil }
func (m *mockReactorAccessor) RegisterPendingTask(taskID string, resultCh chan any) {
	m.pending[taskID] = resultCh
}
func (m *mockReactorAccessor) GetPendingTask(taskID string) (<-chan any, bool) {
	ch, ok := m.pending[taskID]
	return ch, ok
}
func (m *mockReactorAccessor) RemovePendingTask(taskID string) { delete(m.pending, taskID) }
func (m *mockReactorAccessor) RunInline(_ context.Context, prompt string) (string, error) {
	if m.runInlineFn != nil {
		return m.runInlineFn(context.Background(), prompt)
	}
	return "inline result for: " + prompt, nil
}
func (m *mockReactorAccessor) RunSubAgent(_ context.Context, _ string, _, _ string, _ string, resultCh chan<- any) {
	resultCh <- map[string]string{"status": "completed", "output": "subagent done"}
}
func (m *mockReactorAccessor) Scheduler() *core.CronScheduler { return nil }
func (m *mockReactorAccessor) Config() ReactorConfig          { return ReactorConfig{} }

// ============================================================
// SubAgent Tool Tests
// ============================================================

func TestSubAgentTool_Info(t *testing.T) {
	tool := NewSubAgentTool()
	info := tool.Info()
	if info.Name != "subagent" {
		t.Errorf("Name = %q, want %q", info.Name, "subagent")
	}
}

func TestSubAgentTool_Execute_MissingParams(t *testing.T) {
	tool := NewSubAgentTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing params")
	}
}

func TestSubAgentTool_Execute_MissingName(t *testing.T) {
	tool := NewSubAgentTool()
	_, err := tool.Execute(context.Background(), map[string]any{"description": "test", "prompt": "do it"})
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestSubAgentTool_Execute_NilAccessor(t *testing.T) {
	tool := NewSubAgentTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"name": "@worker", "description": "test", "prompt": "do it",
	})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestSubAgentTool_Execute_Success(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewSubAgentTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{
		"name":        "@researcher",
		"description": "analyze code",
		"prompt":      "review this file",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result string")
	}
}

func TestSubAgentTool_Execute_WithTeam(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewSubAgentTool()
	tool.SetAccessor(mockAcc)

	team, _ := mockAcc.messageBus.CreateTeam("dev-team", "development team")

	result, err := tool.Execute(context.Background(), map[string]any{
		"name":        "@coder",
		"description": "write code",
		"prompt":      "implement feature X",
		"team_name":   team.ID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result with team")
	}
}

// ============================================================
// SubAgentList Tool Tests
// ============================================================

func TestSubAgentListTool_Info(t *testing.T) {
	tool := NewSubAgentListTool()
	info := tool.Info()
	if info.Name != "subagent_list" {
		t.Errorf("Name = %q, want %q", info.Name, "subagent_list")
	}
}

func TestSubAgentListTool_Execute_NilAccessor(t *testing.T) {
	tool := NewSubAgentListTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestSubAgentListTool_Execute_Empty(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewSubAgentListTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s != "No SubAgent tasks have been spawned yet." {
		t.Errorf("unexpected empty result: %q", s)
	}
}

func TestSubAgentListTool_Execute_WithTasks(t *testing.T) {
	mockAcc := newMockReactorAccessor()

	task1, _ := mockAcc.taskManager.CreateTask("", "task1 desc", "prompt1")
	task1.Metadata = map[string]any{"subagent_name": "@agent1"}
	_, _ = mockAcc.taskManager.CreateTask("", "task2 desc", "prompt2")

	tool := NewSubAgentListTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty list result")
	}
}

// ============================================================
// SubAgentResult Tool Tests
// ============================================================

func TestSubAgentResultTool_Info(t *testing.T) {
	tool := NewSubAgentResultTool()
	info := tool.Info()
	if info.Name != "subagent_result" {
		t.Errorf("Name = %q, want %q", info.Name, "subagent_result")
	}
}

func TestSubAgentResultTool_Execute_MissingTaskID(t *testing.T) {
	tool := NewSubAgentResultTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing task_id")
	}
}

func TestSubAgentResultTool_Execute_NilAccessor(t *testing.T) {
	tool := NewSubAgentResultTool()
	_, err := tool.Execute(context.Background(), map[string]any{"task_id": "t1"})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestSubAgentResultTool_Execute_PendingWithResult(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	resultCh := make(chan any, 1)
	resultCh <- "subagent completed successfully"
	mockAcc.pending["task-1"] = resultCh

	tool := NewSubAgentResultTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{"task_id": "task-1"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result from pending subagent")
	}
}

func TestSubAgentResultTool_Execute_NotFound(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewSubAgentResultTool()
	tool.SetAccessor(mockAcc)

	_, err := tool.Execute(context.Background(), map[string]any{"task_id": "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent subagent task")
	}
}

// ============================================================
// TaskCreateTool Tests
// ============================================================

func TestTaskCreateTool_Info(t *testing.T) {
	tool := NewTaskCreateTool()
	info := tool.Info()
	if info.Name != "task_create" {
		t.Errorf("Name = %q, want %q", info.Name, "task_create")
	}
}

func TestTaskCreateTool_Execute_MissingParams(t *testing.T) {
	tool := NewTaskCreateTool()
	_, err := tool.Execute(context.Background(), map[string]any{"description": "desc only"})
	if err == nil {
		t.Error("expected error for missing prompt")
	}
}

func TestTaskCreateTool_Execute_NilAccessor(t *testing.T) {
	tool := NewTaskCreateTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"description": "test desc", "prompt": "test prompt",
	})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestTaskCreateTool_Execute_Success(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	mockAcc.runInlineFn = func(_ context.Context, prompt string) (string, error) {
		return "task output: " + prompt, nil
	}

	tool := NewTaskCreateTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{
		"description": "Build API server",
		"prompt":      "Create REST endpoints for users",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result")
	}
}

// ============================================================
// TaskResultTool Tests
// ============================================================

func TestTaskResultTool_Info(t *testing.T) {
	tool := NewTaskResultTool()
	info := tool.Info()
	if info.Name != "task_result" {
		t.Errorf("Name = %q, want %q", info.Name, "task_result")
	}
}

func TestTaskResultTool_Execute_MissingTaskID(t *testing.T) {
	tool := NewTaskResultTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing task_id")
	}
}

func TestTaskResultTool_Execute_NilAccessor(t *testing.T) {
	tool := NewTaskResultTool()
	_, err := tool.Execute(context.Background(), map[string]any{"task_id": "t1"})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestTaskResultTool_Execute_NotFound(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewTaskResultTool()
	tool.SetAccessor(mockAcc)

	_, err := tool.Execute(context.Background(), map[string]any{"task_id": "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestTaskResultTool_Execute_Found(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	task, _ := mockAcc.taskManager.CreateTask("", "test task", "input data")
	mockAcc.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusCompleted, "output here", "")

	tool := NewTaskResultTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{"task_id": task.ID})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result for found task")
	}
}

// ============================================================
// TaskListTool Tests
// ============================================================

func TestTaskListTool_Info(t *testing.T) {
	tool := NewTaskListTool()
	info := tool.Info()
	if info.Name != "task_list" {
		t.Errorf("Name = %q, want %q", info.Name, "task_list")
	}
}

func TestTaskListTool_Execute_NilAccessor(t *testing.T) {
	tool := NewTaskListTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestTaskListTool_Execute_Empty(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewTaskListTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result for empty task list")
	}
}

func TestTaskListTool_Execute_WithTasks(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	mockAcc.taskManager.CreateTask("", "task A", "input A")
	mockAcc.taskManager.CreateTask("", "task B", "input B")

	tool := NewTaskListTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result with tasks")
	}
}

// ============================================================
// TeamCreateTool Tests
// ============================================================

func TestTeamCreateTool_Info(t *testing.T) {
	tool := NewTeamCreateTool()
	info := tool.Info()
	if info.Name != "team_create" {
		t.Errorf("Name = %q, want %q", info.Name, "team_create")
	}
}

func TestTeamCreateTool_Execute_MissingParams(t *testing.T) {
	tool := NewTeamCreateTool()
	_, err := tool.Execute(context.Background(), map[string]any{"name": "only-name"})
	if err == nil {
		t.Error("expected error for missing description")
	}
}

func TestTeamCreateTool_Execute_NilAccessor(t *testing.T) {
	tool := NewTeamCreateTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"name": "dev-team", "description": "build product",
	})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestTeamCreateTool_Execute_Success(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewTeamCreateTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{
		"name":        "dev-team",
		"description": "build the product",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result after creating team")
	}
}

// ============================================================
// SendMessageTool Tests
// ============================================================

func TestSendMessageTool_Info(t *testing.T) {
	tool := NewSendMessageTool()
	info := tool.Info()
	if info.Name != "send_message" {
		t.Errorf("Name = %q, want %q", info.Name, "send_message")
	}
}

func TestSendMessageTool_Execute_MissingParams(t *testing.T) {
	tool := NewSendMessageTool()
	_, err := tool.Execute(context.Background(), map[string]any{"type": "message"})
	if err == nil {
		t.Error("expected error for missing required params")
	}
}

func TestSendMessageTool_Execute_NilAccessor(t *testing.T) {
	tool := NewSendMessageTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"type": "message", "content": "hello", "summary": "greeting",
	})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestSendMessageTool_Execute_NoTeam(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewSendMessageTool()
	tool.SetAccessor(mockAcc)

	_, err := tool.Execute(context.Background(), map[string]any{
		"type": "message", "content": "hello", "summary": "greeting",
	})
	if err == nil {
		t.Error("expected error when agent has no team")
	}
}

func TestSendMessageTool_Execute_DirectMessage(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	team, _ := mockAcc.messageBus.CreateTeam("my-team", "test team")
	mockAcc.messageBus.JoinTeam(team.ID, "sender", "task-1")
	mockAcc.messageBus.JoinTeam(team.ID, "receiver", "task-2")

	tool := NewSendMessageTool()
	tool.SetAccessor(mockAcc)
	tool.SetAgentIdentity("sender", team.ID)

	result, err := tool.Execute(context.Background(), map[string]any{
		"type":    "message",
		"to":      "receiver",
		"content": "hello there",
		"summary": "greeting",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result after sending message")
	}
}

// ============================================================
// ReceiveMessagesTool Tests
// ============================================================

func TestReceiveMessagesTool_Info(t *testing.T) {
	tool := NewReceiveMessagesTool()
	info := tool.Info()
	if info.Name != "receive_messages" {
		t.Errorf("Name = %q, want %q", info.Name, "receive_messages")
	}
}

func TestReceiveMessagesTool_Execute_NilAccessor(t *testing.T) {
	tool := NewReceiveMessagesTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestReceiveMessagesTool_Execute_NoMessages(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	tool := NewReceiveMessagesTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s != "No new messages." {
		t.Errorf("expected 'No new messages.', got %q", s)
	}
}

func TestReceiveMessagesTool_Execute_HasMessages(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	team, _ := mockAcc.messageBus.CreateTeam("msg-team", "msg test")
	mockAcc.messageBus.JoinTeam(team.ID, "agent-a", "t1")
	mockAcc.messageBus.JoinTeam(team.ID, "agent-b", "t2")

	_, _ = mockAcc.messageBus.SendMessage(team.ID, "agent-b", "agent-a", core.MessageDirect, "hello there", "greeting")

	tool := NewReceiveMessagesTool()
	tool.SetAccessor(mockAcc)
	tool.SetAgentIdentity("agent-a")

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result with messages")
	}
}

// ============================================================
// TeamStatusTool Tests
// ============================================================

func TestTeamStatusTool_Info(t *testing.T) {
	tool := NewTeamStatusTool()
	info := tool.Info()
	if info.Name != "team_status" {
		t.Errorf("Name = %q, want %q", info.Name, "team_status")
	}
}

func TestTeamStatusTool_Execute_MissingTeamID(t *testing.T) {
	tool := NewTeamStatusTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing team_id")
	}
}

func TestTeamStatusTool_Execute_NilAccessor(t *testing.T) {
	tool := NewTeamStatusTool()
	_, err := tool.Execute(context.Background(), map[string]any{"team_id": "t1"})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestTeamStatusTool_Execute_Success(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	team, _ := mockAcc.messageBus.CreateTeam("status-team", "for status check")
	mockAcc.messageBus.JoinTeam(team.ID, "member-1", "task-1")

	tool := NewTeamStatusTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{"team_id": team.ID})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty team status result")
	}
}

// ============================================================
// TeamDeleteTool Tests
// ============================================================

func TestTeamDeleteTool_Info(t *testing.T) {
	tool := NewTeamDeleteTool()
	info := tool.Info()
	if info.Name != "team_delete" {
		t.Errorf("Name = %q, want %q", info.Name, "team_delete")
	}
}

func TestTeamDeleteTool_Execute_MissingTeamID(t *testing.T) {
	tool := NewTeamDeleteTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing team_id")
	}
}

func TestTeamDeleteTool_Execute_Success(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	team, _ := mockAcc.messageBus.CreateTeam("del-team", "to be deleted")

	tool := NewTeamDeleteTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{"team_id": team.ID})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result after deleting team")
	}
}

// ============================================================
// WaitTeamTool Tests
// ============================================================

func TestWaitTeamTool_Info(t *testing.T) {
	tool := NewWaitTeamTool()
	info := tool.Info()
	if info.Name != "wait_team" {
		t.Errorf("Name = %q, want %q", info.Name, "wait_team")
	}
}

func TestWaitTeamTool_Execute_MissingTeamID(t *testing.T) {
	tool := NewWaitTeamTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing team_id")
	}
}

func TestWaitTeamTool_Execute_NilAccessor(t *testing.T) {
	tool := NewWaitTeamTool()
	_, err := tool.Execute(context.Background(), map[string]any{"team_id": "t1"})
	if err == nil {
		t.Error("expected error when accessor is nil")
	}
}

func TestWaitTeamTool_Execute_AllDone(t *testing.T) {
	mockAcc := newMockReactorAccessor()
	team, _ := mockAcc.messageBus.CreateTeam("done-team", "all done")
	mockAcc.messageBus.JoinTeam(team.ID, "agent-1", "task-1")
	mockAcc.messageBus.UpdateMemberStatus(team.ID, "agent-1", "completed", "result here")

	tool := NewWaitTeamTool()
	tool.SetAccessor(mockAcc)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, map[string]any{"team_id": team.ID, "timeout_seconds": float64(1)})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result when all members are done")
	}
}
