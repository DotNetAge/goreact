package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/observer"
	"github.com/ray/goreact/pkg/types"
)

// ObserveStep 观察步骤
type ObserveStep struct {
	observer      observer.Observer
	logger        log.Logger
	metrics       metrics.Metrics
	maxRetries    int
	retryInterval time.Duration
}

// NewObserveStep 创建观察步骤
func NewObserveStep(observer observer.Observer, logger log.Logger, m metrics.Metrics, maxRetries int, retryInterval time.Duration) *ObserveStep {
	return &ObserveStep{
		observer:      observer,
		logger:        logger,
		metrics:       m,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

// Name 返回步骤名称
func (s *ObserveStep) Name() string { return "Observe" }

// Execute 执行观察
func (s *ObserveStep) Execute(ctx context.Context, state *ReActState) error {
	if state.LastResult == nil {
		return nil
	}

	var feedback *types.Feedback
	var err error

	for retry := 0; retry <= s.maxRetries; retry++ {
		feedback, err = s.observer.Observe(state.LastResult, state.ExecCtx)
		if err == nil {
			break
		}

		s.logger.Warn("Observation failed",
			log.Int("iteration", state.Iteration),
			log.Int("retry", retry+1),
			log.Err(err),
		)

		if retry >= s.maxRetries {
			return fmt.Errorf("observation failed: %w", err)
		}

		time.Sleep(s.retryInterval)
	}

	state.LastFeedback = feedback
	state.Data.Feedback = feedback

	s.appendHistoryStep(state)
	return nil
}

// appendHistoryStep 追加历史记录到上下文
func (s *ObserveStep) appendHistoryStep(state *ReActState) {
	step := HistoryStep{
		Action:   fmt.Sprintf("%s with params %v", state.LastThought.Action.ToolName, state.LastThought.Action.Parameters),
		Result:   fmt.Sprintf("%v (Success: %v)", state.LastResult.Output, state.LastResult.Success),
		Feedback: state.LastFeedback.Message,
	}

	state.Data.HistorySteps = append(state.Data.HistorySteps, step)
}
