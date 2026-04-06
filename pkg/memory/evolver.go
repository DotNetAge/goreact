package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/skill"
	"github.com/DotNetAge/goreact/pkg/tool"
)

// Evolver handles evolution of sessions into skills, tools, and memories
type Evolver struct {
	memory  *Memory
	config  *EvolutionConfig
	llm     core.Client
	records map[string]*EvolutionRecord // Track evolution records by session name
}

// NewEvolver creates a new Evolver
func NewEvolver(memory *Memory, llm core.Client, config *EvolutionConfig) *Evolver {
	if config == nil {
		config = DefaultEvolutionConfig()
	}
	return &Evolver{
		memory:  memory,
		llm:     llm,
		config:  config,
		records: make(map[string]*EvolutionRecord),
	}
}

// Evolve performs evolution on a session
func (e *Evolver) Evolve(ctx context.Context, sessionName string) (*EvolutionResult, error) {
	// Analyze the session
	analysis, err := e.AnalyzeSessionWithLLM(ctx, sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze session: %w", err)
	}
	
	result := &EvolutionResult{
		SessionName: sessionName,
		EvolvedAt:   time.Now(),
	}
	
	// Extract short-term memory
	if analysis.EvolutionPotential.HasMemoryPotential {
		items, err := e.ExtractShortTermMemory(ctx, analysis)
		if err != nil {
			result.Message = fmt.Sprintf("memory extraction failed: %v", err)
		} else {
			result.ShortTermMemory = items
		}
	}
	
	// Generate skill
	if analysis.EvolutionPotential.HasSkillPotential {
		skill, err := e.GenerateSkill(ctx, analysis)
		if err != nil {
			result.Message = fmt.Sprintf("skill generation failed: %v", err)
		} else {
			result.GeneratedSkill = skill
		}
	}
	
	// Generate tool
	if analysis.EvolutionPotential.HasToolPotential {
		tool, err := e.GenerateTool(ctx, analysis)
		if err != nil {
			result.Message = fmt.Sprintf("tool generation failed: %v", err)
		} else {
			result.GeneratedTool = tool
		}
	}
	
	// Mark session as evolved
	if err := e.MarkEvolved(ctx, sessionName); err != nil {
		result.Message = fmt.Sprintf("failed to mark session as evolved: %v", err)
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
	
	// Get session from memory
	if e.memory == nil {
		return analysis, nil
	}
	
	session, err := e.memory.Sessions().Get(ctx, sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	if session == nil {
		return analysis, nil
	}
	
	// Get messages for this session from memory
	history, err := e.memory.Sessions().GetHistory(ctx, sessionName)
	if err == nil && history != nil {
		for _, msg := range history.Messages {
			// Extract key information
			keyInfo := e.extractKeyInfoFromMessage(msg)
			if keyInfo != nil {
				analysis.KeyInfo = append(analysis.KeyInfo, keyInfo)
			}
		}
	}
	
	// Get trajectory from memory if available
	trajectory, err := e.memory.Trajectories().Get(ctx, sessionName)
	if err == nil && trajectory != nil {
		analysis.Patterns = e.detectPatterns(trajectory)
		analysis.Repetitions = e.detectRepetitions(trajectory)
	}
	
	// Calculate evolution potential
	analysis.EvolutionPotential = e.calculateEvolutionPotential(analysis)
	
	return analysis, nil
}

// AnalyzeSessionWithLLM analyzes a session using LLM
func (e *Evolver) AnalyzeSessionWithLLM(ctx context.Context, sessionName string) (*SessionAnalysis, error) {
	if e.llm == nil {
		return e.AnalyzeSession(ctx, sessionName)
	}
	
	// Get session history from memory
	session, err := e.memory.Sessions().Get(ctx, sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	if session == nil {
		return &SessionAnalysis{
			SessionName: sessionName,
			Patterns:    []PatternMatch{},
			Repetitions: []RepetitionInfo{},
			KeyInfo:     []*KeyInfo{},
		}, nil
	}
	
	// Build analysis prompt
	prompt := e.buildAnalysisPrompt(session)
	
	// Call LLM
	resp, err := e.llm.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		// Fallback to basic analysis
		return e.AnalyzeSession(ctx, sessionName)
	}
	
	// Parse response into SessionAnalysis
	analysis, err := e.parseAnalysisResponse(resp.Content, sessionName)
	if err != nil {
		// Fallback to basic analysis
		return e.AnalyzeSession(ctx, sessionName)
	}
	
	return analysis, nil
}

// GenerateSkill generates a skill from session analysis
func (e *Evolver) GenerateSkill(ctx context.Context, analysis *SessionAnalysis) (*GeneratedSkill, error) {
	if e.llm == nil {
		return nil, fmt.Errorf("LLM client is required for skill generation")
	}
	
	// Check if there are enough repeated patterns
	validPatterns := 0
	for _, pattern := range analysis.Patterns {
		if pattern.SuggestedType == SuggestionTypeSkill && pattern.Occurrences >= e.config.SkillThreshold {
			validPatterns++
		}
	}
	
	if validPatterns == 0 {
		return nil, fmt.Errorf("no valid skill patterns found")
	}
	
	// Build skill generation prompt
	prompt := e.buildSkillGenerationPrompt(analysis)
	
	// Call LLM
	resp, err := e.llm.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Parse skill from response
	genSkill, err := e.parseSkillFromResponse(resp.Content, analysis.SessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}
	
	// Save skill to file system
	if e.config.EnableAutoEvolution {
		if err := e.saveSkillToFile(genSkill); err != nil {
			return nil, fmt.Errorf("failed to save skill: %w", err)
		}
		
		// Register skill in memory
		if e.memory != nil {
			skillNode := &skill.SkillNode{
				Name:         genSkill.Name,
				NodeType:     "skill",
				Description:  genSkill.Description,
				Template:     genSkill.Content,
				Parameters:   e.convertSkillParameters(genSkill.Parameters),
			}
			if err := e.memory.Skills().Add(ctx, skillNode); err != nil {
				return nil, fmt.Errorf("failed to register skill: %w", err)
			}
		}
	}
	
	return genSkill, nil
}

// GenerateTool generates a tool from session analysis
func (e *Evolver) GenerateTool(ctx context.Context, analysis *SessionAnalysis) (*GeneratedTool, error) {
	if e.llm == nil {
		return nil, fmt.Errorf("LLM client is required for tool generation")
	}
	
	// Check if there are enough repeated operations
	validRepetitions := 0
	for _, rep := range analysis.Repetitions {
		if rep.CanBeAutomated && rep.Count >= e.config.ToolThreshold {
			validRepetitions++
		}
	}
	
	if validRepetitions == 0 {
		return nil, fmt.Errorf("no valid tool patterns found")
	}
	
	// Build tool generation prompt
	prompt := e.buildToolGenerationPrompt(analysis)
	
	// Call LLM
	resp, err := e.llm.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Parse tool from response
	genTool, err := e.parseToolFromResponse(resp.Content, analysis.SessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool: %w", err)
	}
	
	// Save tool to file system
	if e.config.EnableAutoEvolution {
		if err := e.saveToolToFile(genTool); err != nil {
			return nil, fmt.Errorf("failed to save tool: %w", err)
		}
		
		// Register tool in memory
		if e.memory != nil {
			toolNode := &tool.ToolNode{
				Name:          genTool.Name,
				NodeType:      "tool",
				Description:   genTool.Description,
				Type:          goreactcommon.ToolType(genTool.ToolType),
				SecurityLevel: genTool.SecurityLevel,
			}
			if err := e.memory.Tools().Add(ctx, toolNode); err != nil {
				return nil, fmt.Errorf("failed to register tool: %w", err)
			}
		}
	}
	
	return genTool, nil
}

// ExtractShortTermMemory extracts short-term memory from analysis
func (e *Evolver) ExtractShortTermMemory(ctx context.Context, analysis *SessionAnalysis) ([]*goreactcore.MemoryItemNode, error) {
	if e.memory == nil {
		return nil, fmt.Errorf("memory is not initialized")
	}
	
	items := []*goreactcore.MemoryItemNode{}
	
	// Extract from key information
	for _, info := range analysis.KeyInfo {
		if info.ShouldMemorize && info.Importance >= e.config.MemoryImportanceThreshold {
			item := goreactcore.NewMemoryItemNode(
				analysis.SessionName,
				info.Content,
				goreactcommon.MemoryItemType(info.Category),
			)
			item.Importance = info.Importance
			item.Source = goreactcommon.MemorySourceEvolution
			items = append(items, item)
		}
	}
	
	// Store items in memory
	for _, item := range items {
		if _, err := e.memory.ShortTerms().Add(ctx, analysis.SessionName, item); err != nil {
			// Log error but continue with other items
			continue
		}
	}
	
	return items, nil
}

// MarkEvolved marks a session as evolved
func (e *Evolver) MarkEvolved(ctx context.Context, sessionName string) error {
	if e.memory == nil {
		return nil
	}
	
	// Get session
	session, err := e.memory.Sessions().Get(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionName)
	}
	
	// Create evolution record
	record := &EvolutionRecord{
		SessionName: sessionName,
		EvolvedAt:   time.Now(),
		Status:      goreactcommon.EvolutionStatusCompleted,
	}
	
	// Store record
	e.records[sessionName] = record
	
	return nil
}

// EvolutionResult represents the result of evolution
type EvolutionResult struct {
	SessionName      string                         `json:"session_name"`
	ShortTermMemory  []*goreactcore.MemoryItemNode  `json:"short_term_memory"`
	GeneratedSkill   *GeneratedSkill                `json:"generated_skill"`
	GeneratedTool    *GeneratedTool                 `json:"generated_tool"`
	EvolvedAt        time.Time                      `json:"evolved_at"`
	Success          bool                           `json:"success"`
	Message          string                         `json:"message"`
}

// GeneratedSkill represents a generated skill from evolution
type GeneratedSkill struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Content       string            `json:"content"`
	FilePath      string            `json:"file_path"`
	Parameters    []SkillParameter  `json:"parameters"`
	Examples      []string          `json:"examples"`
	CreatedAt     time.Time         `json:"created_at"`
	SourceSession string            `json:"source_session"`
}

