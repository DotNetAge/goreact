package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// InMemoryMemory is a simple in-memory implementation of the Memory interface.
// It uses keyword matching for retrieval and a map for storage.
// This is suitable for development, testing, and small-scale deployments.
// For production RAG, use gorag or a dedicated vector database implementation.
type InMemoryMemory struct {
	mu      sync.RWMutex
	records map[string]MemoryRecord
	counter atomic.Int64
}

// scoredRecord pairs a MemoryRecord with its relevance score.
type scoredRecord struct {
	record MemoryRecord
	score  float64
}

// NewInMemoryMemory creates a new empty in-memory store.
func NewInMemoryMemory() *InMemoryMemory {
	return &InMemoryMemory{
		records: make(map[string]MemoryRecord),
	}
}

// Retrieve searches stored records by keyword matching against query, title, content, and tags.
// Results are sorted by relevance score (number of keyword matches, normalized).
func (m *InMemoryMemory) Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]MemoryRecord, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	cfg := DefaultRetrieveConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 5
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Extract query keywords (lowercased, deduplicated)
	queryKeywords := extractSearchKeywords(query)
	if len(queryKeywords) == 0 {
		// No keywords in query, return most recent records matching type/scope filters
		return m.filterAndLimit(nil, cfg), nil
	}

	// Score each record by keyword overlap
	var results []scoredRecord

	for _, r := range m.records {
		// Apply type filter
		if len(cfg.Types) > 0 && !m.matchesType(r.Type, cfg.Types) {
			continue
		}
		// Apply scope filter
		if cfg.Scope != 0 && r.Scope != cfg.Scope {
			continue
		}

		// Calculate relevance score
		score := m.calculateScore(r, queryKeywords)
		if score > 0 {
			results = append(results, scoredRecord{record: r, score: score})
		}
	}

	// Sort by score descending
	sortByScore(results)

	// Apply min score filter and limit
	var output []MemoryRecord
	for _, s := range results {
		if s.score < cfg.MinScore {
			continue
		}
		rec := s.record
		rec.Score = s.score
		output = append(output, rec)
		if len(output) >= cfg.Limit {
			break
		}
	}

	return output, nil
}

// Store saves a new memory record and returns its generated ID.
func (m *InMemoryMemory) Store(ctx context.Context, record MemoryRecord) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if record.Content == "" {
		return "", fmt.Errorf("memory content cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	id := record.ID
	if id == "" {
		id = fmt.Sprintf("mem_%d", m.counter.Add(1))
	}
	record.ID = id
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now

	m.records[id] = record
	return id, nil
}

// Update modifies an existing memory record by ID.
func (m *InMemoryMemory) Update(ctx context.Context, id string, record MemoryRecord) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.records[id]
	if !ok {
		return ErrMemoryNotFound
	}

	record.ID = id
	record.CreatedAt = existing.CreatedAt
	record.UpdatedAt = time.Now()
	m.records[id] = record
	return nil
}

// Delete removes a memory record by ID.
func (m *InMemoryMemory) Delete(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.records[id]; !ok {
		return ErrMemoryNotFound
	}
	delete(m.records, id)
	return nil
}

// --- Internal helpers ---

func (m *InMemoryMemory) matchesType(recordType MemoryType, filterTypes []MemoryType) bool {
	for _, t := range filterTypes {
		if recordType == t {
			return true
		}
	}
	return false
}

func (m *InMemoryMemory) calculateScore(r MemoryRecord, queryKeywords []string) float64 {
	// Build searchable text from title + content + tags
	var text strings.Builder
	text.WriteString(strings.ToLower(r.Title))
	text.WriteString(" ")
	text.WriteString(strings.ToLower(r.Content))
	for _, tag := range r.Tags {
		text.WriteString(" ")
		text.WriteString(strings.ToLower(tag))
	}
	recordText := text.String()

	var matched int
	for _, kw := range queryKeywords {
		if strings.Contains(recordText, kw) {
			matched++
		}
	}

	if matched == 0 {
		return 0
	}
	// Normalize: matched / total keywords
	return float64(matched) / float64(len(queryKeywords))
}

func (m *InMemoryMemory) filterAndLimit(records []MemoryRecord, cfg RetrieveConfig) []MemoryRecord {
	if records == nil {
		m.mu.RLock()
		for _, r := range m.records {
			if len(cfg.Types) > 0 && !m.matchesType(r.Type, cfg.Types) {
				continue
			}
			if cfg.Scope != 0 && r.Scope != cfg.Scope {
				continue
			}
			records = append(records, r)
		}
		m.mu.RUnlock()
	}

	// Sort by UpdatedAt descending (most recent first)
	sort.Slice(records, func(i, j int) bool {
		return records[j].UpdatedAt.Before(records[i].UpdatedAt)
	})

	if cfg.Limit > 0 && len(records) > cfg.Limit {
		records = records[:cfg.Limit]
	}
	return records
}

func extractSearchKeywords(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	seen := make(map[string]bool)
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "have": true,
		"has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "as": true,
		"and": true, "or": true, "if": true, "not": true, "no": true,
		"but": true, "that": true, "this": true, "it": true,
	}
	for _, w := range words {
		if len(w) > 1 && !stopWords[w] && !seen[w] {
			keywords = append(keywords, w)
			seen[w] = true
		}
	}
	return keywords
}

func sortByScore(results []scoredRecord) {
	sort.Slice(results, func(i, j int) bool {
		return results[j].score < results[i].score
	})
}
