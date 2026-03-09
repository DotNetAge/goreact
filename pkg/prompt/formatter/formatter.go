package formatter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ToolDesc 工具描述
type ToolDesc struct {
	Name        string
	Description string
	Parameters  *ParameterSchema
}

// ParameterSchema 参数 Schema
type ParameterSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]*Property   `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Items      *Property              `json:"items,omitempty"`
	Additional map[string]interface{} `json:"-"` // 额外字段
}

// Property 参数属性
type Property struct {
	Type        string        `json:"type"`
	Description string        `json:"description,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Items       *Property     `json:"items,omitempty"`
}

// ToolFormatter 工具格式化器接口
type ToolFormatter interface {
	Format(tools []ToolDesc) string
}

// SimpleTextFormatter 简单文本格式化器
type SimpleTextFormatter struct{}

func NewSimpleTextFormatter() *SimpleTextFormatter {
	return &SimpleTextFormatter{}
}

func (f *SimpleTextFormatter) Format(tools []ToolDesc) string {
	if len(tools) == 0 {
		return "No tools available"
	}

	var sb strings.Builder
	for i, tool := range tools {
		sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, tool.Name, tool.Description))
	}
	return sb.String()
}

// JSONSchemaFormatter JSON Schema 格式化器
type JSONSchemaFormatter struct {
	Indent bool
}

func NewJSONSchemaFormatter(indent bool) *JSONSchemaFormatter {
	return &JSONSchemaFormatter{Indent: indent}
}

func (f *JSONSchemaFormatter) Format(tools []ToolDesc) string {
	if len(tools) == 0 {
		return "[]"
	}

	var toolSchemas []map[string]interface{}
	for _, tool := range tools {
		schema := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		}

		if tool.Parameters != nil {
			schema["parameters"] = f.formatParameters(tool.Parameters)
		}

		toolSchemas = append(toolSchemas, schema)
	}

	var data []byte
	var err error
	if f.Indent {
		data, err = json.MarshalIndent(toolSchemas, "", "  ")
	} else {
		data, err = json.Marshal(toolSchemas)
	}

	if err != nil {
		return "[]"
	}

	return string(data)
}

func (f *JSONSchemaFormatter) formatParameters(params *ParameterSchema) map[string]interface{} {
	result := map[string]interface{}{
		"type": params.Type,
	}

	if len(params.Properties) > 0 {
		props := make(map[string]interface{})
		for name, prop := range params.Properties {
			props[name] = f.formatProperty(prop)
		}
		result["properties"] = props
	}

	if len(params.Required) > 0 {
		result["required"] = params.Required
	}

	if params.Items != nil {
		result["items"] = f.formatProperty(params.Items)
	}

	// 添加额外字段
	for k, v := range params.Additional {
		result[k] = v
	}

	return result
}

func (f *JSONSchemaFormatter) formatProperty(prop *Property) map[string]interface{} {
	result := map[string]interface{}{
		"type": prop.Type,
	}

	if prop.Description != "" {
		result["description"] = prop.Description
	}

	if len(prop.Enum) > 0 {
		result["enum"] = prop.Enum
	}

	if prop.Default != nil {
		result["default"] = prop.Default
	}

	if prop.Items != nil {
		result["items"] = f.formatProperty(prop.Items)
	}

	return result
}

// MarkdownFormatter Markdown 格式化器
type MarkdownFormatter struct{}

func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{}
}

func (f *MarkdownFormatter) Format(tools []ToolDesc) string {
	if len(tools) == 0 {
		return "No tools available"
	}

	var sb strings.Builder
	sb.WriteString("## Available Tools\n\n")

	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("### %s\n\n", tool.Name))
		sb.WriteString(fmt.Sprintf("%s\n\n", tool.Description))

		if tool.Parameters != nil && len(tool.Parameters.Properties) > 0 {
			sb.WriteString("**Parameters:**\n\n")
			for name, prop := range tool.Parameters.Properties {
				required := ""
				if f.isRequired(name, tool.Parameters.Required) {
					required = " (required)"
				}
				sb.WriteString(fmt.Sprintf("- `%s` (%s)%s", name, prop.Type, required))
				if prop.Description != "" {
					sb.WriteString(fmt.Sprintf(": %s", prop.Description))
				}
				if len(prop.Enum) > 0 {
					sb.WriteString(fmt.Sprintf(" - Options: %v", prop.Enum))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (f *MarkdownFormatter) isRequired(name string, required []string) bool {
	for _, r := range required {
		if r == name {
			return true
		}
	}
	return false
}

// CompactFormatter 紧凑格式化器（节省 tokens）
type CompactFormatter struct{}

func NewCompactFormatter() *CompactFormatter {
	return &CompactFormatter{}
}

func (f *CompactFormatter) Format(tools []ToolDesc) string {
	if len(tools) == 0 {
		return "No tools"
	}

	var parts []string
	for _, tool := range tools {
		parts = append(parts, fmt.Sprintf("%s(%s)", tool.Name, tool.Description))
	}

	return strings.Join(parts, "; ")
}
