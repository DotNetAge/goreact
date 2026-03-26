package builder

import (
	"testing"

	"github.com/DotNetAge/goreact/pkg/prompt/compression"
	"github.com/DotNetAge/goreact/pkg/prompt/formatter"
)

func TestNew(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("Expected non-nil builder")
	}
	if b.systemTemplate != DefaultSystemTemplate {
		t.Error("Expected default system template")
	}
	if b.userTemplate != DefaultUserTemplate {
		t.Error("Expected default user template")
	}
	if b.toolFormatter == nil {
		t.Error("Expected tool formatter to be set")
	}
	if b.historyFormatter == nil {
		t.Error("Expected history formatter to be set")
	}
}

func TestFluentPromptBuilder_WithSystemPrompt(t *testing.T) {
	b := New().WithSystemPrompt("custom system")
	if b.systemPrompt != "custom system" {
		t.Errorf("Expected 'custom system', got %q", b.systemPrompt)
	}
}

func TestFluentPromptBuilder_WithSystemTemplate(t *testing.T) {
	b := New().WithSystemTemplate("custom {{.task}}")
	if b.systemTemplate != "custom {{.task}}" {
		t.Errorf("Expected 'custom {{.task}}', got %q", b.systemTemplate)
	}
}

func TestFluentPromptBuilder_WithUserTemplate(t *testing.T) {
	b := New().WithUserTemplate("user {{.task}}")
	if b.userTemplate != "user {{.task}}" {
		t.Errorf("Expected 'user {{.task}}', got %q", b.userTemplate)
	}
}

func TestFluentPromptBuilder_WithTask(t *testing.T) {
	b := New().WithTask("calculate 2+2")
	if b.task != "calculate 2+2" {
		t.Errorf("Expected 'calculate 2+2', got %q", b.task)
	}
}

func TestFluentPromptBuilder_WithTools(t *testing.T) {
	tools := []formatter.ToolDesc{
		{Name: "calc", Description: "calculator"},
	}
	b := New().WithTools(tools)
	if len(b.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(b.tools))
	}
}

func TestFluentPromptBuilder_WithHistory(t *testing.T) {
	history := []Turn{
		{Role: "user", Content: "hello"},
	}
	b := New().WithHistory(history)
	if len(b.history) != 1 {
		t.Errorf("Expected 1 history, got %d", len(b.history))
	}
}

func TestFluentPromptBuilder_WithFewShots(t *testing.T) {
	examples := []FewShotExample{
		{Task: "add 1+1", Thought: "use calculator", Action: "calc"},
	}
	b := New().WithFewShots(examples)
	if len(b.fewShots) != 1 {
		t.Errorf("Expected 1 few-shot, got %d", len(b.fewShots))
	}
}

func TestFluentPromptBuilder_WithVariable(t *testing.T) {
	b := New().WithVariable("key", "value")
	if b.variables["key"] != "value" {
		t.Errorf("Expected 'value', got %v", b.variables["key"])
	}
}

func TestFluentPromptBuilder_WithMaxTokens(t *testing.T) {
	b := New().WithMaxTokens(2000)
	if b.maxTokens != 2000 {
		t.Errorf("Expected 2000, got %d", b.maxTokens)
	}
}

func TestFluentPromptBuilder_WithTokenCounter(t *testing.T) {
	counter := &mockCounter{}
	b := New().WithTokenCounter(counter)
	if b.tokenCounter == nil {
		t.Error("Expected token counter to be set")
	}
}

func TestFluentPromptBuilder_WithCompression(t *testing.T) {
	strategy := compression.NewTruncateStrategy()
	b := New().WithCompression(strategy)
	if b.compressionStrategy == nil {
		t.Error("Expected compression strategy to be set")
	}
}

