package parser

import (
	"testing"
)

func TestParseLLMOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		expectError bool
	}{
		{
			name:        "empty output",
			output:      "",
			expectError: true,
		},
		{
			name:        "whitespace only",
			output:      "   \n\t  ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseLLMOutput(tt.output)
			if tt.expectError && err == nil {
				t.Error("Expected error")
			}
		})
	}
}

func TestParsePlan(t *testing.T) {
	t.Run("empty plan", func(t *testing.T) {
		_, err := ParsePlan("")
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		_, err := ParsePlan("   \n\t  ")
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("valid step plan", func(t *testing.T) {
		plan := "Step 1: Do this\nStep 2: Do that"
		tasks, err := ParsePlan(plan)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("Expected 2 tasks, got %d", len(tasks))
		}
	})

	t.Run("valid if plan", func(t *testing.T) {
		plan := "If file exists then read else create"
		tasks, err := ParsePlan(plan)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		}
	})

	t.Run("valid loop plan", func(t *testing.T) {
		plan := "Repeat check until success"
		tasks, err := ParsePlan(plan)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		}
	})
}

func TestPlannedTask_TaskTypes(t *testing.T) {
	task := PlannedTask{
		Type:     TaskSequence,
		StepName: "Step 1",
		Task:     "Do something",
	}

	if task.Type != TaskSequence {
		t.Errorf("Expected TaskSequence, got %v", task.Type)
	}

	task.Type = TaskIf
	if task.Type != TaskIf {
		t.Errorf("Expected TaskIf, got %v", task.Type)
	}

	task.Type = TaskLoop
	if task.Type != TaskLoop {
		t.Errorf("Expected TaskLoop, got %v", task.Type)
	}
}