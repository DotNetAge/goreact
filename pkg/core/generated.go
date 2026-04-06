package core

import (
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// GeneratedSkillNode represents a skill generated through evolution
type GeneratedSkillNode struct {
	BaseNode
	Description string              `json:"description" yaml:"description"`
	Content     string              `json:"content" yaml:"content"`
	FilePath    string              `json:"file_path" yaml:"file_path"`
	Parameters  []SkillParameter    `json:"parameters" yaml:"parameters"`
	Examples    []string            `json:"examples" yaml:"examples"`
	SourceSession string            `json:"source_session" yaml:"source_session"`
	Status      common.GeneratedStatus `json:"status" yaml:"status"`
	AllowedTools []string           `json:"allowed_tools" yaml:"allowed_tools"`
	Template    string              `json:"template" yaml:"template"`
}

// SkillParameter represents a parameter for a skill
type SkillParameter struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Required    bool   `json:"required" yaml:"required"`
	Default     string `json:"default" yaml:"default"`
	Description string `json:"description" yaml:"description"`
}

// NewGeneratedSkillNode creates a new GeneratedSkillNode
func NewGeneratedSkillNode(name, description, content string) *GeneratedSkillNode {
	return &GeneratedSkillNode{
		BaseNode: BaseNode{
			Name:        name,
			NodeType:    common.NodeTypeGeneratedSkill,
			Description: description,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		Description: description,
		Content:     content,
		Parameters:  []SkillParameter{},
		Examples:    []string{},
		Status:      common.GeneratedStatusDraft,
	}
}

// GeneratedToolNode represents a tool generated through evolution
type GeneratedToolNode struct {
	BaseNode
	Description   string                 `json:"description" yaml:"description"`
	FilePath      string                 `json:"file_path" yaml:"file_path"`
	Code          string                 `json:"code" yaml:"code"`
	PackageName   string                 `json:"package_name" yaml:"package_name"`
	Parameters    []ToolParameter        `json:"parameters" yaml:"parameters"`
	ReturnType    string                 `json:"return_type" yaml:"return_type"`
	SecurityLevel common.SecurityLevel   `json:"security_level" yaml:"security_level"`
	ToolType      common.ToolType        `json:"tool_type" yaml:"tool_type"`
	SourceSession string                 `json:"source_session" yaml:"source_session"`
	Status        common.GeneratedStatus `json:"status" yaml:"status"`
	Schema        map[string]any         `json:"schema" yaml:"schema"`
	Endpoint      string                 `json:"endpoint" yaml:"endpoint"`
}

// ToolParameter represents a parameter for a tool
type ToolParameter struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description" yaml:"description"`
	Required    bool   `json:"required" yaml:"required"`
}

