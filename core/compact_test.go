package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSummaryMessage(t *testing.T) {
	msg := SummaryMessage(100, 5, "test summary")
	if msg.Role != "system" {
		t.Errorf("expected role=system, got %s", msg.Role)
	}
	expected := "[Context Compacted] Previous 100 messages were summarized into 5 messages.\nSummary: test summary"
	if msg.Content != expected {
		t.Errorf("mismatch: got %q", msg.Content)
	}
}

func TestMicroCompact_NilEstimateFn(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "short"},
		{Role: "assistant", Content: "also short"},
	}
	result := MicroCompact(messages, nil, 10)
	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}
}

func TestMicroCompact_WithinBudget(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
		{Role: "user", Content: "how are you"},
	}
	result := MicroCompact(messages, func(s string) int { return len(s) }, 1000)
	if len(result) != 3 {
		t.Errorf("expected all 3 messages preserved when within budget")
	}
}

func TestMicroCompact_TruncatesLargeMessages(t *testing.T) {
	largeContent := make([]byte, 9000)
	for i := range largeContent {
		largeContent[i] = 'A'
	}

	messages := []Message{
		{Role: "user", Content: "small msg 1"},
		{Role: "assistant", Content: string(largeContent)},
		{Role: "user", Content: "small msg 2"},
	}
	result := MicroCompact(messages, func(s string) int { return len(s) / 3 }, 500)

	found := false
	for _, m := range result {
		if len(m.Content) < len(largeContent) && len(m.Content) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected large message to be truncated or skipped")
	}
	totalTokens := int64(0)
	for _, m := range result {
		totalTokens += int64(len(m.Content) / 3)
	}
	if totalTokens > 500*2 {
		t.Errorf("total tokens after compact should be near budget, got %d", totalTokens)
	}
}

func TestMicroCompact_PreservesOrder(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "msg-001-first"},
		{Role: "assistant", Content: "msg-002-second"},
		{Role: "user", Content: "msg-003-third"},
		{Role: "assistant", Content: "msg-004-fourth"},
	}
	result := MicroCompact(messages, func(s string) int { return len(s) }, 20)
	if len(result) == 0 {
		t.Fatal("expected at least some messages")
	}
	for i := 1; i < len(result); i++ {
		if result[i].Content <= result[i-1].Content {
			t.Errorf("messages not in chronological order at index %d: %q before %q", i, result[i-1].Content, result[i].Content)
		}
	}
}

func TestMicroCompact_ShortInput(t *testing.T) {
	messages := []Message{{Role: "user", Content: "only one"}}
	result := MicroCompact(messages, nil, 1)
	if len(result) != 1 {
		t.Errorf("single message should pass through")
	}
	messages2 := []Message{}
	result2 := MicroCompact(messages2, nil, 1)
	if len(result2) != 0 {
		t.Errorf("empty input should return empty")
	}
}

func TestDefaultTokenEstimator(t *testing.T) {
	e := NewDefaultTokenEstimator(4.0)
	if e.Estimate("hello world") != 2 {
		t.Errorf("expected 2 tokens, got %d", e.Estimate("hello world"))
	}
	e2 := NewDefaultTokenEstimator(-1)
	if e2.CharsPerToken != 3.0 {
		t.Errorf("negative charsPerToken should default to 3.0, got %f", e2.CharsPerToken)
	}
	e3 := NewDefaultTokenEstimator(0)
	if e3.CharsPerToken != 3.0 {
		t.Errorf("zero charsPerToken should default to 3.0, got %f", e3.CharsPerToken)
	}
}

func TestCalculateBudget(t *testing.T) {
	b := CalculateBudget(8000, 6000, 0.8)
	if b.UsageRatio != 0.75 {
		t.Errorf("expected ratio 0.75, got %f", b.UsageRatio)
	}
	if b.NeedCompact {
		t.Error("should not need compact at 75%")
	}
	if b.Remaining != 2000 {
		t.Errorf("expected remaining 2000, got %d", b.Remaining)
	}

	b2 := CalculateBudget(8000, 7000, 0.8)
	if !b2.NeedCompact {
		t.Error("should need compact at 87.5%")
	}

	b3 := CalculateBudget(8000, 9000, 0.8)
	if b3.Remaining != 0 {
		t.Errorf("remaining should be clamped to 0, got %d", b3.Remaining)
	}

	b4 := CalculateBudget(0, 100, 0.8)
	if b4.Remaining != 0 || b4.NeedCompact {
		t.Error("zero max tokens should produce safe defaults")
	}
}

func TestTrimJSONResult_ShortString(t *testing.T) {
	input := `{"key": "value"}`
	result := TrimJSONResult(input, 100)
	if result != input {
		t.Errorf("short JSON should pass through unchanged")
	}
}

func TestTrimJSONResult_TrimObject(t *testing.T) {
	largeObj := make(map[string]string)
	for i := 0; i < 50; i++ {
		largeObj[fmt.Sprintf("field_%d", i)] = strings.Repeat("x", 200)
	}
	data, _ := json.Marshal(largeObj)
	input := string(data)

	result := TrimJSONResult(input, 500)
	if len(result) > 550 {
		t.Errorf("trimmed result should be ~500 chars, got %d", len(result))
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("result should still be valid JSON: %v", err)
	}
}

func TestTrimJSONResult_TrimArray(t *testing.T) {
	arr := make([]string, 30)
	for i := range arr {
		arr[i] = string(make([]byte, 200))
	}
	data, _ := json.Marshal(arr)
	input := string(data)

	result := TrimJSONResult(input, 1000)
	if len(result) > 1100 {
		t.Errorf("trimmed array result too long: %d", len(result))
	}
}

func TestTrimJSONResult_NonJSON(t *testing.T) {
	longStr := strings.Repeat("a", 1000)
	result := TrimJSONResult(longStr, 500)
	if len(result) > 520 {
		t.Errorf("non-JSON should be truncated simply, got %d", len(result))
	}
}

func TestTrimJSONResult_TrimLongStrings(t *testing.T) {
	obj := map[string]any{"data": strings.Repeat("y", 1000)}
	data, _ := json.Marshal(obj)
	result := TrimJSONResult(string(data), 200)
	if len(result) > 220 {
		t.Errorf("long string values should be trimmed, got %d", len(result))
	}
}

func TestIsJSONString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"a": 1}`, true},
		{`[1, 2, 3]`, true},
		{`  {"a": 1}  `, true},
		{`  [1, 2]  `, true},
		{"not json", false},
		{"", false},
		{"{invalid}", true},
		{"[invalid]", true},
	}
	for _, tt := range tests {
		got := IsJSONString(tt.input)
		if got != tt.expected {
			t.Errorf("IsJSONString(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestCompactorConfigDefaults(t *testing.T) {
	cfg := DefaultCompactorConfig()
	if cfg.CompactThresholdRatio <= 0 {
		t.Error("CompactThresholdRatio should be positive")
	}
	if cfg.PreserveLastN != 2 {
		t.Errorf("PreserveLastN should be 2, got %d", cfg.PreserveLastN)
	}
	if cfg.MicroCompactThreshold != 0.6 {
		t.Errorf("MicroCompactThreshold should be 0.6, got %f", cfg.MicroCompactThreshold)
	}
}
