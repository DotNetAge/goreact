package agent

// Agent 智能体（极简设计）
// Agent = Name + System Prompt
type Agent struct {
	Name         string // 智能体名称
	SystemPrompt string // 系统提示词（定义 Agent 的角色和行为）
}

// NewAgent 创建新的智能体
func NewAgent(name, systemPrompt string) *Agent {
	return &Agent{
		Name:         name,
		SystemPrompt: systemPrompt,
	}
}
