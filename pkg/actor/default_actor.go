package actor

import (
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/tools"
)

// defaultActor represents the hands of the system.
// It relies on a tools.Manager to resolve what tool matches the Thinker's requested Action.
type defaultActor struct {
	toolManager tools.Manager
}

// Option configures the DefaultActor.
type Option func(*defaultActor)

// WithToolManager registers the centralized tools manager (used for Tool routing).
func WithToolManager(mgr tools.Manager) Option {
	return func(a *defaultActor) {
		a.toolManager = mgr
	}
}

// Default creates a basic, synchronous Actor implementation with semantic tool routing.
func Default(opts ...Option) Actor {
	a := &defaultActor{
		toolManager: tools.NewSimpleManager(), // fallback empty manager
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *defaultActor) Act(ctx *core.PipelineContext) error {
	lastTrace := ctx.LastTrace()

	if lastTrace == nil || lastTrace.Action == nil {
		return nil
	}
	action := lastTrace.Action

	// 1. Tool Routing: Ask the RAG/Semantic Manager for the exact tool
	tool, exists := a.toolManager.GetTool(action.Name)
	if !exists {
		ctx.Logger.Warn("Actor intercepted an invalid hallucinated tool call", "tool", action.Name)
		err := fmt.Errorf("tool %q is not registered or available in this context", action.Name)
		ctx.Set("raw_output", nil)
		ctx.Set("raw_error", err)
		return nil
	}

	// 2. Execution Phase
	ctx.Logger.Info("Actor is executing tool", "tool", action.Name, "input", action.Input)
	start := time.Now()

	result, err := tool.Execute(ctx, action.Input)

	ctx.Metrics.RecordTimer(fmt.Sprintf("tool_%s_latency", action.Name), time.Since(start), nil)

	// 3. Output Hand-off
	if err != nil {
		ctx.Logger.Error(err, "Tool execution failed", "tool", action.Name)
		ctx.Set("raw_output", nil)
		ctx.Set("raw_error", err)
		return nil
	}

	ctx.Logger.Debug("Tool execution succeeded", "tool", action.Name)
	ctx.Set("raw_output", result)
	ctx.Set("raw_error", nil)

	return nil
}
