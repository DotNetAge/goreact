package core

import "context"

// AgentOrchestrator is the minimal interface for multi-agent coordination.
// Defined in core to avoid import cycles between reactor, tools, and orchestration.
// The orchestration.Orchestrator implementation satisfies this interface.
type AgentOrchestrator interface {
	// DelegateTo sends a task to a named agent asynchronously.
	DelegateTo(ctx context.Context, agentName, taskPrompt, parentID string, metadata map[string]any) (*DelegateResult, error)

	// WaitForResult blocks until the specified task completes or times out.
	WaitForResult(ctx context.Context, taskID string) (*Task, error)

	// Agent discovery
	ListAgents() []string
	AgentInfo(name string) *AgentConfig

	// Task store access
	ListTasks(parentID string) ([]*Task, error)
	GetTask(taskID string) (*Task, error)
}

// DelegateResult holds the result of a delegation request.
// Mirrored in reactor and tools; defined in core to avoid import cycles.
type DelegateResult struct {
	TaskID   string
	ResultCh <-chan any
}
