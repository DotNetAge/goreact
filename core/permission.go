package core

import "context"

// PermissionBehavior represents the outcome of a permission check.
type PermissionBehavior string

const (
	// PermissionAllow permits the operation to proceed.
	PermissionAllow PermissionBehavior = "allow"
	// PermissionDeny blocks the operation. The tool will NOT execute.
	PermissionDeny PermissionBehavior = "deny"
	// PermissionAsk suspends execution until the user provides a decision.
	// The permission system will block and wait for an external response.
	PermissionAsk PermissionBehavior = "ask"
)

// PermissionResult represents the outcome of a permission/authorization check.
type PermissionResult struct {
	Behavior PermissionBehavior `json:"behavior"`

	// Message explains the reason when denied, or the question when asking.
	Message string `json:"message,omitempty"`

	// UpdatedInput allows hooks or user to modify tool parameters before execution.
	// Only meaningful when Behavior is Allow or Ask (user approved with modifications).
	UpdatedInput map[string]any `json:"updated_input,omitempty"`
}

// ToolUseContext provides contextual information for permission decisions and hooks.
type ToolUseContext struct {
	// SessionID identifies the conversation session.
	SessionID string

	// TaskID identifies the source task ("main" or subagent task ID).
	TaskID string

	// ToolName is the name of the tool being called.
	ToolName string

	// ToolInfo is the metadata of the tool being called.
	ToolInfo *ToolInfo

	// Params are the original parameters provided by the LLM.
	Params map[string]any

	// Iteration is the current T-A-O cycle number.
	Iteration int

	// Ctx is the context.Context for cancellation support.
	Ctx context.Context
}

// ToolPermissionChecker determines whether a tool execution is permitted.
// Implementations can inspect tool metadata, parameters, and context to make
// authorization decisions. The checker may return PermissionAsk to suspend
// execution until the user responds (via the PermissionResponder mechanism).
//
// This interface is designed to be composable: a chain of checkers can be
// combined, where each checker's result feeds into the next.
type ToolPermissionChecker interface {
	// CheckPermissions evaluates whether the tool call should be allowed.
	// The returned PermissionResult may have Behavior=Ask, in which case
	// the caller must wait for user input before proceeding.
	CheckPermissions(ctx *ToolUseContext) PermissionResult
}

// PermissionResponder allows external code to respond to a pending permission request.
// When CheckPermissions returns PermissionAsk, the system blocks until
// either Respond() or RespondError() is called.
type PermissionResponder interface {
	// Respond delivers the user's permission decision.
	Respond(result PermissionResult)

	// RespondError delivers an error (e.g., timeout, cancellation).
	RespondError(err error)

	// IsWaiting returns true if the system is currently blocked waiting for a response.
	IsWaiting() bool

	// BlockAndWait blocks until the user responds or the context is cancelled.
	// Returns the final permission decision after the user responds.
	BlockAndWait(ctx *ToolUseContext) PermissionResult
}
