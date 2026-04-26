package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// MemoryAccessor provides the memory interface to tools.
// It is set by the reactor when the memory tool is registered.
type MemoryAccessor struct {
	memory core.Memory
}

// SetMemory sets the memory instance for the accessor.
func (a *MemoryAccessor) SetMemory(m core.Memory) {
	a.memory = m
}

// Memory returns the current memory instance.
func (a *MemoryAccessor) Memory() core.Memory {
	return a.memory
}

// memoryAccessor is the package-level accessor shared by memory tools.
var memoryAccessor = &MemoryAccessor{}

// SetMemory sets the package-level memory instance.
// Call this before using the memory tools.
func SetMemory(m core.Memory) {
	memoryAccessor.SetMemory(m)
}

// --- memory_save tool ---

// MemorySave lets the LLM save knowledge to memory.
// This enables the agent to persist important findings, conventions, and decisions
// across sessions — similar to ClueCode's Session Memory mechanism.
type MemorySave struct{}

// NewMemorySaveTool creates a tool that lets the LLM save knowledge to memory.
func NewMemorySaveTool() core.FuncTool {
	return &MemorySave{}
}

func (t *MemorySave) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "memory_save",
		Description: "Save important knowledge, conventions, or findings to long-term memory. Use this to persist information that should be remembered across conversations. Params: {title: string, content: string (required), type: 'session'|'user'|'longterm'|'reflexive'|'experience', scope: 'private'|'team', tags: 'comma,separated', id: 'existing_id_to_update'}",
	}
}

func (t *MemorySave) Execute(ctx context.Context, params map[string]any) (any, error) {
	mem := memoryAccessor.Memory()
	if mem == nil {
		return nil, fmt.Errorf("memory is not configured")
	}

	content, _ := params["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	title, _ := params["title"].(string)
	memTypeStr, _ := params["type"].(string)
	scopeStr, _ := params["scope"].(string)
	tagsStr, _ := params["tags"].(string)
	id, _ := params["id"].(string)

	// Parse tags
	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Parse memory type
	memType := parseMemoryType(memTypeStr)

	// Parse scope
	scope := core.MemoryScopePrivate
	if scopeStr == "team" {
		scope = core.MemoryScopeTeam
	}

	record := core.MemoryRecord{
		Title:   title,
		Content: content,
		Type:    memType,
		Scope:   scope,
		Tags:    tags,
	}

	// Update or create
	if id != "" {
		if err := mem.Update(ctx, id, record); err != nil {
			return nil, fmt.Errorf("failed to update memory: %w", err)
		}
		return fmt.Sprintf("Memory updated: %s", id), nil
	}

	newID, err := mem.Store(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("failed to save memory: %w", err)
	}
	return fmt.Sprintf("Memory saved: %s", newID), nil
}

// --- memory_search tool ---

// MemorySearch lets the LLM search memory.
// This enables the agent to proactively recall relevant knowledge.
type MemorySearch struct{}

// NewMemorySearchTool creates a tool that lets the LLM search memory.
func NewMemorySearchTool() core.FuncTool {
	return &MemorySearch{}
}

func (t *MemorySearch) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "memory_search",
		Description: "Search long-term memory for relevant knowledge, conventions, or past findings. Use this to recall information stored in previous conversations. Params: {query: string (required), type: 'session'|'user'|'longterm'|'reflexive'|'experience', scope: 'private'|'team', limit: int}",
	}
}

func (t *MemorySearch) Execute(ctx context.Context, params map[string]any) (any, error) {
	mem := memoryAccessor.Memory()
	if mem == nil {
		return nil, fmt.Errorf("memory is not configured")
	}

	query, _ := params["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	typeStr, _ := params["type"].(string)
	scopeStr, _ := params["scope"].(string)
	limitRaw := 0
	if raw, ok := params["limit"]; ok {
		if f, ok := raw.(float64); ok {
			limitRaw = int(f)
		} else if i, ok := raw.(int); ok {
			limitRaw = i
		}
	}

	// Build retrieve options
	var opts []core.RetrieveOption
	if typeStr != "" {
		memType := parseMemoryType(typeStr)
		opts = append(opts, core.WithMemoryTypes(memType))
	}
	if scopeStr == "team" {
		opts = append(opts, core.WithMemoryScope(core.MemoryScopeTeam))
	}
	if limitRaw > 0 {
		opts = append(opts, core.WithMemoryLimit(limitRaw))
	}

	records, err := mem.Retrieve(ctx, query, opts...)
	if err != nil {
		return nil, fmt.Errorf("memory search failed: %w", err)
	}

	if len(records) == 0 {
		return "No relevant memories found.", nil
	}

	// Format results
	var result strings.Builder
	for i, r := range records {
		typeName := memoryTypeLabel(r.Type)
		scoreStr := ""
		if r.Score > 0 {
			scoreStr = fmt.Sprintf(" (score: %.2f)", r.Score)
		}
		fmt.Fprintf(&result, "%d. [%s]%s %s\n", i+1, typeName, scoreStr, r.Title)
		fmt.Fprintf(&result, "   %s\n", truncateContent(r.Content, 200))
		if len(r.Tags) > 0 {
			fmt.Fprintf(&result, "   Tags: %s\n", strings.Join(r.Tags, ", "))
		}
		fmt.Fprintf(&result, "   ID: %s\n", r.ID)
	}

	return result.String(), nil
}

// --- Helper functions ---

func parseMemoryType(s string) core.MemoryType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "session", "temporary":
		return core.MemoryTypeSession
	case "user", "preference", "shortterm":
		return core.MemoryTypeUser
	case "longterm", "knowledge", "project", "reference":
		return core.MemoryTypeLongTerm
	case "reflexive", "refactive", "tool", "skill":
		return core.MemoryTypeReflexive
	case "experience", "exp":
		return core.MemoryTypeExperience
	default:
		return core.MemoryTypeSession
	}
}

func memoryTypeLabel(t core.MemoryType) string {
	switch t {
	case core.MemoryTypeSession:
		return "Session"
	case core.MemoryTypeUser:
		return "User"
	case core.MemoryTypeLongTerm:
		return "Long-term"
	case core.MemoryTypeReflexive:
		return "Reflexive"
	case core.MemoryTypeExperience:
		return "Experience"
	default:
		return "Unknown"
	}
}

func truncateContent(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
