package observer

import (
	"github.com/DotNetAge/goreact/pkg/core"
)

// Observer represents the "Senses" of the ReAct engine.
// It acts as the critical bridge transforming raw, noisy physical/digital
// execution outputs from the Actor back into structured cognitive context for the LLM.
type Observer interface {
	// Observe receives the raw output/error emitted from the most recent Actor's action.
	// Its responsibility is to:
	// 1. Parse and extract the key information.
	// 2. Denoisy, summarize, or truncate large data to conserve LLM token windows.
	// 3. Translate raw system errors (like network timeouts, auth failures)
	//    into semantically meaningful natural language for better Thinker reflection.
	//
	// Once the processing is complete, it attaches a fully constructed
	// *core.Observation back to the current Trace inside ctx.LastTrace().
	Observe(ctx *core.PipelineContext) error
}
