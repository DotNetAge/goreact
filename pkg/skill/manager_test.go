package skill

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultManager_New(t *testing.T) {
	m := DefaultManager()
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
	if m.skills == nil {
		t.Error("Expected skills map to be initialized")
	}
	if m.archivedSkills == nil {
		t.Error("Expected archivedSkills map to be initialized")
	}
	if m.selectionMode != Hybrid {
		t.Errorf("Expected Hybrid mode, got %v", m.selectionMode)
	}
	if m.topN != 3 {
		t.Errorf("Expected topN 3, got %d", m.topN)
	}
}

func TestDefaultManager_WithOptions(t *testing.T) {
	m := DefaultManager(
		WithSelectionMode(KeywordOnly),
		WithTopN(5),
	)

	if m.selectionMode != KeywordOnly {
		t.Errorf("Expected KeywordOnly, got %v", m.selectionMode)
	}
	if m.topN != 5 {
		t.Errorf("Expected topN 5, got %d", m.topN)
	}
}

func TestDefaultManager_RegisterSkill(t *testing.T) {
	m := DefaultManager()
	skill := NewSkill("test-skill", "A test skill description for testing")
	skill.Instructions = "Step 1: Do this"

	err := m.RegisterSkill(skill)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	retrieved, err := m.GetSkill("test-skill")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved.Name != "test-skill" {
		t.Errorf("Expected 'test-skill', got %q", retrieved.Name)
	}
}

