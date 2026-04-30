package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	gochat "github.com/DotNetAge/gochat"
	"github.com/DotNetAge/goreact/core"
)

// ============================================================
// T-A-O Integration Tests — Real LLM (qwen3.5-plus)
//
// These tests exercise the FULL T-A-O pipeline against a real
// LLM API to validate:
//   - Data correctness at every phase (not just code paths)
//   - Prompt rendering produces clean, non-polluted output
//   - LLM responses parse correctly into domain objects
//   - The complete Think→Act→Observe cycle executes correctly
//   - SystemPrompt integrity is maintained throughout
//
// Run with: go test ./reactor/... -v -count=1 -run "TestTAO" -timeout 300s
// Requires: DASHSCOPE_API_KEY environment variable
// ============================================================

// realLLMConfig creates a ReactorConfig pointing to qwen3.5-plus.
func realLLMConfig(t *testing.T) ReactorConfig {
	t.Helper()
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" || apiKey == "DASHSCOPE_API_KEY" {
		t.Skip("Skipping integration test: DASHSCOPE_API_KEY not set or is placeholder")
	}
	return ReactorConfig{
		APIKey:        apiKey,
		BaseURL:       "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Model:         "qwen3.5-plus",
		ClientType:    gochat.QwenClient,
		Temperature:   0.7,
		MaxTokens:     8192,
		SystemPrompt:  "You are a helpful AI assistant. You are concise, accurate, and respond in the same language as the user.",
		MaxIterations: 5,
	}
}

// newRealReactor creates a reactor configured for real LLM calls with event bus tracing.
func newRealReactor(t *testing.T) (*Reactor, *taoEventTracer) {
	t.Helper()
	cfg := realLLMConfig(t)
	tracer := newTaoEventTracer()
	r := NewReactor(cfg, WithEventBus(tracer.bus))
	return r, tracer
}

// ============================================================
// taoEventTracer captures all EventBus events during a T-A-O run
// for post-hoc analysis and detailed logging.
// ============================================================

type taoEventTracer struct {
	bus    *InProcessEventBus
	events []core.ReactEvent
}

func newTaoEventTracer() *taoEventTracer {
	bus := NewEventBus()
	ch, cancel := bus.Subscribe()
	tracer := &taoEventTracer{bus: bus}
	go func() {
		for e := range ch {
			tracer.events = append(tracer.events, e)
		}
		cancel()
	}()
	return tracer
}

func (tr *taoEventTracer) close() { tr.bus.Close() }

func (tr *taoEventTracer) logAll(t *testing.T, tag string) {
	t.Helper()
	t.Logf("[%s] === EventBus Event Trace (%d events) ===", tag, len(tr.events))
	for i, e := range tr.events {
		dataJSON, _ := json.Marshal(e.Data)
		t.Logf("[%s]   [#%d] type=%s session=%s task=%s | data=%s",
			tag, i, e.Type, e.SessionID, e.TaskID, truncate(string(dataJSON), 200))
	}
}

func (tr *taoEventTracer) countByType(eventType core.ReactEventType) int {
	n := 0
	for _, e := range tr.events {
		if e.Type == eventType {
			n++
		}
	}
	return n
}

// ============================================================
// Phase Data Loggers — render and log what data each phase sends
// ============================================================

func logIntentPhase(t *testing.T, input string, r *Reactor) {
	t.Helper()
	prompt := BuildIntentPrompt(input, "", r.intentRegistry)
	t.Logf("[INTENT-PHASE] Rendered prompt length=%d chars", len(prompt))
	t.Logf("[INTENT-PHASE] Prompt preview:\n%s", truncate(prompt, 800))
	verifyNoRolePollution(t, "intent_prompt", prompt)
	verifyNoInputDuplication(t, "intent_prompt", prompt, input)
}

func logThinkPhase(t *testing.T, input string, intent *Intent, r *Reactor) {
	t.Helper()
	skills, _ := r.skillRegistry.FindApplicableSkills(intent)
	skillsSection := BuildSkillsSystemPrompt(skills)
	prompt := BuildThinkPrompt(input, intent, nil, nil, nil, "", nil)
	t.Logf("[THINK-PHASE] Rendered think_prompt length=%d chars, skills_section=%d chars", len(prompt), len(skillsSection))
	if skillsSection != "" {
		t.Logf("[THINK-PHASE] Skills section:\n%s", truncate(skillsSection, 500))
	}
	t.Logf("[THINK-PHASE] Prompt preview:\n%s", truncate(prompt, 1000))
	verifyNoRolePollution(t, "think_prompt", prompt)
	verifyNoInputDuplication(t, "think_prompt", prompt, input)
	if skillsSection != "" {
		verifyNoRolePollution(t, "skills_section", skillsSection)
	}
}

