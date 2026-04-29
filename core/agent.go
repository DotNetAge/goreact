package core

type AgentConfig struct {
	Name         string `json:"name" yaml:"name"`                 // Name is the agent name
	Role         string `json:"role" yaml:"role"`                 // Agent act as a role
	Description  string `json:"description" yaml:"description"`   // Description is the agent description
	Introduction string `json:"introduction" yaml:"introduction"` // Introduction is the agent introduction
	Model        string `json:"model" yaml:"model"`               // Model is the model name
}
