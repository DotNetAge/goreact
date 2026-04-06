package memory

import (
	"context"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// Evolver handles evolution of sessions into skills, tools, and memories
type Evolver struct {
	memory  *Memory
	config  *EvolutionConfig
	llm     core.Client
}

// EvolutionConfig contains evolution configuration
type EvolutionConfig struct {
	EnableAutoEvolution       bool                        `json:"enable_auto_evolution" yaml:"enable_auto_evolution"`
	EvolutionTrigger          goreactcommon.EvolutionTrigger `json:"evolution_trigger" yaml:"evolution_trigger"`
	SkillThreshold            int                         `json:"skill_threshold" yaml:"skill_threshold"`
	ToolThreshold             int                         `json:"tool_threshold" yaml:"tool_threshold"`
	MemoryImportanceThreshold float64                     `json:"memory_importance_threshold" yaml:"memory_importance_threshold"`
	MaxSkillsPerSession       int                         `json:"max_skills_per_session" yaml:"max_skills_per_session"`
	MaxToolsPerSession        int                         `json:"max_tools_per_session" yaml:"max_tools_per_session"`
	ReviewGeneratedCode       bool                        `json:"review_generated_code" yaml:"review_generated_code"`
	AllowedToolTypes          []string                    `json:"allowed_tool_types" yaml:"allowed_tool_types"`
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
	}
}

// NewEvolver creates a new Evolver
func NewEvolver(memory *Memory, llm core.Client, config *EvolutionConfig) *Evolver {
	if config == nil {
		config = DefaultEvolutionConfig()
	}
	return &Evolver{
		memory: memory,
		llm:    llm,
		config: config,
	}
}

// Evolve performs evolution on a session
func (e *Evolver) Evolve(ctx context.Context, sessionName string) (*EvolutionResult, error) {
	// Analyze the session
	analysis, err := e.AnalyzeSession(ctx, sessionName)
	if err != nil {
		return nil, err
	}
	
	result := &EvolutionResult{
		SessionName: sessionName,
		EvolvedAt:   time.Now(),
	}
	
	// Extract short-term memory
	if analysis.EvolutionPotential.HasMemoryPotential {
		items, err := e.ExtractShortTermMemory(ctx, analysis)
		if err == nil {
			result.ShortTermMemory = items
		}
	}
	
	// Generate skill
	if analysis.EvolutionPotential.HasSkillPotential {
		skill, err := e.GenerateSkill(ctx, analysis)
		if err == nil {
			result.GeneratedSkill = skill
		}
	}
	
	// Generate tool
	if analysis.EvolutionPotential.HasToolPotential {
		tool, err := e.GenerateTool(ctx, analysis)
		if err == nil {
			result.GeneratedTool = tool
		}
	}
	
	result.Success = true
	return result, nil
}

// AnalyzeSession analyzes a session for evolution potential
func (e *Evolver) AnalyzeSession(ctx context.Context, sessionName string) (*SessionAnalysis, error) {
	analysis := &SessionAnalysis{
		SessionName: sessionName,
		Patterns:    []PatternMatch{},
		Repetitions: []RepetitionInfo{},
		KeyInfo:     []*KeyInfo{},
	}
	
	// Would analyze session content using LLM
	// This is a simplified implementation
	
	return analysis, nil
}

// GenerateSkill generates a skill from session analysis
func (e *Evolver) GenerateSkill(ctx context.Context, analysis *SessionAnalysis) (any, error) {
	// Would use LLM to generate skill definition
	// This is a simplified implementation
	return nil, nil
}

// GenerateTool generates a tool from session analysis
func (e *Evolver) GenerateTool(ctx context.Context, analysis *SessionAnalysis) (any, error) {
	// Would use LLM to generate tool code
	// This is a simplified implementation
	return nil, nil
}

// ExtractShortTermMemory extracts short-term memory from analysis
func (e *Evolver) ExtractShortTermMemory(ctx context.Context, analysis *SessionAnalysis) ([]*goreactcore.MemoryItemNode, error) {
	// Would extract important information
	// This is a simplified implementation
	return nil, nil
}

// MarkEvolved marks a session as evolved
func (e *Evolver) MarkEvolved(ctx context.Context, sessionName string) error {
	// Would update session status
	return nil
}

// EvolutionResult represents the result of evolution
type EvolutionResult struct {
	SessionName     string                       `json:"session_name"`
	ShortTermMemory []*goreactcore.MemoryItemNode `json:"short_term_memory"`
	GeneratedSkill  any                          `json:"generated_skill"`
	GeneratedTool   any                          `json:"generated_tool"`
	EvolvedAt       time.Time                    `json:"evolved_at"`
	Success         bool                         `json:"success"`
	Message         string                       `json:"message"`
}

// SessionAnalysis represents the analysis of a session
type SessionAnalysis struct {
	SessionName       string                   `json:"session_name"`
	Messages          []*goreactcore.MessageNode `json:"messages"`
	Patterns          []PatternMatch           `json:"patterns"`
	Repetitions       []RepetitionInfo         `json:"repetitions"`
	KeyInfo           []*KeyInfo               `json:"key_info"`
	EvolutionPotential EvolutionPotential       `json:"evolution_potential"`
}

// PatternMatch represents a matched pattern
type PatternMatch struct {
	Pattern      string              `json:"pattern"`
	Occurrences  int                 `json:"occurrences"`
	Contexts     []string            `json:"contexts"`
	SuggestedType SuggestionType      `json:"suggested_type"`
}

