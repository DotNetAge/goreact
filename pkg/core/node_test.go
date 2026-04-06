package core

import (
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestNewAgentNode(t *testing.T) {
	node := NewAgentNode("assistant", "AI assistant", "general", "gpt-4")

	if node.Name != "assistant" {
		t.Errorf("Name = %q, want 'assistant'", node.Name)
	}
	if node.Description != "AI assistant" {
		t.Errorf("Description = %q, want 'AI assistant'", node.Description)
	}
	if node.Domain != "general" {
		t.Errorf("Domain = %q, want 'general'", node.Domain)
	}
	if node.Model != "gpt-4" {
		t.Errorf("Model = %q, want 'gpt-4'", node.Model)
	}
	if node.NodeType != common.NodeTypeAgent {
		t.Errorf("NodeType = %q, want %q", node.NodeType, common.NodeTypeAgent)
	}
	if node.Skills == nil {
		t.Error("Skills should not be nil")
	}
	if node.Tools == nil {
		t.Error("Tools should not be nil")
	}
}

func TestNewModelNode(t *testing.T) {
	node := NewModelNode("gpt-4", "OpenAI GPT-4", "openai")

	if node.Name != "gpt-4" {
		t.Errorf("Name = %q, want 'gpt-4'", node.Name)
	}
	if node.Provider != "openai" {
		t.Errorf("Provider = %q, want 'openai'", node.Provider)
	}
	if node.NodeType != common.NodeTypeModel {
		t.Errorf("NodeType = %q, want %q", node.NodeType, common.NodeTypeModel)
	}
	if node.Config == nil {
		t.Error("Config should not be nil")
	}
}

func TestNewSessionNode(t *testing.T) {
	node := NewSessionNode("session-123", "user-1")

	if node.Name != "session-123" {
		t.Errorf("Name = %q, want 'session-123'", node.Name)
	}
	if node.UserName != "user-1" {
		t.Errorf("UserName = %q, want 'user-1'", node.UserName)
	}
	if node.NodeType != common.NodeTypeSession {
		t.Errorf("NodeType = %q, want %q", node.NodeType, common.NodeTypeSession)
	}
	if node.Status != common.SessionStatusActive {
		t.Errorf("Status = %q, want 'active'", node.Status)
	}
}

func TestNewMessageNode(t *testing.T) {
	node := NewMessageNode("session-123", "user", "Hello!")

	if node.SessionName != "session-123" {
		t.Errorf("SessionName = %q, want 'session-123'", node.SessionName)
	}
	if node.Role != "user" {
		t.Errorf("Role = %q, want 'user'", node.Role)
	}
	if node.Content != "Hello!" {
		t.Errorf("Content = %q, want 'Hello!'", node.Content)
	}
	if node.NodeType != common.NodeTypeMessage {
		t.Errorf("NodeType = %q, want %q", node.NodeType, common.NodeTypeMessage)
	}
}

func TestNewMemoryItemNode(t *testing.T) {
	node := NewMemoryItemNode("session-123", "User prefers dark mode", common.MemoryItemTypePreference)

	if node.SessionName != "session-123" {
		t.Errorf("SessionName = %q, want 'session-123'", node.SessionName)
	}
	if node.Content != "User prefers dark mode" {
		t.Errorf("Content = %q, want 'User prefers dark mode'", node.Content)
	}
	if node.Type != common.MemoryItemTypePreference {
		t.Errorf("Type = %q, want 'preference'", node.Type)
	}
	if node.Source != common.MemorySourceUser {
		t.Errorf("Source = %q, want 'user'", node.Source)
	}
	if node.NodeType != common.NodeTypeMemoryItem {
		t.Errorf("NodeType = %q, want %q", node.NodeType, common.NodeTypeMemoryItem)
	}
}

func TestNewPendingQuestionNode(t *testing.T) {
	node := NewPendingQuestionNode("session-123", common.QuestionTypeConfirmation, "Continue?")

	if node.SessionName != "session-123" {
		t.Errorf("SessionName = %q, want 'session-123'", node.SessionName)
	}
	if node.Type != common.QuestionTypeConfirmation {
		t.Errorf("Type = %q, want 'confirmation'", node.Type)
	}
	if node.Question != "Continue?" {
		t.Errorf("Question = %q, want 'Continue?'", node.Question)
	}
	if node.QuestionStatus != common.QuestionStatusPending {
		t.Errorf("QuestionStatus = %q, want 'pending'", node.QuestionStatus)
	}
}

func TestPendingQuestionNode_WithOptions(t *testing.T) {
	node := NewPendingQuestionNode("session", common.QuestionTypeConfirmation, "Continue?")
	node.WithOptions([]string{"Yes", "No"})

	if len(node.Options) != 2 {
		t.Errorf("len(Options) = %d, want 2", len(node.Options))
	}
	if node.Options[0] != "Yes" || node.Options[1] != "No" {
		t.Errorf("Options = %v, want ['Yes', 'No']", node.Options)
	}
}

func TestPendingQuestionNode_WithDefaultAnswer(t *testing.T) {
	node := NewPendingQuestionNode("session", common.QuestionTypeConfirmation, "Continue?")
	node.WithDefaultAnswer("Yes")

	if node.DefaultAnswer != "Yes" {
		t.Errorf("DefaultAnswer = %q, want 'Yes'", node.DefaultAnswer)
	}
}

func TestPendingQuestionNode_WithExpiry(t *testing.T) {
	node := NewPendingQuestionNode("session", common.QuestionTypeConfirmation, "Continue?")
	node.WithExpiry(5 * time.Minute)

	if node.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
	if time.Now().After(node.ExpiresAt) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestPendingQuestionNode_IsExpired(t *testing.T) {
	node := NewPendingQuestionNode("session", common.QuestionTypeConfirmation, "Continue?")

	if node.IsExpired() {
		t.Error("New node should not be expired")
	}

	node.ExpiresAt = time.Now().Add(-1 * time.Hour)
	if !node.IsExpired() {
		t.Error("Node with past ExpiresAt should be expired")
	}
}

func TestPendingQuestionNode_SetAnswer(t *testing.T) {
	node := NewPendingQuestionNode("session", common.QuestionTypeConfirmation, "Continue?")
	node.SetAnswer("Yes")

	if node.Answer != "Yes" {
		t.Errorf("Answer = %q, want 'Yes'", node.Answer)
	}
	if node.QuestionStatus != common.QuestionStatusAnswered {
		t.Errorf("QuestionStatus = %q, want 'answered'", node.QuestionStatus)
	}
}

func TestPendingQuestionNode_Cancel(t *testing.T) {
	node := NewPendingQuestionNode("session", common.QuestionTypeConfirmation, "Continue?")
	node.Cancel()

	if node.QuestionStatus != common.QuestionStatusCancelled {
		t.Errorf("QuestionStatus = %q, want 'cancelled'", node.QuestionStatus)
	}
}

func TestPendingQuestionNode_StatusChecks(t *testing.T) {
	node := NewPendingQuestionNode("session", common.QuestionTypeConfirmation, "Continue?")

	if !node.IsPending() {
		t.Error("New node should be pending")
	}
	if node.IsAnswered() {
		t.Error("New node should not be answered")
	}

	node.SetAnswer("Yes")
	if node.IsPending() {
		t.Error("Answered node should not be pending")
	}
	if !node.IsAnswered() {
		t.Error("Answered node should be answered")
	}
}

func TestBaseNode_Getters(t *testing.T) {
	node := &BaseNode{
		Name:        "test",
		NodeType:    "Test",
		Description: "test description",
		CreatedAt:   time.Now(),
		Metadata:    map[string]any{"key": "value"},
	}

	if node.GetName() != "test" {
		t.Errorf("GetName() = %q, want 'test'", node.GetName())
	}
	if node.GetNodeType() != "Test" {
		t.Errorf("GetNodeType() = %q, want 'Test'", node.GetNodeType())
	}
	if node.GetDescription() != "test description" {
		t.Errorf("GetDescription() = %q, want 'test description'", node.GetDescription())
	}
	if node.GetCreatedAt().IsZero() {
		t.Error("GetCreatedAt() should not return zero time")
	}
	if node.GetMetadata()["key"] != "value" {
		t.Errorf("GetMetadata()[key] = %v, want 'value'", node.GetMetadata()["key"])
	}
}
