package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	// Agent configuration
	Agent AgentConfig `json:"agent" yaml:"agent"`
	
	// Reactor configuration
	Reactor ReactorConfig `json:"reactor" yaml:"reactor"`
	
	// Memory configuration
	Memory MemoryConfig `json:"memory" yaml:"memory"`
	
	// LLM configuration
	LLM LLMConfig `json:"llm" yaml:"llm"`
	
	// Evolution configuration
	Evolution EvolutionConfig `json:"evolution" yaml:"evolution"`
	
	// Observability configuration
	Observability ObservabilityConfig `json:"observability" yaml:"observability"`
	
	// Paths
	Paths PathConfig `json:"paths" yaml:"paths"`
}

// AgentConfig represents agent configuration
type AgentConfig struct {
	Name            string            `json:"name" yaml:"name"`
	Domain          string            `json:"domain" yaml:"domain"`
	Description     string            `json:"description" yaml:"description"`
	Model           string            `json:"model" yaml:"model"`
	PromptTemplate  string            `json:"prompt_template" yaml:"prompt_template"`
	MaxSteps        int               `json:"max_steps" yaml:"max_steps"`
	MaxRetries      int               `json:"max_retries" yaml:"max_retries"`
	Timeout         time.Duration     `json:"timeout" yaml:"timeout"`
	Skills          []string          `json:"skills" yaml:"skills"`
	Tools           []string          `json:"tools" yaml:"tools"`
	Metadata        map[string]any    `json:"metadata" yaml:"metadata"`
}

// ReactorConfig represents reactor configuration
type ReactorConfig struct {
	MaxSteps              int               `json:"max_steps" yaml:"max_steps"`
	MaxRetries            int               `json:"max_retries" yaml:"max_retries"`
	Timeout               time.Duration     `json:"timeout" yaml:"timeout"`
	EnablePlan            bool              `json:"enable_plan" yaml:"enable_plan"`
	EnableReflection      bool              `json:"enable_reflection" yaml:"enable_reflection"`
	EnableEvolution       bool              `json:"enable_evolution" yaml:"enable_evolution"`
	PauseOnToolAuth       bool              `json:"pause_on_tool_auth" yaml:"pause_on_tool_auth"`
}

// MemoryConfig represents memory configuration
type MemoryConfig struct {
	EnableGraphRAG      bool              `json:"enable_graph_rag" yaml:"enable_graph_rag"`
	EnableVectorSearch  bool              `json:"enable_vector_search" yaml:"enable_vector_search"`
	TopK                int               `json:"top_k" yaml:"top_k"`
	MinRelevance        float64           `json:"min_relevance" yaml:"min_relevance"`
	MaxContextSize      int               `json:"max_context_size" yaml:"max_context_size"`
}

// LLMConfig represents LLM configuration
type LLMConfig struct {
	Provider        string            `json:"provider" yaml:"provider"`
	Model           string            `json:"model" yaml:"model"`
	APIKey          string            `json:"api_key" yaml:"api_key"`
	BaseURL         string            `json:"base_url" yaml:"base_url"`
	MaxTokens       int               `json:"max_tokens" yaml:"max_tokens"`
	Temperature     float64           `json:"temperature" yaml:"temperature"`
	Timeout         time.Duration     `json:"timeout" yaml:"timeout"`
	RetryCount      int               `json:"retry_count" yaml:"retry_count"`
	Metadata        map[string]any    `json:"metadata" yaml:"metadata"`
}

// EvolutionConfig represents evolution configuration
type EvolutionConfig struct {
	EnableAutoEvolution         bool              `json:"enable_auto_evolution" yaml:"enable_auto_evolution"`
	EvolutionTrigger            EvolutionTrigger  `json:"evolution_trigger" yaml:"evolution_trigger"`
	SkillThreshold              int               `json:"skill_threshold" yaml:"skill_threshold"`
	ToolThreshold               int               `json:"tool_threshold" yaml:"tool_threshold"`
	MemoryImportanceThreshold   float64           `json:"memory_importance_threshold" yaml:"memory_importance_threshold"`
	MaxSkillsPerSession         int               `json:"max_skills_per_session" yaml:"max_skills_per_session"`
	MaxToolsPerSession          int               `json:"max_tools_per_session" yaml:"max_tools_per_session"`
	ReviewGeneratedCode         bool              `json:"review_generated_code" yaml:"review_generated_code"`
	AllowedToolTypes            []string          `json:"allowed_tool_types" yaml:"allowed_tool_types"`
}

// ObservabilityConfig represents observability configuration
type ObservabilityConfig struct {
	EnableTracing   bool              `json:"enable_tracing" yaml:"enable_tracing"`
	EnableMetrics   bool              `json:"enable_metrics" yaml:"enable_metrics"`
	EnableLogging   bool              `json:"enable_logging" yaml:"enable_logging"`
	LogLevel        string            `json:"log_level" yaml:"log_level"`
	OutputFormat    string            `json:"output_format" yaml:"output_format"`
	SampleRate      float64           `json:"sample_rate" yaml:"sample_rate"`
}

// PathConfig represents path configuration
type PathConfig struct {
	WorkDir         string            `json:"work_dir" yaml:"work_dir"`
	DocumentPath    string            `json:"document_path" yaml:"document_path"`
	SkillPath       string            `json:"skill_path" yaml:"skill_path"`
	ToolPath        string            `json:"tool_path" yaml:"tool_path"`
	ConfigPath      string            `json:"config_path" yaml:"config_path"`
	LogPath         string            `json:"log_path" yaml:"log_path"`
}

