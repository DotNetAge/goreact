package core

import (
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestNewAction(t *testing.T) {
	params := map[string]any{"arg1": "value1"}
	action := NewAction(common.ActionTypeToolCall, "test_tool", params)

	if action.Type != common.ActionTypeToolCall {
		t.Errorf("Type = %q, want 'tool_call'", action.Type)
	}
	if action.Target != "test_tool" {
		t.Errorf("Target = %q, want 'test_tool'", action.Target)
	}
	if action.Params["arg1"] != "value1" {
		t.Errorf("Params[arg1] = %v, want 'value1'", action.Params["arg1"])
	}
	if action.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestAction_WithReasoning(t *testing.T) {
	action := NewAction(common.ActionTypeToolCall, "test_tool", nil)
	action.WithReasoning("This tool is needed because...")

	if action.Reasoning != "This tool is needed because..." {
		t.Errorf("Reasoning = %q, want 'This tool is needed because...'", action.Reasoning)
	}
}

func TestAction_TypeChecks(t *testing.T) {
	tests := []struct {
		name     string
		typ      common.ActionType
		isTool   bool
		isSkill  bool
		isDeleg  bool
		isNoAct  bool
	}{
		{"tool_call", common.ActionTypeToolCall, true, false, false, false},
		{"skill_invoke", common.ActionTypeSkillInvoke, false, true, false, false},
		{"sub_agent_delegate", common.ActionTypeSubAgentDelegate, false, false, true, false},
		{"no_action", common.ActionTypeNoAction, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &Action{Type: tt.typ}

			if action.IsToolCall() != tt.isTool {
				t.Errorf("IsToolCall() = %v, want %v", action.IsToolCall(), tt.isTool)
			}
			if action.IsSkillInvoke() != tt.isSkill {
				t.Errorf("IsSkillInvoke() = %v, want %v", action.IsSkillInvoke(), tt.isSkill)
			}
			if action.IsDelegation() != tt.isDeleg {
				t.Errorf("IsDelegation() = %v, want %v", action.IsDelegation(), tt.isDeleg)
			}
			if action.IsNoAction() != tt.isNoAct {
				t.Errorf("IsNoAction() = %v, want %v", action.IsNoAction(), tt.isNoAct)
			}
		})
	}
}

func TestNewActionResult(t *testing.T) {
	result := NewActionResult(true, "success output")

	if result.Success != true {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.Result != "success output" {
		t.Errorf("Result = %q, want 'success output'", result.Result)
	}
	if result.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

func TestActionResult_WithError(t *testing.T) {
	result := NewActionResult(true, nil)
	result.WithError("something went wrong")

	if result.Error != "something went wrong" {
		t.Errorf("Error = %q, want 'something went wrong'", result.Error)
	}
	if result.Success != false {
		t.Error("Success should be false after error")
	}
}

func TestActionResult_WithDuration(t *testing.T) {
	result := NewActionResult(true, nil)
	result.WithDuration(100 * time.Millisecond)

	if result.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", result.Duration)
	}
}

func TestActionResult_WithMetadata(t *testing.T) {
	result := NewActionResult(true, nil)
	result.WithMetadata("key1", "value1")
	result.WithMetadata("key2", 42)

	if result.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %v, want 'value1'", result.Metadata["key1"])
	}
	if result.Metadata["key2"] != 42 {
		t.Errorf("Metadata[key2] = %v, want 42", result.Metadata["key2"])
	}
}

func TestActionResult_WithNames(t *testing.T) {
	result := NewActionResult(true, nil)

	result.WithTool("my_tool")
	if result.ToolName != "my_tool" {
		t.Errorf("ToolName = %q, want 'my_tool'", result.ToolName)
	}

	result.WithSkill("my_skill")
	if result.SkillName != "my_skill" {
		t.Errorf("SkillName = %q, want 'my_skill'", result.SkillName)
	}

	result.WithSubAgent("my_agent")
	if result.SubAgentName != "my_agent" {
		t.Errorf("SubAgentName = %q, want 'my_agent'", result.SubAgentName)
	}
}
