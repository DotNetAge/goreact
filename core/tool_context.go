package core

import "context"

type toolCtxKeyType struct{}

var toolCtxKey toolCtxKeyType

// ToolContext carries runtime infrastructure accessible to all tools via context.
// Tools that need special capabilities (e.g., delegate creating sub-agents)
// extract it via GetToolContext(). Tools that don't need it ignore it entirely.
type ToolContext struct {
	// EmitEvent publishes a ReactEvent to the agent's event bus.
	EmitEvent func(ReactEvent)

	// ResultStore stores and retrieves async task results.
	ResultStore *ResultStore
}

// WithToolContext injects a ToolContext into the given context.
func WithToolContext(ctx context.Context, tc *ToolContext) context.Context {
	return context.WithValue(ctx, toolCtxKey, tc)
}

// GetToolContext extracts the ToolContext from context.
// Returns nil if not set (tools should handle nil gracefully).
func GetToolContext(ctx context.Context) *ToolContext {
	tc, _ := ctx.Value(toolCtxKey).(*ToolContext)
	return tc
}
