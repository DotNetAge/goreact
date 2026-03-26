package skill

import (
	"testing"
	"time"
)

func TestNewSkill(t *testing.T) {
	skill := NewSkill("test-skill", "A test skill")

	if skill.Name != "test-skill" {
		t.Errorf("Expected 'test-skill', got %q", skill.Name)
	}
	if skill.Description != "A test skill" {
		t.Errorf("Expected 'A test skill', got %q", skill.Description)
	}
	if skill.Statistics == nil {
		t.Error("Expected Statistics to be initialized")
	}
	if skill.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}
	if skill.Scripts == nil {
		t.Error("Expected Scripts to be initialized")
	}
	if skill.References == nil {
		t.Error("Expected References to be initialized")
	}
	if skill.Assets == nil {
		t.Error("Expected Assets to be initialized")
	}
}

func TestSkill_GetMetadata(t *testing.T) {
	skill := &Skill{
		Name:        "test",
		Description: "desc",
	}

	meta := skill.GetMetadata()
	if meta.Name != "test" {
		t.Errorf("Expected 'test', got %q", meta.Name)
	}
	if meta.Description != "desc" {
		t.Errorf("Expected 'desc', got %q", meta.Description)
	}
}

func TestSkillStatistics_UpdateSuccessRate(t *testing.T) {
	stats := &SkillStatistics{
		SuccessCount: 8,
		FailureCount: 2,
	}

	stats.UpdateSuccessRate()
	if stats.SuccessRate != 0.8 {
		t.Errorf("Expected 0.8, got %f", stats.SuccessRate)
	}

	stats2 := &SkillStatistics{}
	stats2.UpdateSuccessRate()
	if stats2.SuccessRate != 0 {
		t.Errorf("Expected 0, got %f", stats2.SuccessRate)
	}
}

func TestSkillStatistics_UpdateAverageExecutionTime(t *testing.T) {
	stats := &SkillStatistics{
		SuccessCount:       2,
		FailureCount:       2,
		TotalExecutionTime: 4000 * time.Millisecond,
	}

	stats.UpdateAverageExecutionTime()
	if stats.AverageExecutionTime != 1000*time.Millisecond {
		t.Errorf("Expected 1s, got %v", stats.AverageExecutionTime)
	}

	stats2 := &SkillStatistics{}
	stats2.UpdateAverageExecutionTime()
	if stats2.AverageExecutionTime != 0 {
		t.Errorf("Expected 0, got %v", stats2.AverageExecutionTime)
	}
}

func TestSkillStatistics_UpdateQualityScore(t *testing.T) {
	stats := &SkillStatistics{
		SuccessCount:    2,
		FailureCount:    2,
		QualityScoreSum: 16.0,
	}

	stats.UpdateQualityScore()
	if stats.QualityScore != 4.0 {
		t.Errorf("Expected 4.0, got %f", stats.QualityScore)
	}
}

func TestSkillStatistics_CalculateOverallScore(t *testing.T) {
	stats := &SkillStatistics{
		SuccessRate:     0.8,
		EfficiencyScore: 0.9,
		QualityScore:    0.85,
		FrequencyScore:  0.5,
	}

	score := stats.CalculateOverallScore()
	expected := 0.8*0.4 + 0.9*0.25 + 0.85*0.25 + 0.5*0.1
	if score < expected-0.001 || score > expected+0.001 {
		t.Errorf("Expected ~%f, got %f", expected, score)
	}
}

func TestConstants(t *testing.T) {
	if SuccessRateWeight != 0.4 {
		t.Errorf("Expected 0.4, got %f", SuccessRateWeight)
	}
	if EfficiencyWeight != 0.25 {
		t.Errorf("Expected 0.25, got %f", EfficiencyWeight)
	}
	if QualityWeight != 0.25 {
		t.Errorf("Expected 0.25, got %f", QualityWeight)
	}
	if FrequencyWeight != 0.1 {
		t.Errorf("Expected 0.1, got %f", FrequencyWeight)
	}
}
