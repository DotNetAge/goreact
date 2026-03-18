package core

import "time"

// Logger defines a structured logging interface for the ReAct engine.
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) Logger
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

func (noopLogger) Debug(msg string, keysAndValues ...interface{})             {}
func (noopLogger) Info(msg string, keysAndValues ...interface{})              {}
func (noopLogger) Warn(msg string, keysAndValues ...interface{})              {}
func (noopLogger) Error(err error, msg string, keysAndValues ...interface{})  {}
func (n noopLogger) With(keysAndValues ...interface{}) Logger                 { return n }

// DefaultLogger returns a Logger that does nothing.
func DefaultLogger() Logger { return noopLogger{} }

// noopMetrics implements Metrics but discards all tracking.
type noopMetrics struct{}

func (noopMetrics) IncCounter(name string, value float64, tags map[string]string) {}
func (noopMetrics) RecordTimer(name string, d time.Duration, tags map[string]string) {}
func (noopMetrics) RecordGauge(name string, value float64, tags map[string]string) {}

// DefaultMetrics returns a Metrics collector that does nothing.
func DefaultMetrics() Metrics { return noopMetrics{} }
