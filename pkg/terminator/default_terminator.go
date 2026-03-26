package terminator

import (
	"github.com/DotNetAge/goreact/pkg/core"
)

// defaultTerminator ensures the ReAct loop does not run forever.
// It checks hard boundaries like MaxSteps.
type defaultTerminator struct{}

// Option configures the DefaultTerminator.
type Option func(*defaultTerminator)

// Default creates a standard safeguard termination rule engine.
func Default(opts ...Option) Terminator {
	t := &defaultTerminator{}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *defaultTerminator) CheckTermination(ctx *core.PipelineContext) (bool, error) {
	// Rule 1: Goal is achieved by the LLM
	if ctx.IsFinished {
		return true, nil
	}

	// Rule 2: Max Steps boundary
	if ctx.CurrentStep >= ctx.MaxSteps {
		ctx.IsFinished = true
		ctx.FinishReason = "MaxStepsExceeded"
		ctx.Logger.Warn("Engine halted due to max iterations limit", "max_steps", ctx.MaxSteps)
		return true, nil
	}

	// Rule 3: Stagnation & Infinite loops (Simple heuristic)
	// If the last 3 traces have the exact same Action name and input, the LLM is looping blindly.
	if len(ctx.Traces) >= 3 {
		t1 := ctx.Traces[len(ctx.Traces)-1]
		t2 := ctx.Traces[len(ctx.Traces)-2]
		t3 := ctx.Traces[len(ctx.Traces)-3]

		if t1.Action != nil && t2.Action != nil && t3.Action != nil {
			if t1.Action.Name == t2.Action.Name && t2.Action.Name == t3.Action.Name {
				// We'd ideally hash or strictly compare the Input Maps here.
				// This is a naive detection to demonstrate the concept.
				// For now, let it be just a warning unless they are absolutely identical.
				ctx.Logger.Warn("Stagnation detected: Agent keeps calling the same tool", "tool", t1.Action.Name)
			}
		}
	}

	return false, nil
}
