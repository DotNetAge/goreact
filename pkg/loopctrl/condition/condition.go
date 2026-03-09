package condition

import (
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// StopCondition 停止条件接口
type StopCondition interface {
	ShouldStop(state *types.LoopState, ctx *core.Context) (bool, string)
}

// CompositeCondition 组合停止条件
type CompositeCondition struct {
	conditions []StopCondition
}

// NewComposite 创建组合停止条件
func NewComposite(conditions ...StopCondition) *CompositeCondition {
	return &CompositeCondition{
		conditions: conditions,
	}
}

// ShouldStop 检查是否应该停止（任一条件满足即停止）
func (c *CompositeCondition) ShouldStop(state *types.LoopState, ctx *core.Context) (bool, string) {
	for _, cond := range c.conditions {
		if stop, reason := cond.ShouldStop(state, ctx); stop {
			return true, reason
		}
	}
	return false, ""
}

// maxIterationCondition 最大迭代次数条件
type maxIterationCondition struct {
	max int
}

// MaxIteration 创建最大迭代次数条件
func MaxIteration(max int) StopCondition {
	return &maxIterationCondition{max: max}
}

func (c *maxIterationCondition) ShouldStop(state *types.LoopState, ctx *core.Context) (bool, string) {
	if state.Iteration >= c.max {
		return true, "reached maximum iterations"
	}
	return false, ""
}

// timeoutCondition 超时条件
type timeoutCondition struct {
	timeout   time.Duration
	startTime time.Time
}

// Timeout 创建超时条件
func Timeout(timeout time.Duration) StopCondition {
	return &timeoutCondition{
		timeout:   timeout,
		startTime: time.Now(),
	}
}

func (c *timeoutCondition) ShouldStop(state *types.LoopState, ctx *core.Context) (bool, string) {
	if time.Since(c.startTime) >= c.timeout {
		return true, "timeout exceeded"
	}
	return false, ""
}

// taskCompleteCondition 任务完成条件
type taskCompleteCondition struct{}

// TaskComplete 创建任务完成条件
func TaskComplete() StopCondition {
	return &taskCompleteCondition{}
}

func (c *taskCompleteCondition) ShouldStop(state *types.LoopState, ctx *core.Context) (bool, string) {
	if state.LastThought != nil && state.LastThought.ShouldFinish {
		return true, "task completed"
	}
	return false, ""
}
