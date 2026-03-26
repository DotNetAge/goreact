package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestMapTool(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		tool := &MapTool{
			ToolName:        "test",
			ToolDescription: "test tool",
			Level:           LevelSafe,
			ExecuteFunc: func(ctx context.Context, input map[string]any) (any, error) {
				return "result", nil
			},
		}

		result, err := tool.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "result" {
			t.Errorf("Expected 'result', got %v", result)
		}
	})

	t.Run("nil execute func", func(t *testing.T) {
		tool := &MapTool{
			ToolName:        "test",
			ToolDescription: "test tool",
			Level:           LevelSafe,
		}

		_, err := tool.Execute(context.Background(), map[string]any{})
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("execution error", func(t *testing.T) {
		expectedErr := errors.New("exec error")
		tool := &MapTool{
			ToolName:        "test",
			ToolDescription: "test tool",
			Level:           LevelSafe,
			ExecuteFunc: func(ctx context.Context, input map[string]any) (any, error) {
				return nil, expectedErr
			},
		}

		_, err := tool.Execute(context.Background(), map[string]any{})
		if err != expectedErr {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
	})
}

func TestMapTool_Name(t *testing.T) {
	tool := &MapTool{ToolName: "mytool"}
	if tool.Name() != "mytool" {
		t.Errorf("Expected 'mytool', got %q", tool.Name())
	}
}

func TestMapTool_Description(t *testing.T) {
	tool := &MapTool{ToolDescription: "my desc"}
	if tool.Description() != "my desc" {
		t.Errorf("Expected 'my desc', got %q", tool.Description())
	}
}

func TestMapTool_SecurityLevel(t *testing.T) {
	tool := &MapTool{Level: LevelHighRisk}
	if tool.SecurityLevel() != LevelHighRisk {
		t.Errorf("Expected LevelHighRisk, got %v", tool.SecurityLevel())
	}
}

func TestExtractInput(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		type Params struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		input := map[string]any{"name": "Alice", "age": 30}
		var params Params

		err := ExtractInput(input, &params)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if params.Name != "Alice" {
			t.Errorf("Expected 'Alice', got %q", params.Name)
		}
		if params.Age != 30 {
			t.Errorf("Expected 30, got %d", params.Age)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		input := map[string]any{"name": func() {}}
		var params struct{}

		err := ExtractInput(input, &params)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestSimpleManager(t *testing.T) {
	m := NewSimpleManager()

	t.Run("register and get tool", func(t *testing.T) {
		tool := &MapTool{ToolName: "test", ToolDescription: "desc", Level: LevelSafe}
		m.Register(tool)

		retrieved, ok := m.GetTool("test")
		if !ok {
			t.Error("Expected tool to be found")
		}
		if retrieved.Name() != "test" {
			t.Errorf("Expected 'test', got %q", retrieved.Name())
		}
	})

	t.Run("get non-existent tool", func(t *testing.T) {
		_, ok := m.GetTool("nonexistent")
		if ok {
			t.Error("Expected tool to not be found")
		}
	})

	t.Run("list available tools", func(t *testing.T) {
		m := NewSimpleManager()
		tool1 := &MapTool{ToolName: "tool1", ToolDescription: "desc1", Level: LevelSafe}
		tool2 := &MapTool{ToolName: "tool2", ToolDescription: "desc2", Level: LevelSensitive}
		m.Register(tool1, tool2)

		tools, err := m.ListAvailableTools(context.Background(), "test")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(tools) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(tools))
		}
	})

	t.Run("register multiple", func(t *testing.T) {
		m := NewSimpleManager()
		tools := []Tool{
			&MapTool{ToolName: "a", ToolDescription: "a", Level: LevelSafe},
			&MapTool{ToolName: "b", ToolDescription: "b", Level: LevelSafe},
		}
		m.Register(tools...)

		retrieved, _ := m.ListAvailableTools(context.Background(), "")
		if len(retrieved) != 2 {
			t.Error("Expected 2 tools")
		}
	})
}

func TestSecurityLevel_Constants(t *testing.T) {
	if LevelSafe != 0 {
		t.Errorf("Expected LevelSafe to be 0, got %d", LevelSafe)
	}
	if LevelSensitive != 1 {
		t.Errorf("Expected LevelSensitive to be 1, got %d", LevelSensitive)
	}
	if LevelHighRisk != 2 {
		t.Errorf("Expected LevelHighRisk to be 2, got %d", LevelHighRisk)
	}
}

func TestToolInterface(t *testing.T) {
	var _ Tool = (*MapTool)(nil)
}

func TestManagerInterface(t *testing.T) {
	var _ Manager = (*SimpleManager)(nil)
}

func TestExtractInput_JSONMarshalError(t *testing.T) {
	type Params struct {
		Name chan int `json:"name"`
	}

	input := map[string]any{"name": make(chan int)}
	var params Params

	err := ExtractInput(input, &params)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestExtractInput_JSONUnmarshalError(t *testing.T) {
	type Params struct {
		Name json.Number `json:"name"`
	}

	input := map[string]any{"name": "not a number"}
	var params Params

	err := ExtractInput(input, &params)
	if err == nil {
		t.Error("Expected error")
	}
}