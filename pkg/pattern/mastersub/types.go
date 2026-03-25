package mastersub

import (
	"context"
	"time"

	"github.com/ray/goreact/pkg/core"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskSuccess   TaskStatus = "success"
	TaskFailed    TaskStatus = "failed"
	TaskSkipped   TaskStatus = "skipped"
)

// Task 表示主从模式下的一个原子或复合任务单元
type Task struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`       // 简短标题
	Description string                 `json:"description"` // 详细任务描述
	Dependencies []string               `json:"dependencies"` // 依赖的任务 ID
	Status      TaskStatus             `json:"status"`
	Input       map[string]interface{} `json:"input"`       // 任务输入参数
	Output      string                 `json:"output"`      // 任务执行结果
	Error       error                  `json:"-"`           // 执行中的错误信息
	
	// 进阶属性
	IsComposite bool                   `json:"is_composite"` // 是否是复合任务（需要开启子 ReAct 循环）
	SkillName   string                 `json:"skill_name"`   // 如果是复合任务，对应的 Skill 名称
}

// TaskResult 封装任务执行的最终产出与全量 Trace
type TaskResult struct {
	TaskID    string
	Success   bool
	Answer    string
	Traces    []core.Trace // 极其重要：这是结晶器 (Crystallizer) 的原材料
	Duration  time.Duration
}

// SubReactor 接口定义了子代理的执行契约
type SubReactor interface {
	Execute(ctx context.Context, task Task) (TaskResult, error)
}
