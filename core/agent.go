package core

type AgentConfig struct {
	Name         string   `json:"name" yaml:"name"`                   // Name is the agent name
	Role         string   `json:"role" yaml:"role"`                   // Agent act as a role
	Description  string   `json:"description" yaml:"description"`     // Description is the agent description
	SystemPrompt string   `json:"system_prompt" yaml:"system_prompt"` // SystemPrompt is the system prompt
	Model        string   `json:"model" yaml:"model"`                 // Model is the model name
	Capabilities []string `json:"capabilities" yaml:"capabilities"`   // Capabilities is the list of available skills
}
