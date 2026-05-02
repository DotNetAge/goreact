// Package orchestration implements the Orchestrator pattern for multi-agent coordination.
//
// Architecture: Hub-and-Spoke topology via Go channels.
//   - Orchestrator = central hub (编排层 + Agent工厂 + 事件聚合器 + Model分配器)
//   - Agents = spokes (independent T-A-O reactors, communicate only via Channel)
//
// Design doc: goreact/docs/SubAgent机制.md
package orchestration

import "github.com/DotNetAge/goreact/core"

// MessageType identifies the type of message sent to the Orchestrator inbox.
type MessageType string

const (
	// MsgDelegate requests a new sub-agent task.
	MsgDelegate MessageType = "delegate"
	// MsgQuery queries task status or list tasks.
	MsgQuery MessageType = "query"
	// MsgCancel cancels a running task.
	MsgCancel MessageType = "cancel"
	// MsgResult reports a completed (or failed) sub-agent result back to the Orchestrator.
	MsgResult MessageType = "result"
	// MsgBroadcast is reserved for future agent-to-agent messaging via Orchestrator routing.
	MsgBroadcast MessageType = "broadcast"
)

// Message is the envelope for all Orchestrator inbox communication.
// All Agent → Orchestrator communication uses this type.
type Message struct {
	Type      MessageType     // Message kind
	TaskID    string          // Associated task ID
	From      string          // Sender identifier (agent name or "system")
	Payload   any             // Type-specific payload
	ReplyCh   chan<- Response // Reply channel (must be buffered!)
	Timestamp int64
}

// Response is the reply sent back through Message.ReplyCh.
type Response struct {
	Data  any
	Error error
}

// DelegateRequest is the Payload for MsgDelegate messages.
// When AgentName is non-empty, this is an explicit routing request (legacy mode).
// When AgentName is empty, the Orchestrator should use LLM Router for smart routing (Design §6.3).
type DelegateRequest struct {
	AgentName string            // Target agent name (from AgentRegistry). Empty = smart routing
	TaskPrompt string           // Task instruction for the sub-agent
	ParentID   string           // Parent task ID
	Metadata  map[string]any   // Optional metadata (e.g., priority, tags)
}

// RouteTaskRequest is the Payload for intelligent routing via LLM Router.
// This is used internally when AgentName is empty in DelegateRequest.
type RouteTaskRequest struct {
	TaskDescription   string         // Full task description for semantic matching
	DesiredCapability string         // Optional: expected capability hint for router
	ParentID          string         // Parent task ID
	Priority          int            // 1=high, 5=low (default 3)
	Metadata          map[string]any // Optional context metadata
}

// DelegateResult is returned immediately when a delegate is accepted.
// The actual result arrives asynchronously via MsgResult → handleResult.
type DelegateResult = core.DelegateResult
