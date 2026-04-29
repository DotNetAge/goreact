package orchestration

import (
	"context"

	"github.com/DotNetAge/goreact"
	"github.com/DotNetAge/goreact/core"
)

// Orchestrator is the central hub for multi-agent coordination.
// It serves four roles simultaneously:
//   1. **编排引擎**: Task delegation, status tracking, result collection via Channel Actor loop
//   2. **Agent 工厂**: GetAgent(name) creates/retrieves fully-configured Agent instances
//   3. **事件聚合器**: Events()/EventsFiltered() provides unified global event stream
//   4. **Model 分配器**: Each Agent gets its Model from the registry based on AgentConfig.Model
//
// All agents (Master and SubAgents) communicate with the Orchestrator exclusively
// through Go channels. Agents never hold direct references to each other.
type Orchestrator interface {
	// === Lifecycle ===
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// === Agent Factory ===
	// GetAgent retrieves or creates an Agent by name from the AgentRegistry.
	// It automatically resolves the Model from ModelRegistry using AgentConfig.Model as key.
	// Results are cached: subsequent calls return the same instance (same Session).
	GetAgent(name string) (*goreact.Agent, error)

	// ReleaseAgent discards a cached Agent instance (drops Session context).
	// Next GetAgent creates a fresh instance.
	ReleaseAgent(name string)

	// === Orchestration Operations ===
	// DelegateTo sends a task to a named agent asynchronously.
	// Returns immediately with a DelegateResult containing a TaskID and result channel.
	// The actual result arrives asynchronously when the sub-agent completes.
	DelegateTo(
		ctx context.Context,
		agentName string,
		taskPrompt string,
		parentID string,
		metadata map[string]any,
	) (*DelegateResult, error)

	// WaitForResult blocks until the specified task completes or times out.
	WaitForResult(ctx context.Context, taskID string) (*core.Task, error)

	// CancelTask cancels a running or pending task.
	CancelTask(taskID string) error

	// === Event Aggregation ===
	// Events subscribes to the global event stream (all agents + orchestrator events).
	// Each event is tagged with AgentID/AgentName for source identification.
	Events() (<-chan core.ReactEvent, func())

	// EventsFiltered subscribes with a filter function.
	EventsFiltered(filter func(core.ReactEvent) bool) (<-chan core.ReactEvent, func())

	// === Low-Level Access ===
	// Send delivers a raw message to the Orchestrator inbox channel.
	// Returns a response channel for request-response patterns.
	Send(msg Message) <-chan Response

	// TaskStore returns the internal task store (read-only view for monitoring/debugging).
	TaskStore() TaskStore

	// ModelRegistry returns the internal model registry (read-only view).
	ModelRegistry() core.ModelRegistry
}
