package orchestration

import (
	"context"
	"time"

	"github.com/DotNetAge/goreact"
	"github.com/DotNetAge/goreact/core"
)

// OrchestratorOption configures a new Orchestrator instance.
type OrchestratorOption func(*orchestratorSetup)

type orchestratorSetup struct {
	modelRegistry  core.ModelRegistry
	defaultModel  *core.ModelConfig
	agentsDir      string
	maxConcurrent  int
	defaultTimeout durationWrapper // time.Duration wrapper to avoid cyclic import
	inboxSize      int
	spawnFunc      SpawnFunction
	registry       *goreact.AgentRegistry

	// --- 智能路由组件 (Design §6 / §8 / §12) ---
	llmRouter    *LLMRouter    // 智能路由引擎 (可选)
	agentFactory *AgentFactory // 动态 Agent 创建工厂 (可选)
	scoreTracker *ScoreTracker // 绩效追踪器 (可选)
}

// OrchestratorConfig is the centralized configuration structure for the Orchestrator (Design §6.2 / P2-1).
// All configuration parameters are grouped here for clarity and external exposure.
type OrchestratorConfig struct {
	// Core settings
	ModelRegistry core.ModelRegistry
	DefaultModel  *core.ModelConfig
	AgentsDir     string
	MaxConcurrent int
	DefaultTimeout time.Duration
	InboxSize      int

	// Intelligent routing components
	LLMRouter    *LLMRouter
	AgentFactory *AgentFactory
	ScoreTracker *ScoreTracker

	// Lifecycle management
	IdleAgentTimeout  time.Duration // P1-4: timeout before marking idle agents as dormant (0 = disabled)
	MinRetainedAgents int           // P1-4: minimum number of agents to retain during cleanup (0 = no cleanup)

	// State machine (P2-4)
	EnableGracefulDrain bool // If true, Stop() enters Draining state before Stopped
}

type durationWrapper struct {
	d interface{} // time.Duration
}

// WithModelRegistry injects a complete ModelRegistry supporting multiple LLM backends.
// This is the recommended option for multi-model deployments where
// different agents can use different models (e.g., researcher→DeepSeek, writer→Qwen-Flash).
func WithModelRegistry(reg core.ModelRegistry) OrchestratorOption {
	return func(s *orchestratorSetup) { s.modelRegistry = reg }
}

// WithDefaultModel registers a single model as "default" for simple single-model setups.
// All agents that don't specify a model field will use this configuration.
// Equivalent to: reg := NewInMemoryModelRegistry(); reg.Register("default", cfg); WithModelRegistry(reg)
func WithDefaultModel(cfg *core.ModelConfig) OrchestratorOption {
	return func(s *orchestratorSetup) { s.defaultModel = cfg }
}

// WithAgentsDir sets the directory to scan for .md agent definition files.
// Files are loaded into the internal AgentRegistry at Start() time.
func WithAgentsDir(dir string) OrchestratorOption {
	return func(s *orchestratorSetup) { s.agentsDir = dir }
}

// WithMaxConcurrent sets the maximum number of parallel sub-agent tasks.
// 0 means unlimited (default). When reached, new delegates block until a slot frees up.
func WithMaxConcurrent(n int) OrchestratorOption {
	return func(s *orchestratorSetup) { s.maxConcurrent = n }
}

// WithDefaultTimeout sets the default timeout for each delegated task.
// Individual DelegateTo calls can override this per-task.
func WithDefaultTimeout(d interface{ // time.Duration to avoid import cycle
	}) OrchestratorOption {
	return func(s *orchestratorSetup) { s.defaultTimeout = durationWrapper{d} }
}

// WithInboxSize sets the buffer size for the internal inbox channel.
// Default is 256. Increase if you expect bursty message traffic.
func WithInboxSize(n int) OrchestratorOption {
	return func(s *orchestratorSetup) { s.inboxSize = n }
}

// WithSpawnFunction sets a custom factory function for creating sub-agent Reactor instances.
// If not set, the default spawn function uses goreact.NewAgent internally.
// The SpawnFunction allows replacing the entire sub-agent creation strategy (e.g., for testing
// or for containerized/distributed execution).
func WithSpawnFunction(fn SpawnFunction) OrchestratorOption {
	return func(s *orchestratorSetup) { s.spawnFunc = fn }
}

// WithAgentRegistry injects a pre-built AgentRegistry (e.g., from goreact.LoadAgentsFrom).
// If not set but WithAgentsDir is provided, one is created automatically at Start().
func WithAgentRegistry(reg *goreact.AgentRegistry) OrchestratorOption {
	return func(s *orchestratorSetup) { s.registry = reg }
}

// SpawnFunction is the factory signature for creating sub-agent instances.
// It is invoked by the Orchestrator's handleDelegate method with all the information
// needed to build and launch a sub-agent. Returning an error prevents task creation.
//
// Parameters:
//   - ctx: Canceled when the parent task or Orchestrator is shutting down
//   - agentConfig: The full AgentConfig from AgentRegistry (SystemPrompt, Role, Model field, etc.)
//   - modelConfig: The resolved ModelConfig from ModelRegistry (LLM backend details)
//   - taskPrompt: The actual task instruction/prompt for this sub-agent run
//   - taskID: The unique task ID assigned by the Orchestrator
//   - resultCh: Channel to send the result (string or error) when done
type SpawnFunction func(
	ctx context.Context,
	agentConfig *core.AgentConfig,
	modelConfig *core.ModelConfig,
	taskPrompt string,
	taskID string,
	resultCh chan<- any,
) error

// WithLLMRouter injects an LLM-powered intelligent routing engine.
// When set, the Orchestrator's RouteTask() method uses semantic matching
// via LLM to select the best agent for a given task (Design §6.3).
// Without it, RouteTask falls back to keyword-based matching.
func WithLLMRouter(router *LLMRouter) OrchestratorOption {
	return func(s *orchestratorSetup) { s.llmRouter = router }
}

// WithAgentFactory injects a dynamic Agent creation factory.
// When the LLM Router returns __CREATE_NEW__, this factory creates
// a new Agent on-the-fly and registers it (Design §12).
func WithAgentFactory(factory *AgentFactory) OrchestratorOption {
	return func(s *orchestratorSetup) { s.agentFactory = factory }
}

// WithScoreTracker injects a performance tracking instance for agent selection.
// When set, the Orchestrator records scores after each task completion
// and uses epsilon-greedy strategy during cold start (Design §8.4-§8.5).
func WithScoreTracker(tracker *ScoreTracker) OrchestratorOption {
	return func(s *orchestratorSetup) { s.scoreTracker = tracker }
}

// WithIdleCleanupConfig enables periodic idle agent scanning and cleanup (P1-4 / Design §12.4).
func WithIdleCleanupConfig(cfg IdleCleanupConfig) OrchestratorOption {
	return func(s *orchestratorSetup) {
		// Idle cleanup is applied via SetIdleCleanupConfig() on the orchestrator after creation
	}
}
