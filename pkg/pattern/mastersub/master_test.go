package mastersub

import (
	"context"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/mock"
	"github.com/DotNetAge/goreact/pkg/thinker"
)

func TestNewMaster(t *testing.T) {
	tk := thinker.Default(mock.NewMockClient([]string{}))
	master := NewMaster(tk)

	if master == nil {
		t.Fatal("Expected non-nil master")
	}
	if master.thinker != tk {
		t.Error("Expected thinker to be set")
	}
}

func TestMaster_Decompose(t *testing.T) {
	masterResponse := `[
		{"id": "t1", "title": "Task 1", "description": "Do thing 1", "dependencies": [], "is_composite": false},
		{"id": "t2", "title": "Task 2", "description": "Do thing 2", "dependencies": ["t1"], "is_composite": true}
	]`

	mockClient := mock.NewMockClient([]string{masterResponse})
	tk := thinker.Default(mockClient)
	master := NewMaster(tk)

	tasks, err := master.Decompose(context.Background(), "Do something", "skill1, skill2")
	if err != nil {
		t.Fatalf("Decompose failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].ID != "t1" {
		t.Errorf("Expected first task ID 't1', got %q", tasks[0].ID)
	}

	if tasks[1].Dependencies[0] != "t1" {
		t.Errorf("Expected t2 to depend on t1, got %v", tasks[1].Dependencies)
	}
}

func TestMaster_Decompose_InvalidJSON(t *testing.T) {
	mockClient := mock.NewMockClient([]string{"not valid json"})
	tk := thinker.Default(mockClient)
	master := NewMaster(tk)

	_, err := master.Decompose(context.Background(), "test goal", "")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestMaster_Decompose_ThinkerError(t *testing.T) {
	mockClient := mock.NewMockClient([]string{"[{\"id\": \"t1\"}]"})
	tk := thinker.Default(mockClient)
	master := NewMaster(tk)

	_, err := master.Decompose(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestMaster_Replan(t *testing.T) {
	tk := thinker.Default(mock.NewMockClient([]string{}))
	master := NewMaster(tk)

	currentTasks := []Task{
		{ID: "t1", Title: "Task 1", Description: "Do thing 1", Status: TaskFailed},
	}

	replanTasks, err := master.Replan(context.Background(), currentTasks, "Task t1 failed")
	if err != nil {
		t.Fatalf("Replan failed: %v", err)
	}

	if replanTasks != nil {
		t.Error("Expected nil tasks (not implemented)")
	}
}

func TestTask_Properties(t *testing.T) {
	task := Task{
		ID:          "test-id",
		Title:       "Test Task",
		Description: "Test description",
		Dependencies: []string{"dep1", "dep2"},
		Status:      TaskPending,
		Input:       map[string]any{"key": "value"},
		Output:      "result",
		IsComposite: true,
		SkillName:   "test-skill",
	}

	if task.ID != "test-id" {
		t.Errorf("Expected 'test-id', got %q", task.ID)
	}
	if task.Status != TaskPending {
		t.Errorf("Expected TaskPending, got %v", task.Status)
	}
	if task.Input["key"] != "value" {
		t.Errorf("Expected 'value', got %v", task.Input["key"])
	}
}

func TestTaskResult_Properties(t *testing.T) {
	result := TaskResult{
		TaskID:   "t1",
		Success:  true,
		Answer:   "answer text",
		Traces:   []core.Trace{{Thought: "thought 1"}, {Thought: "thought 2"}},
		Duration: 100 * time.Millisecond,
	}

	if result.TaskID != "t1" {
		t.Errorf("Expected 't1', got %q", result.TaskID)
	}
	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if len(result.Traces) != 2 {
		t.Errorf("Expected 2 traces, got %d", len(result.Traces))
	}
}

func TestTaskStatus_Constants(t *testing.T) {
	if TaskPending != "pending" {
		t.Errorf("Expected 'pending', got %q", TaskPending)
	}
	if TaskRunning != "running" {
		t.Errorf("Expected 'running', got %q", TaskRunning)
	}
	if TaskSuccess != "success" {
		t.Errorf("Expected 'success', got %q", TaskSuccess)
	}
	if TaskFailed != "failed" {
		t.Errorf("Expected 'failed', got %q", TaskFailed)
	}
	if TaskSkipped != "skipped" {
		t.Errorf("Expected 'skipped', got %q", TaskSkipped)
	}
}

func TestSubReactor_Interface(t *testing.T) {
	var _ SubReactor = (*mockSubReactorForTest)(nil)
}

type mockSubReactorForTest struct{}

func (m *mockSubReactorForTest) Execute(ctx context.Context, task Task) (TaskResult, error) {
	return TaskResult{}, nil
}