package orchestration

import (
	"context"

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
