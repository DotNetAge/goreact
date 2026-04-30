package orchestration

import (
	"math"
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
	// Design §8.4 recommends 0.3 initial, decaying to 0.05.
	DefaultEpsilon = 0.3

	// MinSamplesForExploitation is the minimum number of scores before switching
	// from exploration to exploitation mode for an agent.
	MinSamplesForExploitation = 3

	// DefaultInitialTrustScore is the initial performance score for newly created agents (Design §8.4).
	// Value 2.0 assumes a new agent is of medium quality until proven otherwise.
	DefaultInitialTrustScore = 2.0

	// DefaultScoreHalfLife is the half-life duration for time-decay weighting.
	// Scores older than this are weighted at 50% of recent scores.
	DefaultScoreHalfLife = 24 * time.Hour

	// DefaultEpsilonDecayRate is the per-decision reduction factor for epsilon.
	// Epsilon decays: epsilon = epsilon * (1 - decayRate) each selection cycle.
	DefaultEpsilonDecayRate = 0.005

	// MinEpsilon is the floor for epsilon exploration probability.
	MinEpsilon = 0.05
)

// AgentPerformance holds computed performance metrics for an agent (Design §8.2).
type AgentPerformance struct {
	AvgScore       float64   // weighted average score (time-decay)
	RawAvgScore    float64   // unweighted average for reference
	SampleCount    int       // total number of scores
	SuccessRate    float64   // percentage of successful executions
	LastScoreTime  time.Time // timestamp of most recent score
	LastUpdated    time.Time // last time performance was computed
}

// ScoreTracker tracks and manages agent performance scores using an
// epsilon-greedy strategy for agent selection during cold start and beyond.
type ScoreTracker struct {
	mu          sync.RWMutex
	scores      map[string][]scoreEntry // key: agent ID -> score history
	totalScores map[string]float64      // Running total for quick average
	epsilon     float64                 // Exploration probability
	halfLife    time.Duration           // Half-life for time-decay weighting
	decayRate   float64                 // Epsilon decay per selection
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
		halfLife:    DefaultScoreHalfLife,
		decayRate:   DefaultEpsilonDecayRate,
	}
}

// SetEpsilon updates the exploration probability.
func (st *ScoreTracker) SetEpsilon(eps float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.epsilon = eps
}

// SetHalfLife updates the time-decay half-life duration.
func (st *ScoreTracker) SetHalfLife(hl time.Duration) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.halfLife = hl
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

// decayWeight computes the time-decay weight for a score entry.
// Uses exponential decay: weight = 0.5 ^ (age / halfLife).
// Returns a value in (0, 1], where 1.0 is a just-recorded score.
func (st *ScoreTracker) decayWeight(entry scoreEntry) float64 {
	if st.halfLife <= 0 {
		return 1.0 // no decay
	}
	age := time.Since(entry.Timestamp)
	return math.Pow(0.5, float64(age)/float64(st.halfLife))
}

// GetScore returns the time-decay weighted average score and total count for an agent.
// Returns (0, 0) if no scores exist.
// Design §8.2: uses exponential decay weighting so recent performance weighs more.
func (st *ScoreTracker) GetScore(agentID string) (avg float64, count int) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	entries, ok := st.scores[agentID]
	if !ok || len(entries) == 0 {
		return 0, 0
	}

	var weightedSum, weightTotal float64
	for _, e := range entries {
		w := st.decayWeight(e)
		weightedSum += float64(e.Score) * w
		weightTotal += w
	}

	if weightTotal == 0 {
		return 0, len(entries)
	}
	return weightedSum / weightTotal, len(entries)
}

// GetPerformance returns the full AgentPerformance for an agent (Design §8.2).
func (st *ScoreTracker) GetPerformance(agentID string) AgentPerformance {
	st.mu.RLock()
	defer st.mu.RUnlock()

	entries, ok := st.scores[agentID]
	if !ok || len(entries) == 0 {
		return AgentPerformance{}
	}

	var weightedSum, weightTotal float64
	var rawSum float64
	var successCount int

	for _, e := range entries {
		w := st.decayWeight(e)
		weightedSum += float64(e.Score) * w
		weightTotal += w
		rawSum += float64(e.Score)
		if e.Success {
			successCount++
		}
	}

	weightedAvg := 0.0
	if weightTotal > 0 {
		weightedAvg = weightedSum / weightTotal
	}
	rawAvg := rawSum / float64(len(entries))

	return AgentPerformance{
		AvgScore:      weightedAvg,
		RawAvgScore:   rawAvg,
		SampleCount:   len(entries),
		SuccessRate:   float64(successCount) / float64(len(entries)),
		LastScoreTime: entries[len(entries)-1].Timestamp,
		LastUpdated:   time.Now(),
	}
}

// GetAllScores returns a map of all agent IDs to their (weighted avg, count).
func (st *ScoreTracker) GetAllScores() map[string]struct{ Avg float64; Count int } {
	st.mu.RLock()
	defer st.mu.RUnlock()

	result := make(map[string]struct{ Avg float64; Count int }, len(st.scores))
	for id, entries := range st.scores {
		if len(entries) == 0 {
			continue
		}

		var weightedSum, weightTotal float64
		for _, e := range entries {
			w := st.decayWeight(e)
			weightedSum += float64(e.Score) * w
			weightTotal += w
		}

		avg := 0.0
		if weightTotal > 0 {
			avg = weightedSum / weightTotal
		}
		result[id] = struct {
			Avg   float64
			Count int
		}{Avg: avg, Count: len(entries)}
	}
	return result
}

// SelectBest selects the best agent using epsilon-greedy strategy.
//
// Parameters:
//   - candidates: available agent runtime metadata list
//
// Returns the selected agent index, or -1 if candidates is empty.
// Epsilon decays with each call to gradually reduce exploration.
func (st *ScoreTracker) SelectBest(candidates []*core.AgentRuntimeMeta) int {
	n := len(candidates)
	if n == 0 {
		return -1
	}

	// Single candidate — always select it
	if n == 1 {
		return 0
	}

	st.mu.Lock()
	epsilon := st.epsilon
	// Decay epsilon: reduce exploration over time
	st.epsilon = math.Max(MinEpsilon, st.epsilon*(1-st.decayRate))
	st.mu.Unlock()

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

// DecayEpsilon reduces the exploration probability by one decay step.
// Called periodically to simulate learning over time.
func (st *ScoreTracker) DecayEpsilon() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.epsilon = math.Max(MinEpsilon, st.epsilon*(1-st.decayRate))
}
