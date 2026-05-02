package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// SkillLookupFunc looks up a skill by name and returns it if found.
// The reactor provides this to avoid circular imports.
type SkillLookupFunc func(name string) (*core.Skill, error)

// SkillTool lets the LLM load a skill's full instructions on demand.
//
// The tool is called by the LLM when it determines that a listed skill
// (from the SkillsCatalog in System Prompt) is needed for the current task.
// The tool returns the full skill instructions via tool result, which the
// LLM sees in the next round's Observation.
type SkillTool struct {
	lookup SkillLookupFunc
}

// NewSkillTool creates a SkillTool.
// lookup is provided by the reactor.
func NewSkillTool(lookup SkillLookupFunc) *SkillTool {
	return &SkillTool{lookup: lookup}
}

func (t *SkillTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "Skill",
		Description: "Load a specialized skill by name. Skills are listed in the system prompt with their descriptions. Call this tool when a skill can help with the current task.",
		Prompt: `Load a specialized capability (skill) by name.

When you identify a skill from the available capabilities list that matches the current task, call this tool to load its full instructions. The skill's instructions will be provided in the tool result.

Always load a skill before attempting tasks that require its domain expertise.

Available skills are listed in the system prompt under "## 可用的技能".`,
		Tags: []string{"skill", "capability"},
		Parameters: []core.Parameter{
			{
				Name:        "name",
				Type:        "string",
				Description: "Name of the skill to load (from the available capabilities list).",
				Required:    true,
			},
		},
		IsReadOnly: true,
	}
}

func (t *SkillTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	name, _ := params["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("skill name is required")
	}

	skill, err := t.lookup(name)
	if err != nil {
		return nil, fmt.Errorf("skill %q not found: %w", name, err)
	}

	// Build a comprehensive skill description for the tool result
	result := fmt.Sprintf("=== Skill: %s ===\n\nDescription: %s\n\nInstructions:\n%s",
		skill.Name, skill.Description, skill.Instructions)

	if skill.AllowedTools != "" {
		result += fmt.Sprintf("\n\nAllowed tools: %s", skill.AllowedTools)
	}
	if skill.RootDir != "" {
		result += fmt.Sprintf("\n\nResource base path: %s", skill.RootDir)
	}

	return map[string]any{
		"skill_name":  skill.Name,
		"description": skill.Description,
		"instructions": skill.Instructions,
		"allowed_tools": skill.AllowedTools,
		"content":     result,
		"loaded":      true,
	}, nil
}
