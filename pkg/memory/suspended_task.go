package memory

import (
	"context"
	"fmt"
	"time"

	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// SuspendedTaskService provides suspended task management for UI layer
type SuspendedTaskService interface {
	GetSuspendedTasks(ctx context.Context, opts ...SuspendedTaskOption) (*SuspendedTaskList, error)
	GetSuspendedTask(ctx context.Context, sessionName string) (*SuspendedTask, error)
	ResumeTask(ctx context.Context, sessionName string, answer string) error
	CancelTask(ctx context.Context, sessionName string) error
	GetTaskCount(ctx context.Context, opts ...SuspendedTaskOption) (*TaskCount, error)
}

// SuspendedTaskOption is a function that configures suspended task options
type SuspendedTaskOption func(*SuspendedTaskOptions)

// SuspendedTaskOptions contains options for suspended task queries
type SuspendedTaskOptions struct {
	UserName   string
	AgentName  string
	Status     goreactcommon.FrozenStatus
	Priority   TaskPriority
	Page       int
	PageSize   int
	SortBy     string
	SortDesc   bool
	DateFrom   time.Time
	DateTo     time.Time
}

// WithUserName filters by user name
func WithUserName(userName string) SuspendedTaskOption {
	return func(o *SuspendedTaskOptions) {
		o.UserName = userName
	}
}

// WithAgentName filters by agent name
func WithAgentName(agentName string) SuspendedTaskOption {
	return func(o *SuspendedTaskOptions) {
		o.AgentName = agentName
	}
}

// WithStatus filters by frozen status
func WithStatus(status goreactcommon.FrozenStatus) SuspendedTaskOption {
	return func(o *SuspendedTaskOptions) {
		o.Status = status
	}
}

// WithPriority filters by task priority
func WithPriority(priority TaskPriority) SuspendedTaskOption {
	return func(o *SuspendedTaskOptions) {
		o.Priority = priority
	}
}

// WithPage sets pagination
func WithPage(page, pageSize int) SuspendedTaskOption {
	return func(o *SuspendedTaskOptions) {
		o.Page = page
		o.PageSize = pageSize
	}
}

// WithSort sets sorting
func WithSort(sortBy string, sortDesc bool) SuspendedTaskOption {
	return func(o *SuspendedTaskOptions) {
		o.SortBy = sortBy
		o.SortDesc = sortDesc
	}
}

// WithDateRange filters by date range
func WithDateRange(from, to time.Time) SuspendedTaskOption {
	return func(o *SuspendedTaskOptions) {
		o.DateFrom = from
		o.DateTo = to
	}
}

// TaskPriority represents the priority of a task
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityNormal TaskPriority = "normal"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityUrgent TaskPriority = "urgent"
)

// SuspendedTaskList represents a list of suspended tasks
type SuspendedTaskList struct {
	Tasks      []*SuspendedTask `json:"tasks" yaml:"tasks"`
	Total      int              `json:"total" yaml:"total"`
	Page       int              `json:"page" yaml:"page"`
	PageSize   int              `json:"page_size" yaml:"page_size"`
	TotalPages int              `json:"total_pages" yaml:"total_pages"`
}

// SuspendedTask represents a suspended task for UI display
type SuspendedTask struct {
	TaskID         string           `json:"task_id" yaml:"task_id"`
	SessionName    string           `json:"session_name" yaml:"session_name"`
	UserName       string           `json:"user_name" yaml:"user_name"`
	AgentName      string           `json:"agent_name" yaml:"agent_name"`
	Status         goreactcommon.FrozenStatus `json:"status" yaml:"status"`
	Priority       TaskPriority     `json:"priority" yaml:"priority"`
	SuspendReason  SuspendReason    `json:"suspend_reason" yaml:"suspend_reason"`
	SuspendTime    time.Time        `json:"suspend_time" yaml:"suspend_time"`
	ExpiresAt      time.Time        `json:"expires_at" yaml:"expires_at"`
	Question       *TaskQuestion    `json:"question" yaml:"question"`
	Context        *TaskContext     `json:"context" yaml:"context"`
	Actions        []TaskAction     `json:"actions" yaml:"actions"`
}

// SuspendReason represents why a task was suspended
type SuspendReason string

const (
	SuspendReasonUserAuthorization SuspendReason = "user_authorization"
	SuspendReasonUserConfirmation  SuspendReason = "user_confirmation"
	SuspendReasonUserClarification SuspendReason = "user_clarification"
	SuspendReasonUserCustomInput   SuspendReason = "user_custom_input"
	SuspendReasonToolAuthorization SuspendReason = "tool_authorization"
	SuspendReasonSystemWait        SuspendReason = "system_wait"
)

