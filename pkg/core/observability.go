package core

import "time"

// Logger defines a structured logging interface for the ReAct engine.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(err error, msg string, keysAndValues ...any)
	With(keysAndValues ...any) Logger
}

// Metrics defines the interface to record system behaviors, token usages, and latencies.
type Metrics interface {
	// IncCounter increments a given counter metric (e.g., total_requests, prompt_tokens).
	IncCounter(name string, value float64, tags map[string]string)

	// RecordTimer records the duration of an event (e.g., llm_latency, tool_execution_time).
	RecordTimer(name string, d time.Duration, tags map[string]string)

	// RecordGauge sets a specific metric to a single absolute value (e.g., current_memory).
	RecordGauge(name string, value float64, tags map[string]string)
}

// =======================
// Noop Implementations (Defaults)
// =======================

// noopLogger implements Logger but discards all output.
type noopLogger struct{}

func (noopLogger) Debug(msg string, keysAndValues ...any)            {}
func (noopLogger) Info(msg string, keysAndValues ...any)             {}
func (noopLogger) Warn(msg string, keysAndValues ...any)             {}
func (noopLogger) Error(err error, msg string, keysAndValues ...any) {}
func (n noopLogger) With(keysAndValues ...any) Logger                { return n }

// DefaultLogger returns a Logger that does nothing.
func DefaultLogger() Logger { return noopLogger{} }

// noopMetrics implements Metrics but discards all tracking.
type noopMetrics struct{}

func (noopMetrics) IncCounter(name string, value float64, tags map[string]string)    {}
func (noopMetrics) RecordTimer(name string, d time.Duration, tags map[string]string) {}
func (noopMetrics) RecordGauge(name string, value float64, tags map[string]string)   {}

// DefaultMetrics returns a Metrics collector that does nothing.
func DefaultMetrics() Metrics { return noopMetrics{} }
