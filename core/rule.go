package core

// RuleScope defines the applicability scope of a behavior rule.
type RuleScope string

const (
	// ScopeGlobal applies to all agents across all sessions.
	ScopeGlobal RuleScope = "global"

	// ScopeLocal applies only to the agent it was registered on.
	// Survives across sessions for that agent.
	ScopeLocal RuleScope = "local"

	// ScopeConversation applies only to the current session/conversation.
	// Cleared when the session ends or the agent switches identity.
	ScopeConversation RuleScope = "conversation"
)

// Rule defines a single behavior constraint or guideline for an AI agent.
// Rules are injected into the System Prompt before each LLM call,
// allowing runtime customization of agent behavior without code changes.
//
// Example:
//
//	rule := Rule{
//	    ID:          "no-delete-prod",
//	    Name:        "Production Data Protection",
//	    Description: "Never delete production data",
//	    Scope:       core.ScopeGlobal,
//	    Priority:    100,
//	    Content:     "绝对禁止删除生产环境的数据文件。如需修改，必须先备份。",
//	    Enabled:     true,
//	}
type Rule struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	Scope       RuleScope `json:"scope" yaml:"scope"`
	Priority    int       `json:"priority" yaml:"priority"`
	Enabled     bool      `json:"enabled" yaml:"enabled"`
	Content     string    `json:"content" yaml:"content"`
}

// RuleRegistry manages behavior rules for an agent.
// Rules are rendered into the System Prompt's <behavioral_rules> section
// before each LLM call, allowing dynamic behavior control.
//
// RuleRegistry manages behavioral rules that define WHAT an agent SHOULD do
// or MUST NOT do. Rules are STATIC constraints (e.g., "always ask before executing
// destructive commands") that apply regardless of the current Intent or context.
// This is distinct from IntentRegistry which dynamically classifies WHAT the user
// WANTS to do — rules define "should/must", intent identifies "wants".
type RuleRegistry interface {
	Register(rule Rule) error
	Unregister(id string)
	Get(id string) (*Rule, bool)
	All() []Rule
	GetByScope(scope RuleScope) []Rule
	FormatPromptSection() string
}