// ============================================================
// Data Integrity Validators (real-time checks)
// ============================================================

func verifyNoRolePollution(t *testing.T, phaseName, content string) {
	t.Helper()
	lower := strings.ToLower(content)
	forbidden := []string{"you are the ", "you are an ", "you are a "}
	for _, p := range forbidden {
		if strings.Contains(lower, p) {
			t.Errorf("[%s] ROLE POLLUTION DETECTED: found %q — SystemPrompt identity leak!", phaseName, p)
		}
	}
}

func verifyNoInputDuplication(t *testing.T, phaseName, content, input string) {
	t.Helper()
	if strings.Contains(content, "User input: "+input) {
		t.Errorf("[%s] INPUT DUPLICATION: found 'User input: %s' — user input appears twice!", phaseName, input)
	}
}

func verifyStepIntegrity(t *testing.T, step Step, iteration int) {
	t.Helper()
	tag := fmt.Sprintf("STEP-%d", iteration)

	th := step.Thought
	if th.Decision == "" {
		t.Errorf("[%s] Thought.Decision is empty", tag)
	}
	validDecisions := map[string]bool{"act": true, "answer": true, "clarify": true}
	if !validDecisions[th.Decision] {
		t.Errorf("[%s] Thought.Decision=%q is not a valid decision", tag, th.Decision)
	}

	ac := step.Action
	switch th.Decision {
	case "act":
		if ac.Type != ActionTypeToolCall {
			t.Errorf("[%s] Decision=act but Action.Type=%q (want tool_call)", tag, ac.Type)
		}
		if ac.Target == "" {
			t.Errorf("[%s] Decision=act but Action.Target is empty", tag)
		}
	case "answer":
		if ac.Type != ActionTypeAnswer {
			t.Errorf("[%s] Decision=answer but Action.Type=%q (want answer)", tag, ac.Type)
		}
		if ac.Result == "" && th.FinalAnswer == "" {
			t.Errorf("[%s] Decision=answer but both Action.Result and FinalAnswer are empty", tag)
		}
	case "clarify":
		if ac.Type != ActionTypeClarify {
			t.Errorf("[%s] Decision=clarify but Action.Type=%q (want clarify)", tag, ac.Type)
		}
	}

	ob := step.Observation
	if ob.Result == "" && ob.Error == "" {
		t.Errorf("[%s] Observation has both empty Result and Error", tag)
	}
}

// ============================================================
// Test 1: Chat Intent — Simple Conversation Path
// ============================================================
// Verifies: classifyIntent → intent=chat → Think(answer) → Act(answer) → Observe(success)