// ThinkerConfig represents thinker configuration
type ThinkerConfig struct {
	MaxTokens                  int       `json:"max_tokens" yaml:"max_tokens"`
	Temperature                float64   `json:"temperature" yaml:"temperature"`
	EnableReflectionInjection  bool      `json:"enable_reflection_injection" yaml:"enable_reflection_injection"`
	EnablePlanContext          bool      `json:"enable_plan_context" yaml:"enable_plan_context"`
	MaxHistorySteps            int       `json:"max_history_steps" yaml:"max_history_steps"`
	ConfidenceThreshold        float64   `json:"confidence_threshold" yaml:"confidence_threshold"`
}

// ActorConfig represents actor configuration
type ActorConfig struct {
	MaxRetries            int               `json:"max_retries" yaml:"max_retries"`
	Timeout               time.Duration     `json:"timeout" yaml:"timeout"`
	EnableSkillCache      bool              `json:"enable_skill_cache" yaml:"enable_skill_cache"`
	AllowedToolLevels     []SecurityLevel   `json:"allowed_tool_levels" yaml:"allowed_tool_levels"`
	MaxConcurrentActions  int               `json:"max_concurrent_actions" yaml:"max_concurrent_actions"`
	EnableDryRun          bool              `json:"enable_dry_run" yaml:"enable_dry_run"`
}

// ObserverConfig represents observer configuration
type ObserverConfig struct {
	EnableInsightExtraction    bool      `json:"enable_insight_extraction" yaml:"enable_insight_extraction"`
	EnableRelevanceAssessment  bool      `json:"enable_relevance_assessment" yaml:"enable_relevance_assessment"`
	EnableMemoryUpdate         bool      `json:"enable_memory_update" yaml:"enable_memory_update"`
	MaxInsightsPerObservation  int       `json:"max_insights_per_observation" yaml:"max_insights_per_observation"`
	RelevanceThreshold         float64   `json:"relevance_threshold" yaml:"relevance_threshold"`
	PersistRawResult           bool      `json:"persist_raw_result" yaml:"persist_raw_result"`
	MaxResultSize              int       `json:"max_result_size" yaml:"max_result_size"`
}

// DelegationConfig represents delegation configuration for sub-agents
type DelegationConfig struct {
	MaxDepth       int           `json:"max_depth" yaml:"max_depth"`
	MaxDuration    time.Duration `json:"max_duration" yaml:"max_duration"`
	AllowedAgents  []string      `json:"allowed_agents" yaml:"allowed_agents"`
	InheritTools   bool          `json:"inherit_tools" yaml:"inherit_tools"`
	InheritMemory  bool          `json:"inherit_memory" yaml:"inherit_memory"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:           DefaultAgentName,
			Domain:         DefaultDomain,
			Model:          DefaultModel,
			MaxSteps:       DefaultMaxSteps,
			MaxRetries:     DefaultMaxRetries,
			Timeout:        DefaultTimeout,
			Skills:         []string{},
			Tools:          []string{},
			Metadata:       make(map[string]any),
		},
		Reactor: ReactorConfig{
			MaxSteps:         DefaultMaxSteps,
			MaxRetries:       DefaultMaxRetries,
			Timeout:          DefaultTimeout,
			EnablePlan:       true,
			EnableReflection: true,
			EnableEvolution:  true,
			PauseOnToolAuth:  true,
		},
		Memory: MemoryConfig{
			EnableGraphRAG:     true,
			EnableVectorSearch: true,
			TopK:               DefaultTopK,
			MinRelevance:       0.5,
			MaxContextSize:     MaxContextSize,
		},
		LLM: LLMConfig{
			Provider:    "openai",
			Model:       DefaultModel,
			MaxTokens:   DefaultMaxTokens,
			Temperature: DefaultTemperature,
			Timeout:     60 * time.Second,
			RetryCount:  3,
			Metadata:    make(map[string]any),
		},
		Evolution: EvolutionConfig{
			EnableAutoEvolution:       true,
			EvolutionTrigger:          EvolutionTriggerOnSessionEnd,
			SkillThreshold:            DefaultSkillThreshold,
			ToolThreshold:             DefaultToolThreshold,
			MemoryImportanceThreshold: DefaultMemoryImportanceThreshold,
			MaxSkillsPerSession:       DefaultMaxSkillsPerSession,
			MaxToolsPerSession:        DefaultMaxToolsPerSession,
			ReviewGeneratedCode:       true,
			AllowedToolTypes:          []string{"python", "cli", "bash"},
		},
		Observability: ObservabilityConfig{
			EnableTracing: true,
			EnableMetrics: true,
			EnableLogging: true,
			LogLevel:      "info",
			OutputFormat:  "json",
			SampleRate:    1.0,
		},
		Paths: PathConfig{
			WorkDir:      ".",
			DocumentPath: "./docs",
			SkillPath:    "./skills",
			ToolPath:     "./tools",
			ConfigPath:   "./config",
			LogPath:      "./logs",
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	ext := filepath.Ext(path)
	
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func (c *Config) Save(path string) error {
	var data []byte
	var err error

	ext := filepath.Ext(path)
	
	switch ext {
	case ".json":
		data, err = json.MarshalIndent(c, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(c)
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if c.Agent.Model == "" {
		return fmt.Errorf("agent model is required")
	}
	if c.Reactor.MaxSteps <= 0 {
		return fmt.Errorf("reactor max_steps must be positive")
	}
	if c.Reactor.MaxRetries < 0 {
		return fmt.Errorf("reactor max_retries cannot be negative")
	}
	return nil
}
