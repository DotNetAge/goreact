package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// SkillCreateTool creates new skills that can be registered with the reactor's
// skill registry. A skill consists of a SKILL.md file with YAML frontmatter
// and instructional content that gets injected into the LLM's context when activated.
type SkillCreateTool struct {
	// SkillDirs is the list of directories to search for existing skills.
	// New skills are created in the first writable directory.
	SkillDirs []string
}

// NewSkillCreateTool creates a new SkillCreateTool.
func NewSkillCreateTool() core.FuncTool {
	return &SkillCreateTool{}
}

// SetSkillDirs sets the skill directories for this tool.
func (t *SkillCreateTool) SetSkillDirs(dirs []string) {
	t.SkillDirs = dirs
}

func (t *SkillCreateTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name: "skill_create",
		Description: `Create a new skill that can be used by the agent. A skill is a reusable knowledge module with YAML frontmatter and instructional content.

Skills are activated based on the agent's intent/context and inject specialized instructions into the LLM prompt.

SKILL.md format:
---
name: skill-name
description: Brief description of what the skill does
trigger: keywords,that,trigger,this,skill
---

Detailed instructions for the agent when this skill is active...

Parameters:
- name: unique skill identifier (lowercase, hyphens, required)
- description: what the skill does (required)
- trigger: comma-separated keywords that activate this skill (optional)
- instructions: the skill's instructional content (required)
- category: skill category for organization (optional)
- save_to: specific directory path to save the skill (optional, auto-generated if omitted)`,
		Parameters: []core.Parameter{
			{
				Name:        "name",
				Type:        "string",
				Description: "Unique skill identifier (lowercase, hyphens allowed, e.g. 'code-review').",
				Required:    true,
			},
			{
				Name:        "description",
				Type:        "string",
				Description: "Brief description of what the skill does and when it should be activated.",
				Required:    true,
			},
			{
				Name:        "instructions",
				Type:        "string",
				Description: "Detailed instructions for the agent when this skill is active. This is the core content of the skill.",
				Required:    true,
			},
			{
				Name:        "trigger",
				Type:        "string",
				Description: "Comma-separated keywords that trigger this skill activation (e.g., 'code-review,refactor,quality').",
				Required:    false,
			},
			{
				Name:        "category",
				Type:        "string",
				Description: "Category for organizing skills (e.g., 'development', 'analysis', 'communication').",
				Required:    false,
			},
			{
				Name:        "save_to",
				Type:        "string",
				Description: "Directory path to save the skill. If omitted, saves to the first available skill directory.",
				Required:    false,
			},
		},
	}
}

func (t *SkillCreateTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	name, err := ValidateRequiredString(params, "name")
	if err != nil {
		return nil, err
	}
	description, err := ValidateRequiredString(params, "description")
	if err != nil {
		return nil, err
	}
	instructions, err := ValidateRequiredString(params, "instructions")
	if err != nil {
		return nil, err
	}
	trigger, _ := params["trigger"].(string)
	category, _ := params["category"].(string)
	saveTo, _ := params["save_to"].(string)

	// Validate name
	if !isValidSkillName(name) {
		return nil, fmt.Errorf("invalid skill name %q: must be lowercase, use hyphens instead of spaces, and start with a letter", name)
	}

	// Determine save directory
	skillDir := saveTo
	if skillDir == "" {
		skillDir = t.resolveSkillDir()
		if skillDir == "" {
			return nil, fmt.Errorf("no skill directory available. Use 'save_to' parameter to specify a directory")
		}
	}

	// Create skill directory
	skillPath := filepath.Join(skillDir, name)
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Check if SKILL.md already exists
	skillFile := filepath.Join(skillPath, "SKILL.md")
	if _, err := os.Stat(skillFile); err == nil {
		return nil, fmt.Errorf("skill %q already exists at %s", name, skillFile)
	}

	// Build SKILL.md content
	content := buildSkillMarkdown(name, description, trigger, category, instructions)

	// Write file
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	return fmt.Sprintf("Skill %q created successfully at %s\n\n---\n%s\n---\n\nThe skill will be available after the reactor reloads its skill registry. You can use 'skill_list' to verify it's registered.", name, skillFile, content), nil
}

