package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// --- Team Tools (Multi-Agent Collaboration) ---

// TeamCreateTool creates a team for multi-agent collaboration.
type TeamCreateTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *TeamCreateTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewTeamCreateTool creates a new TeamCreateTool.
func NewTeamCreateTool() *TeamCreateTool {
	return &TeamCreateTool{}
}

func (t *TeamCreateTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "team_create",
		Description: "Create a new team for multi-agent collaboration. After creating a team, spawn SubAgents with the team_name parameter to have them join the team automatically.",
		Parameters: []core.Parameter{
			{Name: "name", Type: "string", Description: "Team name (lowercase, hyphens allowed, e.g. 'refactor-team').", Required: true},
			{Name: "description", Type: "string", Description: "Brief description of the team's purpose and goals.", Required: true},
		},
	}
}

func (t *TeamCreateTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	name, ok1 := params["name"].(string)
	description, ok2 := params["description"].(string)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("missing required parameters: name, description")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	bus := t.accessor.MessageBus()
	if bus == nil {
		return "", fmt.Errorf("message bus not configured on this reactor")
	}

	team, err := bus.CreateTeam(name, description)
	if err != nil {
		return "", fmt.Errorf("failed to create team: %w", err)
	}

	return fmt.Sprintf("Team %q created (ID: %s).\nSpawn SubAgents with team_name=%q to add them to this team.\nUse 'team_status' to monitor progress, 'send_message' for inter-agent communication.", name, team.ID, team.ID), nil
}

// --- Send Message Tool ---

// SendMessageTool allows agents to send messages to each other within a team.
type SendMessageTool struct {
	accessor   ReactorAccessor
	agentName  string
	teamID     string
}

// SetAccessor sets the reactor accessor.
func (t *SendMessageTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// SetAgentIdentity sets the sending agent's name and team.
func (t *SendMessageTool) SetAgentIdentity(name, teamID string) {
	t.agentName = name
	t.teamID = teamID
}

// NewSendMessageTool creates a new SendMessageTool.
func NewSendMessageTool() *SendMessageTool {
	return &SendMessageTool{}
}

func (t *SendMessageTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name: "send_message",
		Description: `Send a message to another agent or broadcast to all team members.
Types:
- "message": direct message to a specific agent (provide 'to' parameter).
- "broadcast": send to all team members (omit 'to' parameter).
- "shutdown_request": ask an agent to terminate gracefully.
- "shutdown_response": respond to a shutdown request (provide 'to').`,
		Parameters: []core.Parameter{
			{Name: "type", Type: "string", Description: "Message type: 'message', 'broadcast', 'shutdown_request', or 'shutdown_response'.", Required: true},
			{Name: "to", Type: "string", Description: "Recipient agent name (required for 'message' and 'shutdown_request'). Use 'main' for the lead agent.", Required: false},
			{Name: "content", Type: "string", Description: "The message content to send.", Required: true},
			{Name: "summary", Type: "string", Description: "A brief 5-10 word summary of the message.", Required: true},
		},
	}
}

