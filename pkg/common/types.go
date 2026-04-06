// Package common provides shared types, constants, errors, and utility functions
// used throughout the goreact framework. This package serves as the foundation
// for all other packages by defining common data structures, status codes,
// security levels, and configuration types.
//
// The package is organized into several key areas:
//   - Status types: Execution status indicators (pending, running, completed, etc.)
//   - Security levels: Tool and action security classifications
//   - Intent types: User intent classification for request handling
//   - Action types: Types of actions the agent can perform
//   - Memory types: Memory item classifications and sources
//   - Session and plan status: State tracking for sessions and plans
//
// Example usage:
//
//	status := common.StatusRunning
//	level := common.LevelSensitive
//	intent := common.IntentTask
package common

import (
	"time"
)

// Status represents the execution status of an agent, session, or task.
// Status values are used to track the lifecycle of operations throughout
// the framework, from initial pending state through completion or failure.
type Status string

const (
	// StatusPending indicates the operation is waiting to start.
	StatusPending Status = "pending"
	// StatusRunning indicates the operation is currently executing.
	StatusRunning Status = "running"
	// StatusPaused indicates the operation is temporarily suspended,
	// typically waiting for user input or authorization.
	StatusPaused Status = "paused"
	// StatusCompleted indicates the operation finished successfully.
	StatusCompleted Status = "completed"
	// StatusFailed indicates the operation encountered an error and could not complete.
	StatusFailed Status = "failed"
	// StatusCanceled indicates the operation was explicitly cancelled by the user or system.
	StatusCanceled Status = "canceled"
)

// SecurityLevel represents the security classification of a tool or action.
// Security levels are used to determine authorization requirements and
// risk assessment for operations that may have side effects.
// Higher security levels require more stringent authorization checks.
type SecurityLevel int

const (
	// LevelSafe indicates pure query operations with no side effects.
	// These operations are read-only and do not modify any state.
	// Examples: reading files, querying databases, searching the web.
	LevelSafe SecurityLevel = iota
	// LevelSensitive indicates operations with bounded write effects.
	// These operations may modify state but with predictable, limited scope.
	// Examples: creating temporary files, updating user preferences, sending messages.
	LevelSensitive
	// LevelHighRisk indicates operations with unpredictable or destructive effects.
	// These operations may cause significant changes and require explicit authorization.
	// Examples: deleting files, executing arbitrary code, modifying system configuration.
	LevelHighRisk
)

// String returns the string representation of SecurityLevel.
// It converts the numeric security level to its corresponding
// string identifier: "safe", "sensitive", or "high_risk".
// Returns "unknown" for undefined security levels.
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

// ParseSecurityLevel parses a string to SecurityLevel.
// It converts string identifiers ("safe", "sensitive", "high_risk")
// to their corresponding SecurityLevel values.
// Returns LevelSafe for unrecognized strings as a safe default.
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

// ActionType represents the type of action the agent can perform.
// Action types determine how the agent interacts with tools, skills,
// and sub-agents to accomplish tasks.
type ActionType string

