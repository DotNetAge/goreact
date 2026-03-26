package prompt

import (
	"testing"
)

func TestDefaultPromptManager_New(t *testing.T) {
	m := NewDefaultPromptManager()
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
	if m.templates == nil {
		t.Error("Expected templates map to be initialized")
	}
}

func TestDefaultPromptManager_RegisterTemplate(t *testing.T) {
	m := NewDefaultPromptManager()
	m.RegisterTemplate("test", "Hello {{name}}")

	if m.templates["test"] != "Hello {{name}}" {
		t.Error("Expected template to be registered")
	}
}

func TestDefaultPromptManager_GetTemplate(t *testing.T) {
	m := NewDefaultPromptManager()
	m.RegisterTemplate("test", "Hello {{name}}")

	t.Run("existing template", func(t *testing.T) {
		result := m.GetTemplate("test")
		if result != "Hello {{name}}" {
			t.Errorf("Expected 'Hello {{name}}', got %q", result)
		}
	})

	t.Run("non-existing template", func(t *testing.T) {
		result := m.GetTemplate("nonexistent")
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})
}

func TestDefaultPromptManager_RenderTemplate(t *testing.T) {
	m := NewDefaultPromptManager()
	m.RegisterTemplate("greeting", "Hello {{name}}, you have {{count}} messages")

	tests := []struct {
		name      string
		template  string
		variables map[string]any
		expected  string
	}{
		{
			name:      "simple render",
			template:  "greeting",
			variables: map[string]any{"name": "Alice", "count": 5},
			expected:  "Hello Alice, you have 5 messages",
		},
		{
			name:      "non-existing template",
			template:  "nonexistent",
			variables: map[string]any{"name": "test"},
			expected:  "",
		},
		{
			name:      "missing variable",
			template:  "greeting",
			variables: map[string]any{"name": "Bob"},
			expected:  "Hello Bob, you have {{count}} messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.RenderTemplate(tt.template, tt.variables)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
