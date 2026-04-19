package core

import (
	"sync"
	"time"
)

// Message represents a single message in a conversation.
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

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

// Prune removes the oldest messages until the token estimate is within budget.
// It always preserves at least the last 2 messages (one user + one assistant).
// tokenEstimate is an external function; if nil, a simple heuristic is used.
func (w *ContextWindow) Prune(estimateTokenFn func(string) int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.Messages) <= 2 {
		return
	}

	estimate := estimateTokenFn
	if estimate == nil {
		estimate = func(s string) int {
			// Rough estimate: 4 chars per token for English, 2 chars per token for CJK
			return len(s) / 3
		}
	}

	// Calculate total tokens from messages
	var totalTokens int64
	for _, m := range w.Messages {
		totalTokens += int64(estimate(m.Content))
	}

	// Remove oldest messages until within budget
	for len(w.Messages) > 2 && totalTokens > w.MaxTokens {
		removed := w.Messages[0]
		totalTokens -= int64(estimate(removed.Content))
		w.Messages = w.Messages[1:]
	}

	w.TokensUsed = totalTokens
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

// TruncateResultSize limits the length of a result string to fit within
// the remaining context budget. Returns the (possibly truncated) string.
func (w *ContextWindow) TruncateResultSize(result string, maxRunes int) string {
	runes := []rune(result)
	if len(runes) <= maxRunes {
		return result
	}
	return string(runes[:maxRunes]) + "... [truncated due to context budget] ..."
}
