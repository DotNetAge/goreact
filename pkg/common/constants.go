package common

import "time"

// Default configuration values
const (
	// Reactor defaults
	DefaultMaxSteps     = 10
	DefaultMaxRetries   = 3
	DefaultTimeout      = 30 * time.Second
	DefaultPlanTimeout  = 5 * time.Minute

	// Thinker defaults
	DefaultMaxTokens              = 4096
	DefaultTemperature            = 0.7
	DefaultConfidenceThreshold    = 0.8
	DefaultMaxHistorySteps        = 10

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
	IntentConfidenceHigh     = 0.7
	IntentConfidenceMedium   = 0.5
	IntentClarifyThreshold   = 0.7
)

// Node types
const (
	NodeTypeAgent          = "Agent"
	NodeTypeModel          = "Model"
	NodeTypeSkill          = "Skill"
	NodeTypeTool           = "Tool"
	NodeTypeSession        = "Session"
	NodeTypeMessage        = "Message"
	NodeTypeMemoryItem     = "MemoryItem"
	NodeTypeReflection     = "Reflection"
	NodeTypePlan           = "Plan"
	NodeTypePlanStep       = "PlanStep"
	NodeTypeTrajectory     = "Trajectory"
	NodeTypeTrajectoryStep = "TrajectoryStep"
	NodeTypeFrozenSession  = "FrozenSession"
	NodeTypePendingQuestion = "PendingQuestion"
	NodeTypeGeneratedSkill = "GeneratedSkill"
	NodeTypeGeneratedTool  = "GeneratedTool"
	NodeTypeSkillExecutionPlan = "SkillExecutionPlan"
)

// Edge types (relationships)
const (
	EdgeTypeHasSkill        = "HAS_SKILL"
	EdgeTypeHasTool         = "HAS_TOOL"
	EdgeTypeUsesModel       = "USES_MODEL"
	EdgeTypeHasMessage      = "HAS_MESSAGE"
	EdgeTypeContains        = "CONTAINS"
	EdgeTypeDerivesFrom     = "DERIVES_FROM"
	EdgeTypeBasedOn         = "BASED_ON"
	EdgeTypeHasTrajectory   = "HAS_TRAJECTORY"
	EdgeTypeIncludes        = "INCLUDES"
	EdgeTypeFollowedBy      = "FOLLOWED_BY"
	EdgeTypeRelatedTo       = "RELATED_TO"
	EdgeTypeSucceeds        = "SUCCEEDS"
	EdgeTypeFailsAt         = "FAILS_AT"
	EdgeTypeHasReflection   = "HAS_REFLECTION"
	EdgeTypeHasPlan         = "HAS_PLAN"
	EdgeTypeHasStep         = "HAS_STEP"
	EdgeTypeSourceSession   = "SOURCE_SESSION"
)

// Prompt template keys
const (
	PromptKeySystem      = "system"
	PromptKeyPlan        = "plan"
	PromptKeyThink       = "think"
	PromptKeyReflect     = "reflect"
	PromptKeyIntent      = "intent"
	PromptKeyEvolution   = "evolution"
)

// Metadata keys
const (
	MetaKeySessionName   = "session_name"
	MetaKeyAgentName     = "agent_name"
	MetaKeyUserName      = "user_name"
	MetaKeyTimestamp     = "timestamp"
	MetaKeyDuration      = "duration"
	MetaKeyTokens        = "tokens"
	MetaKeyStatus        = "status"
	MetaKeyError         = "error"
	MetaKeySource        = "source"
	MetaKeyType          = "type"
)

// Context keys
const (
	ContextKeySession    = "session"
	ContextKeyAgent      = "agent"
	ContextKeyState      = "state"
	ContextKeyMemory     = "memory"
	ContextKeyResources  = "resources"
	ContextKeyTraceID    = "trace_id"
)

// File extensions
const (
	ExtMarkdown = ".md"
	ExtPython   = ".py"
	ExtBash     = ".sh"
	ExtGo       = ".go"
	ExtJSON     = ".json"
	ExtYAML     = ".yaml"
)

// Environment variables
const (
	EnvOpenAIKey      = "OPENAI_API_KEY"
	EnvAnthropicKey   = "ANTHROPIC_API_KEY"
	EnvGeminiKey      = "GOOGLE_API_KEY"
	EnvDefaultModel   = "GOREACT_DEFAULT_MODEL"
	EnvLogLevel       = "GOREACT_LOG_LEVEL"
	EnvConfigPath     = "GOREACT_CONFIG_PATH"
)

// Special values
const (
	DefaultModel      = "gpt-4"
	DefaultAgentName  = "assistant"
	DefaultDomain     = "general"
	MaxContextSize    = 128000
	MaxResponseSize   = 4096
)
