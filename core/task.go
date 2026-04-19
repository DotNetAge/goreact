package core

import (
	"fmt"
	"sync"
	"time"
)

// TaskStatus defines the current state of a task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// Task represents a unit of work that can be handled by an agent.
type Task struct {
	ID          string     `json:"id"`
	ParentID    string     `json:"parent_id,omitempty"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Input       string     `json:"input"`
	Output      string     `json:"output,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TaskManager handles task lifecycle and persistence.
type TaskManager interface {
	CreateTask(parentID string, description string, input string) (*Task, error)
	UpdateTaskStatus(id string, status TaskStatus, output string, err string) error
	GetTask(id string) (*Task, error)
	ListSubTasks(parentID string) ([]*Task, error)
}

// InMemoryTaskManager is a simple in-memory implementation of TaskManager.
type InMemoryTaskManager struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewInMemoryTaskManager creates a new InMemoryTaskManager.
func NewInMemoryTaskManager() *InMemoryTaskManager {
	return &InMemoryTaskManager{
		tasks: make(map[string]*Task),
	}
}

func (m *InMemoryTaskManager) CreateTask(parentID string, description string, input string) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("task_%d", len(m.tasks)+1)
	task := &Task{
		ID:          id,
		ParentID:    parentID,
		Description: description,
		Input:       input,
		Status:      TaskStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.tasks[id] = task
	return task, nil
}

func (m *InMemoryTaskManager) UpdateTaskStatus(id string, status TaskStatus, output string, err string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	task.Status = status
	task.Output = output
	task.Error = err
	task.UpdatedAt = time.Now()
	return nil
}

func (m *InMemoryTaskManager) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	return task, nil
}

func (m *InMemoryTaskManager) ListSubTasks(parentID string) ([]*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var subtasks []*Task
	for _, task := range m.tasks {
		if task.ParentID == parentID {
			subtasks = append(subtasks, task)
		}
	}
	return subtasks, nil
}
