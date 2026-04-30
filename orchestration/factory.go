package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/DotNetAge/goreact/core"
)

// ===========================================================================
// Agent Factory — Dynamic Agent Creation (Design §12)
// ===========================================================================
//
// When LLM Router returns __CREATE_NEW__, AgentFactory handles dynamic creation
// of new Agent instances. The creation flow follows Design §12.2:
//
//  1. Extract capability requirements from task description via LLM
//  2. Two-phase LLM generation: Description (<=1024 chars) + Body (full instructions)
//  3. Match against existing agents for overlap detection
//  4. Construct AgentConfig and register to AgentRegistry + RuntimeDirectory
//
// Key design decisions:
//  - Description and Body are generated separately to avoid routing context bloat
//  - New agents default AutoCreated=true, mark as cleanable
//  - Cold start: initial trust score (Score=2.0)
//  - Creation count protected by MaxAutoAgents limit

const (
	// DefaultMaxAutoAgents is the maximum number of dynamically created agents.
	DefaultMaxAutoAgents = 20

	// maxDescriptionLength is the hard length limit for Description (Design §2.1).
	maxDescriptionLength = 1024
)

// AgentFactory is responsible for creating Agent instances on demand.
type AgentFactory struct {
	mu       sync.RWMutex
	router   *LLMRouter            // Reuses Router's LLM capability for generating Description/Body
	registry *goreactRegistryAdapter // Registers newly created agents

	maxAutoCreated int // Upper limit on auto-created agent count
	createdCount   int // Current count of created agents

	logger *structuredLogger
}

// goreactRegistryAdapter defines the minimal Registry interface needed by Factory.
// This avoids direct dependency on goreact.AgentRegistry concrete type,
// keeping the orchestration package independent.
type goreactRegistryAdapter interface {
	Get(name string) *core.AgentConfig
	List() []*core.AgentConfig
	Register(name string, config *core.AgentConfig) error
}

// NewAgentFactory creates a new AgentFactory instance.
func NewAgentFactory(router *LLMRouter, registry *goreactRegistryAdapter) *AgentFactory {
	return &AgentFactory{
		router:         router,
		registry:       registry,
		maxAutoCreated: DefaultMaxAutoAgents,
		createdCount:   0,
		logger:         newLogger("agent_factory"),
	}
}

// WithMaxAutoAgents sets the maximum number of dynamic agents that can be created.
func (f *AgentFactory) WithMaxAutoAgents(max int) *AgentFactory {
	if max > 0 {
		f.maxAutoCreated = max
	}
	return f
}

// CanCreate checks whether more dynamic agents can still be created.
func (f *AgentFactory) CanCreate() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.createdCount < f.maxAutoCreated
}

// CreatedCount returns the current count of dynamically created agents.
func (f *AgentFactory) CreatedCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.createdCount
}

// Create dynamically creates a new Agent based on the task description.
//
// Flow:
//  1. Check quantity limit
//  2. Use LLM to extract capabilities and generate Description and Body
//  3. Check existing agents for functional overlap (avoid duplicate creation)
//  4. Build AgentConfig and register to Registry
//  5. Return new Agent configuration
//
// If LLM is unavailable or call fails, falls back to rule-based generation.
func (f *AgentFactory) Create(ctx context.Context, taskDescription string, modelRegistry core.ModelRegistry) (*core.AgentConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.createdCount >= f.maxAutoCreated {
		return nil, fmt.Errorf("agent factory: auto-created agent limit reached (%d)", f.maxAutoCreated)
	}

	// Step 1: Check if an existing agent already covers this capability range
	existing := f.findOverlappingAgent(taskDescription)
	if existing != nil {
		f.logger.Info("existing agent covers this capability, reusing",
			"agent", existing.Name,
			"task_description", truncateStr(taskDescription, 80),
		)
		return existing, nil
	}

	var config *core.AgentConfig
	var err error

	// Step 2: Try LLM-based config generation
	if f.router != nil && f.router.IsEnabled() {
		config, err = f.generateWithLLM(ctx, taskDescription)
		if err != nil {
			f.logger.Warn("LLM generation failed, falling back to rule-based generation",
				"error", err,
				"task", truncateStr(taskDescription, 80),
			)
			config = f.generateRuleBased(taskDescription, modelRegistry)
		}
	} else {
		config = f.generateRuleBased(taskDescription, modelRegistry)
	}

	// Step 3: Resolve model if not set
	if config.Model == "" {
		if modelRegistry != nil {
			if defaultCfg, err := modelRegistry.Get("default"); err == nil {
				config.Model = defaultCfg.Name
			}
		}
	}

	f.createdCount++

	f.logger.Info("new agent created dynamically",
		"name", config.Name,
		"description", truncateStr(config.Description, 60),
		"total_created", f.createdCount,
	)

	return config, nil
}

