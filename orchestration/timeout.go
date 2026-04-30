package orchestration

import "time"

// ===========================================================================
// Timeout Configuration — Three-Level Timeout Strategy (Design §10.3)
// ===========================================================================
//
// The Coordinator uses three levels of timeouts to avoid both premature
// termination and infinite waiting. Each level triggers different behavior:
//
//  Level 1: Per-task timeout    = ExpectedDuration * 2     → mark TIMEOUT, continue others
//  Level 2: Global soft timeout = MaxExpected * 3            → warn user, ask to continue
//  Level 3: Global hard timeout = MaxExpected * 5            → force-terminate everything
//
// These multipliers are configurable via TimeoutConfig.

// TimeoutConfig holds the timeout parameters for a Coordinator session.
// All durations are derived from task ExpectedDur fields at dispatch time.
type TimeoutConfig struct {
	// SingleTaskMultiplier is applied to each task's ExpectedDuration for per-task timeout.
	// Default: 2.0 (a task gets twice its estimated time before being marked timed out).
	SingleTaskMultiplier float64

	// SoftTimeoutMultiplier is applied to the longest ExpectedDuration as the global soft threshold.
	// When reached, the Coordinator emits a TimeoutWarningEvent and waits for user decision.
	// Default: 3.0
	SoftTimeoutMultiplier float64

	// HardTimeoutMultiplier is applied to the longest ExpectedDuration as the global hard limit.
	// When reached, the Coordinator force-cancels all remaining tasks unconditionally.
	// Default: 5.0
	HardTimeoutMultiplier float64

	// MaxRetries is the maximum number of retries per failed/timed-out task (Design §10.4).
	// Default: 2
	MaxRetries int

	// RetryInitialDelay is the initial delay before first retry (exponential backoff base).
	// Default: 1 second
	RetryInitialDelay time.Duration

	// FailureRateThreshold is the failure rate (0.0-1.0) above which the Coordinator
	// aborts all tasks and reports partial results. Default: 0.5 (50%).
	FailureRateThreshold float64

	// MinPollInterval is the floor for adaptive poll interval. Default: 500ms.
	MinPollInterval time.Duration

	// MaxPollInterval is the ceiling for adaptive poll interval. Default: 10s.
	MaxPollInterval time.Duration

	// UserDecisionTimeout is how long the Coordinator waits for user response at soft timeout.
	// After this, it defaults to "continue with completed results". Default: 30s.
	UserDecisionTimeout time.Duration
}

// DefaultTimeoutConfig returns the standard timeout configuration matching Design §10.3/§10.4 defaults.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		SingleTaskMultiplier:   2.0,
		SoftTimeoutMultiplier:  3.0,
		HardTimeoutMultiplier:  5.0,
		MaxRetries:             2,
		RetryInitialDelay:      1 * time.Second,
		FailureRateThreshold:   0.5,
		MinPollInterval:        500 * time.Millisecond,
		MaxPollInterval:        10 * time.Second,
		UserDecisionTimeout:     30 * time.Second,
	}
}

// resolveTimeouts computes absolute timeout durations from a TaskProgressTable's expected durations.
// Returns (singleTaskMap, softGlobal, hardGlobal) where singleTaskMap maps taskID→deadline.
func (cfg TimeoutConfig) resolveTimeouts(table *TaskProgressTable) (map[string]time.Time, time.Time, time.Time) {
	now := time.Now()
	singleTask := make(map[string]time.Time)

	var maxExpected time.Duration
	for _, entry := range table.Entries() {
		if entry.ExpectedDur > maxExpected {
			maxExpected = entry.ExpectedDur
		}
		deadline := now.Add(time.Duration(float64(entry.ExpectedDur) * cfg.SingleTaskMultiplier))
		singleTask[entry.TaskID] = deadline
	}

	soft := now.Add(time.Duration(float64(maxExpected) * cfg.SoftTimeoutMultiplier))
	hard := now.Add(time.Duration(float64(maxExpected) * cfg.HardTimeoutMultiplier))

	return singleTask, soft, hard
}

// computePollInterval derives the adaptive poll interval from running task entries.
// Follows Design §10.2 formula: interval = avgRemaining / 5, clamped to [Min, Max].
func (cfg TimeoutConfig) computePollInterval(running []*TaskEntry) time.Duration {
	if len(running) == 0 {
		return cfg.MinPollInterval
	}

	var totalRemaining time.Duration
	now := time.Now()
	for _, t := range running {
		elapsed := now.Sub(t.ActualStart)
		remaining := t.ExpectedDur - elapsed
		if remaining < 0 {
			remaining = 1 * time.Second
		}
		totalRemaining += remaining
	}
	avgRemaining := totalRemaining / time.Duration(len(running))
	interval := avgRemaining / 5

	switch {
	case interval < cfg.MinPollInterval:
		return cfg.MinPollInterval
	case interval > cfg.MaxPollInterval:
		return cfg.MaxPollInterval
	default:
		return interval
	}
}
