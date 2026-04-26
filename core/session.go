package core

import (
	"context"
)

// SlideEvent is emitted when the ContextWindow slides out old messages.
// It contains the messages that were evicted, so consumers (e.g. RAG/Memory)
// can semantically process them into long-term knowledge.
type SlideEvent struct {
	SessionID string   `json:"session_id"`
	Slided    []Message `json:"slided"`
	Remaining int       `json:"remaining"`
	Timestamp int64     `json:"timestamp"`
}

// SlideHandler is the callback type for consuming slide events.
// Implementations can store slid messages into RAG or other long-term storage.
type SlideHandler func(ctx context.Context, event SlideEvent)

// SessionStore is the persistence interface for conversation history (WAL mode).
// It stores messages in order and provides token-budget-aware context retrieval.
//
// Responsibilities:
//   - Append/Retrieve ordered message history per session
//   - CurrentContext returns messages that fit within a token budget (sliding-window read side)
//   - Notify consumers via SlideHandler when messages are evicted from ContextWindow
//
// It does NOT do semantic analysis — that is Memory/RAG's job.
type SessionStore interface {
	// Append adds a message to the end of the specified session's history.
	Append(ctx context.Context, sessionID string, message Message) error

	// Get returns all messages for the specified session (complete history).
	Get(ctx context.Context, sessionID string) ([]Message, error)

	// CurrentContext returns messages suitable for the current context window,
	// selecting from newest to oldest until total tokens <= maxTokens.
	CurrentContext(ctx context.Context, sessionID string, maxTokens int64) ([]Message, error)

	// Delete removes a message by timestamp from the specified session.
	Delete(ctx context.Context, timestamp int64, sessionID string) error

	// Clear removes all messages for the specified session (session reset).
	Clear(ctx context.Context, sessionID string) error

	// SetSlideHandler registers a callback for slide events.
	SetSlideHandler(handler SlideHandler)

	// Close releases any resources held by the store.
	Close() error
}

// NoopSlideHandler is a no-op SlideHandler for implementations that don't need it.
func NoopSlideHandler(_ context.Context, _ SlideEvent) {}

// EmitSlideEvent safely invokes the stored handler if non-nil.
func EmitSlideEvent(handler SlideHandler, ctx context.Context, event SlideEvent) {
	if handler != nil {
		handler(ctx, event)
	}
}
