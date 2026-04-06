package common

import (
	"time"
)

// Status represents the execution status
type Status string

const (
	StatusPending    Status = "pending"
	StatusRunning    Status = "running"
	StatusPaused     Status = "paused"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCanceled   Status = "canceled"
)

// SecurityLevel represents the security level of a tool
type SecurityLevel int

const (
	// LevelSafe - Pure query operations with no side effects
	LevelSafe SecurityLevel = iota
	// LevelSensitive - Sensitive or bounded write operations
	LevelSensitive
	// LevelHighRisk - High-risk, unpredictable destructive operations
	LevelHighRisk
)

// String returns the string representation of SecurityLevel
func (s SecurityLevel) String() string {
	switch s {
	case LevelSafe:
		return "safe"
	case LevelSensitive:
		return "sensitive"
	case LevelHighRisk:
		return "high_risk"
	default:
		return "unknown"
	}
}

// ParseSecurityLevel parses a string to SecurityLevel
func ParseSecurityLevel(s string) SecurityLevel {
	switch s {
	case "safe":
		return LevelSafe
	case "sensitive":
		return LevelSensitive
	case "high_risk":
		return LevelHighRisk
	default:
		return LevelSafe
	}
}

// Intent represents the type of user intent
type Intent string

const (
	// IntentChat - Casual conversation with no specific task
	IntentChat Intent = "chat"
	// IntentTask - Task that needs execution
	IntentTask Intent = "task"
	// IntentClarification - Response to a clarification question
	IntentClarification Intent = "clarification"
	// IntentFollowUp - Follow-up question to previous result
	IntentFollowUp Intent = "follow_up"
	// IntentFeedback - Feedback on execution result
	IntentFeedback Intent = "feedback"
)

// ActionType represents the type of action
type ActionType string

const (
	// ActionTypeToolCall - Tool invocation
	ActionTypeToolCall ActionType = "tool_call"
	// ActionTypeSkillInvoke - Skill invocation
	ActionTypeSkillInvoke ActionType = "skill_invoke"
	// ActionTypeSubAgentDelegate - Sub-agent delegation
	ActionTypeSubAgentDelegate ActionType = "sub_agent_delegate"
	// ActionTypeNoAction - No action needed
	ActionTypeNoAction ActionType = "no_action"
)

// QuestionType represents the type of pending question
type QuestionType string

const (
	// QuestionTypeAuthorization - Authorization request
	QuestionTypeAuthorization QuestionType = "authorization"
	// QuestionTypeConfirmation - Confirmation request
	QuestionTypeConfirmation QuestionType = "confirmation"
	// QuestionTypeClarification - Clarification request
	QuestionTypeClarification QuestionType = "clarification"
	// QuestionTypeCustomInput - Custom input request
	QuestionTypeCustomInput QuestionType = "custom_input"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusPaused   SessionStatus = "paused"
	SessionStatusEnded    SessionStatus = "ended"
	SessionStatusArchived SessionStatus = "archived"
)

// PlanStatus represents the status of a plan
type PlanStatus string

const (
	PlanStatusPending   PlanStatus = "pending"
	PlanStatusRunning   PlanStatus = "running"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
	PlanStatusRevised   PlanStatus = "revised"
)

// StepStatus represents the status of a plan step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// FrozenStatus represents the status of a frozen session
type FrozenStatus string

const (
	FrozenStatusFrozen   FrozenStatus = "frozen"
	FrozenStatusResumed  FrozenStatus = "resumed"
	FrozenStatusCanceled FrozenStatus = "canceled"
	FrozenStatusExpired  FrozenStatus = "expired"
)

// GeneratedStatus represents the status of a generated resource
type GeneratedStatus string

const (
	GeneratedStatusDraft    GeneratedStatus = "draft"
	GeneratedStatusReview   GeneratedStatus = "review"
	GeneratedStatusApproved GeneratedStatus = "approved"
	GeneratedStatusRejected GeneratedStatus = "rejected"
	GeneratedStatusActive   GeneratedStatus = "active"
)

// MemoryItemType represents the type of a memory item
type MemoryItemType string

const (
	MemoryItemTypeFact        MemoryItemType = "fact"
	MemoryItemTypePreference  MemoryItemType = "preference"
	MemoryItemTypePattern     MemoryItemType = "pattern"
	MemoryItemTypeConstraint  MemoryItemType = "constraint"
	MemoryItemTypeCorrection  MemoryItemType = "correction"
	MemoryItemTypeInstruction MemoryItemType = "instruction"
)

// MemorySource represents the source of a memory item
type MemorySource string

const (
	MemorySourceUser      MemorySource = "user"
	MemorySourceSystem    MemorySource = "system"
	MemorySourceInference MemorySource = "inference"
	MemorySourceEvolution MemorySource = "evolution"
)

// EmphasisLevel represents the emphasis level of a memory item
type EmphasisLevel int

