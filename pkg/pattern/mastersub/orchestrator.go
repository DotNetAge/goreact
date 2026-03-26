package mastersub

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/pkg/core"
)

// Orchestrator 负责驱动 Master 与 Sub 的协作循环
type Orchestrator struct {
	master *Master
	sub    SubReactor
	logger core.Logger
}

func NewOrchestrator(m *Master, s SubReactor, l core.Logger) *Orchestrator {
	return &Orchestrator{
		master: m,
		sub:    s,
		logger: l,
	}
}

// Run 开启编排循环
func (o *Orchestrator) Run(ctx context.Context, goal string) ([]TaskResult, error) {
	// 1. 拆解任务
	tasks, err := o.master.Decompose(ctx, goal, "") // 目前没有传递 Skill 上下文
	if err != nil {
		return nil, fmt.Errorf("initial planning failed: %w", err)
	}

	results := make([]TaskResult, 0)
	completedTasks := make(map[string]string) // ID -> Output

	// 2. 简易拓扑执行
	// 为了跑通最小闭环，这里采用串行执行，但会检查依赖
	for i := 0; i < len(tasks); i++ {
		task := &tasks[i]
		
		// 检查依赖 (跳过已执行或无依赖的)
		for _, depID := range task.Dependencies {
			if _, ok := completedTasks[depID]; !ok {
				return nil, fmt.Errorf("task %s dependency %s not met", task.ID, depID)
			}
		}

		o.logger.Info("Starting Task", "id", task.ID, "title", task.Title)
		
		// 3. 执行子任务
		res, err := o.sub.Execute(ctx, *task)
		if err != nil {
			o.logger.Error(err, "Task Execution Failed", "id", task.ID)
			return results, err
		}

		results = append(results, res)
		completedTasks[task.ID] = res.Answer
		task.Status = TaskSuccess
		task.Output = res.Answer
	}

	o.logger.Info("All Tasks Completed Successfully")
	return results, nil
}