func TestFluentPromptBuilder_Build(t *testing.T) {
	b := New().
		WithSystemPrompt("You are a helper").
		WithTask("do something")

	p := b.Build()
	if p.System != "You are a helper" {
		t.Errorf("Expected system prompt 'You are a helper', got %q", p.System)
	}
	if p.User == "" {
		t.Error("Expected user prompt to be set")
	}
}

func TestFluentPromptBuilder_BuildWithCompression(t *testing.T) {
	history := []Turn{
		{Role: "user", Content: "old message"},
	}
	counter := &mockCounter{}

	b := New().
		WithHistory(history).
		WithTask("new task").
		WithMaxTokens(100).
		WithTokenCounter(counter).
		WithCompression(compression.NewTruncateStrategy())

	p := b.Build()
	if p.System == "" {
		t.Error("Expected system prompt to be set")
	}
}

func TestFluentPromptBuilder_prepareVariables(t *testing.T) {
	b := New().WithTask("test task")
	vars := b.prepareVariables()

	if vars["task"] != "test task" {
		t.Errorf("Expected 'test task', got %v", vars["task"])
	}
	if vars["tools_count"] != 0 {
		t.Errorf("Expected 0 tools_count, got %v", vars["tools_count"])
	}
}

func TestFluentPromptBuilder_formatTools(t *testing.T) {
	b := New()
	result := b.formatTools()
	if result != "No tools available" {
		t.Errorf("Expected 'No tools available', got %q", result)
	}

	b = New().WithTools([]formatter.ToolDesc{
		{Name: "calc", Description: "calculator"},
	})
	result = b.formatTools()
	if result == "" {
		t.Error("Expected non-empty tools format")
	}
}

func TestFluentPromptBuilder_formatHistory(t *testing.T) {
	b := New()
	result := b.formatHistory()
	if result != "" {
		t.Errorf("Expected empty, got %q", result)
	}

	b = New().WithHistory([]Turn{{Role: "user", Content: "hello"}})
	result = b.formatHistory()
	if result == "" {
		t.Error("Expected non-empty history format")
	}
}

func TestFluentPromptBuilder_formatFewShots(t *testing.T) {
	b := New()
	result := b.formatFewShots()
	if result != "" {
		t.Errorf("Expected empty, got %q", result)
	}

	b = New().WithFewShots([]FewShotExample{
		{Task: "add", Thought: "think", Action: "calc", Parameters: map[string]any{"a": 1}, Result: "2"},
	})
	result = b.formatFewShots()
	if result == "" {
		t.Error("Expected non-empty few-shots format")
	}
}

func TestFluentPromptBuilder_renderTemplate(t *testing.T) {
	b := New()
	result := b.renderTemplate("Hello {{.name}}", map[string]any{"name": "World"})
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", result)
	}

	result = b.renderTemplate("Invalid {{.template", map[string]any{})
	if result != "Invalid {{.template" {
		t.Errorf("Expected original template on error, got %q", result)
	}
}

func TestSimpleHistoryFormatter_Format(t *testing.T) {
	f := NewSimpleHistoryFormatter()

	result := f.Format(nil)
	if result != "" {
		t.Errorf("Expected empty, got %q", result)
	}

	result = f.Format([]Turn{})
	if result != "" {
		t.Errorf("Expected empty, got %q", result)
	}

	result = f.Format([]Turn{{Role: "user", Content: "hello"}})
	if result == "" {
		t.Error("Expected non-empty format")
	}
}

func TestConversationalFormatter_Format(t *testing.T) {
	f := NewConversationalFormatter()

	result := f.Format([]Turn{{Role: "user", Content: "hello"}})
	if result == "" {
		t.Error("Expected non-empty format")
	}
}

func TestMarkdownHistoryFormatter_Format(t *testing.T) {
	f := NewMarkdownHistoryFormatter()

	result := f.Format([]Turn{{Role: "user", Content: "hello"}})
	if result == "" {
		t.Error("Expected non-empty format")
	}
}

type mockCounter struct{}

func (m *mockCounter) Count(text string) int {
	return 100
}