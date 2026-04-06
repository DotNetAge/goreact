package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/observer"
	"github.com/DotNetAge/goreact/pkg/orchestration"
	"github.com/DotNetAge/goreact/pkg/prompt"
	"github.com/DotNetAge/goreact/pkg/resource"
	goreactskill "github.com/DotNetAge/goreact/pkg/skill"
	"github.com/DotNetAge/goreact/pkg/tool"
)

// TestStateManagement tests the state management workflow
func TestStateManagement(t *testing.T) {
	// Create a new state
	state := core.NewState("test-session", "What is the weather?", 10, 3)

	// Simulate ReAct loop
	thought := core.NewThought("I need to check the weather", "User wants weather info", "act", 0.9)
	thought.WithAction(&core.ActionIntent{
		Type:   "tool_call",
		Target: "weather_api",
		Params: map[string]any{"location": "Beijing"},
	})
	state.AddThought(thought)

	action := thought.ToAction()
	state.AddAction(action)

	observation := &core.Observation{
		Content: "Beijing: Sunny, 25°C",
	}
	state.AddObservation(observation)

	state.IncrementStep()

	// Verify state
	if len(state.Thoughts) != 1 {
		t.Errorf("Expected 1 thought, got %d", len(state.Thoughts))
	}
	if len(state.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(state.Actions))
	}
	if len(state.Observations) != 1 {
		t.Errorf("Expected 1 observation, got %d", len(state.Observations))
	}
	if state.CurrentStep != 1 {
		t.Errorf("Expected current step 1, got %d", state.CurrentStep)
	}
}

// TestPlanExecution tests plan creation and execution
func TestPlanExecution(t *testing.T) {
	plan := core.NewPlan("session-123", "Analyze the codebase")

	// Add steps
	plan.AddStep("read_files", "Read all source files")
	plan.AddStep("analyze", "Analyze code structure")
	plan.AddStep("report", "Generate report")

	// Simulate execution
	for i := 0; i < 3; i++ {
		step := plan.GetCurrentStep()
		if step == nil {
			t.Fatalf("Step %d should not be nil", i)
		}

		step.Status = common.StepStatusRunning
		time.Sleep(1 * time.Millisecond)
		step.Status = common.StepStatusCompleted

		if !plan.IsComplete() {
			plan.AdvanceStep()
		}
	}

	if !plan.IsComplete() {
		t.Error("Plan should be complete")
	}

	plan.MarkSuccess()
	if plan.Status != common.PlanStatusCompleted {
		t.Errorf("Expected status completed, got %s", plan.Status)
	}
}

// TestSkillCreationAndExecution tests skill workflow
func TestSkillCreationAndExecution(t *testing.T) {
	skill := goreactskill.NewSkill("code-review", "Reviews code for quality issues", "assistant")
	skill.WithIntent("review_code")
	skill.WithTemplate("Review the following code:\n{{.Params.code}}")
	skill.WithParameter(goreactskill.Parameter{
		Name:        "code",
		Type:        "string",
		Required:    true,
		Description: "Code to review",
	})
	skill.WithStep(goreactskill.ExecutionStep{
		ToolName:        "read_file",
		Description:     "Read the source file",
		ExpectedOutcome: "File content loaded",
		OnError:         "stop",
	})

	if skill.Name != "code-review" {
		t.Errorf("Expected name 'code-review', got %s", skill.Name)
	}
	if len(skill.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(skill.Parameters))
	}
	if len(skill.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(skill.Steps))
	}

	execPlan := goreactskill.NewSkillExecutionPlan("code-review")
	execPlan.IncrementExecution(true)

	if execPlan.ExecutionCount != 1 {
		t.Errorf("Expected execution count 1, got %d", execPlan.ExecutionCount)
	}
}

// TestToolRegistration tests tool registration workflow
func TestToolRegistration(t *testing.T) {
	rm := resource.NewResourceManager()

	baseTool := tool.NewBaseTool("read_file", "Read file contents", common.LevelSafe, true)
	baseTool.WithParameter(tool.Parameter{
		Name:        "path",
		Type:        "string",
		Required:    true,
		Description: "File path",
	})

	err := rm.RegisterTool("read_file", baseTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	_, exists := rm.GetTool("read_file")
	if !exists {
		t.Error("Tool should exist")
	}

	if baseTool.Name() != "read_file" {
		t.Errorf("Expected name 'read_file', got %s", baseTool.Name())
	}
	if baseTool.SecurityLevel() != common.LevelSafe {
		t.Errorf("Expected security level safe, got %v", baseTool.SecurityLevel())
	}
}

// TestObservabilityWorkflow tests observability workflow
func TestObservabilityWorkflow(t *testing.T) {
	probe := observer.NewProbe(nil)

	span := probe.StartSpan("query_processing")
	span.SetAttr("session", "test-session")
	span.AddEvent("query_received", map[string]any{"query": "test"})

	probe.RecordMetric("requests_total", 1)
	probe.RecordMetric("latency_ms", 150)

	probe.EndSpan(span)

	trace := probe.GetTrace()
	if len(trace.Spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(trace.Spans))
	}

	metrics := probe.GetMetrics()
	if metrics["requests_total"] != 1 {
		t.Errorf("Expected requests_total 1, got %v", metrics["requests_total"])
	}
}

