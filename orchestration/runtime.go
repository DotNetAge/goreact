// Package orchestration re-exports runtime types from core for convenience.
// The actual implementations live in core/runtime.go to avoid import cycles:
//   goreact → orchestration → goreact (cycle if types defined here)
//
// Consumers should use core.AgentState, core.AgentRuntimeMeta, core.RuntimeDirectory directly.
// This file exists to document the relationship and provide orchestration-specific helpers.
package orchestration

import (
	"github.com/DotNetAge/goreact/core"
)

// State constants shorthand — delegates to core package.
type AgentState = core.AgentState

const (
	AgentStateIdle         = core.AgentStateIdle
	AgentStateBusy         = core.AgentStateBusy
	AgentStateCoordinating = core.AgentStateCoordinating
	AgentStateDormant      = core.AgentStateDormant
	AgentStateError        = core.AgentStateError
)

// NewAgentRuntimeMeta creates a runtime metadata entry from an AgentConfig.
// Delegates to core.NewAgentRuntimeMeta.
func NewAgentRuntimeMeta(config *core.AgentConfig) *core.AgentRuntimeMeta {
	return core.NewAgentRuntimeMeta(config)
}

// RuntimeDirectory shorthand — delegates to core package.
type RuntimeDirectory = core.RuntimeDirectory

// NewRuntimeDirectory shorthand — delegates to core package.
func NewRuntimeDirectory(maxSize int) *core.RuntimeDirectory {
	return core.NewRuntimeDirectory(maxSize)
}

// Error shorthand — delegates to core package.
var (
	ErrRuntimeDirDuplicate = core.ErrRuntimeDirDuplicate
	ErrRuntimeDirFull     = core.ErrRuntimeDirFull
	ErrRuntimeDirNotFound = core.ErrRuntimeDirNotFound
)
