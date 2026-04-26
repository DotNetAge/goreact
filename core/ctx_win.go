package core

import (
	"sync"
	"time"
)

// ContextWindow manages multi-turn conversation context for a session.
// It is safe for concurrent use and supports token-aware pruning.
type ContextWindow struct {
	mu             sync.RWMutex
	SessionID      string    `json:"session_id"`
	Messages       []Message `json:"messages"`
	TokensUsed     int64     `json:"tokens_used"`
	MaxTokens      int64     `json:"max_tokens"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

// NewContextWindow creates a new ContextWindow for the given session.
func NewContextWindow(sessionID string, maxTokens int64) *ContextWindow {
	now := time.Now()
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}
	return &ContextWindow{
		SessionID:      sessionID,
		Messages:       make([]Message, 0),
		MaxTokens:      maxTokens,
		CreatedAt:      now,
		LastActivityAt: now,
	}
}

// AddMessage appends a message to the context window.
func (w *ContextWindow) AddMessage(role, content string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Messages = append(w.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	})
	w.LastActivityAt = time.Now()
}

// AddMessageWithTimestamp appends a message with a specific timestamp.
func (w *ContextWindow) AddMessageWithTimestamp(role, content string, ts int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Messages = append(w.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: ts,
	})
	w.LastActivityAt = time.Now()
}

// Messages returns a copy of all messages in the window.
func (w *ContextWindow) GetMessages() []Message {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]Message, len(w.Messages))
	copy(out, w.Messages)
	return out
}

// RecentMessages returns the last N messages for prompt injection.
// If n <= 0, returns all messages.
func (w *ContextWindow) RecentMessages(n int) []Message {
	w.mu.RLock()
	defer w.mu.RUnlock()
	msgs := w.Messages
	if n > 0 && len(msgs) > n {
		msgs = msgs[len(msgs)-n:]
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out
}

// AddTokens accumulates token usage for tracking context window budget.
func (w *ContextWindow) AddTokens(n int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.TokensUsed += n
}

// TokensRemaining returns estimated remaining tokens.
func (w *ContextWindow) TokensRemaining() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	remaining := w.MaxTokens - w.TokensUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset clears all messages and token counts, starting fresh.
func (w *ContextWindow) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Messages = make([]Message, 0)
	w.TokensUsed = 0
	w.LastActivityAt = time.Now()
}

// MessageCount returns the number of messages in the window.
func (w *ContextWindow) MessageCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.Messages)
}

// ---------------------------------------------------------------------------
// Sliding Window Support
// ---------------------------------------------------------------------------

// SlideConfig controls when and how the context window slides out old messages.
type SlideConfig struct {
	SlideTriggerRatio   float64
	TargetRatio         float64
	MinPreserveMessages int
	MaxSlideBatch       int
}

// DefaultSlideConfig returns sensible defaults for T-A-O agent workloads.
// Trigger at 65% usage (before "Lost in the Middle" ~70%), target 45% after slide,
// preserve minimum 4 messages (2 full turns).
var DefaultSlideConfig = SlideConfig{
	SlideTriggerRatio:   0.65,
	TargetRatio:         0.45,
	MinPreserveMessages: 4,
	MaxSlideBatch:       0,
}

// SlidedMessages records the result of a Slide operation.
type SlidedMessages struct {
	Messages   []Message
	TokenCount int64
}

// UsageRatio returns current token usage as a ratio of MaxTokens (0.0–1.0+).
func (w *ContextWindow) UsageRatio() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.MaxTokens <= 0 {
		return 0
	}
	return float64(w.TokensUsed) / float64(w.MaxTokens)
}

// SlideTriggered returns true if token usage meets or exceeds the trigger ratio.
func (w *ContextWindow) SlideTriggered(config SlideConfig) bool {
	return w.UsageRatio() >= config.SlideTriggerRatio
}

// calculateTotalTokens estimates total tokens across all messages using the given function.
func (w *ContextWindow) calculateTotalTokens(estimateFn func(string) int) int64 {
	var total int64
	for _, m := range w.Messages {
		total += int64(estimateFn(m.Content))
	}
	return total
}

// Slide batch-removes oldest messages until token usage drops to TargetRatio.
// It always preserves at least MinPreserveMessages messages.
// Returns the evicted messages so callers can forward them to RAG/Memory.
//
// Performance: O(n) total — uses incremental token tracking instead of
// recalculating from scratch on each iteration.
func (w *ContextWindow) Slide(config SlideConfig, estimateFn func(string) int) SlidedMessages {
	w.mu.Lock()
	defer w.mu.Unlock()

	estimate := estimateFn
	if estimate == nil {
		estimate = EstimateTokens
	}

	targetTokens := int64(float64(w.MaxTokens) * config.TargetRatio)
	var slided []Message
	var slidedTokens int64

	totalTokens := w.calculateTotalTokens(estimate)

	for len(w.Messages) > config.MinPreserveMessages {
		if totalTokens <= targetTokens {
			break
		}

		removed := w.Messages[0]
		removedTokens := int64(estimate(removed.Content))

		w.Messages = w.Messages[1:]
		slided = append(slided, removed)
		slidedTokens += removedTokens

		totalTokens -= removedTokens

		if config.MaxSlideBatch > 0 && len(slided) >= config.MaxSlideBatch {
			break
		}
	}

	w.TokensUsed = totalTokens
	w.LastActivityAt = time.Now()

	return SlidedMessages{Messages: slided, TokenCount: slidedTokens}
}
