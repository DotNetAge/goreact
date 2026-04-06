// Package observer provides observability for the goreact framework.
package observer

import (
	"context"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Probe provides observability probes
type Probe struct {
	mu        sync.RWMutex
	traceID   string
	spans     []*Span
	metrics   *MetricsCollector
	logger    *Logger
	config    *Config
}

// Config represents observability configuration
type Config struct {
	EnableTracing bool
	EnableMetrics bool
	EnableLogging bool
	LogLevel      string
	SampleRate    float64
}

// DefaultConfig returns default observability config
func DefaultConfig() *Config {
	return &Config{
		EnableTracing: true,
		EnableMetrics: true,
		EnableLogging: true,
		LogLevel:      "info",
		SampleRate:    1.0,
	}
}

// NewProbe creates a new Probe
func NewProbe(config *Config) *Probe {
	if config == nil {
		config = DefaultConfig()
	}
	return &Probe{
		traceID: generateTraceID(),
		spans:   []*Span{},
		metrics: NewMetricsCollector(),
		logger:  NewLogger(config.LogLevel),
		config:  config,
	}
}

// StartSpan starts a new span
func (p *Probe) StartSpan(name string) *Span {
	if !p.config.EnableTracing {
		return nil
	}
	
	span := &Span{
		ID:        generateSpanID(),
		TraceID:   p.traceID,
		Name:      name,
		StartTime: time.Now(),
		Status:    "running",
	}
	
	p.mu.Lock()
	p.spans = append(p.spans, span)
	p.mu.Unlock()
	
	return span
}

// EndSpan ends a span
func (p *Probe) EndSpan(span *Span) {
	if span == nil {
		return
	}
	
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = "completed"
	
	p.metrics.RecordDuration(span.Name, span.Duration)
}

// RecordMetric records a metric
func (p *Probe) RecordMetric(name string, value float64) {
	if !p.config.EnableMetrics {
		return
	}
	p.metrics.Record(name, value)
}

// Log logs a message
func (p *Probe) Log(level, message string, fields map[string]any) {
	if !p.config.EnableLogging {
		return
	}
	p.logger.Log(level, message, fields)
}

// GetTrace returns the trace
func (p *Probe) GetTrace() *Trace {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return &Trace{
		ID:    p.traceID,
		Spans: p.spans,
	}
}

// GetMetrics returns the metrics
func (p *Probe) GetMetrics() map[string]float64 {
	return p.metrics.GetAll()
}

// Span represents a tracing span
type Span struct {
	ID        string
	TraceID   string
	ParentID  string
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Status    string
	Attrs     map[string]any
	Events    []*Event
}

// AddEvent adds an event to the span
func (s *Span) AddEvent(name string, attrs map[string]any) {
	s.Events = append(s.Events, &Event{
		Name:      name,
		Timestamp: time.Now(),
		Attrs:     attrs,
	})
}

// SetAttr sets an attribute
func (s *Span) SetAttr(key string, value any) {
	if s.Attrs == nil {
		s.Attrs = make(map[string]any)
	}
	s.Attrs[key] = value
}

// Event represents a span event
type Event struct {
	Name      string
	Timestamp time.Time
	Attrs     map[string]any
}

// Trace represents a full trace
type Trace struct {
	ID    string
	Spans []*Span
}

// MetricsCollector collects metrics
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]float64
	counters map[string]int64
	histograms map[string][]float64
}

// NewMetricsCollector creates a new MetricsCollector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics:    make(map[string]float64),
		counters:   make(map[string]int64),
		histograms: make(map[string][]float64),
	}
}

// Record records a metric value
func (m *MetricsCollector) Record(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics[name] = value
}

// Increment increments a counter
func (m *MetricsCollector) Increment(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name]++
}

// RecordDuration records a duration
func (m *MetricsCollector) RecordDuration(name string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], float64(d.Milliseconds()))
}

// Get gets a metric value
func (m *MetricsCollector) Get(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics[name]
}

// GetAll gets all metrics
func (m *MetricsCollector) GetAll() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]float64, len(m.metrics))
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

// Logger provides logging
type Logger struct {
	level  string
	fields map[string]any
}

// NewLogger creates a new Logger
func NewLogger(level string) *Logger {
	return &Logger{
		level:  level,
		fields: make(map[string]any),
	}
}

// Log logs a message
func (l *Logger) Log(level, message string, fields map[string]any) {
	// Would implement actual logging
	// Simplified - just format and print
}

// WithFields adds fields to the logger
func (l *Logger) WithFields(fields map[string]any) *Logger {
	newLogger := &Logger{
		level:  l.level,
		fields: make(map[string]any),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// TokenTracker tracks token usage
type TokenTracker struct {
	mu           sync.RWMutex
	totalPrompt  int
	totalCompletion int
	byModel      map[string]*common.TokenUsage
}

// NewTokenTracker creates a new TokenTracker
func NewTokenTracker() *TokenTracker {
	return &TokenTracker{
		byModel: make(map[string]*common.TokenUsage),
	}
}

// Track tracks token usage
func (t *TokenTracker) Track(model string, prompt, completion int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.totalPrompt += prompt
	t.totalCompletion += completion
	
	if _, exists := t.byModel[model]; !exists {
		t.byModel[model] = &common.TokenUsage{}
	}
	t.byModel[model].PromptTokens += prompt
	t.byModel[model].CompletionTokens += completion
	t.byModel[model].TotalTokens += prompt + completion
}

// GetTotal returns total token usage
func (t *TokenTracker) GetTotal() *common.TokenUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return &common.TokenUsage{
		PromptTokens:     t.totalPrompt,
		CompletionTokens: t.totalCompletion,
		TotalTokens:      t.totalPrompt + t.totalCompletion,
	}
}

// GetByModel returns token usage by model
func (t *TokenTracker) GetByModel(model string) *common.TokenUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if usage, exists := t.byModel[model]; exists {
		return &common.TokenUsage{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.TotalTokens,
		}
	}
	return nil
}

// ObserverFunc is a function that observes execution
type ObserverFunc func(ctx context.Context, event string, data map[string]any)

// Observable represents something that can be observed
type Observable interface {
	Observe(ctx context.Context, event string, data map[string]any)
}

// Helper functions

func generateTraceID() string {
	return "trace-" + randomString(16)
}

func generateSpanID() string {
	return "span-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
