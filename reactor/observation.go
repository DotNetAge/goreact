package reactor

import "time"

// Observation represents the output of the Observe phase.
type Observation struct {
	Success     bool      `json:"success" yaml:"success"`
	Result      string    `json:"result" yaml:"result"`
	Insights    []string  `json:"insights,omitempty" yaml:"insights,omitempty"`
	ShouldRetry bool      `json:"should_retry" yaml:"should_retry"`
	Error       string    `json:"error,omitempty" yaml:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp" yaml:"timestamp"`
}

// NewSuccessObservation creates an observation for a successful action.
func NewSuccessObservation(result string, insights ...string) *Observation {
	return &Observation{
		Success:   true,
		Result:    result,
		Insights:  insights,
		Timestamp: time.Now(),
	}
}

// NewErrorObservation creates an observation for a failed action.
func NewErrorObservation(err string, shouldRetry bool) *Observation {
	return &Observation{
		Success:     false,
		Error:       err,
		ShouldRetry: shouldRetry,
		Timestamp:   time.Now(),
	}
}

// NewRetryObservation creates an observation indicating the action should be retried.
func NewRetryObservation(reason string) *Observation {
	return &Observation{
		Success:     false,
		ShouldRetry: true,
		Insights:    []string{reason},
		Timestamp:   time.Now(),
	}
}
