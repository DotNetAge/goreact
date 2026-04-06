package orchestration

import (
	"testing"
	"time"
)

func TestDefaultConcurrencyConfig(t *testing.T) {
	config := DefaultConcurrencyConfig()

	if config.MaxConcurrent != 5 {
		t.Errorf("MaxConcurrent = %d, want 5", config.MaxConcurrent)
	}
	if config.RateLimitPerAgent != 10 {
		t.Errorf("RateLimitPerAgent = %d, want 10", config.RateLimitPerAgent)
	}
	if config.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", config.Timeout)
	}
	if config.RetryCount != 3 {
		t.Errorf("RetryCount = %d, want 3", config.RetryCount)
	}
}

func TestTask(t *testing.T) {
	task := &Task{
		ID:          "task-001",
		Name:        "Process Data",
		Description: "Process user data files",
		Input:       map[string]any{"file": "data.csv"},
		Priority:    1,
		Timeout:     time.Minute,
	}

	if task.ID != "task-001" {
		t.Errorf("ID = %q, want 'task-001'", task.ID)
	}
	if task.Priority != 1 {
		t.Errorf("Priority = %d, want 1", task.Priority)
	}
}

func TestSubTask(t *testing.T) {
	subtask := &SubTask{
		Name:                 "parse-file",
		ParentName:           "process-data",
		Description:          "Parse the CSV file",
		RequiredCapabilities: []string{"file-reading", "csv-parsing"},
		Dependencies:         []string{"download-file"},
		Priority:             1,
	}

	if len(subtask.RequiredCapabilities) != 2 {
		t.Errorf("len(RequiredCapabilities) = %d, want 2", len(subtask.RequiredCapabilities))
	}
	if subtask.ParentName != "process-data" {
		t.Errorf("ParentName = %q, want 'process-data'", subtask.ParentName)
	}
}

func TestOrchestrationPlan(t *testing.T) {
	plan := &OrchestrationPlan{
		Name:     "data-processing-plan",
		TaskName: "process-data",
		SubTasks: []*SubTask{
			{Name: "parse"},
			{Name: "transform"},
		},
		ExecutionOrder: [][]string{
			{"parse"},
			{"transform"},
		},
	}

	if len(plan.SubTasks) != 2 {
		t.Errorf("len(SubTasks) = %d, want 2", len(plan.SubTasks))
	}
	if len(plan.ExecutionOrder) != 2 {
		t.Errorf("len(ExecutionOrder) = %d, want 2", len(plan.ExecutionOrder))
	}
}

func TestCapabilities(t *testing.T) {
	caps := &Capabilities{
		Skills:       []string{"code-review", "testing"},
		Tools:        []string{"bash", "read_file"},
		Domains:      []string{"software"},
		Languages:    []string{"go", "python"},
		MaxComplexity: 10,
	}

	if len(caps.Skills) != 2 {
		t.Errorf("len(Skills) = %d, want 2", len(caps.Skills))
	}
	if len(caps.Tools) != 2 {
		t.Errorf("len(Tools) = %d, want 2", len(caps.Tools))
	}
}

func TestAgentMatch(t *testing.T) {
	match := &AgentMatch{
		AgentName:       "assistant",
		CapabilityScore: 0.95,
		LoadScore:       0.8,
		HistoryScore:    0.9,
		TotalScore:      0.88,
	}

	if match.AgentName != "assistant" {
		t.Errorf("AgentName = %q, want 'assistant'", match.AgentName)
	}
	if match.TotalScore != 0.88 {
		t.Errorf("TotalScore = %f, want 0.88", match.TotalScore)
	}
}

func TestExecutionPhase(t *testing.T) {
	phases := []ExecutionPhase{
		PhaseIdle,
		PhasePlanning,
		PhaseSelecting,
		PhaseExecuting,
		PhaseAggregating,
		PhaseSuspended,
		PhaseRetrying,
		PhaseCompleted,
		PhaseFailed,
	}

	for _, phase := range phases {
		if string(phase) == "" {
			t.Errorf("Phase %v should not be empty string", phase)
		}
	}
}

func TestAgentStatus(t *testing.T) {
	statuses := []AgentStatus{
		AgentStatusPending,
		AgentStatusRunning,
		AgentStatusSuspended,
		AgentStatusCompleted,
		AgentStatusFailed,
		AgentStatusBlocked,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("Status %v should not be empty string", status)
		}
	}
}

func TestAgentState(t *testing.T) {
	state := &AgentState{
		AgentName:   "assistant",
		SubTaskName: "parse-file",
		Status:      AgentStatusRunning,
		StartTime:   time.Now(),
	}

	if state.AgentName != "assistant" {
		t.Errorf("AgentName = %q, want 'assistant'", state.AgentName)
	}
	if state.Status != AgentStatusRunning {
		t.Errorf("Status = %q, want 'running'", state.Status)
	}
}

func TestOrchestrationState(t *testing.T) {
	state := &OrchestrationState{
		SessionName:      "session-123",
		ExecutionPhase:   PhaseExecuting,
		AgentStates:      make(map[string]*AgentState),
		CompletedSubTasks: []string{"parse", "validate"},
		FailedSubTasks:    []string{},
	}

	if state.SessionName != "session-123" {
		t.Errorf("SessionName = %q, want 'session-123'", state.SessionName)
	}
	if len(state.CompletedSubTasks) != 2 {
		t.Errorf("len(CompletedSubTasks) = %d, want 2", len(state.CompletedSubTasks))
	}
}

