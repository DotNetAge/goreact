package reactor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	gochat "github.com/DotNetAge/gochat"
	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/tools"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

func testConfig(t *testing.T) ReactorConfig {
	t.Helper()
	cfg := DefaultReactorConfig()
	cfg.APIKey = "DASHSCOPE_API_KEY="
	cfg.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	cfg.Model = "qwen3.5-flash"
	cfg.ClientType = gochat.QwenClient
	return cfg
}

// newTestReactor creates a reactor pre-loaded with all built-in tools.
func newTestReactor(t *testing.T) *defaultReactor {
	t.Helper()
	r := NewReactor(testConfig(t))

	allTools := []core.FuncTool{
		tools.NewEcho(),
		tools.NewCalculator(),
		tools.NewLS(),
		tools.NewRead(),
		tools.NewWrite(),
		tools.NewEdit(),
		tools.NewGrep(),
		tools.NewGlob(),
		tools.NewReplace(),
		tools.NewCron(),
		tools.NewBash(),
	}
	for _, tool := range allTools {
		if err := r.RegisterTool(tool); err != nil {
			t.Fatalf("RegisterTool failed: %v", err)
		}
	}
	return r
}

func logResult(t *testing.T, tag string, result *RunResult) {
	t.Helper()
	t.Logf("[%s] intent=%s confidence=%.2f iterations=%d termination=%s",
		tag, result.Intent.Type, result.Confidence, result.TotalIterations, result.TerminationReason)
	for _, s := range result.Steps {
		t.Logf("[%s]   step %d: thought.decision=%s action.type=%s action.target=%s observation.success=%v",
			tag, s.Iteration, s.Thought.Decision, s.Action.Type, s.Action.Target, s.Observation.Success)
	}
	t.Logf("[%s]   answer: %s", tag, truncate(result.Answer, 300))
	if result.ClarificationNeeded {
		t.Logf("[%s]   clarification: %s", tag, result.ClarificationQuestion)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestReactor_ChatIntent(t *testing.T) {
	r := newTestReactor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "你好，最近怎么样？", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	logResult(t, "Chat", result)

	if result.Answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if result.Intent.Type != "chat" {
		t.Errorf("expected intent=chat, got %q", result.Intent.Type)
	}
}

func TestReactor_TaskIntent_ToolCall(t *testing.T) {
	r := newTestReactor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "帮我算一下 123 乘以 456 等于多少", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	logResult(t, "Task", result)

	if result.Answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if result.Intent.Type != "task" {
		t.Errorf("expected intent=task, got %q", result.Intent.Type)
	}

	// At least one step should be a tool_call
	hasToolCall := false
	for _, s := range result.Steps {
		if s.Action.Type == ActionTypeToolCall {
			hasToolCall = true
			break
		}
	}
	if !hasToolCall {
		t.Error("expected at least one tool_call step")
	}
}

func TestReactor_ClarificationIntent(t *testing.T) {
	r := newTestReactor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "帮我发个邮件", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	logResult(t, "Clarify", result)

	if result.ClarificationNeeded || result.Intent.RequiresClarification {
		t.Logf("correctly triggered clarification")
	}
}

func TestReactor_FollowUpIntent(t *testing.T) {
	r := newTestReactor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	now := time.Now().Unix()
	history := ConversationHistory{
		{Role: "user", Content: "帮我算一下 25 乘以 4", Timestamp: now - 60},
		{Role: "assistant", Content: "25 乘以 4 等于 100", Timestamp: now - 30},
	}

	result, err := r.Run(ctx, "再加上 50 呢？", history)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	logResult(t, "FollowUp", result)

	if result.Answer == "" {
		t.Fatal("expected non-empty answer")
	}
}

func TestReactor_MultiTurnConversation(t *testing.T) {
	r := newTestReactor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Turn 1
	result1, err := r.Run(ctx, "帮我回显一下：hello reactor", nil)
	if err != nil {
		t.Fatalf("Turn 1 failed: %v", err)
	}
	logResult(t, "T1", result1)

	// Turn 2 with history
	now := time.Now().Unix()
	history := ConversationHistory{
		{Role: "user", Content: "帮我回显一下：hello reactor", Timestamp: now - 60},
		{Role: "assistant", Content: result1.Answer, Timestamp: now - 30},
	}
	result2, err := r.Run(ctx, "能再帮我算一下 200 加 300 吗", history)
	if err != nil {
		t.Fatalf("Turn 2 failed: %v", err)
	}
	logResult(t, "T2", result2)

	if result2.Answer == "" {
		t.Fatal("expected non-empty answer on turn 2")
	}
}

func TestReactor_Termination_MaxIterations(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxIterations = 3
	r := NewReactor(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "Hello", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	logResult(t, "MaxIter", result)

	if result.TotalIterations > 3 {
		t.Errorf("expected <= 3 iterations, got %d", result.TotalIterations)
	}
}

func TestReactor_Termination_ContextCancel(t *testing.T) {
	r := newTestReactor(t)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately after starting
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := r.Run(ctx, "A very long and complex question that might take time...", nil)
	if err != nil {
		t.Logf("Run returned error (expected for cancel): %v", err)
	}
}

func TestReactor_FullResultDump(t *testing.T) {
	r := newTestReactor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := r.Run(ctx, "列出当前目录的文件", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Full RunResult:\n%s", string(b))
}

// TestReactor_IntentRegistry_Dynamic tests dynamic intent registration.
func TestReactor_IntentRegistry_Dynamic(t *testing.T) {
	r := newTestReactor(t)

	err := r.RegisterIntent(IntentDefinition{
		Type:          "code_review",
		Description:   "User wants to review or analyze source code",
		DescriptionCN: "用户希望审查或分析源代码",
	})
	if err != nil {
		t.Fatalf("RegisterIntent failed: %v", err)
	}

	defs := r.IntentRegistry().All()
	found := false
	for _, d := range defs {
		if d.Type == "code_review" {
			found = true
			break
		}
	}
	if !found {
		t.Error("code_review intent not found in registry")
	}
}
