package thinker

import (
	"github.com/ray/goreact/pkg/core"
)

// Thinker represents the "Brain" of the ReAct engine.
// It is responsible for Intent Recognition, Tool/Skill orchestration, RAG retrieval,
// and making decisions on the next action based on the execution history.
type Thinker interface {
	// Think evaluates the current state of the ReAct loop and reasoning context.
	// It must do one of two things:
	// 1. Decide on a new Action, attaching it (along with its reasoning 'Thought')
	//    as a new Trace onto the PipelineContext.
	// 2. Decide the task is completed (or failed), setting ctx.IsFinished = true
	//    and populating ctx.FinalResult.
	//
	// It returns an error only if the Thinking process itself fatally fails
	// (e.g., LLM network crash, prompt generation failure).
	Think(ctx *core.PipelineContext) error
}
