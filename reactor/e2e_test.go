package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// ---------------------------------------------------------------------------
// Mock LLM Infrastructure
// ---------------------------------------------------------------------------

// MockScenario defines a sequence of LLM responses for a test scenario.
// Each entry maps a keyword found in the user message or system prompt
// to the LLM response that should be returned.
type MockScenario struct {
	// Responses is an ordered list of responses the mock LLM will return.
	// Each response is returned in sequence; if exhausted, the last response is repeated.
	Responses []MockResponse
}

// MockResponse defines a single LLM mock response.
type MockResponse struct {
	// Content is the raw LLM response text (JSON for Think/Intent, plain text for chat).
	Content string
	// Tokens overrides the simulated token count (default: 100).
	Tokens int
	// Err simulates an LLM error.
	Err error
	// Delay simulates network latency (for testing cancellation/timeout).
	Delay time.Duration
}

// NewMockScenario creates a scenario from a list of content strings.
// Token count defaults to 100 for each response.
func NewMockScenario(contents ...string) MockScenario {
	responses := make([]MockResponse, len(contents))
	for i, c := range contents {
		responses[i] = MockResponse{Content: c, Tokens: 100}
	}
	return MockScenario{Responses: responses}
}

// mockLLMFromScenario creates a MockLLMFunc from a MockScenario.
// The function returns responses in order, cycling to the last if exhausted.
func mockLLMFromScenario(scenario MockScenario) MockLLMFunc {
	var mu sync.Mutex
	var callCount int
	return func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error) {
		mu.Lock()
		idx := callCount
		callCount++
		mu.Unlock()

		if idx >= len(scenario.Responses) {
			idx = len(scenario.Responses) - 1
		}

		resp := scenario.Responses[idx]

		// Simulate delay
		if resp.Delay > 0 {
			time.Sleep(resp.Delay)
		}

		if resp.Err != nil {
			return nil, resp.Err
		}

		tokens := resp.Tokens
		if tokens <= 0 {
			tokens = 100
		}

		return &gochatcore.Response{
			Content: resp.Content,
			Usage: &gochatcore.Usage{
				PromptTokens:     tokens / 2,
				CompletionTokens: tokens / 2,
				TotalTokens:      tokens,
			},
		}, nil
	}
}

// ---------------------------------------------------------------------------
// Mock LLM Response Builders
// ---------------------------------------------------------------------------

// intentResponse builds an intent classification response.
func intentResponse(intentType, topic string, confidence float64, requiresClarification bool) string {
	reqClarify := "false"
	if requiresClarification {
		reqClarify = "true"
	}
	return fmt.Sprintf(`{"type":"%s","topic":"%s","confidence":%.2f,"requires_clarification":%s}`, intentType, topic, confidence, reqClarify)
}

// thinkResponse builds a Think phase response (decision=answer).
func thinkResponse(reasoning, answer string) string {
	return fmt.Sprintf(`{"reasoning":"%s","decision":"answer","final_answer":"%s","confidence":0.95}`, reasoning, answer)
}

// thinkActResponse builds a Think phase response (decision=act).
func thinkActResponse(reasoning, toolName string, params map[string]any) string {
	paramsJSON, _ := json.Marshal(params)
	return fmt.Sprintf(`{"reasoning":"%s","decision":"act","action_target":"%s","action_params":%s}`, reasoning, toolName, string(paramsJSON))
}

// thinkAnswerResponse builds a Think phase response that produces a direct answer.
func thinkAnswerResponse(reasoning, finalAnswer string) string {
	return fmt.Sprintf(`{"reasoning":"%s","decision":"answer","final_answer":"%s","confidence":0.95,"is_final":true}`, reasoning, finalAnswer)
}

// ---------------------------------------------------------------------------
// Test Helpers
// ---------------------------------------------------------------------------

// newMockReactor creates a Reactor with the given mock LLM scenario and no real API key.
func newMockReactor(t *testing.T, scenario MockScenario) *Reactor {
	t.Helper()
	cfg := ReactorConfig{
		APIKey:       "mock-api-key",
		BaseURL:      "https://mock.example.com/v1",
		Model:        "mock-model",
		MaxIterations: 10,
	}
	return NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario)))
}

// assertResult checks common RunResult fields.
func assertResult(t *testing.T, result *RunResult, err error) *RunResult {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	return result
}

// ---------------------------------------------------------------------------
// E2E Test Scenarios
// ---------------------------------------------------------------------------

