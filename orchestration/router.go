package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DotNetAge/gochat"
	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// ===========================================================================
// LLM Router — Intelligent Routing Engine (Design §6.3)
// ===========================================================================
//
// LLM Router is the core decision component of the orchestrator, responsible
// for semantically matching tasks to the most suitable Agent.
//
// Design inspired by Progressive Disclosure Level 1, but applied to Agents
// instead of Tools/Skills.
//
// Workflow:
//  1. Extract lightweight metadata from RuntimeDirectory for all Active Agents
//     (Name + Description + State + Score + TaskCount) — NEVER loads Body
//  2. Inject System Prompt via Go template (similar to Level 1 lightweight routing)
//  3. Single LLM call → RoutingDecision{SelectedAgent, Reasoning, Confidence}
//  4. If confidence too low or no match → return __CREATE_NEW__ to trigger dynamic creation
//
// Token budget: Even with 50 agents, total Descriptions is ~50KB,
// negligible burden on the Router's context window.

const (
	// CreateNewAgent is the special agent name returned by LLM Router when
	// no existing agent matches the task. Triggers AgentFactory dynamic creation.
	CreateNewAgent = "__CREATE_NEW__"

	// defaultRouterModelKey is the ModelRegistry key for looking up a router-specific model.
	defaultRouterModelKey = "router"

	// routeCacheTTL defines the validity duration for cached routing results.
	routeCacheTTL = 10 * time.Minute

	// minConfidenceThreshold is the minimum confidence to accept a routing decision.
	minConfidenceThreshold = 0.4

	// defaultRouterMaxTokens is the max tokens for routing LLM responses.
	defaultRouterMaxTokens = 512

	// defaultRouterTemperature is the temperature for routing (low = deterministic).
	defaultRouterTemperature = 0.1
)

// RoutingDecision is the output of the LLM Router.
type RoutingDecision struct {
	SelectedAgent string  // Selected agent name, or "__CREATE_NEW__" for dynamic creation
	Reasoning     string  // LLM's reasoning for the selection
	Confidence    float64 // Confidence 0.0-1.0
	Cached        bool    // Whether this result came from cache
}

// RouteRequest encapsulates all input for a single routing request.
type RouteRequest struct {
	TaskDescription   string // Task description to be routed (required)
	DesiredCapability string // Optional capability hint for the router
	SourceAgentID     string // Requesting agent ID (for logging/tracing)
}

// ===========================================================================
// Router Interface — abstracts LLM-based routing for dependency inversion
// ===========================================================================

// Router is the interface that LLM-based task routing must implement.
type Router interface {
	// Route selects the best agent for a task from the given candidates.
	Route(ctx context.Context, req RouteRequest, agents []*core.AgentRuntimeMeta) (*RoutingDecision, error)
}

// LLMRouter is the intelligent routing engine instance of the orchestrator.
// It implements the Router interface.
type LLMRouter struct {
	mu       sync.RWMutex
	client   gochat.ClientBuilder // LLM client builder for making calls
	modelCfg *core.ModelConfig   // Router-specific model configuration

	// Cache layer: avoid repeated LLM calls for identical tasks
	cache      map[string]*routeCacheEntry
	cacheMu    sync.RWMutex
	maxCache   int // Max cache entries (default 256)

	logger *structuredLogger
}

// routeCacheEntry is a cache entry with TTL-based expiration.
type routeCacheEntry struct {
	decision *RoutingDecision
	expiry   time.Time
}

// NewLLMRouter creates a new LLM Router instance.
// modelCfg is used to build the LLM client (API Key / BaseURL etc.).
// If nil, the Router is in disabled state (all routing falls back to keyword matching).
func NewLLMRouter(modelCfg *core.ModelConfig) (*LLMRouter, error) {
	if modelCfg == nil || modelCfg.APIKey == "" {
		return &LLMRouter{
			cache:    make(map[string]*routeCacheEntry),
			maxCache: 256,
			logger:   newLogger("llm_router"),
		}, nil
	}

	client := gochat.Client().Config(
		gochat.WithAPIKey(modelCfg.APIKey),
		gochat.WithBaseURL(modelCfg.BaseURL),
	)

	return &LLMRouter{
		client:   client,
		modelCfg: modelCfg,
		cache:    make(map[string]*routeCacheEntry),
		maxCache: 256,
		logger:   newLogger("llm_router"),
	}, nil
}

