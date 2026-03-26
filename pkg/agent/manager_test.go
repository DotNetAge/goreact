package agent

import (
	"context"
	"testing"

	chatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/DotNetAge/goreact/pkg/model"
)

func TestManagerSelectionAndAssembly(t *testing.T) {
	// 1. Setup Model Manager
	mm := model.NewManager()
	err := mm.RegisterModel(&model.Model{
		Name:      "test-model",
		Provider:  "ollama",
		ModelID:   "llama3",
		Timeout:   30,
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	// 2. Setup Agent Manager
	am := NewManager(mm)

	// 3. Register Agents
	mathAgent := NewAgent("MathExpert", "Expert in mathematical calculations", "You are a math expert.", "test-model")
	codeAgent := NewAgent("CodeWizard", "Expert in writing Go code", "You are a coding wizard.", "test-model")

	am.Register(mathAgent)
	am.Register(codeAgent)

	// 4. Test Selection by keyword
	selected, err := am.SelectAgent("solve a math problem")
	if err != nil {
		t.Fatalf("Failed to select agent: %v", err)
	}
	if selected.AgentName != "MathExpert" {
		t.Errorf("Expected MathExpert, got %s", selected.AgentName)
	}

	// 5. Test Automatic Assembly
	// When selected via SelectAgent, it should be assembled
	if selected.reactor == nil {
		t.Error("Selected agent was not automatically assembled")
	}

	// 6. Test Manual Get and Assembly
	gotCode, err := am.Get("CodeWizard")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}
	if gotCode.reactor == nil {
		t.Error("Agent retrieved via Get was not automatically assembled")
	}

	// 7. Test Fallback
	// (Current implementation returns the first agent if no keywords match)
	fallback, err := am.SelectAgent("something completely unrelated")
	if err != nil {
		t.Fatalf("Failed to select fallback agent: %v", err)
	}
	if fallback == nil {
		t.Fatal("Expected a fallback agent, got nil")
	}
}

func TestManagerWithSemanticSelection(t *testing.T) {
	mm := model.NewManager()
	am := NewManager(mm)

	// Setup multiple agents
	am.Register(NewAgent("ApplesExpert", "Expert in apples", "...", "m"))
	am.Register(NewAgent("BananasExpert", "Expert in bananas", "...", "m"))

	// Mock LLM for semantic selection
	mockLLM := &semanticMockLLM{expectedResponse: "ApplesExpert"}
	am.WithLLMClient(mockLLM)

	// Trigger semantic selection (by having multiple keyword matches, but here we'll just test the logic)
	// We need to trigger multiple candidates for semantic to be called
	selected, _ := am.selectBySemantic("I want to know about red fruit", am.List())

	if selected == nil || selected.AgentName != "ApplesExpert" {
		t.Errorf("Expected ApplesExpert from semantic selection, got %v", selected)
	}
}

type semanticMockLLM struct {
	mockLLMClient
	expectedResponse string
}

func (m *semanticMockLLM) Chat(ctx context.Context, messages []chatcore.Message, opts ...chatcore.Option) (*chatcore.Response, error) {
	return &chatcore.Response{
		Content: m.expectedResponse,
		Usage:   &chatcore.Usage{TotalTokens: 10},
	}, nil
}
