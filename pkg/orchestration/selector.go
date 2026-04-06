package orchestration

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Agent Selector Implementation
// =============================================================================

// AgentSelector implements Selector interface
type AgentSelector struct {
	matcher       CapabilityMatcher
	loadBalancer  LoadBalancer
	config        *SelectorConfig
	loadInfo      map[string]*LoadInfo
	loadInfoMu    sync.RWMutex
}

// SelectorConfig represents selector configuration
type SelectorConfig struct {
	CapabilityWeight float64 `json:"capability_weight"` // Default 0.5
	LoadWeight       float64 `json:"load_weight"`       // Default 0.3
	HistoryWeight    float64 `json:"history_weight"`    // Default 0.2
	MaxCandidates    int     `json:"max_candidates"`    // Max candidates to consider
}

// DefaultSelectorConfig returns default selector config
func DefaultSelectorConfig() *SelectorConfig {
	return &SelectorConfig{
		CapabilityWeight: 0.5,
		LoadWeight:       0.3,
		HistoryWeight:    0.2,
		MaxCandidates:    10,
	}
}

// NewAgentSelector creates a new agent selector
func NewAgentSelector(config *SelectorConfig) *AgentSelector {
	if config == nil {
		config = DefaultSelectorConfig()
	}
	return &AgentSelector{
		matcher:      NewDefaultCapabilityMatcher(),
		loadBalancer: NewDefaultLoadBalancer(),
		config:       config,
		loadInfo:     make(map[string]*LoadInfo),
	}
}

// Select selects the best agent for a single sub-task
func (s *AgentSelector) Select(subTask *SubTask, candidates []Agent) (Agent, error) {
	if len(candidates) == 0 {
		return nil, NewOrchestrationError(ErrorAgentSelectionFailed, "no candidates available", nil)
	}

	// Score all candidates
	matches := s.scoreCandidates(subTask, candidates)
	
	if len(matches) == 0 {
		return nil, NewOrchestrationError(ErrorAgentSelectionFailed,
			"no suitable agent found for sub-task", nil)
	}

	// Sort by total score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].TotalScore > matches[j].TotalScore
	})

	// Return the best match
	for _, m := range matches {
		for _, c := range candidates {
			if c.Name() == m.AgentName {
				return c, nil
			}
		}
	}

	return nil, NewOrchestrationError(ErrorAgentSelectionFailed,
		"failed to find agent for sub-task", nil)
}

// SelectBatch selects agents for multiple sub-tasks
func (s *AgentSelector) SelectBatch(subTasks []*SubTask, candidates []Agent) (map[string]Agent, error) {
	result := make(map[string]Agent)
	
	for _, st := range subTasks {
		agent, err := s.Select(st, candidates)
		if err != nil {
			return nil, err
		}
		result[st.Name] = agent
	}
	
	return result, nil
}

// Capabilities returns the capabilities of an agent
func (s *AgentSelector) Capabilities(agent Agent) (*Capabilities, error) {
	return agent.Capabilities(), nil
}

// UpdateLoadInfo updates load information for an agent
func (s *AgentSelector) UpdateLoadInfo(agentName string, info *LoadInfo) {
	s.loadInfoMu.Lock()
	defer s.loadInfoMu.Unlock()
	info.LastUpdateTime = time.Now()
	s.loadInfo[agentName] = info
}

// GetLoadInfo gets load information for an agent
func (s *AgentSelector) GetLoadInfo(agentName string) *LoadInfo {
	s.loadInfoMu.RLock()
	defer s.loadInfoMu.RUnlock()
	return s.loadInfo[agentName]
}

// scoreCandidates scores all candidates for a sub-task
func (s *AgentSelector) scoreCandidates(subTask *SubTask, candidates []Agent) []*AgentMatch {
	matches := []*AgentMatch{}
	
	requiredCaps := &Capabilities{
		Skills: subTask.RequiredCapabilities,
	}
	
	for _, agent := range candidates {
		agentCaps := agent.Capabilities()
		
		// Calculate capability score
		capScore := s.matcher.Match(requiredCaps, agentCaps)
		
		// Skip if no match
		if capScore == 0 {
			continue
		}
		
		// Calculate load score
		loadInfo := s.GetLoadInfo(agent.Name())
		loadScore := s.calculateLoadScore(loadInfo)
		
		// Calculate history score (simplified)
		historyScore := s.calculateHistoryScore(agent.Name())
		
		// Calculate total score
		totalScore := s.config.CapabilityWeight*capScore +
			s.config.LoadWeight*loadScore +
			s.config.HistoryWeight*historyScore
		
		matches = append(matches, &AgentMatch{
			AgentName:       agent.Name(),
			CapabilityScore: capScore,
			LoadScore:       loadScore,
			HistoryScore:    historyScore,
			TotalScore:      totalScore,
		})
	}
	
	return matches
}

