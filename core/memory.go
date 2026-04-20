package core

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Memory errors
var (
	ErrMemoryNotFound  = errors.New("memory not found")
	ErrMemoryStorage   = errors.New("memory storage failed")
	ErrMemoryRetrieval = errors.New("memory retrieval failed")
)

// MemoryType classifies the kind of knowledge stored in a MemoryRecord.
type MemoryType int

const (
	MemoryTypeSession    MemoryType = iota // temporary session context (conversation history)
	MemoryTypeUser                         // short-term: user preferences, conventions
	MemoryTypeLongTerm                     // long-term: knowledge base (project docs, domain knowledge)
	MemoryTypeRefactive                    // reflexive: semantic index of tools/skills/agents
	MemoryTypeExperience                   // experience: analysis results from successful task executions
)

// MemoryScope defines the visibility of a memory record.
type MemoryScope int

const (
	MemoryScopePrivate MemoryScope = iota // visible only to the current user
	MemoryScopeTeam                      // shared across the team
)

// MemoryRecord represents a single piece of stored knowledge.
type MemoryRecord struct {
	ID        string       `json:"id"`
	Type      MemoryType   `json:"type"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Scope     MemoryScope  `json:"scope"`
	Tags      []string     `json:"tags,omitempty"`
	Score     float64      `json:"score,omitempty"` // relevance score from Retrieve (0 = unset)
	Meta      any          `json:"meta,omitempty"`  // typed metadata (e.g., *ExperienceData for Type=Experience)
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// Memory is the core interface for knowledge retrieval and storage.
// Implementations range from simple in-memory to full RAG.
// The interface is designed so that external callers can provide their own
// implementation — from a simple database CRUD to a full vector search RAG.
type Memory interface {
	// Retrieve searches memory for records relevant to the query.
	// Returns records sorted by relevance (highest score first).
	Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]MemoryRecord, error)

	// Store persists a new memory record and returns its ID.
	Store(ctx context.Context, record MemoryRecord) (string, error)

	// Update modifies an existing memory record by ID.
	// If the record does not exist, returns ErrMemoryNotFound.
	Update(ctx context.Context, id string, record MemoryRecord) error

	// Delete removes a memory record by ID.
	// If the record does not exist, returns ErrMemoryNotFound.
	Delete(ctx context.Context, id string) error
}

// ReNewer is an optional interface that Memory implementations can implement
// to provide semantic context rebuild capability. When a ContextWindow needs
// compaction, the Reactor checks if Memory also implements ReNewer.
// If so, it calls ReNew instead of traditional LLM-based summarization.
//
// The Memory implementation is free to use any strategy: semantic search
// over session history, graph-based context assembly, or hybrid approaches.
// The Reactor simply replaces ConversationHistory with the returned messages.
type ReNewer interface {
	// ReNew rebuilds the context window for the given session.
	// It receives the current intent and all session messages, and returns
	// a refined set of messages that preserves the most relevant context.
	// This is called when the context window exceeds its token budget.
	ReNew(ctx context.Context, sessionID string, intent string, messages []Message) ([]Message, error)
}

// --- Experience types (exposed for Memory implementations) ---

// ExperienceData is the structured content of a MemoryTypeExperience record.
// When the Reactor completes a task successfully, it builds an ExperienceData
// and stores it in a MemoryRecord with Type=MemoryTypeExperience.
//
// The Content field holds the JSON serialization of this struct, and the
// Meta field holds the typed pointer. Memory implementations can use Meta
// directly (type assertion) or unmarshal Content — both are valid.
//
// This structure is designed for two purposes:
//  1. Retrieval: Title + Tags serve as the "problem" semantic index for recall.
//  2. Self-growing: Memory implementations can generate SKILL.md from this data,
//     enabling ReActor to become more capable over time.
type ExperienceData struct {
	// Problem describes what the user asked / what issue was being solved.
	// This becomes the semantic index for future recall.
	Problem string `json:"problem"`

	// Analysis is the LLM's reasoning from the Think phase(s).
	// This is the most expensive token output and the most valuable to reuse.
	Analysis string `json:"analysis"`

	// Tools is the ordered list of tools called during execution.
	Tools []string `json:"tools,omitempty"`

	// SubAgents is the list of subagents/tasks spawned during execution.
	// This captures orchestration patterns (task_create, subagent calls).
	SubAgents []ExperienceSubAgent `json:"sub_agents,omitempty"`

	// Steps is a compact summary of each T-A-O cycle.
	Steps []ExperienceStep `json:"steps,omitempty"`

	// Answer is the final answer given to the user.
	Answer string `json:"answer,omitempty"`

	// TokenCost records how many tokens this execution consumed.
	// Useful for the Memory implementation to prioritize high-value experiences.
	TokenCost int `json:"token_cost,omitempty"`
}

// ExperienceStep is a compact representation of one T-A-O cycle within an experience.
type ExperienceStep struct {
	Thought  string `json:"thought,omitempty"`
	Action   string `json:"action,omitempty"`    // "tool_name(params)"
	Result   string `json:"result,omitempty"`    // truncated observation result
	HasError bool   `json:"has_error,omitempty"` // whether this step had an error
}

// ExperienceSubAgent captures information about a subagent/task spawned during execution.
type ExperienceSubAgent struct {
	Name    string `json:"name,omitempty"`    // subagent/task name
	Tool    string `json:"tool,omitempty"`    // "task_create" or "subagent"
	Prompt  string `json:"prompt,omitempty"`  // truncated prompt sent to the subagent
	Success bool   `json:"success,omitempty"` // whether the subagent completed successfully
}

// --- Retrieve options ---

// RetrieveConfig holds configuration for a Retrieve call.
type RetrieveConfig struct {
	Types    []MemoryType // filter by types (empty = all types)
	Scope    MemoryScope  // filter by scope (0 = no filter)
	Limit    int          // max results to return (0 = default 5)
	MinScore float64      // minimum relevance score threshold (0 = no filter)
}

// RetrieveOption configures Retrieve behavior.
type RetrieveOption func(*RetrieveConfig)

// WithMemoryTypes filters Retrieve results to specific memory types.
func WithMemoryTypes(types ...MemoryType) RetrieveOption {
	return func(c *RetrieveConfig) {
		c.Types = types
	}
}

// WithMemoryScope filters Retrieve results by scope.
func WithMemoryScope(scope MemoryScope) RetrieveOption {
	return func(c *RetrieveConfig) {
		c.Scope = scope
	}
}

// WithMemoryLimit sets the maximum number of results to return.
func WithMemoryLimit(n int) RetrieveOption {
	return func(c *RetrieveConfig) {
		if n > 0 {
			c.Limit = n
		}
	}
}

// WithMinScore sets a minimum relevance score threshold.
func WithMinScore(score float64) RetrieveOption {
	return func(c *RetrieveConfig) {
		c.MinScore = score
	}
}

// DefaultRetrieveConfig returns the default retrieve configuration.
func DefaultRetrieveConfig() RetrieveConfig {
	return RetrieveConfig{
		Types:    nil,
		Scope:    0,
		Limit:    5,
		MinScore: 0,
	}
}

// --- Formatting helpers ---

// FormatMemoryRecords converts memory records into a text block for prompt injection.
// This is used in the Think phase to inject relevant memory into the LLM prompt.
func FormatMemoryRecords(records []MemoryRecord) string {
	if len(records) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, r := range records {
		typeName := memoryTypeLabel(r.Type)
		if r.Title != "" {
			sb.WriteString("## ")
			sb.WriteString(typeName)
			sb.WriteString(": ")
			sb.WriteString(r.Title)
			sb.WriteString("\n")
		}
		sb.WriteString(r.Content)
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

func memoryTypeLabel(t MemoryType) string {
	switch t {
	case MemoryTypeSession:
		return "Session Memory"
	case MemoryTypeUser:
		return "User Preference"
	case MemoryTypeLongTerm:
		return "Long-term Knowledge"
	case MemoryTypeRefactive:
		return "Reflexive Index"
	case MemoryTypeExperience:
		return "Experience"
	default:
		return "Unknown"
	}
}
