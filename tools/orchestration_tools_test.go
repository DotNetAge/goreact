package tools

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/core"
)

// --- Mock OrchestrationAccessor for testing ---

type mockOrchestratorAccessor struct {
	taskManager *core.InMemoryTaskManager
	runInlineFn func(ctx context.Context, prompt string) (string, error)
}

func newMockOrchestratorAccessor() *mockOrchestratorAccessor {
	return &mockOrchestratorAccessor{
		taskManager: core.NewInMemoryTaskManager(),
	}
}

// mockAgentOrchestrator provides a minimal AgentOrchestrator for testing.
type mockAgentOrchestrator struct {
	tm *core.InMemoryTaskManager
}

func (m *mockAgentOrchestrator) DelegateTo(_ context.Context, _, taskPrompt, _ string, _ map[string]any) (*core.DelegateResult, error) {
	// Create a task and complete it inline for testing
	task, _ := m.tm.CreateTask("", taskPrompt, taskPrompt)
	_ = m.tm.UpdateTaskStatus(task.ID, core.TaskStatusCompleted, "mock result: "+taskPrompt, "")
	resultCh := make(chan any, 1)
	resultCh <- "mock result: " + taskPrompt
	return &core.DelegateResult{TaskID: task.ID, ResultCh: resultCh}, nil
}
func (m *mockAgentOrchestrator) WaitForResult(_ context.Context, taskID string) (*core.Task, error) {
	return m.tm.GetTask(taskID)
}
func (m *mockAgentOrchestrator) ListAgents() []string                              { return nil }
func (m *mockAgentOrchestrator) AgentInfo(_ string) *core.AgentConfig              { return nil }
func (m *mockAgentOrchestrator) ListTasks(parentID string) ([]*core.Task, error)    { return m.tm.ListAllTasks() }
func (m *mockAgentOrchestrator) GetTask(taskID string) (*core.Task, error)           { return m.tm.GetTask(taskID) }

func (m *mockOrchestratorAccessor) Orchestrator() core.AgentOrchestrator {
	return &mockAgentOrchestrator{tm: m.taskManager}
}
func (m *mockOrchestratorAccessor) EventEmitter() func(core.ReactEvent) { return nil }
func (m *mockOrchestratorAccessor) RunInline(_ context.Context, prompt string) (string, error) {
	if m.runInlineFn != nil {
		return m.runInlineFn(context.Background(), prompt)
	}
	return "inline result for: " + prompt, nil
}
func (m *mockOrchestratorAccessor) Config() ReactorConfig { return ReactorConfig{} }

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
	mockAcc := newMockOrchestratorAccessor()
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
	mockAcc := newMockOrchestratorAccessor()

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

func TestSubAgentResultTool_Execute_Completed(t *testing.T) {
	mockAcc := newMockOrchestratorAccessor()
	task, _ := mockAcc.taskManager.CreateTask("", "test subagent", "prompt")
	mockAcc.taskManager.UpdateTaskStatus(task.ID, core.TaskStatusCompleted, "subagent completed successfully", "")

	tool := NewSubAgentResultTool()
	tool.SetAccessor(mockAcc)

	result, err := tool.Execute(context.Background(), map[string]any{"task_id": task.ID})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result from completed subagent")
	}
}

func TestSubAgentResultTool_Execute_NotFound(t *testing.T) {
	mockAcc := newMockOrchestratorAccessor()
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
	mockAcc := newMockOrchestratorAccessor()
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
	mockAcc := newMockOrchestratorAccessor()
	tool := NewTaskResultTool()
	tool.SetAccessor(mockAcc)

	_, err := tool.Execute(context.Background(), map[string]any{"task_id": "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestTaskResultTool_Execute_Found(t *testing.T) {
	mockAcc := newMockOrchestratorAccessor()
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
	mockAcc := newMockOrchestratorAccessor()
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
	mockAcc := newMockOrchestratorAccessor()
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
