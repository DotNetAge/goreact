package prompt

import (
	"fmt"
	"strings"
)

// DefaultPromptManager 默认提示管理器
type DefaultPromptManager struct {
	templates map[string]string
}

// NewDefaultPromptManager 创建默认提示管理器
func NewDefaultPromptManager() *DefaultPromptManager {
	return &DefaultPromptManager{
		templates: make(map[string]string),
	}
}

// RegisterTemplate 注册提示模板
func (m *DefaultPromptManager) RegisterTemplate(name string, template string) {
	m.templates[name] = template
}

// GetTemplate 获取提示模板
func (m *DefaultPromptManager) GetTemplate(name string) string {
	return m.templates[name]
}

// RenderTemplate 渲染提示模板
func (m *DefaultPromptManager) RenderTemplate(name string, variables map[string]interface{}) string {
	template := m.templates[name]
	if template == "" {
		return ""
	}

	// 简单的模板渲染：替换 {{var}} 形式的变量
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result
}
