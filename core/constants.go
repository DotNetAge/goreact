package core

import "time"

// Default configuration values
const (
	// Reactor defaults
	DefaultMaxSteps    = 10
	DefaultMaxRetries  = 3
	DefaultTimeout     = 30 * time.Second
	DefaultPlanTimeout = 5 * time.Minute

	// Thinker defaults
	DefaultMaxTokens           = 4096
	DefaultTemperature         = 0.7
	DefaultConfidenceThreshold = 0.8
	DefaultMaxHistorySteps     = 10

	// Actor defaults
	DefaultMaxConcurrentActions = 5
	DefaultActorTimeout         = 30 * time.Second
	DefaultActorMaxRetries      = 3

	// Observer defaults
	DefaultMaxInsightsPerObservation = 5
	DefaultRelevanceThreshold        = 0.5
	DefaultMaxResultSize             = 1048576 // 1MB

	// Evolution defaults
	DefaultSkillThreshold            = 2
	DefaultToolThreshold             = 3
	DefaultMemoryImportanceThreshold = 0.7
	DefaultMaxSkillsPerSession       = 1
	DefaultMaxToolsPerSession        = 1

	// Memory defaults
	DefaultTopK = 10

	// Intent thresholds
	IntentConfidenceHigh   = 0.7
	IntentConfidenceMedium = 0.5
	IntentClarifyThreshold = 0.7
)

// Metadata keys
const (
	MetaKeyTimestamp = "timestamp"
	MetaKeyDuration  = "duration"
	MetaKeyTokens    = "tokens"
	MetaKeyError     = "error"
)

// Context keys
const (
	ContextKeyTraceID = "trace_id"
)

// File extensions
const (
	ExtGo = ".go"
)

// Special values
const (
	DefaultModel     = "gpt-4"
	DefaultAgentName = "assistant"
)
