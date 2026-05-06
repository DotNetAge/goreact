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
func (h *testHook) Execute(ctx *HookContext) HookResult {
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
		PermissionResult:    &PermissionResult{Behavior: PermissionAllow},
		UpdatedInput:        map[string]any{"key": "val"},
		PreventContinuation: true,
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

	result := hook.Execute(&HookContext{
		ToolUseContext: &ToolUseContext{
			ToolName: "Bash",
			Params:   map[string]any{"cmd": "rm -rf /"},
		},
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

	result := hook.Execute(&HookContext{
		ToolUseContext: &ToolUseContext{
			ToolName: "Write",
			Params:   map[string]any{"path": "/tmp/file"},
		},
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

	ctx := &HookContext{
		PostToolUseContext: &PostToolUseContext{
			ToolUseContext: &ToolUseContext{
				ToolName: "Read",
				Params:   map[string]any{"path": "/tmp/f"},
			},
			Result:   "file content here",
			Duration: 42,
		},
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
		ToolName: "Grep",
		Params:   map[string]any{"pattern": "TODO"},
	}
	pctx := &PostToolUseContext{
		ToolUseContext: base,
		Result:         "3 matches found",
		Err:            context.DeadlineExceeded,
		Duration:       15,
	}
	if pctx.ToolName != "Grep" {
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

type mockHook struct {
	eventType HookEventType
	executeFn func(ctx *HookContext) HookResult
	callCount int
}

func (h *mockHook) EventType() HookEventType { return h.eventType }
func (h *mockHook) Execute(ctx *HookContext) HookResult {
	h.callCount++
	return h.executeFn(ctx)
}

func TestHook_PreToolUse_PreventContinuation(t *testing.T) {
	hook := &mockHook{
		eventType: HookPreToolUse,
		executeFn: func(ctx *HookContext) HookResult {
			return HookResult{
				PreventContinuation: true,
				Message:             "Blocked by policy",
			}
		},
	}

	tuc := &ToolUseContext{ToolName: "Bash", Params: map[string]any{"command": "rm -rf /"}}
	ctx := &HookContext{ToolUseContext: tuc}
	result := hook.Execute(ctx)

	if !result.PreventContinuation {
		t.Error("expected hook to block execution")
	}
	if hook.callCount != 1 {
		t.Errorf("expected 1 call, got %d", hook.callCount)
	}
}

func TestHook_PreToolUse_ModifyInput(t *testing.T) {
	hook := &mockHook{
		eventType: HookPreToolUse,
		executeFn: func(ctx *HookContext) HookResult {
			newInput := map[string]any{"command": "echo safe"}
			return HookResult{UpdatedInput: newInput}
		},
	}

	tuc := &ToolUseContext{ToolName: "Bash", Params: map[string]any{"command": "rm -rf /"}}
	ctx := &HookContext{ToolUseContext: tuc}
	result := hook.Execute(ctx)

	if result.UpdatedInput == nil {
		t.Fatal("expected UpdatedInput to be set")
	}
	if result.UpdatedInput["command"] != "echo safe" {
		t.Errorf("expected modified command, got %v", result.UpdatedInput["command"])
	}
}

func TestHook_PreToolUse_PermissionOverride(t *testing.T) {
	hook := &mockHook{
		eventType: HookPreToolUse,
		executeFn: func(ctx *HookContext) HookResult {
			return HookResult{
				PermissionResult: &PermissionResult{
					Behavior: PermissionAllow,
					Message:  "Auto-approved by hook",
				},
			}
		},
	}

	tuc := &ToolUseContext{ToolName: "Read"}
	ctx := &HookContext{ToolUseContext: tuc}
	result := hook.Execute(ctx)

	if result.PermissionResult == nil {
		t.Fatal("expected PermissionResult")
	}
	if result.PermissionResult.Behavior != PermissionAllow {
		t.Errorf("expected Allow, got %v", result.PermissionResult.Behavior)
	}
}

func TestHook_PostToolUse(t *testing.T) {
	hook := &mockHook{
		eventType: HookPostToolUse,
		executeFn: func(ctx *HookContext) HookResult {
			if ctx.PostToolUseContext == nil {
				t.Fatal("expected PostToolUseContext to be populated")
			}
			return HookResult{Message: "Post-hook executed"}
		},
	}

	tuc := &ToolUseContext{ToolName: "Write"}
	postCtx := &PostToolUseContext{
		ToolUseContext: tuc,
		Result:         "file written",
		Duration:       42,
	}
	ctx := &HookContext{
		ToolUseContext:     tuc,
		PostToolUseContext: postCtx,
	}
	result := hook.Execute(ctx)

	if hook.EventType() != HookPostToolUse {
		t.Errorf("expected PostToolUse, got %v", hook.EventType())
	}
	if result.Message != "Post-hook executed" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestHook_SessionStart(t *testing.T) {
	callCount := 0
	hook := &mockHook{
		eventType: HookSessionStart,
		executeFn: func(ctx *HookContext) HookResult {
			callCount++
			return HookResult{Message: "Session initialized"}
		},
	}

	ctx := &HookContext{}
	result := hook.Execute(ctx)

	if callCount != 1 {
		t.Error("hook should have been called once")
	}
	if result.Message != "Session initialized" {
		t.Errorf("unexpected message: %s", result.Message)
	}
	if hook.EventType() != HookSessionStart {
		t.Errorf("expected SessionStart, got %v", hook.EventType())
	}
}

func TestHook_Stop(t *testing.T) {
	hook := &mockHook{
		eventType: HookStop,
		executeFn: func(ctx *HookContext) HookResult {
			return HookResult{Message: "Cleanup completed"}
		},
	}

	ctx := &HookContext{}
	result := hook.Execute(ctx)

	if hook.EventType() != HookStop {
		t.Errorf("expected Stop, got %v", hook.EventType())
	}
	if result.Message != "Cleanup completed" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}