func TestDefaultManager_RegisterSkill_Validation(t *testing.T) {
	m := DefaultManager()

	t.Run("nil skill", func(t *testing.T) {
		err := m.RegisterSkill(nil)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		skill := NewSkill("", "desc")
		skill.Instructions = "do something"
		err := m.RegisterSkill(skill)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty description", func(t *testing.T) {
		skill := NewSkill("test", "")
		skill.Instructions = "do something"
		err := m.RegisterSkill(skill)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty instructions", func(t *testing.T) {
		skill := NewSkill("test", "desc")
		skill.Instructions = ""
		err := m.RegisterSkill(skill)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("invalid name characters", func(t *testing.T) {
		skill := NewSkill("Test_Skill", "desc")
		skill.Instructions = "do something"
		err := m.RegisterSkill(skill)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestDefaultManager_GetSkill(t *testing.T) {
	m := DefaultManager()
	skill := NewSkill("test", "desc")
	skill.Instructions = "do something"
	m.RegisterSkill(skill)

	t.Run("existing skill", func(t *testing.T) {
		retrieved, err := m.GetSkill("test")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retrieved.Name != "test" {
			t.Errorf("Expected 'test', got %q", retrieved.Name)
		}
	})

	t.Run("non-existing skill", func(t *testing.T) {
		_, err := m.GetSkill("nonexistent")
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestDefaultManager_ListSkills(t *testing.T) {
	m := DefaultManager()
	skill1 := NewSkill("skill1", "desc1")
	skill1.Instructions = "do something"
	skill2 := NewSkill("skill2", "desc2")
	skill2.Instructions = "do something"
	m.RegisterSkill(skill1)
	m.RegisterSkill(skill2)

	skills := m.ListSkills()
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}
}

func TestDefaultManager_SelectSkill(t *testing.T) {
	m := DefaultManager(
		WithSelectionMode(KeywordOnly),
	)
	mathSkill := NewSkill("math-wizard", "A skill for mathematical calculations")
	mathSkill.Instructions = "do math"
	m.RegisterSkill(mathSkill)
	searchSkill := NewSkill("search-expert", "A skill for web searching")
	searchSkill.Instructions = "do search"
	m.RegisterSkill(searchSkill)

	t.Run("no skills available", func(t *testing.T) {
		empty := DefaultManager()
		_, err := empty.SelectSkill("test")
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("select by keyword", func(t *testing.T) {
		skill, err := m.SelectSkill("calculate 2+2")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if skill == nil {
			t.Error("Expected skill to be selected")
		}
	})
}

func TestDefaultManager_SelectSkill_SemanticMode(t *testing.T) {
	m := DefaultManager(
		WithSelectionMode(SemanticOnly),
	)
	mathSkill := NewSkill("math", "math skill")
	mathSkill.Instructions = "do math"
	m.RegisterSkill(mathSkill)

	_, err := m.SelectSkill("calculate")
	if err == nil {
		t.Error("Expected error (LLM client required for semantic)")
	}
}

func TestDefaultManager_filterCandidatesByKeyword(t *testing.T) {
	m := DefaultManager()
	mathSkill := NewSkill("math", "mathematical calculator")
	mathSkill.Instructions = "do math"
	m.RegisterSkill(mathSkill)
	searchSkill := NewSkill("search", "web search")
	searchSkill.Instructions = "do search"
	m.RegisterSkill(searchSkill)

	candidates, err := m.filterCandidatesByKeyword("calculate math", 2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(candidates) == 0 {
		t.Error("Expected at least one candidate")
	}
}

func TestDefaultManager_filterCandidatesByKeyword_NoMatch(t *testing.T) {
	m := DefaultManager()
	mathSkill := NewSkill("math", "mathematical calculator")
	mathSkill.Instructions = "do math"
	m.RegisterSkill(mathSkill)

	_, err := m.filterCandidatesByKeyword("xyzabc123", 2)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestDefaultManager_calculateKeywordScore(t *testing.T) {
	m := DefaultManager()
	skill := NewSkill("math-wizard", "A skill for mathematical calculations")
	skill.Instructions = "do math"
	skill.Statistics = &SkillStatistics{OverallScore: 0.5}

	score := m.calculateKeywordScore("calculate 2+2", skill)
	if score <= 0 {
		t.Error("Expected positive score for matching keywords")
	}
}

func TestDefaultManager_RecordExecution(t *testing.T) {
	m := DefaultManager()
	testSkill := NewSkill("test", "desc")
	testSkill.Instructions = "do something"
	m.RegisterSkill(testSkill)

	err := m.RecordExecution("test", true, time.Second, 100, 0.9)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	stats, err := m.GetSkillStatistics("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", stats.SuccessCount)
	}
}

func TestDefaultManager_RecordExecution_Failure(t *testing.T) {
	m := DefaultManager()
	testSkill := NewSkill("test", "desc")
	testSkill.Instructions = "do something"
	m.RegisterSkill(testSkill)

	m.RecordExecution("test", false, time.Second, 50, 0.3)

	stats, _ := m.GetSkillStatistics("test")
	if stats.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.FailureCount)
	}
}

func TestDefaultManager_RecordExecution_NonExistent(t *testing.T) {
	m := DefaultManager()
	err := m.RecordExecution("nonexistent", true, time.Second, 100, 0.9)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestDefaultManager_GetSkillStatistics(t *testing.T) {
	m := DefaultManager()
	testSkill := NewSkill("test", "desc")
	testSkill.Instructions = "do something"
	m.RegisterSkill(testSkill)

	stats, err := m.GetSkillStatistics("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if stats == nil {
		t.Error("Expected non-nil stats")
	}
}

func TestDefaultManager_GetSkillRanking(t *testing.T) {
	m := DefaultManager()
	skill1 := NewSkill("skill1", "desc")
	skill1.Instructions = "do something"
	m.RegisterSkill(skill1)
	skill2 := NewSkill("skill2", "desc")
	skill2.Instructions = "do something"
	m.RegisterSkill(skill2)

	stats1, _ := m.GetSkillStatistics("skill1")
	stats1.SuccessCount = 10
	stats1.CalculateOverallScore()

	rankings := m.GetSkillRanking()
	if len(rankings) != 2 {
		t.Errorf("Expected 2 rankings, got %d", len(rankings))
	}
}

func TestDefaultManager_EvolveSkills(t *testing.T) {
	m := DefaultManager()
	oldSkill := NewSkill("old-skill", "unused skill")
	oldSkill.Instructions = "do something old"
	m.RegisterSkill(oldSkill)

	err := m.EvolveSkills()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestDefaultManager_ArchiveSkill(t *testing.T) {
	m := DefaultManager()
	testSkill := NewSkill("test", "desc")
	testSkill.Instructions = "do something"
	m.RegisterSkill(testSkill)

	err := m.ArchiveSkill("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	_, err = m.GetSkill("test")
	if err == nil {
		t.Error("Expected error (skill should be archived)")
	}
}

func TestDefaultManager_ArchiveSkill_NotFound(t *testing.T) {
	m := DefaultManager()
	err := m.ArchiveSkill("nonexistent")
	if err == nil {
		t.Error("Expected error")
	}
}

func TestDefaultManager_RestoreSkill(t *testing.T) {
	m := DefaultManager()
	testSkill := NewSkill("test", "desc")
	testSkill.Instructions = "do something"
	m.RegisterSkill(testSkill)
	m.ArchiveSkill("test")

	err := m.RestoreSkill("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	_, err = m.GetSkill("test")
	if err != nil {
		t.Error("Expected skill to be restored")
	}
}

func TestDefaultManager_RestoreSkill_NotFound(t *testing.T) {
	m := DefaultManager()
	err := m.RestoreSkill("nonexistent")
	if err == nil {
		t.Error("Expected error")
	}
}

func TestDefaultManager_LoadSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	skillMd := `---
name: test-skill
description: A test skill
allowed-tools: calculator, bash
---
# Instructions
Step 1: Do something
Step 2: Done`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644)

	m := DefaultManager()
	skill, err := m.LoadSkill(skillDir)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if skill.Name != "test-skill" {
		t.Errorf("Expected 'test-skill', got %q", skill.Name)
	}
	if len(skill.AllowedTools) != 2 {
		t.Errorf("Expected 2 allowed tools, got %d", len(skill.AllowedTools))
	}
}

func TestDefaultManager_LoadSkill_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "bad-skill")
	os.MkdirAll(skillDir, 0755)

	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("no frontmatter"), 0644)

	m := DefaultManager()
	_, err := m.LoadSkill(skillDir)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestDefaultManager_LoadSkill_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	m := DefaultManager()
	_, err := m.LoadSkill(tmpDir)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestDefaultManager_loadOptionalDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	skillMd := `---
name: test
description: test
---
# Instructions`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644)

	scriptsDir := filepath.Join(skillDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)
	os.WriteFile(filepath.Join(scriptsDir, "helper.sh"), []byte("#!/bin/bash\necho hi"), 0644)

	m := DefaultManager()
	skill, err := m.LoadSkill(skillDir)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(skill.Scripts) != 1 {
		t.Errorf("Expected 1 script, got %d", len(skill.Scripts))
	}
}

func TestSelectionMode_Constants(t *testing.T) {
	if KeywordOnly != 0 {
		t.Errorf("Expected KeywordOnly to be 0, got %d", KeywordOnly)
	}
	if SemanticOnly != 1 {
		t.Errorf("Expected SemanticOnly to be 1, got %d", SemanticOnly)
	}
	if Hybrid != 2 {
		t.Errorf("Expected Hybrid to be 2, got %d", Hybrid)
	}
}