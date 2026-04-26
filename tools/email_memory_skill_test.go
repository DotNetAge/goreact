package tools

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/core"
)

// --- Email Tool Tests ---

func TestEmailTool_Info(t *testing.T) {
	tool := NewEmailTool(EmailConfig{})
	info := tool.Info()
	if info.Name != "email" {
		t.Errorf("Name = %q, want %q", info.Name, "email")
	}
	if info.Description == "" {
		t.Error("expected description")
	}
}

func TestEmailTool_Execute_MissingParams(t *testing.T) {
	tool := NewEmailTool(EmailConfig{})
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing params")
	}
}

func TestEmailTool_Execute_MissingTo(t *testing.T) {
	tool := NewEmailTool(EmailConfig{})
	_, err := tool.Execute(context.Background(), map[string]any{
		"subject": "test",
		"body":   "hello",
	})
	if err == nil {
		t.Error("expected error for missing to")
	}
}

func TestEmailTool_Execute_MissingSubject(t *testing.T) {
	tool := NewEmailTool(EmailConfig{})
	_, err := tool.Execute(context.Background(), map[string]any{
		"to":    "a@b.com",
		"body":  "hello",
	})
	if err == nil {
		t.Error("expected error for missing subject")
	}
}

func TestEmailTool_Execute_MissingBody(t *testing.T) {
	tool := NewEmailTool(EmailConfig{})
	_, err := tool.Execute(context.Background(), map[string]any{
		"to":      "a@b.com",
		"subject": "test",
	})
	if err == nil {
		t.Error("expected error for missing body")
	}
}

func TestEmailTool_Execute_InvalidTo(t *testing.T) {
	tool := NewEmailTool(EmailConfig{})
	_, err := tool.Execute(context.Background(), map[string]any{
		"to":      123,
		"subject": "test",
		"body":   "hello",
	})
	if err == nil {
		t.Error("expected error for non-string to")
	}
}

func TestEmailTool_Execute_AllParams(t *testing.T) {
	tool := NewEmailTool(EmailConfig{})
	_, err := tool.Execute(context.Background(), map[string]any{
		"operation": "send",
		"to":        "test@example.com",
		"subject":   "Test Subject",
		"body":      "Hello World",
	})
	if err == nil {
		t.Error("expected error (no real SMTP server)")
	}
}

// --- Memory Tool Tests (with mock Memory) ---

type mockMemoryForTools struct {
	store []core.MemoryRecord
	err   error
}

func (m *mockMemoryForTools) Store(_ context.Context, record core.MemoryRecord) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	m.store = append(m.store, record)
	return record.ID, nil
}

func (m *mockMemoryForTools) Retrieve(_ context.Context, query string, opts ...core.RetrieveOption) ([]core.MemoryRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.store, nil
}

func (m *mockMemoryForTools) Update(_ context.Context, id string, record core.MemoryRecord) error { return nil }
func (m *mockMemoryForTools) Delete(_ context.Context, id string) error                 { return nil }

func TestMemorySaveTool_Info(t *testing.T) {
	SetMemory(&mockMemoryForTools{})
	defer SetMemory(nil)

	tool := NewMemorySaveTool()
	info := tool.Info()
	if info.Name != "memory_save" {
		t.Errorf("Name = %q, want %q", info.Name, "memory_save")
	}
}

func TestMemorySaveTool_Execute(t *testing.T) {
	mockMem := &mockMemoryForTools{}
	SetMemory(mockMem)
	defer SetMemory(nil)

	tool := NewMemorySaveTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"title":   "test note",
		"content": "important information",
		"type":    "session",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty result string")
	}
	if len(mockMem.store) != 1 {
		t.Errorf("store should have 1 record, got %d", len(mockMem.store))
	}
}

func TestMemorySaveTool_Execute_NoMemorySet(t *testing.T) {
	SetMemory(nil)
	defer SetMemory(nil)

	tool := NewMemorySaveTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"title":   "test",
		"content": "content",
	})
	if err == nil {
		t.Error("expected error when memory is not configured")
	}
}