// TaskQuestion represents the question that needs to be answered
type TaskQuestion struct {
	QuestionID    string                      `json:"question_id" yaml:"question_id"`
	Type          goreactcommon.QuestionType  `json:"type" yaml:"type"`
	Content       string                      `json:"content" yaml:"content"`
	Options       []string                    `json:"options" yaml:"options"`
	DefaultAnswer string                      `json:"default_answer" yaml:"default_answer"`
	IsRequired    bool                        `json:"is_required" yaml:"is_required"`
}

// TaskContext provides context for the suspended task
type TaskContext struct {
	Summary           string                   `json:"summary" yaml:"summary"`
	UserIntent        string                   `json:"user_intent" yaml:"user_intent"`
	CurrentStep       string                   `json:"current_step" yaml:"current_step"`
	Progress          float64                  `json:"progress" yaml:"progress"`
	RelatedFiles      []string                 `json:"related_files" yaml:"related_files"`
	ConversationSnippet []*goreactcore.MessageNode `json:"conversation_snippet" yaml:"conversation_snippet"`
}

// TaskAction represents an available action for the task
type TaskAction struct {
	Type        ActionType `json:"type" yaml:"type"`
	Label       string     `json:"label" yaml:"label"`
	Description string     `json:"description" yaml:"description"`
	IsDefault   bool       `json:"is_default" yaml:"is_default"`
}

// ActionType represents the type of task action
type ActionType string

const (
	ActionTypeAnswer    ActionType = "answer"
	ActionTypeCancel    ActionType = "cancel"
	ActionTypeRetry     ActionType = "retry"
	ActionTypeSkip      ActionType = "skip"
	ActionTypeModify    ActionType = "modify"
	ActionTypeAuthorize ActionType = "authorize"
)

// TaskCount represents task count statistics
type TaskCount struct {
	Total     int `json:"total" yaml:"total"`
	Pending   int `json:"pending" yaml:"pending"`
	Expired   int `json:"expired" yaml:"expired"`
	ByUser    map[string]int `json:"by_user" yaml:"by_user"`
	ByAgent   map[string]int `json:"by_agent" yaml:"by_agent"`
	ByPriority map[TaskPriority]int `json:"by_priority" yaml:"by_priority"`
}

// suspendedTaskService implements SuspendedTaskService
type suspendedTaskService struct {
	memory *Memory
}

// NewSuspendedTaskService creates a new SuspendedTaskService
func NewSuspendedTaskService(memory *Memory) SuspendedTaskService {
	return &suspendedTaskService{memory: memory}
}