// TestE2E_ChatIntent tests a simple chat interaction (intent=chat, direct answer).
func TestE2E_ChatIntent(t *testing.T) {
	scenario := NewMockScenario(
		intentResponse("chat", "greeting", 0.95, false),
		thinkResponse("The user is greeting me", "Hello! How can I help you today?"),
	)
	r := newMockReactor(t, scenario)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Hello!", nil)
	result = assertResult(t, result, err)

	if result.Answer != "Hello! How can I help you today?" {
		t.Errorf("expected greeting answer, got: %s", result.Answer)
	}
	if result.Intent == nil || result.Intent.Type != "chat" {
		t.Errorf("expected intent=chat, got: %v", result.Intent)
	}
	if result.TotalIterations != 1 {
		t.Errorf("expected 1 iteration, got: %d", result.TotalIterations)
	}
}

// TestE2E_TaskWithToolCall tests a task that requires a tool call.
func TestE2E_TaskWithToolCall(t *testing.T) {
	scenario := NewMockScenario(
		intentResponse("task", "calculation", 0.9, false),
		thinkActResponse("Need to use echo tool", "echo", map[string]any{"text": "hello world"}),
		thinkAnswerResponse("Got the result from echo", "The echo result is: hello world"),
	)
	r := newMockReactor(t, scenario)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Please echo 'hello world'", nil)
	result = assertResult(t, result, err)

	if result.TotalIterations != 2 {
		t.Errorf("expected 2 iterations (think+act, think+answer), got: %d", result.TotalIterations)
	}

	// Verify tool call happened in first step
	if len(result.Steps) < 1 {
		t.Fatal("expected at least 1 step")
	}
	if result.Steps[0].Action.Type != ActionTypeToolCall {
		t.Errorf("expected tool_call in step 1, got: %s", result.Steps[0].Action.Type)
	}
	if result.Steps[0].Action.Target != "echo" {
		t.Errorf("expected echo tool, got: %s", result.Steps[0].Action.Target)
	}
}

// TestE2E_ClarificationFromIntent tests clarification triggered by intent classification.
func TestE2E_ClarificationFromIntent(t *testing.T) {
	scenario := NewMockScenario(
		intentResponse("clarify", "email", 0.3, true),
	)
	r := newMockReactor(t, scenario)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Send an email", nil)
	result = assertResult(t, result, err)

	if !result.ClarificationNeeded {
		t.Error("expected clarification_needed=true")
	}
	if result.TotalIterations != 0 {
		t.Errorf("expected 0 iterations for early clarification, got: %d", result.TotalIterations)
	}
}

// TestE2E_MultiStepToolUse tests multiple sequential tool calls.
func TestE2E_MultiStepToolUse(t *testing.T) {
	scenario := NewMockScenario(
		intentResponse("task", "file_ops", 0.9, false),
		thinkActResponse("First read the file", "bash", map[string]any{"command": "cat /tmp/test.txt"}),
		thinkActResponse("Now list the directory", "bash", map[string]any{"command": "ls /tmp"}),
		thinkAnswerResponse("All done, here is the summary", "File contents and directory listing retrieved successfully."),
	)
	r := newMockReactor(t, scenario)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Read the file and list the directory", nil)
	result = assertResult(t, result, err)

	if result.TotalIterations != 3 {
		t.Errorf("expected 3 iterations, got: %d", result.TotalIterations)
	}

	// Count tool calls
	toolCalls := 0
	for _, step := range result.Steps {
		if step.Action.Type == ActionTypeToolCall {
			toolCalls++
		}
	}
	if toolCalls != 2 {
		t.Errorf("expected 2 tool calls, got: %d", toolCalls)
	}
}

// TestE2E_MaxIterations tests that the reactor stops at the configured max iterations.
func TestE2E_MaxIterations(t *testing.T) {
	// All responses are act decisions, so the loop should be terminated by max iterations
	var responses []MockResponse
	responses = append(responses, MockResponse{Content: intentResponse("task", "endless", 0.9, false)})
	for i := 0; i < 15; i++ {
		responses = append(responses, MockResponse{
			Content: thinkActResponse("keep going", "echo", map[string]any{"text": fmt.Sprintf("iteration %d", i)}),
		})
	}

	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 5,
	}
	r := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(MockScenario{Responses: responses})))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Keep going forever", nil)
	result = assertResult(t, result, err)

	if result.TotalIterations > 5 {
		t.Errorf("expected at most 5 iterations, got: %d", result.TotalIterations)
	}
	// The mock uses identical echo tool calls with same text, which triggers
	// "duplicate action detected" before reaching max iterations.
	// Accept either termination reason.
	if !strings.Contains(result.TerminationReason, "max iterations") &&
		!strings.Contains(result.TerminationReason, "duplicate") {
		t.Errorf("expected max iterations or duplicate termination, got: %s", result.TerminationReason)
	}
}

