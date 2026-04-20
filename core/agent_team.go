package core

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MessageType identifies the type of inter-agent message.
type MessageType string

const (
	// MessageDirect is a point-to-point message from one agent to another.
	MessageDirect MessageType = "message"
	// MessageBroadcast is a message from one agent to all team members.
	MessageBroadcast MessageType = "broadcast"
	// MessageShutdownRequest asks an agent to terminate gracefully.
	MessageShutdownRequest MessageType = "shutdown_request"
	// MessageShutdownResponse confirms an agent has shut down.
	MessageShutdownResponse MessageType = "shutdown_response"
)

// AgentMessage is a message sent between agents within a team.
type AgentMessage struct {
	ID        string      `json:"id"`
	From      string      `json:"from"`    // sender agent name
	To        string      `json:"to"`      // recipient agent name (empty for broadcast)
	Type      MessageType `json:"type"`
	Content   string      `json:"content"`
	Summary   string      `json:"summary"` // brief summary for context efficiency
	Timestamp int64       `json:"timestamp"`
}

// TeamMember represents an agent that is part of a team.
type TeamMember struct {
	Name   string `json:"name"`
	TaskID string `json:"task_id"`
	Status string `json:"status"` // "running", "completed", "failed"
	Result string `json:"result,omitempty"`
}

// AgentTeam represents a team of collaborating agents.
type AgentTeam struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Members     map[string]*TeamMember `json:"members"` // agent_name -> member
	CreatedAt   int64                  `json:"created_at"`
}

// AgentMessageBus manages teams and inter-agent message delivery.
// It provides channel-based messaging between agents, enabling
// true team collaboration where agents can send/receive messages
// while running in their own independent goroutines.
type AgentMessageBus struct {
	teams    map[string]*AgentTeam
	mailboxes map[string]chan *AgentMessage // agent_name -> mailbox channel
	mu       sync.RWMutex
	nextID   atomic.Int64
}

// NewAgentMessageBus creates a new message bus.
func NewAgentMessageBus() *AgentMessageBus {
	return &AgentMessageBus{
		teams:     make(map[string]*AgentTeam),
		mailboxes: make(map[string]chan *AgentMessage),
	}
}

// CreateTeam creates a new team and returns its ID.
func (b *AgentMessageBus) CreateTeam(name, description string) (*AgentTeam, error) {
	if name == "" {
		return nil, fmt.Errorf("team name must not be empty")
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	id := fmt.Sprintf("team_%d", b.nextID.Add(1))
	team := &AgentTeam{
		ID:          id,
		Name:        name,
		Description: description,
		Members:     make(map[string]*TeamMember),
		CreatedAt:   time.Now().UnixMilli(),
	}
	b.teams[id] = team
	return team, nil
}

// DeleteTeam removes a team and cleans up all member mailboxes.
func (b *AgentMessageBus) DeleteTeam(teamID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	team, ok := b.teams[teamID]
	if !ok {
		return fmt.Errorf("team %q not found", teamID)
	}

	// Close and remove mailboxes for all team members
	for name := range team.Members {
		if ch, exists := b.mailboxes[name]; exists {
			close(ch)
			delete(b.mailboxes, name)
		}
	}

	delete(b.teams, teamID)
	return nil
}

// GetTeam returns a team by ID.
func (b *AgentMessageBus) GetTeam(teamID string) (*AgentTeam, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	team, ok := b.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team %q not found", teamID)
	}
	return team, nil
}

// ListTeams returns all teams.
func (b *AgentMessageBus) ListTeams() []*AgentTeam {
	b.mu.RLock()
	defer b.mu.RUnlock()

	teams := make([]*AgentTeam, 0, len(b.teams))
	for _, t := range b.teams {
		teams = append(teams, t)
	}
	return teams
}

// JoinTeam adds an agent as a member of a team and creates its mailbox.
func (b *AgentMessageBus) JoinTeam(teamID, agentName, taskID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	team, ok := b.teams[teamID]
	if !ok {
		return fmt.Errorf("team %q not found", teamID)
	}

	team.Members[agentName] = &TeamMember{
		Name:   agentName,
		TaskID: taskID,
		Status: "running",
	}

	// Create mailbox if not already exists (buffered for async delivery)
	if _, exists := b.mailboxes[agentName]; !exists {
		b.mailboxes[agentName] = make(chan *AgentMessage, 64)
	}

	return nil
}

// LeaveTeam removes an agent from a team.
func (b *AgentMessageBus) LeaveTeam(teamID, agentName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	team, ok := b.teams[teamID]
	if !ok {
		return fmt.Errorf("team %q not found", teamID)
	}

	delete(team.Members, agentName)
	// Note: don't close mailbox here — agent may still be reading pending messages
	return nil
}

// SendMessage delivers a message to an agent's mailbox.
// For broadcast messages (To is empty), the message is delivered to all team members except the sender.
func (b *AgentMessageBus) SendMessage(teamID, from, to string, msgType MessageType, content, summary string) (*AgentMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	team, ok := b.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team %q not found", teamID)
	}

	msg := &AgentMessage{
		ID:        fmt.Sprintf("msg_%d", b.nextID.Add(1)),
		From:      from,
		To:        to,
		Type:      msgType,
		Content:   content,
		Summary:   summary,
		Timestamp: time.Now().UnixMilli(),
	}

	if to == "" {
		// Broadcast to all team members except sender
		for memberName := range team.Members {
			if memberName != from {
				if ch, exists := b.mailboxes[memberName]; exists {
					select {
					case ch <- msg:
					default:
						// mailbox full, skip (non-blocking)
					}
				}
			}
		}
	} else {
		// Direct message to specific agent
		ch, exists := b.mailboxes[to]
		if !exists {
			return nil, fmt.Errorf("agent %q has no mailbox (not in a team)", to)
		}
		select {
		case ch <- msg:
		default:
			return nil, fmt.Errorf("agent %q mailbox is full", to)
		}
	}

	return msg, nil
}

// ReceiveMessages reads all pending messages from an agent's mailbox.
// Returns immediately with whatever messages are available (non-blocking).
func (b *AgentMessageBus) ReceiveMessages(agentName string) []*AgentMessage {
	b.mu.RLock()
	ch, exists := b.mailboxes[agentName]
	b.mu.RUnlock()

	if !exists {
		return nil
	}

	var messages []*AgentMessage
	for {
		select {
		case msg := <-ch:
			messages = append(messages, msg)
		default:
			return messages
		}
	}
}

// UpdateMemberStatus updates the status of a team member.
func (b *AgentMessageBus) UpdateMemberStatus(teamID, agentName, status, result string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	team, ok := b.teams[teamID]
	if !ok {
		return fmt.Errorf("team %q not found", teamID)
	}

	member, exists := team.Members[agentName]
	if !exists {
		return fmt.Errorf("agent %q not found in team %q", agentName, teamID)
	}

	member.Status = status
	member.Result = result
	return nil
}

// WaitMailbox returns the mailbox channel for an agent, allowing
// the agent to block on incoming messages via channel receive.
func (b *AgentMessageBus) WaitMailbox(agentName string) (<-chan *AgentMessage, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	ch, exists := b.mailboxes[agentName]
	if !exists {
		return nil, fmt.Errorf("agent %q has no mailbox", agentName)
	}
	return ch, nil
}

// TeamMemberCount returns the number of members in a team.
func (b *AgentMessageBus) TeamMemberCount(teamID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	team, ok := b.teams[teamID]
	if !ok {
		return 0
	}
	return len(team.Members)
}
