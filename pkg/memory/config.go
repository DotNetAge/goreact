package memory

import (
	"time"

	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
)

// Config contains all Memory configuration
type Config struct {
	Session    *SessionConfig    `json:"session" yaml:"session"`
	Evolution  *EvolutionConfig  `json:"evolution" yaml:"evolution"`
	Reflection *ReflectionConfig `json:"reflection" yaml:"reflection"`
	Plan       *PlanConfig       `json:"plan" yaml:"plan"`
	ShortTerm  *ShortTermConfig  `json:"short_term" yaml:"short_term"`
	LongTerm   *LongTermConfig   `json:"long_term" yaml:"long_term"`
	Security   *SecurityConfig   `json:"security" yaml:"security"`
}

// SessionConfig contains session-related configuration
type SessionConfig struct {
	MaxHistoryTurns int           `json:"max_history_turns" yaml:"max_history_turns"`
	ContextWindow   int           `json:"context_window" yaml:"context_window"`
	EnableAutoSave  bool          `json:"enable_auto_save" yaml:"enable_auto_save"`
	SaveInterval    time.Duration `json:"save_interval" yaml:"save_interval"`
	DefaultTurns    int           `json:"default_turns" yaml:"default_turns"`
	MinTurns        int           `json:"min_turns" yaml:"min_turns"`
	MaxTurns        int           `json:"max_turns" yaml:"max_turns"`
	TimeWindow      time.Duration `json:"time_window" yaml:"time_window"`
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		MaxHistoryTurns: 10,
		ContextWindow:   4000,
		EnableAutoSave:  true,
		SaveInterval:    30 * time.Second,
		DefaultTurns:    5,
		MinTurns:        3,
		MaxTurns:        10,
		TimeWindow:      30 * time.Minute,
	}
}

// EvolutionConfig contains evolution configuration
type EvolutionConfig struct {
	EnableAutoEvolution       bool                           `json:"enable_auto_evolution" yaml:"enable_auto_evolution"`
	EvolutionTrigger          goreactcommon.EvolutionTrigger `json:"evolution_trigger" yaml:"evolution_trigger"`
	SkillThreshold            int                            `json:"skill_threshold" yaml:"skill_threshold"`
	ToolThreshold             int                            `json:"tool_threshold" yaml:"tool_threshold"`
	MemoryImportanceThreshold float64                        `json:"memory_importance_threshold" yaml:"memory_importance_threshold"`
	MaxSkillsPerSession       int                            `json:"max_skills_per_session" yaml:"max_skills_per_session"`
	MaxToolsPerSession        int                            `json:"max_tools_per_session" yaml:"max_tools_per_session"`
	ReviewGeneratedCode       bool                           `json:"review_generated_code" yaml:"review_generated_code"`
	AllowedToolTypes          []string                       `json:"allowed_tool_types" yaml:"allowed_tool_types"`
	SkillOutputPath           string                         `json:"skill_output_path" yaml:"skill_output_path"`
	ToolOutputPath            string                         `json:"tool_output_path" yaml:"tool_output_path"`
}

// DefaultEvolutionConfig returns default evolution config
func DefaultEvolutionConfig() *EvolutionConfig {
	return &EvolutionConfig{
		EnableAutoEvolution:       true,
		EvolutionTrigger:          goreactcommon.EvolutionTriggerOnSessionEnd,
		SkillThreshold:            goreactcommon.DefaultSkillThreshold,
		ToolThreshold:             goreactcommon.DefaultToolThreshold,
		MemoryImportanceThreshold: goreactcommon.DefaultMemoryImportanceThreshold,
		MaxSkillsPerSession:       goreactcommon.DefaultMaxSkillsPerSession,
		MaxToolsPerSession:        goreactcommon.DefaultMaxToolsPerSession,
		ReviewGeneratedCode:       true,
		AllowedToolTypes:          []string{"python", "cli", "bash"},
		SkillOutputPath:           "./skills",
		ToolOutputPath:            "./tools",
	}
}

// ReflectionConfig contains reflection-related configuration
type ReflectionConfig struct {
	Enabled              bool    `json:"enabled" yaml:"enabled"`
	MinScoreThreshold    float64 `json:"min_score_threshold" yaml:"min_score_threshold"`
	MaxPerDay            int     `json:"max_per_day" yaml:"max_per_day"`
	RetentionDays        int     `json:"retention_days" yaml:"retention_days"`
	EnableAutoReflection bool    `json:"enable_auto_reflection" yaml:"enable_auto_reflection"`
}

// DefaultReflectionConfig returns default reflection configuration
func DefaultReflectionConfig() *ReflectionConfig {
	return &ReflectionConfig{
		Enabled:              true,
		MinScoreThreshold:    0.6,
		MaxPerDay:            100,
		RetentionDays:        30,
		EnableAutoReflection: true,
	}
}

