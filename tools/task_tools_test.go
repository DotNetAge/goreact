package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/core"
)

func newTestKVStore(t *testing.T) (core.KVStore, func()) {
	tmpDir := t.TempDir()
	store, err := core.NewFileSystemKVStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create KVStore: %v", err)
	}
	return store, func() {}
}

func withKVStoreContext(ctx context.Context, kv core.KVStore, sessionID string) context.Context {
	toolCtx := &core.ToolContext{
		KVStore:   kv,
		SessionID: sessionID,
		EmitEvent: func(e core.ReactEvent) {},
	}
	return core.WithToolContext(ctx, toolCtx)
}

func TestTaskCreateTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	spawnFunc := func(ctx context.Context, agentName, task string) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "result for " + task, nil
	}

	tool := NewTaskCreateTool(spawnFunc)

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-1")

	params := map[string]any{
		"task_description": "Analyze data",
		"agent_name":       "data-analyst",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Execute() result is not map[string]any")
	}

	if _, ok := resultMap["task_id"]; !ok {
		t.Errorf("Execute() missing task_id")
	}
	if status, _ := resultMap["status"].(string); status != "running" {
		t.Errorf("Execute() status = %q, want 'running'", status)
	}

	time.Sleep(200 * time.Millisecond)

	task, err := GetTask(ctx, "test-session-1", resultMap["task_id"].(string))
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.Status != TaskCompleted {
		t.Errorf("Task status = %v, want %v", task.Status, TaskCompleted)
	}
	if task.Result != "result for Analyze data" {
		t.Errorf("Task result = %q, want 'result for Analyze data'", task.Result)
	}
}

func TestTaskListTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-2")

	task1 := &Task{
		ID:          "task-1",
		Type:        TaskTypeAgent,
		Description: "Task one",
		Status:      TaskCompleted,
		AgentName:   "agent-a",
	}
	task2 := &Task{
		ID:          "task-2",
		Type:        TaskTypeAgent,
		Description: "Task two",
		Status:      TaskRunning,
		AgentName:   "agent-b",
	}

	if err := CreateTask(ctx, "test-session-2", task1); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := CreateTask(ctx, "test-session-2", task2); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	tool := NewTaskListTool()
	result, err := tool.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap := result.(map[string]any)
	tasks := resultMap["tasks"].([]map[string]any)
	if len(tasks) != 2 {
		t.Errorf("List returned %d tasks, want 2", len(tasks))
	}
}

func TestTaskGetTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-3")

	task := &Task{
		ID:          "task-abc",
		Type:        TaskTypeAgent,
		Description: "Test task",
		Status:      TaskCompleted,
		AgentName:   "agent-x",
		Result:      "done",
	}
	if err := CreateTask(ctx, "test-session-3", task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	tool := NewTaskGetTool()
	params := map[string]any{"task_id": "task-abc"}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap := result.(map[string]any)
	if resultMap["task_id"] != "task-abc" {
		t.Errorf("task_id = %v, want 'task-abc'", resultMap["task_id"])
	}
	if resultMap["status"] != "completed" {
		t.Errorf("status = %v, want 'completed'", resultMap["status"])
	}
}

func TestTaskUpdateTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-4")

	task := &Task{
		ID:          "task-update",
		Type:        TaskTypeAgent,
		Description: "Original description",
		Status:      TaskRunning,
	}
	if err := CreateTask(ctx, "test-session-4", task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	tool := NewTaskUpdateTool()
	params := map[string]any{
		"task_id":     "task-update",
		"description": "Updated description",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap := result.(map[string]any)
	if resultMap["success"] != true {
		t.Errorf("success = %v, want true", resultMap["success"])
	}

	task, err = GetTask(ctx, "test-session-4", "task-update")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.Description != "Updated description" {
		t.Errorf("Description = %q, want 'Updated description'", task.Description)
	}
}

func TestTaskStopTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-5")

	task := &Task{
		ID:          "task-stop",
		Type:        TaskTypeAgent,
		Description: "Stop me",
		Status:      TaskRunning,
	}
	if err := CreateTask(ctx, "test-session-5", task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	tool := NewTaskStopTool()
	params := map[string]any{"task_id": "task-stop"}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap := result.(map[string]any)
	if resultMap["success"] != true {
		t.Errorf("success = %v, want true", resultMap["success"])
	}

	task, err = GetTask(ctx, "test-session-5", "task-stop")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.Status != TaskStopped {
		t.Errorf("Status = %v, want %v", task.Status, TaskStopped)
	}
}

func TestTeamCreateTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	spawnFunc := func(ctx context.Context, agentName, task string) (string, error) {
		return "team result", nil
	}

	tool := NewTeamCreateTool(spawnFunc)

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-6")

	params := map[string]any{
		"team_name":   "data-team",
		"description": "Analyze customer data",
		"leader":      "coordinator",
		"members":     []any{"analyst-1", "analyst-2"},
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap := result.(map[string]any)
	if resultMap["team_name"] != "data-team" {
		t.Errorf("team_name = %v, want 'data-team'", resultMap["team_name"])
	}
	if resultMap["leader"] != "coordinator" {
		t.Errorf("leader = %v, want 'coordinator'", resultMap["leader"])
	}
}

func TestTeamListTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-7")

	team1 := &Team{
		Name:    "team-alpha",
		Leader:  "leader-1",
		Members: []string{"member-1", "member-2"},
		Status:  "active",
	}
	team2 := &Team{
		Name:    "team-beta",
		Leader:  "leader-2",
		Members: []string{"member-3"},
		Status:  "active",
	}

	if err := CreateTeam(ctx, "test-session-7", team1); err != nil {
		t.Fatalf("CreateTeam() error = %v", err)
	}
	if err := CreateTeam(ctx, "test-session-7", team2); err != nil {
		t.Fatalf("CreateTeam() error = %v", err)
	}

	tool := NewTeamListTool()
	result, err := tool.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap := result.(map[string]any)
	teams := resultMap["teams"].([]map[string]any)
	if len(teams) != 2 {
		t.Errorf("List returned %d teams, want 2", len(teams))
	}
}

func TestTeamDeleteTool_Execute(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx := context.Background()
	ctx = withKVStoreContext(ctx, kv, "test-session-8")

	team := &Team{
		Name:    "team-to-delete",
		Leader:  "leader",
		Members: []string{"member"},
		Status:  "active",
	}
	if err := CreateTeam(ctx, "test-session-8", team); err != nil {
		t.Fatalf("CreateTeam() error = %v", err)
	}

	tool := NewTeamDeleteTool()
	params := map[string]any{"team_name": "team-to-delete"}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultMap := result.(map[string]any)
	if resultMap["success"] != true {
		t.Errorf("success = %v, want true", resultMap["success"])
	}

	team, err = GetTeam(ctx, "test-session-8", "team-to-delete")
	if err != nil {
		t.Fatalf("GetTeam() error = %v", err)
	}
	if team != nil {
		t.Errorf("Team should be deleted, but still exists")
	}
}

func TestTaskSessionIsolation(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx1 := withKVStoreContext(context.Background(), kv, "session-a")
	ctx2 := withKVStoreContext(context.Background(), kv, "session-b")

	task := &Task{
		ID:          "shared-task",
		Type:        TaskTypeAgent,
		Description: "Isolation test",
		Status:      TaskPending,
	}

	if err := CreateTask(ctx1, "session-a", task); err != nil {
		t.Fatalf("CreateTask in session-a error = %v", err)
	}

	task, err := GetTask(ctx2, "session-b", "shared-task")
	if err != nil {
		t.Fatalf("GetTask in session-b error = %v", err)
	}
	if task != nil {
		t.Errorf("Task from session-a should not be visible in session-b")
	}
}

func TestConcurrentTaskOperations(t *testing.T) {
	kv, cleanup := newTestKVStore(t)
	defer cleanup()

	ctx := withKVStoreContext(context.Background(), kv, "test-session-concurrent")

	for i := 0; i < 10; i++ {
		task := &Task{
			ID:          fmt.Sprintf("task-concurrent-%d", i),
			Type:        TaskTypeAgent,
			Description: "Concurrent task",
			Status:      TaskPending,
		}
		if err := CreateTask(ctx, "test-session-concurrent", task); err != nil {
			t.Fatalf("CreateTask error = %v", err)
		}
	}

	taskIDs, err := ListTasks(ctx, "test-session-concurrent")
	if err != nil {
		t.Fatalf("ListTasks error = %v", err)
	}
	if len(taskIDs) != 10 {
		t.Errorf("Expected 10 tasks, got %d", len(taskIDs))
	}
}
