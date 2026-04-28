package core

import (
	"context"
	"sort"
	"sync"
	"time"
)

// sessionMeta holds per-session metadata for role-based lookup.
type sessionMeta struct {
	role           string
	createdAt      time.Time
	lastActivityAt time.Time
}

// MemorySessionStore is an in-memory SessionStore implementation.
// Suitable for development, testing, and single-process deployments.
//
// It supports role-based session isolation: each session can be bound to an
// agent role so that Agent.Switch() resumes the most recent session for that
// role instead of creating a new one.
type MemorySessionStore struct {
	mu      sync.RWMutex
	store   map[string][]Message    // sessionID -> messages
	metas   map[string]*sessionMeta // sessionID -> metadata (role, timestamps)
	handler SlideHandler
}

// NewMemorySessionStore creates a new empty in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		store:   make(map[string][]Message),
		metas:   make(map[string]*sessionMeta),
		handler: NoopSlideHandler,
	}
}

func (s *MemorySessionStore) Append(_ context.Context, sessionID string, message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[sessionID] = append(s.store[sessionID], message)

	// Update last activity timestamp in metadata
	if meta, ok := s.metas[sessionID]; ok {
		meta.lastActivityAt = time.Now()
	} else {
		s.metas[sessionID] = &sessionMeta{
			lastActivityAt: time.Now(),
			createdAt:      time.Now(),
		}
	}
	return nil
}

func reverseMessages(msgs []Message) {
	for i, j := 0, len(msgs)-1; i < j; {
		msgs[i], msgs[j] = msgs[j], msgs[i]
		i++
		j--
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

// RegisterRole binds a session ID to an agent role.
// This should be called when a ContextWindow with a role is set on the Reactor.
func (s *MemorySessionStore) RegisterRole(sessionID, role string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if existing, ok := s.metas[sessionID]; ok {
		existing.role = role
	} else {
		s.metas[sessionID] = &sessionMeta{
			role:           role,
			createdAt:      now,
			lastActivityAt: now,
		}
	}
}

// GetByRole returns the most recent SessionInfo for the given role.
// Returns ErrSessionNotFound if no session exists for that role.
func (s *MemorySessionStore) GetByRole(_ context.Context, role string) (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bestID string
	var bestTime time.Time
	for sid, meta := range s.metas {
		if meta.role == role && meta.lastActivityAt.After(bestTime) {
			bestID = sid
			bestTime = meta.lastActivityAt
		}
	}
	if bestID == "" {
		return nil, ErrSessionNotFound
	}
	meta := s.metas[bestID]
	return &SessionInfo{
		SessionID:      bestID,
		Role:           meta.role,
		MessageCount:   len(s.store[bestID]),
		LastActivityAt: meta.lastActivityAt,
		CreatedAt:      meta.createdAt,
	}, nil
}

// ListSessions returns all sessions sorted by LastActivityAt descending (newest first).
func (s *MemorySessionStore) ListSessions(_ context.Context) ([]SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SessionInfo, 0, len(s.metas))
	for sid, meta := range s.metas {
		result = append(result, SessionInfo{
			SessionID:      sid,
			Role:           meta.role,
			MessageCount:   len(s.store[sid]),
			LastActivityAt: meta.lastActivityAt,
			CreatedAt:      meta.createdAt,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastActivityAt.After(result[j].LastActivityAt)
	})
	return result, nil
}
