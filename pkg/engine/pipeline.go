package engine

import (
	"time"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/observer"
	"github.com/ray/goreact/pkg/steps"
	"github.com/ray/goreact/pkg/terminator"
)

// BuildReActPipeline 构建 ReAct Pipeline
func BuildReActPipeline(
	thinker thinker.Thinker,
	actor actor.Actor,
	observer observer.Observer,
	controller terminator.Terminator,
	logger log.Logger,
	m metrics.Metrics,
	maxRetries int,
	retryInterval time.Duration,
	maxIter int,
) *pipeline.Pipeline[*steps.ReActState] {
	p := pipeline.New[*steps.ReActState]()

	p.AddSteps(
		steps.NewThinkStep(thinker, logger, m, maxRetries, retryInterval),
		steps.NewCheckFinishStep(logger),
		steps.NewActStep(actor, logger, m, maxRetries, retryInterval),
		steps.NewObserveStep(observer, logger, m, maxRetries, retryInterval),
		steps.NewLoopControlStep(controller, maxIter, logger),
	)

	p.AddHook(NewTraceHook())
	p.AddHook(NewMetricsHook(m))

	return p
}
