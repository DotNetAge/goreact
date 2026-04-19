package core

import (
	"fmt"
	"time"
)

// Skill represents a reusable capability unit
type Skill struct {
	Name          string         `json:"name" yaml:"name"`                   // Name is the skill name (1-64 chars, lowercase letters, numbers, hyphens)
	Description   string         `json:"description" yaml:"description"`     // Describes what the skill does and when to use it (1-1024 chars)
	Path          string         `json:"path" yaml:"path"`                   // Path is the skill directory path
	License       string         `json:"license" yaml:"license"`             // License is the skill license
	Compatibility string         `json:"compatibility" yaml:"compatibility"` // Compatibility is the environment requirements
	AllowedTools  []string       `json:"allowed_tools" yaml:"allowed_tools"` // AllowedTools are the tools this skill can use (space-separated in frontmatter)
	Metadata      map[string]any `json:"metadata" yaml:"metadata"`           // Metadata contains additional metadata from frontmatter
	Content       string         `json:"content" yaml:"content"`             // Content is the skill content (SKILL.md body content)
	ContentHash   string         `json:"content_hash" yaml:"content_hash"`   // ContentHash is the hash of SKILL.md content for cache invalidation
	CreatedAt     time.Time      `json:"created_at" yaml:"created_at"`       // CreatedAt is the creation timestamp
	UpdatedAt     time.Time      `json:"updated_at" yaml:"updated_at"`       // UpdatedAt is the last update timestamp
}

// ComputeContentHash computes the content hash from template and steps
func (s *Skill) ComputeContentHash() string {
	// TODO: Replace this Simple hash computation - in production would use crypto/sha256
	hash := fmt.Sprintf("%s-%d-%d", s.Name)
	return hash
}

// SkillParser parses SKILL.md files
type SkillParser struct{}

// NewSkillParser creates a new SkillParser
func NewSkillParser() *SkillParser {
	return &SkillParser{}
}

// Parse parses a SKILL.md content into a Skill
func (p *SkillParser) Parse(content string, path string) (*Skill, error) {
	frontmatter, body, err := p.splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	skill := &Skill{
		Name:          frontmatter.Name,
		Description:   frontmatter.Description,
		Path:          path,
		License:       frontmatter.License,
		Compatibility: frontmatter.Compatibility,
		AllowedTools:  frontmatter.AllowedTools,
		Metadata:      frontmatter.Metadata,
		Content:       body,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Compute content hash
	skill.ContentHash = skill.ComputeContentHash()

	return skill, nil
}

// splitFrontmatter splits content into frontmatter and body
func (p *SkillParser) splitFrontmatter(content string) (*Skill, string, error) {
	// TODO: Implement proper YAML frontmatter parsing
	// Simple implementation - would use proper YAML parser in production
	// Look for --- delimiters
	return &Skill{
		Name:        "unknown",
		Description: "",
	}, content, nil
}
