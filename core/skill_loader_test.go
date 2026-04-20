package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSkillName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "code-review", false},
		{"valid with numbers", "pdf-processing-2", false},
		{"valid single char", "a", false},
		{"valid max length", string(make([]byte, 64)), false}, // needs actual chars
		{"invalid empty", "", true},
		{"invalid uppercase", "PDF-Processing", true},
		{"invalid start hyphen", "-pdf", true},
		{"invalid end hyphen", "pdf-", true},
		{"invalid consecutive hyphens", "pdf--processing", true},
		{"invalid special chars", "pdf_processing", true},
		{"invalid spaces", "pdf processing", true},
	}

	// Generate a valid max-length name
	maxName := make([]byte, 64)
	for i := range maxName {
		maxName[i] = 'a'
	}
	tests[3].input = string(maxName)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSkillName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSkillName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSkillDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid short", "A short description.", false},
		{"valid long", string(make([]byte, 1024)), false},
		{"invalid empty", "", true},
		{"invalid too long", string(make([]byte, 1025)), true},
	}

	// Generate a valid max-length description
	maxDesc := make([]byte, 1024)
	for i := range maxDesc {
		maxDesc[i] = 'a'
	}
	tests[1].input = string(maxDesc)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSkillDescription(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSkillDescription() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseYamlFrontmatter(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantName     string
		wantDesc     string
		wantLicense  string
		wantBodyHas  string
		wantErr      bool
	}{
		{
			name: "minimal valid",
			input: `---
name: test-skill
description: A test skill.
---
# Instructions
Some instructions here.`,
			wantName:    "test-skill",
			wantDesc:    "A test skill.",
			wantBodyHas: "# Instructions",
		},
		{
			name: "with optional fields",
			input: `---
name: pdf-processing
description: Extract PDF text and fill forms.
license: Apache-2.0
allowed-tools: Bash Read Write
---
## Phase 1
Do something.`,
			wantName:    "pdf-processing",
			wantDesc:    "Extract PDF text and fill forms.",
			wantLicense: "Apache-2.0",
			wantBodyHas: "## Phase 1",
		},
		{
			name: "with metadata",
			input: `---
name: data-analysis
description: Analyze data files.
metadata:
  author: example-org
  version: "1.0"
---
Instructions.`,
			wantName:    "data-analysis",
			wantDesc:    "Analyze data files.",
			wantBodyHas: "Instructions.",
		},
		{
			name: "missing frontmatter",
			input: `# No frontmatter
Just content.`,
			wantErr: true,
		},
		{
			name: "unclosed frontmatter",
			input: `---
name: test
description: test
No closing delimiter`,
			wantErr: true,
		},
		{
			name: "missing name",
			input: `---
description: No name field.
---
Content.`,
			wantErr:     false, // parseYamlFrontmatter only extracts; validation is separate
			wantDesc:    "No name field.",
			wantBodyHas: "Content.",
		},
		{
			name: "missing description",
			input: `---
name: no-desc
---
Content.`,
			wantErr:     false, // parseYamlFrontmatter only extracts; validation is separate
			wantName:    "no-desc",
			wantBodyHas: "Content.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := parseYamlFrontmatter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseYamlFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if fm.Name != tt.wantName {
				t.Errorf("name = %q, want %q", fm.Name, tt.wantName)
			}
			if fm.Description != tt.wantDesc {
				t.Errorf("description = %q, want %q", fm.Description, tt.wantDesc)
			}
			if tt.wantLicense != "" && fm.License != tt.wantLicense {
				t.Errorf("license = %q, want %q", fm.License, tt.wantLicense)
			}
			if tt.wantBodyHas != "" && body == "" {
				t.Error("body is empty, want non-empty")
			}
		})
	}
}

func TestFileSystemSkillLoader(t *testing.T) {
	// Create a temp directory with a skill
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMd := `---
name: my-skill
description: A test skill loaded from filesystem.
---
## Instructions
Do something useful.`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewFileSystemSkillLoader(tmpDir)
	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Load() returned %d skills, want 1", len(skills))
	}

	skill := skills[0]
	if skill.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "my-skill")
	}
	if skill.Description != "A test skill loaded from filesystem." {
		t.Errorf("Description = %q", skill.Description)
	}
	if skill.Source != "filesystem" {
		t.Errorf("Source = %q, want %q", skill.Source, "filesystem")
	}
	if skill.RootDir != skillDir {
		t.Errorf("RootDir = %q, want %q", skill.RootDir, skillDir)
	}
	if skill.Instructions == "" {
		t.Error("Instructions is empty")
	}
}

func TestFileSystemSkillLoader_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewFileSystemSkillLoader(tmpDir)
	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Load() returned %d skills, want 0", len(skills))
	}
}

func TestFileSystemSkillLoader_NonExistentDir(t *testing.T) {
	loader := NewFileSystemSkillLoader("/nonexistent/path")
	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Load() returned %d skills, want 0", len(skills))
	}
}

func TestFileSystemSkillLoader_SkipNonSkillDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory without SKILL.md
	noSkillDir := filepath.Join(tmpDir, "not-a-skill")
	if err := os.MkdirAll(noSkillDir, 0755); err != nil {
		t.Fatal(err)
	}

	loader := NewFileSystemSkillLoader(tmpDir)
	skills, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Load() returned %d skills, want 0", len(skills))
	}
}

func TestFileSystemSkillLoader_InvalidSkillName(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "invalid-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Invalid: uppercase name
	skillMd := `---
name: Invalid-Name
description: Test.
---
Instructions.`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewFileSystemSkillLoader(tmpDir)
	_, err := loader.Load()
	if err == nil {
		t.Error("Load() should return error for invalid skill name")
	}
}
