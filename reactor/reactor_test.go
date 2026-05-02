package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// ============================================================================
// Prompt System Tests (KV Cache, Dynamic Boundary, CloneForSkill)
// ============================================================================

func TestPrompt_ToSectionedMessages_StaticOrder(t *testing.T) {
	tests := []struct {
		name     string
		prompt   *Prompt
		wantStatic []string  // expected static section texts (before boundary)
		wantDynamic []string // expected dynamic section texts (after boundary)
		wantTotal int
	}{
		{
			name: "all fields filled",
			prompt: &Prompt{
				Identity:            "You are a test agent.",
				Rules:               "1. Be helpful.",
				ExecutionGuidelines: "Be cautious with writes.",
				SkillsCatalog:       "- skill_a",
				ToolUsage:           "Use tools wisely.",
				ThinkInstr:          "Decide act/answer.",
				AgentCoordination:   "Find and delegate to expert agents.",
				ToneAndStyle:        "Be concise.",
				SystemReminders:     "Remember context limits.",
				OutputEfficiency:    "Use prose.",
				Language:            "Always respond in English.",
				EnvironmentInfo:     "cwd: /tmp",
			},
			wantStatic: []string{
				"You are a test agent.",
				"## Behavioral Rules\n1. Be helpful.",
				"Be cautious with writes.",
				"- skill_a",
				"Use tools wisely.",
				"Decide act/answer.",
				"Find and delegate to expert agents.",
				"Be concise.",
				"Remember context limits.",
			},
			wantDynamic: []string{
				"Use prose.",
				"Always respond in English.",
				"cwd: /tmp",
			},
			wantTotal: 13, // 9 static + 1 boundary + 3 dynamic
		},
		{
			name: "only identity",
			prompt: &Prompt{
				Identity: "Minimal agent.",
			},
			wantStatic: []string{
				"Minimal agent.",
			},
			wantDynamic: nil,
			wantTotal:   2, // 1 static + 1 boundary
		},
		{
			name: "active skill appends to ThinkInstr",
			prompt: &Prompt{
				Identity:                "Agent.",
				ThinkInstr:              "Think carefully.",
				HasActiveSkill:          true,
				ActiveSkillName:         "debug",
				ActiveSkillDesc:         "Debug code",
				ActiveSkillInstructions: "1. Read 2. Fix",
				FilteredToolList:        "read, write",
				ResourceBasePath:        "/project",
			},
			wantStatic: []string{
				"Agent.",
				"Think carefully.\n\n<active_skill>\n=== SKILL: debug ===\nDescription: Debug code\n\n1. Read 2. Fix\n\nAvailable tools: read, write\nResource base path: /project\n</active_skill>",
			},
			wantDynamic: nil,
			wantTotal:   3, // 2 static + 1 boundary (Rules empty → skipped)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := tt.prompt.ToSectionedMessages()

			if len(msgs) != tt.wantTotal {
				t.Fatalf("len(messages) = %d, want %d", len(msgs), tt.wantTotal)
			}

			// Verify static sections
			for i, want := range tt.wantStatic {
				got := msgs[i].Content[0].Text
				if got != want {
					t.Errorf("static section [%d] content = %q, want %q", i, got, want)
				}
			}

			// Verify boundary
			boundaryIdx := len(tt.wantStatic)
			if msgs[boundaryIdx].Content[0].Text != DynamicBoundary {
				t.Errorf("message[%d] expected DynamicBoundary, got %q", boundaryIdx, msgs[boundaryIdx].Content[0].Text)
			}

			// Verify dynamic sections
			dynStart := boundaryIdx + 1
			for i, want := range tt.wantDynamic {
				got := msgs[dynStart+i].Content[0].Text
				if got != want {
					t.Errorf("dynamic section [%d] content = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestPrompt_ToSectionedMessages_EmptyFieldsSkipped(t *testing.T) {
	p := &Prompt{
		Identity: "You are a minimal agent.",
	}

	msgs := p.ToSectionedMessages()

	// Only Identity + DynamicBoundary should be present
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages (identity + boundary), got %d", len(msgs))
	}
}

func TestPrompt_CloneForSkill(t *testing.T) {
	original := &Prompt{
		Identity:       "You are an agent.",
		Rules:          "Be helpful.",
		ThinkInstr:     "Decide what to do.",
		HasActiveSkill: false,
	}

	cloned := original.CloneForSkill(
		"file-search",
		"Search files using glob and grep",
		"Follow these steps: 1. glob 2. grep",
		"glob, grep, read",
		"/workspace",
	)

	if cloned.HasActiveSkill != true {
		t.Error("CloneForSkill should set HasActiveSkill=true")
	}
	if cloned.ActiveSkillName != "file-search" {
		t.Errorf("expected skill name 'file-search', got '%s'", cloned.ActiveSkillName)
	}
	if cloned.ActiveSkillDesc != "Search files using glob and grep" {
		t.Errorf("unexpected skill desc: '%s'", cloned.ActiveSkillDesc)
	}

	// Original must not be modified
	if original.HasActiveSkill {
		t.Error("original prompt should not be modified by CloneForSkill")
	}
}

func TestPrompt_ToSectionedMessages_WithActiveSkill(t *testing.T) {
	p := &Prompt{
		Identity:                "You are an agent.",
		ThinkInstr:              "Decide act/answer.",
		HasActiveSkill:          true,
		ActiveSkillName:         "code-edit",
		ActiveSkillDesc:         "Edit code safely",
		ActiveSkillInstructions: "1. Read file 2. Apply changes 3. Verify",
		FilteredToolList:        "read, write, file_edit",
		ResourceBasePath:        "/project",
	}

	msgs := p.ToSectionedMessages()

	// Find the ThinkInstr message (should contain skill block)
	foundSkillBlock := false
	for _, m := range msgs {
		for _, block := range m.Content {
			if strings.Contains(block.Text, "<active_skill>") &&
				strings.Contains(block.Text, "code-edit") {
				foundSkillBlock = true
				break
			}
		}
	}
	if !foundSkillBlock {
		t.Fatal("expected <active_skill> block in ThinkInstr section")
	}
}

func TestPrompt_RenderToLLMInput(t *testing.T) {
	p := &Prompt{
		Identity:   "You are a test agent.",
		ThinkInstr: "Think step by step.",
	}

	input := p.RenderToLLMInput(
		"Hello world",
		ConversationHistory{
			{Role: "assistant", Content: "Hi!"},
		},
		[]gochatcore.Tool{},
	)

	if input.UserMessage != "Hello world" {
		t.Errorf("expected user message 'Hello world', got '%s'", input.UserMessage)
	}
	if len(input.History) != 1 {
		t.Errorf("expected 1 history message, got %d", len(input.History))
	}
	if len(input.SystemPromptSections) == 0 {
		t.Error("expected non-empty system prompt sections")
	}
}

// ============================================================================
// Reactor.Run() with MockLLM — Complete T-A-O Loop Tests
// ============================================================================

func newTestReactor(mockFn MockLLMFunc, opts ...ReactorOption) *Reactor {
	cfg := ReactorConfig{
		Model:         "test-model",
		MaxIterations: 10,
	}
	allOpts := []ReactorOption{
		WithMockLLM(mockFn),
		WithoutBundledSkills(),
	}
	allOpts = append(allOpts, opts...)
	return NewReactor(cfg, allOpts...)
}

func TestReactor_Run_MockLLM_AnswerImmediately(t *testing.T) {
	callCount := 0
	r := newTestReactor(func(ctx context.Context, input CallInput) (*gochatcore.Response, error) {
		callCount++
		return &gochatcore.Response{
			Content: `{"decision": "answer", "final_answer": "Hello, user!", "reasoning": "I can answer directly."}`,
		}, nil
	})

	result, err := r.Run(context.Background(), "Say hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 LLM call, got %d", callCount)
	}
	if result.TotalIterations != 1 {
		t.Errorf("expected 1 iteration, got %d", result.TotalIterations)
	}
	if result.Answer != "Hello, user!" {
		t.Errorf("expected answer 'Hello, user!', got '%s'", result.Answer)
	}
	if result.TerminationReason != "direct answer produced" {
		t.Errorf("expected termination 'direct answer produced', got '%s'", result.TerminationReason)
	}
}

func TestReactor_Run_MockLLM_ActThenAnswer(t *testing.T) {
	callCount := 0
	var secondCallInput CallInput
	r := newTestReactor(func(ctx context.Context, input CallInput) (*gochatcore.Response, error) {
		callCount++
		if callCount == 1 {
			// First call: decide to act (call a tool) using native tool calls
			return &gochatcore.Response{
				Message: gochatcore.Message{
					ToolCalls: []gochatcore.ToolCall{
						{
							Name:      "echo_tool",
							Arguments: `{"message": "hello"}`,
						},
					},
				},
			}, nil
		}
		// Capture second call input for history verification
		secondCallInput = input
		// Second call: answer after tool result
		return &gochatcore.Response{
			Content: `{"decision": "answer", "final_answer": "Done.", "reasoning": "Tool returned successfully."}`,
		}, nil
	}, WithExtraTools(&mockEchoTool{}))

	result, err := r.Run(context.Background(), "Run the tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", callCount)
	}
	if result.TotalIterations < 2 {
		t.Errorf("expected at least 2 iterations, got %d", result.TotalIterations)
	}
	if result.Answer != "Done." {
		t.Errorf("expected answer 'Done.', got '%s'", result.Answer)
	}
	if result.TerminationReason != "direct answer produced" {
		t.Errorf("expected termination 'direct answer produced', got '%s'", result.TerminationReason)
	}

	// Verify that the second LLM call received history containing the tool execution result.
	// persistStep adds assistant+tool messages to ConversationHistory after each cycle.
	if len(secondCallInput.History) == 0 {
		t.Error("expected second LLM call to have non-empty History (ConversationHistory)")
	}
	foundToolResult := false
	for _, msg := range secondCallInput.History {
		if strings.Contains(msg.Content, "Echo: hello") {
			foundToolResult = true
			break
		}
	}
	if !foundToolResult {
		t.Error("second LLM call History should contain tool execution result 'Echo: hello' from persistStep")
	}
}

func TestReactor_Run_MockLLM_ContextCancelled(t *testing.T) {
	r := newTestReactor(func(ctx context.Context, input CallInput) (*gochatcore.Response, error) {
		// This should not be called since context is pre-cancelled
		// (runLoop checks Cancelled before Think in each iteration)
		return &gochatcore.Response{
			Content: `{"decision": "answer", "final_answer": "should not reach"}`,
		}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := r.Run(ctx, "Do something slow", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TerminationReason != "request cancelled" {
		t.Errorf("expected termination reason 'request cancelled', got '%s'", result.TerminationReason)
	}
	if result.TotalIterations != 0 {
		t.Errorf("expected 0 iterations for pre-cancelled context, got %d", result.TotalIterations)
	}
}

// ============================================================================
// Think / Act / Observe Individual Phase Tests
// ============================================================================

func TestReactor_Think_ProducesThought(t *testing.T) {
	r := newTestReactor(func(ctx context.Context, input CallInput) (*gochatcore.Response, error) {
		return &gochatcore.Response{
			Content: `{"decision": "answer", "final_answer": "Done.", "reasoning": "Analyzing the request."}`,
		}, nil
	})

	ctx := NewReactContext(context.Background(), "Test input", nil, 10)

	tokens, err := r.Think(ctx)
	if err != nil {
		t.Fatalf("Think failed: %v", err)
	}
	if ctx.LastThought == nil {
		t.Fatal("expected LastThought to be set")
	}
	if ctx.LastThought.Decision != DecisionAnswer {
		t.Errorf("expected DecisionAnswer, got %s", ctx.LastThought.Decision)
	}
	if ctx.LastThought.FinalAnswer != "Done." {
		t.Errorf("expected FinalAnswer 'Done.', got '%s'", ctx.LastThought.FinalAnswer)
	}
	if tokens < 0 {
		t.Errorf("expected non-negative token count, got %d", tokens)
	}
}

func TestReactor_Think_NativeToolCalls(t *testing.T) {
	r := newTestReactor(func(ctx context.Context, input CallInput) (*gochatcore.Response, error) {
		return &gochatcore.Response{
			Message: gochatcore.Message{
				ToolCalls: []gochatcore.ToolCall{
					{Name: "read", Arguments: `{"path": "/tmp/test.txt"}`},
				},
			},
		}, nil
	}, WithExtraTools(&mockReadTool{}))

	ctx := NewReactContext(context.Background(), "Read a file", nil, 10)

	_, err := r.Think(ctx)
	if err != nil {
		t.Fatalf("Think failed: %v", err)
	}
	if ctx.LastThought == nil {
		t.Fatal("expected LastThought to be set")
	}
	if ctx.LastThought.Decision != DecisionAct {
		t.Errorf("expected DecisionAct, got %s", ctx.LastThought.Decision)
	}
	if len(ctx.LastThought.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(ctx.LastThought.ToolCalls))
	}
	if _, ok := ctx.LastThought.ToolCalls["read"]; !ok {
		t.Error("expected 'read' in ToolCalls")
	}
}

func TestReactor_Act_AnswerDecision(t *testing.T) {
	r := newTestReactor(nil) // No LLM needed for Act test
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.LastThought = &Thought{
		Decision:    DecisionAnswer,
		FinalAnswer: "The answer is 42.",
	}

	err := r.Act(ctx)
	if err != nil {
		t.Fatalf("Act failed: %v", err)
	}
	if ctx.LastAction == nil {
		t.Fatal("expected LastAction to be set")
	}
	if ctx.LastAction.Type != ActionTypeAnswer {
		t.Errorf("expected ActionTypeAnswer, got %s", ctx.LastAction.Type)
	}
	if ctx.LastAction.Result != "The answer is 42." {
		t.Errorf("expected result 'The answer is 42.', got '%s'", ctx.LastAction.Result)
	}
}

func TestReactor_Act_ClarifyDecision(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.LastThought = &Thought{
		Decision:              DecisionClarify,
		ClarificationQuestion: "What file should I read?",
	}

	err := r.Act(ctx)
	if err != nil {
		t.Fatalf("Act failed: %v", err)
	}
	if ctx.LastAction.Type != ActionTypeClarify {
		t.Errorf("expected ActionTypeClarify, got %s", ctx.LastAction.Type)
	}
}

func TestReactor_Act_NoThought(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)

	err := r.Act(ctx)
	if err == nil {
		t.Fatal("expected error when Act called without Thought")
	}
}

func TestReactor_Observe_ToolCallResult(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.LastAction = &Action{
		Type:   ActionTypeToolCall,
		Target: "read",
		Result: "file contents here",
	}

	err := r.Observe(ctx)
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if ctx.LastObservation == nil {
		t.Fatal("expected LastObservation to be set")
	}
	if ctx.LastObservation.Result != "file contents here" {
		t.Errorf("expected observation result 'file contents here', got '%s'", ctx.LastObservation.Result)
	}
}

func TestReactor_Observe_ToolCallError(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.LastAction = &Action{
		Type:     ActionTypeToolCall,
		Target:   "read",
		Error:    fmt.Errorf("file not found"),
		ErrorMsg: "file not found",
	}

	err := r.Observe(ctx)
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if ctx.LastObservation.Error == "" {
		t.Error("expected observation error to be set")
	}
}

func TestReactor_Observe_NoAction(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)

	err := r.Observe(ctx)
	if err == nil {
		t.Fatal("expected error when Observe called without Action")
	}
}

// ============================================================================
// Termination Detection Tests
// ============================================================================

func TestCheckTermination_MaxIterations(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 3)
	ctx.CurrentIteration = 3

	terminated, reason := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination at max iterations")
	}
	if reason != "reached max iterations" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestCheckTermination_ContextCancelled(t *testing.T) {
	r := newTestReactor(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reactCtx := NewReactContext(ctx, "Test", nil, 10)

	terminated, reason := r.CheckTermination(reactCtx)
	if !terminated {
		t.Error("expected termination on cancelled context")
	}
	if reason != "request cancelled" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestCheckTermination_FinalAnswer(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.LastThought = &Thought{
		Decision: DecisionAnswer,
		IsFinal:  true,
	}

	terminated, reason := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination on final answer")
	}
	if reason != "thinker produced final answer" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestCheckTermination_DirectAnswer(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.LastAction = &Action{Type: ActionTypeAnswer}

	terminated, reason := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination on direct answer")
	}
	if reason != "direct answer produced" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestCheckTermination_Clarification(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.LastAction = &Action{Type: ActionTypeClarify}

	terminated, reason := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination on clarification")
	}
	if reason != "clarification needed" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestCheckTermination_DestructiveLoop(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.History = []Step{
		{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /"}}, Observation: Observation{Error: "permission denied"}},
		{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /"}}, Observation: Observation{Error: "permission denied"}},
		{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /"}}, Observation: Observation{Error: "permission denied"}},
	}

	terminated, _ := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination on destructive loop")
	}
}

func TestCheckTermination_AgentStuck(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	// 4 consecutive answer iterations (no tool calls)
	for i := 0; i < 4; i++ {
		ctx.History = append(ctx.History, Step{
			Action: Action{Type: ActionTypeAnswer, Result: "stuck answer"},
		})
	}

	terminated, reason := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination when agent is stuck")
	}
	if reason != "agent stuck: no tool progress in recent iterations" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestCheckTermination_ResultConverged(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	// 3 identical action results
	for i := 0; i < 3; i++ {
		ctx.History = append(ctx.History, Step{
			Action: Action{Type: ActionTypeToolCall, Target: "read", Result: "same result"},
		})
	}

	terminated, reason := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination on result convergence")
	}
	if reason != "result converged" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestCheckTermination_DuplicateAction(t *testing.T) {
	r := newTestReactor(nil)
	ctx := NewReactContext(context.Background(), "Test", nil, 10)
	ctx.History = []Step{
		{Action: Action{Type: ActionTypeToolCall, Target: "read", Result: "same"}},
		{Action: Action{Type: ActionTypeToolCall, Target: "read", Result: "same"}},
	}

	terminated, reason := r.CheckTermination(ctx)
	if !terminated {
		t.Error("expected termination on duplicate action")
	}
	if reason != "duplicate action detected" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

// -- Negative cases: conditions that should NOT trigger termination --

func TestCheckTermination_DestructiveLoop_NotTriggered(t *testing.T) {
	t.Run("different params should not trigger", func(t *testing.T) {
		history := []Step{
			{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /"}}, Observation: Observation{Error: "permission denied"}},
			{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /tmp"}}, Observation: Observation{Error: "permission denied"}},
			{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /home"}}, Observation: Observation{Error: "permission denied"}},
		}
		if isDestructiveLoop(history) {
			t.Error("isDestructiveLoop should return false: different params per call")
		}
	})

	t.Run("fewer than 3 calls should not trigger", func(t *testing.T) {
		history := []Step{
			{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /"}}, Observation: Observation{Error: "denied"}},
			{Action: Action{Type: ActionTypeToolCall, Target: "bash", Params: map[string]any{"cmd": "rm -rf /"}}, Observation: Observation{Error: "denied"}},
		}
		if isDestructiveLoop(history) {
			t.Error("isDestructiveLoop should return false: only 2 calls")
		}
	})

	t.Run("answer actions should not trigger", func(t *testing.T) {
		history := []Step{
			{Action: Action{Type: ActionTypeAnswer, Result: "ok"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "ok"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "ok"}},
		}
		if isDestructiveLoop(history) {
			t.Error("isDestructiveLoop should return false: no tool calls")
		}
	})
}

func TestCheckTermination_AgentStuck_NotTriggered(t *testing.T) {
	t.Run("3 answer actions (not enough)", func(t *testing.T) {
		history := []Step{
			{Action: Action{Type: ActionTypeAnswer, Result: "stuck"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "stuck"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "stuck"}},
		}
		if isAgentStuck(history) {
			t.Error("isAgentStuck should return false: only 3 answers, need 4")
		}
	})

	t.Run("a recent tool call among last 4 should not trigger", func(t *testing.T) {
		history := []Step{
			{Action: Action{Type: ActionTypeToolCall, Target: "read"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "ok"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "ok"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "ok"}},
		}
		if isAgentStuck(history) {
			t.Error("isAgentStuck should return false: the first entry of the window is a tool call")
		}
	})
}

func TestCheckTermination_ResultConverged_NotTriggered(t *testing.T) {
	t.Run("empty results should not trigger", func(t *testing.T) {
		history := []Step{
			{Action: Action{Type: ActionTypeToolCall, Target: "read", Result: ""}},
			{Action: Action{Type: ActionTypeToolCall, Target: "grep", Result: ""}},
			{Action: Action{Type: ActionTypeToolCall, Target: "write", Result: ""}},
		}
		if isResultConverged(history) {
			t.Error("isResultConverged should return false: empty results are skipped by guard")
		}
	})

	t.Run("only 2 identical results should not trigger", func(t *testing.T) {
		history := []Step{
			{Action: Action{Type: ActionTypeToolCall, Target: "read", Result: "same"}},
			{Action: Action{Type: ActionTypeToolCall, Target: "read", Result: "same"}},
		}
		if isResultConverged(history) {
			t.Error("isResultConverged should return false: need at least 3 steps")
		}
	})
}

func TestCheckTermination_DuplicateAction_NotTriggered(t *testing.T) {
	r := newTestReactor(nil)

	t.Run("different targets should not trigger", func(t *testing.T) {
		ctx := NewReactContext(context.Background(), "Test", nil, 10)
		ctx.History = []Step{
			{Action: Action{Type: ActionTypeToolCall, Target: "read", Result: "abc"}},
			{Action: Action{Type: ActionTypeToolCall, Target: "write", Result: "abc"}},
		}
		terminated, _ := r.CheckTermination(ctx)
		if terminated {
			t.Error("should NOT terminate: different tool targets")
		}
	})

	t.Run("answer actions should not trigger", func(t *testing.T) {
		ctx := NewReactContext(context.Background(), "Test", nil, 10)
		ctx.History = []Step{
			{Action: Action{Type: ActionTypeAnswer, Result: "hello"}},
			{Action: Action{Type: ActionTypeAnswer, Result: "hello"}},
		}
		terminated, _ := r.CheckTermination(ctx)
		if terminated {
			t.Error("should NOT terminate: Answer actions are not tool calls")
		}
	})
}

// ============================================================================
// ParseThinkResponse — JSON Format Noise Tests
// ============================================================================

func TestParseThinkResponse_JSONFormats(t *testing.T) {
	t.Run("plain JSON", func(t *testing.T) {
		thought, err := ParseThinkResponse(`{"decision": "answer", "final_answer": "done", "reasoning": "ok"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if thought.Decision != DecisionAnswer {
			t.Errorf("expected DecisionAnswer, got %s", thought.Decision)
		}
		if thought.FinalAnswer != "done" {
			t.Errorf("expected 'done', got '%s'", thought.FinalAnswer)
		}
		if thought.Reasoning != "ok" {
			t.Errorf("expected 'ok', got '%s'", thought.Reasoning)
		}
	})

	t.Run("JSON in code fence", func(t *testing.T) {
		input := "```json\n{\"decision\": \"answer\", \"final_answer\": \"fenced\", \"reasoning\": \"fence\"}\n```"
		thought, err := ParseThinkResponse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if thought.Decision != DecisionAnswer {
			t.Errorf("expected DecisionAnswer, got %s", thought.Decision)
		}
		if thought.FinalAnswer != "fenced" {
			t.Errorf("expected 'fenced', got '%s'", thought.FinalAnswer)
		}
	})

	t.Run("JSON in code fence without language tag", func(t *testing.T) {
		input := "```\n{\"decision\": \"answer\", \"final_answer\": \"bare\", \"reasoning\": \"bare\"}\n```"
		thought, err := ParseThinkResponse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if thought.FinalAnswer != "bare" {
			t.Errorf("expected 'bare', got '%s'", thought.FinalAnswer)
		}
	})

	t.Run("mixed case decision is normalized to lowercase", func(t *testing.T) {
		thought, err := ParseThinkResponse(`{"decision": "Answer", "final_answer": "mixed", "reasoning": "case"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if thought.Decision != DecisionAnswer {
			t.Errorf("expected normalized DecisionAnswer, got %s", thought.Decision)
		}
	})

	t.Run("unknown decision defaults to answer", func(t *testing.T) {
		thought, err := ParseThinkResponse(`{"decision": "fly", "final_answer": "defaulted", "reasoning": "unknown"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if thought.Decision != DecisionAnswer {
			t.Errorf("expected fallback DecisionAnswer, got %s", thought.Decision)
		}
		if thought.FinalAnswer != "defaulted" {
			t.Errorf("expected 'defaulted', got '%s'", thought.FinalAnswer)
		}
	})

	t.Run("missing fields get zero values", func(t *testing.T) {
		thought, err := ParseThinkResponse(`{}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if thought.Decision != DecisionAnswer {
			t.Errorf("expected fallback DecisionAnswer for empty JSON, got %s", thought.Decision)
		}
		if thought.Timestamp.IsZero() {
			t.Error("expected Timestamp to be set when missing")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := ParseThinkResponse(`{invalid json`)
		if err == nil {
			t.Error("expected parse error for invalid JSON")
		}
	})
}

// ============================================================================
// Snapshot — JSON Serialization Roundtrip
// ============================================================================

func TestSnapshot_JSONRoundtrip(t *testing.T) {
	original := NewReactContext(context.Background(), "json input", nil, 10)
	original.SessionID = "json-123"
	original.TaskID = "task-789"
	original.CurrentIteration = 2
	original.LastThought = &Thought{Decision: DecisionAct, Reasoning: "need info", FinalAnswer: ""}
	original.LastAction = &Action{Type: ActionTypeToolCall, Target: "grep", Result: "line 42"}
	original.LastObservation = &Observation{Result: "line 42"}
	original.History = []Step{
		{Iteration: 1, Thought: Thought{Decision: DecisionAct, Reasoning: "first"}, Action: Action{Type: ActionTypeToolCall, Target: "read", Result: "data"}, Observation: Observation{Result: "data"}},
	}

	snap := original.ToSnapshot()

	// Marshal to JSON
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("Marshal snapshot: %v", err)
	}

	// Unmarshal back
	var restoredSnap RunSnapshot
	if err := json.Unmarshal(data, &restoredSnap); err != nil {
		t.Fatalf("Unmarshal snapshot: %v", err)
	}

	// Verify fields
	if restoredSnap.SessionID != "json-123" {
		t.Errorf("SessionID = %q, want %q", restoredSnap.SessionID, "json-123")
	}
	if restoredSnap.Input != "json input" {
		t.Errorf("Input = %q, want %q", restoredSnap.Input, "json input")
	}
	if restoredSnap.CurrentIteration != 2 {
		t.Errorf("CurrentIteration = %d, want 2", restoredSnap.CurrentIteration)
	}
	if len(restoredSnap.History) != 1 {
		t.Errorf("len(History) = %d, want 1", len(restoredSnap.History))
	}
	if restoredSnap.LastThought == nil || restoredSnap.LastThought.Decision != DecisionAct {
		t.Error("LastThought.Decision should be 'act' after roundtrip")
	}
	if restoredSnap.LastAction == nil || restoredSnap.LastAction.Target != "grep" {
		t.Error("LastAction.Target should be 'grep' after roundtrip")
	}
	if restoredSnap.LastObservation == nil || restoredSnap.LastObservation.Result != "line 42" {
		t.Error("LastObservation.Result should be 'line 42' after roundtrip")
	}
	if restoredSnap.PausedAt.IsZero() {
		t.Error("PausedAt should be set by ToSnapshot")
	}

	// Restore context from the deserialized snapshot
	restoredCtx := NewReactContextFromSnapshot(context.Background(), &restoredSnap)
	if restoredCtx.SessionID != "json-123" {
		t.Errorf("restoredCtx.SessionID = %q", restoredCtx.SessionID)
	}
	if restoredCtx.CurrentIteration != 2 {
		t.Errorf("restoredCtx.CurrentIteration = %d, want 2", restoredCtx.CurrentIteration)
	}

	// Verify the restored context can be used for execution
	if restoredCtx.Ctx().Err() != nil {
		t.Error("restored context should have a fresh, non-cancelled context")
	}
}

// ============================================================================
// Snapshot / Pause-Resume Tests
// ============================================================================

func TestSnapshot_Roundtrip(t *testing.T) {
	original := NewReactContext(context.Background(), "original input", nil, 10)
	original.SessionID = "session-123"
	original.TaskID = "task-456"
	original.CurrentIteration = 3
	original.LastThought = &Thought{Decision: DecisionAct, Reasoning: "need tool"}
	original.LastAction = &Action{Type: ActionTypeToolCall, Target: "read"}
	original.LastObservation = &Observation{Result: "file content"}
	original.History = []Step{
		{Iteration: 1, Thought: Thought{Decision: DecisionAct}, Action: Action{Type: ActionTypeToolCall}},
		{Iteration: 2, Thought: Thought{Decision: DecisionAct}, Action: Action{Type: ActionTypeToolCall}},
	}

	snap := original.ToSnapshot()

	restored := NewReactContextFromSnapshot(context.Background(), snap)

	if restored.SessionID != "session-123" {
		t.Errorf("expected session-id 'session-123', got '%s'", restored.SessionID)
	}
	if restored.Input != "original input" {
		t.Errorf("expected input 'original input', got '%s'", restored.Input)
	}
	if restored.CurrentIteration != 3 {
		t.Errorf("expected iteration 3, got %d", restored.CurrentIteration)
	}
	if len(restored.History) != 2 {
		t.Errorf("expected 2 history steps, got %d", len(restored.History))
	}
	if restored.LastThought.Decision != DecisionAct {
		t.Errorf("expected DecisionAct, got %s", restored.LastThought.Decision)
	}
}

func TestReactor_RunFromSnapshot(t *testing.T) {
	callCount := 0
	r := newTestReactor(func(ctx context.Context, input CallInput) (*gochatcore.Response, error) {
		callCount++
		return &gochatcore.Response{
			Content: "Thought: Resumed execution.\nDecision: answer\nFinalAnswer: Resumed and done.",
		}, nil
	})

	// First create a snapshot
	ctx := NewReactContext(context.Background(), "initial task", nil, 10)
	ctx.CurrentIteration = 1
	snap := ctx.ToSnapshot()

	result, err := r.RunFromSnapshot(context.Background(), snap, "new input after resume")
	if err != nil {
		t.Fatalf("RunFromSnapshot failed: %v", err)
	}
	if result.Answer == "" {
		t.Error("expected non-empty answer from resumed run")
	}
}

func TestReactor_PauseAndTakeSnapshot(t *testing.T) {
	var r *Reactor
	r = newTestReactor(func(ctx context.Context, input CallInput) (*gochatcore.Response, error) {
		// Request pause on first call
		r.SetPauseRequested()
		return &gochatcore.Response{
			Content: "Thought: Pausing.\nDecision: answer\nFinalAnswer: paused state",
		}, nil
	})

	_, err := r.Run(context.Background(), "Pause after this", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	snap := r.TakeSnapshot()
	if snap == nil {
		t.Fatal("expected non-nil snapshot after pause")
	}
	if snap.TerminationReason != "paused" {
		t.Errorf("expected termination reason 'paused', got '%s'", snap.TerminationReason)
	}
}

// ============================================================================
// CloneReactor Tests (Child Agent Inheritance and Isolation)
// ============================================================================

func TestCloneReactor_InheritsToolRegistry(t *testing.T) {
	parent := newTestReactor(nil)
	parent.RegisterTool(&mockEchoTool{})

	childReactor := parent.CloneReactor(ReactorConfig{})

	// Child should have access to parent's tools
	tools := childReactor.ToolRegistry().All()
	found := false
	for _, tool := range tools {
		if tool.Info().Name == "echo_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("child reactor should inherit parent's tool registry")
	}
}

func TestCloneReactor_IndependentConfig(t *testing.T) {
	parent := newTestReactor(nil)

	childReactor := parent.CloneReactor(ReactorConfig{
		Model:         "child-model",
		Temperature:   0.5,
		SystemPrompt:  "child system prompt",
		MaxIterations: 5,
	})

	if childReactor.config.Model != "child-model" {
		t.Errorf("expected child model 'child-model', got '%s'", childReactor.config.Model)
	}
	if childReactor.config.Temperature != 0.5 {
		t.Errorf("expected child temperature 0.5, got %f", childReactor.config.Temperature)
	}
	if childReactor.config.SystemPrompt != "child system prompt" {
		t.Errorf("expected child system prompt, got '%s'", childReactor.config.SystemPrompt)
	}
}

func TestCloneReactor_ParentPromptNotLeaked(t *testing.T) {
	parent := newTestReactor(nil)
	parent.config.SystemPrompt = "parent identity"

	// Clone without explicit system prompt
	childReactor := parent.CloneReactor(ReactorConfig{})

	// Child should NOT inherit parent's system prompt (security)
	if childReactor.config.SystemPrompt != "" {
		t.Error("child reactor should not inherit parent's system prompt when not explicitly set")
	}
}

func TestCloneReactor_IndependentContextWindow(t *testing.T) {
	parent := newTestReactor(nil)

	childReactor := parent.CloneReactor(ReactorConfig{})

	// Both start with nil context window (not initialized until first LLM call).
	// The important property is that they are independently settable.
	if parent.ContextWindow() != nil || childReactor.ContextWindow() != nil {
		// If both are nil, they are independent (not shared)
	}

	// Set different context windows on each
	parentCw := &core.ContextWindow{}
	childCw := &core.ContextWindow{}
	parent.SetContextWindow(parentCw)
	childReactor.SetContextWindow(childCw)

	if parent.ContextWindow() != parentCw {
		t.Error("parent context window not set correctly")
	}
	if childReactor.ContextWindow() != childCw {
		t.Error("child context window not set correctly")
	}
	if parent.ContextWindow() == childReactor.ContextWindow() {
		t.Error("parent and child should have independent context windows")
	}
}

func TestCloneReactor_SharesMemoryAndEventBus(t *testing.T) {
	memory := &mockMemoryImpl{}
	bus := NewEventBus()

	cfg := ReactorConfig{Model: "test", MaxIterations: 10}
	parentReactor := NewReactor(cfg, WithMockLLM(nil), WithoutBundledSkills(), WithMemory(memory), WithEventBus(bus))

	childReactor := parentReactor.CloneReactor(ReactorConfig{})

	if childReactor.Memory() != memory {
		t.Error("child reactor should share parent's memory")
	}
	if childReactor.EventBus() != bus {
		t.Error("child reactor should share parent's event bus")
	}
}

// ============================================================================
// Mock Tool Implementations for Testing
// ============================================================================

type mockEchoTool struct{}

func (t *mockEchoTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "echo_tool",
		Description: "Echo a message back",
		Parameters: []core.Parameter{
			{Name: "message", Type: "string", Required: true, Description: "Message to echo"},
		},
	}
}

func (t *mockEchoTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	msg := ""
	if m, ok := params["message"].(string); ok {
		msg = m
	}
	return fmt.Sprintf("Echo: %s", msg), nil
}

type mockReadTool struct{}

func (t *mockReadTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "read",
		Description: "Read a file",
		Parameters: []core.Parameter{
			{Name: "path", Type: "string", Required: true, Description: "File path"},
		},
	}
}

func (t *mockReadTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	path := ""
	if p, ok := params["path"].(string); ok {
		path = p
	}
	return fmt.Sprintf("contents of %s", path), nil
}

type mockMemoryImpl struct{}

func (m *mockMemoryImpl) Retrieve(ctx context.Context, query string, opts ...core.RetrieveOption) ([]core.MemoryRecord, error) {
	return nil, nil
}
func (m *mockMemoryImpl) Store(ctx context.Context, record core.MemoryRecord) (string, error) {
	return "", nil
}
func (m *mockMemoryImpl) Delete(ctx context.Context, id string) error {
	return nil
}