// GeneratedTool represents a generated tool from evolution
type GeneratedTool struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	FilePath      string            `json:"file_path"`
	ToolType      ToolType          `json:"tool_type"`
	Parameters    []ToolParameter   `json:"parameters"`
	ReturnType    string            `json:"return_type"`
	SecurityLevel goreactcommon.SecurityLevel `json:"security_level"`
	CreatedAt     time.Time         `json:"created_at"`
	SourceSession string            `json:"source_session"`
}

// ToolType represents the type of generated tool
type ToolType string

const (
	ToolTypePython ToolType = "python"
	ToolTypeCLI    ToolType = "cli"
	ToolTypeBash   ToolType = "bash"
)

// SkillParameter represents a parameter for a generated skill
type SkillParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

// ToolParameter represents a parameter for a generated tool
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// SessionAnalysis represents the analysis of a session
type SessionAnalysis struct {
	SessionName       string                     `json:"session_name"`
	Messages          []*goreactcore.MessageNode `json:"messages"`
	Patterns          []PatternMatch            `json:"patterns"`
	Repetitions       []RepetitionInfo          `json:"repetitions"`
	KeyInfo           []*KeyInfo                `json:"key_info"`
	EvolutionPotential EvolutionPotential        `json:"evolution_potential"`
}

