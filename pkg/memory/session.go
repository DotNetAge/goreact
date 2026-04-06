package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// SessionAccessor manages Session and Message nodes
type SessionAccessor struct {
	BaseAccessor
}

// NewSessionAccessor creates a new SessionAccessor
func NewSessionAccessor(graphRAG pattern.GraphRAGPattern) *SessionAccessor {
	return &SessionAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeSession,
		},
	}
}

// Get retrieves a session by name
func (a *SessionAccessor) Get(ctx context.Context, sessionName string) (*goreactcore.SessionNode, error) {
	node, err := a.BaseAccessor.Get(ctx, sessionName)
	if err != nil {
		return nil, err
	}
	return nodeToSessionNode(node), nil
}

// GetHistory retrieves session history with messages
func (a *SessionAccessor) GetHistory(ctx context.Context, sessionName string, opts ...SessionOption) (*SessionHistory, error) {
	// Parse options
	options := &SessionOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Get the session node
	sessionNode, err := a.Get(ctx, sessionName)
	if err != nil {
		return nil, err
	}

	// Query messages for this session
	query := fmt.Sprintf(
		"MATCH (s:%s {id: $sessionId})-[:HAS_MESSAGE]->(m:%s) RETURN m ORDER BY m.timestamp DESC",
		goreactcommon.NodeTypeSession, goreactcommon.NodeTypeMessage,
	)
	if options.MaxTurns > 0 {
		query = fmt.Sprintf(
			"MATCH (s:%s {id: $sessionId})-[:HAS_MESSAGE]->(m:%s) RETURN m ORDER BY m.timestamp DESC LIMIT %d",
			goreactcommon.NodeTypeSession, goreactcommon.NodeTypeMessage, options.MaxTurns,
		)
	}

	results, err := a.graphRAG.QueryGraph(ctx, query, map[string]any{"sessionId": sessionName})
	if err != nil {
		return nil, err
	}

	messages := make([]*goreactcore.MessageNode, 0, len(results))
	for _, result := range results {
		if msgData, ok := result["m"].(map[string]any); ok {
			msg := &goreactcore.MessageNode{
				SessionName: sessionName,
				Role:        msgData["role"].(string),
				Content:     msgData["content"].(string),
				Timestamp:   parseTime(msgData["timestamp"]),
			}
			if base, ok := msgData["base"].(map[string]any); ok {
				msg.Name = base["name"].(string)
				msg.NodeType = base["node_type"].(string)
			}
			messages = append(messages, msg)
		}
	}

	return &SessionHistory{
		SessionName: sessionName,
		Messages:    messages,
		TotalTurns:  len(messages),
		Session:     sessionNode,
	}, nil
}

// Create creates a new session
func (a *SessionAccessor) Create(ctx context.Context, userName string) (*goreactcore.SessionNode, error) {
	session := goreactcore.NewSessionNode(generateSessionID(), userName)

	node := &core.Node{
		ID:   session.Name,
		Type: goreactcommon.NodeTypeSession,
		Properties: map[string]any{
			"name":       session.Name,
			"node_type":  goreactcommon.NodeTypeSession,
			"user_name":  userName,
			"start_time": session.StartTime.Format(time.RFC3339),
			"status":     string(session.Status),
			"created_at": session.CreatedAt.Format(time.RFC3339),
		},
	}

	if err := a.graphRAG.AddNode(ctx, node); err != nil {
		return nil, err
	}

	return session, nil
}

// AddMessage adds a message to a session
func (a *SessionAccessor) AddMessage(ctx context.Context, sessionName string, message *goreactcore.MessageNode) (*goreactcore.MessageNode, error) {
	message.SessionName = sessionName
	message.Timestamp = time.Now()

	node := &core.Node{
		ID:   message.Name,
		Type: goreactcommon.NodeTypeMessage,
		Properties: map[string]any{
			"name":         message.Name,
			"node_type":    goreactcommon.NodeTypeMessage,
			"session_name": sessionName,
			"role":         message.Role,
			"content":      message.Content,
			"timestamp":    message.Timestamp.Format(time.RFC3339),
			"created_at":   time.Now().Format(time.RFC3339),
		},
	}

	if err := a.graphRAG.AddNode(ctx, node); err != nil {
		return nil, err
	}

	// Create edge from session to message
	edge := &core.Edge{
		ID:     fmt.Sprintf("session-%s-message-%s", sessionName, message.Name),
		Type:   "HAS_MESSAGE",
		Source: sessionName,
		Target: message.Name,
		Properties: map[string]any{
			"created_at": time.Now().Format(time.RFC3339),
		},
	}

	if err := a.graphRAG.AddEdge(ctx, edge); err != nil {
		return nil, err
	}

	return message, nil
}

// SessionHistory represents session history with messages
type SessionHistory struct {
	SessionName string
	Messages    []*goreactcore.MessageNode
	TotalTurns  int
	Session     *goreactcore.SessionNode
}

// SessionOption is a function that configures session options
type SessionOption func(*SessionOptions)

// SessionOptions contains options for session retrieval
type SessionOptions struct {
	MaxTurns           int
	IncludeReflections bool
	IncludePlans       bool
}

// WithMaxTurns sets the maximum number of turns to retrieve
func WithMaxTurns(maxTurns int) SessionOption {
	return func(o *SessionOptions) {
		o.MaxTurns = maxTurns
	}
}

// WithIncludeReflections sets whether to include reflections
func WithIncludeReflections(include bool) SessionOption {
	return func(o *SessionOptions) {
		o.IncludeReflections = include
	}
}

// WithIncludePlans sets whether to include plans
func WithIncludePlans(include bool) SessionOption {
	return func(o *SessionOptions) {
		o.IncludePlans = include
	}
}
