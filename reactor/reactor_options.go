package reactor

import (
	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// ReactorOption configures a Reactor during creation.
type ReactorOption func(*reactorSetup)

// WithExtraTools adds additional tools to the reactor beyond the bundled ones.
func WithExtraTools(tools ...core.FuncTool) ReactorOption {
	return func(s *reactorSetup) {
		s.extraTools = append(s.extraTools, tools...)
	}
}

// WithoutBundledTools skips registration of all built-in tools (orchestration tools are still registered).
func WithoutBundledTools() ReactorOption {
	return func(s *reactorSetup) {
		s.skipAllBundled = true
	}
}

// WithoutTool skips registration of a specific built-in tool by name.
func WithoutTool(name string) ReactorOption {
	return func(s *reactorSetup) {
		if s.skipTools == nil {
			s.skipTools = make(map[string]bool)
		}
		s.skipTools[name] = true
	}
}

// WithResultLimits configures tool result size thresholds (second layer defense).
func WithResultLimits(limits core.ToolResultLimits) ReactorOption {
	return func(s *reactorSetup) {
		s.resultLimits = limits
	}
}

// WithTokenEstimator sets a custom token estimator for budget tracking.
func WithTokenEstimator(estimator core.TokenEstimator) ReactorOption {
	return func(s *reactorSetup) {
		s.tokenEstimator = estimator
	}
}

// WithEventBus sets the event bus for streaming agent events.
// If not set, a new InProcessEventBus is created automatically.
func WithEventBus(bus EventBus) ReactorOption {
	return func(s *reactorSetup) {
		s.eventBus = bus
	}
}

// WithMCPRegistry sets an MCP tool registry for discovering and calling
// tools from external MCP servers.
func WithMCPRegistry(registry *core.MCPToolRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.mcpRegistry = registry
	}
}

// WithSkillDir specifies external directories to load skills from.
// Each directory should contain subdirectories, each with a SKILL.md file.
// Skills loaded from these directories are registered in addition to bundled skills.
// Multiple directories can be specified by calling WithSkillDir multiple times.
func WithSkillDir(dir string) ReactorOption {
	return func(s *reactorSetup) {
		s.skillDirs = append(s.skillDirs, dir)
	}
}

// WithoutBundledSkills skips registration of all built-in bundled skills.
func WithoutBundledSkills() ReactorOption {
	return func(s *reactorSetup) {
		s.skipBundledSkills = true
	}
}

// WithSkills specifies which skills to load. If empty, all skills are loaded.
// If specified, only skills with matching names will be loaded.
// This applies to both bundled skills and skills loaded from skill directories.
func WithSkills(skillNames ...string) ReactorOption {
	return func(s *reactorSetup) {
		s.skills = append(s.skills, skillNames...)
	}
}

// WithMessageBus sets an AgentMessageBus for inter-agent team communication.
// SubAgents spawned with a team_name will join teams and can communicate
// via send_message/receive_messages tools. The bus is shared across the
// main reactor and all subagent tasks.
func WithMessageBus(bus *core.AgentMessageBus) ReactorOption {
	return func(s *reactorSetup) {
		s.messageBus = bus
	}
}

// WithMemory sets a Memory implementation for knowledge retrieval.
// Memory is queried during the Think phase to inject relevant knowledge
// into the LLM prompt, suppressing hallucination.
// If not set, the reactor operates without memory augmentation.
func WithMemory(mem core.Memory) ReactorOption {
	return func(s *reactorSetup) {
		s.memory = mem
	}
}

// MockLLMFunc is the signature for a mock LLM function used in testing.
// When provided via WithMockLLM, the reactor delegates all LLM calls
// to this function instead of the real API client.
type MockLLMFunc func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error)

// WithMockLLM replaces the real LLM client with a deterministic mock function.
// This is intended for end-to-end testing without requiring real API keys or network access.
// The mock function receives the full prompt context (system prompt, user message, history)
// and must return a complete LLM response.
func WithMockLLM(fn MockLLMFunc) ReactorOption {
	return func(s *reactorSetup) {
		s.mockLLM = fn
	}
}

func WithSystemPrompt(prompt string) ReactorOption {
	return func(rs *reactorSetup) {
		rs.systemPrompt = prompt
	}
}

// --- Registry Injection Options ---

// WithIntentRegistry sets a custom IntentRegistry implementation.
// Use this to provide LLM-based intent classification, custom intent types, etc.
// If not set, DefaultIntentRegistry with built-in definitions is used automatically.
//
// Example: embedding-enhanced semantic intent matching:
//
//	type SemanticIntentRegistry struct {
//	    *reactor.DefaultIntentRegistry
//	    embedder *embedding.Client
//	}
//	func (s *SemanticIntentRegistry) FormatPromptSection() string { /* ... */ }
//
//	r := reactor.NewReactor(config, reactor.WithIntentRegistry(&SemanticIntentRegistry{...}))
func WithIntentRegistry(reg IntentRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.intentRegistry = reg
	}
}

// WithToolRegistry sets a custom ToolRegistry implementation.
// Use this to add dynamic tool discovery, MCP integration, semantic filtering, etc.
// If not set, DefaultToolRegistry is used automatically.
//
// Example: MCP-integrated tool registry that merges local + remote tools:
//
//	type MCPToolRegistry struct {
//	    *reactor.DefaultToolRegistry
//	    mcpClient *mcp.Client
//	}
//	func (m *MCPToolRegistry) FindAvailable(filter core.ToolFilter) []core.FuncTool { /* merge local+remote */ }
func WithToolRegistry(reg core.ToolRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.toolRegistry = reg
	}
}

// WithSkillRegistry sets a custom SkillRegistry implementation.
// Use this to provide embedding-based semantic skill matching, etc.
// If not set, DefaultSkillRegistry is used automatically.
func WithSkillRegistry(reg core.SkillRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.skillRegistry = reg
	}
}

// WithSessionStore sets a SessionStore for conversation persistence.
// The session store provides the backing store for the sliding window mechanism,
// enabling unlimited context through message persistence and token-budget-aware retrieval.
func WithSessionStore(store core.SessionStore) ReactorOption {
	return func(s *reactorSetup) {
		s.sessionStore = store
	}
}

// WithRuleRegistry sets a custom RuleRegistry for behavior rule management.
// Rules are injected into the System Prompt's <behavioral_rules> section,
// allowing runtime customization of agent behavior without code changes.
//
// If not set, a new empty DefaultRuleRegistry is created (no custom rules,
// default behavioral rules from template are used).
func WithRuleRegistry(reg core.RuleRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.ruleRegistry = reg
	}
}