// PatternMatch represents a matched pattern
type PatternMatch struct {
	Pattern       string         `json:"pattern"`
	Occurrences   int            `json:"occurrences"`
	Contexts      []string       `json:"contexts"`
	SuggestedType SuggestionType `json:"suggested_type"`
}

// RepetitionInfo represents repeated operations
type RepetitionInfo struct {
	Operation      string  `json:"operation"`
	Count          int     `json:"count"`
	Similarity     float64 `json:"similarity"`
	CanBeAutomated bool    `json:"can_be_automated"`
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
	SessionName      string                       `json:"session_name"`
	EvolvedAt        time.Time                    `json:"evolved_at"`
	GeneratedSkills  []string                     `json:"generated_skills"`
	GeneratedTools   []string                     `json:"generated_tools"`
	ExtractedMemory  []string                     `json:"extracted_memory"`
	Status           goreactcommon.EvolutionStatus `json:"status"`
	Error            string                       `json:"error"`
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
	// Check in-memory records first
	if record, exists := e.records[sessionName]; exists {
		return record, nil
	}
	
	// Query from memory/GraphRAG
	if e.memory != nil && e.memory.graphRAG != nil {
		// Query evolution records from GraphRAG
		results, err := e.memory.graphRAG.QueryGraph(ctx, fmt.Sprintf(
			"MATCH (e:EvolutionRecord {session_name: '%s'}) RETURN e", sessionName), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query evolution history: %w", err)
		}

		if len(results) > 0 {
			// Parse result from map[string]any
			if recordData, ok := results[0]["e"].(map[string]any); ok {
				record := &EvolutionRecord{}
				if sessionNameVal, ok := recordData["session_name"].(string); ok {
					record.SessionName = sessionNameVal
				}
				return record, nil
			}
		}
	}
	
	return nil, fmt.Errorf("no evolution history found for session: %s", sessionName)
}

