package core

import (
	"context"
)

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
