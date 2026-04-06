package core

import (
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestNewState(t *testing.T) {
	state := NewState("test-session", "Hello world", 10, 3)

	if state.SessionName != "test-session" {
		t.Errorf("SessionName = %q, want 'test-session'", state.SessionName)
	}
	if state.Input != "Hello world" {
		t.Errorf("Input = %q, want 'Hello world'", state.Input)
	}
	if state.MaxSteps != 10 {
		t.Errorf("MaxSteps = %d, want 10", state.MaxSteps)
	}
	if state.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", state.MaxRetries)
	}
	if state.CurrentStep != 0 {
		t.Errorf("CurrentStep = %d, want 0", state.CurrentStep)
	}
	if state.Status != common.StatusPending {
		t.Errorf("Status = %q, want 'pending'", state.Status)
	}
	if state.Thoughts == nil {
		t.Error("Thoughts should not be nil")
	}
	if state.Actions == nil {
		t.Error("Actions should not be nil")
	}
	if state.Observations == nil {
		t.Error("Observations should not be nil")
	}
	if state.Context == nil {
		t.Error("Context should not be nil")
	}
}

func TestState_AddThought(t *testing.T) {
	state := NewState("test", "input", 10, 3)
	thought := &Thought{Content: "test thought"}

	state.AddThought(thought)

	if len(state.Thoughts) != 1 {
		t.Errorf("len(Thoughts) = %d, want 1", len(state.Thoughts))
	}
	if state.Thoughts[0] != thought {
		t.Error("Thought not added correctly")
	}
}

func TestState_AddAction(t *testing.T) {
	state := NewState("test", "input", 10, 3)
	action := &Action{Target: "test_tool"}

	state.AddAction(action)

	if len(state.Actions) != 1 {
		t.Errorf("len(Actions) = %d, want 1", len(state.Actions))
	}
}

func TestState_AddObservation(t *testing.T) {
	state := NewState("test", "input", 10, 3)
	obs := &Observation{Content: "test observation"}

	state.AddObservation(obs)

	if len(state.Observations) != 1 {
		t.Errorf("len(Observations) = %d, want 1", len(state.Observations))
	}
}

func TestState_AddReflection(t *testing.T) {
	state := NewState("test", "input", 10, 3)
	reflection := &Reflection{FailureReason: "test failure"}

	state.AddReflection(reflection)

	if len(state.Reflections) != 1 {
		t.Errorf("len(Reflections) = %d, want 1", len(state.Reflections))
	}
}

func TestState_IncrementStep(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	for i := 0; i < 5; i++ {
		state.IncrementStep()
	}

	if state.CurrentStep != 5 {
		t.Errorf("CurrentStep = %d, want 5", state.CurrentStep)
	}
}

func TestState_IncrementRetry(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	state.IncrementRetry()
	state.IncrementRetry()

	if state.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", state.RetryCount)
	}
}

func TestState_CanRetry(t *testing.T) {
	state := NewState("test", "input", 10, 2)

	if !state.CanRetry() {
		t.Error("CanRetry() = false, want true")
	}

	state.IncrementRetry()
	state.IncrementRetry()

	if state.CanRetry() {
		t.Error("CanRetry() = true after max retries, want false")
	}
}

func TestState_IsComplete(t *testing.T) {
	state := NewState("test", "input", 3, 3)

	if state.IsComplete() {
		t.Error("IsComplete() = true at step 0, want false")
	}

	state.IncrementStep()
	state.IncrementStep()
	state.IncrementStep()

	if !state.IsComplete() {
		t.Error("IsComplete() = false at max steps, want true")
	}
}

func TestState_SetStatus(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	state.SetStatus(common.StatusRunning)
	if state.Status != common.StatusRunning {
		t.Errorf("Status = %q, want 'running'", state.Status)
	}
	if !state.EndTime.IsZero() {
		t.Error("EndTime should be zero for running status")
	}

	state.SetStatus(common.StatusCompleted)
	if state.Status != common.StatusCompleted {
		t.Errorf("Status = %q, want 'completed'", state.Status)
	}
	if state.EndTime.IsZero() {
		t.Error("EndTime should not be zero for completed status")
	}
}