// IsEnabled returns whether the Router has a valid LLM configuration and can make calls.
func (r *LLMRouter) IsEnabled() bool {
	return r.client != nil && r.modelCfg != nil
}

// Route executes intelligent routing: selects the best agent for a given task.
//
// Parameters:
//   - ctx: Cancellable context
//   - req: Task description and optional capability hint
//   - agents: All candidate agents' runtime metadata (from RuntimeDirectory.ListActive())
//
// Returns RoutingDecision. If Router is disabled or LLM call fails, falls back gracefully.
func (r *LLMRouter) Route(ctx context.Context, req RouteRequest, agents []*core.AgentRuntimeMeta) (*RoutingDecision, error) {
	if len(agents) == 0 {
		return &RoutingDecision{
			SelectedAgent: CreateNewAgent,
			Reasoning:     "no registered agents available",
			Confidence:    1.0,
		}, nil
	}

	// Check cache first
	cacheKey := req.TaskDescription + "|" + req.DesiredCapability
	if cached := r.getFromCache(cacheKey); cached != nil {
		cached.Cached = true
		return cached, nil
	}

	// Router disabled → fallback to keyword matching
	if !r.IsEnabled() {
		return r.fallbackRoute(req, agents), nil
	}

	// Build prompt using template and call LLM
	systemPrompt, err := r.buildRoutingPrompt(agents, req)
	if err != nil {
		r.logger.Warn("failed to build routing prompt, falling back to keyword matching", "error", err)
		return r.fallbackRoute(req, agents), nil
	}

	resp, err := r.callLLM(ctx, systemPrompt, req.TaskDescription)
	if err != nil {
		r.logger.Warn("LLM call failed, falling back to keyword matching", "error", err)
		return r.fallbackRoute(req, agents), nil
	}

	decision, err := r.parseRoutingResponse(resp.Content)
	if err != nil {
		r.logger.Warn("failed to parse routing response, falling back to keyword matching", "error", err)
		return r.fallbackRoute(req, agents), nil
	}

	decision.Cached = false
	r.putToCache(cacheKey, decision)

	r.logger.Info("routing decision made",
		"selected_agent", decision.SelectedAgent,
		"confidence", decision.Confidence,
		"reasoning", decision.Reasoning,
		"cached", false,
	)

	return decision, nil
}

// callLLM executes a non-streaming LLM call.
func (r *LLMRouter) callLLM(ctx context.Context, systemPrompt, userMessage string) (*gochatcore.Response, error) {
	builder := r.client.Model(r.modelCfg.Name).
		Temperature(defaultRouterTemperature).
		MaxTokens(defaultRouterMaxTokens)

	if r.modelCfg.TopP > 0 {
		builder.TopP(r.modelCfg.TopP)
	}

	builder.SystemMessage(systemPrompt)
	builder.UserMessage(userMessage)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return builder.GetResponseFor(gochat.QwenClient)
}

// buildRoutingPrompt builds the routing decision system prompt using the embedded Go template.
// Follows Design §6.3 prompt template specification.
func (r *LLMRouter) buildRoutingPrompt(agents []*core.AgentRuntimeMeta, req RouteRequest) (string, error) {
	data := routingPromptData{
		Agents:            toAgentViews(agents),
		TaskDescription:   req.TaskDescription,
		DesiredCapability: req.DesiredCapability,
	}
	return renderRoutingPrompt(data)
}

// parseRoutingResponse parses the LLM's JSON output into a RoutingDecision.
func (r *LLMRouter) parseRoutingResponse(content string) (*RoutingDecision, error) {
	content = strings.TrimSpace(content)
	// Strip possible markdown JSON code block markers
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		var cleaned []string
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "```") {
				break
			}
			cleaned = append(cleaned, line)
		}
		content = strings.TrimSpace(strings.Join(cleaned, "\n"))
	}

	var raw struct {
		SelectedAgent string  `json:"selected_agent"`
		Reasoning     string  `json:"reasoning"`
		Confidence    float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("invalid routing response JSON: %w", err)
	}

	// Validate confidence range
	if raw.Confidence < 0 || raw.Confidence > 1.0 {
		raw.Confidence = 0.5 // Default medium confidence
	}

	decision := &RoutingDecision{
		SelectedAgent: raw.SelectedAgent,
		Reasoning:     raw.Reasoning,
		Confidence:    raw.Confidence,
	}

	// If confidence below threshold and not CREATE_NEW, fall back to CREATE_NEW
	if decision.Confidence < minConfidenceThreshold && decision.SelectedAgent != CreateNewAgent {
		decision.SelectedAgent = CreateNewAgent
		decision.Reasoning += fmt.Sprintf(" [confidence %.2f below threshold %.2f, suggest creating new agent",
			decision.Confidence, minConfidenceThreshold)
	}

	return decision, nil
}

