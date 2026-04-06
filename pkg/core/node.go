// Package core provides core interfaces and types for the goreact framework.
package core

import (
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Node represents a node in the memory graph
type Node interface {
	// GetName returns the unique identifier of the node
	GetName() string
	
	// GetNodeType returns the type of the node
	GetNodeType() string
	
	// GetDescription returns the description of the node
	GetDescription() string
	
	// GetCreatedAt returns the creation timestamp
	GetCreatedAt() time.Time
	
	// GetMetadata returns the node metadata
	GetMetadata() map[string]any
}

// BaseNode provides a base implementation for nodes
type BaseNode struct {
	Name        string         `json:"name" yaml:"name"`
	NodeType    string         `json:"node_type" yaml:"node_type"`
	Description string         `json:"description" yaml:"description"`
	CreatedAt   time.Time      `json:"created_at" yaml:"created_at"`
	Metadata    map[string]any `json:"metadata" yaml:"metadata"`
}

// GetName returns the unique identifier of the node
func (n *BaseNode) GetName() string {
	return n.Name
}

// GetNodeType returns the type of the node
func (n *BaseNode) GetNodeType() string {
	return n.NodeType
}

// GetDescription returns the description of the node
func (n *BaseNode) GetDescription() string {
	return n.Description
}

// GetCreatedAt returns the creation timestamp
func (n *BaseNode) GetCreatedAt() time.Time {
	return n.CreatedAt
}

// GetMetadata returns the node metadata
func (n *BaseNode) GetMetadata() map[string]any {
	return n.Metadata
}

// AgentNode represents an Agent in the memory graph
type AgentNode struct {
	BaseNode
	Domain       string   `json:"domain" yaml:"domain"`
	Model        string   `json:"model" yaml:"model"`
	Skills       []string `json:"skills" yaml:"skills"`
	Tools        []string `json:"tools" yaml:"tools"`
	PromptTemplate string `json:"prompt_template" yaml:"prompt_template"`
}

// NewAgentNode creates a new AgentNode
func NewAgentNode(name, description, domain, model string) *AgentNode {
	return &AgentNode{
		BaseNode: BaseNode{
			Name:        name,
			NodeType:    common.NodeTypeAgent,
			Description: description,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		Domain: domain,
		Model:  model,
		Skills: []string{},
		Tools:  []string{},
	}
}

// ModelNode represents a Model in the memory graph
type ModelNode struct {
	BaseNode
	Provider string         `json:"provider" yaml:"provider"`
	Config   map[string]any `json:"config" yaml:"config"`
}

// NewModelNode creates a new ModelNode
func NewModelNode(name, description, provider string) *ModelNode {
	return &ModelNode{
		BaseNode: BaseNode{
			Name:        name,
			NodeType:    common.NodeTypeModel,
			Description: description,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		Provider: provider,
		Config:   make(map[string]any),
	}
}

// SessionNode represents a Session in the memory graph
type SessionNode struct {
	BaseNode
	UserName  string                 `json:"user_name" yaml:"user_name"`
	StartTime time.Time              `json:"start_time" yaml:"start_time"`
	EndTime   time.Time              `json:"end_time" yaml:"end_time"`
	Status    common.SessionStatus   `json:"status" yaml:"status"`
}

// NewSessionNode creates a new SessionNode
func NewSessionNode(name, userName string) *SessionNode {
	return &SessionNode{
		BaseNode: BaseNode{
			Name:        name,
			NodeType:    common.NodeTypeSession,
			Description: "",
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		UserName:  userName,
		StartTime: time.Now(),
		Status:    common.SessionStatusActive,
	}
}

// MessageNode represents a Message in the memory graph
type MessageNode struct {
	BaseNode
	SessionName string            `json:"session_name" yaml:"session_name"`
	Role        string            `json:"role" yaml:"role"`
	Content     string            `json:"content" yaml:"content"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	TokenUsage  *common.TokenUsage `json:"token_usage" yaml:"token_usage"`
}

// NewMessageNode creates a new MessageNode
func NewMessageNode(sessionName, role, content string) *MessageNode {
	return &MessageNode{
		BaseNode: BaseNode{
			Name:        generateMessageID(),
			NodeType:    common.NodeTypeMessage,
			Description: "",
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		SessionName: sessionName,
		Role:        role,
		Content:     content,
		Timestamp:   time.Now(),
	}
}

// MemoryItemNode represents a MemoryItem in the memory graph
type MemoryItemNode struct {
	BaseNode
	SessionName   string                  `json:"session_name" yaml:"session_name"`
	Content       string                  `json:"content" yaml:"content"`
	Type          common.MemoryItemType   `json:"type" yaml:"type"`
	Source        common.MemorySource     `json:"source" yaml:"source"`
	Importance    float64                 `json:"importance" yaml:"importance"`
	EmphasisLevel common.EmphasisLevel    `json:"emphasis_level" yaml:"emphasis_level"`
}

// NewMemoryItemNode creates a new MemoryItemNode
func NewMemoryItemNode(sessionName, content string, memType common.MemoryItemType) *MemoryItemNode {
	return &MemoryItemNode{
		BaseNode: BaseNode{
			Name:        generateMemoryItemID(),
			NodeType:    common.NodeTypeMemoryItem,
			Description: content,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		SessionName:   sessionName,
		Content:       content,
		Type:          memType,
		Source:        common.MemorySourceUser,
		Importance:    0.5,
		EmphasisLevel: common.EmphasisLevelNormal,
	}
}

// FrozenSessionNode represents a FrozenSession in the memory graph
type FrozenSessionNode struct {
	BaseNode
	SessionName   string                `json:"session_name" yaml:"session_name"`
	QuestionID    string                `json:"question_id" yaml:"question_id"`
	StateData     []byte                `json:"state_data" yaml:"state_data"`
	CreatedAt     time.Time             `json:"created_at" yaml:"created_at"`
	ExpiresAt     time.Time             `json:"expires_at" yaml:"expires_at"`
	Status        common.FrozenStatus   `json:"status" yaml:"status"`
	UserName      string                `json:"user_name" yaml:"user_name"`
	AgentName     string                `json:"agent_name" yaml:"agent_name"`
	SuspendReason string                `json:"suspend_reason" yaml:"suspend_reason"`
}

// PendingQuestionNode represents a PendingQuestion in the memory graph
type PendingQuestionNode struct {
	BaseNode
	SessionName   string              `json:"session_name" yaml:"session_name"`
	Type          common.QuestionType `json:"type" yaml:"type"`
	Question      string              `json:"question" yaml:"question"`
	Options       []string            `json:"options" yaml:"options"`
	DefaultAnswer string              `json:"default_answer" yaml:"default_answer"`
	// Additional fields for design compliance
	RelatedAction *Action             `json:"related_action" yaml:"related_action"`
	ExpiresAt     time.Time           `json:"expires_at" yaml:"expires_at"`
	QuestionStatus common.QuestionStatus `json:"question_status" yaml:"question_status"`
	Answer        string              `json:"answer" yaml:"answer"`
}

// NewPendingQuestionNode creates a new PendingQuestionNode
func NewPendingQuestionNode(sessionName string, qType common.QuestionType, question string) *PendingQuestionNode {
	return &PendingQuestionNode{
		BaseNode: BaseNode{
			Name:        generateQuestionID(),
			NodeType:    common.NodeTypePendingQuestion,
			Description: question,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		SessionName:    sessionName,
		Type:           qType,
		Question:       question,
		Options:        []string{},
		QuestionStatus: common.QuestionStatusPending,
	}
}

// WithOptions sets the options
func (p *PendingQuestionNode) WithOptions(options []string) *PendingQuestionNode {
	p.Options = options
	return p
}

// WithDefaultAnswer sets the default answer
func (p *PendingQuestionNode) WithDefaultAnswer(answer string) *PendingQuestionNode {
	p.DefaultAnswer = answer
	return p
}

// WithContext adds context to metadata
func (p *PendingQuestionNode) WithContext(key string, value any) *PendingQuestionNode {
	if p.Metadata == nil {
		p.Metadata = make(map[string]any)
	}
	p.Metadata[key] = value
	return p
}

// WithRelatedAction sets the related action
func (p *PendingQuestionNode) WithRelatedAction(action *Action) *PendingQuestionNode {
	p.RelatedAction = action
	return p
}

// WithExpiry sets the expiration time
func (p *PendingQuestionNode) WithExpiry(duration time.Duration) *PendingQuestionNode {
	p.ExpiresAt = time.Now().Add(duration)
	return p
}

// SetAnswer sets the answer
func (p *PendingQuestionNode) SetAnswer(answer string) {
	p.Answer = answer
	p.QuestionStatus = common.QuestionStatusAnswered
}

// IsExpired checks if the question is expired
func (p *PendingQuestionNode) IsExpired() bool {
	if p.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(p.ExpiresAt)
}

// Cancel cancels the question
func (p *PendingQuestionNode) Cancel() {
	p.QuestionStatus = common.QuestionStatusCancelled
}

// IsAnswered checks if the question is answered
func (p *PendingQuestionNode) IsAnswered() bool {
	return p.QuestionStatus == common.QuestionStatusAnswered
}

// IsPending checks if the question is pending
func (p *PendingQuestionNode) IsPending() bool {
	return p.QuestionStatus == common.QuestionStatusPending && !p.IsExpired()
}

// Helper functions to generate IDs
func generateMessageID() string {
	return "msg-" + generateID()
}

func generateMemoryItemID() string {
	return "mem-" + generateID()
}

func generateQuestionID() string {
	return "q-" + generateID()
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
