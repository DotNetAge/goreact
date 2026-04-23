package core

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestCountTokens_EmptyString(t *testing.T) {
	count, err := CountTokens("")
	if err != nil {
		t.Fatalf("CountTokens(\"\") error = %v", err)
	}
	if count != 0 {
		t.Errorf("CountTokens(\"\") = %d, want 0", count)
	}
}

func TestCountTokens_ShortASCII(t *testing.T) {
	tests := []struct {
		text     string
		min, max int // GPT-4o token range (allowing for encoding variations)
	}{
		{"hi", 1, 2},
		{"Hello", 1, 2},
		{"Hello, world!", 3, 4},
		{"The quick brown fox", 4, 5},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q", tc.text), func(t *testing.T) {
			count, err := CountTokens(tc.text)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if count < tc.min || count > tc.max {
				t.Errorf("CountTokens(%q) = %d, want [%d,%d]", tc.text, count, tc.min, tc.max)
			}
		})
	}
}

func TestCountTokens_PureCJK(t *testing.T) {
	count, err := CountTokens("你好世界")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if count < 2 || count > 8 {
		t.Errorf("CountTokens(\"你好世界\") = %d, want reasonable CJK token count [2,8]", count)
	}
}

func TestCountTokens_MixedContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		min      int
	}{
		{"EN/CJK mix", "Hello 世界!", 3},
		{"Emoji only", "🙂🙂🙂", 1},
		{"Code snippet", "func main() { fmt.Println(\"hello\") }", 8},
		{"JSON", `{"key": "value", "count": 42}`, 5},
		{"Markdown", "# Title\n\nSome **bold** text with `code`.", 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			count, err := CountTokens(tc.text)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if count < tc.min {
				t.Errorf("CountTokens(%q) = %d, want >= %d", tc.text, count, tc.min)
			}
		})
	}
}

func TestCountTokens_SpecialCharacters(t *testing.T) {
	specialInputs := []string{
		"\n\t\r",
		"line1\nline2\nline3",
		"tab\there",
		"quote\"escape\\slash",
		"<html>&amp;entity</html>",
		"${VAR} $(command)",
		"path/with/slashes\\and\\backslashes",
		"unicode: \u0000\u001f\ufffd",
		strings.Repeat("x", 10000),
	}
	for i, input := range specialInputs {
		t.Run(fmt.Sprintf("special_%d", i), func(t *testing.T) {
			count, err := CountTokens(input)
			if err != nil {
				t.Fatalf("error on special input: %v", err)
			}
			if count < 0 {
				t.Errorf("negative token count: %d", count)
			}
		})
	}
}

func TestCountTokens_VeryLongText(t *testing.T) {
	longText := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 10000) // ~470K chars
	count, err := CountTokens(longText)
	if err != nil {
		t.Fatalf("error on long text: %v", err)
	}
	if count <= 0 {
		t.Error("expected positive token count for long text")
	}
	ratio := float64(count) / float64(len(longText))
	if ratio > 1.0 || ratio < 0.1 {
		t.Errorf("token/char ratio %.3f seems unreasonable for long text", ratio)
	}
}

func TestCountTokens_UnicodeEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		text string
		min  int
	}{
		{"Zero-width joiner", "👨‍👩‍👧‍👦", 1},
		{"Combining characters", "e\u0301", 1},
		{"RTL text", "مرحبا بالعالم", 2},
		{"Fullwidth ASCII", "ＨｅｌＬＯ", 3},
		{"Math symbols", "∑∫∂∇≈≠≤≥∞", 5},
		{"Arrows and symbols", "←→↑↓↔↕⇐⇑⇒⇓⇔⇕", 5},
		{"Musical notes", "♩♪♫♬♭♮♯", 5},
		{"Chess pieces", "♔♕♖♗♘♙", 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			count, err := CountTokens(tc.text)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if count < tc.min {
				t.Errorf("CountTokens(%q) = %d, want >= %d", tc.text, count, tc.min)
			}
		})
	}
}

func TestEstimateTokens_DelegatesToTiktoken(t *testing.T) {
	count := EstimateTokens("Hello, world!")
	if count <= 0 {
		t.Errorf("EstimateTokens returned non-positive: %d", count)
	}
	count2, _ := CountTokens("Hello, world!")
	if count != count2 {
		t.Errorf("EstimateTokens and CountTokens disagree: %d vs %d", count, count2)
	}
}