func (t *SendMessageTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	msgType, ok1 := params["type"].(string)
	content, ok2 := params["content"].(string)
	summary, ok3 := params["summary"].(string)
	if !ok1 || !ok2 || !ok3 {
		return "", fmt.Errorf("missing required parameters: type, content, summary")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	to, _ := params["to"].(string)
	bus := t.accessor.MessageBus()
	if bus == nil {
		return "", fmt.Errorf("message bus not configured")
	}
	if t.teamID == "" {
		return "", fmt.Errorf("this agent is not part of any team")
	}

	from := t.agentName
	if from == "" {
		from = "main"
	}

	msg, err := bus.SendMessage(t.teamID, from, to, core.MessageType(msgType), content, summary)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	if to != "" {
		return fmt.Sprintf("Message sent to %q (type: %s, msg_id: %s).", to, msgType, msg.ID), nil
	}
	return fmt.Sprintf("Broadcast sent to team (type: %s, msg_id: %s).", msgType, msg.ID), nil
}

// --- Receive Messages Tool ---

// ReceiveMessagesTool allows an agent to read its pending messages.
type ReceiveMessagesTool struct {
	accessor   ReactorAccessor
	agentName string
}

// SetAccessor sets the reactor accessor.
func (t *ReceiveMessagesTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// SetAgentIdentity sets the receiving agent's name.
func (t *ReceiveMessagesTool) SetAgentIdentity(name string) {
	t.agentName = name
}

// NewReceiveMessagesTool creates a new ReceiveMessagesTool.
func NewReceiveMessagesTool() *ReceiveMessagesTool {
	return &ReceiveMessagesTool{}
}

func (t *ReceiveMessagesTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "receive_messages",
		Description: "Read all pending messages from this agent's mailbox. Returns immediately with whatever messages are available.",
		IsReadOnly:  true,
		Parameters: []core.Parameter{
			{Name: "wait_seconds", Type: "integer", Description: "Optional: block up to N seconds waiting for at least one message (default: 0, non-blocking).", Required: false},
		},
	}
}

func (t *ReceiveMessagesTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	bus := t.accessor.MessageBus()
	if bus == nil {
		return "", fmt.Errorf("message bus not configured")
	}

	agent := t.agentName
	if agent == "" {
		agent = "main"
	}

	waitSec := 0
	if raw, ok := params["wait_seconds"]; ok {
		if v, ok := ToFloat64(raw); ok {
			waitSec = int(v)
		}
	}

	if waitSec > 0 {
		mailbox, err := bus.WaitMailbox(agent)
		if err != nil {
			return "No messages.", nil
		}
		select {
		case msg := <-mailbox:
			return formatMessages([]*core.AgentMessage{msg}), nil
		case <-time.After(time.Duration(waitSec) * time.Second):
		}
	}

	messages := bus.ReceiveMessages(agent)
	if len(messages) == 0 {
		return "No new messages.", nil
	}
	return formatMessages(messages), nil
}

func formatMessages(messages []*core.AgentMessage) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Received %d message(s):\n", len(messages))
	for _, msg := range messages {
		fmt.Fprintf(&sb, "  [%s] from=%s to=%s (type: %s, id: %s)\n", msg.Summary, msg.From, msg.To, msg.Type, msg.ID)
		fmt.Fprintf(&sb, "    %s\n", msg.Content)
	}
	return sb.String()
}

// --- Team Status Tool ---

// TeamStatusTool shows the current status of a team and its members.
type TeamStatusTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *TeamStatusTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewTeamStatusTool creates a new TeamStatusTool.
func NewTeamStatusTool() *TeamStatusTool {
	return &TeamStatusTool{}
}

func (t *TeamStatusTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "team_status",
		Description: "Show the current status of a team and all its members.",
		IsReadOnly:  true,
		Parameters: []core.Parameter{
			{Name: "team_id", Type: "string", Description: "The team ID to check status for.", Required: true},
		},
	}
}

