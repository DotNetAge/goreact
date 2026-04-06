package core

import (
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Trajectory represents the execution trajectory of a session
type Trajectory struct {
	// Name is the unique identifier
	Name string `json:"name" yaml:"name"`
	
	// SessionName is the session name
	SessionName string `json:"session_name" yaml:"session_name"`
	
	// Steps are the trajectory steps
	Steps []*TrajectoryStep `json:"steps" yaml:"steps"`
	
	// Thoughts are all thoughts
	Thoughts []*Thought `json:"thoughts" yaml:"thoughts"`
	
	// Actions are all actions
	Actions []*Action `json:"actions" yaml:"actions"`
	
	// Observations are all observations
	Observations []*Observation `json:"observations" yaml:"observations"`
	
	// Success indicates if the trajectory succeeded
	Success bool `json:"success" yaml:"success"`
	
	// FailurePoint is the step index where failure occurred (-1 if no failure)
	FailurePoint int `json:"failure_point" yaml:"failure_point"`
	
	// FinalResult is the final result
	FinalResult string `json:"final_result" yaml:"final_result"`
	
	// Duration is the total execution duration
	Duration time.Duration `json:"duration" yaml:"duration"`
	
	// Summary is the trajectory summary
	Summary string `json:"summary" yaml:"summary"`
	
	// StartTime is the start timestamp
	StartTime time.Time `json:"start_time" yaml:"start_time"`
	
	// EndTime is the end timestamp
	EndTime time.Time `json:"end_time" yaml:"end_time"`
}

// TrajectoryStep represents a single step in the trajectory
type TrajectoryStep struct {
	// Index is the step index
	Index int `json:"index" yaml:"index"`
	
	// Thought is the thought at this step
	Thought *Thought `json:"thought" yaml:"thought"`
	
	// Action is the action at this step
	Action *Action `json:"action" yaml:"action"`
	
	// Observation is the observation at this step
	Observation *Observation `json:"observation" yaml:"observation"`
	
	// Timestamp is the step timestamp
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	
	// Duration is the step duration
	Duration time.Duration `json:"duration" yaml:"duration"`
}

// NewTrajectory creates a new Trajectory
func NewTrajectory(sessionName string) *Trajectory {
	return &Trajectory{
		Name:         "trajectory-" + generateID(),
		SessionName:  sessionName,
		Steps:        []*TrajectoryStep{},
		Thoughts:     []*Thought{},
		Actions:      []*Action{},
		Observations: []*Observation{},
		FailurePoint: -1,
		StartTime:    time.Now(),
	}
}

// AddStep adds a step to the trajectory
func (t *Trajectory) AddStep(thought *Thought, action *Action, observation *Observation) *TrajectoryStep {
	step := &TrajectoryStep{
		Index:      len(t.Steps),
		Thought:    thought,
		Action:     action,
		Observation: observation,
		Timestamp:  time.Now(),
	}
	
	t.Steps = append(t.Steps, step)
	
	if thought != nil {
		t.Thoughts = append(t.Thoughts, thought)
	}
	if action != nil {
		t.Actions = append(t.Actions, action)
	}
	if observation != nil {
		t.Observations = append(t.Observations, observation)
	}
	
	return step
}

// MarkSuccess marks the trajectory as successful
func (t *Trajectory) MarkSuccess(result string) {
	t.Success = true
	t.FinalResult = result
	t.EndTime = time.Now()
	t.Duration = t.EndTime.Sub(t.StartTime)
}

// MarkFailure marks the trajectory as failed
func (t *Trajectory) MarkFailure(reason string) {
	t.Success = false
	t.FinalResult = reason
	t.FailurePoint = len(t.Steps) - 1
	t.EndTime = time.Now()
	t.Duration = t.EndTime.Sub(t.StartTime)
}

// GetFailureContext returns the context around the failure point
func (t *Trajectory) GetFailureContext() []*TrajectoryStep {
	if t.FailurePoint < 0 || t.FailurePoint >= len(t.Steps) {
		return nil
	}
	
	start := t.FailurePoint - 2
	if start < 0 {
		start = 0
	}
	end := t.FailurePoint + 2
	if end > len(t.Steps) {
		end = len(t.Steps)
	}
	
	return t.Steps[start:end]
}

// GetStepCount returns the total step count
func (t *Trajectory) GetStepCount() int {
	return len(t.Steps)
}

// GetActionCount returns the total action count
func (t *Trajectory) GetActionCount() int {
	return len(t.Actions)
}

// GetSuccessRate returns the success rate of actions
func (t *Trajectory) GetSuccessRate() float64 {
	if len(t.Observations) == 0 {
		return 0
	}
	
	successCount := 0
	for _, obs := range t.Observations {
		if obs.Success {
			successCount++
		}
	}
	
	return float64(successCount) / float64(len(t.Observations))
}

// TrajectoryNode represents a Trajectory node in the memory graph
type TrajectoryNode struct {
	BaseNode
	SessionName string           `json:"session_name" yaml:"session_name"`
	Steps       []*TrajectoryStep `json:"steps" yaml:"steps"`
	Success     bool             `json:"success" yaml:"success"`
	FailurePoint int             `json:"failure_point" yaml:"failure_point"`
	FinalResult string           `json:"final_result" yaml:"final_result"`
	Duration    time.Duration    `json:"duration" yaml:"duration"`
	Summary     string           `json:"summary" yaml:"summary"`
}

// NewTrajectoryNode creates a new TrajectoryNode
func NewTrajectoryNode(sessionName string) *TrajectoryNode {
	return &TrajectoryNode{
		BaseNode: BaseNode{
			Name:        "trajectory-" + generateID(),
			NodeType:    common.NodeTypeTrajectory,
			Description: "",
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		SessionName:  sessionName,
		Steps:        []*TrajectoryStep{},
		FailurePoint: -1,
	}
}
