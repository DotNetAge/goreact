package core

import (
	"context"
	"sync"
)

// MemorySessionStore is an in-memory SessionStore implementation.
// Suitable for development, testing, and single-process deployments.
type MemorySessionStore struct {
	mu      sync.RWMutex
	store   map[string][]Message
	handler SlideHandler
}

// NewMemorySessionStore creates a new empty in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		store:   make(map[string][]Message),
		handler: NoopSlideHandler,
	}
}

func (s *MemorySessionStore) Append(_ context.Context, sessionID string, message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[sessionID] = append(s.store[sessionID], message)
	return nil
}

func reverseMessages(msgs []Message) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

func (s *MemorySessionStore) Get(_ context.Context, sessionID string) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.store[sessionID]
	if msgs == nil {
		return nil, nil
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

func (s *MemorySessionStore) CurrentContext(_ context.Context, sessionID string, maxTokens int64) ([]Message, error) {
	s.mu.RLock()
	msgs := make([]Message, len(s.store[sessionID]))
	copy(msgs, s.store[sessionID])
	s.mu.RUnlock()

	if len(msgs) == 0 {
		return nil, nil
	}

	var selected []Message
	var usedTokens int64
	for i := len(msgs) - 1; i >= 0; i-- {
		msgTokens := int64(EstimateTokens(msgs[i].Content))
		if usedTokens+msgTokens > maxTokens {
			break
		}
		selected = append(selected, msgs[i])
		usedTokens += msgTokens
	}
	reverseMessages(selected)
	return selected, nil
}

func (s *MemorySessionStore) Delete(_ context.Context, timestamp int64, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.store[sessionID]
	filtered := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Timestamp != timestamp {
			filtered = append(filtered, m)
		}
	}
	s.store[sessionID] = filtered
	return nil
}

func (s *MemorySessionStore) Clear(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[sessionID] = nil
	return nil
}

func (s *MemorySessionStore) SetSlideHandler(handler SlideHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

func (s *MemorySessionStore) Close() error {
	return nil
}