// calculateLoadScore calculates score based on load (higher score = lower load)
func (s *AgentSelector) calculateLoadScore(info *LoadInfo) float64 {
	if info == nil {
		return 0.5 // Default score when no info
	}
	
	// Calculate load score inversely proportional to load
	loadFactor := float64(info.ActiveTasks+info.QueueLength) / 10.0
	if loadFactor > 1 {
		loadFactor = 1
	}
	
	cpuFactor := info.CPUPercent / 100.0
	memFactor := info.MemoryPercent / 100.0
	
	// Combined load (lower is better)
	combinedLoad := (loadFactor + cpuFactor + memFactor) / 3.0
	
	// Score is inverse of load
	return 1.0 - combinedLoad
}

// calculateHistoryScore calculates score based on historical success
func (s *AgentSelector) calculateHistoryScore(agentName string) float64 {
	// Simplified: would track actual success/failure history
	return 0.5
}

// =============================================================================
// Capability Matcher Implementation
// =============================================================================

// DefaultCapabilityMatcher implements CapabilityMatcher
type DefaultCapabilityMatcher struct{}

// NewDefaultCapabilityMatcher creates a default capability matcher
func NewDefaultCapabilityMatcher() *DefaultCapabilityMatcher {
	return &DefaultCapabilityMatcher{}
}

// Match returns a score for capability matching (0.0 - 1.0)
func (m *DefaultCapabilityMatcher) Match(required, provided *Capabilities) float64 {
	if required == nil || provided == nil {
		return 0.0
	}
	
	// Match skills
	skillScore := m.matchSkills(required.Skills, provided.Skills)
	
	// Match tools
	toolScore := m.matchLists(required.Tools, provided.Tools)
	
	// Match domains
	domainScore := m.matchLists(required.Domains, provided.Domains)
	
	// Match languages
	langScore := m.matchLists(required.Languages, provided.Languages)
	
	// Check complexity
	complexityScore := 1.0
	if required.MaxComplexity > 0 && provided.MaxComplexity > 0 {
		if provided.MaxComplexity < required.MaxComplexity {
			complexityScore = float64(provided.MaxComplexity) / float64(required.MaxComplexity)
		}
	}
	
	// Weighted average
	totalScore := (skillScore*0.4 + toolScore*0.2 + domainScore*0.2 + langScore*0.1 + complexityScore*0.1)
	
	return totalScore
}

// matchSkills matches skills with semantic similarity
func (m *DefaultCapabilityMatcher) matchSkills(required, provided []string) float64 {
	if len(required) == 0 {
		return 1.0
	}
	
	matched := 0
	for _, req := range required {
		for _, prov := range provided {
			score := m.skillMatchScore(req, prov)
			if score >= 0.7 {
				matched++
				break
			}
		}
	}
	
	return float64(matched) / float64(len(required))
}

// skillMatchScore returns a score for matching two skills
func (m *DefaultCapabilityMatcher) skillMatchScore(req, prov string) float64 {
	req = strings.ToLower(req)
	prov = strings.ToLower(prov)
	
	// Exact match
	if req == prov {
		return 1.0
	}
	
	// Contains match
	if strings.Contains(prov, req) || strings.Contains(req, prov) {
		return 0.9
	}
	
	// Similarity-based match (simplified)
	// Would use embeddings in production
	return m.stringSimilarity(req, prov)
}

// stringSimilarity calculates string similarity using Levenshtein-like approach
func (m *DefaultCapabilityMatcher) stringSimilarity(a, b string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	
	// Simple character overlap similarity
	aChars := make(map[rune]bool)
	for _, c := range a {
		aChars[c] = true
	}
	
	common := 0
	for _, c := range b {
		if aChars[c] {
			common++
		}
	}
	
	return float64(common) / math.Max(float64(len(a)), float64(len(b)))
}

// matchLists matches two string lists
func (m *DefaultCapabilityMatcher) matchLists(required, provided []string) float64 {
	if len(required) == 0 {
		return 1.0
	}
	
	matched := 0
	for _, req := range required {
		for _, prov := range provided {
			if strings.EqualFold(req, prov) {
				matched++
				break
			}
		}
	}
	
	return float64(matched) / float64(len(required))
}

// =============================================================================
// Load Balancer Implementation
// =============================================================================

// DefaultLoadBalancer implements LoadBalancer
type DefaultLoadBalancer struct {
	config *LoadBalancerConfig
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Strategy LoadBalanceStrategy `json:"strategy"`
}

// LoadBalanceStrategy defines load balancing strategy
type LoadBalanceStrategy string

