package engine

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/pipeline"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/steps"
)

// MetricsHook 记录指标的 Hook
type MetricsHook struct {
	metrics metrics.Metrics
}

// NewMetricsHook 创建 Metrics Hook
func NewMetricsHook(m metrics.Metrics) *MetricsHook {
	return &MetricsHook{metrics: m}
}

// OnStepStart 步骤开始时调用
func (h *MetricsHook) OnStepStart(ctx context.Context, step pipeline.Step[*steps.ReActState], state *steps.ReActState) {
}

// OnStepComplete 步骤完成时调用
func (h *MetricsHook) OnStepComplete(ctx context.Context, step pipeline.Step[*steps.ReActState], state *steps.ReActState) {
	h.metrics.RecordSuccess(fmt.Sprintf("engine.%s", step.Name()))
}

// OnStepError 步骤出错时调用
func (h *MetricsHook) OnStepError(ctx context.Context, step pipeline.Step[*steps.ReActState], state *steps.ReActState, err error) {
	h.metrics.RecordError(fmt.Sprintf("engine.%s", step.Name()), err)
}
