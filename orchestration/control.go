// Package orchestration re-exports control types from core to avoid import cycles.
package orchestration

import "github.com/DotNetAge/goreact/core"

// ControlCommand is a lifecycle control instruction for Coordinator mode.
type ControlCommand = core.ControlCommand

// Control command constants.
const (
	CmdInterrupt = core.CmdInterrupt
	CmdResume    = core.CmdResume
	CmdCancel    = core.CmdCancel
)
