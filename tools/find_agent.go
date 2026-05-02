package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// FindAgentTool searches for registered agents by name, role, or description.
type FindAgentTool struct {
	runtimeDir *core.RuntimeDirectory
}

// NewFindAgentTool creates a FindAgentTool.
func NewFindAgentTool(runtimeDir *core.RuntimeDirectory) *FindAgentTool {
	return &FindAgentTool{runtimeDir: runtimeDir}
}

func (t *FindAgentTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "FindAgent",
		Description: "Search for registered agents by name, role, description, or capability. Returns matching agents with their current state, score, and task count.",
		Prompt: `Search for an expert agent when the current task falls outside your role or expertise. Use this before Delegate to find the right agent to hand the task to.

When to use:
- The user's request is outside your defined area of expertise.
- You need a specialist capability that you do not have.
- You are looking for an agent whose role matches the task domain.

How it works:
- Describe what capability you need — e.g. "data analysis", "security audit", "legal review"
- Search checks agent name, role, and description (case-insensitive)
- Results show each agent's current state (idle/busy/error) and past performance score
- Pick an available agent with a good score, then use Delegate to dispatch the task
- If no matching agent exists, consider CreateAgent to define a new one`,
		Tags: []string{"agent", "search", "discovery", "orchestration"},
		Parameters: []core.Parameter{
			{Name: "query", Type: "string", Description: "Search query — matches agent name, role, or description (case-insensitive)", Required: true},
			{Name: "min_score", Type: "number", Description: "Optional minimum performance score filter (0-3, default: 0)", Required: false},
			{Name: "available_only", Type: "boolean", Description: "If true, only return agents in idle or dormant state (default: false)", Required: false},
		},
	}
}

func (t *FindAgentTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if t.runtimeDir == nil {
		return nil, fmt.Errorf("agent runtime directory not configured")
	}

	results := t.runtimeDir.FindByDescription(query)
	if results == nil {
		results = []*core.AgentRuntimeMeta{}
	}

	// Post-filter
	minScore := 0.0
	if ms, ok := params["min_score"].(float64); ok {
		minScore = ms
	}
	availableOnly, _ := params["available_only"].(bool)

	var filtered []map[string]any
	for _, meta := range results {
		if meta.Score < minScore {
			continue
		}
		if availableOnly && !meta.IsAvailable() {
			continue
		}
		filtered = append(filtered, map[string]any{
			"name":        meta.Config.Name,
			"role":        meta.Config.Role,
			"description": meta.Config.Description,
			"state":       string(meta.State),
			"score":       meta.Score,
			"task_count":  meta.TaskCount,
			"available":   meta.IsAvailable(),
		})
	}

	return map[string]any{
		"agents":     filtered,
		"count":      len(filtered),
		"total_found": len(results),
	}, nil
}

