package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/terminator"
	"github.com/ray/goreact/pkg/types"
)

// LoopControlStep 循环控制步骤
type LoopControlStep struct {
	controller terminator.Terminator
	maxIter    int
	logger     log.Logger
}

// NewLoopControlStep 创建循环控制步骤
func NewLoopControlStep(controller terminator.Terminator, maxIter int, logger log.Logger) *LoopControlStep {
	return &LoopControlStep{
		controller: controller,
		maxIter:    maxIter,
		logger:     logger,
	}
}

// Name 返回步骤名称
func (s *LoopControlStep) Name() string { return "LoopControl" }

// Execute 执行循环控制
func (s *LoopControlStep) Execute(ctx context.Context, state *ReActState) error {
	state.Iteration++

	loopState := &types.LoopState{
		Iteration:    state.Iteration,
		Task:         state.Task,
		LastThought:  state.LastThought,
		LastResult:   state.LastResult,
		LastFeedback: state.LastFeedback,
	}

	action := s.controller.Control(loopState)

	if !action.ShouldContinue {
		state.ShouldStop = true

		if state.Iteration >= s.maxIter {
			state.Result.Success = false
			state.Result.Error = fmt.Errorf("max iterations reached without completion")
		} else {
			state.Result.Success = true
		}

		state.Result.Output = action.Reason
		state.Result.EndTime = time.Now()

		s.logger.Debug("Loop control decided to stop",
			log.Int("iteration", state.Iteration),
			log.String("reason", action.Reason),
		)
	}

	return nil
}
