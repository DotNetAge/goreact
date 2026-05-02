package core

import "context"

type FuncTool interface {
	Info() *ToolInfo
	Execute(ctx context.Context, params map[string]any) (any, error)
}

type ToolInfo struct {
	Name              string        `json:"name" yaml:"name"`
	Description       string        `json:"description" yaml:"description"`
	Prompt            string        `json:"prompt,omitempty" yaml:"prompt,omitempty"` // 详尽的工具使用说明，第一轮加载到 NativeTools.description
	Tags              []string      `json:"tags" yaml:"tags"`
	SecurityLevel     SecurityLevel `json:"security_level" yaml:"security_level"`
	IsIdempotent      bool          `json:"is_idempotent" yaml:"is_idempotent"`
	Parameters        []Parameter   `json:"parameters" yaml:"parameters"`
	ReturnType        string        `json:"return_type" yaml:"return_type"`
	Examples          []string      `json:"examples" yaml:"examples"`
	MaxResultSizeChars int         `json:"max_result_size_chars,omitempty" yaml:"max_result_size_chars,omitempty"`
	IsReadOnly        bool          `json:"is_read_only,omitempty" yaml:"is_read_only,omitempty"`
}

type Parameter struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Required    bool   `json:"required" yaml:"required"`
	Default     any    `json:"default" yaml:"default"`
	Description string `json:"description" yaml:"description"`
	Enum        []any  `json:"enum" yaml:"enum"`
}
