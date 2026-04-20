package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SkillLoader defines the interface for loading skills from a source.
type SkillLoader interface {
	// Load discovers and loads all skills from the source.
	// Returns a list of loaded skills or an error.
	Load() ([]*Skill, error)
}

// --- SKILL.md Frontmatter ---

// skillFrontmatter represents the YAML frontmatter of a SKILL.md file.
type skillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`
}

// --- Validation ---

// ValidateSkillName checks if a skill name conforms to the Agent Skills spec.
func ValidateSkillName(name string) error {
	if len(name) < 1 || len(name) > 64 {
		return fmt.Errorf("skill name must be 1-64 characters, got %d", len(name))
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("skill name must not start or end with a hyphen: %q", name)
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("skill name must not contain consecutive hyphens: %q", name)
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("skill name must only contain lowercase letters, numbers, and hyphens: %q", name)
		}
	}
	return nil
}

// ValidateSkillDescription checks if a skill description conforms to the spec.
func ValidateSkillDescription(desc string) error {
	if len(desc) < 1 || len(desc) > 1024 {
		return fmt.Errorf("skill description must be 1-1024 characters, got %d", len(desc))
	}
	return nil
}

// --- File System Skill Loader ---

// FileSystemSkillLoader loads skills from a directory on the filesystem.
// Each subdirectory containing a SKILL.md file is loaded as a skill.
type FileSystemSkillLoader struct {
	RootDir string // Root directory containing skill subdirectories.
}

// NewFileSystemSkillLoader creates a loader that reads skills from the given directory.
func NewFileSystemSkillLoader(rootDir string) *FileSystemSkillLoader {
	return &FileSystemSkillLoader{RootDir: rootDir}
}

// Load scans the root directory for subdirectories containing SKILL.md files.
func (l *FileSystemSkillLoader) Load() ([]*Skill, error) {
	entries, err := os.ReadDir(l.RootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // empty skill directory is OK
		}
		return nil, fmt.Errorf("failed to read skill directory %q: %w", l.RootDir, err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(l.RootDir, entry.Name())
		skill, err := loadSkillFromDir(skillDir, "filesystem")
		if err != nil {
			return nil, fmt.Errorf("failed to load skill from %q: %w", skillDir, err)
		}
		if skill != nil {
			skills = append(skills, skill)
		}
	}
	return skills, nil
}

// --- Bundled (Embedded) Skill Loader ---

// BundledSkillLoader loads skills from an embedded filesystem (embed.FS).
// The FS should have the structure: skills/<skill-name>/SKILL.md
type BundledSkillLoader struct {
	FS       fs.FS
	SkillsDir string // Subdirectory within FS that contains skill directories (e.g., "skills").
}

// NewBundledSkillLoader creates a loader that reads skills from an embedded FS.
func NewBundledSkillLoader(fsys fs.FS, skillsDir string) *BundledSkillLoader {
	return &BundledSkillLoader{FS: fsys, SkillsDir: skillsDir}
}

// Load scans the embedded filesystem for skill directories containing SKILL.md files.
func (l *BundledSkillLoader) Load() ([]*Skill, error) {
	entries, err := fs.ReadDir(l.FS, l.SkillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded skill directory %q: %w", l.SkillsDir, err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := l.SkillsDir + "/" + entry.Name()
		// For embedded skills, we read the file content directly from FS
		skill, err := loadSkillFromEmbedded(l.FS, skillDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load bundled skill %q: %w", entry.Name(), err)
		}
		if skill != nil {
			skills = append(skills, skill)
		}
	}
	return skills, nil
}

// --- Internal loading functions ---

// loadSkillFromDir loads a skill from a filesystem directory containing a SKILL.md file.
func loadSkillFromDir(dir string, source string) (*Skill, error) {
	skillMdPath := filepath.Join(dir, "SKILL.md")

	data, err := os.ReadFile(skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no SKILL.md, skip this directory
		}
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	return parseSkillMd(data, dir, source)
}

// loadSkillFromEmbedded loads a skill from an embedded filesystem.
func loadSkillFromEmbedded(fsys fs.FS, skillDir string) (*Skill, error) {
	skillMdPath := skillDir + "/SKILL.md"

	data, err := fs.ReadFile(fsys, skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read embedded SKILL.md: %w", err)
	}

	return parseSkillMd(data, "", "bundled")
}

// parseSkillMd parses a SKILL.md file (YAML frontmatter + Markdown body) into a Skill.
func parseSkillMd(data []byte, rootDir string, source string) (*Skill, error) {
	content := string(data)
	fm, body, err := parseYamlFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md frontmatter: %w", err)
	}

	if fm.Name == "" {
		return nil, fmt.Errorf("SKILL.md is missing required 'name' field in frontmatter")
	}
	if fm.Description == "" {
		return nil, fmt.Errorf("SKILL.md is missing required 'description' field in frontmatter")
	}

	if err := ValidateSkillName(fm.Name); err != nil {
		return nil, err
	}
	if err := ValidateSkillDescription(fm.Description); err != nil {
		return nil, err
	}

	instructions := strings.TrimSpace(body)

	// Resolve template variables in instructions.
	// {base_dir} → the absolute path of the skill directory on disk.
	// For bundled skills (rootDir is empty), the variable is replaced with an empty string.
	resolved := instructions
	if strings.Contains(resolved, "{base_dir}") {
		resolved = strings.ReplaceAll(resolved, "{base_dir}", rootDir)
	}
	if strings.Contains(resolved, "{skill_name}") {
		resolved = strings.ReplaceAll(resolved, "{skill_name}", fm.Name)
	}

	return &Skill{
		Name:          fm.Name,
		Description:   fm.Description,
		License:       fm.License,
		Compatibility: fm.Compatibility,
		Metadata:      fm.Metadata,
		AllowedTools:  fm.AllowedTools,
		Instructions:  resolved,
		RootDir:       rootDir,
		Source:        source,
	}, nil
}