// ListPendingEvolution lists sessions pending evolution
func (e *Evolver) ListPendingEvolution(ctx context.Context) ([]string, error) {
	if e.memory == nil {
		return nil, fmt.Errorf("memory is not initialized")
	}
	
	// Get all sessions
	sessions, err := e.memory.Sessions().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	
	pending := []string{}
	for _, node := range sessions {
		// Get session name from node ID
		sessionName := node.ID
		if sessionName == "" {
			continue
		}
		
		// Check if session has evolution potential
		analysis, err := e.AnalyzeSession(ctx, sessionName)
		if err != nil {
			continue
		}
		
		if analysis.EvolutionPotential.HasMemoryPotential ||
			analysis.EvolutionPotential.HasSkillPotential ||
			analysis.EvolutionPotential.HasToolPotential {
			pending = append(pending, sessionName)
		}
	}
	
	return pending, nil
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

// Private helper methods

func (e *Evolver) buildAnalysisPrompt(session *goreactcore.SessionNode) string {
	var sb strings.Builder
	
	sb.WriteString(`Analyze this conversation session for evolution potential.
Identify:
1. Repeated patterns that could become skills
2. Tool usage patterns that could become tools
3. Important information that should be remembered

Session: ` + session.Name + `

`)
	
	// Get messages from memory if available
	if e.memory != nil {
		history, err := e.memory.Sessions().GetHistory(context.Background(), session.Name)
		if err == nil && history != nil {
			sb.WriteString("Messages:\n")
			for i, msg := range history.Messages {
				sb.WriteString(fmt.Sprintf("%d. [%s]: %s\n", i+1, msg.Role, msg.Content))
			}
		}
	}
	
	sb.WriteString(`
Analyze and respond in JSON format:
{
  "patterns": [{"pattern": "description", "occurrences": N, "suggested_type": "skill|tool|memory"}],
  "repetitions": [{"operation": "description", "count": N, "similarity": 0.0-1.0, "can_be_automated": true|false}],
  "key_info": [{"content": "information", "importance": 0.0-1.0, "category": "rule|fact|knowledge|task", "should_memorize": true|false}],
  "evolution_potential": {"has_skill_potential": true|false, "has_tool_potential": true|false, "has_memory_potential": true|false, "confidence": 0.0-1.0}
}`)
	
	return sb.String()
}

func (e *Evolver) parseAnalysisResponse(response, sessionName string) (*SessionAnalysis, error) {
	// Try to parse JSON response
	var parsed struct {
		Patterns          []PatternMatch     `json:"patterns"`
		Repetitions       []RepetitionInfo   `json:"repetitions"`
		KeyInfo           []*KeyInfo         `json:"key_info"`
		EvolutionPotential EvolutionPotential `json:"evolution_potential"`
	}

	if err := goreactcommon.ParseJSONObject(response, &parsed); err != nil {
		return nil, err
	}
	
	return &SessionAnalysis{
		SessionName:       sessionName,
		Patterns:          parsed.Patterns,
		Repetitions:       parsed.Repetitions,
		KeyInfo:           parsed.KeyInfo,
		EvolutionPotential: parsed.EvolutionPotential,
	}, nil
}

func (e *Evolver) buildSkillGenerationPrompt(analysis *SessionAnalysis) string {
	return fmt.Sprintf(`Based on the following session analysis, generate a SKILL.md definition.

Session: %s
Patterns: %v

Generate a skill definition in the following format:
---
name: skill-name
description: What this skill does
allowed-tools: tool1 tool2
---

# Skill Instructions
[Detailed instructions for using this skill]

## Parameters
- param1: description (required)
- param2: description (optional, default: value)

## Steps
1. Step one
2. Step two
3. Step three

## Examples
- Example 1
- Example 2
`, analysis.SessionName, analysis.Patterns)
}

func (e *Evolver) buildToolGenerationPrompt(analysis *SessionAnalysis) string {
	return fmt.Sprintf(`Based on the following session analysis, generate a tool implementation.

Session: %s
Repetitions: %v

Generate a tool with:
1. Clear parameter definitions
2. Return type specification
3. Security level assessment

Respond in JSON format:
{
  "name": "tool-name",
  "description": "What this tool does",
  "tool_type": "python|cli|bash",
  "parameters": [{"name": "param", "type": "string", "description": "param description"}],
  "return_type": "description of return value",
  "security_level": "safe|sensitive|high_risk",
  "code": "the actual implementation code"
}
`, analysis.SessionName, analysis.Repetitions)
}

func (e *Evolver) parseSkillFromResponse(response, sessionName string) (*GeneratedSkill, error) {
	// Extract skill name from response
	name := e.extractSkillName(response)
	if name == "" {
		name = fmt.Sprintf("skill-%s-%d", sessionName, time.Now().Unix())
	}
	
	// Get output path from config or use default
	outputPath := "./skills"
	if e.config != nil {
		outputPath = e.config.SkillOutputPath
		if outputPath == "" {
			outputPath = "./skills"
		}
	}
	
	skill := &GeneratedSkill{
		Name:          name,
		Description:   e.extractDescription(response),
		Content:       response,
		FilePath:      filepath.Join(outputPath, name, "SKILL.md"),
		Parameters:    e.extractParameters(response),
		Examples:      e.extractExamples(response),
		CreatedAt:     time.Now(),
		SourceSession: sessionName,
	}
	
	return skill, nil
}

func (e *Evolver) parseToolFromResponse(response, sessionName string) (*GeneratedTool, error) {
	// Try to parse JSON response
	var parsed struct {
		Name          string                          `json:"name"`
		Description   string                          `json:"description"`
		ToolType      string                          `json:"tool_type"`
		Parameters    []ToolParameter                 `json:"parameters"`
		ReturnType    string                          `json:"return_type"`
		SecurityLevel goreactcommon.SecurityLevel     `json:"security_level"`
	}
	
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	
	// Get output path from config or use default
	outputPath := "./tools"
	if e.config != nil {
		outputPath = e.config.ToolOutputPath
		if outputPath == "" {
			outputPath = "./tools"
		}
	}
	
	if jsonStart != -1 && jsonEnd != -1 {
		jsonStr := response[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
			return &GeneratedTool{
				Name:          parsed.Name,
				Description:   parsed.Description,
				FilePath:      filepath.Join(outputPath, parsed.Name+".py"),
				ToolType:      ToolType(parsed.ToolType),
				Parameters:    parsed.Parameters,
				ReturnType:    parsed.ReturnType,
				SecurityLevel: parsed.SecurityLevel,
				CreatedAt:     time.Now(),
				SourceSession: sessionName,
			}, nil
		}
	}
	
	// Fallback: create tool from response
	name := fmt.Sprintf("tool-%s-%d", sessionName, time.Now().Unix())
	return &GeneratedTool{
		Name:          name,
		Description:   e.extractDescription(response),
		FilePath:      filepath.Join(outputPath, name+".py"),
		ToolType:      ToolTypePython,
		Parameters:    []ToolParameter{},
		ReturnType:    "map[string]any",
		SecurityLevel: goreactcommon.LevelSensitive,
		CreatedAt:     time.Now(),
		SourceSession: sessionName,
	}, nil
}

func (e *Evolver) saveSkillToFile(skill *GeneratedSkill) error {
	// Create skill directory
	skillDir := filepath.Dir(skill.FilePath)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}
	
	// Write skill content
	if err := os.WriteFile(skill.FilePath, []byte(skill.Content), 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}
	
	return nil
}

