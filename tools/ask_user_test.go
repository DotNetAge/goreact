package tools

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/core"
)

func TestAskUser_Info(t *testing.T) {
	tool := NewAskUserTool()
	info := tool.Info()

	if info.Name != "ask_user" {
		t.Errorf("expected tool name 'ask_user', got %q", info.Name)
	}
	if !info.IsReadOnly {
		t.Error("expected IsReadOnly to be true")
	}
	if len(info.Parameters) == 0 {
		t.Error("expected parameters to be defined")
	}

	var hasQuestion bool
	for _, p := range info.Parameters {
		if p.Name == "question" && p.Required {
			hasQuestion = true
		}
	}
	if !hasQuestion {
		t.Error("expected 'question' parameter to be required")
	}

	if len(info.Tags) == 0 {
		t.Error("expected Tags to be defined")
	}
}

func TestAskUser_ExecuteReturnsInteractionRequest(t *testing.T) {
	tool := NewAskUserTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"question": "What is your name?",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if m["status"] != "waiting_for_user" {
		t.Errorf("expected status 'waiting_for_user', got %v", m["status"])
	}

	interaction, ok := m["_interaction"].(*core.InteractionRequest)
	if !ok {
		t.Fatalf("expected _interaction to be *core.InteractionRequest, got %T", m["_interaction"])
	}

	if interaction.Type != core.InteractionAskUser {
		t.Errorf("expected type %s, got %s", core.InteractionAskUser, interaction.Type)
	}
	if interaction.Question != "What is your name?" {
		t.Errorf("expected question 'What is your name?', got %s", interaction.Question)
	}
	if interaction.ToolName != "ask_user" {
		t.Errorf("expected tool_name 'ask_user', got %s", interaction.ToolName)
	}
}

func TestAskUser_ExecuteIsNonBlocking(t *testing.T) {
	tool := NewAskUserTool()

	done := make(chan struct{})
	go func() {
		defer close(done)
		tool.Execute(context.Background(), map[string]any{"question": "test"})
	}()

	select {
	case <-done:
	case <-context.Background().Done():
		t.Fatal("Execute blocked - expected non-blocking return")
	}
}

func TestAskUser_MissingParam(t *testing.T) {
	tool := NewAskUserTool()

	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing question parameter")
	}
}

func TestAskUser_EmptyQuestion(t *testing.T) {
	tool := NewAskUserTool()

	_, err := tool.Execute(context.Background(), map[string]any{
		"question": "",
	})
	if err == nil {
		t.Error("expected error for empty question parameter")
	}
}

func TestAskUser_NoReactorDependency(t *testing.T) {
	tool := NewAskUserTool()

	if _, ok := tool.(interface{ SetEventEmitter(any) }); ok {
		t.Error("AskUser should not have SetEventEmitter method after decoupling")
	}
	if _, ok := tool.(interface{ Respond(string) error }); ok {
		t.Error("AskUser should not have Respond method after decoupling")
	}
}
