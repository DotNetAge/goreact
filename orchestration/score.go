package orchestration

import (
	"math/rand"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ===========================================================================
// Score Tracker — performance tracking
// ===========================================================================

const (
	// ScorePerfect is for flawless execution (3/3)
	ScorePerfect = 3
	// ScoreSuccess is for successful execution with minor issues (2/3)
	ScoreSuccess = 2
	// ScorePartial is for barely successful execution (1/3)
	ScorePartial = 1
	// ScoreFailed is for failed execution (0/3)
	ScoreFailed = 0

	// DefaultEpsilon is the exploration probability for epsilon-greedy selection.
	// At cold start, agents are randomly selected with this probability.
	DefaultEpsilon = 0.1

	// MinSamplesForExploitation is the minimum number of scores before switching
	// from exploration to exploitation mode for an agent.
	MinSamplesForExploitation = 3

	// DefaultInitialTrustScore is the initial performance score for newly created agents (Design §8.4).
	// Value 2.0 assumes a new agent is of medium quality until proven otherwise.
	DefaultInitialTrustScore = 2.0
)

// ScoreTracker tracks and manages agent performance scores using an
// epsilon-greedy strategy for agent selection during cold start and beyond.
type ScoreTracker struct {
	mu          sync.RWMutex
	scores      map[string][]scoreEntry // key: agent ID -> score history
	totalScores map[string]float64      // Running total for quick average
	epsilon     float64                 // Exploration probability
}

type scoreEntry struct {
	Score     int
	Success   bool
	Timestamp time.Time
	TaskID    string
}

// NewScoreTracker creates a new ScoreTracker with default epsilon-greedy parameters.
func NewScoreTracker() *ScoreTracker {
	return &ScoreTracker{
		scores:      make(map[string][]scoreEntry),
		totalScores: make(map[string]float64),
		epsilon:     DefaultEpsilon,
	}
}

// SetEpsilon updates the exploration probability.
func (st *ScoreTracker) SetEpsilon(eps float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.epsilon = eps
}

// RecordScore records a performance score for an agent.
func (st *ScoreTracker) RecordScore(agentID string, score int, success bool, taskID string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	entry := scoreEntry{
		Score:     score,
		Success:   success,
		Timestamp: time.Now(),
		TaskID:    taskID,
	}
	st.scores[agentID] = append(st.scores[agentID], entry)
	st.totalScores[agentID] += float64(score)
}

// GetScore returns the average score and total count for an agent.
// Returns (0, 0) if no scores exist.
func (st *ScoreTracker) GetScore(agentID string) (avg float64, count int) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	entries, ok := st.scores[agentID]
	if !ok || len(entries) == 0 {
		return 0, 0
	}
	return st.totalScores[agentID] / float64(len(entries)), len(entries)
}

// GetAllScores returns a map of all agent IDs to their (average, count).
func (st *ScoreTracker) GetAllScores() map[string]struct{ Avg float64; Count int } {
	st.mu.RLock()
	defer st.mu.RUnlock()

	result := make(map[string]struct{ Avg float64; Count int }, len(st.scores))
	for id, entries := range st.scores {
		if len(entries) > 0 {
			result[id] = struct {
				Avg   float64
				Count int
			}{
				Avg:   st.totalScores[id] / float64(len(entries)),
				Count: len(entries),
			}
		}
	}
	return result
}

// SelectBest selects the best agent using epsilon-greedy strategy.
//
// Parameters:
//   - candidates: available agent runtime metadata list
//
// Returns the selected agent index, or -1 if candidates is empty.
func (st *ScoreTracker) SelectBest(candidates []*core.AgentRuntimeMeta) int {
	n := len(candidates)
	if n == 0 {
		return -1
	}

	// Single candidate — always select it
	if n == 1 {
		return 0
	}

	st.mu.RLock()
	epsilon := st.epsilon
	st.mu.RUnlock()

	// Epsilon-greedy: with probability ε, explore (random); otherwise exploit (best score)
	if rand.Float64() < epsilon {
		return rand.Intn(n)
	}

	// Exploitation: pick highest scoring agent that has enough samples
	bestIdx := -1
	bestScore := -1.0

	for i, meta := range candidates {
		avg, count := st.GetScore(meta.ID())

		// Agents without enough data get a neutral initial score
		if count < MinSamplesForExploitation {
			avg = 1.5 // Neutral middle ground
		}

		if avg > bestScore {
			bestScore = avg
			bestIdx = i
		}
	}

	// Fallback if no scores at all — pick random among idle agents
	if bestIdx < 0 {
		return rand.Intn(n)
	}

	return bestIdx
}

// GetHistory returns the full score history for an agent (for debugging/analysis).
func (st *ScoreTracker) GetHistory(agentID string) []scoreEntry {
	st.mu.RLock()
	defer st.mu.RUnlock()

	entries := st.scores[agentID]
	if entries == nil {
		return nil
	}
	cp := make([]scoreEntry, len(entries))
	copy(cp, entries)
	return cp
}

// Reset clears all scores for an agent. Use with caution.
func (st *ScoreTracker) Reset(agentID string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.scores, agentID)
	delete(st.totalScores, agentID)
}