// TestTokenTracking tests token tracking workflow
func TestTokenTracking(t *testing.T) {
	tracker := observer.NewTokenTracker()

	tracker.Track("gpt-4", 100, 50)
	tracker.Track("gpt-4", 200, 100)
	tracker.Track("gpt-3.5-turbo", 50, 25)

	total := tracker.GetTotal()
	if total.PromptTokens != 350 {
		t.Errorf("Expected prompt tokens 350, got %d", total.PromptTokens)
	}
	if total.TotalTokens != 525 {
		t.Errorf("Expected total tokens 525, got %d", total.TotalTokens)
	}

	gpt4Usage := tracker.GetByModel("gpt-4")
	if gpt4Usage.TotalTokens != 450 {
		t.Errorf("Expected gpt-4 total tokens 450, got %d", gpt4Usage.TotalTokens)
	}
}

// TestPromptBuilding tests prompt building workflow
func TestPromptBuilding(t *testing.T) {
	sysPrompt := &prompt.SystemPrompt{
		Role:         "Assistant",
		Behavior:     "Helpful and accurate",
		Constraints:  "Do not make up information",
		OutputFormat: "Thought/Action/Observation",
	}

	ragContext := &prompt.RAGContext{
		Query: "What is Go?",
		Mode:  prompt.RAGModeHybrid,
		Documents: []*prompt.Document{
			{ID: "doc1", Content: "Go is a programming language", Score: 0.95},
		},
	}

	example := &prompt.Example{
		ID:           "ex-001",
		Question:     "What is 2+2?",
		Thoughts:     []string{"Need to add 2 and 2"},
		Actions:      []string{"calculate[2+2]"},
		Observations: []string{"4"},
		FinalAnswer:  "4",
	}

	if sysPrompt.Role != "Assistant" {
		t.Errorf("Expected role 'Assistant', got %s", sysPrompt.Role)
	}
	if len(ragContext.Documents) != 1 {
		t.Errorf("Expected 1 document, got %d", len(ragContext.Documents))
	}
	if example.FinalAnswer != "4" {
		t.Errorf("Expected answer '4', got %s", example.FinalAnswer)
	}
}

// TestOrchestrationWorkflow tests orchestration workflow
func TestOrchestrationWorkflow(t *testing.T) {
	state := &orchestration.OrchestrationState{
		SessionName:      "orch-123",
		ExecutionPhase:   orchestration.PhasePlanning,
		AgentStates:      make(map[string]*orchestration.AgentState),
		CompletedSubTasks: []string{},
	}

	state.AgentStates["assistant"] = &orchestration.AgentState{
		AgentName:   "assistant",
		SubTaskName: "analyze",
		Status:      orchestration.AgentStatusRunning,
		StartTime:   time.Now(),
	}

	subTasks := []*orchestration.SubTask{
		{Name: "parse", Description: "Parse input"},
		{Name: "analyze", Description: "Analyze data"},
		{Name: "report", Description: "Generate report"},
	}

	plan := &orchestration.OrchestrationPlan{
		Name:           "data-processing",
		TaskName:       "process-data",
		SubTasks:       subTasks,
		ExecutionOrder: [][]string{{"parse"}, {"analyze"}, {"report"}},
	}

	state.Plan = plan
	state.ExecutionPhase = orchestration.PhaseExecuting

	if state.ExecutionPhase != orchestration.PhaseExecuting {
		t.Errorf("Expected phase executing, got %s", state.ExecutionPhase)
	}
	if len(state.Plan.SubTasks) != 3 {
		t.Errorf("Expected 3 sub-tasks, got %d", len(state.Plan.SubTasks))
	}
}