func (e *Evolver) saveToolToFile(tool *GeneratedTool) error {
	// Create tools directory if not exists
	toolsDir := filepath.Dir(tool.FilePath)
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}
	
	// Extract code from tool if available
	code := ""
	if tool.ToolType == ToolTypePython {
		code = e.generatePythonToolCode(tool)
	} else if tool.ToolType == ToolTypeBash {
		code = e.generateBashToolCode(tool)
	}
	
	// Write tool file
	if err := os.WriteFile(tool.FilePath, []byte(code), 0755); err != nil {
		return fmt.Errorf("failed to write tool file: %w", err)
	}
	
	return nil
}

func (e *Evolver) generatePythonToolCode(tool *GeneratedTool) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf(`#!/usr/bin/env python3
# Auto-generated tool: %s
# Description: %s
# Source: %s

import json
import sys

def %s(**kwargs):
    """
    %s
    """
    # TODO: Implement tool logic
    result = {
        "success": True,
        "output": kwargs
    }
    return result

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
`, tool.Name, tool.Description, tool.SourceSession, tool.Name, tool.Description))
	
	for _, param := range tool.Parameters {
		sb.WriteString(fmt.Sprintf("    parser.add_argument('--%s', required=True, help='%s')\n", param.Name, param.Description))
	}
	
	sb.WriteString(`
    args = parser.parse_args()
    result = ` + tool.Name + `(**vars(args))
    print(json.dumps(result))
`)
	
	return sb.String()
}

