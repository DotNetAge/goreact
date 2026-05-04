package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/core"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskStopped   TaskStatus = "stopped"
)

type TaskType string

const (
	TaskTypeAgent TaskType = "agent"
	TaskTypeShell TaskType = "shell"
)

type Task struct {
	ID          string     `json:"id"`
	Type        TaskType   `json:"type"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	AgentName   string     `json:"agent_name,omitempty"`
	Prompt      string     `json:"prompt,omitempty"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	OutputPath  string     `json:"output_path,omitempty"`
}

type Team struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Leader      string    `json:"leader"`
	Members     []string  `json:"members"`
	TaskIDs     []string  `json:"task_ids"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
}

const (
	taskKeyPrefix = "tasks:"
	teamKeyPrefix = "teams:"
)

func taskKey(sessionID, taskID string) string {
	return fmt.Sprintf("%s%s:%s", taskKeyPrefix, sessionID, taskID)
}

func teamKey(sessionID, teamName string) string {
	return fmt.Sprintf("%s%s:%s", teamKeyPrefix, sessionID, teamName)
}

func taskListKey(sessionID string) string {
	return fmt.Sprintf("%s%s:__list__", taskKeyPrefix, sessionID)
}

func teamListKey(sessionID string) string {
	return fmt.Sprintf("%s%s:__list__", teamKeyPrefix, sessionID)
}

func CreateTask(ctx context.Context, sessionID string, task *Task) error {
	kv := getKVStore(ctx)
	if kv == nil {
		return fmt.Errorf("KVStore not available")
	}

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Status == "" {
		task.Status = TaskPending
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	if err := kv.Set(ctx, sessionID, taskKey(sessionID, task.ID), data, 0); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	list, _ := getTaskList(ctx, sessionID)
	list = append(list, task.ID)
	listData, _ := json.Marshal(list)
	return kv.Set(ctx, sessionID, taskListKey(sessionID), listData, 0)
}

func GetTask(ctx context.Context, sessionID, taskID string) (*Task, error) {
	kv := getKVStore(ctx)
	if kv == nil {
		return nil, fmt.Errorf("KVStore not available")
	}

	data, err := kv.Get(ctx, sessionID, taskKey(sessionID, taskID))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}
	return &task, nil
}

func UpdateTask(ctx context.Context, sessionID string, task *Task) error {
	kv := getKVStore(ctx)
	if kv == nil {
		return fmt.Errorf("KVStore not available")
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return kv.Set(ctx, sessionID, taskKey(sessionID, task.ID), data, 0)
}

func ListTasks(ctx context.Context, sessionID string) ([]string, error) {
	return getTaskList(ctx, sessionID)
}

func CreateTeam(ctx context.Context, sessionID string, team *Team) error {
	kv := getKVStore(ctx)
	if kv == nil {
		return fmt.Errorf("KVStore not available")
	}

	team.CreatedAt = time.Now()
	team.Status = "active"

	data, err := json.Marshal(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %w", err)
	}

	if err := kv.Set(ctx, sessionID, teamKey(sessionID, team.Name), data, 0); err != nil {
		return fmt.Errorf("failed to save team: %w", err)
	}

	list, _ := getTeamList(ctx, sessionID)
	list = append(list, team.Name)
	listData, _ := json.Marshal(list)
	return kv.Set(ctx, sessionID, teamListKey(sessionID), listData, 0)
}

func GetTeam(ctx context.Context, sessionID, teamName string) (*Team, error) {
	kv := getKVStore(ctx)
	if kv == nil {
		return nil, fmt.Errorf("KVStore not available")
	}

	data, err := kv.Get(ctx, sessionID, teamKey(sessionID, teamName))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var team Team
	if err := json.Unmarshal(data, &team); err != nil {
		return nil, fmt.Errorf("failed to unmarshal team: %w", err)
	}
	return &team, nil
}

func ListTeams(ctx context.Context, sessionID string) ([]string, error) {
	return getTeamList(ctx, sessionID)
}

func DeleteTeam(ctx context.Context, sessionID, teamName string) error {
	kv := getKVStore(ctx)
	if kv == nil {
		return fmt.Errorf("KVStore not available")
	}

	if err := kv.Delete(ctx, sessionID, teamKey(sessionID, teamName)); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	list, _ := getTeamList(ctx, sessionID)
	newList := make([]string, 0, len(list))
	for _, name := range list {
		if name != teamName {
			newList = append(newList, name)
		}
	}
	listData, _ := json.Marshal(newList)
	return kv.Set(ctx, sessionID, teamListKey(sessionID), listData, 0)
}

func getTaskList(ctx context.Context, sessionID string) ([]string, error) {
	kv := getKVStore(ctx)
	if kv == nil {
		return nil, fmt.Errorf("KVStore not available")
	}

	data, err := kv.Get(ctx, sessionID, taskListKey(sessionID))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task list: %w", err)
	}
	return list, nil
}

func getTeamList(ctx context.Context, sessionID string) ([]string, error) {
	kv := getKVStore(ctx)
	if kv == nil {
		return nil, fmt.Errorf("KVStore not available")
	}

	data, err := kv.Get(ctx, sessionID, teamListKey(sessionID))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal team list: %w", err)
	}
	return list, nil
}

func getKVStore(ctx context.Context) core.KVStore {
	tc := core.GetToolContext(ctx)
	if tc == nil {
		return nil
	}
	return tc.KVStore
}