// RepetitionInfo represents repeated operations
type RepetitionInfo struct {
	Operation       string  `json:"operation"`
	Count           int     `json:"count"`
	Similarity      float64 `json:"similarity"`
	CanBeAutomated  bool    `json:"can_be_automated"`
}

// KeyInfo represents important information
type KeyInfo struct {
	Content        string  `json:"content"`
	Importance     float64 `json:"importance"`
	Category       string  `json:"category"`
	ShouldMemorize bool    `json:"should_memorize"`
}

// EvolutionPotential represents the evolution potential
type EvolutionPotential struct {
	HasSkillPotential  bool    `json:"has_skill_potential"`
	HasToolPotential   bool    `json:"has_tool_potential"`
	HasMemoryPotential bool    `json:"has_memory_potential"`
	Confidence         float64 `json:"confidence"`
}

// SuggestionType represents the type of suggestion
type SuggestionType string

const (
	SuggestionTypeSkill  SuggestionType = "skill"
	SuggestionTypeTool   SuggestionType = "tool"
	SuggestionTypeMemory SuggestionType = "memory"
	SuggestionTypeNone   SuggestionType = "none"
)

// EvolutionService provides evolution service methods
type EvolutionService interface {
	EvolveSession(ctx context.Context, sessionName string) (*EvolutionResult, error)
	EvolveBatch(ctx context.Context, sessionNames []string) ([]*EvolutionResult, error)
	GetEvolutionHistory(ctx context.Context, sessionName string) (*EvolutionRecord, error)
	ListPendingEvolution(ctx context.Context) ([]string, error)
	ReviewGeneratedSkill(ctx context.Context, skillName string, approved bool) error
	ReviewGeneratedTool(ctx context.Context, toolName string, approved bool) error
}

// EvolutionRecord represents an evolution record
type EvolutionRecord struct {
	SessionName     string                    `json:"session_name"`
	EvolvedAt       time.Time                 `json:"evolved_at"`
	GeneratedSkills []string                  `json:"generated_skills"`
	GeneratedTools  []string                  `json:"generated_tools"`
	ExtractedMemory []string                  `json:"extracted_memory"`
	Status          goreactcommon.EvolutionStatus `json:"status"`
	Error           string                    `json:"error"`
}

// Implement EvolutionService interface

// EvolveSession evolves a single session
func (e *Evolver) EvolveSession(ctx context.Context, sessionName string) (*EvolutionResult, error) {
	return e.Evolve(ctx, sessionName)
}

// EvolveBatch evolves multiple sessions
func (e *Evolver) EvolveBatch(ctx context.Context, sessionNames []string) ([]*EvolutionResult, error) {
	results := make([]*EvolutionResult, 0, len(sessionNames))
	for _, name := range sessionNames {
		result, err := e.Evolve(ctx, name)
		if err != nil {
			results = append(results, &EvolutionResult{
				SessionName: name,
				Success:     false,
				Message:     err.Error(),
			})
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

// GetEvolutionHistory gets evolution history for a session
func (e *Evolver) GetEvolutionHistory(ctx context.Context, sessionName string) (*EvolutionRecord, error) {
	// Would query GraphRAG for evolution records
	return nil, nil
}

// ListPendingEvolution lists sessions pending evolution
func (e *Evolver) ListPendingEvolution(ctx context.Context) ([]string, error) {
	// Would query GraphRAG for sessions that haven't been evolved
	return nil, nil
}

// ReviewGeneratedSkill reviews and approves/rejects a generated skill
func (e *Evolver) ReviewGeneratedSkill(ctx context.Context, skillName string, approved bool) error {
	if e.memory == nil {
		return nil
	}
	if approved {
		return e.memory.Skills().ApproveGenerated(ctx, skillName)
	}
	return e.memory.Skills().Delete(ctx, skillName)
}

// ReviewGeneratedTool reviews and approves/rejects a generated tool
func (e *Evolver) ReviewGeneratedTool(ctx context.Context, toolName string, approved bool) error {
	if e.memory == nil {
		return nil
	}
	if approved {
		return e.memory.Tools().ApproveGenerated(ctx, toolName)
	}
	return e.memory.Tools().Delete(ctx, toolName)
}

// AnalyzeSessionWithLLM analyzes a session using LLM
func (e *Evolver) AnalyzeSessionWithLLM(ctx context.Context, sessionName string) (*SessionAnalysis, error) {
	if e.llm == nil {
		return e.AnalyzeSession(ctx, sessionName)
	}
	
	// Get session history from GraphRAG
	session, err := e.memory.Sessions().Get(ctx, sessionName)
	if err != nil {
		return nil, err
	}
	
	// Build analysis prompt
	prompt := buildAnalysisPrompt(session)
	
	// Call LLM
	resp, err := e.llm.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return nil, err
	}
	
	// Parse response into SessionAnalysis
	return parseAnalysisResponse(resp.Content, sessionName)
}

func buildAnalysisPrompt(session *goreactcore.SessionNode) string {
	return `Analyze this conversation session for evolution potential.
Identify:
1. Repeated patterns that could become skills
2. Tool usage patterns that could become tools
3. Important information that should be remembered

Session: ` + session.Name
}

func parseAnalysisResponse(response, sessionName string) (*SessionAnalysis, error) {
	return &SessionAnalysis{
		SessionName: sessionName,
		Patterns:    []PatternMatch{},
		Repetitions: []RepetitionInfo{},
		KeyInfo:     []*KeyInfo{},
	}, nil
}
