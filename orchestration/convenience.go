package orchestration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DotNetAge/goreact"
	"github.com/DotNetAge/goreact/core"
)

// NewWithAgentsDir is a convenience constructor that combines:
//   - Loading agent definitions from a .md file directory
//   - Building an Orchestrator with those agents + the provided options
//
// Usage:
//
//	orch, err := orchestration.NewWithAgentsDir("./agents/",
//	    orchestration.WithModelRegistry(registry),  // or WithDefaultModel(cfg)
//	    orchestration.WithMaxConcurrent(10),
//	)
//	orch.Start(ctx)
//	defer orch.Stop(ctx)
//	pm, _ := orch.GetAgent("project-manager")
//	pm.Ask("...")
func NewWithAgentsDir(dir string, opts ...OrchestratorOption) (*ChannelOrchestrator, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agents dir: %w", err)
	}

	registry, err := goreact.LoadAgentsFrom(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load agents from %q: %w", absPath, err)
	}

	opts = append(opts, WithAgentRegistry(registry))
	return New(opts...)
}

// NewWithDefaultModel is a convenience constructor for single-model setups.
// It registers the given model as "default" so all agents without
// an explicit `model:` field in their .md definition will use it.
//
// Usage:
//
//	orch, err := orchestration.NewWithDefaultModel(goreact.DefaultModel(),
//	    orchestration.WithAgentsDir("./agents"),
//	)
func NewWithDefaultModel(cfg *core.ModelConfig, opts ...OrchestratorOption) (*ChannelOrchestrator, error) {
	// Ensure APIKey is set for single-model usage
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ModelConfig.APIKey must be set for WithDefaultModel")
	}
	opts = append(opts, WithDefaultModel(cfg))
	return New(opts...)
}

// ListAgents returns the names of all registered agents.
// Useful for debugging and for building progressive disclosure prompts.
func (o *ChannelOrchestrator) ListAgents() []string {
	if o.registry == nil {
		return nil
	}
	agents := o.registry.List()
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		names = append(names, a.Name)
	}
	return names
}

// AgentInfo returns the full AgentConfig for a named agent (nil if not found).
// Useful for inspecting what model/capabilities an agent has.
func (o *ChannelOrchestrator) AgentInfo(name string) *core.AgentConfig {
	if o.registry == nil {
		return nil
	}
	return o.registry.Get(name)
}

// RegisteredModels returns all model names in the ModelRegistry.
// Useful for validation at startup.
func (o *ChannelOrchestrator) RegisteredModels() []string {
	if o.modelRegistry == nil {
		return nil
	}
	return o.modelRegistry.List()
}

// ValidateStartup checks that all agents' model references can be resolved.
// Call this after New + before Start to catch configuration errors early.
// Returns warnings (non-fatal) and errors (fatal).
func (o *ChannelOrchestrator) ValidateStartup() ([]string, error) {
	var warnings []string
	var errs []string

	if o.registry == nil {
		warnings = append(warnings, "no agent registry loaded (WithAgentsDir or WithAgentRegistry required)")
	}

	if o.modelRegistry == nil {
		errs = append(errs, "no model registry configured (WithModelRegistry or WithDefaultModel required)")
	}

	// Check each agent's model field resolves correctly
	if o.registry != nil && o.modelRegistry != nil {
		for _, agent := range o.registry.List() {
			modelName := agent.Model
			if modelName == "" {
				modelName = "default" // will use default model
			}
			if _, err := o.modelRegistry.Get(modelName); err != nil {
				warnings = append(warnings,
					fmt.Sprintf("agent %q references model %q which is not registered (will fail at GetAgent time)",
						agent.Name, modelName))
			}
		}
	}

	if len(errs) > 0 {
		return warnings, fmt.Errorf("startup validation failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return warnings, nil
}

// Stats returns runtime statistics about the Orchestrator.
func (o *ChannelOrchestrator) Stats() map[string]interface{} {
	return map[string]interface{}{
		"cached_agents":    o.agentCache.Len(),
		"active_tasks":     o.store.ActiveTasks(),
		"registered_agents": len(o.ListAgents()),
		"registered_models": len(o.RegisteredModels()),
		"started":          o.started,
		"event_subscribers": func() int {
			o.eventSubsMu.RLock()
			n := len(o.eventSubs)
			o.eventSubsMu.RUnlock()
			return n
		}(),
	}
}
