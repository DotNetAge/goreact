package core

import (
	"github.com/DotNetAge/goreact/pkg/common"
)

// EvaluationResult represents the result of task evaluation
type EvaluationResult struct {
	// Success indicates if the task succeeded
	Success bool `json:"success" yaml:"success"`
	
	// Reason is the reason for the result
	Reason string `json:"reason" yaml:"reason"`
	
	// Score is the quality score (0.0-1.0)
	Score float64 `json:"score" yaml:"score"`
	
	// Metrics contains detailed metrics
	Metrics map[string]float64 `json:"metrics" yaml:"metrics"`
	
	// Suggestions are improvement suggestions
	Suggestions []string `json:"suggestions" yaml:"suggestions"`
	
	// TerminationReason is the reason for termination
	TerminationReason common.TerminationReason `json:"termination_reason" yaml:"termination_reason"`
}

// NewEvaluationResult creates a new EvaluationResult
func NewEvaluationResult(success bool, reason string) *EvaluationResult {
	return &EvaluationResult{
		Success:     success,
		Reason:      reason,
		Score:       0.5,
		Metrics:     make(map[string]float64),
		Suggestions: []string{},
	}
}

// WithScore sets the score
func (e *EvaluationResult) WithScore(score float64) *EvaluationResult {
	e.Score = score
	return e
}

// WithMetric adds a metric
func (e *EvaluationResult) WithMetric(name string, value float64) *EvaluationResult {
	if e.Metrics == nil {
		e.Metrics = make(map[string]float64)
	}
	e.Metrics[name] = value
	return e
}

// WithSuggestions sets the suggestions
func (e *EvaluationResult) WithSuggestions(suggestions []string) *EvaluationResult {
	e.Suggestions = suggestions
	return e
}

// AddSuggestion adds a suggestion
func (e *EvaluationResult) AddSuggestion(suggestion string) {
	e.Suggestions = append(e.Suggestions, suggestion)
}

// WithTerminationReason sets the termination reason
func (e *EvaluationResult) WithTerminationReason(reason common.TerminationReason) *EvaluationResult {
	e.TerminationReason = reason
	return e
}

// Evaluator interface for evaluating task results
type Evaluator interface {
	// Evaluate evaluates the state
	Evaluate(state *State) *EvaluationResult
}

// GoalEvaluator evaluates based on goal achievement
type GoalEvaluator struct {
	goalChecker func(state *State) bool
}

// NewGoalEvaluator creates a new GoalEvaluator
func NewGoalEvaluator(checker func(state *State) bool) *GoalEvaluator {
	return &GoalEvaluator{goalChecker: checker}
}

// Evaluate evaluates the state
func (e *GoalEvaluator) Evaluate(state *State) *EvaluationResult {
	if e.goalChecker == nil {
		return NewEvaluationResult(false, "no goal checker defined")
	}
	
	success := e.goalChecker(state)
	result := NewEvaluationResult(success, "goal evaluation")
	
	if success {
		result.WithScore(1.0).WithTerminationReason(common.TerminationReasonGoalAchieved)
	} else {
		result.WithScore(0.0)
	}
	
	return result
}

// RuleEvaluator evaluates based on rules
type RuleEvaluator struct {
	rules []EvaluationRule
}

// EvaluationRule represents an evaluation rule
type EvaluationRule struct {
	Name        string
	Condition   func(state *State) bool
	ScoreImpact float64
	Message     string
}

// NewRuleEvaluator creates a new RuleEvaluator
func NewRuleEvaluator(rules []EvaluationRule) *RuleEvaluator {
	return &RuleEvaluator{rules: rules}
}

// Evaluate evaluates the state
func (e *RuleEvaluator) Evaluate(state *State) *EvaluationResult {
	result := NewEvaluationResult(true, "rule evaluation")
	totalScore := 0.0
	ruleCount := 0
	
	for _, rule := range e.rules {
		if rule.Condition(state) {
			totalScore += rule.ScoreImpact
			ruleCount++
			if rule.ScoreImpact < 0 {
				result.AddSuggestion(rule.Message)
			}
		}
	}
	
	if ruleCount > 0 {
		result.Score = totalScore / float64(ruleCount)
	}
	
	result.Success = result.Score >= 0.5
	return result
}

// LLMEvaluator evaluates using LLM
type LLMEvaluator struct {
	llmClient any // Would be LLMClient
	prompt    string
}

// NewLLMEvaluator creates a new LLMEvaluator
func NewLLMEvaluator(llmClient any, prompt string) *LLMEvaluator {
	return &LLMEvaluator{
		llmClient: llmClient,
		prompt:    prompt,
	}
}

// Evaluate evaluates the state
func (e *LLMEvaluator) Evaluate(state *State) *EvaluationResult {
	// Would use LLM to evaluate
	// Simplified implementation
	result := NewEvaluationResult(true, "LLM evaluation (placeholder)")
	result.Score = 0.7
	return result
}
