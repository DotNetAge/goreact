package core

import (
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Thought represents a thinking result from the Thinker
type Thought struct {
	// Content is the complete expression of the thinking
	Content string `json:"content" yaml:"content"`
	
	// Reasoning is the reasoning process and logical chain
	Reasoning string `json:"reasoning" yaml:"reasoning"`
	
	// Decision is the decision conclusion (act/answer)
	Decision string `json:"decision" yaml:"decision"`
	
	// Confidence is the decision confidence (0.0-1.0)
	Confidence float64 `json:"confidence" yaml:"confidence"`
	
	// Action is the action intent if decision is to act
	Action *ActionIntent `json:"action" yaml:"action"`
	
	// FinalAnswer is the final answer if decision is to answer
	FinalAnswer string `json:"final_answer" yaml:"final_answer"`
	
	// Timestamp is the thinking timestamp
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// ActionIntent represents an intent to perform an action
type ActionIntent struct {
	// Type is the action type
	Type string `json:"type" yaml:"type"`
	
	// Target is the target name (tool/skill/agent name)
	Target string `json:"target" yaml:"target"`
	
	// Params are the action parameters
	Params map[string]any `json:"params" yaml:"params"`
	
	// Reasoning is why this action was chosen
	Reasoning string `json:"reasoning" yaml:"reasoning"`
}

// IntentResult represents the result of intent classification
type IntentResult struct {
	// Type is the classified intent type
	Type string `json:"type" yaml:"type"`
	
	// Confidence is the classification confidence
	Confidence float64 `json:"confidence" yaml:"confidence"`
	
	// Reasoning is the classification reasoning
	Reasoning string `json:"reasoning" yaml:"reasoning"`
	
	// Context contains additional context
	Context map[string]any `json:"context" yaml:"context"`
	
	// RelatedSession is the related session (for follow-up)
	RelatedSession string `json:"related_session" yaml:"related_session"`
	
	// PendingQuestion is the pending question (for clarification)
	PendingQuestion string `json:"pending_question" yaml:"pending_question"`
	
	// ExtractedAnswer is the extracted answer (for clarification response)
	ExtractedAnswer string `json:"extracted_answer" yaml:"extracted_answer"`
}

// NewThought creates a new Thought
func NewThought(content, reasoning, decision string, confidence float64) *Thought {
	return &Thought{
		Content:    content,
		Reasoning:  reasoning,
		Decision:   decision,
		Confidence: confidence,
		Timestamp:  time.Now(),
	}
}

// WithAction sets the action intent
func (t *Thought) WithAction(action *ActionIntent) *Thought {
	t.Action = action
	return t
}

// WithFinalAnswer sets the final answer
func (t *Thought) WithFinalAnswer(answer string) *Thought {
	t.FinalAnswer = answer
	return t
}

// IsAct checks if the decision is to act
func (t *Thought) IsAct() bool {
	return t.Decision == "act"
}

// IsAnswer checks if the decision is to answer
func (t *Thought) IsAnswer() bool {
	return t.Decision == "answer"
}

// ToAction converts the thought's action intent to an Action
func (t *Thought) ToAction() *Action {
	if t.Action == nil {
		return nil
	}
	
	return &Action{
		Type:      common.ActionType(t.Action.Type),
		Target:    t.Action.Target,
		Params:    t.Action.Params,
		Reasoning: t.Action.Reasoning,
		Timestamp: time.Now(),
	}
}

// IntentFallbackStrategy represents the fallback strategy for intent recognition
type IntentFallbackStrategy struct {
	// MinConfidence is the minimum confidence threshold (default 0.5)
	MinConfidence float64 `json:"min_confidence" yaml:"min_confidence"`
	
	// ClarifyThreshold is the clarification request threshold (default 0.7)
	ClarifyThreshold float64 `json:"clarify_threshold" yaml:"clarify_threshold"`
	
	// DefaultIntent is the default intent when confidence is low
	DefaultIntent string `json:"default_intent" yaml:"default_intent"`
	
	// MaxRetries is the maximum number of retries
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
	
	// EnableClarification is whether to enable clarification requests
	EnableClarification bool `json:"enable_clarification" yaml:"enable_clarification"`
}

// DefaultIntentFallbackStrategy returns the default fallback strategy
func DefaultIntentFallbackStrategy() *IntentFallbackStrategy {
	return &IntentFallbackStrategy{
		MinConfidence:       0.5,
		ClarifyThreshold:    0.7,
		DefaultIntent:       "task",
		MaxRetries:          2,
		EnableClarification: true,
	}
}