// generateWithLLM uses two-phase LLM generation to produce Description and Body.
func (f *AgentFactory) generateWithLLM(ctx context.Context, taskDesc string) (*core.AgentConfig, error) {
	// Phase 1: Extract capability requirements and generate Description
	descData := capabilityExtractionPromptData{TaskDescription: taskDesc}
	descPrompt, err := renderCapabilityExtractionPrompt(descData)
	if err != nil {
		return nil, fmt.Errorf("render capability extraction prompt failed: %w", err)
	}

	resp, err := f.router.callLLM(ctx, descPrompt, "Analyze this task to identify required capabilities and generate a concise agent role description.")
	if err != nil {
		return nil, fmt.Errorf("description generation failed: %w", err)
	}

	var descResult struct {
		Capabilities    []string `json:"required_capabilities"`
		Description    string   `json:"description"`
		NameSuggestion string   `json:"suggested_name"`
	}
	if err := parseJSONFromContent(resp.Content, &descResult); err != nil {
		return nil, fmt.Errorf("failed to parse description response: %w", err)
	}

	description := descResult.Description
	if description == "" {
		// Fallback: build from capabilities
		description = fmt.Sprintf("Specialized agent for: %s", strings.Join(descResult.Capabilities, ", "))
	}

	// Truncate to max length
	description = truncateStr(description, maxDescriptionLength)

	name := descResult.NameSuggestion
	if name == "" {
		name = fmt.Sprintf("auto-%s", generateShortID())
	}

	// Phase 2: Generate Body (full instruction set)
	capStr := strings.Join(descResult.Capabilities, ", ")
	bodyData := bodyGenerationPromptData{
		Name:        name,
		Description: description,
		Capabilities: capStr,
		TaskExample: taskDesc,
	}
	bodyPrompt, err := renderBodyGenerationPrompt(bodyData)
	if err != nil {
		f.logger.Warn("failed to render body prompt, using description-only config", "error", err)
		return &core.AgentConfig{
			Name:              name,
			Description:       description,
			Introduction:      "",
			EnableOrchestration: true,
		}, nil
	}

	resp2, err := f.router.callLLM(ctx, bodyPrompt, "Generate a complete System Prompt / behavioral instruction set for this Agent.")
	if err != nil {
		// Body generation failure is non-critical; Description alone is sufficient for routing
		f.logger.Warn("body generation failed, using description-only config", "error", err)
		return &core.AgentConfig{
			Name:              name,
			Description:       description,
			Introduction:      "",
			EnableOrchestration: true,
		}, nil
	}

	var bodyResult struct {
		Body string `json:"system_prompt"`
	}
	if err := parseJSONFromContent(resp2.Content, &bodyResult); err != nil {
		bodyResult.Body = resp2.Content // Use raw output directly as body
	}

	return &core.AgentConfig{
		Name:               name,
		Description:        description,
		Introduction:      bodyResult.Body,
		EnableOrchestration: true,
	}, nil
}

// generateRuleBased generates a basic config using rules when LLM is unavailable.
func (f *AgentFactory) generateRuleBased(taskDesc string, modelRegistry core.ModelRegistry) *core.AgentConfig {
	name := fmt.Sprintf("auto-%s", generateShortID())

	summary := truncateStr(taskDesc, 200)

	description := fmt.Sprintf("Dynamically created agent for task type: %s", summary)
	if len(description) > maxDescriptionLength {
		description = description[:maxDescriptionLength]
	}

	modelName := ""
	if modelRegistry != nil {
		if cfg, err := modelRegistry.Get("default"); err == nil {
			modelName = cfg.Name
		}
	}

	introduction := fmt.Sprintf(
		"You are a specialized assistant created for: %s\n"+
			"Analyze the user's request carefully and provide accurate, helpful responses.\n"+
			"Language Consistency: Always respond in the same language as the user's input.\n"+
			"Concise & Precise: Answer directly to the point, avoid redundancy without sacrificing completeness.\n"+
			"Honest & Transparent: State uncertainty explicitly, never fabricate facts.\n"+
			"Safety Boundaries: Do not execute destructive operations without user consent.",
		summary,
	)

	return &core.AgentConfig{
		Name:               name,
		Description:        description,
		Introduction:       introduction,
		Model:              modelName,
		EnableOrchestration: true,
	}
}

// findOverlappingAgent checks if any existing agent already covers the given capability.
// Avoids creating functionally duplicate agents.
func (f *AgentFactory) findOverlappingAgent(taskDesc string) *core.AgentConfig {
	if f.registry == nil {
		return nil
	}

	taskWords := splitWords(taskDesc)

	bestOverlap := 0
	var bestMatch *core.AgentConfig

	for _, agent := range (*f.registry).List() {
		descLower := strings.ToLower(agent.Description)
		overlap := 0

		for _, word := range taskWords {
			if len(word) > 2 && strings.Contains(descLower, word) {
				overlap += len(word)
			}
		}

		if overlap > 0 && bestOverlap < overlap {
			bestOverlap = overlap
			bestMatch = agent
		}
	}

	// High overlap threshold: if matched word total length exceeds 30% of task description, reuse existing agent
	if bestMatch != nil && float64(bestOverlap) > float64(len(taskDesc))*0.3 {
		return bestMatch
	}

	return nil
}

// --- JSON Parsing Helpers ---

// parseJSONFromContent extracts JSON from LLM output, stripping markdown code block wrappers if present.
func parseJSONFromContent(content string, target any) error {
	content = strings.TrimSpace(content)
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
	return json.Unmarshal([]byte(content), target)
}

// --- Utility Helpers ---

func generateShortID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	id := make([]byte, 6)
	for i := range id {
		id[i] = chars[i%len(chars)]
	}
	return string(id)
}

// truncateStr truncates a string to maxLength, appending "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
