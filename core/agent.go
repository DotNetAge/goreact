package core

type AgentConfig struct {
	Name         string   `json:"name" yaml:"name"`                   // Name is the agent name
	Domain       string   `json:"domain" yaml:"domain"`               // Agent 所擅长的领域
	Description  string   `json:"description" yaml:"description"`     // Description is the agent description
	SystemPrompt string   `json:"system_prompt" yaml:"system_prompt"` // SystemPrompt is the system prompt
	Model        string   `json:"model" yaml:"model"`                 // Model is the model name
	Tools        []string `json:"tools" yaml:"tools"`                 // Tools is the list of available tools
}
