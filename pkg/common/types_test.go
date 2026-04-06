package common

import (
	"testing"
	"time"
)

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{"pending", StatusPending, "pending"},
		{"running", StatusRunning, "running"},
		{"paused", StatusPaused, "paused"},
		{"completed", StatusCompleted, "completed"},
		{"failed", StatusFailed, "failed"},
		{"canceled", StatusCanceled, "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestSecurityLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    SecurityLevel
		expected string
	}{
		{"safe", LevelSafe, "safe"},
		{"sensitive", LevelSensitive, "sensitive"},
		{"high_risk", LevelHighRisk, "high_risk"},
		{"unknown", SecurityLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("SecurityLevel.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseSecurityLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SecurityLevel
	}{
		{"safe", "safe", LevelSafe},
		{"sensitive", "sensitive", LevelSensitive},
		{"high_risk", "high_risk", LevelHighRisk},
		{"unknown", "invalid", LevelSafe},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseSecurityLevel(tt.input); got != tt.expected {
				t.Errorf("ParseSecurityLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIntentConstants(t *testing.T) {
	intents := []struct {
		name     string
		intent   Intent
		expected string
	}{
		{"chat", IntentChat, "chat"},
		{"task", IntentTask, "task"},
		{"clarification", IntentClarification, "clarification"},
		{"follow_up", IntentFollowUp, "follow_up"},
		{"feedback", IntentFeedback, "feedback"},
	}

	for _, tt := range intents {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.intent) != tt.expected {
				t.Errorf("Intent %s = %q, want %q", tt.name, tt.intent, tt.expected)
			}
		})
	}
}

func TestActionTypeConstants(t *testing.T) {
 actionTypes := []struct {
		name     string
		at       ActionType
		expected string
	}{
		{"tool_call", ActionTypeToolCall, "tool_call"},
		{"skill_invoke", ActionTypeSkillInvoke, "skill_invoke"},
		{"sub_agent_delegate", ActionTypeSubAgentDelegate, "sub_agent_delegate"},
		{"no_action", ActionTypeNoAction, "no_action"},
	}

	for _, tt := range actionTypes {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.at) != tt.expected {
				t.Errorf("ActionType %s = %q, want %q", tt.name, tt.at, tt.expected)
			}
		})
	}
}

func TestQuestionTypeConstants(t *testing.T) {
	types := []struct {
		name     string
		qt       QuestionType
		expected string
	}{
		{"authorization", QuestionTypeAuthorization, "authorization"},
		{"confirmation", QuestionTypeConfirmation, "confirmation"},
		{"clarification", QuestionTypeClarification, "clarification"},
		{"custom_input", QuestionTypeCustomInput, "custom_input"},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.qt) != tt.expected {
				t.Errorf("QuestionType %s = %q, want %q", tt.name, tt.qt, tt.expected)
			}
		})
	}
}

func TestSessionStatusConstants(t *testing.T) {
	statuses := []struct {
		name     string
		status   SessionStatus
		expected string
	}{
		{"active", SessionStatusActive, "active"},
		{"paused", SessionStatusPaused, "paused"},
		{"ended", SessionStatusEnded, "ended"},
		{"archived", SessionStatusArchived, "archived"},
	}

	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("SessionStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestPlanStatusConstants(t *testing.T) {
	statuses := []struct {
		name     string
		status   PlanStatus
		expected string
	}{
		{"pending", PlanStatusPending, "pending"},
		{"running", PlanStatusRunning, "running"},
		{"completed", PlanStatusCompleted, "completed"},
		{"failed", PlanStatusFailed, "failed"},
		{"revised", PlanStatusRevised, "revised"},
	}

	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("PlanStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestStepStatusConstants(t *testing.T) {
	statuses := []struct {
		name     string
		status   StepStatus
		expected string
	}{
		{"pending", StepStatusPending, "pending"},
		{"running", StepStatusRunning, "running"},
		{"completed", StepStatusCompleted, "completed"},
		{"failed", StepStatusFailed, "failed"},
		{"skipped", StepStatusSkipped, "skipped"},
	}

	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("StepStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestMemoryItemTypeConstants(t *testing.T) {
	types := []struct {
		name     string
		mt       MemoryItemType
		expected string
	}{
		{"fact", MemoryItemTypeFact, "fact"},
		{"preference", MemoryItemTypePreference, "preference"},
		{"pattern", MemoryItemTypePattern, "pattern"},
		{"constraint", MemoryItemTypeConstraint, "constraint"},
		{"correction", MemoryItemTypeCorrection, "correction"},
		{"instruction", MemoryItemTypeInstruction, "instruction"},
		{"observation", MemoryItemTypeObservation, "observation"},
		{"thought", MemoryItemTypeThought, "thought"},
		{"action", MemoryItemTypeAction, "action"},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mt) != tt.expected {
				t.Errorf("MemoryItemType %s = %q, want %q", tt.name, tt.mt, tt.expected)
			}
		})
	}
}

func TestMemorySourceConstants(t *testing.T) {
	sources := []struct {
		name     string
		ms       MemorySource
		expected string
	}{
		{"user", MemorySourceUser, "user"},
		{"system", MemorySourceSystem, "system"},
		{"inference", MemorySourceInference, "inference"},
		{"evolution", MemorySourceEvolution, "evolution"},
		{"action", MemorySourceAction, "action"},
		{"tool", MemorySourceTool, "tool"},
	}

	for _, tt := range sources {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ms) != tt.expected {
				t.Errorf("MemorySource %s = %q, want %q", tt.name, tt.ms, tt.expected)
			}
		})
	}
}

func TestTokenUsage(t *testing.T) {
	usage := TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", usage.CompletionTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", usage.TotalTokens)
	}
}

func TestDuration(t *testing.T) {
	d := NewDuration()
	
	if d.Start.IsZero() {
		t.Error("Start time should not be zero")
	}
	if !d.End.IsZero() {
		t.Error("End time should be zero before Stop()")
	}

	time.Sleep(10 * time.Millisecond)
	d.Stop()

	if d.End.IsZero() {
		t.Error("End time should not be zero after Stop()")
	}
	if d.Total <= 0 {
		t.Errorf("Total duration should be positive, got %v", d.Total)
	}
}

func TestPairs(t *testing.T) {
	pairs := Pairs{
		{Key: "name", Value: "test"},
		{Key: "count", Value: 42},
		{Key: "active", Value: true},
	}

	m := pairs.ToMap()

	if len(m) != 3 {
		t.Errorf("ToMap() length = %d, want 3", len(m))
	}
	if m["name"] != "test" {
		t.Errorf("ToMap()[name] = %v, want 'test'", m["name"])
	}
	if m["count"] != 42 {
		t.Errorf("ToMap()[count] = %v, want 42", m["count"])
	}
	if m["active"] != true {
		t.Errorf("ToMap()[active] = %v, want true", m["active"])
	}
}

func TestPairs_Empty(t *testing.T) {
	pairs := Pairs{}
	m := pairs.ToMap()

	if m == nil {
		t.Error("ToMap() should return non-nil map for empty Pairs")
	}
	if len(m) != 0 {
		t.Errorf("ToMap() length = %d, want 0", len(m))
	}
}