// PlanConfig contains plan-related configuration
type PlanConfig struct {
	EnableReuse         bool    `json:"enable_reuse" yaml:"enable_reuse"`
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"`
	MaxSteps            int     `json:"max_steps" yaml:"max_steps"`
	RetentionDays       int     `json:"retention_days" yaml:"retention_days"`
}

// DefaultPlanConfig returns default plan configuration
func DefaultPlanConfig() *PlanConfig {
	return &PlanConfig{
		EnableReuse:         true,
		SimilarityThreshold: 0.7,
		MaxSteps:            20,
		RetentionDays:       90,
	}
}

// ShortTermConfig contains short-term memory configuration
type ShortTermConfig struct {
	Enabled              bool          `json:"enabled" yaml:"enabled"`
	MaxItems             int           `json:"max_items" yaml:"max_items"`
	ImportanceThreshold  float64       `json:"importance_threshold" yaml:"importance_threshold"`
	EnableAutoExtraction bool          `json:"enable_auto_extraction" yaml:"enable_auto_extraction"`
	MaxItemsPerSession   int           `json:"max_items_per_session" yaml:"max_items_per_session"`
	DefaultExpiration    time.Duration `json:"default_expiration" yaml:"default_expiration"`
	EnableSemanticSearch bool          `json:"enable_semantic_search" yaml:"enable_semantic_search"`
}

// DefaultShortTermConfig returns default short-term memory configuration
func DefaultShortTermConfig() *ShortTermConfig {
	return &ShortTermConfig{
		Enabled:              true,
		MaxItems:             50,
		ImportanceThreshold:  0.7,
		EnableAutoExtraction: true,
		MaxItemsPerSession:   100,
		DefaultExpiration:    24 * time.Hour,
		EnableSemanticSearch: true,
	}
}

// LongTermConfig contains long-term memory configuration
type LongTermConfig struct {
	EnableWatcher       bool          `json:"enable_watcher" yaml:"enable_watcher"`
	IndexPath           string        `json:"index_path" yaml:"index_path"`
	MaxIndexSize        int64         `json:"max_index_size" yaml:"max_index_size"`
	IndexInterval       time.Duration `json:"index_interval" yaml:"index_interval"`
	SimilarityThreshold float64       `json:"similarity_threshold" yaml:"similarity_threshold"`
	TopK                int           `json:"top_k" yaml:"top_k"`
	IncludeAdjacent     bool          `json:"include_adjacent" yaml:"include_adjacent"`
	AdjacentWindow      int           `json:"adjacent_window" yaml:"adjacent_window"`
}

// DefaultLongTermConfig returns default long-term memory configuration
func DefaultLongTermConfig() *LongTermConfig {
	return &LongTermConfig{
		EnableWatcher:       true,
		IndexPath:           "./documents",
		MaxIndexSize:        1024 * 1024 * 1024, // 1GB
		IndexInterval:       5 * time.Minute,
		SimilarityThreshold: 0.7,
		TopK:                5,
		IncludeAdjacent:     true,
		AdjacentWindow:      2,
	}
}

// TrajectoryConfig contains trajectory-related configuration
type TrajectoryConfig struct {
	EnableTrajectoryStore   bool          `json:"enable_trajectory_store" yaml:"enable_trajectory_store"`
	MaxTrajectorySteps      int           `json:"max_trajectory_steps" yaml:"max_trajectory_steps"`
	TrajectoryRetentionDays int           `json:"trajectory_retention_days" yaml:"trajectory_retention_days"`
	EnableSummary           bool          `json:"enable_summary" yaml:"enable_summary"`
	MaxStepDuration         time.Duration `json:"max_step_duration" yaml:"max_step_duration"`
}

// DefaultTrajectoryConfig returns default trajectory configuration
func DefaultTrajectoryConfig() *TrajectoryConfig {
	return &TrajectoryConfig{
		EnableTrajectoryStore:   true,
		MaxTrajectorySteps:      100,
		TrajectoryRetentionDays: 30,
		EnableSummary:           true,
		MaxStepDuration:         5 * time.Minute,
	}
}

// ConsolidationTrigger defines when consolidation happens
type ConsolidationTrigger string

const (
	ConsolidationTriggerOnSessionEnd ConsolidationTrigger = "session_end"
	ConsolidationTriggerOnThreshold  ConsolidationTrigger = "threshold"
	ConsolidationTriggerOnSchedule   ConsolidationTrigger = "schedule"
	ConsolidationTriggerManual       ConsolidationTrigger = "manual"
)

// CategoryClassifier defines content categories
type CategoryClassifier string

const (
	CategoryRule      CategoryClassifier = "rule"
	CategoryKnowledge CategoryClassifier = "knowledge"
	CategoryChat      CategoryClassifier = "chat"
	CategoryFact      CategoryClassifier = "fact"
	CategoryTask      CategoryClassifier = "task"
)

