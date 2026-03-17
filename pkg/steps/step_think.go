package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/types"
)

// ThinkStep 思考步骤
type ThinkStep struct {
	thinker       thinker.Thinker
	logger        log.Logger
	metrics       metrics.Metrics
	maxRetries    int
	retryInterval time.Duration
}

// NewThinkStep 创建思考步骤
func NewThinkStep(thinker thinker.Thinker, logger log.Logger, m metrics.Metrics, maxRetries int, retryInterval time.Duration) *ThinkStep {
	return &ThinkStep{
		thinker:       thinker,
		logger:        logger,
		metrics:       m,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

// Name 返回步骤名称
func (s *ThinkStep) Name() string { return "Think" }

// Execute 执行思考
func (s *ThinkStep) Execute(ctx context.Context, state *ReActState) error {
	task := state.Task

	var thought *types.Thought
	var err error

	for retry := 0; retry <= s.maxRetries; retry++ {
		thought, err = s.thinker.Think(task, state.ExecCtx)
		if err == nil {
			break
		}

		s.logger.Warn("Thinking failed",
			log.Int("iteration", state.Iteration),
			log.Int("retry", retry+1),
			log.Err(err),
		)

		if retry >= s.maxRetries {
			return fmt.Errorf("thinking failed after %d retries: %w", s.maxRetries, err)
		}

		time.Sleep(s.retryInterval)
	}

	state.LastThought = thought
	state.Data.Thought = thought

	s.metrics.RecordSuccess("engine.think")
	return nil
}
