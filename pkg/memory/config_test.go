package memory

import (
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestConfig_Defaults(t *testing.T) {
	config := DefaultConfig()

	if config.Session == nil {
		t.Error("Session config should not be nil")
	}
	if config.Evolution == nil {
		t.Error("Evolution config should not be nil")
	}
	if config.Reflection == nil {
		t.Error("Reflection config should not be nil")
	}
	if config.Plan == nil {
		t.Error("Plan config should not be nil")
	}
	if config.ShortTerm == nil {
		t.Error("ShortTerm config should not be nil")
	}
	if config.LongTerm == nil {
		t.Error("LongTerm config should not be nil")
	}
}

func TestSessionConfig_Defaults(t *testing.T) {
	config := DefaultSessionConfig()

	if config.MaxHistoryTurns <= 0 {
		t.Error("MaxHistoryTurns should be positive")
	}
	if config.ContextWindow <= 0 {
		t.Error("ContextWindow should be positive")
	}
	if config.SaveInterval <= 0 {
		t.Error("SaveInterval should be positive")
	}
}

func TestEvolutionConfig_Defaults(t *testing.T) {
	config := DefaultEvolutionConfig()

	if config.SkillThreshold < 1 {
		t.Error("SkillThreshold should be at least 1")
	}
	if config.ToolThreshold < 1 {
		t.Error("ToolThreshold should be at least 1")
	}
	if config.MemoryImportanceThreshold < 0 || config.MemoryImportanceThreshold > 1 {
		t.Errorf("MemoryImportanceThreshold = %f, should be between 0 and 1", config.MemoryImportanceThreshold)
	}
	if !config.EnableAutoEvolution {
		t.Error("EnableAutoEvolution should be true by default")
	}
}

func TestEvolutionConfig_Validation(t *testing.T) {
	config := &EvolutionConfig{
		EnableAutoEvolution:       true,
		EvolutionTrigger:          common.EvolutionTriggerOnSessionEnd,
		SkillThreshold:            2,
		ToolThreshold:             3,
		MemoryImportanceThreshold: 0.5,
		MaxSkillsPerSession:       10,
		MaxToolsPerSession:        10,
		ReviewGeneratedCode:       true,
	}

	if config.SkillThreshold != 2 {
		t.Errorf("SkillThreshold = %d, want 2", config.SkillThreshold)
	}
	if config.ToolThreshold != 3 {
		t.Errorf("ToolThreshold = %d, want 3", config.ToolThreshold)
	}
	if config.EvolutionTrigger != common.EvolutionTriggerOnSessionEnd {
		t.Errorf("EvolutionTrigger = %v, want OnSessionEnd", config.EvolutionTrigger)
	}
}

func TestReflectionConfig_Defaults(t *testing.T) {
	config := DefaultReflectionConfig()

	if !config.Enabled {
		t.Error("Enabled should be true by default")
	}
	if config.MinScoreThreshold < 0 || config.MinScoreThreshold > 1 {
		t.Errorf("MinScoreThreshold = %f, should be between 0 and 1", config.MinScoreThreshold)
	}
}

func TestPlanConfig_Defaults(t *testing.T) {
	config := DefaultPlanConfig()

	if !config.EnableReuse {
		t.Error("EnableReuse should be true by default")
	}
	if config.SimilarityThreshold < 0 || config.SimilarityThreshold > 1 {
		t.Errorf("SimilarityThreshold = %f, should be between 0 and 1", config.SimilarityThreshold)
	}
	if config.MaxSteps < 1 {
		t.Error("MaxSteps should be at least 1")
	}
}

func TestShortTermConfig_Defaults(t *testing.T) {
	config := DefaultShortTermConfig()

	if !config.Enabled {
		t.Error("Enabled should be true by default")
	}
	if config.MaxItems <= 0 {
		t.Error("MaxItems should be positive")
	}
	if config.MaxItemsPerSession <= 0 {
		t.Error("MaxItemsPerSession should be positive")
	}
	if config.ImportanceThreshold < 0 || config.ImportanceThreshold > 1 {
		t.Errorf("ImportanceThreshold = %f, should be between 0 and 1", config.ImportanceThreshold)
	}
}

func TestLongTermConfig_Defaults(t *testing.T) {
	config := DefaultLongTermConfig()

	if !config.EnableWatcher {
		t.Error("EnableWatcher should be true by default")
	}
	if config.TopK <= 0 {
		t.Error("TopK should be positive")
	}
	if config.SimilarityThreshold < 0 || config.SimilarityThreshold > 1 {
		t.Errorf("SimilarityThreshold = %f, should be between 0 and 1", config.SimilarityThreshold)
	}
}

func TestConsolidationConfig_Defaults(t *testing.T) {
	config := DefaultConsolidationConfig()

	if !config.EnableAutoConsolidation {
		t.Error("EnableAutoConsolidation should be true by default")
	}
	if config.Trigger != ConsolidationTriggerOnSessionEnd {
		t.Errorf("Trigger = %v, want OnSessionEnd", config.Trigger)
	}
	if config.MaxItemsPerConsolidation <= 0 {
		t.Error("MaxItemsPerConsolidation should be positive")
	}
	if config.ImportanceThreshold < 0 || config.ImportanceThreshold > 1 {
		t.Errorf("ImportanceThreshold = %f, should be between 0 and 1", config.ImportanceThreshold)
	}
}

func TestTrajectoryConfig_Defaults(t *testing.T) {
	config := DefaultTrajectoryConfig()

	if !config.EnableTrajectoryStore {
		t.Error("EnableTrajectoryStore should be true by default")
	}
	if config.MaxTrajectorySteps <= 0 {
		t.Error("MaxTrajectorySteps should be positive")
	}
	if config.TrajectoryRetentionDays <= 0 {
		t.Error("TrajectoryRetentionDays should be positive")
	}
}

func TestFrozenSessionConfig_Defaults(t *testing.T) {
	config := DefaultFrozenSessionConfig()

	if config.DefaultExpiryDuration <= 0 {
		t.Error("DefaultExpiryDuration should be positive")
	}
	if config.MaxFrozenSessions <= 0 {
		t.Error("MaxFrozenSessions should be positive")
	}
	if !config.EnableAutoCleanup {
		t.Error("EnableAutoCleanup should be true by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid session config",
			config: &Config{
				Session: &SessionConfig{MaxHistoryTurns: -1},
			},
			wantErr: true,
		},
		{
			name: "invalid evolution config",
			config: &Config{
				Evolution: &EvolutionConfig{SkillThreshold: 0},
			},
			wantErr: true,
		},
		{
			name: "invalid reflection config",
			config: &Config{
				Reflection: &ReflectionConfig{MinScoreThreshold: 2.0},
			},
			wantErr: true,
		},
		{
			name: "invalid plan config",
			config: &Config{
				Plan: &PlanConfig{SimilarityThreshold: -0.5},
			},
			wantErr: true,
		},
		{
			name: "invalid short term config",
			config: &Config{
				ShortTerm: &ShortTermConfig{ImportanceThreshold: 2.0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	base := DefaultConfig()
	other := &Config{
		Session: &SessionConfig{
			MaxHistoryTurns: 20,
			ContextWindow:   8000,
		},
	}

	result := base.Merge(other)

	if result.Session.MaxHistoryTurns != 20 {
		t.Errorf("MaxHistoryTurns = %d, want 20", result.Session.MaxHistoryTurns)
	}
	if result.Session.ContextWindow != 8000 {
		t.Errorf("ContextWindow = %d, want 8000", result.Session.ContextWindow)
	}
}

func TestConfigOptions(t *testing.T) {
	sessionCfg := &SessionConfig{
		MaxHistoryTurns: 15,
		ContextWindow:   6000,
	}
	evolutionCfg := &EvolutionConfig{
		SkillThreshold: 5,
		ToolThreshold:  8,
	}

	config := NewConfig(
		WithSessionConfig(sessionCfg),
		WithEvolutionConfig(evolutionCfg),
	)

	if config.Session.MaxHistoryTurns != 15 {
		t.Errorf("Session.MaxHistoryTurns = %d, want 15", config.Session.MaxHistoryTurns)
	}
	if config.Evolution.SkillThreshold != 5 {
		t.Errorf("Evolution.SkillThreshold = %d, want 5", config.Evolution.SkillThreshold)
	}
}

func TestMemoryConfigBuilder(t *testing.T) {
	config := NewMemoryConfigBuilder().
		WithSession(25, 10000, true).
		WithEvolution(10, 15, 0.8).
		WithReflection(0.7, 60).
		WithPlan(0.85, 30).
		Build()

	if config.Session.MaxHistoryTurns != 25 {
		t.Errorf("Session.MaxHistoryTurns = %d, want 25", config.Session.MaxHistoryTurns)
	}
	if config.Session.ContextWindow != 10000 {
		t.Errorf("Session.ContextWindow = %d, want 10000", config.Session.ContextWindow)
	}
	if config.Evolution.SkillThreshold != 10 {
		t.Errorf("Evolution.SkillThreshold = %d, want 10", config.Evolution.SkillThreshold)
	}
	if config.Evolution.ToolThreshold != 15 {
		t.Errorf("Evolution.ToolThreshold = %d, want 15", config.Evolution.ToolThreshold)
	}
	if config.Reflection.MinScoreThreshold != 0.7 {
		t.Errorf("Reflection.MinScoreThreshold = %f, want 0.7", config.Reflection.MinScoreThreshold)
	}
	if config.Plan.SimilarityThreshold != 0.85 {
		t.Errorf("Plan.SimilarityThreshold = %f, want 0.85", config.Plan.SimilarityThreshold)
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "config error: test_field test message"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestConsolidationTrigger_Constants(t *testing.T) {
	if ConsolidationTriggerOnSessionEnd != "session_end" {
		t.Errorf("ConsolidationTriggerOnSessionEnd = %q, want 'session_end'", ConsolidationTriggerOnSessionEnd)
	}
	if ConsolidationTriggerOnThreshold != "threshold" {
		t.Errorf("ConsolidationTriggerOnThreshold = %q, want 'threshold'", ConsolidationTriggerOnThreshold)
	}
	if ConsolidationTriggerOnSchedule != "schedule" {
		t.Errorf("ConsolidationTriggerOnSchedule = %q, want 'schedule'", ConsolidationTriggerOnSchedule)
	}
	if ConsolidationTriggerManual != "manual" {
		t.Errorf("ConsolidationTriggerManual = %q, want 'manual'", ConsolidationTriggerManual)
	}
}

func TestCategoryClassifier_Constants(t *testing.T) {
	categories := []struct {
		name     CategoryClassifier
		expected string
	}{
		{CategoryRule, "rule"},
		{CategoryKnowledge, "knowledge"},
		{CategoryChat, "chat"},
		{CategoryFact, "fact"},
		{CategoryTask, "task"},
	}

	for _, tc := range categories {
		if tc.name != CategoryClassifier(tc.expected) {
			t.Errorf("%s = %q, want %q", tc.expected, tc.name, tc.expected)
		}
	}
}

func TestConfig_Durations(t *testing.T) {
	config := DefaultConfig()
	consolidation := DefaultConsolidationConfig()

	// Verify durations are properly set
	if config.Session.SaveInterval <= 0 {
		t.Error("Session.SaveInterval should be positive")
	}
	if config.Session.TimeWindow <= 0 {
		t.Error("Session.TimeWindow should be positive")
	}
	if config.ShortTerm.DefaultExpiration != 24*time.Hour {
		t.Errorf("ShortTerm.DefaultExpiration = %v, want 24h", config.ShortTerm.DefaultExpiration)
	}
	if config.LongTerm.IndexInterval <= 0 {
		t.Error("LongTerm.IndexInterval should be positive")
	}
	if consolidation.ScheduleInterval <= 0 {
		t.Error("Consolidation.ScheduleInterval should be positive")
	}
}