// ConsolidationConfig contains consolidation configuration
type ConsolidationConfig struct {
	EnableAutoConsolidation  bool                 `json:"enable_auto_consolidation" yaml:"enable_auto_consolidation"`
	Trigger                  ConsolidationTrigger `json:"trigger" yaml:"trigger"`
	ImportanceThreshold      float64              `json:"importance_threshold" yaml:"importance_threshold"`
	MaxItemsPerConsolidation int                  `json:"max_items_per_consolidation" yaml:"max_items_per_consolidation"`
	CategoryClassifier       CategoryClassifier   `json:"category_classifier" yaml:"category_classifier"`
	DocumentPathTemplate     string               `json:"document_path_template" yaml:"document_path_template"`
	ScheduleInterval         time.Duration        `json:"schedule_interval" yaml:"schedule_interval"`
}

// DefaultConsolidationConfig returns default consolidation configuration
func DefaultConsolidationConfig() *ConsolidationConfig {
	return &ConsolidationConfig{
		EnableAutoConsolidation:  true,
		Trigger:                  ConsolidationTriggerOnSessionEnd,
		ImportanceThreshold:      0.6,
		MaxItemsPerConsolidation: 100,
		CategoryClassifier:       CategoryKnowledge,
		DocumentPathTemplate:     "memory/{{.AgentName}}/{{.Category}}/{{.Date}}.md",
		ScheduleInterval:         1 * time.Hour,
	}
}

// FrozenSessionConfig contains frozen session configuration
type FrozenSessionConfig struct {
	DefaultExpiryDuration time.Duration `json:"default_expiry_duration" yaml:"default_expiry_duration"`
	MaxFrozenSessions     int           `json:"max_frozen_sessions" yaml:"max_frozen_sessions"`
	CleanupInterval       time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
	EnableAutoCleanup     bool          `json:"enable_auto_cleanup" yaml:"enable_auto_cleanup"`
}

// DefaultFrozenSessionConfig returns default frozen session configuration
func DefaultFrozenSessionConfig() *FrozenSessionConfig {
	return &FrozenSessionConfig{
		DefaultExpiryDuration: 24 * time.Hour,
		MaxFrozenSessions:     1000,
		CleanupInterval:       1 * time.Hour,
		EnableAutoCleanup:     true,
	}
}

