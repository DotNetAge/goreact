package tools

import (
	"context"
	"testing"
)

func resetTodoStore() {
	todoStoreMu.Lock()
	defer todoStoreMu.Unlock()
	todoStore = nil
	todoCounter = 0
}

func TestTodoWriteTool_Info(t *testing.T) {
	tool := NewTodoWriteTool()
	info := tool.Info()
	if info.Name != "todo_write" {
		t.Errorf("Name = %q, want %q", info.Name, "todo_write")
	}
	if len(info.Parameters) < 2 {
		t.Errorf("expected at least 2 parameters, got %d", len(info.Parameters))
	}
}

func TestTodoWriteTool_Execute_Create(t *testing.T) {
	resetTodoStore()
	tool := NewTodoWriteTool()

	todos := `[{"content":"task 1","status":"pending","priority":1},{"content":"task 2","status":"pending","priority":0}]`
	result, err := tool.Execute(context.Background(), map[string]any{
		"todos": todos,
		"merge": false,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	m := result.(map[string]any)
	if m["success"] != true {
		t.Error("expected success=true")
	}
	if count := m["count"].(int); count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestTodoWriteTool_Execute_Merge(t *testing.T) {
	resetTodoStore()
	tool := NewTodoWriteTool()

	result1, _ := tool.Execute(context.Background(), map[string]any{
		"todos": `[{"id":"t1","content":"first","status":"pending"}]`,
		"merge": false,
	})
	m1 := result1.(map[string]any)
	if m1["count"].(int) != 1 {
		t.Fatalf("initial count = %d, want 1", m1["count"])
	}

	result2, _ := tool.Execute(context.Background(), map[string]any{
		"todos": `[{"id":"t1","content":"first updated","status":"in_progress"},{"id":"t2","content":"second","status":"pending"}]`,
		"merge": true,
	})
	m2 := result2.(map[string]any)
	if c := m2["count"].(int); c != 2 {
		t.Errorf("after merge count = %d, want 2", c)
	}
}

func TestTodoWriteTool_Execute_MissingTodos(t *testing.T) {
	resetTodoStore()
	tool := NewTodoWriteTool()
	_, err := tool.Execute(context.Background(), map[string]any{"merge": true})
	if err == nil {
		t.Error("expected error for missing todos")
	}
}

func TestTodoWriteTool_Execute_InvalidJSON(t *testing.T) {
	resetTodoStore()
	tool := NewTodoWriteTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"todos": "not json {{{",
		"merge": false,
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTodoReadTool_Info(t *testing.T) {
	tool := NewTodoReadTool()
	info := tool.Info()
	if info.Name != "todo_read" {
		t.Errorf("Name = %q, want %q", info.Name, "todo_read")
	}
}

func TestTodoReadTool_Execute_AllItems(t *testing.T) {
	resetTodoStore()
	write := NewTodoWriteTool()
	write.Execute(context.Background(), map[string]any{
		"todos": `[{"content":"read test","status":"completed"}]`,
		"merge": false,
	})

	read := NewTodoReadTool()
	result, err := read.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	m := result.(map[string]any)
	if m["count"].(int) != 1 {
		t.Errorf("count = %d, want 1", m["count"])
	}
	items := m["items"].([]TodoItem)
	if len(items) != 1 || items[0].Content != "read test" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestTodoReadTool_Execute_FilterByStatus(t *testing.T) {
	resetTodoStore()
	write := NewTodoWriteTool()
	write.Execute(context.Background(), map[string]any{
		"todos": `[{"content":"a","status":"pending"},{"content":"b","status":"completed"}]`,
		"merge": false,
	})

	read := NewTodoReadTool()
	result, _ := read.Execute(context.Background(), map[string]any{"status": "completed"})
	m := result.(map[string]any)
	if m["count"].(int) != 1 {
		t.Errorf("filtered count = %d, want 1", m["count"])
	}
}

func TestTodoReadTool_Execute_EmptyStore(t *testing.T) {
	resetTodoStore()
	read := NewTodoReadTool()
	result, err := read.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	m := result.(map[string]any)
	if m["count"].(int) != 0 {
		t.Errorf("empty store should return 0 items")
	}
}

func TestTodoExecuteTool_Info(t *testing.T) {
	tool := NewTodoExecuteTool()
	info := tool.Info()
	if info.Name != "todo_execute" {
		t.Errorf("Name = %q, want %q", info.Name, "todo_execute")
	}
}

func TestTodoExecuteTool_Execute_PendingOnly(t *testing.T) {
	resetTodoStore()
	write := NewTodoWriteTool()
	write.Execute(context.Background(), map[string]any{
		"todos": `[
			{"id":"t1","content":"done task","status":"completed"},
			{"id":"t2","content":"ready task","status":"pending","priority":1},
			{"id":"t3","content":"blocked task","status":"pending","dependencies":["t99"]}
		]`,
		"merge": false,
	})

	exec := NewTodoExecuteTool()
	result, err := exec.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	m := result.(map[string]any)
	if rc := m["ready_count"].(int); rc != 1 {
		t.Errorf("ready_count = %d, want 1 (only t2 is ready, t3 blocked by non-existent t99)", rc)
	}
	if bc := m["blocked_count"].(int); bc != 1 {
		t.Errorf("blocked_count = %d, want 1 (t3 blocked by t99)", bc)
	}
	steps := m["steps"]
	if steps == nil {
		t.Error("expected non-nil steps")
	}
}

func TestTodoExecuteTool_Execute_SortByPriority(t *testing.T) {
	resetTodoStore()
	write := NewTodoWriteTool()
	write.Execute(context.Background(), map[string]any{
		"todos": `[
			{"id":"t1","content":"low priority","status":"pending","priority":10},
			{"id":"t2","content":"high priority","status":"pending","priority":1},
			{"id":"t3","content":"medium priority","status":"pending","priority":5}
		]`,
		"merge": false,
	})

	exec := NewTodoExecuteTool()
	result, _ := exec.Execute(context.Background(), nil)
	m := result.(map[string]any)
	if rc := m["ready_count"].(int); rc != 3 {
		t.Errorf("ready_count = %d, want 3", rc)
	}
	_ = m["steps"]
}

func TestFormatTodoSummary(t *testing.T) {
	items := []TodoItem{
		{ID: "t1", Status: "pending", Content: "test item", Priority: 1, ToolCall: "write"},
		{ID: "t2", Status: "in_progress", Content: "another"},
	}
	summary := formatTodoSummary(items)
	if !containsString(summary, "[pending]") {
		t.Error("summary should contain [pending]")
	}
	if !containsString(summary, "test item") {
		t.Error("summary should contain content")
	}
}

func containsString(s, substr string) bool { return len(s) >= len(substr) && containsAny(s, substr) }
func containsAny(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