const (
	// ActionTypeToolCall indicates a direct tool invocation.
	// The agent calls a specific tool with provided parameters.
	ActionTypeToolCall ActionType = "tool_call"
	// ActionTypeSkillInvoke indicates a skill invocation.
	// The agent executes a pre-defined skill composed of multiple steps.
	ActionTypeSkillInvoke ActionType = "skill_invoke"
	// ActionTypeSubAgentDelegate indicates delegation to a sub-agent.
	// The agent delegates a subtask to a specialized agent.
	ActionTypeSubAgentDelegate ActionType = "sub_agent_delegate"
	// ActionTypeNoAction indicates no action is needed.
	// The agent has determined that no further action is required.
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

// SessionStatus represents the lifecycle status of a session.
// Sessions track the state of interactions between the user and agent.
type SessionStatus string

const (
	// SessionStatusActive indicates the session is currently active and accepting input.
	SessionStatusActive SessionStatus = "active"
	// SessionStatusPaused indicates the session is temporarily paused, waiting for user response.
	SessionStatusPaused SessionStatus = "paused"
	// SessionStatusEnded indicates the session has ended normally.
	SessionStatusEnded SessionStatus = "ended"
	// SessionStatusArchived indicates the session has been archived for long-term storage.
	SessionStatusArchived SessionStatus = "archived"
)

// PlanStatus represents the execution status of a plan.
// Plans organize tasks into structured steps for systematic execution.
type PlanStatus string

const (
	// PlanStatusPending indicates the plan is waiting to start execution.
	PlanStatusPending PlanStatus = "pending"
	// PlanStatusRunning indicates the plan is currently being executed.
	PlanStatusRunning PlanStatus = "running"
	// PlanStatusCompleted indicates all plan steps have been successfully completed.
	PlanStatusCompleted PlanStatus = "completed"
	// PlanStatusFailed indicates the plan execution encountered an unrecoverable error.
	PlanStatusFailed PlanStatus = "failed"
	// PlanStatusRevised indicates the plan has been modified during execution.
	PlanStatusRevised PlanStatus = "revised"
)

// StepStatus represents the execution status of a single plan step.
type StepStatus string

const (
	// StepStatusPending indicates the step is waiting to be executed.
	StepStatusPending StepStatus = "pending"
	// StepStatusRunning indicates the step is currently being executed.
	StepStatusRunning StepStatus = "running"
	// StepStatusCompleted indicates the step has finished successfully.
	StepStatusCompleted StepStatus = "completed"
	// StepStatusFailed indicates the step execution failed.
	StepStatusFailed StepStatus = "failed"
	// StepStatusSkipped indicates the step was skipped due to conditions or dependencies.
	StepStatusSkipped StepStatus = "skipped"
)

// FrozenStatus represents the status of a frozen (serialized) session.
// Frozen sessions allow pausing and resuming execution across process boundaries.
type FrozenStatus string

const (
	// FrozenStatusFrozen indicates the session is frozen and awaiting resumption.
	FrozenStatusFrozen FrozenStatus = "frozen"
	// FrozenStatusResumed indicates the frozen session has been successfully resumed.
	FrozenStatusResumed FrozenStatus = "resumed"
	// FrozenStatusCanceled indicates the frozen session was cancelled before resumption.
	FrozenStatusCanceled FrozenStatus = "canceled"
	// FrozenStatusExpired indicates the frozen session has passed its expiration time.
	FrozenStatusExpired FrozenStatus = "expired"
)

// GeneratedStatus represents the lifecycle status of a generated resource.
// Used for tracking the review and approval process of auto-generated skills and tools.
type GeneratedStatus string

const (
	// GeneratedStatusDraft indicates the resource is a draft awaiting review.
	GeneratedStatusDraft GeneratedStatus = "draft"
	// GeneratedStatusReview indicates the resource is currently under review.
	GeneratedStatusReview GeneratedStatus = "review"
	// GeneratedStatusApproved indicates the resource has been approved for use.
	GeneratedStatusApproved GeneratedStatus = "approved"
	// GeneratedStatusRejected indicates the resource was rejected during review.
	GeneratedStatusRejected GeneratedStatus = "rejected"
	// GeneratedStatusActive indicates the resource is active and available for use.
	GeneratedStatusActive GeneratedStatus = "active"
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
	MemoryItemTypeObservation MemoryItemType = "observation"
	MemoryItemTypeThought     MemoryItemType = "thought"
	MemoryItemTypeAction      MemoryItemType = "action"
)

// MemorySource represents the source of a memory item
type MemorySource string

const (
	MemorySourceUser      MemorySource = "user"
	MemorySourceSystem    MemorySource = "system"
	MemorySourceInference MemorySource = "inference"
	MemorySourceEvolution MemorySource = "evolution"
	MemorySourceAction    MemorySource = "action"
	MemorySourceTool      MemorySource = "tool"
)

// EmphasisLevel represents the importance level of a memory item.
// Higher emphasis levels indicate more critical information that should
// be prioritized in retrieval and decision-making.
type EmphasisLevel int

const (
	// EmphasisLevelNormal indicates standard importance.
	EmphasisLevelNormal EmphasisLevel = iota
	// EmphasisLevelImportant indicates elevated importance.
	EmphasisLevelImportant
	// EmphasisLevelCritical indicates critical importance requiring special attention.
	EmphasisLevelCritical
)

// ToolType represents the implementation type of a tool.
// Different tool types have different execution mechanisms and requirements.
type ToolType string

const (
	// ToolTypePython indicates a Python-based tool implementation.
	ToolTypePython ToolType = "python"
	// ToolTypeCLI indicates a command-line interface tool.
	ToolTypeCLI ToolType = "cli"
	// ToolTypeBash indicates a Bash script tool.
	ToolTypeBash ToolType = "bash"
	// ToolTypeGo indicates a Go-native tool implementation.
	ToolTypeGo ToolType = "go"
)

// EvolutionTrigger represents when evolution processes are initiated.
// Evolution allows the agent to learn and adapt by creating new skills and tools.
type EvolutionTrigger string

const (
	// EvolutionTriggerOnSessionEnd triggers evolution after a session completes.
	EvolutionTriggerOnSessionEnd EvolutionTrigger = "on_session_end"
	// EvolutionTriggerOnSchedule triggers evolution on a scheduled basis.
	EvolutionTriggerOnSchedule EvolutionTrigger = "on_schedule"
	// EvolutionTriggerManual triggers evolution only when explicitly requested.
	EvolutionTriggerManual EvolutionTrigger = "manual"
	// EvolutionTriggerOnThreshold triggers evolution when usage thresholds are met.
	EvolutionTriggerOnThreshold EvolutionTrigger = "on_threshold"
)

// EvolutionStatus represents the current status of an evolution process.
type EvolutionStatus string

const (
	// EvolutionStatusPending indicates evolution is waiting to start.
	EvolutionStatusPending EvolutionStatus = "pending"
	// EvolutionStatusInProgress indicates evolution is currently running.
	EvolutionStatusInProgress EvolutionStatus = "in_progress"
	// EvolutionStatusCompleted indicates evolution finished successfully.
	EvolutionStatusCompleted EvolutionStatus = "completed"
	// EvolutionStatusFailed indicates evolution encountered an error.
	EvolutionStatusFailed EvolutionStatus = "failed"
	// EvolutionStatusSkipped indicates evolution was skipped.
	EvolutionStatusSkipped EvolutionStatus = "skipped"
)

// ReviewPolicy represents the review policy for generated resources.
// Determines how auto-generated skills and tools are validated before use.
type ReviewPolicy int

const (
	// ReviewAutomatic indicates automatic review by rule engine without human intervention.
	ReviewAutomatic ReviewPolicy = iota
	// ReviewManual indicates manual review by developer or administrator.
	ReviewManual
	// ReviewHybrid indicates automatic check followed by manual confirmation.
	ReviewHybrid
)

// InsightType represents the classification of an insight extracted from observations.
type InsightType string

const (
	// InsightTypePatternMatch indicates a recognized pattern in data or behavior.
	InsightTypePatternMatch InsightType = "pattern_match"
	// InsightTypeKeyFinding indicates an important discovery from analysis.
	InsightTypeKeyFinding InsightType = "key_finding"
	// InsightTypeAnomaly indicates an unusual or unexpected observation.
	InsightTypeAnomaly InsightType = "anomaly"
	// InsightTypeTrend indicates a detected trend or direction in data.
	InsightTypeTrend InsightType = "trend"
	// InsightTypeRecommendation indicates a suggested action or improvement.
	InsightTypeRecommendation InsightType = "recommendation"
)

// TaskPriority represents the priority level of a task.
type TaskPriority string

const (
	// TaskPriorityLow indicates low priority, can be deferred.
	TaskPriorityLow TaskPriority = "low"
	// TaskPriorityNormal indicates standard priority.
	TaskPriorityNormal TaskPriority = "normal"
	// TaskPriorityHigh indicates high priority, should be processed promptly.
	TaskPriorityHigh TaskPriority = "high"
	// TaskPriorityUrgent indicates urgent priority, requires immediate attention.
	TaskPriorityUrgent TaskPriority = "urgent"
)

// SuspendReason represents why a task was suspended.
// Used to track the cause of execution pauses for proper resumption.
type SuspendReason string

const (
	// SuspendReasonUserAuthorization indicates waiting for user authorization.
	SuspendReasonUserAuthorization SuspendReason = "user_authorization"
	// SuspendReasonUserConfirmation indicates waiting for user confirmation.
	SuspendReasonUserConfirmation SuspendReason = "user_confirmation"
	// SuspendReasonUserClarification indicates waiting for user clarification.
	SuspendReasonUserClarification SuspendReason = "user_clarification"
	// SuspendReasonUserCustomInput indicates waiting for custom user input.
	SuspendReasonUserCustomInput SuspendReason = "user_custom_input"
	// SuspendReasonToolAuthorization indicates waiting for tool-level authorization.
	SuspendReasonToolAuthorization SuspendReason = "tool_authorization"
	// SuspendReasonSystemWait indicates waiting for system resources or events.
	SuspendReasonSystemWait SuspendReason = "system_wait"
)

// TerminationReason represents why an execution terminated.
// Used for analytics and improving agent behavior.
type TerminationReason string

const (
	// TerminationReasonGoalAchieved indicates the task goal was successfully achieved.
	TerminationReasonGoalAchieved TerminationReason = "goal_achieved"
	// TerminationReasonMaxSteps indicates the maximum step limit was reached.
	TerminationReasonMaxSteps TerminationReason = "max_steps_reached"
	// TerminationReasonStuckDetected indicates the agent was detected to be stuck in a loop.
	TerminationReasonStuckDetected TerminationReason = "stuck_detected"
	// TerminationReasonUserInterrupted indicates the user interrupted the execution.
	TerminationReasonUserInterrupted TerminationReason = "user_interrupted"
	// TerminationReasonErrorOccurred indicates an unrecoverable error occurred.
	TerminationReasonErrorOccurred TerminationReason = "error_occurred"
	// TerminationReasonMaxRetries indicates the maximum retry limit was exceeded.
	TerminationReasonMaxRetries TerminationReason = "max_retries_exceeded"
)

// QuestionStatus represents the status of a pending question.
type QuestionStatus string

const (
	// QuestionStatusPending indicates the question is awaiting a response.
	QuestionStatusPending QuestionStatus = "pending"
	// QuestionStatusAnswered indicates the question has been answered.
	QuestionStatusAnswered QuestionStatus = "answered"
	// QuestionStatusExpired indicates the question has expired without response.
	QuestionStatusExpired QuestionStatus = "expired"
	// QuestionStatusCancelled indicates the question was cancelled.
	QuestionStatusCancelled QuestionStatus = "cancelled"
)

// HeuristicType represents the type of heuristic lesson learned.
// Heuristics guide future agent behavior based on past experience.
type HeuristicType string

const (
	// HeuristicTypeActionSuggestion indicates a suggested action for similar situations.
	HeuristicTypeActionSuggestion HeuristicType = "action_suggestion"
	// HeuristicTypeParameterAdjustment indicates a recommended parameter modification.
	HeuristicTypeParameterAdjustment HeuristicType = "parameter_adjustment"
	// HeuristicTypeStrategyChange indicates a recommended strategy change.
	HeuristicTypeStrategyChange HeuristicType = "strategy_change"
	// HeuristicTypeToolSelection indicates a recommended tool selection.
	HeuristicTypeToolSelection HeuristicType = "tool_selection"
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
