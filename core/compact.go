package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CompactRequest defines the parameters for a context compaction.
type CompactRequest struct {
	// Messages are the messages to be compacted.
	Messages []Message `json:"messages"`

	// PreserveLastN indicates how many recent messages to keep untouched.
	// Default: 2 (one user + one assistant).
	PreserveLastN int `json:"preserve_last_n"`

	// MaxTokens is the target budget after compaction.
	MaxTokens int64 `json:"max_tokens"`

	// CustomInstruction allows injecting specific instructions for the compaction LLM call.
	CustomInstruction string `json:"custom_instruction,omitempty"`
}

// CompactResult holds the result of a context compaction.
type CompactResult struct {
	// CompactedMessages are the messages after compaction.
	CompactedMessages []Message `json:"compacted_messages"`

	// OriginalTokenCount is the estimated token count before compaction.
	OriginalTokenCount int64 `json:"original_token_count"`

	// CompactedTokenCount is the estimated token count after compaction.
	CompactedTokenCount int64 `json:"compacted_token_count"`

	// SummaryTokenCount is the number of tokens used for the summary.
	SummaryTokenCount int64 `json:"summary_token_count"`
}

// ContextCompactor is the interface for context window compaction.
// Inspired by ClueCode's cludecode/services/compact/compact.ts
type ContextCompactor interface {
	// Compact compresses the given messages into a shorter form while
	// preserving essential information. It uses an LLM call to generate a summary.
	Compact(ctx context.Context, req CompactRequest) (*CompactResult, error)
}

// CompactorConfig holds configuration for the context compactor.
type CompactorConfig struct {
	// CompactThresholdRatio triggers compaction when tokens used exceed
	// this ratio of max tokens. Default: 0.8.
	CompactThresholdRatio float64

	// PreserveLastN messages are always kept (not compacted). Default: 2.
	PreserveLastN int

	// MicroCompactThreshold is a lower threshold for quick compaction
	// that only summarizes observation results without LLM call.
	MicroCompactThreshold float64
}

// DefaultCompactorConfig returns sensible defaults.
func DefaultCompactorConfig() CompactorConfig {
	return CompactorConfig{
		CompactThresholdRatio:  DefaultToolResultLimits().CompactThresholdRatio,
		PreserveLastN:          2,
		MicroCompactThreshold:  0.6,
	}
}

// SummaryMessage generates a compact boundary message that indicates
// where compaction occurred, similar to ClueCode's SystemCompactBoundaryMessage.
func SummaryMessage(originalCount int, compactedCount int, summary string) Message {
	return Message{
		Role:    "system",
		Content: fmt.Sprintf(
			"[Context Compacted] Previous %d messages were summarized into %d messages.\nSummary: %s",
			originalCount, compactedCount, summary,
		),
	}
}

// MicroCompact performs a quick, non-LLM compaction by truncating large
// observation/tool-result messages while preserving small messages intact.
// This is ClueCode's "microCompact" equivalent.
func MicroCompact(messages []Message, estimateFn func(string) int, targetTokens int64) []Message {
	if estimateFn == nil {
		estimateFn = func(s string) int { return len(s) / 3 }
	}
	if len(messages) <= 2 {
		return messages
	}

	var totalTokens int64
	for _, m := range messages {
		totalTokens += int64(estimateFn(m.Content))
	}

	if totalTokens <= targetTokens {
		return messages
	}

	// Strategy: truncate large messages proportionally to fit budget
	remaining := targetTokens
	result := make([]Message, 0, len(messages))

	// First pass: keep all messages, calculate their token costs
	type msgToken struct {
		msg    Message
		tokens int64
	}
	items := make([]msgToken, len(messages))
	for i, m := range messages {
		items[i] = msgToken{msg: m, tokens: int64(estimateFn(m.Content))}
	}

	// Process from oldest to newest, preserving newest messages
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].tokens <= remaining {
			result = append(result, items[i].msg)
			remaining -= items[i].tokens
		} else if remaining > 100 {
			// Truncate to fit remaining budget
			charsBudget := remaining * 3 // rough token-to-char ratio
			runes := []rune(items[i].msg.Content)
			if len(runes) > int(charsBudget) {
				truncated := string(runes[:int(charsBudget)]) + "\n... [compacted] ..."
				result = append(result, Message{
					Role:      items[i].msg.Role,
					Content:   truncated,
					Timestamp: items[i].msg.Timestamp,
				})
				remaining = 0
			} else {
				result = append(result, items[i].msg)
				remaining -= items[i].tokens
			}
		}
		// else: skip this message (not enough budget)
	}

	// Reverse to maintain chronological order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// TokenEstimator provides token counting for strings.