// TestConcurrencyConfig tests concurrency configuration
func TestConcurrencyConfig(t *testing.T) {
	config := orchestration.DefaultConcurrencyConfig()

	if config.MaxConcurrent != 5 {
		t.Errorf("Expected max concurrent 5, got %d", config.MaxConcurrent)
	}
	if config.RetryCount != 3 {
		t.Errorf("Expected retry count 3, got %d", config.RetryCount)
	}
}

// TestContextPropagation tests context propagation through components
func TestContextPropagation(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, common.ContextKeySession, "test-session")
	ctx = context.WithValue(ctx, common.ContextKeyTraceID, "trace-123")

	session := ctx.Value(common.ContextKeySession)
	if session != "test-session" {
		t.Errorf("Expected session 'test-session', got %v", session)
	}

	traceID := ctx.Value(common.ContextKeyTraceID)
	if traceID != "trace-123" {
		t.Errorf("Expected trace ID 'trace-123', got %v", traceID)
	}
}

// TestErrorHandling tests error handling workflow
func TestErrorHandling(t *testing.T) {
	state := core.NewState("test", "input", 10, 3)

	action := core.NewAction(common.ActionTypeToolCall, "failing_tool", nil)
	state.AddAction(action)

	result := core.NewActionResult(false, nil).
		WithError("Tool execution failed").
		WithDuration(100 * time.Millisecond)

	if result.Success {
		t.Error("Result should not be successful")
	}
	if result.Error != "Tool execution failed" {
		t.Errorf("Expected error message, got %s", result.Error)
	}

	state.IncrementRetry()
	if !state.CanRetry() {
		t.Error("Should be able to retry")
	}
}

// TestNegativePrompts tests negative prompt configuration
func TestNegativePrompts(t *testing.T) {
	groups := prompt.DefaultNegativePromptGroups()

	if len(groups) < 1 {
		t.Error("Should have at least one negative prompt group")
	}

	found := false
	for _, g := range groups {
		if g.ID == "safety" {
			found = true
			if len(g.Prompts) < 1 {
				t.Error("Safety group should have prompts")
			}
		}
	}
	if !found {
		t.Error("Safety group should exist")
	}
}

// TestExamples tests example configuration
func TestExamples(t *testing.T) {
	examples := prompt.DefaultExamples()

	if len(examples) < 1 {
		t.Error("Should have at least one example")
	}

	for _, ex := range examples {
		if ex.Question == "" {
			t.Error("Example should have a question")
		}
		if len(ex.Thoughts) == 0 {
			t.Error("Example should have thoughts")
		}
		if ex.FinalAnswer == "" {
			t.Error("Example should have a final answer")
		}
	}
}

// TestResourceManagement tests resource manager workflow
func TestResourceManagement(t *testing.T) {
	rm := resource.NewResourceManager()

	rm.RegisterAgent("assistant", map[string]any{"model": "gpt-4"})
	rm.RegisterTool("bash", map[string]any{"type": "cli"})
	rm.RegisterSkill("code-review", goreactskill.NewSkill("code-review", "test", "assistant"))
	rm.RegisterModel("gpt-4", &resource.Model{
		Name:     "gpt-4",
		Provider: "openai",
	})

	rm.RegisterAgentTools("assistant", []string{"bash"})

	tools := rm.GetToolsByAgent("assistant")
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	resources := rm.GetAllResources()
	if len(resources["agents"]) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(resources["agents"]))
	}
}

// TestNodeCreation tests various node creation
func TestNodeCreation(t *testing.T) {
	agentNode := core.NewAgentNode("assistant", "AI assistant", "general", "gpt-4")
	if agentNode.Name != "assistant" {
		t.Errorf("Expected name 'assistant', got %s", agentNode.Name)
	}

	sessionNode := core.NewSessionNode("session-123", "user-1")
	if sessionNode.UserName != "user-1" {
		t.Errorf("Expected user name 'user-1', got %s", sessionNode.UserName)
	}

	messageNode := core.NewMessageNode("session-123", "user", "Hello!")
	if messageNode.Role != "user" {
		t.Errorf("Expected role 'user', got %s", messageNode.Role)
	}

	memoryNode := core.NewMemoryItemNode("session-123", "User prefers dark mode", common.MemoryItemTypePreference)
	if memoryNode.Type != common.MemoryItemTypePreference {
		t.Errorf("Expected type 'preference', got %s", memoryNode.Type)
	}

	questionNode := core.NewPendingQuestionNode("session-123", common.QuestionTypeConfirmation, "Continue?")
	if questionNode.Question != "Continue?" {
		t.Errorf("Expected question 'Continue?', got %s", questionNode.Question)
	}
}
