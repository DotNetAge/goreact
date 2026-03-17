package steps

import (
	"context"
	"time"

	"github.com/ray/goreact/pkg/log"
)

// CheckFinishStep 检查是否应该结束
type CheckFinishStep struct {
	logger log.Logger
}

// NewCheckFinishStep 创建结束检查步骤
func NewCheckFinishStep(logger log.Logger) *CheckFinishStep {
	return &CheckFinishStep{logger: logger}
}

// Name 返回步骤名称
func (s *CheckFinishStep) Name() string { return "CheckFinish" }

// Execute 检查是否应该结束
func (s *CheckFinishStep) Execute(ctx context.Context, state *ReActState) error {
	if state.LastThought == nil {
		return nil
	}

	if state.LastThought.ShouldFinish {
		state.ShouldStop = true
		state.Result.Success = true
		state.Result.Output = state.LastThought.FinalAnswer
		state.Result.EndTime = time.Now()

		s.logger.Debug("Task completed",
			log.String("output", state.LastThought.FinalAnswer),
		)
	}

	return nil
}
