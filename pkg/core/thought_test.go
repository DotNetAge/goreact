package core

import (
	"testing"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestNewThought(t *testing.T) {
	thought := NewThought("I need to analyze this", "Step by step reasoning", "act", 0.85)

	if thought.Content != "I need to analyze this" {
		t.Errorf("Content = %q, want 'I need to analyze this'", thought.Content)
	}
	if thought.Reasoning != "Step by step reasoning" {
		t.Errorf("Reasoning = %q, want 'Step by step reasoning'", thought.Reasoning)
	}
	if thought.Decision != "act" {
		t.Errorf("Decision = %q, want 'act'", thought.Decision)
	}
	if thought.Confidence != 0.85 {
		t.Errorf("Confidence = %f, want 0.85", thought.Confidence)
	}
	if thought.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestThought_WithAction(t *testing.T) {
	thought := NewThought("content", "reasoning", "act", 0.9)
	action := &ActionIntent{
		Type:   "tool_call",
		Target: "read_file",
		Params: map[string]any{"path": "/tmp/test.txt"},
	}

	thought.WithAction(action)

	if thought.Action != action {
		t.Error("Action not set correctly")
	}
}

func TestThought_WithFinalAnswer(t *testing.T) {
	thought := NewThought("content", "reasoning", "answer", 0.95)
	thought.WithFinalAnswer("The answer is 42")

	if thought.FinalAnswer != "The answer is 42" {
		t.Errorf("FinalAnswer = %q, want 'The answer is 42'", thought.FinalAnswer)
	}
}

func TestThought_IsAct(t *testing.T) {
	thought := NewThought("content", "reasoning", "act", 0.9)

	if !thought.IsAct() {
		t.Error("IsAct() should return true for 'act' decision")
	}

	thought.Decision = "answer"
	if thought.IsAct() {
		t.Error("IsAct() should return false for 'answer' decision")
	}
}

func TestThought_IsAnswer(t *testing.T) {
	thought := NewThought("content", "reasoning", "answer", 0.9)

	if !thought.IsAnswer() {
		t.Error("IsAnswer() should return true for 'answer' decision")
	}

	thought.Decision = "act"
	if thought.IsAnswer() {
		t.Error("IsAnswer() should return false for 'act' decision")
	}
}

func TestThought_ToAction(t *testing.T) {
	thought := NewThought("content", "reasoning", "act", 0.9)
	thought.WithAction(&ActionIntent{
		Type:      "tool_call",
		Target:    "read_file",
		Params:    map[string]any{"path": "/tmp/test.txt"},
		Reasoning: "Need to read the file",
	})

	action := thought.ToAction()

	if action == nil {
		t.Fatal("ToAction() returned nil")
	}
	if action.Type != common.ActionTypeToolCall {
		t.Errorf("Type = %q, want 'tool_call'", action.Type)
	}
	if action.Target != "read_file" {
		t.Errorf("Target = %q, want 'read_file'", action.Target)
	}
	if action.Params["path"] != "/tmp/test.txt" {
		t.Errorf("Params[path] = %v, want '/tmp/test.txt'", action.Params["path"])
	}
}

func TestThought_ToAction_NilAction(t *testing.T) {
	thought := NewThought("content", "reasoning", "answer", 0.9)

	action := thought.ToAction()

	if action != nil {
		t.Errorf("ToAction() should return nil for thought without action, got %v", action)
	}
}

func TestActionIntent(t *testing.T) {
	intent := &ActionIntent{
		Type:      "skill_invoke",
		Target:    "code_review",
		Params:    map[string]any{"language": "go"},
		Reasoning: "Need to review the code",
	}

	if intent.Type != "skill_invoke" {
		t.Errorf("Type = %q, want 'skill_invoke'", intent.Type)
	}
	if intent.Target != "code_review" {
		t.Errorf("Target = %q, want 'code_review'", intent.Target)
	}
	if intent.Params["language"] != "go" {
		t.Errorf("Params[language] = %v, want 'go'", intent.Params["language"])
	}
}

func TestIntentResult(t *testing.T) {
	result := &IntentResult{
		Type:            "task",
		Confidence:      0.92,
		Reasoning:       "User wants to perform a task",
		Context:         map[string]any{"urgency": "high"},
		RelatedSession:  "session-123",
		PendingQuestion: "What file format?",
		ExtractedAnswer: "JSON",
	}

	if result.Type != "task" {
		t.Errorf("Type = %q, want 'task'", result.Type)
	}
	if result.Confidence != 0.92 {
		t.Errorf("Confidence = %f, want 0.92", result.Confidence)
	}
	if result.Context["urgency"] != "high" {
		t.Errorf("Context[urgency] = %v, want 'high'", result.Context["urgency"])
	}
}

func TestDefaultIntentFallbackStrategy(t *testing.T) {
	strategy := DefaultIntentFallbackStrategy()

	if strategy.MinConfidence != 0.5 {
		t.Errorf("MinConfidence = %f, want 0.5", strategy.MinConfidence)
	}
	if strategy.ClarifyThreshold != 0.7 {
		t.Errorf("ClarifyThreshold = %f, want 0.7", strategy.ClarifyThreshold)
	}
	if strategy.DefaultIntent != "task" {
		t.Errorf("DefaultIntent = %q, want 'task'", strategy.DefaultIntent)
	}
	if strategy.MaxRetries != 2 {
		t.Errorf("MaxRetries = %d, want 2", strategy.MaxRetries)
	}
	if !strategy.EnableClarification {
		t.Error("EnableClarification should be true")
	}
}
