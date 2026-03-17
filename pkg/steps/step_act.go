package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/types"
)

// ActStep 行动步骤
type ActStep struct {
	actor         actor.Actor
	logger        log.Logger
	metrics       metrics.Metrics
	maxRetries    int
	retryInterval time.Duration
}

// NewActStep 创建行动步骤
func NewActStep(actor actor.Actor, logger log.Logger, m metrics.Metrics, maxRetries int, retryInterval time.Duration) *ActStep {
	return &ActStep{
		actor:         actor,
		logger:        logger,
		metrics:       m,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

// Name 返回步骤名称
func (s *ActStep) Name() string { return "Act" }

// Execute 执行行动
func (s *ActStep) Execute(ctx context.Context, state *ReActState) error {
	if state.LastThought == nil || state.LastThought.Action == nil {
		return nil
	}

	action := state.LastThought.Action
	var execResult *types.ExecutionResult
	var err error

	for retry := 0; retry <= s.maxRetries; retry++ {
		execResult, err = s.actor.Act(action, state.ExecCtx)
		if err == nil {
			break
		}

		s.logger.Warn("Action failed",
			log.Int("iteration", state.Iteration),
			log.String("tool", action.ToolName),
			log.Int("retry", retry+1),
			log.Err(err),
		)

		if retry >= s.maxRetries {
			return fmt.Errorf("action failed: %w", err)
		}

		time.Sleep(s.retryInterval)
	}

	state.LastResult = execResult
	state.Data.ExecResult = execResult
	state.ExecCtx.Set("last_action", action)
	state.ExecCtx.Set("last_result", execResult)

	s.metrics.RecordSuccess("engine.act")
	return nil
}