func (t *TeamStatusTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	teamID, ok := params["team_id"].(string)
	if !ok || teamID == "" {
		return "", fmt.Errorf("missing required parameter: team_id")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	bus := t.accessor.MessageBus()
	if bus == nil {
		return "", fmt.Errorf("message bus not configured")
	}

	team, err := bus.GetTeam(teamID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Team %q (ID: %s)\n", team.Name, team.ID)
	if team.Description != "" {
		fmt.Fprintf(&sb, "  Description: %s\n", team.Description)
	}
	fmt.Fprintf(&sb, "  Members: %d\n", len(team.Members))

	running, completed, failed := 0, 0, 0
	for _, member := range team.Members {
		fmt.Fprintf(&sb, "    - %s (task: %s) [status: %s]\n", member.Name, member.TaskID, member.Status)
		switch member.Status {
		case "running":
			running++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}
	fmt.Fprintf(&sb, "  Summary: %d running, %d completed, %d failed\n", running, completed, failed)
	return sb.String(), nil
}

// --- Team Delete Tool ---

// TeamDeleteTool deletes a team and cleans up all resources.
type TeamDeleteTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *TeamDeleteTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewTeamDeleteTool creates a new TeamDeleteTool.
func NewTeamDeleteTool() *TeamDeleteTool {
	return &TeamDeleteTool{}
}

func (t *TeamDeleteTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "team_delete",
		Description: "Delete a team and clean up all member mailboxes. Only call this after all team members have completed their work.",
		Parameters: []core.Parameter{
			{Name: "team_id", Type: "string", Description: "The team ID to delete.", Required: true},
		},
	}
}

func (t *TeamDeleteTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	teamID, ok := params["team_id"].(string)
	if !ok || teamID == "" {
		return "", fmt.Errorf("missing required parameter: team_id")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	bus := t.accessor.MessageBus()
	if bus == nil {
		return "", fmt.Errorf("message bus not configured")
	}

	if err := bus.DeleteTeam(teamID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Team %q deleted. All member mailboxes cleaned up.", teamID), nil
}

// --- Wait Team Tool ---

// WaitTeamTool blocks until all team members have completed.
type WaitTeamTool struct {
	accessor ReactorAccessor
}

// SetAccessor sets the reactor accessor.
func (t *WaitTeamTool) SetAccessor(a ReactorAccessor) {
	t.accessor = a
}

// NewWaitTeamTool creates a new WaitTeamTool.
func NewWaitTeamTool() *WaitTeamTool {
	return &WaitTeamTool{}
}

func (t *WaitTeamTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "wait_team",
		Description: `Wait for all team members to complete and collect their results. This is the primary tool for the lead agent to gather team output for final synthesis. The tool blocks until all members have status "completed" or "failed", or the timeout expires.`,
		Parameters: []core.Parameter{
			{Name: "team_id", Type: "string", Description: "The team ID to wait for.", Required: true},
			{Name: "timeout_seconds", Type: "integer", Description: "Maximum time to wait in seconds (default: 300).", Required: false},
		},
	}
}

func (t *WaitTeamTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	teamID, ok := params["team_id"].(string)
	if !ok || teamID == "" {
		return "", fmt.Errorf("missing required parameter: team_id")
	}
	if t.accessor == nil {
		return nil, fmt.Errorf("reactor accessor not configured")
	}

	bus := t.accessor.MessageBus()
	if bus == nil {
		return "", fmt.Errorf("message bus not configured")
	}

	timeout := 300 * time.Second
	if raw, ok := params["timeout_seconds"]; ok {
		if v, ok := ToFloat64(raw); ok && v > 0 {
			timeout = time.Duration(v) * time.Second
		}
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		team, err := bus.GetTeam(teamID)
		if err != nil {
			return "", err
		}

		allDone := true
		for _, member := range team.Members {
			if member.Status == "running" {
				allDone = false
				break
			}
		}

		if allDone {
			return buildTeamResult(team), nil
		}

		time.Sleep(2 * time.Second)
	}

	team, _ := bus.GetTeam(teamID)
	if team == nil {
		return "", fmt.Errorf("team %q not found after wait", teamID)
	}
	return buildTeamResult(team) + "\n\n[WARNING: timeout reached, some members may still be running]", nil
}

func buildTeamResult(team *core.AgentTeam) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Team %q — All members finished.\n\n", team.Name)

	for _, member := range team.Members {
		fmt.Fprintf(&sb, "=== %s (status: %s) ===\n", member.Name, member.Status)
		if member.Result != "" {
			result := member.Result
			if len(result) > 500 {
				result = result[:500] + "... [truncated]"
			}
			fmt.Fprintf(&sb, "%s\n\n", result)
		} else {
			fmt.Fprintf(&sb, "(no result)\n\n")
		}
	}
	return sb.String()
}
