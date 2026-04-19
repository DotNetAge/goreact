package reactor

import (
	"io/fs"

	"github.com/DotNetAge/goreact/core"
)

// skillTemplates maps skill names to their embedded template file paths.
var skillTemplates = map[string]string{
	"Bug Hunter": "prompts/skill_debug.tmpl",
	"Architect":  "prompts/skill_architect.tmpl",
	"Batch":      "prompts/skill_batch.tmpl",
	"Verify":     "prompts/skill_verify.tmpl",
	"Remember":   "prompts/skill_remember.tmpl",
	"Stuck":      "prompts/skill_stuck.tmpl",
	"Simplify":   "prompts/skill_simplify.tmpl",
}

// skillMeta holds the metadata for each bundled skill (name, description, tools, triggers).
type skillMeta struct {
	Name         string
	Description  string
	Tools        []string
	TriggerRules []string
}

// bundledSkillDefs defines the metadata for all built-in skills.
var bundledSkillDefs = []skillMeta{
	{
		Name:         "Bug Hunter",
		Description:  "Expert SOP for locating, isolating and fixing complex bugs.",
		Tools:        []string{"grep", "glob", "bash", "task_create", "read_file"},
		TriggerRules: []string{"bug", "fix", "error", "crash", "failed", "debug"},
	},
	{
		Name:         "Architect",
		Description:  "High-level orchestration for system design and major migrations.",
		Tools:        []string{"glob", "grep", "task_create", "task_list", "task_result", "todo_write"},
		TriggerRules: []string{"architecture", "design", "refactor", "migrate"},
	},
	{
		Name:         "Batch",
		Description:  "Parallel orchestration of large-scale mechanical changes.",
		Tools:        []string{"grep", "task_create", "task_list", "task_result", "bash"},
		TriggerRules: []string{"batch", "bulk", "replace all", "migrate all"},
	},
	{
		Name:         "Verify",
		Description:  "Rigorous verification of changes through testing and execution.",
		Tools:        []string{"bash", "todo_write"},
		TriggerRules: []string{"verify", "test", "check", "qa"},
	},
	{
		Name:         "Remember",
		Description:  "Manage project conventions, instructions, and shared memory.",
		Tools:        []string{"grep", "bash", "read_file"},
		TriggerRules: []string{"remember", "convention", "instruction", "memory", "documentation"},
	},
	{
		Name:         "Stuck",
		Description:  "Strategy to break free when the agent is repeating actions or failing.",
		Tools:        []string{"grep", "glob", "bash"},
		TriggerRules: []string{"stuck", "loop", "repeat", "failing"},
	},
	{
		Name:         "Simplify",
		Description:  "Post-implementation cleanup to ensure code quality and simplicity.",
		Tools:        []string{"replace_in_file", "bash", "read_file"},
		TriggerRules: []string{"simplify", "cleanup", "refactor", "polish"},
	},
}

// RegisterBundledSkills registers common high-level skills based on CludeCode patterns.
// Skill instructions are loaded from embedded .tmpl files under prompts/.
func RegisterBundledSkills(registry core.SkillRegistry) {
	for _, meta := range bundledSkillDefs {
		tmplPath, ok := skillTemplates[meta.Name]
		if !ok {
			continue
		}

		instructions, err := loadSkillTemplate(tmplPath)
		if err != nil {
			// Log but don't panic — embedded files should always be available
			instructions = "ERROR: failed to load skill template: " + tmplPath
		}

		_ = registry.RegisterSkill(&core.Skill{
			Name:         meta.Name,
			Description:  meta.Description,
			Instructions: instructions,
			Tools:        meta.Tools,
			TriggerRules: meta.TriggerRules,
		})
	}
}

// loadSkillTemplate reads a skill prompt from the embedded filesystem.
func loadSkillTemplate(path string) (string, error) {
	data, err := fs.ReadFile(promptTemplates, path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
