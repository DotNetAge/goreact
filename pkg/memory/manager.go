package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultMemoryManager is a basic in-memory KV store for session states.
// It is NOT a semantic RAG memory, but satisfies the Manager interface.
type DefaultMemoryManager struct {
	memories    map[string]map[string]interface{}
	mutex       sync.RWMutex
	persistPath string
}

// NewDefaultMemoryManager creates a default memory manager
func NewDefaultMemoryManager(persistPath string) (*DefaultMemoryManager, error) {
	if persistPath == "" {
		persistPath = "./memory"
	}

	if err := os.MkdirAll(persistPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	return &DefaultMemoryManager{
		memories:    make(map[string]map[string]interface{}),
		persistPath: persistPath,
	}, nil
}

func (m *DefaultMemoryManager) Store(ctx context.Context, sessionID string, key string, value interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.memories[sessionID]; !ok {
		m.memories[sessionID] = make(map[string]interface{})
	}
	m.memories[sessionID][key] = value
	return nil
}

func (m *DefaultMemoryManager) Retrieve(ctx context.Context, sessionID string, key string) (interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if session, ok := m.memories[sessionID]; ok {
		if val, exists := session[key]; exists {
			return val, nil
		}
	}
	return nil, nil // Or an explicit ErrNotFound
}

// Recall in the default manager just returns a basic JSON dump of all keys 
// (which is a naive replacement for proper RAG semantic recall).
func (m *DefaultMemoryManager) Recall(ctx context.Context, sessionID string, intent string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, ok := m.memories[sessionID]
	if !ok || len(session) == 0 {
		return "", nil // No memory found
	}
	
	// Just dump the entire state
	data, _ := json.Marshal(session)
	return fmt.Sprintf("Known state/preferences: %s", string(data)), nil
}

func (m *DefaultMemoryManager) Compress(ctx context.Context, sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if session, ok := m.memories[sessionID]; ok {
		for key, value := range session {
			if value == nil {
				delete(session, key)
			}
		}
	}
	return nil
}

func (m *DefaultMemoryManager) Persist(ctx context.Context, sessionID string) error {
	m.mutex.RLock() // RLock is fine since json.Marshal only reads
	defer m.mutex.RUnlock()

	session, ok := m.memories[sessionID]
	if !ok {
		return errors.New("session not found")
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(m.persistPath, sessionID+"_memory.json")
	return os.WriteFile(filePath, data, 0644)
}

func (m *DefaultMemoryManager) Load(ctx context.Context, sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	filePath := filepath.Join(m.persistPath, sessionID+"_memory.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // OK if doesn't exist yet
		}
		return err
	}

	session := make(map[string]interface{})
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	m.memories[sessionID] = session
	return nil
}