// NewGeneratedToolNode creates a new GeneratedToolNode
func NewGeneratedToolNode(name, description, code string) *GeneratedToolNode {
	return &GeneratedToolNode{
		BaseNode: BaseNode{
			Name:        name,
			NodeType:    common.NodeTypeGeneratedTool,
			Description: description,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		Description:   description,
		Code:          code,
		Parameters:    []ToolParameter{},
		SecurityLevel: common.LevelSafe,
		ToolType:      common.ToolTypePython,
		Status:        common.GeneratedStatusDraft,
		Schema:        make(map[string]any),
	}
}

// GeneratedSkillBuilder builds a GeneratedSkillNode
type GeneratedSkillBuilder struct {
	skill *GeneratedSkillNode
}

// NewGeneratedSkillBuilder creates a new skill builder
func NewGeneratedSkillBuilder(name string) *GeneratedSkillBuilder {
	return &GeneratedSkillBuilder{
		skill: &GeneratedSkillNode{
			BaseNode: BaseNode{
				Name:        name,
				NodeType:    common.NodeTypeGeneratedSkill,
				CreatedAt:   time.Now(),
				Metadata:    make(map[string]any),
			},
			Parameters: []SkillParameter{},
			Examples:   []string{},
			Status:     common.GeneratedStatusDraft,
		},
	}
}

// WithDescription sets the description
func (b *GeneratedSkillBuilder) WithDescription(desc string) *GeneratedSkillBuilder {
	b.skill.Description = desc
	b.skill.BaseNode.Description = desc
	return b
}

// WithContent sets the content
func (b *GeneratedSkillBuilder) WithContent(content string) *GeneratedSkillBuilder {
	b.skill.Content = content
	return b
}

// WithFilePath sets the file path
func (b *GeneratedSkillBuilder) WithFilePath(path string) *GeneratedSkillBuilder {
	b.skill.FilePath = path
	return b
}

// WithSourceSession sets the source session
func (b *GeneratedSkillBuilder) WithSourceSession(sessionName string) *GeneratedSkillBuilder {
	b.skill.SourceSession = sessionName
	return b
}

// AddParameter adds a parameter
func (b *GeneratedSkillBuilder) AddParameter(name, typ string, required bool, desc string) *GeneratedSkillBuilder {
	b.skill.Parameters = append(b.skill.Parameters, SkillParameter{
		Name:        name,
		Type:        typ,
		Required:    required,
		Description: desc,
	})
	return b
}

// AddExample adds an example
func (b *GeneratedSkillBuilder) AddExample(example string) *GeneratedSkillBuilder {
	b.skill.Examples = append(b.skill.Examples, example)
	return b
}

// WithAllowedTools sets allowed tools
func (b *GeneratedSkillBuilder) WithAllowedTools(tools []string) *GeneratedSkillBuilder {
	b.skill.AllowedTools = tools
	return b
}

// Build builds the skill
func (b *GeneratedSkillBuilder) Build() *GeneratedSkillNode {
	return b.skill
}

// GeneratedToolBuilder builds a GeneratedToolNode
type GeneratedToolBuilder struct {
	tool *GeneratedToolNode
}

// NewGeneratedToolBuilder creates a new tool builder
func NewGeneratedToolBuilder(name string) *GeneratedToolBuilder {
	return &GeneratedToolBuilder{
		tool: &GeneratedToolNode{
			BaseNode: BaseNode{
				Name:        name,
				NodeType:    common.NodeTypeGeneratedTool,
				CreatedAt:   time.Now(),
				Metadata:    make(map[string]any),
			},
			Parameters:    []ToolParameter{},
			SecurityLevel: common.LevelSafe,
			ToolType:      common.ToolTypePython,
			Status:        common.GeneratedStatusDraft,
			Schema:        make(map[string]any),
		},
	}
}

// WithDescription sets the description
func (b *GeneratedToolBuilder) WithDescription(desc string) *GeneratedToolBuilder {
	b.tool.Description = desc
	b.tool.BaseNode.Description = desc
	return b
}

// WithCode sets the code
func (b *GeneratedToolBuilder) WithCode(code string) *GeneratedToolBuilder {
	b.tool.Code = code
	return b
}

// WithFilePath sets the file path
func (b *GeneratedToolBuilder) WithFilePath(path string) *GeneratedToolBuilder {
	b.tool.FilePath = path
	return b
}

// WithSourceSession sets the source session
func (b *GeneratedToolBuilder) WithSourceSession(sessionName string) *GeneratedToolBuilder {
	b.tool.SourceSession = sessionName
	return b
}

// WithSecurityLevel sets the security level
func (b *GeneratedToolBuilder) WithSecurityLevel(level common.SecurityLevel) *GeneratedToolBuilder {
	b.tool.SecurityLevel = level
	return b
}

// WithToolType sets the tool type
func (b *GeneratedToolBuilder) WithToolType(toolType common.ToolType) *GeneratedToolBuilder {
	b.tool.ToolType = toolType
	return b
}

// AddParameter adds a parameter
func (b *GeneratedToolBuilder) AddParameter(name, typ, desc string, required bool) *GeneratedToolBuilder {
	b.tool.Parameters = append(b.tool.Parameters, ToolParameter{
		Name:        name,
		Type:        typ,
		Description: desc,
		Required:    required,
	})
	return b
}

// WithReturnType sets the return type
func (b *GeneratedToolBuilder) WithReturnType(returnType string) *GeneratedToolBuilder {
	b.tool.ReturnType = returnType
	return b
}

// WithSchema sets the schema
func (b *GeneratedToolBuilder) WithSchema(schema map[string]any) *GeneratedToolBuilder {
	b.tool.Schema = schema
	return b
}

// Build builds the tool
func (b *GeneratedToolBuilder) Build() *GeneratedToolNode {
	return b.tool
}
