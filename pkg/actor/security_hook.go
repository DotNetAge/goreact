package actor

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
	"github.com/DotNetAge/goreact/pkg/tools"
)

// ErrOperationRejectedByUser is returned when a human in the loop rejects a high-risk tool execution.
var ErrOperationRejectedByUser = fmt.Errorf("operation rejected by user")

// SecurityHook handles Human-In-The-Loop (HITL) for high-risk tool execution using gochat pipeline hooks.
type SecurityHook struct {
	toolManager tools.Manager
	memoryBank  memory.MemoryBank
	approver    func(ctx *core.PipelineContext, tool tools.Tool, input map[string]any) (bool, error)
}

// Ensure SecurityHook implements pipeline.Hook for our PipelineContext
var _ pipeline.Hook[*core.PipelineContext] = (*SecurityHook)(nil)

// NewSecurityHook creates a new SecurityHook.
func NewSecurityHook(tm tools.Manager, mb memory.MemoryBank, approver func(ctx *core.PipelineContext, tool tools.Tool, input map[string]any) (bool, error)) *SecurityHook {
	return &SecurityHook{
		toolManager: tm,
		memoryBank:  mb,
		approver:    approver,
	}
}

// OnStepStart intercepts the actor step to block and wait for human authorization for high-risk tools.
func (h *SecurityHook) OnStepStart(ctx context.Context, step pipeline.Step[*core.PipelineContext], state *core.PipelineContext) {
	if step.Name() != "actor" {
		return
	}

	// If there's already a pipeline error, don't intervene
	if state.Error != nil {
		return
	}

	lastTrace := state.LastTrace()
	if lastTrace == nil || lastTrace.Action == nil {
		return
	}

	tool, exists := h.toolManager.GetTool(lastTrace.Action.Name)
	if !exists {
		return
	}

	// Only LevelHighRisk requires authorization by default.
	if tool.SecurityLevel() < tools.LevelHighRisk {
		return
	}

	// 1. Check Session-level Whitelist
	whitelistKey := fmt.Sprintf("whitelist:%s", tool.Name())
	if val, ok := state.Get(whitelistKey); ok && val == true {
		state.Logger.Debug("Bypassing HITL: Tool is whitelisted in current session", "tool", tool.Name())
		return
	}

	// 2. Check Long-term Whitelist (if memory manager is available)
	if h.memoryBank != nil {
		if val, err := h.memoryBank.Working().Retrieve(ctx, state.SessionID, whitelistKey); err == nil && val == true {
			state.Logger.Debug("Bypassing HITL: Tool is whitelisted in long-term memory", "tool", tool.Name())
			return
		}
	}

	// 3. Trigger HITL Authorization
	if h.approver == nil {
		state.Error = fmt.Errorf("high-risk tool %q requires human authorization, but no approver is configured", tool.Name())
		return
	}

	state.Logger.Warn("High-risk tool execution detected, waiting for human authorization...", "tool", tool.Name())
	approved, err := h.approver(state, tool, lastTrace.Action.Input)
	if err != nil {
		state.Error = fmt.Errorf("authorization process failed: %w", err)
		return
	}

	if !approved {
		state.Logger.Info("High-risk tool execution REJECTED by user", "tool", tool.Name())
		state.Error = ErrOperationRejectedByUser
		return
	}

	state.Logger.Info("High-risk tool execution APPROVED by user", "tool", tool.Name())

	// Auto-whitelist for this session after approval
	state.Set(whitelistKey, true)
}

func (h *SecurityHook) OnStepError(ctx context.Context, step pipeline.Step[*core.PipelineContext], state *core.PipelineContext, err error) {
}

func (h *SecurityHook) OnStepComplete(ctx context.Context, step pipeline.Step[*core.PipelineContext], state *core.PipelineContext) {
}