const (
	EmphasisLevelNormal EmphasisLevel = iota
	EmphasisLevelImportant
	EmphasisLevelCritical
)

// ToolType represents the type of a tool
type ToolType string

const (
	ToolTypePython ToolType = "python"
	ToolTypeCLI    ToolType = "cli"
	ToolTypeBash   ToolType = "bash"
	ToolTypeGo     ToolType = "go"
)

// EvolutionTrigger represents when evolution is triggered
type EvolutionTrigger string

const (
	EvolutionTriggerOnSessionEnd EvolutionTrigger = "on_session_end"
	EvolutionTriggerOnSchedule   EvolutionTrigger = "on_schedule"
	EvolutionTriggerManual       EvolutionTrigger = "manual"
	EvolutionTriggerOnThreshold  EvolutionTrigger = "on_threshold"
)

// EvolutionStatus represents the status of evolution
type EvolutionStatus string

const (
	EvolutionStatusPending    EvolutionStatus = "pending"
	EvolutionStatusInProgress EvolutionStatus = "in_progress"
	EvolutionStatusCompleted  EvolutionStatus = "completed"
	EvolutionStatusFailed     EvolutionStatus = "failed"
	EvolutionStatusSkipped    EvolutionStatus = "skipped"
)

// ReviewPolicy represents the review policy for generated resources
type ReviewPolicy int

const (
	// ReviewAutomatic - Automatic review by rule engine
	ReviewAutomatic ReviewPolicy = iota
	// ReviewManual - Manual review by developer/admin
	ReviewManual
	// ReviewHybrid - Automatic check then manual confirmation
	ReviewHybrid
)

// InsightType represents the type of an insight
type InsightType string

const (
	InsightTypePatternMatch   InsightType = "pattern_match"
	InsightTypeKeyFinding     InsightType = "key_finding"
	InsightTypeAnomaly        InsightType = "anomaly"
	InsightTypeTrend          InsightType = "trend"
	InsightTypeRecommendation InsightType = "recommendation"
)

// TerminationReason represents the reason for termination
type TerminationReason string

const (
	TerminationReasonGoalAchieved   TerminationReason = "goal_achieved"
	TerminationReasonMaxSteps       TerminationReason = "max_steps_reached"
	TerminationReasonStuckDetected  TerminationReason = "stuck_detected"
	TerminationReasonUserInterrupted TerminationReason = "user_interrupted"
	TerminationReasonErrorOccurred  TerminationReason = "error_occurred"
	TerminationReasonMaxRetries     TerminationReason = "max_retries_exceeded"
)

// QuestionStatus represents the status of a pending question
type QuestionStatus string

const (
	QuestionStatusPending   QuestionStatus = "pending"
	QuestionStatusAnswered  QuestionStatus = "answered"
	QuestionStatusExpired   QuestionStatus = "expired"
	QuestionStatusCancelled QuestionStatus = "cancelled"
)

// HeuristicType represents the type of heuristic lesson
type HeuristicType string

const (
	HeuristicTypeActionSuggestion    HeuristicType = "action_suggestion"
	HeuristicTypeParameterAdjustment HeuristicType = "parameter_adjustment"
	HeuristicTypeStrategyChange      HeuristicType = "strategy_change"
	HeuristicTypeToolSelection       HeuristicType = "tool_selection"
)

// EngineStatus represents the engine state status
type EngineStatus string

const (
	EngineStatusIdle       EngineStatus = "idle"
	EngineStatusPlanning   EngineStatus = "planning"
	EngineStatusRunning    EngineStatus = "running"
	EngineStatusReplanning EngineStatus = "replanning"
	EngineStatusReflecting EngineStatus = "reflecting"
	EngineStatusRetrying   EngineStatus = "retrying"
	EngineStatusSuspended  EngineStatus = "suspended"
	EngineStatusCompleted  EngineStatus = "completed"
	EngineStatusFailed     EngineStatus = "failed"
	EngineStatusStopped    EngineStatus = "stopped"
)

// TokenUsage tracks token usage
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Duration tracks time duration
type Duration struct {
	Start time.Time     `json:"start"`
	End   time.Time     `json:"end"`
	Total time.Duration `json:"total"`
}

// NewDuration creates a new Duration starting now
func NewDuration() *Duration {
	return &Duration{
		Start: time.Now(),
	}
}

// Stop stops the duration timer
func (d *Duration) Stop() {
	d.End = time.Now()
	d.Total = d.End.Sub(d.Start)
}

// Pair represents a key-value pair
type Pair struct {
	Key   string
	Value any
}

// Pairs is a collection of key-value pairs
type Pairs []Pair

// ToMap converts Pairs to a map
func (p Pairs) ToMap() map[string]any {
	m := make(map[string]any, len(p))
	for _, pair := range p {
		m[pair.Key] = pair.Value
	}
	return m
}
