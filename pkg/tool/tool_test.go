package tool

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/pkg/common"
)

// MockTool is a mock implementation of Tool for testing
type MockTool struct {
	*BaseTool
	runFunc func(ctx context.Context, params map[string]any) (any, error)
}

func (m *MockTool) Run(ctx context.Context, params map[string]any) (any, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, params)
	}
	return nil, nil
}

func TestNewBaseTool(t *testing.T) {
	tool := NewBaseTool("test_tool", "A test tool", common.LevelSafe, true)

	if tool.Name() != "test_tool" {
		t.Errorf("Name() = %q, want 'test_tool'", tool.Name())
	}
	if tool.Description() != "A test tool" {
		t.Errorf("Description() = %q, want 'A test tool'", tool.Description())
	}
	if tool.SecurityLevel() != common.LevelSafe {
		t.Errorf("SecurityLevel() = %v, want %v", tool.SecurityLevel(), common.LevelSafe)
	}
	if !tool.IsIdempotent() {
		t.Error("IsIdempotent() should be true")
	}
}

func TestBaseTool_Type(t *testing.T) {
	tool := NewBaseTool("test", "test", common.LevelSafe, true)

	if tool.Type() != "tool" {
		t.Errorf("Type() = %q, want 'tool'", tool.Type())
	}
}

func TestBaseTool_Properties(t *testing.T) {
	tool := NewBaseTool("test", "A test tool", common.LevelSensitive, false)
	props := tool.Properties()

	if props["description"] != "A test tool" {
		t.Errorf("Properties[description] = %v, want 'A test tool'", props["description"])
	}
	if props["security_level"] != "sensitive" {
		t.Errorf("Properties[security_level] = %v, want 'sensitive'", props["security_level"])
	}
	if props["is_idempotent"] != false {
		t.Errorf("Properties[is_idempotent] = %v, want false", props["is_idempotent"])
	}
}

func TestBaseTool_WithParameter(t *testing.T) {
	tool := NewBaseTool("test", "test", common.LevelSafe, true)
	tool.WithParameter(Parameter{
		Name:        "path",
		Type:        "string",
		Required:    true,
		Description: "File path",
	})

	params := tool.Parameters()
	if len(params) != 1 {
		t.Errorf("len(Parameters) = %d, want 1", len(params))
	}
	if params[0].Name != "path" {
		t.Errorf("Parameter name = %q, want 'path'", params[0].Name)
	}
}

func TestParameter(t *testing.T) {
	param := Parameter{
		Name:        "count",
		Type:        "integer",
		Required:    true,
		Default:     10,
		Description: "Number of items",
		Enum:        []any{1, 5, 10, 20},
	}

	if param.Name != "count" {
		t.Errorf("Name = %q, want 'count'", param.Name)
	}
	if param.Type != "integer" {
		t.Errorf("Type = %q, want 'integer'", param.Type)
	}
	if !param.Required {
		t.Error("Required should be true")
	}
	if param.Default != 10 {
		t.Errorf("Default = %v, want 10", param.Default)
	}
	if len(param.Enum) != 4 {
		t.Errorf("len(Enum) = %d, want 4", len(param.Enum))
	}
}

func TestToolInfo(t *testing.T) {
	// Create a mock tool that implements Tool interface
	mockTool := &MockTool{
		BaseTool: NewBaseTool("read_file", "Read file contents", common.LevelSafe, true),
	}
	mockTool.WithParameter(Parameter{
		Name:     "path",
		Type:     "string",
		Required: true,
	})

	info := GetToolInfo(mockTool)

	if info.Name != "read_file" {
		t.Errorf("Name = %q, want 'read_file'", info.Name)
	}
	if info.Description != "Read file contents" {
		t.Errorf("Description = %q, want 'Read file contents'", info.Description)
	}
	if info.SecurityLevel != common.LevelSafe {
		t.Errorf("SecurityLevel = %v, want safe", info.SecurityLevel)
	}
	if !info.IsIdempotent {
		t.Error("IsIdempotent should be true")
	}
}

func TestToolNode(t *testing.T) {
	node := &ToolNode{
		Name:           "bash",
		NodeType:       "Tool",
		Description:    "Execute bash commands",
		Type:           common.ToolTypeBash,
		SecurityLevel:  common.LevelHighRisk,
		IsIdempotent:   false,
		ExecutionCount: 100,
		SuccessRate:    0.95,
	}

	if node.Name != "bash" {
		t.Errorf("Name = %q, want 'bash'", node.Name)
	}
	if node.Type != common.ToolTypeBash {
		t.Errorf("Type = %q, want 'bash'", node.Type)
	}
	if node.SecurityLevel != common.LevelHighRisk {
		t.Errorf("SecurityLevel = %v, want high_risk", node.SecurityLevel)
	}
	if node.SuccessRate != 0.95 {
		t.Errorf("SuccessRate = %f, want 0.95", node.SuccessRate)
	}
}

func TestGeneratedTool(t *testing.T) {
	genTool := &GeneratedTool{
		Name:           "custom_analyzer",
		Description:    "Custom data analyzer",
		FilePath:       "/tools/custom_analyzer.py",
		Type:           common.ToolTypePython,
		SecurityLevel:  common.LevelSensitive,
		SourceSession:  "session-123",
		Status:         common.GeneratedStatusActive,
	}

	if genTool.Name != "custom_analyzer" {
		t.Errorf("Name = %q, want 'custom_analyzer'", genTool.Name)
	}
	if genTool.Type != common.ToolTypePython {
		t.Errorf("Type = %q, want 'python'", genTool.Type)
	}
	if genTool.Status != common.GeneratedStatusActive {
		t.Errorf("Status = %q, want 'active'", genTool.Status)
	}
}

func TestToolParameter(t *testing.T) {
	param := &ToolParameter{
		Name:        "input",
		Type:        "string",
		Description: "Input data",
		Required:    true,
		Default:     "",
	}

	if param.Name != "input" {
		t.Errorf("Name = %q, want 'input'", param.Name)
	}
	if param.Type != "string" {
		t.Errorf("Type = %q, want 'string'", param.Type)
	}
	if !param.Required {
		t.Error("Required should be true")
	}
}
