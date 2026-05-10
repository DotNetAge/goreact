package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// CreateAgentTool creates a new agent definition and registers it in the agent directory.
type CreateAgentTool struct {
	registry   AgentDefinitionRegistry
	runtimeDir *core.RuntimeDirectory
}

// AgentDefinitionRegistry is the interface for saving agent definitions.
// Implemented by goreact.AgentRegistry.
type AgentDefinitionRegistry interface {
	Get(name string) *core.AgentConfig
	SaveTo(agent *core.AgentConfig) error
	List() []*core.AgentConfig
}

// NewCreateAgentTool creates a CreateAgentTool.
func NewCreateAgentTool(registry AgentDefinitionRegistry, runtimeDir *core.RuntimeDirectory) *CreateAgentTool {
	return &CreateAgentTool{registry: registry, runtimeDir: runtimeDir}
}

func (t *CreateAgentTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "CreateAgent",
		Description: "Create a new agent with a specific name, role, description, and instructions. The agent becomes available for delegation via Delegate and searchable via FindAgent.",
		Prompt: `Define a new agent when a specialized task type falls outside your role and no existing agent covers it.

When to use:
- A recurring task pattern is outside your area of expertise and there is no existing agent for it.
- The user asks you to define a dedicated expert role with its own system prompt.
- You identified a gap in the agent registry that would benefit from a custom specialist.

How to create a good agent:

Before defining the fields below, call SkillList to query all available skills and ModelList to see all available models — you need this information to make informed choices.

- Name: short, descriptive, kebab-case (e.g. "code-reviewer", "data-analyst", "security-auditor")
- Role: the agent's function (e.g. "code reviewer", "data analyst", "security auditor"), write a clear job title that immediately conveys the Agent's area of expertise.
- Description: concise capability summary (max 1024 chars) — this is what FindAgent searches against
	- **Third-person perspective** — describe what this position does, not "You are..."
	- **Job posting style** — use "Responsible for..." framing
	- **Concise** — keep it brief and impactful
	- Brief position summary in third-person perspective
- Introduction: the agent's full system prompt — define its behavior, tools, rules, and output format
- Skills: array of skill names the agent should have.
	- **Prerequisite**: call SkillList to learn what skills are available.
	- **Selection rule**: choose only skills whose domain aligns with the agent's Role, Description, and Introduction. Each skill must serve the agent's stated area of expertise, its responsibilities, and the specific scope of work defined in its instructions. Do not assign skills that are irrelevant to the agent's purpose or outside the boundaries of its role.
	- Example: a "security-auditor" agent responsible for code vulnerability scanning should receive skills like ["code-review", "security"], not ["data-visualization", "marketing"].
- Model: pick the model from ModelList results — set this to the model name that best fits the agent's role

Once created, the agent appears in FindAgent results and can receive tasks via Delegate.`,
		Tags: []string{"agent", "create", "definition", "orchestration"},
		Parameters: []core.Parameter{
			{Name: "name", Type: "string", Description: "Agent name (kebab-case, e.g. 'code-reviewer')", Required: true},
			{Name: "role", Type: "string", Description: "Agent role (e.g. 'code reviewer', 'data analyst') ", Required: true},
			{Name: "description", Type: "string", Description: "Capability summary (max 1024 chars) — used by FindAgent for search matching", Required: true},
			{Name: "introduction", Type: "string", Description: "Full system prompt / instructions defining the agent's behavior, rules, and output format", Required: true},
			{Name: "skills", Type: "array", Description: "Array of skill names the agent should have. Check SkillList for available skills first.", Required: false},
			{Name: "model", Type: "string", Description: "Model name from ModelList. Call ModelList first to see available models. Defaults to parent agent's model if empty.", Required: false},
		},
	}
}

func (t *CreateAgentTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	name, _ := params["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	role, _ := params["role"].(string)
	if role == "" {
		return nil, fmt.Errorf("role is required")
	}
	description, _ := params["description"].(string)
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	introduction, _ := params["introduction"].(string)
	if introduction == "" {
		return nil, fmt.Errorf("introduction (system prompt) is required")
	}
	model, _ := params["model"].(string)
	var skills []string
	if rawSkills, ok := params["skills"].([]any); ok {
		for _, s := range rawSkills {
			if str, ok := s.(string); ok {
				skills = append(skills, str)
			}
		}
	} else if strSkills, ok := params["skills"].(string); ok && strSkills != "" {
		skills = append(skills, strSkills)
	}

	if t.registry == nil {
		return nil, fmt.Errorf("agent registry not configured")
	}

	// Check if already exists
	if existing := t.registry.Get(name); existing != nil {
		return nil, fmt.Errorf("agent %q already exists — use a different name or update the existing definition", name)
	}

	agent := &core.AgentConfig{
		Name:         name,
		Role:         role,
		Description:  description,
		Introduction: introduction,
		Model:        model,
		Skills:       skills,
	}

	if err := t.registry.SaveTo(agent); err != nil {
		return nil, fmt.Errorf("failed to save agent: %w", err)
	}

	// Register in runtime directory if available
	if t.runtimeDir != nil {
		_ = t.runtimeDir.Register(core.NewAgentRuntimeMeta(agent))
	}

	return map[string]any{
		"created":     true,
		"agent_name":  name,
		"role":        role,
		"description": description,
		"skills":      skills, // []string
		"message":     fmt.Sprintf("Agent %q created successfully. Use Delegate to dispatch tasks to it.", name),
	}, nil
}
