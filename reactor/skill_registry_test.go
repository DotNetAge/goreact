package reactor

import (
	"testing"

	"github.com/DotNetAge/goreact/core"
)

func TestNewSkillRegistry(t *testing.T) {
	r := NewSkillRegistry()
	if r == nil {
		t.Fatal("NewSkillRegistry() returned nil")
	}
}

func TestDefaultSkillRegistry_RegisterAndGet(t *testing.T) {
	r := NewSkillRegistry()

	skill := &core.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Instructions: "# Instructions\nDo something.",
	}

	if err := r.RegisterSkill(skill); err != nil {
		t.Fatalf("RegisterSkill() error = %v", err)
	}

	got, err := r.GetSkill("test-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if got.Name != skill.Name {
		t.Errorf("GetSkill() Name = %q, want %q", got.Name, skill.Name)
	}
}

func TestDefaultSkillRegistry_GetNotFound(t *testing.T) {
	r := NewSkillRegistry()

	_, err := r.GetSkill("nonexistent")
	if err != core.ErrSkillNotFound {
		t.Errorf("GetSkill() error = %v, want %v", err, core.ErrSkillNotFound)
	}
}

func TestDefaultSkillRegistry_RegisterEmptyName(t *testing.T) {
	r := NewSkillRegistry()

	err := r.RegisterSkill(&core.Skill{Name: ""})
	if err == nil {
		t.Error("RegisterSkill() with empty name should return error")
	}
}

func TestDefaultSkillRegistry_RegisterNil(t *testing.T) {
	r := NewSkillRegistry()

	err := r.RegisterSkill(nil)
	if err == nil {
		t.Error("RegisterSkill() with nil should return error")
	}
}

func TestDefaultSkillRegistry_ListSkills(t *testing.T) {
	r := NewSkillRegistry()

	skills := r.ListSkills()
	if len(skills) != 0 {
		t.Errorf("ListSkills() on empty registry = %d, want 0", len(skills))
	}

	_ = r.RegisterSkill(&core.Skill{Name: "skill-1", Description: "First"})
	_ = r.RegisterSkill(&core.Skill{Name: "skill-2", Description: "Second"})

	skills = r.ListSkills()
	if len(skills) != 2 {
		t.Errorf("ListSkills() = %d, want 2", len(skills))
	}
}

func TestDefaultSkillRegistry_FindApplicableSkills(t *testing.T) {
	r := NewSkillRegistry()

	_ = r.RegisterSkill(&core.Skill{
		Name:        "bug-hunter",
		Description: "Expert SOP for locating, isolating and fixing complex bugs. Use when debugging.",
		Instructions: "# Debug",
	})
	_ = r.RegisterSkill(&core.Skill{
		Name:        "architect",
		Description: "High-level orchestration for system design and major migrations.",
		Instructions: "# Architect",
	})

	// Test with matching intent
	intent := &Intent{
		Type:    "coding",
		Topic:   "debug this bug",
		Summary: "There is a bug in the code",
	}

	results, err := r.FindApplicableSkills(intent)
	if err != nil {
		t.Fatalf("FindApplicableSkills() error = %v", err)
	}

	found := false
	for _, s := range results {
		if s.Name == "bug-hunter" {
			found = true
			break
		}
	}
	if !found {
		t.Error("FindApplicableSkills() should find bug-hunter for debug intent")
	}

	// Test with non-matching intent
	intent2 := &Intent{
		Type:    "chat",
		Topic:   "hello world",
		Summary: "Just saying hi",
	}
	results2, _ := r.FindApplicableSkills(intent2)
	if len(results2) > 0 {
		t.Errorf("FindApplicableSkills() for non-matching intent = %d, want 0", len(results2))
	}

	// Test with nil intent
	results3, _ := r.FindApplicableSkills(nil)
	if len(results3) != 0 {
		t.Errorf("FindApplicableSkills() with nil = %d, want 0", len(results3))
	}

	// Test with wrong type
	results4, _ := r.FindApplicableSkills("not an intent")
	if len(results4) != 0 {
		t.Errorf("FindApplicableSkills() with wrong type = %d, want 0", len(results4))
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		input string
		min   int // minimum expected keywords
	}{
		{"Bug Hunter debug fix", 2},
		{"the a an is are", 0},        // all stop words
		{"system design architecture", 3},
	}

	for _, tt := range tests {
		keywords := extractKeywords(tt.input)
		if len(keywords) < tt.min {
			t.Errorf("extractKeywords(%q) = %d keywords, want >= %d", tt.input, len(keywords), tt.min)
		}
	}
}

func TestMatchSkill_ImprovedPrecision(t *testing.T) {
	skills := []*core.Skill{
		{
			Name:        "bug-hunter",
			Description: "Expert SOP for locating, isolating and fixing complex bugs. Use when debugging.",
		},
		{
			Name:        "architect",
			Description: "High-level orchestration for system design and major migrations.",
		},
		{
			Name:        "simplify",
			Description: "Simplifies complex code by reducing cognitive load and improving readability.",
		},
		{
			Name:        "verify",
			Description: "Verifies code correctness through systematic testing and validation.",
		},
	}

	tests := []struct {
		name           string
		intentText     string
		expectedMatch  string // empty = no match expected
	}{
		{
			name:          "Strong match - multiple keywords",
			intentText:    "debug this bug in my code",
			expectedMatch: "bug-hunter",
		},
		{
			name:          "Strong match - skill name substring",
			intentText:    "I need to architect a new system",
			expectedMatch: "architect",
		},
		{
			name:          "Weak single keyword should NOT match (below threshold)",
			intentText:    "fix the code",
			expectedMatch: "",
		},
		{
			name:          "Long specific word alone can match",
			intentText:    "help me simplify this code module for readability",
			expectedMatch: "simplify",
		},
		{
			name:          "No relevant keywords",
			intentText:    "hello world just chatting",
			expectedMatch: "",
		},
		{
			name:          "Two short words meet threshold",
			intentText:    "verify code works correctly now",
			expectedMatch: "verify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matched *core.Skill
			for _, s := range skills {
				if matchSkill(s, tt.intentText) {
					matched = s
					break
				}
			}
			if tt.expectedMatch == "" {
				if matched != nil {
					t.Errorf("expected no match, got %q", matched.Name)
				}
			} else {
				if matched == nil || matched.Name != tt.expectedMatch {
					t.Errorf("expected match %q, got %v", tt.expectedMatch, matched)
				}
			}
		})
	}
}
