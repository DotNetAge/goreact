package actor

import (
	"context"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/tools"
)

type dummyStep struct {
	name string
}

func (s *dummyStep) Name() string { return s.name }
func (s *dummyStep) Execute(ctx context.Context, state *core.PipelineContext) error {
	return nil
}

type dummyTool struct {
	name  string
	level tools.SecurityLevel
}

func (t *dummyTool) Name() string                       { return t.name }
func (t *dummyTool) Description() string                { return "Dummy Tool" }
func (t *dummyTool) SecurityLevel() tools.SecurityLevel { return t.level }
func (t *dummyTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	return nil, nil
}

func TestSecurityHook_Approval(t *testing.T) {
	tm := tools.NewSimpleManager()
	tm.Register(&dummyTool{name: "safe_tool", level: tools.LevelSafe})
	tm.Register(&dummyTool{name: "high_risk_tool", level: tools.LevelHighRisk})

	// Setup context with a HighRisk action trace
	ctx := core.NewPipelineContext(context.Background(), "test-session", "do dangerous things")
	ctx.AppendTrace(&core.Trace{
		Action: &core.Action{
			Name:  "high_risk_tool",
			Input: map[string]any{"arg": 1},
		},
	})

	var approvalCalled bool
	hook := NewSecurityHook(tm, nil, func(c *core.PipelineContext, tool tools.Tool, input map[string]any) (bool, error) {
		approvalCalled = true
		return true, nil // Approved
	})

	step := &dummyStep{name: "actor"}

	// 1. Test HighRisk tool triggers approval
	hook.OnStepStart(ctx, step, ctx)

	if !approvalCalled {
		t.Error("Expected approval to be called for high-risk tool")
	}

	if ctx.Error != nil {
		t.Errorf("Expected no error after approval, got: %v", ctx.Error)
	}

	// The whitelist should have been set
	val, ok := ctx.Get("whitelist:high_risk_tool")
	if !ok || val != true {
		t.Error("Expected tool to be whitelisted for the session after approval")
	}

	// 2. Test Whitelist bypasses approval
	approvalCalled = false // reset flag
	hook.OnStepStart(ctx, step, ctx)

	if approvalCalled {
		t.Error("Expected approval NOT to be called again for whitelisted tool")
	}

	if ctx.Error != nil {
		t.Errorf("Expected no error, got: %v", ctx.Error)
	}
}

func TestSecurityHook_Rejection(t *testing.T) {
	tm := tools.NewSimpleManager()
	tm.Register(&dummyTool{name: "high_risk_tool", level: tools.LevelHighRisk})

	ctx := core.NewPipelineContext(context.Background(), "test-session", "do dangerous things")
	ctx.AppendTrace(&core.Trace{
		Action: &core.Action{
			Name: "high_risk_tool",
		},
	})

	hook := NewSecurityHook(tm, nil, func(c *core.PipelineContext, tool tools.Tool, input map[string]any) (bool, error) {
		return false, nil // Rejected
	})

	step := &dummyStep{name: "actor"}
	hook.OnStepStart(ctx, step, ctx)

	if ctx.Error != ErrOperationRejectedByUser {
		t.Errorf("Expected ErrOperationRejectedByUser, got %v", ctx.Error)
	}
}
