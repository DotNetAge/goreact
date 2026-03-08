package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// DefaultMemoryManager 默认内存管理器
type DefaultMemoryManager struct {
	memories map[string]map[string]interface{}
	mutex    sync.RWMutex
	persistPath string
}

// NewDefaultMemoryManager 创建默认内存管理器
func NewDefaultMemoryManager(persistPath string) *DefaultMemoryManager {
	if persistPath == "" {
		persistPath = "./memory"
	}

	// 确保持久化目录存在
	if err := os.MkdirAll(persistPath, 0755); err != nil {
		panic(err)
	}

	return &DefaultMemoryManager{
		memories:    make(map[string]map[string]interface{}),
		persistPath: persistPath,
	}
}

// Store 存储内存数据
func (m *DefaultMemoryManager) Store(sessionId string, key string, value interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.memories[sessionId]; !ok {
		m.memories[sessionId] = make(map[string]interface{})
	}

	m.memories[sessionId][key] = value
}

// Retrieve 检索内存数据
func (m *DefaultMemoryManager) Retrieve(sessionId string, key string) interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if session, ok := m.memories[sessionId]; ok {
		return session[key]
	}

	return nil
}

// Compress 压缩内存数据
func (m *DefaultMemoryManager) Compress(sessionId string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 简单的压缩逻辑：移除空值
	if session, ok := m.memories[sessionId]; ok {
		for key, value := range session {
			if value == nil {
				delete(session, key)
			}
		}
	}

	return nil
}

// Persist 持久化内存数据
func (m *DefaultMemoryManager) Persist(sessionId string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, ok := m.memories[sessionId]
	if !ok {
		return errors.New("session not found")
	}

	// 序列化内存数据
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	// 写入文件
	filePath := filepath.Join(m.persistPath, sessionId+"_memory.json")
	return os.WriteFile(filePath, data, 0644)
}

// Load 加载内存数据
func (m *DefaultMemoryManager) Load(sessionId string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 读取文件
	filePath := filepath.Join(m.persistPath, sessionId+"_memory.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 反序列化内存数据
	session := make(map[string]interface{})
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	m.memories[sessionId] = session
	return nil
}
