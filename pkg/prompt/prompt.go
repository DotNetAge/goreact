package prompt

// PromptManager 提示管理器接口
type PromptManager interface {
	// RegisterTemplate 注册提示模板
	RegisterTemplate(name string, template string)
	// GetTemplate 获取提示模板
	GetTemplate(name string) string
	// RenderTemplate 渲染提示模板
	RenderTemplate(name string, variables map[string]interface{}) string
}