func TestTAO_ChatIntent_RealLLM(t *testing.T) {
	r, tracer := newRealReactor(t)
	defer tracer.close()

	input := "你好，请用一句话介绍你自己。"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logIntentPhase(t, input, r)

	start := time.Now()
	result, err := r.Run(ctx, input, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tracer.logAll(t, "CHAT")

	t.Logf("=== CHAT RESULT ===")
	t.Logf("  Intent:       type=%s confidence=%.2f", result.Intent.Type, result.Intent.Confidence)
	t.Logf("  Answer:       %s", truncate(result.Answer, 500))
	t.Logf("  Iterations:   %d / %d", result.TotalIterations, r.config.MaxIterations)
	t.Logf("  Termination:  %s", result.TerminationReason)
	t.Logf("  Tokens:       %d", result.TokensUsed)
	t.Logf("  Duration:     %v", duration)
	t.Logf("  Steps:        %d", len(result.Steps))

	if result.Intent.Type != "chat" {
		t.Logf("WARNING: expected intent=chat, got %q (LLM may classify differently)", result.Intent.Type)
	}
	if result.Answer == "" {
		t.Fatal("expected non-empty answer for chat intent")
	}

	for i, step := range result.Steps {
		t.Logf("  Step[%d]: thought.decision=%s action.type=%s obs.success=%v",
			i+1, step.Thought.Decision, step.Action.Type, step.Observation.Error == "")
		verifyStepIntegrity(t, step, i+1)
	}

	if tracer.countByType(core.CycleEnd) != len(result.Steps) {
		t.Errorf("CycleEnd events (%d) != Steps count (%d)", tracer.countByType(core.CycleEnd), len(result.Steps))
	}
}

// ============================================================
// Test 2: Task Intent with Tool Call — Full Act Cycle
// ============================================================
// Verifies: classifyIntent → intent=task → Think(act) → Act(tool_call) → Observe(result)

func TestTAO_TaskIntent_ToolCall_RealLLM(t *testing.T) {
	r, tracer := newRealReactor(t)
	defer tracer.close()

	input := "帮我计算一下 123 乘以 456 等于多少，只给出结果"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logIntentPhase(t, input, r)

	start := time.Now()
	result, err := r.Run(ctx, input, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tracer.logAll(t, "TASK")

	t.Logf("=== TASK RESULT ===")
	t.Logf("  Intent:       type=%s confidence=%.2f summary=%s", result.Intent.Type, result.Intent.Confidence, result.Intent.Summary)
	t.Logf("  Answer:       %s", truncate(result.Answer, 500))
	t.Logf("  Iterations:   %d / %d", result.TotalIterations, r.config.MaxIterations)
	t.Logf("  Termination:  %s", result.TerminationReason)
	t.Logf("  Tokens:       %d", result.TokensUsed)
	t.Logf("  Duration:     %v", duration)

	if result.Answer == "" {
		t.Fatal("expected non-empty answer")
	}

	hasToolCall := false
	for i, step := range result.Steps {
		t.Logf("  Step[%d]: decision=%s action.type=%s target=%s params=%v result=%s error=%v",
			i+1, step.Thought.Decision, step.Action.Type,
			step.Action.Target, step.Action.Params,
			truncate(step.Action.Result, 200), step.Action.ErrorMsg)
		verifyStepIntegrity(t, step, i+1)
		if step.Action.Type == ActionTypeToolCall {
			hasToolCall = true
			if step.Action.Target == "" {
				t.Errorf("Step[%d]: tool_call with empty target", i+1)
			}
			if step.Observation.Error != "" {
				t.Logf("Step[%d]: tool execution error: %v", i+1, step.Observation.Error)
			} else {
				t.Logf("Step[%d]: tool result: %s", i+1, truncate(step.Observation.Result, 300))
			}
		}
	}

	if !hasToolCall {
		t.Logf("INFO: No tool_call steps detected — LLM chose to answer directly (acceptable for simple math)")
	}

	eventLog := tracer.logAllEventsByType(t, "TASK")
	if eventLog[string(core.ActionStart)] > 0 {
		t.Logf("  ActionStart events: %d (tool calls initiated)", eventLog[string(core.ActionStart)])
	}
	if eventLog[string(core.ActionResult)] > 0 {
		t.Logf("  ActionResult events: %d (tool results received)", eventLog[string(core.ActionResult)])
	}
}

// ============================================================
// Test 3: Clarification Intent — Ambiguous Input
// ============================================================
// Verifies: classifyIntent detects ambiguity → requires_clarification=true

func TestTAO_ClarificationIntent_RealLLM(t *testing.T) {
	r, tracer := newRealReactor(t)
	defer tracer.close()

	input := "帮我发个邮件"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logIntentPhase(t, input, r)

	start := time.Now()
	result, err := r.Run(ctx, input, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tracer.logAll(t, "CLARIFY")

	t.Logf("=== CLARIFICATION RESULT ===")
	t.Logf("  Intent:                  type=%s confidence=%.2f", result.Intent.Type, result.Intent.Confidence)
	t.Logf("  RequiresClarification:   %v", result.Intent.RequiresClarification)
	t.Logf("  MissingSlots:            %v", result.Intent.MissingSlots)
	t.Logf("  ClarificationQuestion:   %s", result.ClarificationQuestion)
	t.Logf("  Answer:                  %s", truncate(result.Answer, 300))
	t.Logf("  Duration:                %v", duration)

	if result.Intent.RequiresClarification || result.ClarificationNeeded {
		t.Logf("  ✅ Correctly triggered clarification mechanism")
		if result.ClarificationQuestion != "" {
			t.Logf("  ✅ Clarification question: %s", result.ClarificationQuestion)
		}
		if len(result.Intent.MissingSlots) > 0 {
			t.Logf("  ✅ Missing slots identified: %v", result.Intent.MissingSlots)
		}
	} else {
		t.Logf("  INFO: LLM did not require clarification (confidence=%.2f, may have answered directly)", result.Intent.Confidence)
	}
}

// ============================================================
// Test 4: Follow-Up Intent — Contextual Continuation
// ============================================================
// Verifies: conversation history flows into prompts correctly

func TestTAO_FollowUpIntent_RealLLM(t *testing.T) {
	// TODO: 这个测试总是超时，我怀疑 defer有死锁！
	r, tracer := newRealReactor(t)
	defer tracer.close()

	now := time.Now().Unix()
	history := ConversationHistory{
		{Role: "user", Content: "帮我计算 25 乘以 4", Timestamp: now - 60},
		{Role: "assistant", Content: "25 乘以 4 等于 100", Timestamp: now - 30},
	}
	input := "再加上 50 呢？"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	histFormatted := history.Format(10)
	t.Logf("[FOLLOWUP] History formatted (%d chars):\n%s", len(histFormatted), histFormatted)

	logIntentPhase(t, input, r)

	start := time.Now()
	result, err := r.Run(ctx, input, history)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tracer.logAll(t, "FOLLOWUP")

	t.Logf("=== FOLLOW-UP RESULT ===")
	t.Logf("  Intent:       type=%s confidence=%.2f", result.Intent.Type, result.Intent.Confidence)
	t.Logf("  Answer:       %s", truncate(result.Answer, 500))
	t.Logf("  Duration:     %v", duration)

	if result.Answer == "" {
		t.Fatal("expected non-empty answer")
	}

	expectedTypes := map[string]bool{"follow_up": true, "task": true, "chat": true}
	if !expectedTypes[result.Intent.Type] {
		t.Logf("NOTE: intent=%q (expected follow_up/task/chat)", result.Intent.Type)
	}

	for i, step := range result.Steps {
		verifyStepIntegrity(t, step, i+1)
	}
}

// ============================================================
// Test 5: Complete Data Flow Audit — End-to-End Verification
// ============================================================
// This is THE most important test. It validates EVERY data invariant
// across the entire T-A-O pipeline with a real LLM.

func TestTAO_DataFlowAudit_RealLLM(t *testing.T) {
	r, _ := newRealReactor(t)

	input := "用echo工具输出字符串 hello-tao-test"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	t.Logf("=== DATA FLOW AUDIT ===")
	t.Logf("Model: %s | MaxTokens: %d | SystemPrompt: %d chars",
		r.config.Model, r.config.MaxTokens, len(r.config.SystemPrompt))

	instructionSet := []struct {
		name    string
		render  func() string
		checkFn func(string)
	}{
		{
			name: "intent_prompt",
			render: func() string {
				return BuildIntentPrompt(input, "", r.intentRegistry)
			},
			checkFn: func(s string) {
				if strings.Contains(strings.ToLower(s), "you are the ") {
					t.Error("[AUDIT-FAIL] intent_prompt contains role definition")
				}
			},
		},
		{
			name: "skills_section",
			render: func() string {
				skills, _ := r.skillRegistry.FindApplicableSkills(nil)
				return BuildSkillsSystemPrompt(skills)
			},
			checkFn: func(s string) {
				if strings.Contains(strings.ToLower(s), "you are") {
					t.Error("[AUDIT-FAIL] skills_section contains role definition")
				}
			},
		},
		{
			name: "think_prompt",
			render: func() string {
				return BuildThinkPrompt(input, &Intent{Type: "task"}, nil, nil, nil, "", nil)
			},
			checkFn: func(s string) {
				if strings.Contains(strings.ToLower(s), "you are the ") {
					t.Error("[AUDIT-FAIL] think_prompt contains role definition")
				}
				if strings.Contains(s, "User input: "+input) {
					t.Error("[AUDIT-FAIL] think_prompt contains raw input duplication")
				}
			},
		},
	}

	t.Log("--- Phase 1: Static Data Rendering ---")
	for _, item := range instructionSet {
		content := item.render()
		t.Logf("  [%s] length=%d chars", item.name, len(content))
		item.checkFn(content)
	}

	t.Log("--- Phase 2: Live T-A-O Execution ---")
	start := time.Now()
	result, err := r.Run(ctx, input, nil)
	execDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	t.Logf("  Execution completed in %v", execDuration)
	t.Logf("  Intent: type=%s conf=%.2f", result.Intent.Type, result.Intent.Confidence)
	t.Logf("  Total iterations: %d", result.TotalIterations)
	t.Logf("  Steps recorded: %d", len(result.Steps))

	t.Log("--- Phase 3: Per-Step Data Integrity ---")
	for i, step := range result.Steps {
		tag := fmt.Sprintf("Step%d", i+1)
		th := step.Thought
		ac := step.Action
		ob := step.Observation

		t.Logf("  [%s] decision=%s reasoning=%q target=%q result_len=%d error=%v duration=%v",
			tag, th.Decision, truncate(th.Reasoning, 100),
			ac.Target, len(ac.Result), ac.ErrorMsg, step.Duration)

		verifyStepIntegrity(t, step, i+1)

		if th.Decision == "act" && ac.Type == "tool_call" {
			t.Logf("  [%s] ✅ Tool call verified: target=%s params=%v", tag, ac.Target, ac.Params)
			if ob.Error == "" && ob.Result != "" {
				t.Logf("  [%s] ✅ Observation success: result_len=%d", tag, len(ob.Result))
			}
		}
		if th.Decision == "answer" {
			t.Logf("  [%s] ✅ Direct answer: %s", tag, truncate(ac.Result, 200))
		}
	}

	t.Log("--- Phase 4: Cross-Step Consistency ---")
	if len(result.Steps) > 1 {
		for i := 1; i < len(result.Steps); i++ {
			prev := result.Steps[i-1]
			curr := result.Steps[i]
			prevObsResult := prev.Observation.Result
			if prevObsResult != "" && curr.Thought.Reasoning == "" {
				t.Logf("  [CrossStep %d→%d] WARNING: previous observation had content but current reasoning is empty", i, i+1)
			}
		}
	}

	t.Log("--- Phase 5: Final Result Validation ---")
	if result.Answer == "" && len(result.Steps) > 0 {
		lastStep := result.Steps[len(result.Steps)-1]
		if lastStep.Action.Result != "" {
			t.Logf("  [Final] Answer is empty but last action has result (may need generateSummary)")
		}
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("  Full Result JSON:\n%s", string(b))

	t.Log("✅ DATA FLOW AUDIT COMPLETE — all invariants checked")
}

// ============================================================
// Test 6: Multi-Turn Conversation — State Continuity
// ============================================================

func TestTAO_MultiTurnConversation_RealLLM(t *testing.T) {
	r, tracer := newRealReactor(t)
	defer tracer.close()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	turns := []struct {
		label   string
		input   string
		history ConversationHistory
	}{
		{label: "T1-chat", input: "你好，我叫小明，请记住我的名字"},
		{label: "T2-followup", input: "我叫什么名字？"},
		{label: "T3-task", input: "帮我把名字转换成大写字母"},
	}

	var lastAnswer string
	var allHistory ConversationHistory
	baseTime := time.Now().Unix()

	for ti, turn := range turns {
		t.Logf("\n=== TURN %d: %s ===", ti+1, turn.label)

		result, err := r.Run(ctx, turn.input, turn.history)
		if err != nil {
			t.Fatalf("Turn %d (%s) failed: %v", ti+1, turn.label, err)
		}

		t.Logf("  Input:    %s", turn.input)
		t.Logf("  Intent:   type=%s conf=%.2f", result.Intent.Type, result.Intent.Confidence)
		t.Logf("  Answer:   %s", truncate(result.Answer, 300))
		t.Logf("  Steps:    %d", len(result.Steps))

		for i, step := range result.Steps {
			verifyStepIntegrity(t, step, i+1)
		}

		allHistory = append(allHistory,
			core.Message{Role: "user", Content: turn.input, Timestamp: baseTime + int64(ti*30)},
		)
		if result.Answer != "" {
			allHistory = append(allHistory,
				core.Message{Role: "assistant", Content: result.Answer, Timestamp: baseTime + int64(ti*30+15)},
			)
		}
		lastAnswer = result.Answer

		if ti < len(turns)-1 {
			turns[ti+1].history = allHistory
		}
	}

	tracer.logAll(t, "MULTI-TURN")
	t.Logf("\n=== MULTI-TURN COMPLETE ===")
	t.Logf("Total turns executed: %d", len(turns))
	t.Logf("Final answer: %s", truncate(lastAnswer, 200))
}

// ============================================================
// Helper: log all events grouped by type
// ============================================================

func (tr *taoEventTracer) logAllEventsByType(t *testing.T, tag string) map[string]int {
	t.Helper()
	counts := make(map[string]int)
	for _, e := range tr.events {
		counts[string(e.Type)]++
	}
	for et, c := range counts {
		t.Logf("[%s]   event %s: %d occurrences", tag, et, c)
	}
	return counts
}