func TestState_PendingQuestion(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	question := NewPendingQuestionNode("test", common.QuestionTypeConfirmation, "Continue?")
	state.SetPendingQuestion(question)

	if state.PendingQuestion != question {
		t.Error("PendingQuestion not set correctly")
	}
	if state.Status != common.StatusPaused {
		t.Errorf("Status = %q, want 'paused'", state.Status)
	}

	state.ClearPendingQuestion()

	if state.PendingQuestion != nil {
		t.Error("PendingQuestion should be nil after clear")
	}
	if state.Status != common.StatusRunning {
		t.Errorf("Status = %q, want 'running'", state.Status)
	}
}

func TestState_TokenUsage(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	state.AddTokenUsage(100, 50)

	if state.TokenUsage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", state.TokenUsage.PromptTokens)
	}
	if state.TokenUsage.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", state.TokenUsage.CompletionTokens)
	}
	if state.TokenUsage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", state.TokenUsage.TotalTokens)
	}

	state.AddTokenUsage(50, 25)

	if state.TokenUsage.PromptTokens != 150 {
		t.Errorf("PromptTokens = %d, want 150", state.TokenUsage.PromptTokens)
	}
	if state.TokenUsage.TotalTokens != 225 {
		t.Errorf("TotalTokens = %d, want 225", state.TokenUsage.TotalTokens)
	}
}

func TestState_GetLastItems(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	if state.GetLastThought() != nil {
		t.Error("GetLastThought() should return nil for empty thoughts")
	}
	if state.GetLastAction() != nil {
		t.Error("GetLastAction() should return nil for empty actions")
	}
	if state.GetLastObservation() != nil {
		t.Error("GetLastObservation() should return nil for empty observations")
	}

	thought1 := &Thought{Content: "thought 1"}
	thought2 := &Thought{Content: "thought 2"}
	state.AddThought(thought1)
	state.AddThought(thought2)

	if state.GetLastThought() != thought2 {
		t.Error("GetLastThought() should return last thought")
	}
}

func TestState_GetDuration(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	time.Sleep(10 * time.Millisecond)
	duration := state.GetDuration()

	if duration <= 0 {
		t.Errorf("GetDuration() = %v, want positive duration", duration)
	}

	state.EndTime = time.Now()
	duration = state.GetDuration()

	if duration <= 0 {
		t.Errorf("GetDuration() with EndTime = %v, want positive duration", duration)
	}
}

func TestState_FreezeThaw(t *testing.T) {
	state := NewState("test", "input", 10, 3)
	state.AddTokenUsage(100, 50)
	state.Context["key"] = "value"

	data, err := state.Freeze()
	if err != nil {
		t.Fatalf("Freeze() error = %v", err)
	}

	newState := &State{}
	err = newState.Thaw(data)
	if err != nil {
		t.Fatalf("Thaw() error = %v", err)
	}

	if newState.SessionName != state.SessionName {
		t.Errorf("SessionName = %q, want %q", newState.SessionName, state.SessionName)
	}
	if newState.Input != state.Input {
		t.Errorf("Input = %q, want %q", newState.Input, state.Input)
	}
	if newState.TokenUsage.TotalTokens != state.TokenUsage.TotalTokens {
		t.Errorf("TotalTokens = %d, want %d", newState.TokenUsage.TotalTokens, state.TokenUsage.TotalTokens)
	}
}

func TestState_Clone(t *testing.T) {
	state := NewState("test", "input", 10, 3)
	state.Context["key"] = "value"
	state.AddThought(&Thought{Content: "test"})

	clone := state.Clone()

	if clone.SessionName != state.SessionName {
		t.Errorf("Clone SessionName = %q, want %q", clone.SessionName, state.SessionName)
	}
	if clone.Context["key"] != "value" {
		t.Errorf("Clone Context[key] = %v, want 'value'", clone.Context["key"])
	}

	// Modify original should not affect clone
	state.Context["key"] = "modified"
	if clone.Context["key"] == "modified" {
		t.Error("Clone Context should not be affected by original modifications")
	}
}

func TestState_InjectReflection(t *testing.T) {
	state := NewState("test", "input", 10, 3)

	reflection := &Reflection{FailureReason: "test"}
	state.InjectReflection(reflection)

	if len(state.ActiveReflections) != 1 {
		t.Errorf("len(ActiveReflections) = %d, want 1", len(state.ActiveReflections))
	}

	state.ClearActiveReflections()

	if len(state.ActiveReflections) != 0 {
		t.Errorf("len(ActiveReflections) = %d, want 0", len(state.ActiveReflections))
	}
}
