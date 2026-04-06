package core

import (
	"fmt"
	"strings"
)

// TrajectoryBuilder builds and analyzes trajectories
type TrajectoryBuilder struct {
	state *State
}

// NewTrajectoryBuilder creates a new TrajectoryBuilder
func NewTrajectoryBuilder(state *State) *TrajectoryBuilder {
	return &TrajectoryBuilder{state: state}
}

// Build builds a trajectory from the state
func (b *TrajectoryBuilder) Build() *Trajectory {
	if b.state.Trajectory != nil {
		return b.state.Trajectory
	}
	
	trajectory := NewTrajectory(b.state.SessionName)
	
	// Build steps from thoughts, actions, observations
	for i := 0; i < len(b.state.Thoughts); i++ {
		var thought *Thought
		var action *Action
		var observation *Observation
		
		if i < len(b.state.Thoughts) {
			thought = b.state.Thoughts[i]
		}
		if i < len(b.state.Actions) {
			action = b.state.Actions[i]
		}
		if i < len(b.state.Observations) {
			observation = b.state.Observations[i]
		}
		
		trajectory.AddStep(thought, action, observation)
	}
	
	return trajectory
}

// ExtractKeyDecisions extracts key decisions from the trajectory
func (b *TrajectoryBuilder) ExtractKeyDecisions() []*Decision {
	decisions := []*Decision{}
	trajectory := b.Build()
	
	for i, step := range trajectory.Steps {
		if step.Thought == nil {
			continue
		}
		
		// Check if this is a key decision
		isKeyDecision := false
		
		// High confidence decisions are key
		if step.Thought.Confidence >= 0.8 {
			isKeyDecision = true
		}
		
		// Decisions that changed direction are key
		if step.Action != nil && i > 0 {
			prevAction := trajectory.Steps[i-1].Action
			if prevAction != nil && prevAction.Type != step.Action.Type {
				isKeyDecision = true
			}
		}
		
		// Decisions after errors are key
		if step.Observation != nil && !step.Observation.Success {
			isKeyDecision = true
		}
		
		if isKeyDecision {
			decision := &Decision{
				Step:         i,
				Thought:      step.Thought.Content,
				Reasoning:    step.Thought.Reasoning,
				IsKeyDecision: true,
			}
			
			if step.Action != nil {
				decision.Action = fmt.Sprintf("%s: %s", step.Action.Type, step.Action.Target)
			}
			
			if step.Observation != nil {
				decision.Outcome = step.Observation.Content
			}
			
			decisions = append(decisions, decision)
		}
	}
	
	return decisions
}

// IdentifyFailurePoint identifies the point of failure
func (b *TrajectoryBuilder) IdentifyFailurePoint() int {
	trajectory := b.Build()
	
	for i, step := range trajectory.Steps {
		if step.Observation != nil && !step.Observation.Success {
			return i
		}
	}
	
	return -1
}

// Summarize generates a summary of the trajectory
func (b *TrajectoryBuilder) Summarize() string {
	trajectory := b.Build()
	
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("Trajectory: %s\n", trajectory.Name))
	sb.WriteString(fmt.Sprintf("Steps: %d\n", len(trajectory.Steps)))
	sb.WriteString(fmt.Sprintf("Success: %v\n", trajectory.Success))
	
	if trajectory.FailurePoint >= 0 {
		sb.WriteString(fmt.Sprintf("Failure Point: Step %d\n", trajectory.FailurePoint))
	}
	
	keyDecisions := b.ExtractKeyDecisions()
	if len(keyDecisions) > 0 {
		sb.WriteString("\nKey Decisions:\n")
		for _, d := range keyDecisions {
			sb.WriteString(fmt.Sprintf("  Step %d: %s\n", d.Step, d.Thought))
		}
	}
	
	return sb.String()
}

// Decision represents a key decision point in the trajectory
type Decision struct {
	// Step is the step index
	Step int `json:"step" yaml:"step"`
	
	// Thought is the thought content
	Thought string `json:"thought" yaml:"thought"`
	
	// Action is the action taken
	Action string `json:"action" yaml:"action"`
	
	// Reasoning is the reasoning behind the decision
	Reasoning string `json:"reasoning" yaml:"reasoning"`
	
	// Outcome is the outcome of the decision
	Outcome string `json:"outcome" yaml:"outcome"`
	
	// IsKeyDecision indicates if this is a key decision
	IsKeyDecision bool `json:"is_key_decision" yaml:"is_key_decision"`
}

// NewDecision creates a new Decision
func NewDecision(step int, thought string) *Decision {
	return &Decision{
		Step:   step,
		Thought: thought,
	}
}

// WithAction sets the action
func (d *Decision) WithAction(action string) *Decision {
	d.Action = action
	return d
}

// WithReasoning sets the reasoning
func (d *Decision) WithReasoning(reasoning string) *Decision {
	d.Reasoning = reasoning
	return d
}

// WithOutcome sets the outcome
func (d *Decision) WithOutcome(outcome string) *Decision {
	d.Outcome = outcome
	return d
}

// MarkAsKey marks the decision as key
func (d *Decision) MarkAsKey() *Decision {
	d.IsKeyDecision = true
	return d
}
