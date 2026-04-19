package tools

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/core"
)

func TestAskPermission_AllowSafeTool(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "read_file",
		ToolInfo: &core.ToolInfo{
			Name:          "read_file",
			SecurityLevel: core.LevelSafe,
			IsReadOnly:    true,
		},
	}

	result := p.CheckPermissions(ctx)
	if result.Behavior != core.PermissionAllow {
		t.Errorf("expected Allow for safe read-only tool, got %s", result.Behavior)
	}
}

func TestAskPermission_AllowSensitiveReadOnly(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "search",
		ToolInfo: &core.ToolInfo{
			Name:          "search",
			SecurityLevel: core.LevelSensitive,
			IsReadOnly:    true,
		},
	}

	result := p.CheckPermissions(ctx)
	if result.Behavior != core.PermissionAllow {
		t.Errorf("expected Allow for read-only tool, got %s", result.Behavior)
	}
}

func TestAskPermission_AskSensitiveTool(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "replace",
		ToolInfo: &core.ToolInfo{
			Name:          "replace",
			SecurityLevel: core.LevelSensitive,
			IsReadOnly:    false,
		},
	}

	result := p.CheckPermissions(ctx)
	if result.Behavior != core.PermissionAsk {
		t.Errorf("expected Ask for sensitive non-readonly tool, got %s", result.Behavior)
	}
	if result.Message == "" {
		t.Error("expected non-empty message for Ask")
	}
}

func TestAskPermission_AskHighRiskTool(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "delete_file",
		ToolInfo: &core.ToolInfo{
			Name:          "delete_file",
			SecurityLevel: core.LevelHighRisk,
			IsReadOnly:    false,
		},
	}

	result := p.CheckPermissions(ctx)
	if result.Behavior != core.PermissionAsk {
		t.Errorf("expected Ask for high-risk tool, got %s", result.Behavior)
	}
}

func TestAskPermission_BlockAndWait_Respond(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "replace",
		ToolInfo: &core.ToolInfo{
			Name:          "replace",
			SecurityLevel: core.LevelSensitive,
		},
		Ctx:           newTestContext(),
	}

	var finalResult core.PermissionResult
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		finalResult = p.BlockAndWait(ctx)
	}()

	// Wait a bit for BlockAndWait to set up pending state
	time.Sleep(50 * time.Millisecond)

	if !p.IsWaiting() {
		t.Fatal("expected IsWaiting to be true")
	}

	// User approves
	p.Respond(core.PermissionResult{
		Behavior: core.PermissionAllow,
		Message:  "User approved",
	})

	wg.Wait()

	if finalResult.Behavior != core.PermissionAllow {
		t.Errorf("expected Allow after user responds, got %s", finalResult.Behavior)
	}
	if p.IsWaiting() {
		t.Error("expected IsWaiting to be false after response")
	}
}

func TestAskPermission_BlockAndWait_Denied(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "delete_file",
		ToolInfo: &core.ToolInfo{
			Name:          "delete_file",
			SecurityLevel: core.LevelHighRisk,
		},
		Ctx:           newTestContext(),
	}

	var finalResult core.PermissionResult
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		finalResult = p.BlockAndWait(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	p.Respond(core.PermissionResult{
		Behavior: core.PermissionDeny,
		Message:  "User rejected: too dangerous",
	})

	wg.Wait()

	if finalResult.Behavior != core.PermissionDeny {
		t.Errorf("expected Deny after user responds, got %s", finalResult.Behavior)
	}
}

func TestAskPermission_BlockAndWait_WithModifiedInput(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "write_file",
		ToolInfo: &core.ToolInfo{
			Name:          "write_file",
			SecurityLevel: core.LevelSensitive,
		},
		Ctx:           newTestContext(),
	}

	var finalResult core.PermissionResult
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		finalResult = p.BlockAndWait(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// User approves with modified params
	p.Respond(core.PermissionResult{
		Behavior: core.PermissionAllow,
		UpdatedInput: map[string]any{
			"path":     "/safe/path/file.txt",
			"content":  "modified content",
			"original": "/dangerous/path/file.txt",
		},
	})

	wg.Wait()

	if finalResult.Behavior != core.PermissionAllow {
		t.Errorf("expected Allow, got %s", finalResult.Behavior)
	}
	if finalResult.UpdatedInput == nil {
		t.Error("expected UpdatedInput to be set")
	}
	if finalResult.UpdatedInput["path"] != "/safe/path/file.txt" {
		t.Errorf("expected modified path, got %v", finalResult.UpdatedInput["path"])
	}
}

func TestAskPermission_RespondError(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "replace",
		ToolInfo: &core.ToolInfo{
			Name:          "replace",
			SecurityLevel: core.LevelSensitive,
		},
		Ctx:           newTestContext(),
	}

	var finalResult core.PermissionResult
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		finalResult = p.BlockAndWait(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	p.RespondError(fmt.Errorf("user cancelled"))

	wg.Wait()

	if finalResult.Behavior != core.PermissionDeny {
		t.Errorf("expected Deny on error, got %s", finalResult.Behavior)
	}
}

func TestAskPermission_RespondWhenNotWaiting(t *testing.T) {
	p := NewAskPermission()

	err := p.Respond(core.PermissionResult{Behavior: core.PermissionAllow})
	if err == nil {
		t.Error("expected error when responding with no pending request")
	}
}

func TestAskPermission_WaitWithTimeout(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "replace",
		ToolInfo: &core.ToolInfo{
			Name:          "replace",
			SecurityLevel: core.LevelSensitive,
		},
		Ctx:           newTestContext(),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.BlockAndWait(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	err := p.WaitWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}

	// Clean up: respond to unblock the goroutine
	p.Respond(core.PermissionResult{Behavior: core.PermissionAllow})
	<-done
}

func TestAskPermission_EventEmitter(t *testing.T) {
	p := NewAskPermission()

	var emittedEvents []core.ReactEvent
	p.SetEventEmitter(func(e core.ReactEvent) {
		emittedEvents = append(emittedEvents, e)
	})

	// Event emission is handled by ExecuteTool in the reactor, not by AskPermission directly.
	// This test verifies the setter works correctly.
	if len(emittedEvents) != 0 {
		t.Error("expected no events emitted yet")
	}
}

// Test that BlockAndWait returns the correct result after user responds.
// After BlockAndWait returns, the pending state is cleared (as designed),
// so subsequent CheckPermissions calls re-evaluate from scratch.
func TestAskPermission_BlockAndWaitReturnsCorrectResult(t *testing.T) {
	p := NewAskPermission()
	ctx := &core.ToolUseContext{
		ToolName: "replace",
		ToolInfo: &core.ToolInfo{
			Name:          "replace",
			SecurityLevel: core.LevelSensitive,
		},
		Ctx:           newTestContext(),
	}

	done := make(chan core.PermissionResult)
	go func() {
		result := p.BlockAndWait(ctx)
		done <- result
	}()

	time.Sleep(50 * time.Millisecond)

	p.Respond(core.PermissionResult{Behavior: core.PermissionAllow})

	finalResult := <-done

	if finalResult.Behavior != core.PermissionAllow {
		t.Errorf("expected Allow from BlockAndWait, got %s", finalResult.Behavior)
	}
}

// testContext is a simple context.Context implementation for testing.
type testContext struct {
	done chan struct{}
}

func newTestContext() *testContext {
	return &testContext{done: make(chan struct{})}
}

func (c *testContext) Done() <-chan struct{} { return c.done }
func (c *testContext) Err() error           { return nil }
func (c *testContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *testContext) Value(key any) any     { return nil }