func (e *Evolver) generateBashToolCode(tool *GeneratedTool) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf(`#!/bin/bash
# Auto-generated tool: %s
# Description: %s
# Source: %s

`, tool.Name, tool.Description, tool.SourceSession))
	
	for i, param := range tool.Parameters {
		sb.WriteString(fmt.Sprintf("%s=\"${%d:-}\"\n", strings.ToUpper(param.Name), i+1))
	}
	
	sb.WriteString(`
# TODO: Implement tool logic
echo "{\"success\": true, \"output\": \"Tool executed\"}"
`)
	
	return sb.String()
}

func (e *Evolver) extractKeyInfoFromMessage(msg *goreactcore.MessageNode) *KeyInfo {
	if msg == nil || msg.Content == "" {
		return nil
	}
	
	content := msg.Content
	
	// Check for rule patterns
	ruleKeywords := []string{"必须", "禁止", "要记住", "不要", "记住", "must", "always", "never"}
	for _, kw := range ruleKeywords {
		if strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
			return &KeyInfo{
				Content:        content,
				Importance:     0.8,
				Category:       "rule",
				ShouldMemorize: true,
			}
		}
	}
	
	// Check for fact patterns
	factKeywords := []string{"是", "位于", "成立于", "is located", "was founded", "成立于"}
	for _, kw := range factKeywords {
		if strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
			return &KeyInfo{
				Content:        content,
				Importance:     0.7,
				Category:       "fact",
				ShouldMemorize: true,
			}
		}
	}
	
	// Check for knowledge patterns
	knowledgeKeywords := []string{"如何", "方法", "步骤", "how to", "method", "step"}
	for _, kw := range knowledgeKeywords {
		if strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
			return &KeyInfo{
				Content:        content,
				Importance:     0.6,
				Category:       "knowledge",
				ShouldMemorize: true,
			}
		}
	}
	
	return nil
}

