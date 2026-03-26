package formatter

import (
	"encoding/json"
	"testing"
)

func TestToolDesc_Properties(t *testing.T) {
	desc := ToolDesc{
		Name:        "test",
		Description: "test tool",
		Parameters: &ParameterSchema{
			Type: "object",
			Properties: map[string]*Property{
				"name": {Type: "string", Description: "param name"},
			},
			Required: []string{"name"},
		},
	}

	if desc.Name != "test" {
		t.Errorf("Expected 'test', got %q", desc.Name)
	}
	if desc.Parameters.Type != "object" {
		t.Errorf("Expected 'object', got %q", desc.Parameters.Type)
	}
}

func TestSimpleTextFormatter_Format(t *testing.T) {
	f := NewSimpleTextFormatter()

	t.Run("empty tools", func(t *testing.T) {
		result := f.Format([]ToolDesc{})
		if result != "No tools available" {
			t.Errorf("Expected 'No tools available', got %q", result)
		}
	})

	t.Run("single tool", func(t *testing.T) {
		tools := []ToolDesc{
			{Name: "calc", Description: "calculator"},
		}
		result := f.Format(tools)
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("multiple tools", func(t *testing.T) {
		tools := []ToolDesc{
			{Name: "calc", Description: "calculator"},
			{Name: "search", Description: "search engine"},
		}
		result := f.Format(tools)
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})
}

func TestJSONSchemaFormatter_Format(t *testing.T) {
	t.Run("empty tools", func(t *testing.T) {
		f := NewJSONSchemaFormatter(false)
		result := f.Format([]ToolDesc{})
		if result != "[]" {
			t.Errorf("Expected '[]', got %q", result)
		}
	})

	t.Run("with indent", func(t *testing.T) {
		f := NewJSONSchemaFormatter(true)
		tools := []ToolDesc{
			{Name: "calc", Description: "calculator"},
		}
		result := f.Format(tools)
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("tool with parameters", func(t *testing.T) {
		f := NewJSONSchemaFormatter(false)
		tools := []ToolDesc{
			{
				Name:        "calc",
				Description: "calculator",
				Parameters: &ParameterSchema{
					Type: "object",
					Properties: map[string]*Property{
						"expr": {Type: "string", Description: "expression"},
					},
					Required: []string{"expr"},
				},
			},
		}
		result := f.Format(tools)

		var parsed []map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Errorf("Failed to parse JSON: %v", err)
		}
		if len(parsed) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(parsed))
		}
	})
}

func TestJSONSchemaFormatter_formatParameters(t *testing.T) {
	f := &JSONSchemaFormatter{}

	t.Run("basic parameters", func(t *testing.T) {
		params := &ParameterSchema{
			Type: "object",
			Properties: map[string]*Property{
				"name": {Type: "string"},
			},
			Required: []string{"name"},
		}
		result := f.formatParameters(params)
		if result["type"] != "object" {
			t.Errorf("Expected 'object', got %v", result["type"])
		}
		if result["required"] == nil {
			t.Error("Expected required field")
		}
	})

	t.Run("parameters with items", func(t *testing.T) {
		params := &ParameterSchema{
			Type:  "array",
			Items: &Property{Type: "string"},
		}
		result := f.formatParameters(params)
		if result["items"] == nil {
			t.Error("Expected items field")
		}
	})

	t.Run("parameters with additional fields", func(t *testing.T) {
		params := &ParameterSchema{
			Type:       "object",
			Additional: map[string]any{"custom": "value"},
		}
		result := f.formatParameters(params)
		if result["custom"] != "value" {
			t.Errorf("Expected 'value', got %v", result["custom"])
		}
	})
}

func TestJSONSchemaFormatter_formatProperty(t *testing.T) {
	f := &JSONSchemaFormatter{}

	t.Run("basic property", func(t *testing.T) {
		prop := &Property{Type: "string"}
		result := f.formatProperty(prop)
		if result["type"] != "string" {
			t.Errorf("Expected 'string', got %v", result["type"])
		}
	})

	t.Run("property with description", func(t *testing.T) {
		prop := &Property{Type: "string", Description: "test desc"}
		result := f.formatProperty(prop)
		if result["description"] != "test desc" {
			t.Errorf("Expected 'test desc', got %v", result["description"])
		}
	})

	t.Run("property with enum", func(t *testing.T) {
		prop := &Property{Type: "string", Enum: []any{"a", "b", "c"}}
		result := f.formatProperty(prop)
		if result["enum"] == nil {
			t.Error("Expected enum field")
		}
	})

	t.Run("property with default", func(t *testing.T) {
		prop := &Property{Type: "string", Default: "default"}
		result := f.formatProperty(prop)
		if result["default"] != "default" {
			t.Errorf("Expected 'default', got %v", result["default"])
		}
	})

	t.Run("nested items", func(t *testing.T) {
		prop := &Property{
			Type:  "array",
			Items: &Property{Type: "string"},
		}
		result := f.formatProperty(prop)
		if result["items"] == nil {
			t.Error("Expected nested items")
		}
	})
}

func TestMarkdownFormatter_Format(t *testing.T) {
	f := NewMarkdownFormatter()

	t.Run("empty tools", func(t *testing.T) {
		result := f.Format([]ToolDesc{})
		if result != "No tools available" {
			t.Errorf("Expected 'No tools available', got %q", result)
		}
	})

	t.Run("tool without parameters", func(t *testing.T) {
		tools := []ToolDesc{
			{Name: "calc", Description: "calculator"},
		}
		result := f.Format(tools)
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("tool with parameters", func(t *testing.T) {
		tools := []ToolDesc{
			{
				Name:        "calc",
				Description: "calculator",
				Parameters: &ParameterSchema{
					Properties: map[string]*Property{
						"expr": {Type: "string", Description: "expression"},
					},
					Required: []string{"expr"},
				},
			},
		}
		result := f.Format(tools)
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})
}

func TestMarkdownFormatter_isRequired(t *testing.T) {
	f := &MarkdownFormatter{}

	if !f.isRequired("name", []string{"name", "age"}) {
		t.Error("Expected 'name' to be required")
	}

	if f.isRequired("height", []string{"name", "age"}) {
		t.Error("Expected 'height' to not be required")
	}
}

func TestCompactFormatter_Format(t *testing.T) {
	f := NewCompactFormatter()

	t.Run("empty tools", func(t *testing.T) {
		result := f.Format([]ToolDesc{})
		if result != "No tools" {
			t.Errorf("Expected 'No tools', got %q", result)
		}
	})

	t.Run("single tool", func(t *testing.T) {
		tools := []ToolDesc{
			{Name: "calc", Description: "calculator"},
		}
		result := f.Format(tools)
		expected := "calc(calculator)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("multiple tools", func(t *testing.T) {
		tools := []ToolDesc{
			{Name: "calc", Description: "calculator"},
			{Name: "search", Description: "search"},
		}
		result := f.Format(tools)
		expected := "calc(calculator); search(search)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
}