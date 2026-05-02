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
//
// P2-SOLID-02: This composite interface is assembled from smaller focused interfaces
// below for better API design and dependency injection.
type Orchestrator interface {
	TaskOrchestrator
	AgentFactoryOps
	EventAggregator
	RuntimeAccess
	Lifecycle
}

// Lifecycle handles start/stop of the orchestrator.
type Lifecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// TaskOrchestrator handles task delegation and coordination.
type TaskOrchestrator interface {
	// DelegateTo sends a task to a named agent asynchronously.
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

	// Send delivers a raw message to the Orchestrator inbox channel.
	Send(msg Message) <-chan Response
}

// AgentFactoryOps handles agent creation and caching.
type AgentFactoryOps interface {
	// GetAgent retrieves or creates an Agent by name from the AgentRegistry.
	GetAgent(name string) (*goreact.Agent, error)

	// ReleaseAgent discards a cached Agent instance.
	ReleaseAgent(name string)

	// RegisterAgent registers an agent's runtime metadata.
	RegisterAgent(meta *core.AgentRuntimeMeta) error
}

// EventAggregator provides event streaming.
type EventAggregator interface {
	// Events subscribes to the global event stream.
	Events() (<-chan core.ReactEvent, func())

	// EventsFiltered subscribes with a filter function.
	EventsFiltered(filter func(core.ReactEvent) bool) (<-chan core.ReactEvent, func())
}

// RuntimeAccess provides read-only views for monitoring and debugging.
type RuntimeAccess interface {
	// TaskStore returns the internal task store.
	TaskStore() TaskStore

	// ModelRegistry returns the internal model registry.
	ModelRegistry() core.ModelRegistry

	// RuntimeDir returns the runtime state directory.
	RuntimeDir() *core.RuntimeDirectory
}
