package agent

// Agent 智能体配置（纯配置，不持有运行时对象）
// Agent = System Prompt + Model Name
// 通过 System Prompt 定义角色和行为，通过 Model Name 指定使用的模型
type Agent struct {
	Name         string            // 智能体名称
	Description  string            // 智能体描述（用于选择匹配）
	SystemPrompt string            // 系统提示词（定义 Agent 的角色和行为）
	ModelName    string            // 使用的模型名称（引用 Model 配置）
	Metadata     map[string]string // 元数据（可选）
}

// NewAgent 创建新的智能体
func NewAgent(name, description, systemPrompt, modelName string) *Agent {
	return &Agent{
		Name:         name,
		Description:  description,
		SystemPrompt: systemPrompt,
		ModelName:    modelName,
		Metadata:     make(map[string]string),
	}
}
