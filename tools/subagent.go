package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// --- SubAgent Result Tool ---

// SubAgentResultTool retrieves the result of an async subagent.
type SubAgentResultTool struct {
	accessor OrchestrationAccessor
}

// SetAccessor sets the orchestration accessor.
func (t *SubAgentResultTool) SetAccessor(a OrchestrationAccessor) {
	t.accessor = a
}

// NewSubAgentResultTool creates a new SubAgentResultTool.
func NewSubAgentResultTool() *SubAgentResultTool {
	return &SubAgentResultTool{}
}

func (t *SubAgentResultTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "subagent_result",
		Description: "Retrieve the result of a specific SubAgent by task ID. Blocks until the SubAgent completes or times out.",
		Tags:        []string{"agent", "subagent", "result", "orchestration"},
		Parameters: []core.Parameter{
			{Name: "task_id", Type: "string", Description: "The SubAgent's task ID.", Required: true},
			{Name: "wait_seconds", Type: "integer", Description: "How long to wait in seconds (default: 60).", Required: false},
		},
	}
}

func (t *SubAgentResultTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	taskID, ok := params["task_id"].(string)
	if !ok || taskID == "" {
		return "", fmt.Errorf("missing required parameter: task_id")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("orchestration accessor not configured")
	}

	orch := t.accessor.Orchestrator()
	if orch == nil {
		return nil, fmt.Errorf("no orchestrator configured")
	}

	waitSeconds := 60
	if raw, ok := params["wait_seconds"]; ok {
		if v, ok := ToFloat64(raw); ok {
			waitSeconds = int(v)
		}
	}

	// Apply timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(waitSeconds)*time.Second)
	defer cancel()

	task, err := orch.WaitForResult(ctxWithTimeout, taskID)
	if err != nil {
		return nil, fmt.Errorf("subagent task %q wait failed: %w", taskID, err)
	}
	if task == nil {
		return nil, fmt.Errorf("subagent task %q not found", taskID)
	}
	if task == nil {
		return fmt.Sprintf("SubAgent Task %q not found", taskID), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "SubAgent Task %q\n", taskID)
	fmt.Fprintf(&sb, "  Status:      %s\n", task.Status)
	fmt.Fprintf(&sb, "  Description: %s\n", task.Description)
	if task.Output != "" {
		fmt.Fprintf(&sb, "  Output:      %s\n", task.Output)
	}
	if task.Error != "" {
		fmt.Fprintf(&sb, "  Error:       %s\n", task.Error)
	}
	return sb.String(), nil
}
