package core

import (
	"context"
	"time"
)

// SecurityPolicy is a function that determines whether a tool execution is allowed.
// It receives the tool name and its security level; return true to allow, false to block.
//
// Deprecated: Use SetPermissionChecker and AddHook instead for the full permission pipeline.
type SecurityPolicy func(toolName string, level SecurityLevel) bool

// ToolRegistryInterface defines the contract for tool lifecycle management.
// This interface covers both tool operations and configuration methods,
// enabling full replacement or selective enhancement via embedding.
type ToolRegistryInterface interface {
	// --- Core Operations ---

	// Register adds a tool. Returns error if name already exists.
	Register(tool FuncTool) error

	// Get returns a tool by exact name match.
	Get(name string) (FuncTool, bool)

	// All returns all registered tools.
	All() []FuncTool

	// ToToolInfos extracts metadata from all tools for prompt building.
	ToToolInfos() []ToolInfo

	// GetWithSemantic finds a tool by name, falling back to memory-based
	// semantic search if exact match fails.
	GetWithSemantic(ctx context.Context, name string, intent string) (FuncTool, bool)

	// ExecuteTool runs a tool with full permission pipeline support.
	ExecuteTool(ctx context.Context, name string, params map[string]any) (string, time.Duration, error)

	// --- Configuration Methods ---

	// SetSecurityPolicy sets the legacy security policy (deprecated).
	SetSecurityPolicy(policy SecurityPolicy)

	// SetPermissionChecker sets the fine-grained permission checker.
	SetPermissionChecker(checker ToolPermissionChecker)

	// SetResultStorage configures result persistence for context defense.
	SetResultStorage(storage ToolResultStorage)

	// SetResultLimits configures per-result size limits.
	SetResultLimits(limits ToolResultLimits)

	// SetMemory injects Memory for reflexive semantic search.
	SetMemory(mem Memory)

	// AddHook registers a lifecycle hook (PreToolUse / PostToolUse).
	AddHook(hook Hook)

	// SetEventEmitter sets the callback for permission events.
	SetEventEmitter(fn func(ReactEvent))

	// ResetMessageCharCounter resets per-cycle char tracking.
	ResetMessageCharCounter()
}

// FuncTool represents an executable tool
type FuncTool interface {
	Info() *ToolInfo
	Execute(ctx context.Context, params map[string]any) (any, error) // Run executes the tool
}

// ToolInfo represents tool metadata
type ToolInfo struct {
	Name          string        `json:"name" yaml:"name"`                     // Name is the tool name
	Description   string        `json:"description" yaml:"description"`       // Description is the tool description
	SecurityLevel SecurityLevel `json:"security_level" yaml:"security_level"` // SecurityLevel is the security level
	IsIdempotent  bool          `json:"is_idempotent" yaml:"is_idempotent"`   // IsIdempotent indicates if the tool is idempotent
	Parameters    []Parameter   `json:"parameters" yaml:"parameters"`         // Parameters are the tool parameters
	ReturnType    string        `json:"return_type" yaml:"return_type"`       // ReturnType is the return type
	Examples      []string      `json:"examples" yaml:"examples"`             // Examples are usage examples

	// MaxResultSizeChars overrides the default per-tool result size threshold.
	// Set to -1 (math.MinInt) to disable persistence for this tool (e.g., read tool).
	// Set to 0 to use the global default.
	MaxResultSizeChars int `json:"max_result_size_chars,omitempty" yaml:"max_result_size_chars,omitempty"`

	// IsReadOnly indicates if the tool only reads data without side effects.
	IsReadOnly bool `json:"is_read_only,omitempty" yaml:"is_read_only,omitempty"`
}

// Parameter represents a tool parameter
type Parameter struct {
	Name        string `json:"name" yaml:"name"`               // Name is the parameter name
	Type        string `json:"type" yaml:"type"`               // Type is the parameter type
	Required    bool   `json:"required" yaml:"required"`       // Required indicates if the parameter is required
	Default     any    `json:"default" yaml:"default"`         // Default is the default value
	Description string `json:"description" yaml:"description"` // Description is the parameter description
	Enum        []any  `json:"enum" yaml:"enum"`               // Enum is the list of allowed values
}
