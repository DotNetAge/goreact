package agent

// TaskDecomposer 任务拆解器接口（外部接入点）
// 可以由外部集成 AgenticRAG 等系统来实现任务拆解
type TaskDecomposer interface {
	// Decompose 将复杂任务拆解为多个子任务
	Decompose(task string, context interface{}) ([]SubTask, error)
}

// SubTask 子任务
type SubTask struct {
	ID          string                 // 子任务 ID
	Description string                 // 子任务描述
	Dependencies []string              // 依赖的子任务 ID
	Metadata    map[string]interface{} // 元数据
}

// DefaultTaskDecomposer 默认任务拆解器（简单实现）
type DefaultTaskDecomposer struct{}

// NewDefaultTaskDecomposer 创建默认任务拆解器
func NewDefaultTaskDecomposer() *DefaultTaskDecomposer {
	return &DefaultTaskDecomposer{}
}

// Decompose 默认实现：不拆解，直接返回原任务
func (d *DefaultTaskDecomposer) Decompose(task string, context interface{}) ([]SubTask, error) {
	return []SubTask{
		{
			ID:          "task-1",
			Description: task,
			Dependencies: []string{},
			Metadata:    make(map[string]interface{}),
		},
	}, nil
}