// DefaultConfig returns default Memory configuration
func DefaultConfig() *Config {
	return &Config{
		Session:    DefaultSessionConfig(),
		Evolution:  DefaultEvolutionConfig(),
		Reflection: DefaultReflectionConfig(),
		Plan:       DefaultPlanConfig(),
		ShortTerm:  DefaultShortTermConfig(),
		LongTerm:   DefaultLongTermConfig(),
		Security:   DefaultSecurityConfig(),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Session != nil {
		if c.Session.MaxHistoryTurns < 0 {
			return &ConfigError{Field: "session.max_history_turns", Message: "must be non-negative"}
		}
		if c.Session.ContextWindow < 0 {
			return &ConfigError{Field: "session.context_window", Message: "must be non-negative"}
		}
	}

	if c.Evolution != nil {
		if c.Evolution.SkillThreshold < 1 {
			return &ConfigError{Field: "evolution.skill_threshold", Message: "must be at least 1"}
		}
		if c.Evolution.ToolThreshold < 1 {
			return &ConfigError{Field: "evolution.tool_threshold", Message: "must be at least 1"}
		}
		if c.Evolution.MemoryImportanceThreshold < 0 || c.Evolution.MemoryImportanceThreshold > 1 {
			return &ConfigError{Field: "evolution.memory_importance_threshold", Message: "must be between 0 and 1"}
		}
	}

	if c.Reflection != nil {
		if c.Reflection.MinScoreThreshold < 0 || c.Reflection.MinScoreThreshold > 1 {
			return &ConfigError{Field: "reflection.min_score_threshold", Message: "must be between 0 and 1"}
		}
		if c.Reflection.RetentionDays < 0 {
			return &ConfigError{Field: "reflection.retention_days", Message: "must be non-negative"}
		}
	}

	if c.Plan != nil {
		if c.Plan.SimilarityThreshold < 0 || c.Plan.SimilarityThreshold > 1 {
			return &ConfigError{Field: "plan.similarity_threshold", Message: "must be between 0 and 1"}
		}
		if c.Plan.MaxSteps < 1 {
			return &ConfigError{Field: "plan.max_steps", Message: "must be at least 1"}
		}
	}

	if c.ShortTerm != nil {
		if c.ShortTerm.MaxItems < 0 {
			return &ConfigError{Field: "short_term.max_items", Message: "must be non-negative"}
		}
		if c.ShortTerm.ImportanceThreshold < 0 || c.ShortTerm.ImportanceThreshold > 1 {
			return &ConfigError{Field: "short_term.importance_threshold", Message: "must be between 0 and 1"}
		}
	}

	return nil
}

// Merge merges another config into this one (non-nil values override)
func (c *Config) Merge(other *Config) *Config {
	if other == nil {
		return c
	}

	if other.Session != nil {
		c.Session = other.Session
	}
	if other.Evolution != nil {
		c.Evolution = other.Evolution
	}
	if other.Reflection != nil {
		c.Reflection = other.Reflection
	}
	if other.Plan != nil {
		c.Plan = other.Plan
	}
	if other.ShortTerm != nil {
		c.ShortTerm = other.ShortTerm
	}
	if other.LongTerm != nil {
		c.LongTerm = other.LongTerm
	}
	if other.Security != nil {
		c.Security = other.Security
	}

	return c
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string `json:"field" yaml:"field"`
	Message string `json:"message" yaml:"message"`
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " " + e.Message
}

// ConfigOption is a function that modifies configuration
type ConfigOption func(*Config)

// WithSessionConfig sets session configuration
func WithSessionConfig(cfg *SessionConfig) ConfigOption {
	return func(c *Config) {
		c.Session = cfg
	}
}

// WithEvolutionConfig sets evolution configuration
func WithEvolutionConfig(cfg *EvolutionConfig) ConfigOption {
	return func(c *Config) {
		c.Evolution = cfg
	}
}

// WithReflectionConfig sets reflection configuration
func WithReflectionConfig(cfg *ReflectionConfig) ConfigOption {
	return func(c *Config) {
		c.Reflection = cfg
	}
}

// WithPlanConfig sets plan configuration
func WithPlanConfig(cfg *PlanConfig) ConfigOption {
	return func(c *Config) {
		c.Plan = cfg
	}
}

// WithShortTermConfig sets short-term memory configuration
func WithShortTermConfig(cfg *ShortTermConfig) ConfigOption {
	return func(c *Config) {
		c.ShortTerm = cfg
	}
}

// WithLongTermConfig sets long-term memory configuration
func WithLongTermConfig(cfg *LongTermConfig) ConfigOption {
	return func(c *Config) {
		c.LongTerm = cfg
	}
}

// WithSecurityConfig sets security configuration
func WithSecurityConfig(cfg *SecurityConfig) ConfigOption {
	return func(c *Config) {
		c.Security = cfg
	}
}

// NewConfig creates a new configuration with options
func NewConfig(opts ...ConfigOption) *Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// ConfigLoader loads configuration from various sources
type ConfigLoader interface {
	Load(path string) (*Config, error)
	LoadFromYAML(data []byte) (*Config, error)
	LoadFromJSON(data []byte) (*Config, error)
}

// MemoryConfigBuilder builds Memory configuration step by step
type MemoryConfigBuilder struct {
	config *Config
}

// NewMemoryConfigBuilder creates a new config builder
func NewMemoryConfigBuilder() *MemoryConfigBuilder {
	return &MemoryConfigBuilder{
		config: DefaultConfig(),
	}
}

// WithSession sets session config
func (b *MemoryConfigBuilder) WithSession(maxHistoryTurns, contextWindow int, enableAutoSave bool) *MemoryConfigBuilder {
	b.config.Session = &SessionConfig{
		MaxHistoryTurns: maxHistoryTurns,
		ContextWindow:   contextWindow,
		EnableAutoSave:  enableAutoSave,
	}
	return b
}

// WithEvolution sets evolution config
func (b *MemoryConfigBuilder) WithEvolution(skillThreshold, toolThreshold int, memoryThreshold float64) *MemoryConfigBuilder {
	b.config.Evolution = &EvolutionConfig{
		EnableAutoEvolution:       true,
		EvolutionTrigger:          goreactcommon.EvolutionTriggerOnSessionEnd,
		SkillThreshold:            skillThreshold,
		ToolThreshold:             toolThreshold,
		MemoryImportanceThreshold: memoryThreshold,
	}
	return b
}

// WithReflection sets reflection config
func (b *MemoryConfigBuilder) WithReflection(minScore float64, retentionDays int) *MemoryConfigBuilder {
	b.config.Reflection = &ReflectionConfig{
		Enabled:           true,
		MinScoreThreshold: minScore,
		RetentionDays:     retentionDays,
	}
	return b
}

// WithPlan sets plan config
func (b *MemoryConfigBuilder) WithPlan(similarityThreshold float64, maxSteps int) *MemoryConfigBuilder {
	b.config.Plan = &PlanConfig{
		EnableReuse:         true,
		SimilarityThreshold: similarityThreshold,
		MaxSteps:            maxSteps,
	}
	return b
}

// Build builds the final configuration
func (b *MemoryConfigBuilder) Build() *Config {
	return b.config
}
