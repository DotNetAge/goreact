package core

import (
	"context"
	"testing"
)

type testHook struct {
	eventType HookEventType
	block     bool
	modify    bool
	message   string
}

func (h *testHook) EventType() HookEventType { return h.eventType }
func (h *testHook) Execute(ctx any) HookResult {
	result := HookResult{Message: h.message}
	if h.block {
		result.PermissionResult = &PermissionResult{Behavior: PermissionDeny, Message: "blocked by test hook"}
	}
	if h.modify {
		result.UpdatedInput = map[string]any{"injected": "value"}
	}
	return result
}

func TestHookResult_Fields(t *testing.T) {
	r := HookResult{
		PermissionResult:     &PermissionResult{Behavior: PermissionAllow},
		UpdatedInput:         map[string]any{"key": "val"},
		PreventContinuation:  true,
		Message:             "test message",
	}
	if !r.PreventContinuation {
		t.Error("PreventContinuation should be true")
	}
	if r.Message != "test message" {
		t.Errorf("Message mismatch: %s", r.Message)
	}
}

func TestHook_PreToolUse_Block(t *testing.T) {
	hook := &testHook{
		eventType: HookPreToolUse,
		block:     true,
		message:   "denied",
	}
	if hook.EventType() != HookPreToolUse {
		t.Errorf("expected %s, got %s", HookPreToolUse, hook.EventType())
	}

	result := hook.Execute(&ToolUseContext{
		ToolName: "bash",
		Params:   map[string]any{"cmd": "rm -rf /"},
	})
	if result.PermissionResult == nil || result.PermissionResult.Behavior != PermissionDeny {
		t.Error("pre-tool-use hook should block execution")
	}
	if result.Message != "denied" {
		t.Errorf("message mismatch: %s", result.Message)
	}
}

func TestHook_PreToolUse_Modify(t *testing.T) {
	hook := &testHook{
		eventType: HookPreToolUse,
		modify:    true,
	}

	result := hook.Execute(&ToolUseContext{
		ToolName: "write",
		Params:   map[string]any{"path": "/tmp/file"},
	})
	if result.UpdatedInput == nil {
		t.Error("hook should modify input")
	}
	if result.UpdatedInput["injected"] != "value" {
		t.Errorf("updated input mismatch: %+v", result.UpdatedInput)
	}
}

func TestHook_PostToolUse_Context(t *testing.T) {
	hook := &testHook{
		eventType: HookPostToolUse,
		message:   "logged",
	}

	ctx := &PostToolUseContext{
		ToolUseContext: &ToolUseContext{
			ToolName: "read",
			Params:   map[string]any{"path": "/tmp/f"},
		},
		Result:   "file content here",
		Duration: 42,
	}

	result := hook.Execute(ctx)
	if result.Message != "logged" {
		t.Errorf("message mismatch: %s", result.Message)
	}
}

func TestHookEventTypes_Constants(t *testing.T) {
	constants := []struct {
		val  HookEventType
		name string
	}{
		{HookPreToolUse, "pre_tool_use"},
		{HookPostToolUse, "post_tool_use"},
		{HookSessionStart, "session_start"},
		{HookStop, "stop"},
	}
	for _, c := range constants {
		if string(c.val) != c.name {
			t.Errorf("constant value mismatch: got %q, want %q", c.val, c.name)
		}
	}
}

func TestPostToolUseContext_Fields(t *testing.T) {
	base := &ToolUseContext{
		ToolName: "grep",
		Params:   map[string]any{"pattern": "TODO"},
	}
	pctx := &PostToolUseContext{
		ToolUseContext: base,
		Result:        "3 matches found",
		Err:           context.DeadlineExceeded,
		Duration:      15,
	}
	if pctx.ToolName != "grep" {
		t.Errorf("embedded ToolName should be accessible: %s", pctx.ToolName)
	}
	if pctx.Result != "3 matches found" {
		t.Errorf("Result mismatch: %s", pctx.Result)
	}
	if pctx.Err == nil {
		t.Error("Err should be non-nil")
	}
	if pctx.Duration != 15 {
		t.Errorf("Duration mismatch: %d", pctx.Duration)
	}
}

func TestHookInterface_Satisfaction(t *testing.T) {
	var _ Hook = (*testHook)(nil)
}

func TestPermissionResult_InHookResult(t *testing.T) {
	pr := &PermissionResult{Behavior: PermissionDeny, Message: "security policy"}
	hr := HookResult{PermissionResult: pr}
	if hr.PermissionResult.Behavior == PermissionAllow {
		t.Error("should be denied")
	}
}
