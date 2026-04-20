package tools

import (
	"context"

	"github.com/DotNetAge/goreact/core"
)

// ReactorAccessor provides reactor dependencies to tools that need them.
// This interface decouples tools from the concrete reactor package,
// allowing all tools to live in the tools/ directory while still
// accessing reactor-managed resources (task manager, message bus, event bus, etc.).
type ReactorAccessor interface {
	// TaskManager returns the reactor's task manager for task lifecycle tracking.
	TaskManager() core.TaskManager

	// MessageBus returns the reactor's agent message bus for team communication.
	// Returns nil if team communication is not configured.
	MessageBus() *core.AgentMessageBus

	// EventEmitter returns a function to emit ReactEvents via the event bus.
	// The returned function may be nil if no event bus is configured.
	EventEmitter() func(core.ReactEvent)

	// RegisterPendingTask adds a pending subagent task for async result retrieval.
	RegisterPendingTask(taskID string, resultCh chan any)

	// GetPendingTask retrieves the channel for a pending subagent task.
	// Returns nil if the task is not found.
	GetPendingTask(taskID string) (<-chan any, bool)

	// RemovePendingTask removes a completed pending task.
	RemovePendingTask(taskID string)

	// RunInline executes a synchronous inline task using the same reactor context.
	// Used by TaskCreateTool for plan→execute sequential workflow.
	// Returns the answer string from the completed run.
	RunInline(ctx context.Context, prompt string) (answer string, err error)

	// RunSubAgent spawns an independent agent asynchronously in a goroutine.
	// The result is sent to the provided resultCh when execution completes.
	// This enables true async SubAgent execution.
	RunSubAgent(ctx context.Context, taskID string, systemPrompt, prompt string, model string, resultCh chan<- any)

	// Scheduler returns the reactor's CronScheduler for scheduled task management.
	// Returns nil if scheduling is not configured.
	Scheduler() *core.CronScheduler

	// Config returns the reactor's configuration (model, API key, etc.).
	Config() ReactorConfig
}

// ReactorConfig exposes reactor configuration needed by SubAgentTool.
type ReactorConfig struct {
	// APIKey for LLM access.
	APIKey string
	// BaseURL of the LLM API endpoint.
	BaseURL string
	// Model name (e.g., "gpt-4o", "claude-3-opus").
	Model string
	// ClientType for the gochat client builder.
	ClientType any // gochat.ClientType, using any to avoid direct import
	// SystemPrompt for the agent.
	SystemPrompt string
	// Temperature for LLM calls.
	Temperature float64
	// MaxTokens for LLM responses.
	MaxTokens int
	// MaxIterations limits the T-A-O loop.
	MaxIterations int
	// IsLocal indicates whether the reactor uses a local model.
	// When true, subagent spawning defaults to synchronous execution.
	IsLocal bool
}
