package orchestration

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// TaskStore manages task lifecycle for all orchestrated sub-agent work.
// It wraps core.TaskManager with Orchestrator-specific semantics:
//   - Pending → InProgress → Completed/Failed/Cancelled state machine
//   - Active task tracking (for concurrency limiting and cleanup)
//   - Result channel management
//
// Implementations may be in-memory (default) or persistent (SQLite/BadgerDB).
type TaskStore interface {
	// CreateTask creates a new task record in Pending state.
	CreateTask(parentID string, description string, input string) (*core.Task, error)

	// UpdateTaskStatus transitions a task to a new status.
	UpdateTaskStatus(id string, status core.TaskStatus, output string, errMsg string) error

	// GetTask retrieves a task by ID.
	GetTask(id string) (*core.Task, error)

	// ListSubTasks returns all tasks with the given parent ID.
	ListSubTasks(parentID string) ([]*core.Task, error)

	// ListAllTasks returns all tasks.
	ListAllTasks() ([]*core.Task, error)

	// CancelTask marks a task as cancelled. Returns error if already terminal.
	CancelTask(id string) error

	// SetResultCh associates an async result channel with a task.
	// The Orchestrator reads from this channel when the sub-agent finishes.
	SetResultCh(taskID string, ch <-chan any)

	// GetResultCh retrieves the result channel for a task.
	GetResultCh(taskID string) (<-chan any, bool)

	// RemoveResultCh cleans up the result channel after consumption.
	RemoveResultCh(taskID string)

	// ActiveTasks returns the number of currently running (InProgress) tasks.
	ActiveTasks() int

	// Close shuts down the store, releasing all resources.
	Close(ctx context.Context) error
}

// InMemoryTaskStore is the default TaskStore implementation using in-memory maps.
// It combines core.InMemoryTaskManager with active-task tracking and async channels.
type InMemoryTaskStore struct {
	// Embed (or delegate to) the existing core.TaskManager for CRUD
	tasks     map[string]*core.Task
	resultChs map[string]<-chan any
	mu        taskStoreMu // Protects both maps + active count
	active    int
	nextID     int64
}

type taskStoreMu struct {
	mu *interface{} // placeholder - use sync.RWMutex in implementation
}

// NewInMemoryTaskStore creates a fresh in-memory task store.
func NewInMemoryTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		tasks:     make(map[string]*core.Task),
		resultChs: make(map[string]<-chan any),
		mu:        taskStoreMu{mu: new(interface{})},
	}
}

// ActiveTasks returns the number of currently running tasks.
func (s *InMemoryTaskStore) ActiveTasks() int {
	return s.active
}

func (s *InMemoryTaskStore) nextTaskID() string {
	s.nextID++
	return fmt.Sprintf("task-%d", s.nextID)
}

// CreateTask creates a new task record in Pending state.
func (s *InMemoryTaskStore) CreateTask(parentID string, description string, input string) (*core.Task, error) {
	id := s.nextTaskID()
	task := &core.Task{
		ID:          id,
		ParentID:    parentID,
		Description: description,
		Input:       input,
		Status:      core.TaskStatusPending,
	}
	s.tasks[id] = task
	return task, nil
}

// UpdateTaskStatus transitions a task to a new status.
func (s *InMemoryTaskStore) UpdateTaskStatus(id string, status core.TaskStatus, output string, errMsg string) error {
	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	wasInProgress := task.Status == core.TaskStatusInProgress
	task.Status = status
	if output != "" {
		task.Output = output
	}
	if errMsg != "" {
		task.Error = errMsg
	}
	if status == core.TaskStatusInProgress && !wasInProgress {
		s.active++
	}
	if (status == core.TaskStatusCompleted || status == core.TaskStatusFailed || status == core.TaskStatusCancelled) && wasInProgress {
		s.active--
	}
	return nil
}

// GetTask retrieves a task by ID.
func (s *InMemoryTaskStore) GetTask(id string) (*core.Task, error) {
	task, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task %q not found", id)
	}
	return task, nil
}

// ListSubTasks returns all tasks with the given parent ID.
func (s *InMemoryTaskStore) ListSubTasks(parentID string) ([]*core.Task, error) {
	var result []*core.Task
	for _, t := range s.tasks {
		if t.ParentID == parentID {
			result = append(result, t)
		}
	}
	return result, nil
}

// ListAllTasks returns all tasks.
func (s *InMemoryTaskStore) ListAllTasks() ([]*core.Task, error) {
	result := make([]*core.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, t)
	}
	return result, nil
}

// CancelTask marks a task as cancelled.
func (s *InMemoryTaskStore) CancelTask(id string) error {
	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	if isTerminal(task.Status) {
		return fmt.Errorf("task %q already in terminal state %v", id, task.Status)
	}
	if task.Status == core.TaskStatusInProgress {
		s.active--
	}
	task.Status = core.TaskStatusCancelled
	return nil
}

// SetResultCh associates an async result channel with a task.
func (s *InMemoryTaskStore) SetResultCh(taskID string, ch <-chan any) {
	s.resultChs[taskID] = ch
}

// GetResultCh retrieves the result channel for a task.
func (s *InMemoryTaskStore) GetResultCh(taskID string) (<-chan any, bool) {
	ch, ok := s.resultChs[taskID]
	return ch, ok
}

// RemoveResultCh cleans up the result channel after consumption.
func (s *InMemoryTaskStore) RemoveResultCh(taskID string) {
	delete(s.resultChs, taskID)
}

// Close shuts down the store, releasing all resources.
func (s *InMemoryTaskStore) Close(ctx context.Context) error {
	s.tasks = nil
	s.resultChs = nil
	return nil
}

func isTerminal(s core.TaskStatus) bool {
	return s == core.TaskStatusCompleted || s == core.TaskStatusFailed || s == core.TaskStatusCancelled
}