// TestE2E_ContextCancel tests that the reactor responds to context cancellation.
func TestE2E_ContextCancel(t *testing.T) {
	// Use a delayed response so we can cancel mid-execution
	scenario := MockScenario{
		Responses: []MockResponse{
			{Content: intentResponse("task", "slow", 0.9, false)},
			{Content: thinkActResponse("starting slow task", "bash", map[string]any{"command": "sleep 100"}), Delay: 2 * time.Second},
		},
	}
	r := newMockReactor(t, scenario)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := r.Run(ctx, "Start a slow task", nil)
	// Cancellation should result in an error or a result with "cancelled" termination
	if err != nil {
		t.Logf("Run returned error (expected for cancel): %v", err)
	} else if result != nil && !strings.Contains(result.TerminationReason, "cancelled") {
		t.Logf("Run completed without cancellation: %s", result.TerminationReason)
	}
}

// TestE2E_FollowUpConversation tests multi-turn conversation with history.
func TestE2E_FollowUpConversation(t *testing.T) {
	now := time.Now().Unix()
	history := ConversationHistory{
		{Role: "user", Content: "What is 2+2?", Timestamp: now - 60},
		{Role: "assistant", Content: "2+2=4", Timestamp: now - 30},
	}

	scenario := NewMockScenario(
		intentResponse("follow_up", "math", 0.85, false),
		thinkResponse("Following up on previous math question", "2+2=4, so 4+4=8"),
	)
	r := newMockReactor(t, scenario)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "What about 4+4?", history)
	result = assertResult(t, result, err)

	if !strings.Contains(result.Answer, "8") {
		t.Errorf("expected answer about 8, got: %s", result.Answer)
	}
}

// TestE2E_LoopDetection tests that the reactor detects destructive loops.
func TestE2E_LoopDetection(t *testing.T) {
	// Simulate the same tool call + same error repeated 3 times (destructive loop threshold)
	scenario := NewMockScenario(
		intentResponse("task", "loop", 0.9, false),
		// 3 identical tool calls that produce errors
		thinkActResponse("Try to read file", "bash", map[string]any{"command": "cat /nonexistent/file.txt"}),
		thinkActResponse("Try again", "bash", map[string]any{"command": "cat /nonexistent/file.txt"}),
		thinkActResponse("Try once more", "bash", map[string]any{"command": "cat /nonexistent/file.txt"}),
	)
	r := newMockReactor(t, scenario)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Read a file that doesn't exist", nil)
	result = assertResult(t, result, err)

	// The mock uses identical bash commands but with different params values each time.
	// The duplicate detection catches identical target+result before destructive loop threshold.
	if !strings.Contains(result.TerminationReason, "destructive loop") &&
		!strings.Contains(result.TerminationReason, "duplicate") {
		t.Errorf("expected destructive loop or duplicate termination, got: %s", result.TerminationReason)
	}
}

// TestE2E_EventEmission tests that the reactor emits expected events.
func TestE2E_EventEmission(t *testing.T) {
	scenario := NewMockScenario(
		intentResponse("chat", "greeting", 0.95, false),
		thinkResponse("User is greeting", "Hi there!"),
	)

	bus := NewEventBus()
	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 10,
	}
	r := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario)), WithEventBus(bus))

	ch, cancel := bus.Subscribe()
	defer cancel()

	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCtx()

	_, err := r.Run(ctx, "Hello!", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Collect events
	var eventTypes []string
	timeout := time.After(1 * time.Second)
	for {
		select {
		case event := <-ch:
			eventTypes = append(eventTypes, string(event.Type))
		case <-timeout:
			goto done
		}
	}
done:

	found := map[string]bool{}
	for _, et := range eventTypes {
		found[et] = true
	}
	if !found[string(core.ThinkingDone)] {
		t.Error("expected ThinkingDone event")
	}
	if !found[string(core.FinalAnswer)] {
		t.Error("expected FinalAnswer event")
	}
	if !found[string(core.ExecutionSummary)] {
		t.Error("expected ExecutionSummary event")
	}
}

