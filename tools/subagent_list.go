package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// SubAgentListTool lists all SubAgent tasks.
type SubAgentListTool struct {
	accessor OrchestrationAccessor
}

// SetAccessor sets the orchestration accessor.
func (t *SubAgentListTool) SetAccessor(a OrchestrationAccessor) {
	t.accessor = a
}

// NewSubAgentListTool creates a new SubAgentListTool.
func NewSubAgentListTool() *SubAgentListTool {
	return &SubAgentListTool{}
}

func (t *SubAgentListTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "subagent_list",
		Description: "List all spawned SubAgent tasks and their statuses.",
		IsReadOnly:  true,
		Parameters: []core.Parameter{
			{Name: "parent_id", Type: "string", Description: "Optional: filter by parent task ID.", Required: false},
		},
	}
}

func (t *SubAgentListTool) Execute(_ context.Context, params map[string]any) (any, error) {
	parentID, _ := params["parent_id"].(string)
	if t.accessor == nil {
		return nil, fmt.Errorf("orchestration accessor not configured")
	}

	orch := t.accessor.Orchestrator()
	if orch == nil {
		return "No SubAgent tasks have been spawned yet (no orchestrator configured).", nil
	}

	tm := orch
	var tasks []*core.Task
	var err error
	tasks, err = tm.ListTasks(parentID)
	if err != nil {
		return "", fmt.Errorf("failed to list subagent tasks: %w", err)
	}

	var agentTasks []*core.Task
	for _, task := range tasks {
		if task.Metadata != nil && task.Metadata["subagent_name"] != nil {
			agentTasks = append(agentTasks, task)
		}
	}

	if len(agentTasks) == 0 {
		return "No SubAgent tasks have been spawned yet.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d SubAgent task(s):\n", len(agentTasks))
	for _, task := range agentTasks {
		agentName, _ := task.Metadata["subagent_name"].(string)
		agentModel, _ := task.Metadata["subagent_model"].(string)
		teamID, _ := task.Metadata["subagent_team_id"].(string)
		fmt.Fprintf(&sb, "  - %s | Agent: %s | Model: %s | Team: %s | Status: %s\n",
			task.ID, agentName, agentModel, teamID, task.Status)
		if task.Output != "" {
			output := task.Output
			if len(output) > 200 {
				output = output[:200] + "... [truncated]"
			}
			fmt.Fprintf(&sb, "    Output: %s\n", output)
		}
		if task.Error != "" {
			fmt.Fprintf(&sb, "    Error: %s\n", task.Error)
		}
	}
	return sb.String(), nil
}