// GetSuspendedTasks gets a list of suspended tasks
func (s *suspendedTaskService) GetSuspendedTasks(ctx context.Context, opts ...SuspendedTaskOption) (*SuspendedTaskList, error) {
	options := &SuspendedTaskOptions{
		Page:     1,
		PageSize: 10,
		SortBy:   "suspend_time",
		SortDesc: true,
	}
	for _, opt := range opts {
		opt(options)
	}
	
	// Get frozen sessions from memory
	frozenSessions, err := s.memory.FrozenSessions().List(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert to suspended tasks
	tasks := make([]*SuspendedTask, 0, len(frozenSessions))
	for _, frozen := range frozenSessions {
		task, err := s.convertToSuspendedTask(ctx, frozen)
		if err != nil {
			continue
		}
		
		// Apply filters
		if options.UserName != "" && task.UserName != options.UserName {
			continue
		}
		if options.AgentName != "" && task.AgentName != options.AgentName {
			continue
		}
		if options.Status != "" && task.Status != options.Status {
			continue
		}
		
		tasks = append(tasks, task)
	}
	
	// Calculate pagination
	total := len(tasks)
	totalPages := (total + options.PageSize - 1) / options.PageSize
	
	// Apply pagination
	start := (options.Page - 1) * options.PageSize
	end := start + options.PageSize
	if start >= total {
		start = 0
		end = 0
	} else if end > total {
		end = total
	}
	
	return &SuspendedTaskList{
		Tasks:      tasks[start:end],
		Total:      total,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetSuspendedTask gets a single suspended task
func (s *suspendedTaskService) GetSuspendedTask(ctx context.Context, sessionName string) (*SuspendedTask, error) {
	frozen, err := s.memory.FrozenSessions().Get(ctx, sessionName)
	if err != nil {
		return nil, err
	}
	
	return s.convertToSuspendedTask(ctx, frozen)
}

// ResumeTask resumes a suspended task
func (s *suspendedTaskService) ResumeTask(ctx context.Context, sessionName string, answer string) error {
	return s.memory.FrozenSessions().Resume(ctx, sessionName, answer)
}

// CancelTask cancels a suspended task
func (s *suspendedTaskService) CancelTask(ctx context.Context, sessionName string) error {
	return s.memory.FrozenSessions().Cancel(ctx, sessionName)
}

// GetTaskCount gets task count statistics
func (s *suspendedTaskService) GetTaskCount(ctx context.Context, opts ...SuspendedTaskOption) (*TaskCount, error) {
	options := &SuspendedTaskOptions{}
	for _, opt := range opts {
		opt(options)
	}
	
	frozenSessions, err := s.memory.FrozenSessions().List(ctx)
	if err != nil {
		return nil, err
	}
	
	count := &TaskCount{
		Total:      len(frozenSessions),
		ByUser:     make(map[string]int),
		ByAgent:    make(map[string]int),
		ByPriority: make(map[TaskPriority]int),
	}
	
	for _, frozen := range frozenSessions {
		// Count by status
		switch frozen.Status {
		case goreactcommon.FrozenStatusFrozen:
			count.Pending++
		case goreactcommon.FrozenStatusExpired:
			count.Expired++
		}
		
		// Count by user
		if frozen.UserName != "" {
			count.ByUser[frozen.UserName]++
		}
		
		// Count by agent
		if frozen.AgentName != "" {
			count.ByAgent[frozen.AgentName]++
		}
		
		// Default priority
		count.ByPriority[TaskPriorityNormal]++
	}
	
	return count, nil
}

// convertToSuspendedTask converts a FrozenSessionNode to SuspendedTask
func (s *suspendedTaskService) convertToSuspendedTask(ctx context.Context, frozen *goreactcore.FrozenSessionNode) (*SuspendedTask, error) {
	task := &SuspendedTask{
		TaskID:        frozen.Name,
		SessionName:   frozen.SessionName,
		UserName:      frozen.UserName,
		AgentName:     frozen.AgentName,
		Status:        frozen.Status,
		Priority:      TaskPriorityNormal,
		SuspendReason: SuspendReason(frozen.SuspendReason),
		SuspendTime:   frozen.CreatedAt,
		ExpiresAt:     frozen.ExpiresAt,
	}
	
	// Build question
	if frozen.QuestionID != "" {
		task.Question = &TaskQuestion{
			QuestionID: frozen.QuestionID,
			Content:    frozen.SuspendReason,
			Type:       goreactcommon.QuestionTypeConfirmation,
		}
	}
	
	// Build context
	task.Context = &TaskContext{
		Summary:    frozen.SuspendReason,
		Progress:   0.5, // Default progress
	}
	
	// Build default actions
	task.Actions = []TaskAction{
		{
			Type:        ActionTypeAnswer,
			Label:       "Provide Answer",
			Description: "Answer the question to continue",
			IsDefault:   true,
		},
		{
			Type:        ActionTypeCancel,
			Label:       "Cancel Task",
			Description: "Cancel this suspended task",
			IsDefault:   false,
		},
	}
	
	return task, nil
}

// GetExpiredTasks returns tasks that have expired
func (s *suspendedTaskService) GetExpiredTasks(ctx context.Context) ([]*SuspendedTask, error) {
	frozenSessions, err := s.memory.FrozenSessions().List(ctx)
	if err != nil {
		return nil, err
	}
	
	tasks := make([]*SuspendedTask, 0)
	now := time.Now()
	
	for _, frozen := range frozenSessions {
		if frozen.ExpiresAt.Before(now) {
			task, err := s.convertToSuspendedTask(ctx, frozen)
			if err != nil {
				continue
			}
			tasks = append(tasks, task)
		}
	}
	
	return tasks, nil
}

// CleanupExpiredTasks removes expired tasks
func (s *suspendedTaskService) CleanupExpiredTasks(ctx context.Context) (int, error) {
	frozenSessions, err := s.memory.FrozenSessions().List(ctx)
	if err != nil {
		return 0, err
	}
	
	count := 0
	now := time.Now()
	
	for _, frozen := range frozenSessions {
		if frozen.ExpiresAt.Before(now) {
			if err := s.memory.FrozenSessions().Cancel(ctx, frozen.SessionName); err == nil {
				count++
			}
		}
	}
	
	return count, nil
}

// FormatSuspendReason formats a suspend reason for display
func FormatSuspendReason(reason SuspendReason) string {
	switch reason {
	case SuspendReasonUserAuthorization:
		return "Waiting for user authorization"
	case SuspendReasonUserConfirmation:
		return "Waiting for user confirmation"
	case SuspendReasonUserClarification:
		return "Waiting for user clarification"
	case SuspendReasonUserCustomInput:
		return "Waiting for user input"
	case SuspendReasonToolAuthorization:
		return "Waiting for tool authorization"
	case SuspendReasonSystemWait:
		return "System waiting"
	default:
		return "Unknown reason"
	}
}

// FormatTaskPriority formats a task priority for display
func FormatTaskPriority(priority TaskPriority) string {
	switch priority {
	case TaskPriorityLow:
		return "Low"
	case TaskPriorityNormal:
		return "Normal"
	case TaskPriorityHigh:
		return "High"
	case TaskPriorityUrgent:
		return "Urgent"
	default:
		return "Normal"
	}
}

// GetTimeRemaining returns the time remaining before a task expires
func GetTimeRemaining(task *SuspendedTask) time.Duration {
	if task == nil {
		return 0
	}
	
	remaining := time.Until(task.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsExpired checks if a task is expired
func IsExpired(task *SuspendedTask) bool {
	if task == nil {
		return true
	}
	return task.ExpiresAt.Before(time.Now())
}

// CanResume checks if a task can be resumed
func CanResume(task *SuspendedTask) bool {
	if task == nil {
		return false
	}
	return task.Status == goreactcommon.FrozenStatusFrozen && !IsExpired(task)
}

// FilterTasksByUser filters tasks by user name
func FilterTasksByUser(tasks []*SuspendedTask, userName string) []*SuspendedTask {
	if userName == "" {
		return tasks
	}
	
	filtered := make([]*SuspendedTask, 0)
	for _, task := range tasks {
		if task.UserName == userName {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// FilterTasksByAgent filters tasks by agent name
func FilterTasksByAgent(tasks []*SuspendedTask, agentName string) []*SuspendedTask {
	if agentName == "" {
		return tasks
	}
	
	filtered := make([]*SuspendedTask, 0)
	for _, task := range tasks {
		if task.AgentName == agentName {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// SortTasksByTime sorts tasks by suspend time
func SortTasksByTime(tasks []*SuspendedTask, desc bool) {
	if desc {
		// Sort descending (newest first)
		for i := 0; i < len(tasks)-1; i++ {
			for j := i + 1; j < len(tasks); j++ {
				if tasks[i].SuspendTime.Before(tasks[j].SuspendTime) {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	} else {
		// Sort ascending (oldest first)
		for i := 0; i < len(tasks)-1; i++ {
			for j := i + 1; j < len(tasks); j++ {
				if tasks[i].SuspendTime.After(tasks[j].SuspendTime) {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

// SortTasksByPriority sorts tasks by priority
func SortTasksByPriority(tasks []*SuspendedTask) {
	priorityOrder := map[TaskPriority]int{
		TaskPriorityUrgent: 0,
		TaskPriorityHigh:   1,
		TaskPriorityNormal: 2,
		TaskPriorityLow:    3,
	}
	
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			if priorityOrder[tasks[i].Priority] > priorityOrder[tasks[j].Priority] {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

// GroupTasksByStatus groups tasks by status
func GroupTasksByStatus(tasks []*SuspendedTask) map[goreactcommon.FrozenStatus][]*SuspendedTask {
	groups := make(map[goreactcommon.FrozenStatus][]*SuspendedTask)
	
	for _, task := range tasks {
		groups[task.Status] = append(groups[task.Status], task)
	}
	
	return groups
}

// GroupTasksByUser groups tasks by user
func GroupTasksByUser(tasks []*SuspendedTask) map[string][]*SuspendedTask {
	groups := make(map[string][]*SuspendedTask)
	
	for _, task := range tasks {
		groups[task.UserName] = append(groups[task.UserName], task)
	}
	
	return groups
}

// GenerateTaskSummary generates a summary of suspended tasks
func GenerateTaskSummary(tasks []*SuspendedTask) string {
	if len(tasks) == 0 {
		return "No suspended tasks"
	}
	
	total := len(tasks)
	expired := 0
	urgent := 0
	
	for _, task := range tasks {
		if IsExpired(task) {
			expired++
		}
		if task.Priority == TaskPriorityUrgent {
			urgent++
		}
	}
	
	return fmt.Sprintf("Total: %d, Expired: %d, Urgent: %d", total, expired, urgent)
}