func TestCountTokens_Consistency(t *testing.T) {
	text := "Testing consistency across multiple calls with mixed content: 你好世界 🌍 123"
	var counts []int
	for i := 0; i < 100; i++ {
		count, err := CountTokens(text)
		if err != nil {
			t.Fatalf("call %d error: %v", i, err)
		}
		counts = append(counts, count)
	}
	first := counts[0]
	for i, c := range counts {
		if c != first {
			t.Errorf("inconsistent result at call %d: got %d, expected %d", i, c, first)
		}
	}
}

func TestCountTokens_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := CountTokens(fmt.Sprintf("concurrent test %d: 你好世界 🎉", id))
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent error: %v", err)
	}
}

func TestNewTiktokenTokenCounter(t *testing.T) {
	tc, err := NewTiktokenTokenCounter("gpt-4o")
	if err != nil {
		t.Fatalf("NewTiktokenTokenCounter(gpt-4o): %v", err)
	}
	if tc.Model() != "gpt-4o" {
		t.Errorf("Model() = %q, want gpt-4o", tc.Model())
	}

	count, err := tc.Count("Hello, world!")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count <= 0 {
		t.Errorf("Count returned %d, want > 0", count)
	}
}

func TestNewTiktokenTokenCounter_DefaultModel(t *testing.T) {
	tc, err := NewTiktokenTokenCounter("")
	if err != nil {
		t.Fatalf("NewTiktokenTokenCounter(\"\"): %v", err)
	}
	if tc.Model() != "gpt-4o" {
		t.Errorf("default model = %q, want gpt-4o", tc.Model())
	}
}

func TestNewTiktokenTokenCounter_InvalidModel(t *testing.T) {
	_, err := NewTiktokenTokenCounter("nonexistent-model-v999")
	if err == nil {
		t.Error("expected error for invalid model")
	}
}

func TestTiktokenTokenCounter_EmptyString(t *testing.T) {
	tc, _ := NewTiktokenTokenCounter("gpt-4o")
	count, err := tc.Count("")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if count != 0 {
		t.Errorf("Count(\"\") = %d, want 0", count)
	}
}

func TestHeuristicTokenCounter(t *testing.T) {
	hc := NewHeuristicTokenCounter()
	if hc.Model() != "heuristic-v2" {
		t.Errorf("Model() = %q, want heuristic-v2", hc.Model())
	}

	count, err := hc.Count("Hello world")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if count <= 0 {
		t.Errorf("Count returned %d, want > 0", count)
	}

	count, err = hc.Count("")
	if err != nil || count != 0 {
		t.Errorf("Count(\"\") = (%d, %v), want (0, nil)", count, err)
	}
}

func TestTokenCounter_InterfaceSatisfaction(t *testing.T) {
	var _ TokenCounter = (*TiktokenTokenCounter)(nil)
	var _ TokenCounter = (*HeuristicTokenCounter)(nil)
}

func TestCountTokens_RoundtripDecode(t *testing.T) {
	enc, err := getGlobalEncoder()
	if err != nil {
		t.Skipf("skipping: encoder init failed: %v", err)
	}

	testStrings := []string{
		"Hello, world!",
		"你好世界",
		"🎉🎊🎈",
		"line1\nline2\ttab",
		`{"key": "value"}`,
	}

	for _, s := range testStrings {
		t.Run(fmt.Sprintf("%q", s[:min(len(s), 20)]), func(t *testing.T) {
			tokens := enc.Encode(s, nil, nil)
			decoded := enc.Decode(tokens)
			if decoded != s {
				t.Errorf("roundtrip mismatch:\n  original: %q\n  decoded:  %q\n  tokens:  %v", s, decoded, tokens)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestEstimateTokens_KnownGPT4oValues(t *testing.T) {
	tests := []struct {
		text          string
		expectedExact int // exact o200k_base token count for GPT-4o
	}{
		{"", 0},
		{" ", 1},
		{"  ", 1},
		{"hi", 1},
		{"hi ", 2},
		{"\n", 1},
		{"\n\n", 1},
		{"0", 1},
		{"00", 1},
		{"000", 1},
		{"0000", 2},
		{"00000", 2},
	}
	for _, tc := range tests {
		name := fmt.Sprintf("%q", tc.text)
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		t.Run(name, func(t *testing.T) {
			got := EstimateTokens(tc.text)
			if got != tc.expectedExact {
				t.Errorf("EstimateTokens(%q) = %d, want %d (exact o200k_base)", tc.text, got, tc.expectedExact)
			}
		})
	}
}