const (
	StrategyLeastTasks    LoadBalanceStrategy = "least_tasks"
	StrategyShortestQueue LoadBalanceStrategy = "shortest_queue"
	StrategyWeightedRound LoadBalanceStrategy = "weighted_round"
	StrategyRandom        LoadBalanceStrategy = "random"
)

// DefaultLoadBalancerConfig returns default config
func DefaultLoadBalancerConfig() *LoadBalancerConfig {
	return &LoadBalancerConfig{
		Strategy: StrategyLeastTasks,
	}
}

// NewDefaultLoadBalancer creates a default load balancer
func NewDefaultLoadBalancer() *DefaultLoadBalancer {
	return &DefaultLoadBalancer{config: DefaultLoadBalancerConfig()}
}

// Select selects an agent based on load balancing strategy
func (b *DefaultLoadBalancer) Select(candidates []Agent, loads map[string]*LoadInfo) (Agent, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}
	
	switch b.config.Strategy {
	case StrategyLeastTasks:
		return b.selectLeastTasks(candidates, loads)
	case StrategyShortestQueue:
		return b.selectShortestQueue(candidates, loads)
	case StrategyWeightedRound:
		return b.selectWeightedRound(candidates, loads)
	case StrategyRandom:
		return b.selectRandom(candidates)
	default:
		return candidates[0], nil
	}
}

// selectLeastTasks selects agent with least active tasks
func (b *DefaultLoadBalancer) selectLeastTasks(candidates []Agent, loads map[string]*LoadInfo) (Agent, error) {
	var best Agent
	minTasks := math.MaxInt32
	
	for _, agent := range candidates {
		load := loads[agent.Name()]
		tasks := 0
		if load != nil {
			tasks = load.ActiveTasks
		}
		
		if tasks < minTasks {
			minTasks = tasks
			best = agent
		}
	}
	
	return best, nil
}

// selectShortestQueue selects agent with shortest queue
func (b *DefaultLoadBalancer) selectShortestQueue(candidates []Agent, loads map[string]*LoadInfo) (Agent, error) {
	var best Agent
	minQueue := math.MaxInt32
	
	for _, agent := range candidates {
		load := loads[agent.Name()]
		queue := 0
		if load != nil {
			queue = load.QueueLength
		}
		
		if queue < minQueue {
			minQueue = queue
			best = agent
		}
	}
	
	return best, nil
}

// selectWeightedRound selects using weighted round robin
func (b *DefaultLoadBalancer) selectWeightedRound(candidates []Agent, loads map[string]*LoadInfo) (Agent, error) {
	// Calculate weights based on load
	weights := make([]float64, len(candidates))
	totalWeight := 0.0
	
	for i, agent := range candidates {
		load := loads[agent.Name()]
		weight := 1.0
		if load != nil {
			// Lower load = higher weight
			weight = 1.0 / (1.0 + float64(load.ActiveTasks+load.QueueLength))
		}
		weights[i] = weight
		totalWeight += weight
	}
	
	// Normalize and select
	target := 0.5 // Middle selection
	cumWeight := 0.0
	for i, agent := range candidates {
		cumWeight += weights[i] / totalWeight
		if cumWeight >= target {
			return agent, nil
		}
	}
	
	return candidates[0], nil
}

// selectRandom selects a random agent
func (b *DefaultLoadBalancer) selectRandom(candidates []Agent) (Agent, error) {
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	// Simple selection - would use crypto/rand in production
	return candidates[len(candidates)/2], nil
}

// =============================================================================
// Health Checker Implementation
// =============================================================================

// DefaultHealthChecker implements HealthChecker
type DefaultHealthChecker struct {
	timeout time.Duration
}

// NewDefaultHealthChecker creates a default health checker
func NewDefaultHealthChecker(timeout time.Duration) *DefaultHealthChecker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &DefaultHealthChecker{timeout: timeout}
}

// Check checks the health of an agent
func (h *DefaultHealthChecker) Check(agent Agent) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	
	// Simple health check by getting capabilities
	select {
	case <-ctx.Done():
		return fmt.Errorf("health check timeout for agent %s", agent.Name())
	default:
		caps := agent.Capabilities()
		if caps == nil {
			return fmt.Errorf("agent %s returned nil capabilities", agent.Name())
		}
		return nil
	}
}

// CheckAll checks the health of all agents
func (h *DefaultHealthChecker) CheckAll(agents []Agent) map[string]error {
	results := make(map[string]error)
	var mu sync.Mutex
	
	var wg sync.WaitGroup
	for _, agent := range agents {
		wg.Add(1)
		go func(a Agent) {
			defer wg.Done()
			err := h.Check(a)
			mu.Lock()
			results[a.Name()] = err
			mu.Unlock()
		}(agent)
	}
	
	wg.Wait()
	return results
}
