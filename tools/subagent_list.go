package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// SubAgentListTool lists all SubAgent tasks.
type SubAgentListTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *SubAgentListTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewSubAgentListTool creates a new SubAgentListTool.
func NewSubAgentListTool() *SubAgentListTool {
	return &SubAgentListTool{}
}

func (t *SubAgentListTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "subagent_list",
		Description: "List all spawned SubAgents and their statuses. For team-mode agents, prefer 'team_status'.",
		IsReadOnly:  true,
		Parameters: []core.Parameter{
			{Name: "parent_id", Type: "string", Description: "Optional: filter by parent task ID.", Required: false},
		},
	}
}

func (t *SubAgentListTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	parentID, _ := params["parent_id"].(string)
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	tm := t.accessor.TaskManager()
	var tasks []*core.Task
	var err error
	if parentID != "" {
		tasks, err = tm.ListSubTasks(parentID)
	} else {
		tasks, err = tm.ListAllTasks()
	}
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
