package core

import (
	"time"
)

// Reflection represents a reflection on a failed execution
type Reflection struct {
	// Name is the unique identifier
	Name string `json:"name" yaml:"name"`
	
	// SessionName is the session name
	SessionName string `json:"session_name" yaml:"session_name"`
	
	// TrajectoryName is the trajectory name
	TrajectoryName string `json:"trajectory_name" yaml:"trajectory_name"`
	
	// FailureReason is the reason for failure
	FailureReason string `json:"failure_reason" yaml:"failure_reason"`
	
	// Analysis is the detailed analysis
	Analysis string `json:"analysis" yaml:"analysis"`
	
	// Heuristic is the heuristic lesson learned
	Heuristic string `json:"heuristic" yaml:"heuristic"`
	
	// Suggestions are the actionable suggestions
	Suggestions []string `json:"suggestions" yaml:"suggestions"`
	
	// Score is the reflection quality score
	Score float64 `json:"score" yaml:"score"`
	
	// TaskType is the type of task
	TaskType string `json:"task_type" yaml:"task_type"`
	
	// Timestamp is the reflection timestamp
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// NewReflection creates a new Reflection
func NewReflection(sessionName, trajectoryName, failureReason string) *Reflection {
	return &Reflection{
		Name:          "reflection-" + generateID(),
		SessionName:   sessionName,
		TrajectoryName: trajectoryName,
		FailureReason: failureReason,
		Suggestions:   []string{},
		Score:         0.5,
		Timestamp:     time.Now(),
	}
}

// WithAnalysis sets the analysis
func (r *Reflection) WithAnalysis(analysis string) *Reflection {
	r.Analysis = analysis
	return r
}

// WithHeuristic sets the heuristic
func (r *Reflection) WithHeuristic(heuristic string) *Reflection {
	r.Heuristic = heuristic
	return r
}

// WithSuggestions sets the suggestions
func (r *Reflection) WithSuggestions(suggestions []string) *Reflection {
	r.Suggestions = suggestions
	return r
}

// WithScore sets the score
func (r *Reflection) WithScore(score float64) *Reflection {
	r.Score = score
	return r
}

// WithTaskType sets the task type
func (r *Reflection) WithTaskType(taskType string) *Reflection {
	r.TaskType = taskType
	return r
}

// AddSuggestion adds a suggestion
func (r *Reflection) AddSuggestion(suggestion string) {
	r.Suggestions = append(r.Suggestions, suggestion)
}

// ReflectionNode represents a Reflection node in the memory graph
type ReflectionNode struct {
	BaseNode
	SessionName   string    `json:"session_name" yaml:"session_name"`
	TrajectoryName string   `json:"trajectory_name" yaml:"trajectory_name"`
	FailureReason string    `json:"failure_reason" yaml:"failure_reason"`
	Analysis      string    `json:"analysis" yaml:"analysis"`
	Heuristic     string    `json:"heuristic" yaml:"heuristic"`
	Suggestions   []string  `json:"suggestions" yaml:"suggestions"`
	Score         float64   `json:"score" yaml:"score"`
	TaskType      string    `json:"task_type" yaml:"task_type"`
}

// NewReflectionNode creates a new ReflectionNode
func NewReflectionNode(sessionName, trajectoryName, failureReason string) *ReflectionNode {
	return &ReflectionNode{
		BaseNode: BaseNode{
			Name:        "reflection-" + generateID(),
			NodeType:    "Reflection",
			Description: failureReason,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]any),
		},
		SessionName:    sessionName,
		TrajectoryName: trajectoryName,
		FailureReason:  failureReason,
		Suggestions:    []string{},
		Score:          0.5,
	}
}