func TestMemorySearchTool_Info(t *testing.T) {
	SetMemory(&mockMemoryForTools{})
	defer SetMemory(nil)

	tool := NewMemorySearchTool()
	info := tool.Info()
	if info.Name != "memory_search" {
		t.Errorf("Name = %q, want %q", info.Name, "memory_search")
	}
}

func TestMemorySearchTool_Execute(t *testing.T) {
	mockMem := &mockMemoryForTools{
		store: []core.MemoryRecord{
			{ID: "r1", Title: "test record", Content: "hello world"},
		},
	}
	SetMemory(mockMem)
	defer SetMemory(nil)

	tool := NewMemorySearchTool()
	result, err := tool.Execute(context.Background(), map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty search result")
	}
}

func TestMemorySearchTool_Execute_MissingQuery(t *testing.T) {
	SetMemory(&mockMemoryForTools{})
	defer SetMemory(nil)

	tool := NewMemorySearchTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing query")
	}
}

// --- Skill Create Tool Tests ---

func TestSkillCreateTool_Info(t *testing.T) {
	tool := NewSkillCreateTool()
	info := tool.Info()
	if info.Name != "skill_create" {
		t.Errorf("Name = %q, want %q", info.Name, "skill_create")
	}
}

func TestSkillCreateTool_Execute_MissingRequired(t *testing.T) {
	tool := NewSkillCreateTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing required params")
	}
}

func TestSkillCreateTool_Execute_InvalidName(t *testing.T) {
	tool := NewSkillCreateTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"name":         "Invalid Name!",
		"description":  "desc",
		"instructions": "instructions",
	})
	if err == nil {
		t.Error("expected error for invalid skill name (spaces)")
	}
}

func TestIsValidSkillName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"valid-name", true},
		{"a", true},
		{"name-with-dashes-123", true},
		{"", false},
		{"UpperCase", false},
		{"123start-number", false},
		{"has spaces", false},
		{"very-long-name-that-exceeds-sixty-four-character-limit-for-skill-names-in-the-system", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSkillName(tt.name)
			if got != tt.ok {
				t.Errorf("isValidSkillName(%q) = %v, want %v", tt.name, got, tt.ok)
			}
		})
	}
}

func TestBuildSkillMarkdown(t *testing.T) {
	result := buildSkillMarkdown("my-skill", "A skill", "trigger words", "dev", "Do X then Y")
	if !containsString(result, "---") {
		t.Error("should start with frontmatter delimiter")
	}
	if !containsString(result, "name: my-skill") {
		t.Error("should contain name field")
	}
	if !containsString(result, "Do X then Y") {
		t.Error("should contain instructions")
	}
}

// --- Skill List Tool Tests ---

func TestSkillListTool_Info(t *testing.T) {
	tool := NewSkillListTool()
	info := tool.Info()
	if info.Name != "skill_list" {
		t.Errorf("Name = %q, want %q", info.Name, "skill_list")
	}
}

func TestParseSkillFrontmatter(t *testing.T) {
	content := `---
name: coder
description: Write code
trigger: code,refactor
category: development
---
Instructions here...`

	info := parseSkillFrontmatter(content)
	if info.Name != "coder" {
		t.Errorf("Name = %q, want %q", info.Name, "coder")
	}
	if info.Description != "Write code" {
		t.Errorf("Description = %q, want %q", info.Description, "Write code")
	}
	if info.Trigger != "code,refactor" {
		t.Errorf("Trigger = %q, want %q", info.Trigger, "code,refactor")
	}
	if info.Category != "development" {
		t.Errorf("Category = %q, want %q", info.Category, "development")
	}
}

func TestParseSkillFrontmatter_NoFrontmatter(t *testing.T) {
	info := parseSkillFrontmatter("just plain text without frontmatter")
	if info.Name != "" || info.Description != "" {
		t.Error("should return empty info for non-frontmatter content")
	}
}
