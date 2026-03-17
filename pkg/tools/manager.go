package tools

import (
	"fmt"
	"strings"
	"sync"
)

// Manager 工具管理器
type Manager struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	descCache string // 缓存的工具描述
	dirty     bool   // 标记描述是否需要重新生成
}

// NewManager 创建工具管理器
func NewManager() *Manager {
	return &Manager{
		tools: make(map[string]Tool),
		dirty: true,
	}
}

// RegisterTool 注册单个工具
func (m *Manager) RegisterTool(tool Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name()] = tool
	m.dirty = true
}

// RegisterTools 注册多个工具
func (m *Manager) RegisterTools(tools ...Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, tool := range tools {
		m.tools[tool.Name()] = tool
	}
	m.dirty = true
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
func (m *Manager) ExecuteTool(name string, params map[string]any) (any, error) {
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
// 使用缓存避免重复生成
func (m *Manager) GetToolDescriptions() string {
	m.mu.RLock()
	if !m.dirty {
		desc := m.descCache
		m.mu.RUnlock()
		return desc
	}
	m.mu.RUnlock()

	// 需要写锁来更新缓存
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if !m.dirty {
		return m.descCache
	}

	if len(m.tools) == 0 {
		m.descCache = "No tools available."
		m.dirty = false
		return m.descCache
	}

	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	for _, tool := range m.tools {
		fmt.Fprintf(&sb, "- %s: %s\n", tool.Name(), tool.Description())
	}
	m.descCache = sb.String()
	m.dirty = false
	return m.descCache
}
