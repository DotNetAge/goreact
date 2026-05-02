package tools

import (
	"context"

	"github.com/DotNetAge/goreact/core"
)

// OrchestrationAccessor provides orchestration dependencies to tools.
// This interface decouples tools from the concrete reactor/orchestrator
// while providing access to orchestration resources.
//
// Design principles (v2 — aligned with SubAgent architecture):
//   - Orchestrator() is the SINGLE entry point for all orchestration operations
//   - Config() provides reactor configuration access
//   - Deprecated: EventEmitter()/RunInline()/TaskStore() kept for migration compat only
type OrchestrationAccessor interface {
	// Orchestrator returns the orchestrator for task delegation, agent queries, etc.
	// Returns nil if no orchestrator is configured (single-agent mode).
	Orchestrator() core.AgentOrchestrator

	// Config returns the reactor configuration (model, API key, system prompt, etc.).
	Config() ReactorConfig

	// --- Legacy methods (deprecated, will be removed after full migration) ---

	// EventEmitter returns a function to emit ReactEvents via the event bus.
	// Deprecated: Events should flow through Orchestrator's event aggregator.
	EventEmitter() func(core.ReactEvent)

	// RunInline executes a synchronous inline task using the same reactor context.
	// Deprecated: Use Orchestrator().DelegateTo() with appropriate parameters instead.
	RunInline(ctx context.Context, prompt string) (answer string, err error)
}

// ReactorConfig exposes configuration needed by orchestration tools.
type ReactorConfig struct {
	APIKey         string
	BaseURL        string
	Model          string
	ClientType     any // gochat.ClientType; using any to avoid direct import
	SystemPrompt   string
	Temperature    float64
	MaxTokens      int
	MaxIterations  int
	IsLocal        bool
}

// Deprecated: ReactorAccessor is the legacy interface name.
// Use OrchestrationAccessor instead. The two interfaces have identical shape;
// this alias exists purely for source-level backward compatibility during migration.
//
// Tools currently being migrated: task_tools.go
type ReactorAccessor = OrchestrationAccessor