func (t *SkillCreateTool) resolveSkillDir() string {
	// Try configured skill directories first
	for _, dir := range t.SkillDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	// Default: ./skills relative to cwd
	if cwd, err := os.Getwd(); err == nil {
		defaultDir := filepath.Join(cwd, "skills")
		if info, err := os.Stat(defaultDir); err == nil && info.IsDir() {
			return defaultDir
		}
	}
	return ""
}

func isValidSkillName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	if name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

func buildSkillMarkdown(name, description, trigger, category, instructions string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", name)
	fmt.Fprintf(&sb, "description: %s\n", description)
	if trigger != "" {
		fmt.Fprintf(&sb, "trigger: %s\n", strings.ToLower(trigger))
	}
	if category != "" {
		fmt.Fprintf(&sb, "category: %s\n", strings.ToLower(category))
	}
	sb.WriteString("---\n\n")

	sb.WriteString(instructions)
	if !strings.HasSuffix(instructions, "\n") {
		sb.WriteString("\n")
	}

	return sb.String()
}

// --- Skill List Tool ---

// SkillListTool lists all registered and available skills.
type SkillListTool struct {
	// SkillDirs is the list of directories to search for skills.
	SkillDirs []string
}

// NewSkillListTool creates a new SkillListTool.
func NewSkillListTool() core.FuncTool {
	return &SkillListTool{}
}

// SetSkillDirs sets the skill directories for this tool.
func (t *SkillListTool) SetSkillDirs(dirs []string) {
	t.SkillDirs = dirs
}

func (t *SkillListTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "skill_list",
		Description: "List all available skills with their names, descriptions, and trigger keywords. Skills are reusable knowledge modules that inject specialized instructions when activated.",
		IsReadOnly:  true,
		Parameters: []core.Parameter{
			{
				Name:        "category",
				Type:        "string",
				Description: "Filter skills by category (optional).",
				Required:    false,
			},
		},
	}
}

func (t *SkillListTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	category, _ := params["category"].(string)

	dirs := t.SkillDirs
	if len(dirs) == 0 {
		if cwd, err := os.Getwd(); err == nil {
			dirs = []string{filepath.Join(cwd, "skills")}
		}
	}

	type skillEntry struct {
		Name        string
		Description string
		Trigger     string
		Category    string
		Path        string
	}

	var skills []skillEntry
	seen := make(map[string]bool)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
			if seen[entry.Name()] {
				continue
			}

			data, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			fm := parseSkillFrontmatter(string(data))

			if category != "" && fm.Category != category {
				continue
			}

			skills = append(skills, skillEntry{
				Name:        entry.Name(),
				Description: fm.Description,
				Trigger:     fm.Trigger,
				Category:    fm.Category,
				Path:        skillFile,
			})
			seen[entry.Name()] = true
		}
	}

	if len(skills) == 0 {
		if category != "" {
			return fmt.Sprintf("No skills found in category %q.", category), nil
		}
		return "No skills found. Use 'skill_create' to create a new skill.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d skill(s):\n\n", len(skills))
	for _, s := range skills {
		fmt.Fprintf(&sb, "## %s\n", s.Name)
		fmt.Fprintf(&sb, "  Description: %s\n", s.Description)
		if s.Trigger != "" {
			fmt.Fprintf(&sb, "  Trigger: %s\n", s.Trigger)
		}
		if s.Category != "" {
			fmt.Fprintf(&sb, "  Category: %s\n", s.Category)
		}
		fmt.Fprintf(&sb, "  Path: %s\n\n", s.Path)
	}

	return sb.String(), nil
}

func parseSkillFrontmatter(content string) skillFrontmatterInfo {
	info := skillFrontmatterInfo{}
	if !strings.HasPrefix(content, "---") {
		return info
	}
	end := strings.Index(content[3:], "---")
	if end == -1 {
		return info
	}
	fm := content[3 : 3+end]

	for line := range strings.SplitSeq(fm, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			switch strings.ToLower(key) {
			case "name":
				info.Name = val
			case "description":
				info.Description = val
			case "trigger":
				info.Trigger = val
			case "category":
				info.Category = val
			}
		}
	}
	return info
}

type skillFrontmatterInfo struct {
	Name        string
	Description string
	Trigger     string
	Category    string
}
