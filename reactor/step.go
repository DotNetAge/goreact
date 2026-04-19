package reactor

import "time"

// Step represents one complete T-A-O cycle.
type Step struct {
	Iteration  int         `json:"iteration" yaml:"iteration"`
	Thought    Thought     `json:"thought" yaml:"thought"`
	Action     Action      `json:"action" yaml:"action"`
	Observation Observation `json:"observation" yaml:"observation"`
	Timestamp  time.Time   `json:"timestamp" yaml:"timestamp"`
	Duration   time.Duration `json:"duration" yaml:"duration"`
}
