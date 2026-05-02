package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// RankTool records a performance score for a sub-agent after task completion.
type RankTool struct {
	runtimeDir *core.RuntimeDirectory
}

// NewRankTool creates a RankTool.
func NewRankTool(runtimeDir *core.RuntimeDirectory) *RankTool {
	return &RankTool{runtimeDir: runtimeDir}
}

func (t *RankTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "Rank",
		Description: "Record a performance score for a sub-agent after task completion. Scoring helps the system learn which agents are most reliable for which tasks.",
		Prompt: `Rate a sub-agent's performance after a delegated task completes. Scores are used by FindAgent to prioritize more reliable agents in future searches.

Usage:
- Call this after CollectResults returns a completed task result
- Score based on result quality: 0=poor, 1=below average, 2=average, 3=excellent
- Provide a brief reasoning to explain the score (this is stored for audit)
- Only score tasks that completed — do not score failed or cancelled tasks

Scoring guidelines:
- 3 (excellent): Exceeded expectations, thorough, minimal supervision needed
- 2 (good): Met expectations, did the job correctly
- 1 (needs improvement): Partial result, required significant corrections
- 0 (poor): Low quality, irrelevant output, or required full redo`,
		Tags: []string{"agent", "score", "rating", "performance", "orchestration"},
		Parameters: []core.Parameter{
			{Name: "agent_name", Type: "string", Description: "Name of the agent being scored", Required: true},
			{Name: "task_id", Type: "string", Description: "Task ID of the completed task (from Delegate response)", Required: true},
			{Name: "score", Type: "number", Description: "Performance score: 0 (poor) to 3 (excellent)", Required: true},
			{Name: "reasoning", Type: "string", Description: "Brief explanation of the score for audit trail", Required: false},
		},
	}
}

func (t *RankTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	agentName, _ := params["agent_name"].(string)
	if agentName == "" {
		return nil, fmt.Errorf("agent_name is required")
	}
	taskID, _ := params["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	score, ok := params["score"].(float64)
	if !ok || score < 0 || score > 3 {
		return nil, fmt.Errorf("score must be a number between 0 and 3")
	}
	reasoning, _ := params["reasoning"].(string)

	if t.runtimeDir == nil {
		return nil, fmt.Errorf("agent runtime directory not configured")
	}

	meta := t.runtimeDir.Get(agentName)
	if meta == nil {
		return nil, fmt.Errorf("agent %q not found in runtime directory", agentName)
	}

	t.runtimeDir.SetScore(agentName, score)
	t.runtimeDir.IncrementTaskCount(agentName)

	return map[string]any{
		"recorded":   true,
		"agent_name": agentName,
		"task_id":    taskID,
		"score":      score,
		"reasoning":  reasoning,
		"timestamp":  time.Now().Unix(),
	}, nil
}
