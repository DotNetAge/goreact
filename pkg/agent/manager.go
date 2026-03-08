package agent

import (
	"sync"
)

// Manager 智能体管理器
type Manager struct {
	agents map[string]*Agent
	mutex  sync.RWMutex
}

// NewManager 创建智能体管理器
func NewManager() *Manager {
	return &Manager{
		agents: make(map[string]*Agent),
	}
}

// Register 注册智能体
func (m *Manager) Register(agent *Agent) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.agents[agent.Name] = agent
}

// Get 获取智能体
func (m *Manager) Get(name string) *Agent {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.agents[name]
}

// List 列出所有智能体
func (m *Manager) List() []*Agent {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}