// fallbackRoute provides rule-based degradation routing when LLM is unavailable.
// Implements a three-level fallback strategy per Design §6.3.
func (r *LLMRouter) fallbackRoute(req RouteRequest, agents []*core.AgentRuntimeMeta) *RoutingDecision {
	// Strategy 1: Description keyword matching with performance weighting
	bestMatch := ""
	bestScore := 0.0
	taskLower := strings.ToLower(req.TaskDescription)
	desiredLower := strings.ToLower(req.DesiredCapability)

	for _, agent := range agents {
		if !agent.IsAvailable() {
			continue
		}

		descLower := strings.ToLower(agent.Description())
		score := 0.0

		// Simple keyword hit scoring
		for _, keyword := range splitWords(taskLower) {
			if len(keyword) > 2 && strings.Contains(descLower, keyword) {
				score += float64(len(keyword)) // Longer words get higher weight
			}
		}
		if desiredLower != "" {
			for _, keyword := range splitWords(desiredLower) {
				if len(keyword) > 2 && strings.Contains(descLower, keyword) {
					score += float64(len(keyword)) * 1.5 // Desired capability match bonus
				}
			}
		}

		// Performance score bonus
		score += agent.Score * 0.5

		if score > bestScore {
			bestScore = score
			bestMatch = agent.ID()
		}
	}

	if bestMatch != "" && bestScore > 1.0 {
		return &RoutingDecision{
			SelectedAgent: bestMatch,
			Reasoning:     fmt.Sprintf("keyword-based fallback match (score=%.2f)", bestScore),
			Confidence:    minF64(bestScore/10.0, 0.7), // Fallback routing has lower max confidence
			Cached:        false,
		}
	}

	// Strategy 2: Select highest-scored idle agent
	var bestAgent *core.AgentRuntimeMeta
	for _, agent := range agents {
		if !agent.IsAvailable() {
			continue
		}
		if bestAgent == nil || agent.Score > bestAgent.Score || (agent.Score == bestAgent.Score && agent.LastActive.After(bestAgent.LastActive)) {
			bestAgent = agent
		}
	}

	if bestAgent != nil {
		return &RoutingDecision{
			SelectedAgent: bestAgent.ID(),
			Reasoning:     "fallback: selected highest-scored available agent",
			Confidence:    0.3,
			Cached:        false,
		}
	}

	// Strategy 3: No available agent → suggest creating new one
	return &RoutingDecision{
		SelectedAgent: CreateNewAgent,
		Reasoning:     "no available agents matched, recommend creating new one",
		Confidence:    1.0,
		Cached:        false,
	}
}

// --- Cache management ---

func (r *LLMRouter) getFromCache(key string) *RoutingDecision {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	entry, ok := r.cache[key]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiry) {
		return nil // Expired
	}

	cp := *entry.decision
	return &cp
}

func (r *LLMRouter) putToCache(key string, decision *RoutingDecision) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	// Evict expired entries
	now := time.Now()
	for k, v := range r.cache {
		if now.After(v.expiry) {
			delete(r.cache, k)
		}
	}

	// Enforce size limit
	if len(r.cache) >= r.maxCache {
		// Remove oldest entry (simple FIFO)
		var oldestKey string
		var oldestExpiry time.Time
		for k, v := range r.cache {
			if oldestKey == "" || v.expiry.Before(oldestExpiry) {
				oldestKey = k
				oldestExpiry = v.expiry
			}
		}
		if oldestKey != "" {
			delete(r.cache, oldestKey)
		}
	}

	cp := *decision
	r.cache[key] = &routeCacheEntry{decision: &cp, expiry: now.Add(routeCacheTTL)}
}

// ClearCache clears all routing cache entries (for testing).
func (r *LLMRouter) ClearCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache = make(map[string]*routeCacheEntry)
}