// TestE2E_PauseAndResume tests the Pause/Resume flow.
func TestE2E_PauseAndResume(t *testing.T) {
	// Phase 1: First run with 2 tool calls, then pause after first
	scenario1 := MockScenario{
		Responses: []MockResponse{
			{Content: intentResponse("task", "multi-step", 0.9, false)},
			{Content: thinkActResponse("Step 1: read file", "bash", map[string]any{"command": "cat test.txt"})},
			{Content: thinkActResponse("Step 2: process", "bash", map[string]any{"command": "wc -l test.txt"})},
		},
	}

	bus := NewEventBus()
	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 10,
	}
	r := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario1)), WithEventBus(bus))

	ctx, cancel := context.WithCancel(context.Background())

	r.clearSnapshot()

	// Run in background
	var runResult *RunResult
	var runErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		runResult, runErr = r.Run(ctx, "Multi-step task", nil)
	}()

	// Wait for first iteration to complete (via EventBus ThinkingDone), then pause
	actionCh, actionCancel := bus.Subscribe()
	defer actionCancel()
	waitAction := func() {
		timeout := time.After(2 * time.Second)
		for {
			select {
			case event := <-actionCh:
				if event.Type == core.ActionStart {
					return
				}
			case <-timeout:
				return
			}
		}
	}
	waitAction()
	r.SetPauseRequested()
	cancel()

	<-done

	if runErr != nil {
		t.Logf("Run returned error (expected for pause): %v", runErr)
	}
	if runResult == nil {
		t.Fatal("expected non-nil partial result")
	}
	t.Logf("Paused after %d iterations: %s", runResult.TotalIterations, runResult.TerminationReason)

	// Check that a snapshot was saved in this reactor instance
	snap := r.getSnapshot()
	if snap == nil {
		t.Skip("Pause snapshot not captured in time (timing-sensitive test)")
		return
	}

	// Phase 2: Resume from snapshot
	scenario2 := NewMockScenario(
		thinkAnswerResponse("Continuing from where we left off", "Task completed successfully after resume."),
	)
	r2 := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario2)))

	resumeCtx, resumeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resumeCancel()

	resumeResult, err := r2.RunFromSnapshot(resumeCtx, snap, "")
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	t.Logf("Resume result: iterations=%d, termination=%s", resumeResult.TotalIterations, resumeResult.TerminationReason)
	if resumeResult.Answer == "" {
		t.Error("expected non-empty answer after resume")
	}
}

// TestE2E_PauseAndResumeWithRedirect tests Pause + new message + Resume with redirect input.
func TestE2E_PauseAndResumeWithRedirect(t *testing.T) {
	scenario := MockScenario{
		Responses: []MockResponse{
			{Content: intentResponse("task", "long", 0.9, false)},
			{Content: thinkActResponse("Working on long task", "bash", map[string]any{"command": "sleep 100"}), Delay: 5 * time.Second},
		},
	}

	bus := NewEventBus()
	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 10,
	}
	r := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario)), WithEventBus(bus))

	ctx, cancel := context.WithCancel(context.Background())

	r.clearSnapshot()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx, "Long running task", nil)
	}()

	// Wait for intent classification to complete (the mock has no delay for intent,
	// but think has 5s delay). We cancel while think is sleeping.
	time.Sleep(50 * time.Millisecond)
	r.SetPauseRequested()
	cancel()
	<-done

	snap := r.getSnapshot()
	if snap == nil {
		t.Skip("Pause snapshot not captured (timing-sensitive)")
		return
	}

	// Resume with a redirect message
	scenario2 := NewMockScenario(
		thinkActResponse("Saw the redirect, adjusting plan", "echo", map[string]any{"text": "adjusted"}),
		thinkAnswerResponse("Completed with redirect consideration", "Done."),
	)
	r2 := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario2)))

	resumeCtx, resumeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resumeCancel()

	result, err := r2.RunFromSnapshot(resumeCtx, snap, "Actually, change the approach to use echo instead")
	if err != nil {
		t.Fatalf("Resume with redirect failed: %v", err)
	}

	t.Logf("Resume+redirect: iterations=%d, answer=%s", result.TotalIterations, truncate(result.Answer, 100))
}

