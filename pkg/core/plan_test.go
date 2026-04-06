package core

import (
	"testing"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestNewPlan(t *testing.T) {
	plan := NewPlan("session-123", "Complete the task")

	if plan.SessionName != "session-123" {
		t.Errorf("SessionName = %q, want 'session-123'", plan.SessionName)
	}
	if plan.Goal != "Complete the task" {
		t.Errorf("Goal = %q, want 'Complete the task'", plan.Goal)
	}
	if plan.Status != common.PlanStatusPending {
		t.Errorf("Status = %q, want 'pending'", plan.Status)
	}
	if plan.CurrentStepIndex != 0 {
		t.Errorf("CurrentStepIndex = %d, want 0", plan.CurrentStepIndex)
	}
	if plan.Steps == nil {
		t.Error("Steps should not be nil")
	}
}

func TestPlan_AddStep(t *testing.T) {
	plan := NewPlan("session", "goal")

	step1 := plan.AddStep("action1", "First step")
	step2 := plan.AddStep("action2", "Second step")

	if len(plan.Steps) != 2 {
		t.Errorf("len(Steps) = %d, want 2", len(plan.Steps))
	}
	if step1.Index != 0 {
		t.Errorf("step1.Index = %d, want 0", step1.Index)
	}
	if step2.Index != 1 {
		t.Errorf("step2.Index = %d, want 1", step2.Index)
	}
	if step1.Status != common.StepStatusPending {
		t.Errorf("step1.Status = %q, want 'pending'", step1.Status)
	}
}

func TestPlan_GetCurrentStep(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.AddStep("action1", "First step")
	plan.AddStep("action2", "Second step")

	step := plan.GetCurrentStep()
	if step == nil {
		t.Fatal("GetCurrentStep() returned nil")
	}
	if step.Index != 0 {
		t.Errorf("Current step Index = %d, want 0", step.Index)
	}
}

func TestPlan_GetCurrentStep_Empty(t *testing.T) {
	plan := NewPlan("session", "goal")

	step := plan.GetCurrentStep()
	if step != nil {
		t.Errorf("GetCurrentStep() on empty plan should return nil, got %v", step)
	}
}

func TestPlan_AdvanceStep(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.AddStep("action1", "First step")
	plan.AddStep("action2", "Second step")
	plan.AddStep("action3", "Third step")

	// Advance from step 0 to 1
	if !plan.AdvanceStep() {
		t.Error("AdvanceStep() should return true")
	}
	if plan.CurrentStepIndex != 1 {
		t.Errorf("CurrentStepIndex = %d, want 1", plan.CurrentStepIndex)
	}

	// Advance from step 1 to 2
	if !plan.AdvanceStep() {
		t.Error("AdvanceStep() should return true")
	}
	if plan.CurrentStepIndex != 2 {
		t.Errorf("CurrentStepIndex = %d, want 2", plan.CurrentStepIndex)
	}

	// Cannot advance past last step
	if plan.AdvanceStep() {
		t.Error("AdvanceStep() should return false at last step")
	}
}

func TestPlan_CompleteStep(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.AddStep("action1", "First step")

	plan.CompleteStep("Success!")

	step := plan.GetCurrentStep()
	if step.Status != common.StepStatusCompleted {
		t.Errorf("Status = %q, want 'completed'", step.Status)
	}
	if step.Outcome != "Success!" {
		t.Errorf("Outcome = %q, want 'Success!'", step.Outcome)
	}
}

func TestPlan_FailStep(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.AddStep("action1", "First step")

	plan.FailStep("Something went wrong")

	step := plan.GetCurrentStep()
	if step.Status != common.StepStatusFailed {
		t.Errorf("Status = %q, want 'failed'", step.Status)
	}
	if step.Outcome != "Something went wrong" {
		t.Errorf("Outcome = %q, want 'Something went wrong'", step.Outcome)
	}
}

func TestPlan_IsComplete(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.AddStep("action1", "First step")
	plan.AddStep("action2", "Second step")

	if plan.IsComplete() {
		t.Error("IsComplete() should be false with pending steps")
	}

	plan.Steps[0].Status = common.StepStatusCompleted
	if plan.IsComplete() {
		t.Error("IsComplete() should be false with one incomplete step")
	}

	plan.Steps[1].Status = common.StepStatusCompleted
	if !plan.IsComplete() {
		t.Error("IsComplete() should be true with all steps completed")
	}
}

func TestPlan_IsComplete_WithSkipped(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.AddStep("action1", "First step")
	plan.AddStep("action2", "Second step")

	plan.Steps[0].Status = common.StepStatusCompleted
	plan.Steps[1].Status = common.StepStatusSkipped

	if !plan.IsComplete() {
		t.Error("IsComplete() should be true with completed and skipped steps")
	}
}

func TestPlan_MarkSuccess(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.MarkSuccess()

	if plan.Status != common.PlanStatusCompleted {
		t.Errorf("Status = %q, want 'completed'", plan.Status)
	}
	if !plan.Success {
		t.Error("Success should be true")
	}
}

func TestPlan_MarkFailed(t *testing.T) {
	plan := NewPlan("session", "goal")
	plan.MarkFailed()

	if plan.Status != common.PlanStatusFailed {
		t.Errorf("Status = %q, want 'failed'", plan.Status)
	}
	if plan.Success {
		t.Error("Success should be false")
	}
}

func TestNewPlanNode(t *testing.T) {
	node := NewPlanNode("session-123", "Complete the task")

	if node.SessionName != "session-123" {
		t.Errorf("SessionName = %q, want 'session-123'", node.SessionName)
	}
	if node.Goal != "Complete the task" {
		t.Errorf("Goal = %q, want 'Complete the task'", node.Goal)
	}
	if node.NodeType != common.NodeTypePlan {
		t.Errorf("NodeType = %q, want 'Plan'", node.NodeType)
	}
	if node.Status != common.PlanStatusPending {
		t.Errorf("Status = %q, want 'pending'", node.Status)
	}
}

func TestPlanStep(t *testing.T) {
	step := &PlanStep{
		Index:            0,
		Action:           "read_file",
		Description:      "Read the configuration file",
		Status:           common.StepStatusPending,
		ExpectedOutcome:  "File content loaded",
		Tools:            []string{"read_file"},
		Dependencies:     []int{},
	}

	if step.Index != 0 {
		t.Errorf("Index = %d, want 0", step.Index)
	}
	if step.Action != "read_file" {
		t.Errorf("Action = %q, want 'read_file'", step.Action)
	}
	if len(step.Tools) != 1 {
		t.Errorf("len(Tools) = %d, want 1", len(step.Tools))
	}
}
