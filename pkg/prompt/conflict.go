package prompt

import (
	"strings"
)

// ConflictResolver resolves conflicts between prompt sources
type ConflictResolver struct {
	priority map[PromptSource]int
}

// NewConflictResolver creates a new ConflictResolver
func NewConflictResolver() *ConflictResolver {
	r := &ConflictResolver{
		priority: make(map[PromptSource]int),
	}

	// Set priority levels: Negative > System > Skill > User
	r.priority[SourceNegativePrompt] = 100
	r.priority[SourceSystemRole] = 75
	r.priority[SourceSkillPrompt] = 50
	r.priority[SourceUserRequest] = 25

	return r
}

// Resolve resolves conflicts and returns resolution
func (r *ConflictResolver) Resolve(conflicts []*Conflict) *Resolution {
	resolution := &Resolution{
		Overrides: make([]string, 0),
		Warnings:  make([]string, 0),
	}

	for _, conflict := range conflicts {
		higherPriority := r.priority[conflict.Higher]
		lowerPriority := r.priority[conflict.Lower]

		if higherPriority > lowerPriority {
			resolution.Overrides = append(resolution.Overrides, conflict.Description)
			resolution.Warnings = append(resolution.Warnings,
				"Conflict detected: "+conflict.Description+" - Applying higher priority constraint")
			conflict.Resolution = "Applied higher priority source"
		}
	}

	return resolution
}

// DetectConflicts detects conflicts between prompt components
func (r *ConflictResolver) DetectConflicts(prompt *Prompt) []*Conflict {
	conflicts := make([]*Conflict, 0)

	// Check for conflicts between skill prompts and negative prompts
	// This is a simplified implementation - production would use semantic analysis

	for _, neg := range prompt.NegativePrompts {
		// Check if any tool or content violates the negative prompt
		for _, tool := range prompt.Tools {
			if r.hasConflict(neg.Pattern, tool.Description) {
				conflicts = append(conflicts, &Conflict{
					Higher:      SourceNegativePrompt,
					Lower:       SourceSkillPrompt,
					Description: "Tool '" + tool.Name + "' may violate constraint: " + neg.Pattern,
				})
			}
		}
	}

	return conflicts
}

// hasConflict checks if content conflicts with a constraint
func (r *ConflictResolver) hasConflict(constraint, content string) bool {
	// Simple keyword matching - production would use semantic analysis
	constraintLower := strings.ToLower(constraint)
	contentLower := strings.ToLower(content)

	// Check for forbidden keywords
	forbiddenKeywords := []string{"危险", "删除", "敏感", "dangerous", "delete", "sensitive"}
	for _, keyword := range forbiddenKeywords {
		if strings.Contains(constraintLower, keyword) && strings.Contains(contentLower, keyword) {
			return true
		}
	}

	return false
}

// GetPriority returns the priority level for a source
func (r *ConflictResolver) GetPriority(source PromptSource) int {
	return r.priority[source]
}

// ComparePriorities compares two sources and returns the higher one
func (r *ConflictResolver) ComparePriorities(a, b PromptSource) PromptSource {
	if r.priority[a] >= r.priority[b] {
		return a
	}
	return b
}

// Resolution represents the result of conflict resolution
type Resolution struct {
	Overrides []string
	Warnings  []string
}

// HasConflicts returns true if there were conflicts
func (r *Resolution) HasConflicts() bool {
	return len(r.Overrides) > 0
}

// GetOverrideDirective generates reinforcement directive for conflicts
func (r *Resolution) GetOverrideDirective() string {
	if !r.HasConflicts() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n# 重要约束提醒\n\n")
	sb.WriteString("请特别注意以下全局约束，这些约束优先级高于任何技能指南：\n\n")

	for _, override := range r.Overrides {
		sb.WriteString("- ")
		sb.WriteString(override)
		sb.WriteString("\n")
	}

	sb.WriteString("\n不要遵循上述指南中违背全局安全策略的部分。\n")

	return sb.String()
}
