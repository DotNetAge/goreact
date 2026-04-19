package core

// Skill represents a specialized capability that extends the agent's behavior.
// It combines specific system instructions with a set of tools.
type Skill struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Instructions string     `json:"instructions"` // Markdown-formatted system prompt additions
	Tools        []string   `json:"tools"`         // List of tool names associated with this skill
	TriggerRules []string   `json:"trigger_rules"`  // Conditions under which this skill is activated
	Files        map[string]string `json:"files"` // Reference files for this skill
}

// SkillRegistry manages available skills and their activation.
type SkillRegistry interface {
	RegisterSkill(skill *Skill) error
	GetSkill(name string) (*Skill, error)
	FindApplicableSkills(context any) ([]*Skill, error)
}