func TestSubResult(t *testing.T) {
	result := &SubResult{
		SubTaskName: "parse-file",
		AgentName:   "assistant",
		Success:     true,
		Output:      map[string]any{"rows": 100},
		Duration:    500 * time.Millisecond,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Duration != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", result.Duration)
	}
}

func TestResult(t *testing.T) {
	result := &Result{
		TaskName:    "process-data",
		FinalOutput: map[string]any{"processed": true},
		Success:     true,
		Duration:    2 * time.Second,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Duration != 2*time.Second {
		t.Errorf("Duration = %v, want 2s", result.Duration)
	}
}

func TestDependencyType(t *testing.T) {
	types := []DependencyType{
		DependencySequential,
		DependencyData,
		DependencyResource,
		DependencyConditional,
	}

	for _, dt := range types {
		if string(dt) == "" {
			t.Errorf("DependencyType %v should not be empty", dt)
		}
	}
}

func TestMergeStrategy(t *testing.T) {
	strategies := []MergeStrategy{
		MergeStrategyConcat,
		MergeStrategyStructured,
		MergeStrategyLLM,
		MergeStrategyVoting,
	}

	for _, s := range strategies {
		if string(s) == "" {
			t.Errorf("MergeStrategy %v should not be empty", s)
		}
	}
}

func TestGraph(t *testing.T) {
	graph := &Graph{
		Nodes: []string{"A", "B", "C"},
		Edges: []*Edge{
			{From: "A", To: "B", Type: DependencySequential},
			{From: "B", To: "C", Type: DependencySequential},
		},
	}

	if len(graph.Nodes) != 3 {
		t.Errorf("len(Nodes) = %d, want 3", len(graph.Nodes))
	}
	if len(graph.Edges) != 2 {
		t.Errorf("len(Edges) = %d, want 2", len(graph.Edges))
	}
}

func TestPendingQuestion(t *testing.T) {
	pq := &PendingQuestion{
		ID:           "q-001",
		AgentName:    "assistant",
		SubTaskName:  "process",
		Question:     "Continue with operation?",
		QuestionType: "confirmation",
		Options:      []string{"Yes", "No"},
	}

	if pq.ID != "q-001" {
		t.Errorf("ID = %q, want 'q-001'", pq.ID)
	}
	if len(pq.Options) != 2 {
		t.Errorf("len(Options) = %d, want 2", len(pq.Options))
	}
}

func TestSnapshotLevel(t *testing.T) {
	levels := []SnapshotLevel{
		SnapshotLevelOrchestration,
		SnapshotLevelAgent,
	}

	for _, level := range levels {
		if string(level) == "" {
			t.Errorf("SnapshotLevel %v should not be empty", level)
		}
	}
}

func TestOrchestrationSnapshot(t *testing.T) {
	snapshot := &OrchestrationSnapshot{
		SessionName:      "session-123",
		ExecutionPhase:   PhaseSuspended,
		AgentStates:      make(map[string]*AgentState),
		PendingQuestions: []*PendingQuestion{{ID: "q-001"}},
		Checksum:         "abc123",
	}

	if snapshot.Checksum != "abc123" {
		t.Errorf("Checksum = %q, want 'abc123'", snapshot.Checksum)
	}
	if len(snapshot.PendingQuestions) != 1 {
		t.Errorf("len(PendingQuestions) = %d, want 1", len(snapshot.PendingQuestions))
	}
}

func TestErrorType(t *testing.T) {
	errors := []ErrorType{
		ErrorPlanningFailed,
		ErrorAgentSelectionFailed,
		ErrorExecutionFailed,
		ErrorTimeout,
		ErrorResourceExhausted,
		ErrorDependencyViolation,
	}

	for _, et := range errors {
		if string(et) == "" {
			t.Errorf("ErrorType %v should not be empty", et)
		}
	}
}

func TestAlert(t *testing.T) {
	alert := &Alert{
		Name:      "high_latency",
		Condition: "latency > 5s",
		Level:     AlertLevelWarning,
		Message:   "Request latency is high",
		Timestamp: time.Now(),
	}

	if alert.Level != AlertLevelWarning {
		t.Errorf("Level = %q, want 'warning'", alert.Level)
	}
}

func TestDecompositionStrategy(t *testing.T) {
	strategies := []DecompositionStrategy{
		DecompositionRule,
		DecompositionLLM,
		DecompositionHybrid,
	}

	for _, s := range strategies {
		if string(s) == "" {
			t.Errorf("DecompositionStrategy %v should not be empty", s)
		}
	}
}

func TestMetrics(t *testing.T) {
	metrics := &Metrics{
		OrchestrationLatency:   100 * time.Millisecond,
		ExecutionTime:          2 * time.Second,
		SubTaskSuccessRate:     0.95,
		AgentUtilization:       0.8,
		ParallelismEfficiency:  0.9,
		ActiveAgents:           3,
		QueuedTasks:            2,
	}

	if metrics.SubTaskSuccessRate != 0.95 {
		t.Errorf("SubTaskSuccessRate = %f, want 0.95", metrics.SubTaskSuccessRate)
	}
	if metrics.ActiveAgents != 3 {
		t.Errorf("ActiveAgents = %d, want 3", metrics.ActiveAgents)
	}
}