func (e *Evolver) detectPatterns(trajectory *goreactcore.TrajectoryNode) []PatternMatch {
	patterns := []PatternMatch{}
	
	if trajectory == nil || len(trajectory.Steps) == 0 {
		return patterns
	}
	
	// Detect action patterns
	actionCounts := make(map[string]int)
	actionContexts := make(map[string][]string)
	
	for _, step := range trajectory.Steps {
		if step.Action != nil && step.Action.Target != "" {
			actionKey := string(step.Action.Type) + ":" + step.Action.Target
			actionCounts[actionKey]++
			if step.Thought != nil {
				actionContexts[actionKey] = append(actionContexts[actionKey], step.Thought.Content)
			}
		}
	}
	
	for action, count := range actionCounts {
		if count >= e.config.SkillThreshold {
			patterns = append(patterns, PatternMatch{
				Pattern:       action,
				Occurrences:   count,
				Contexts:      actionContexts[action],
				SuggestedType: SuggestionTypeSkill,
			})
		}
	}
	
	return patterns
}

func (e *Evolver) detectRepetitions(trajectory *goreactcore.TrajectoryNode) []RepetitionInfo {
	repetitions := []RepetitionInfo{}
	
	if trajectory == nil || len(trajectory.Steps) == 0 {
		return repetitions
	}
	
	// Detect repeated tool calls
	toolCounts := make(map[string]int)
	toolSimilarity := make(map[string]float64)
	
	for _, step := range trajectory.Steps {
		if step.Action != nil && step.Action.Target != "" {
			toolCounts[step.Action.Target]++
			toolSimilarity[step.Action.Target] = 0.9 // High similarity for exact matches
		}
	}
	
	for tool, count := range toolCounts {
		if count >= e.config.ToolThreshold {
			repetitions = append(repetitions, RepetitionInfo{
				Operation:      tool,
				Count:          count,
				Similarity:     toolSimilarity[tool],
				CanBeAutomated: true,
			})
		}
	}
	
	return repetitions
}

func (e *Evolver) calculateEvolutionPotential(analysis *SessionAnalysis) EvolutionPotential {
	potential := EvolutionPotential{
		HasSkillPotential:  false,
		HasToolPotential:   false,
		HasMemoryPotential: false,
		Confidence:         0.5,
	}
	
	// Check for skill potential
	for _, pattern := range analysis.Patterns {
		if pattern.Occurrences >= e.config.SkillThreshold {
			potential.HasSkillPotential = true
			potential.Confidence += 0.2
		}
	}
	
	// Check for tool potential
	for _, rep := range analysis.Repetitions {
		if rep.Count >= e.config.ToolThreshold && rep.CanBeAutomated {
			potential.HasToolPotential = true
			potential.Confidence += 0.2
		}
	}
	
	// Check for memory potential
	for _, info := range analysis.KeyInfo {
		if info.Importance >= e.config.MemoryImportanceThreshold && info.ShouldMemorize {
			potential.HasMemoryPotential = true
			potential.Confidence += 0.1
		}
	}
	
	// Cap confidence at 1.0
	if potential.Confidence > 1.0 {
		potential.Confidence = 1.0
	}
	
	return potential
}

