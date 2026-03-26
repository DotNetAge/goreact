package terminator

import (
	"github.com/DotNetAge/goreact/pkg/core"
)

// Terminator acts as the final "Judicial Defense Line" in the continuous ReAct loop.
// Its sole objective is safely interrupting potentially infinite or stagnant iterations,
// gracefully ending loops upon success, error thresholds, max steps, or resource exhaustion.
type Terminator interface {
	// CheckTermination takes the current PipelineContext and evaluates it against
	// pre-configured safety, bounds, or intent-success conditions.
	//
	// It returns true (with a valid state mutation inside Context, such as FinalResult
	// or FinishReason) if the Agent must halt the current ReAct workflow loop.
	//
	// The return err signifies a catastrophic failure in the evaluation rule engine itself
	// (not Agent task failure), and usually forces an immediate abort of the application.
	CheckTermination(ctx *core.PipelineContext) (bool, error)
}
