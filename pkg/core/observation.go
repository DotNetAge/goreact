package core

import (
	"time"
)

// Observation represents an observation from an action execution
type Observation struct {
	// Content is the processed result content
	Content string `json:"content" yaml:"content"`
	
	// Source is the source (tool/skill/agent name)
	Source string `json:"source" yaml:"source"`
	
	// Timestamp is the observation timestamp
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	
	// Insights are the extracted insights
	Insights []string `json:"insights" yaml:"insights"`
	
	// Relevance is the relevance to the current task (0.0-1.0)
	Relevance float64 `json:"relevance" yaml:"relevance"`
	
	// Success indicates if the execution was successful
	Success bool `json:"success" yaml:"success"`
	
	// Error is the error message if execution failed
	Error string `json:"error" yaml:"error"`
	
	// Metadata contains additional metadata
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
	
	// RelatedActions are the related action names
	RelatedActions []string `json:"related_actions" yaml:"related_actions"`
	
	// RelatedThoughts are the related thought contents
	RelatedThoughts []string `json:"related_thoughts" yaml:"related_thoughts"`
}

// ObservationContext represents the context for observation processing
type ObservationContext struct {
	// TaskInput is the original task input
	TaskInput string `json:"task_input" yaml:"task_input"`
	
	// CurrentStep is the current execution step
	CurrentStep int `json:"current_step" yaml:"current_step"`
	
	// PlanStep is the current plan step description
	PlanStep string `json:"plan_step" yaml:"plan_step"`
	
	// PreviousObservations are the previous observations
	PreviousObservations []*Observation `json:"previous_observations" yaml:"previous_observations"`
	
	// ExpectedOutcome is the expected outcome
	ExpectedOutcome string `json:"expected_outcome" yaml:"expected_outcome"`
	
	// ActualOutcome is the actual outcome
	ActualOutcome string `json:"actual_outcome" yaml:"actual_outcome"`
	
	// Deviation describes the deviation from expected
	Deviation string `json:"deviation" yaml:"deviation"`
}

// NewObservation creates a new Observation
func NewObservation(content, source string, success bool) *Observation {
	return &Observation{
		Content:         content,
		Source:          source,
		Timestamp:       time.Now(),
		Insights:        []string{},
		Relevance:       0.5,
		Success:         success,
		Metadata:        make(map[string]any),
		RelatedActions:  []string{},
		RelatedThoughts: []string{},
	}
}

// WithInsights sets the insights
func (o *Observation) WithInsights(insights []string) *Observation {
	o.Insights = insights
	return o
}

// WithRelevance sets the relevance
func (o *Observation) WithRelevance(relevance float64) *Observation {
	o.Relevance = relevance
	return o
}

// WithError sets the error
func (o *Observation) WithError(err string) *Observation {
	o.Error = err
	o.Success = false
	return o
}

// WithMetadata adds metadata
func (o *Observation) WithMetadata(key string, value any) *Observation {
	if o.Metadata == nil {
		o.Metadata = make(map[string]any)
	}
	o.Metadata[key] = value
	return o
}

// AddInsight adds an insight
func (o *Observation) AddInsight(insight string) {
	o.Insights = append(o.Insights, insight)
}

// AddRelatedAction adds a related action
func (o *Observation) AddRelatedAction(action string) {
	o.RelatedActions = append(o.RelatedActions, action)
}

// AddRelatedThought adds a related thought
func (o *Observation) AddRelatedThought(thought string) {
	o.RelatedThoughts = append(o.RelatedThoughts, thought)
}
