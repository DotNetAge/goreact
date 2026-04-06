package core

import (
	"encoding/json"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// State represents the execution state of the reactor
type State struct {
	// Session information
	SessionName string `json:"session_name" yaml:"session_name"`
	
	// Step tracking
	CurrentStep int `json:"current_step" yaml:"current_step"`
	MaxSteps    int `json:"max_steps" yaml:"max_steps"`
	
	// Input
	Input string `json:"input" yaml:"input"`
	Files []string `json:"files" yaml:"files"`
	
	// Plan
	Plan            *Plan `json:"plan" yaml:"plan"`
	CurrentPlanStep int   `json:"current_plan_step" yaml:"current_plan_step"`
	
	// ReAct loop data
	Thoughts     []*Thought     `json:"thoughts" yaml:"thoughts"`
	Actions      []*Action      `json:"actions" yaml:"actions"`
	Observations []*Observation `json:"observations" yaml:"observations"`
	Reflections  []*Reflection  `json:"reflections" yaml:"reflections"`
	
	// Trajectory
	Trajectory *Trajectory `json:"trajectory" yaml:"trajectory"`
	
	// Retry tracking
	RetryCount int `json:"retry_count" yaml:"retry_count"`
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
	
	// Context
	Context map[string]any `json:"context" yaml:"context"`
	
	// Status
	Status    common.Status `json:"status" yaml:"status"`
	StartTime time.Time     `json:"start_time" yaml:"start_time"`
	EndTime   time.Time     `json:"end_time" yaml:"end_time"`
	
	// Current step data (for quick access)
	CurrentThought     *Thought     `json:"current_thought" yaml:"current_thought"`
	CurrentAction      *Action      `json:"current_action" yaml:"current_action"`
	CurrentObservation *Observation `json:"current_observation" yaml:"current_observation"`
	
	// Active reflections (for retry injection)
	ActiveReflections []*Reflection `json:"active_reflections" yaml:"active_reflections"`
	
	// Pending question (for pause-resume)
	PendingQuestion *PendingQuestionNode `json:"pending_question" yaml:"pending_question"`
	
	// Frozen state (for serialization)
	FrozenState []byte `json:"frozen_state" yaml:"frozen_state"`
	
	// Token usage
	TokenUsage *common.TokenUsage `json:"token_usage" yaml:"token_usage"`
}

// NewState creates a new State
func NewState(sessionName, input string, maxSteps, maxRetries int) *State {
	return &State{
		SessionName:  sessionName,
		CurrentStep:  0,
		MaxSteps:     maxSteps,
		Input:        input,
		Files:        []string{},
		Plan:         nil,
		CurrentPlanStep: 0,
		Thoughts:     []*Thought{},
		Actions:      []*Action{},
		Observations: []*Observation{},
		Reflections:  []*Reflection{},
		Trajectory:   NewTrajectory(sessionName),
		RetryCount:   0,
		MaxRetries:   maxRetries,
		Context:      make(map[string]any),
		Status:       common.StatusPending,
		StartTime:    time.Now(),
		TokenUsage:   &common.TokenUsage{},
	}
}

// AddThought adds a thought to the state
func (s *State) AddThought(thought *Thought) {
	s.Thoughts = append(s.Thoughts, thought)
}

// AddAction adds an action to the state
func (s *State) AddAction(action *Action) {
	s.Actions = append(s.Actions, action)
}

// AddObservation adds an observation to the state
func (s *State) AddObservation(observation *Observation) {
	s.Observations = append(s.Observations, observation)
}

// AddReflection adds a reflection to the state
func (s *State) AddReflection(reflection *Reflection) {
	s.Reflections = append(s.Reflections, reflection)
}

// IncrementStep increments the current step
func (s *State) IncrementStep() {
	s.CurrentStep++
}

// IncrementRetry increments the retry count
func (s *State) IncrementRetry() {
	s.RetryCount++
}

// CanRetry checks if more retries are possible
func (s *State) CanRetry() bool {
	return s.RetryCount < s.MaxRetries
}

// IsComplete checks if the state has reached max steps
func (s *State) IsComplete() bool {
	return s.CurrentStep >= s.MaxSteps
}

// SetStatus sets the status
func (s *State) SetStatus(status common.Status) {
	s.Status = status
	if status == common.StatusCompleted || status == common.StatusFailed || status == common.StatusCanceled {
		s.EndTime = time.Now()
	}
}

// SetPendingQuestion sets the pending question
func (s *State) SetPendingQuestion(question *PendingQuestionNode) {
	s.PendingQuestion = question
	s.Status = common.StatusPaused
}

// ClearPendingQuestion clears the pending question
func (s *State) ClearPendingQuestion() {
	s.PendingQuestion = nil
	s.Status = common.StatusRunning
}

// AddTokenUsage adds to the token usage
func (s *State) AddTokenUsage(promptTokens, completionTokens int) {
	if s.TokenUsage == nil {
		s.TokenUsage = &common.TokenUsage{}
	}
	s.TokenUsage.PromptTokens += promptTokens
	s.TokenUsage.CompletionTokens += completionTokens
	s.TokenUsage.TotalTokens += promptTokens + completionTokens
}

// GetLastThought returns the last thought
func (s *State) GetLastThought() *Thought {
	if len(s.Thoughts) == 0 {
		return nil
	}
	return s.Thoughts[len(s.Thoughts)-1]
}

// GetLastAction returns the last action
func (s *State) GetLastAction() *Action {
	if len(s.Actions) == 0 {
		return nil
	}
	return s.Actions[len(s.Actions)-1]
}

// GetLastObservation returns the last observation
func (s *State) GetLastObservation() *Observation {
	if len(s.Observations) == 0 {
		return nil
	}
	return s.Observations[len(s.Observations)-1]
}

// GetDuration returns the execution duration
func (s *State) GetDuration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// Freeze serializes the state
func (s *State) Freeze() ([]byte, error) {
	return json.Marshal(s)
}

// Thaw deserializes the state
func (s *State) Thaw(data []byte) error {
	return json.Unmarshal(data, s)
}

// InjectReflection injects a reflection into the context
func (s *State) InjectReflection(reflection *Reflection) {
	s.ActiveReflections = append(s.ActiveReflections, reflection)
}

// ClearActiveReflections clears the active reflections
func (s *State) ClearActiveReflections() {
	s.ActiveReflections = []*Reflection{}
}

// SetCurrentThought sets the current thought
func (s *State) SetCurrentThought(thought *Thought) {
	s.CurrentThought = thought
}

// SetCurrentAction sets the current action
func (s *State) SetCurrentAction(action *Action) {
	s.CurrentAction = action
}

// SetCurrentObservation sets the current observation
func (s *State) SetCurrentObservation(observation *Observation) {
	s.CurrentObservation = observation
}

// BuildTrajectory builds the trajectory
func (s *State) BuildTrajectory() *Trajectory {
	builder := NewTrajectoryBuilder(s)
	return builder.Build()
}

// Clone creates a deep copy of the state
func (s *State) Clone() *State {
	clone := &State{
		SessionName:      s.SessionName,
		CurrentStep:      s.CurrentStep,
		MaxSteps:         s.MaxSteps,
		Input:            s.Input,
		Files:            append([]string{}, s.Files...),
		Plan:             s.Plan,
		CurrentPlanStep:  s.CurrentPlanStep,
		Thoughts:         append([]*Thought{}, s.Thoughts...),
		Actions:          append([]*Action{}, s.Actions...),
		Observations:     append([]*Observation{}, s.Observations...),
		Reflections:      append([]*Reflection{}, s.Reflections...),
		Trajectory:       s.Trajectory,
		RetryCount:       s.RetryCount,
		MaxRetries:       s.MaxRetries,
		Context:          copyMap(s.Context),
		Status:           s.Status,
		StartTime:        s.StartTime,
		EndTime:          s.EndTime,
		PendingQuestion:  s.PendingQuestion,
		FrozenState:      append([]byte{}, s.FrozenState...),
	}
	if s.TokenUsage != nil {
		clone.TokenUsage = &common.TokenUsage{
			PromptTokens:     s.TokenUsage.PromptTokens,
			CompletionTokens: s.TokenUsage.CompletionTokens,
			TotalTokens:      s.TokenUsage.TotalTokens,
		}
	}
	return clone
}

// copyMap creates a deep copy of a map
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	copy := make(map[string]any, len(m))
	for k, v := range m {
		copy[k] = v
	}
	return copy
}
