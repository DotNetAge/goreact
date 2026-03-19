package observer

import (
	"encoding/json"
	"fmt"

	"github.com/ray/goreact/pkg/core"
)

// defaultObserver represents the senses of the system.
// It translates raw binary/structural execution data from the Actor into plain text
// that the LLM (Thinker) can comprehend.
type defaultObserver struct{}

// Option configures the DefaultObserver.
type Option func(*defaultObserver)

// Default creates a basic text-parsing Observer implementation.
func Default(opts ...Option) Observer {
	o := &defaultObserver{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *defaultObserver) Observe(ctx *core.PipelineContext) error {
	lastTrace := ctx.LastTrace()

	// Observation is only needed if there was a Tool execution attempted in this step.
	if lastTrace == nil || lastTrace.Action == nil {
		return nil
	}

	// 1. Fetch raw data deposited by the Actor in the Context Bus.
	rawErrVal, hasErr := ctx.Get("raw_error")
	rawOutVal, hasOut := ctx.Get("raw_output")

	if !hasErr && !hasOut {
		// Nothing was executed (e.g., actor skipped it for some reason)
		return nil
	}

	obs := &core.Observation{
		IsSuccess: true,
	}

	// 2. Semanticize Errors (Self-Correction/Reflexion feedback)
	if hasErr && rawErrVal != nil {
		if err, ok := rawErrVal.(error); ok {
			obs.IsSuccess = false
			obs.Error = err
			// Convert the raw Go panic or HTTP timeout into a polite text observation
			// so the Thinker knows to retry or pivot.
			obs.Data = fmt.Sprintf("Action failed to execute. Error details: %v", err)
			obs.Raw = err
		}
	} else if hasOut && rawOutVal != nil {
		// 3. Serialize and truncate Output
		obs.Raw = rawOutVal

		// Attempt to format the output into a readable string for the LLM.
		switch v := rawOutVal.(type) {
		case string:
			obs.Data = v
		case []byte:
			obs.Data = string(v)
		default:
			// For complex struct returns, marshal to JSON string.
			// Advanced Observers would summarize large JSONs using an LLM here to save context space.
			b, err := json.Marshal(v)
			if err != nil {
				obs.Data = fmt.Sprintf("Action succeeded but returned unparseable binary data: %v", v)
			} else {
				obs.Data = string(b)
			}
		}
	} else {
		obs.Data = "Action executed successfully, but returned no output."
	}

	// 4. Clean up the Context Bus & Attach to Trace
	ctx.Set("raw_output", nil)
	ctx.Set("raw_error", nil)

	lastTrace.Observation = obs
	ctx.Logger.Debug("Observer processed actor results", "status_success", obs.IsSuccess)

	return nil
}