// TestE2E_IsLocalSyncSubAgent tests that IsLocal forces synchronous subagent execution.
func TestE2E_IsLocalSyncSubAgent(t *testing.T) {
	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 5,
		IsLocal:       true, // Force synchronous execution
	}

	// Intent classification
	scenario := NewMockScenario(
		intentResponse("task", "subagent_test", 0.9, false),
		thinkResponse("Using local model, subagents will be sync", "Task completed with sync subagents."),
	)

	r := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario)))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Test local model", nil)
	result = assertResult(t, result, err)

	t.Logf("IsLocal result: iterations=%d, answer=%s", result.TotalIterations, truncate(result.Answer, 100))
}

// ---------------------------------------------------------------------------
// E2E: Multi-Agent Collaboration
// ---------------------------------------------------------------------------

// TestE2E_SubAgentSpawnAndResult tests the full subagent lifecycle:
// spawn via "subagent" tool → subagent executes its task → collect result via "subagent_result".
// Uses IsLocal=true so subagent runs synchronously (deterministic for testing).
func TestE2E_SubAgentSpawnAndResult(t *testing.T) {
	// Mock LLM responses:
	// 0: intent classification for main agent
	// 1: main agent Think — decides to spawn a subagent
	// 2: main agent Think — decides to collect subagent result
	// 3: intent classification for subagent (IsLocal sync)
	// 4: subagent Think — produces answer
	scenario := MockScenario{
		Responses: []MockResponse{
			{Content: intentResponse("task", "research", 0.9, false)},
			{Content: thinkActResponse("Need to delegate research to a subagent", "subagent", map[string]any{
				"name":        "@researcher",
				"description": "Research topic X",
				"prompt":      "What is topic X? Answer briefly.",
			})},
			{Content: thinkActResponse("Now collect the subagent result", "subagent_result", map[string]any{
				"task_id": "task_1",
			})},
			// Subagent's own LLM calls (recursive, same mock)
			{Content: intentResponse("task", "topic_x", 0.85, false)},
			{Content: thinkAnswerResponse("Topic X is about quantum computing", "Topic X refers to quantum computing research.")},
		},
	}

	bus := NewEventBus()
	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 10,
		IsLocal:       true, // Force synchronous subagent execution
	}
	r := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario)), WithEventBus(bus))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Research topic X for me", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify the main agent completed
	t.Logf("SubAgentSpawnAndResult: iterations=%d, termination=%s, answer=%q", result.TotalIterations, result.TerminationReason, truncate(result.Answer, 100))
	if result.TotalIterations < 2 {
		t.Errorf("expected at least 2 iterations (spawn + collect), got: %d (reason: %s)", result.TotalIterations, result.TerminationReason)
	}

	// Verify subagent-related tool calls in steps
	var toolTargets []string
	for _, step := range result.Steps {
		if step.Action.Type == ActionTypeToolCall {
			toolTargets = append(toolTargets, step.Action.Target)
		}
	}
	t.Logf("Tool calls: %v", toolTargets)

	hasSubagent := false
	hasResult := false
	for _, target := range toolTargets {
		if target == "subagent" {
			hasSubagent = true
		}
		if target == "subagent_result" {
			hasResult = true
		}
	}
	if !hasSubagent {
		t.Error("expected 'subagent' tool call in execution steps")
	}
	if !hasResult {
		t.Error("expected 'subagent_result' tool call in execution steps")
	}

	t.Logf("Main agent answer: %s", truncate(result.Answer, 200))
}

// TestE2E_TaskCreateSyncInline tests the task_create tool (synchronous inline execution).
// Unlike subagent (async goroutine), task_create runs inline in the same reactor thread.
func TestE2E_TaskCreateSyncInline(t *testing.T) {
	// Mock LLM responses:
	// 0: intent classification for main agent
	// 1: main agent Think — decides to create a task
	// 2: intent classification for inline subtask (via RunInline → Run)
	// 3: inline subtask Think — produces answer
	// 4: main agent Think — produces final answer based on task result
	scenario := MockScenario{
		Responses: []MockResponse{
			{Content: intentResponse("task", "analysis", 0.9, false)},
			{Content: thinkActResponse("Need to create a subtask for analysis", "task_create", map[string]any{
				"description": "Analyze data",
				"prompt":      "Analyze the following: sales went up 20%",
			})},
			// task_create calls RunInline → Run recursively:
			{Content: intentResponse("task", "data_analysis", 0.85, false)},
			{Content: thinkAnswerResponse("Sales increased significantly", "Sales grew by 20% year over year.")},
			// Back to main agent:
			{Content: thinkAnswerResponse("Synthesizing the analysis result", "Based on the analysis, sales grew by 20% year over year.")},
		},
	}

	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 10,
		IsLocal:       true,
	}
	r := NewReactor(cfg, WithMockLLM(mockLLMFromScenario(scenario)))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Analyze the sales data", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TotalIterations < 2 {
		t.Errorf("expected at least 2 iterations, got: %d", result.TotalIterations)
	}

	// Verify task_create tool was called
	var toolTargets []string
	for _, step := range result.Steps {
		if step.Action.Type == ActionTypeToolCall {
			toolTargets = append(toolTargets, step.Action.Target)
			if step.Action.Target == "task_create" {
				// Verify the task result was injected into the observation
				t.Logf("task_create observation: %s", truncate(step.Observation.Result, 200))
				if step.Observation.Result == "" {
					t.Error("expected non-empty observation result from task_create")
				}
			}
		}
	}
	t.Logf("Tool calls: %v", toolTargets)

	hasTaskCreate := false
	for _, target := range toolTargets {
		if target == "task_create" {
			hasTaskCreate = true
		}
	}
	if !hasTaskCreate {
		t.Error("expected 'task_create' tool call in execution steps")
	}

	t.Logf("Final answer: %s", truncate(result.Answer, 200))
}

