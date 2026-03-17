package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/ray/goreact/pkg/steps"
	"github.com/ray/goreact/pkg/types"
)

// TraceHook 记录 Trace 的 Hook
type TraceHook struct{}

// NewTraceHook 创建 Trace Hook
func NewTraceHook() *TraceHook {
	return &TraceHook{}
}

// OnStepStart 步骤开始时调用
func (h *TraceHook) OnStepStart(ctx context.Context, step pipeline.Step[*steps.ReActState], state *steps.ReActState) {
	state.Result.Trace = append(state.Result.Trace, types.TraceStep{
		Step:      state.Iteration,
		Type:      fmt.Sprintf("%s_start", step.Name()),
		Content:   fmt.Sprintf("Starting step: %s", step.Name()),
		Timestamp: time.Now(),
	})
}

// OnStepComplete 步骤完成时调用
func (h *TraceHook) OnStepComplete(ctx context.Context, step pipeline.Step[*steps.ReActState], state *steps.ReActState) {
	switch step.Name() {
	case "Think":
		if thought := state.LastThought; thought != nil {
			state.Result.Trace = append(state.Result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "think",
				Content:   fmt.Sprintf("Reasoning: %s", thought.Reasoning),
				Timestamp: time.Now(),
			})
		}
	case "Act":
		if result := state.LastResult; result != nil {
			state.Result.Trace = append(state.Result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "result",
				Content:   fmt.Sprintf("Result: %v (Success: %v)", result.Output, result.Success),
				Timestamp: time.Now(),
			})
		}
	case "Observe":
		if feedback := state.LastFeedback; feedback != nil {
			state.Result.Trace = append(state.Result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "observe",
				Content:   fmt.Sprintf("Feedback: %s", feedback.Message),
				Timestamp: time.Now(),
			})
		}
	}
}

// OnStepError 步骤出错时调用
func (h *TraceHook) OnStepError(ctx context.Context, step pipeline.Step[*steps.ReActState], state *steps.ReActState, err error) {
	state.Result.Trace = append(state.Result.Trace, types.TraceStep{
		Step:      state.Iteration,
		Type:      fmt.Sprintf("%s_error", step.Name()),
		Content:   fmt.Sprintf("Step failed: %v", err),
		Timestamp: time.Now(),
	})
}