type TokenEstimator interface {
	// Estimate returns the approximate token count for the given text.
	Estimate(text string) int
}

// DefaultTokenEstimator provides a simple heuristic-based token estimate.
type DefaultTokenEstimator struct {
	// CharsPerToken is the assumed average characters per token.
	// English: ~4, CJK: ~2, mixed: ~3.
	CharsPerToken float64
}

// NewDefaultTokenEstimator creates an estimator with the given chars-per-token ratio.
func NewDefaultTokenEstimator(charsPerToken float64) *DefaultTokenEstimator {
	if charsPerToken <= 0 {
		charsPerToken = 3.0
	}
	return &DefaultTokenEstimator{CharsPerToken: charsPerToken}
}

// Estimate returns the approximate token count.
func (e *DefaultTokenEstimator) Estimate(text string) int {
	return int(float64(len(text)) / e.CharsPerToken)
}

// ContextBudget represents the current state of the context window budget.
type ContextBudget struct {
	MaxTokens    int64 `json:"max_tokens"`
	UsedTokens   int64 `json:"used_tokens"`
	Remaining    int64 `json:"remaining"`
	UsageRatio   float64 `json:"usage_ratio"`
	NeedCompact  bool   `json:"need_compact"`
}

// CalculateBudget computes the context budget state.
func CalculateBudget(maxTokens int64, usedTokens int64, compactRatio float64) ContextBudget {
	if maxTokens <= 0 {
		return ContextBudget{MaxTokens: maxTokens, UsedTokens: usedTokens}
	}
	remaining := maxTokens - usedTokens
	if remaining < 0 {
		remaining = 0
	}
	ratio := float64(usedTokens) / float64(maxTokens)
	return ContextBudget{
		MaxTokens:   maxTokens,
		UsedTokens:  usedTokens,
		Remaining:   remaining,
		UsageRatio:  ratio,
		NeedCompact: ratio >= compactRatio,
	}
}

// TrimJSONResult attempts to trim a JSON result by removing large array elements
// or string values while keeping the structure intact. Useful for tool results
// that return structured data (e.g., search results, file listings).
func TrimJSONResult(jsonStr string, maxChars int) string {
	if len(jsonStr) <= maxChars {
		return jsonStr
	}

	// Try to parse as JSON and trim intelligently
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// Not valid JSON, fall back to simple truncation
		runes := []rune(jsonStr)
		if len(runes) > maxChars {
			return string(runes[:maxChars]) + "\n... [truncated] ..."
		}
		return jsonStr
	}

	trimmed := trimValue(data, maxChars)
	result, _ := json.Marshal(trimmed)
	if len(result) > maxChars {
		result = result[:maxChars]
		result = append(result, []byte("\n... [truncated] ...")...)
	}
	return string(result)
}

func trimValue(v any, maxChars int) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		currentSize := 0
		for k, v := range val {
			if currentSize > maxChars/2 {
				result[k] = fmt.Sprintf("[trimmed: %d remaining keys skipped]", len(val)-len(result))
				break
			}
			trimmed := trimValue(v, maxChars/2)
			b, _ := json.Marshal(trimmed)
			currentSize += len(b)
			result[k] = trimmed
		}
		return result
	case []any:
		if len(val) > 20 {
			// Keep first 10 and last 5
			kept := make([]any, 0, 15)
			for i := 0; i < 10 && i < len(val); i++ {
				kept = append(kept, trimValue(val[i], maxChars/15))
			}
			kept = append(kept, fmt.Sprintf("[%d items omitted]", len(val)-15))
			for i := len(val) - 5; i < len(val); i++ {
				kept = append(kept, trimValue(val[i], maxChars/15))
			}
			return kept
		}
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = trimValue(item, maxChars/len(val))
		}
		return result
	case string:
		runes := []rune(val)
		if len(runes) > 500 {
			return string(runes[:500]) + "... [trimmed] ..."
		}
		return val
	default:
		return v
	}
}

// IsJSONString checks if the given string appears to be JSON.
func IsJSONString(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}
