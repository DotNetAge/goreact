package prompt

import (
	"testing"
)

func TestPrompt_String(t *testing.T) {
	tests := []struct {
		name     string
		prompt   Prompt
		expected string
	}{
		{
			name:     "empty prompt",
			prompt:   Prompt{},
			expected: "",
		},
		{
			name:     "system only",
			prompt:   Prompt{System: "system prompt"},
			expected: "system prompt\n\n",
		},
		{
			name:     "user only",
			prompt:   Prompt{User: "user prompt"},
			expected: "user prompt",
		},
		{
			name:     "both",
			prompt:   Prompt{System: "system", User: "user"},
			expected: "system\n\nuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prompt.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPrompt_Tokens(t *testing.T) {
	t.Run("with nil counter", func(t *testing.T) {
		p := Prompt{System: "system", User: "user"}
		tokens := p.Tokens(nil)
		expected := len(p.String()) / 4
		if tokens != expected {
			t.Errorf("Expected %d, got %d", expected, tokens)
		}
	})

	t.Run("with counter", func(t *testing.T) {
		p := Prompt{System: "system", User: "user"}
		counter := &mockTokenCounter{count: 10}
		tokens := p.Tokens(counter)
		if tokens != 10 {
			t.Errorf("Expected 10, got %d", tokens)
		}
	})
}

type mockTokenCounter struct {
	count int
}

func (m *mockTokenCounter) Count(text string) int {
	return m.count
}