// CacheSize returns the current number of cache entries (for monitoring).
func (r *LLMRouter) CacheSize() int {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return len(r.cache)
}

// --- Multi-factor Ranking (Design §8.5) ---

// RankedAgent represents an agent with its composite ranking score.
type RankedAgent struct {
	Agent *core.AgentRuntimeMeta
	Score float64
}

// rankAgents ranks candidates using multi-factor scoring per Design §8.5:
//   - Factor 1: Performance score (weight 40%)
//   - Factor 2: Semantic match from LLM Router confidence (weight 30%)
//   - Factor 3: Current availability/idleness (weight 20%)
//   - Factor 4: Recent activity (weight 10%, 30-day linear decay)
func (r *LLMRouter) rankAgents(candidates []*core.AgentRuntimeMeta, taskDesc string) []*RankedAgent {
	if len(candidates) == 0 {
		return nil
	}

	now := time.Now()
	ranked := make([]*RankedAgent, 0, len(candidates))

	for _, agent := range candidates {
		score := 0.0

		// Factor 1: Performance score (40%)
		perfScore := agent.Score // Already 0-3 scale normalized by ScoreTracker
		// Normalize to 0-1 range assuming max score of 3
		score += (perfScore / 3.0) * 0.4

		// Factor 2: Keyword-based semantic approximation (30%)
		// In full LLM mode, this would be replaced by LLM Router's confidence value
		taskLower := strings.ToLower(taskDesc)
		descLower := strings.ToLower(agent.Description())
		matchScore := 0.0
		matchCount := 0
		for _, kw := range splitWords(taskLower) {
			if len(kw) > 2 && strings.Contains(descLower, kw) {
				matchScore += float64(len(kw))
				matchCount++
			}
		}
		// Normalize: cap at a reasonable maximum
		if matchScore > 100 {
			matchScore = 100
		}
		score += (matchScore / 100.0) * 0.3

		// Factor 3: Availability (20%)
		if agent.State == core.AgentStateIdle {
			score += 1.0 * 0.2
		} else {
			score += 0.3 * 0.2 // Busy but still selectable
		}

		// Factor 4: Recent activity (10%) — 30-day linear decay
		daysInactive := now.Sub(agent.LastActive).Hours() / 24
		activityScore := 1.0 - minF64(daysInactive/30.0, 1.0)
		if activityScore < 0 {
			activityScore = 0
		}
		score += activityScore * 0.1

		ranked = append(ranked, &RankedAgent{Agent: agent, Score: score})
	}

	// Sort descending by score
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].Score > ranked[i].Score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	return ranked
}

// SelectBest uses epsilon-greedy strategy (Design §8.4) to select from ranked candidates.
// During cold start, it explores new agents with probability epsilon.
// Falls back to ScoreTracker's built-in SelectBest when available.
func (r *LLMRouter) SelectBest(candidates []*core.AgentRuntimeMeta, taskDesc string, tracker *ScoreTracker) *core.AgentRuntimeMeta {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Use ScoreTracker's epsilon-greedy selection if available
	if tracker != nil && len(candidates) > 0 {
		idx := tracker.SelectBest(candidates)
		if idx >= 0 && idx < len(candidates) {
			return candidates[idx]
		}
	}

	// Fallback: use multi-factor ranking without epsilon-greedy
	ranked := r.rankAgents(candidates, taskDesc)
	return ranked[0].Agent
}

// --- Helpers ---

func splitWords(s string) []string {
	s = strings.ToLower(s)
	var words []string
	current := strings.Builder{}
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			current.WriteRune(ch)
		} else if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minF64 returns the smaller of two float64 values.
func minF64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// --- Structured logger (avoids importing slog to keep orchestration self-contained) ---

type structuredLogger struct {
	prefix string
}

func newLogger(prefix string) *structuredLogger { return &structuredLogger{prefix: prefix} }

func (l *structuredLogger) Info(msg string, pairs ...any) {
	fmt.Printf("[INFO] [%s] %s %+v\n", l.prefix, msg, pairs)
}

func (l *structuredLogger) Warn(msg string, pairs ...any) {
	fmt.Printf("[WARN] [%s] %s %+v\n", l.prefix, msg, pairs)
}

func (l *structuredLogger) Debug(msg string, pairs ...any) {
	fmt.Printf("[DEBUG] [%s] %s %+v\n", l.prefix, msg, pairs)
}