func (e *Evolver) extractSkillName(response string) string {
	// Try to extract from frontmatter
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "name:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
	}
	return ""
}

func (e *Evolver) extractDescription(response string) string {
	// Try to extract from frontmatter
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return "Auto-generated skill"
}

func (e *Evolver) extractParameters(response string) []SkillParameter {
	params := []SkillParameter{}
	
	// Look for parameter section
	inParamsSection := false
	lines := strings.Split(response, "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "## Parameters") || strings.Contains(line, "## 参数") {
			inParamsSection = true
			continue
		}
		
		if inParamsSection && strings.HasPrefix(line, "##") {
			break
		}
		
		if inParamsSection && strings.HasPrefix(line, "-") {
			// Parse parameter line
			param := e.parseParameterLine(line)
			if param.Name != "" {
				params = append(params, param)
			}
		}
	}
	
	return params
}

func (e *Evolver) parseParameterLine(line string) SkillParameter {
	// Format: "- param: description (required)" or "- param: description (optional, default: value)"
	line = strings.TrimPrefix(line, "- ")
	parts := strings.SplitN(line, ":", 2)
	
	param := SkillParameter{}
	if len(parts) >= 1 {
		param.Name = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		desc := strings.TrimSpace(parts[1])
		
		if strings.Contains(desc, "(required)") || strings.Contains(desc, "（必需）") {
			param.Required = true
			desc = strings.ReplaceAll(desc, "(required)", "")
			desc = strings.ReplaceAll(desc, "（必需）", "")
		} else if strings.Contains(desc, "(optional") || strings.Contains(desc, "（可选") {
			param.Required = false
			// Extract default value
			if idx := strings.Index(desc, "default:"); idx != -1 {
				defaultPart := desc[idx+8:]
				if endIdx := strings.Index(defaultPart, ")"); endIdx != -1 {
					param.Default = strings.TrimSpace(defaultPart[:endIdx])
				}
			}
			desc = strings.Split(desc, "(")[0]
			desc = strings.Split(desc, "（")[0]
		}
		
		param.Description = strings.TrimSpace(desc)
	}
	
	param.Type = "string" // Default type
	
	return param
}

func (e *Evolver) extractExamples(response string) []string {
	examples := []string{}
	
	// Look for examples section
	inExamplesSection := false
	lines := strings.Split(response, "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "## Examples") || strings.Contains(line, "## 示例") {
			inExamplesSection = true
			continue
		}
		
		if inExamplesSection && strings.HasPrefix(line, "##") {
			break
		}
		
		if inExamplesSection && strings.HasPrefix(line, "-") {
			example := strings.TrimPrefix(line, "- ")
			examples = append(examples, strings.TrimSpace(example))
		}
	}
	
	return examples
}

func (e *Evolver) convertSkillParameters(params []SkillParameter) []skill.Parameter {
	result := make([]skill.Parameter, len(params))
	for i, p := range params {
		result[i] = skill.Parameter{
			Name:        p.Name,
			Type:        p.Type,
			Required:    p.Required,
			Default:     p.Default,
			Description: p.Description,
		}
	}
	return result
}

func (e *Evolver) convertToolParameters(params []ToolParameter) []goreactcore.ParameterSpec {
	result := make([]goreactcore.ParameterSpec, len(params))
	for i, p := range params {
		result[i] = goreactcore.ParameterSpec{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
		}
	}
	return result
}