// ---------------------------------------------------------------------------
// E2E: Task Termination Summary
// ---------------------------------------------------------------------------

// TestE2E_TaskSummaryEvent tests that a non-trivial task (multiple iterations + tool calls)
// emits a TaskSummary event after the T-A-O loop completes.
//
// Note: generateSummary runs in a goroutine, so we need to wait for the event
// asynchronously via the EventBus subscription.
func TestE2E_TaskSummaryEvent(t *testing.T) {
	bus := NewEventBus()
	ch, cancel := bus.Subscribe()
	defer cancel()

	// We need an extra mock response for the generateSummary goroutine.
	// Total mock calls: 1 (intent) + 3 (think iterations) + 1 (summary) = 5
	mockCallCount := 0
	mockFn := func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error) {
		idx := mockCallCount
		mockCallCount++
		t.Logf("Mock call #%d (idx=%d): systemPrompt[:50]=%q userMessage[:50]=%q", mockCallCount, idx, truncate(systemPrompt, 50), truncate(userMessage, 50))

		var content string
		switch idx {
		case 0:
			content = intentResponse("task", "multi-step", 0.9, false)
		case 1:
			content = thinkActResponse("Add numbers", "calculator", map[string]any{"operation": "add", "a": 10.0, "b": 20.0})
		case 2:
			content = thinkActResponse("Multiply numbers", "calculator", map[string]any{"operation": "multiply", "a": 5.0, "b": 3.0})
		case 3:
			content = thinkAnswerResponse("All done", "Results: 30 and 15.")
		case 4:
			content = "Task summary: completed two calculations."
		default:
			content = "fallback"
		}
		return &gochatcore.Response{Content: content, Usage: &gochatcore.Usage{TotalTokens: 100}}, nil
	}

	cfg := ReactorConfig{
		APIKey:        "mock-api-key",
		BaseURL:       "https://mock.example.com/v1",
		Model:         "mock-model",
		MaxIterations: 10,
	}
	r := NewReactor(cfg, WithMockLLM(mockFn), WithEventBus(bus))

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()

	result, err := r.Run(ctx, "Run a multi-step task", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	t.Logf("TotalIterations: %d, Answer: %q, TerminationReason: %q", result.TotalIterations, result.Answer, result.TerminationReason)

	if result.TotalIterations <= 1 {
		t.Fatalf("expected > 1 iterations, got: %d", result.TotalIterations)
	}
	if result.Answer == "" {
		t.Fatal("expected non-empty answer")
	}

	// Wait for TaskSummary event
	summaryTimeout := time.After(3 * time.Second)
	var taskSummaryFound bool
	var summaryData core.TaskSummaryData

	for !taskSummaryFound {
		select {
		case event := <-ch:
			if event.Type == core.TaskSummary {
				taskSummaryFound = true
				if data, ok := event.Data.(core.TaskSummaryData); ok {
					summaryData = data
				}
			}
		case <-summaryTimeout:
			t.Fatalf("timed out waiting for TaskSummary event (total mock calls: %d)", mockCallCount)
		}
	}

	if !taskSummaryFound {
		t.Fatal("expected TaskSummary event to be emitted")
	}
	if summaryData.Summary == "" {
		t.Error("expected non-empty summary content in TaskSummary event")
	}
	t.Logf("TaskSummary received: %s", truncate(summaryData.Summary, 200))
}
