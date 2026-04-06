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

// FrozenSessionAccessor manages FrozenSession nodes
type FrozenSessionAccessor struct {
	BaseAccessor
}

// NewFrozenSessionAccessor creates a new FrozenSessionAccessor
func NewFrozenSessionAccessor(graphRAG pattern.GraphRAGPattern) *FrozenSessionAccessor {
	return &FrozenSessionAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeFrozenSession,
		},
	}
}

// Get retrieves a frozen session
func (a *FrozenSessionAccessor) Get(ctx context.Context, sessionName string) (*goreactcore.FrozenSessionNode, error) {
	node, err := a.BaseAccessor.Get(ctx, sessionName)
	if err != nil {
		return nil, err
	}
	return nodeToFrozenSessionNode(node), nil
}

// List lists frozen sessions
func (a *FrozenSessionAccessor) List(ctx context.Context, opts ...ListOption) ([]*goreactcore.FrozenSessionNode, error) {
	nodes, err := a.BaseAccessor.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	sessions := make([]*goreactcore.FrozenSessionNode, 0, len(nodes))
	for _, node := range nodes {
		sessions = append(sessions, nodeToFrozenSessionNode(node))
	}

	return sessions, nil
}

// Freeze freezes a session with a pending question
func (a *FrozenSessionAccessor) Freeze(ctx context.Context, sessionName string, question *goreactcore.PendingQuestionNode) (*goreactcore.FrozenSessionNode, error) {
	// Get session to retrieve user and agent information
	session, err := a.graphRAG.GetNode(ctx, sessionName)
	if err != nil {
		// Continue without session info if not found
		session = nil
	}

	userName := ""
	agentName := ""
	if session != nil {
		if un, ok := session.Properties["user_name"].(string); ok {
			userName = un
		}
		if an, ok := session.Properties["agent_name"].(string); ok {
			agentName = an
		}
	}

	frozen := &goreactcore.FrozenSessionNode{
		BaseNode: goreactcore.BaseNode{
			Name:        sessionName,
			NodeType:    goreactcommon.NodeTypeFrozenSession,
			Description: question.Question,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		SessionName:   sessionName,
		QuestionID:    question.Name,
		Status:        goreactcommon.FrozenStatusFrozen,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour), // Default 24 hour expiry
		SuspendReason: question.Question,
		UserName:      userName,
		AgentName:     agentName,
		Priority:      goreactcommon.TaskPriorityNormal, // Default priority
	}

	node := &core.Node{
		ID:   frozen.Name,
		Type: goreactcommon.NodeTypeFrozenSession,
		Properties: map[string]any{
			"name":           frozen.Name,
			"node_type":      goreactcommon.NodeTypeFrozenSession,
			"session_name":   sessionName,
			"question_id":    question.Name,
			"state_data":     frozen.StateData,
			"created_at":     frozen.CreatedAt.Format(time.RFC3339),
			"expires_at":     frozen.ExpiresAt.Format(time.RFC3339),
			"status":         string(frozen.Status),
			"suspend_reason": frozen.SuspendReason,
			"user_name":      frozen.UserName,
			"agent_name":     frozen.AgentName,
			"priority":       string(frozen.Priority),
		},
	}

	if err := a.graphRAG.AddNode(ctx, node); err != nil {
		return nil, err
	}

	return frozen, nil
}

// Resume resumes a frozen session
func (a *FrozenSessionAccessor) Resume(ctx context.Context, sessionName string, answer string) error {
	frozen, err := a.Get(ctx, sessionName)
	if err != nil {
		return err
	}

	frozen.Status = goreactcommon.FrozenStatusResumed

	node := &core.Node{
		ID:   frozen.Name,
		Type: goreactcommon.NodeTypeFrozenSession,
		Properties: map[string]any{
			"name":           frozen.Name,
			"node_type":      goreactcommon.NodeTypeFrozenSession,
			"session_name":   sessionName,
			"question_id":    frozen.QuestionID,
			"state_data":     frozen.StateData,
			"created_at":     frozen.CreatedAt.Format(time.RFC3339),
			"expires_at":     frozen.ExpiresAt.Format(time.RFC3339),
			"status":         string(frozen.Status),
			"suspend_reason": frozen.SuspendReason,
			"answer":         answer,
			"answered_at":    time.Now().Format(time.RFC3339),
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// Cancel cancels a frozen session
func (a *FrozenSessionAccessor) Cancel(ctx context.Context, sessionName string) error {
	frozen, err := a.Get(ctx, sessionName)
	if err != nil {
		return err
	}

	frozen.Status = goreactcommon.FrozenStatusExpired

	return a.Delete(ctx, sessionName)
}

// Answer answers a pending question
func (a *FrozenSessionAccessor) Answer(ctx context.Context, questionID string, answer string) error {
	// Find the frozen session with this question ID
	query := fmt.Sprintf(
		"MATCH (n:%s {question_id: $questionId}) RETURN n",
		goreactcommon.NodeTypeFrozenSession,
	)

	results, err := a.graphRAG.QueryGraph(ctx, query, map[string]any{"questionId": questionID})
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("frozen session not found for question ID: %s", questionID)
	}

	// Get the session name
	if nData, ok := results[0]["n"].(map[string]any); ok {
		if sessionName, ok := nData["session_name"].(string); ok {
			return a.Resume(ctx, sessionName, answer)
		}
	}

	return fmt.Errorf("failed to get session name from frozen session")
}
