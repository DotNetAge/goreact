package mastersub

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/engine"
)

// DefaultSubReactor 基于 engine.Reactor 实现子代理
type DefaultSubReactor struct {
	reactor *engine.Reactor
}

func NewSubReactor(r *engine.Reactor) *DefaultSubReactor {
	return &DefaultSubReactor{reactor: r}
}

// Execute 执行一个具体的 Task
func (s *DefaultSubReactor) Execute(ctx context.Context, task Task) (TaskResult, error) {
	start := time.Now()

	fmt.Printf("[Sub] 正在执行任务: [%s] %s\n", task.ID, task.Title)

	// 调用底层 Reactor 执行 ReAct 循环
	// 我们为每个子任务生成一个独立的 SessionID
	sessionID := fmt.Sprintf("sub-%s-%d", task.ID, start.Unix())
	reactCtx, err := s.reactor.Run(ctx, sessionID, task.Description)
	if err != nil {
		return TaskResult{
			TaskID:  task.ID,
			Success: false,
			Duration: time.Since(start),
		}, err
	}

	// 将 []*core.Trace 转换为 []core.Trace (值拷贝以确保解耦)
	traces := make([]core.Trace, len(reactCtx.Traces))
	for i, t := range reactCtx.Traces {
		traces[i] = *t
	}

	return TaskResult{
		TaskID:   task.ID,
		Success:  reactCtx.IsFinished && reactCtx.Error == nil,
		Answer:   reactCtx.FinalResult,
		Duration: time.Since(start),
		Traces:   traces,
	}, nil
}
