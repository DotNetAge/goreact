package core

// HookEventType identifies the type of lifecycle hook.
type HookEventType string

const (
	// HookPreToolUse fires before a tool is executed.
	// Hooks can: block execution, modify input, or make a permission decision.
	HookPreToolUse HookEventType = "pre_tool_use"

	// HookPostToolUse fires after a tool has executed (success or failure).
	// Hooks receive the tool result and can modify the observation.
	HookPostToolUse HookEventType = "post_tool_use"

	// HookSessionStart fires when a new reactor session begins.
	HookSessionStart HookEventType = "session_start"

	// HookStop fires when the reactor is about to stop (termination or error).
	HookStop HookEventType = "stop"
)

// HookResult is the output of a hook execution.
type HookResult struct {
	// PermissionResult, if non-nil, overrides the default permission decision.
	// Only meaningful for PreToolUse hooks.
	*PermissionResult

	// UpdatedInput, if non-nil, replaces the tool's input parameters.
	// Only meaningful for PreToolUse hooks.
	UpdatedInput map[string]any

	// PreventContinuation stops the tool execution entirely.
	// When true, the tool is treated as denied with no error message.
	PreventContinuation bool

	// Message is an optional informational message to log or surface to the user.
	Message string
}

// PostToolUseContext provides context for post-execution hooks.
type PostToolUseContext struct {
	// ToolUseContext from the pre-execution phase.
	*ToolUseContext

	// Result is the tool's execution result (empty string on error).
	Result string

	// Err is the execution error, if any.
	Err error

	// Duration is how long the tool took to execute.
	Duration int64 // milliseconds
}

// Hook is the interface for lifecycle hooks.
// Hooks are called at specific points in the reactor's execution lifecycle
// and can influence control flow (permission, input modification, abort).
type Hook interface {
	// EventType returns which lifecycle event this hook handles.
	EventType() HookEventType

	// Execute runs the hook logic.
	// For PreToolUse: receives *ToolUseContext, returns HookResult.
	// For PostToolUse: receives *PostToolUseContext, returns HookResult.
	// For other events: receives *ToolUseContext, returns HookResult.
	Execute(ctx any) HookResult
}
