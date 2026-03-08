package tool

import (
	"fmt"
	"sync"
)

// Manager 工具管理器
type Manager struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewManager 创建工具管理器
func NewManager() *Manager {
	return &Manager{
		tools: make(map[string]Tool),
	}
}

// RegisterTool 注册单个工具
func (m *Manager) RegisterTool(tool Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name()] = tool
}

// RegisterTools 注册多个工具
func (m *Manager) RegisterTools(tools ...Tool) {
	for _, tool := range tools {
		m.RegisterTool(tool)
	}
}

// GetTool 获取工具
func (m *Manager) GetTool(name string) (Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tool, ok := m.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// ExecuteTool 执行工具
func (m *Manager) ExecuteTool(name string, params map[string]interface{}) (interface{}, error) {
	tool, err := m.GetTool(name)
	if err != nil {
		return nil, err
	}
	return tool.Execute(params)
}

// ListTools 列出所有工具
func (m *Manager) ListTools() []Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tools := make([]Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolDescriptions 获取所有工具的描述（用于 LLM prompt）
func (m *Manager) GetToolDescriptions() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.tools) == 0 {
		return "No tools available."
	}

	desc := "Available tools:\n"
	for _, tool := range m.tools {
		desc += fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description())
	}
	return desc
}
