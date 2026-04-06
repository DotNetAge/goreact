package observer

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.EnableTracing {
		t.Error("EnableTracing should be true by default")
	}
	if !config.EnableMetrics {
		t.Error("EnableMetrics should be true by default")
	}
	if !config.EnableLogging {
		t.Error("EnableLogging should be true by default")
	}
	if config.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want 'info'", config.LogLevel)
	}
	if config.SampleRate != 1.0 {
		t.Errorf("SampleRate = %f, want 1.0", config.SampleRate)
	}
}

func TestNewProbe(t *testing.T) {
	probe := NewProbe(nil)

	if probe == nil {
		t.Fatal("NewProbe() returned nil")
	}
	if probe.traceID == "" {
		t.Error("traceID should not be empty")
	}
	if probe.metrics == nil {
		t.Error("metrics should not be nil")
	}
	if probe.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestNewProbe_WithConfig(t *testing.T) {
	config := &Config{
		EnableTracing: true,
		EnableMetrics: false,
		EnableLogging: true,
		LogLevel:      "debug",
	}
	probe := NewProbe(config)

	if probe.config == nil {
		t.Fatal("config should not be nil")
	}
	if probe.config.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want 'debug'", probe.config.LogLevel)
	}
}

func TestProbe_StartEndSpan(t *testing.T) {
	probe := NewProbe(nil)

	span := probe.StartSpan("test-operation")
	if span == nil {
		t.Fatal("StartSpan() returned nil")
	}
	if span.Name != "test-operation" {
		t.Errorf("Span name = %q, want 'test-operation'", span.Name)
	}
	if span.Status != "running" {
		t.Errorf("Span status = %q, want 'running'", span.Status)
	}

	probe.EndSpan(span)
	if span.Status != "completed" {
		t.Errorf("Span status after EndSpan = %q, want 'completed'", span.Status)
	}
	if span.Duration <= 0 {
		t.Error("Span duration should be positive after EndSpan")
	}
}

func TestProbe_StartSpan_Disabled(t *testing.T) {
	config := &Config{EnableTracing: false}
	probe := NewProbe(config)

	span := probe.StartSpan("test")
	if span != nil {
		t.Error("StartSpan() should return nil when tracing is disabled")
	}
}

func TestProbe_RecordMetric(t *testing.T) {
	probe := NewProbe(nil)

	probe.RecordMetric("requests_total", 100)
	probe.RecordMetric("latency_ms", 50.5)

	metrics := probe.GetMetrics()
	if metrics["requests_total"] != 100 {
		t.Errorf("requests_total = %v, want 100", metrics["requests_total"])
	}
	if metrics["latency_ms"] != 50.5 {
		t.Errorf("latency_ms = %v, want 50.5", metrics["latency_ms"])
	}
}

func TestProbe_RecordMetric_Disabled(t *testing.T) {
	config := &Config{EnableMetrics: false}
	probe := NewProbe(config)

	probe.RecordMetric("test", 100)
	metrics := probe.GetMetrics()

	if len(metrics) != 0 {
		t.Errorf("Metrics should be empty when disabled, got %d", len(metrics))
	}
}

func TestProbe_GetTrace(t *testing.T) {
	probe := NewProbe(nil)

	span1 := probe.StartSpan("op1")
	span2 := probe.StartSpan("op2")

	trace := probe.GetTrace()
	if trace == nil {
		t.Fatal("GetTrace() returned nil")
	}
	if trace.ID != probe.traceID {
		t.Errorf("Trace ID = %q, want %q", trace.ID, probe.traceID)
	}
	if len(trace.Spans) != 2 {
		t.Errorf("len(Spans) = %d, want 2", len(trace.Spans))
	}

	probe.EndSpan(span1)
	probe.EndSpan(span2)
}

func TestSpan_AddEvent(t *testing.T) {
	span := &Span{
		ID:        "span-123",
		Name:      "test",
		StartTime: time.Now(),
	}

	span.AddEvent("checkpoint", map[string]any{"step": 1})
	span.AddEvent("error", map[string]any{"message": "test error"})

	if len(span.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(span.Events))
	}
	if span.Events[0].Name != "checkpoint" {
		t.Errorf("Event name = %q, want 'checkpoint'", span.Events[0].Name)
	}
}

func TestSpan_SetAttr(t *testing.T) {
	span := &Span{
		ID:        "span-123",
		Name:      "test",
		StartTime: time.Now(),
	}

	span.SetAttr("user_id", "user-001")
	span.SetAttr("session_id", "session-123")

	if span.Attrs["user_id"] != "user-001" {
		t.Errorf("Attrs[user_id] = %v, want 'user-001'", span.Attrs["user_id"])
	}
	if span.Attrs["session_id"] != "session-123" {
		t.Errorf("Attrs[session_id] = %v, want 'session-123'", span.Attrs["session_id"])
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	collector.Record("latency", 100.5)
	collector.Record("throughput", 1000.0)
	collector.Increment("requests")
	collector.Increment("requests")
	collector.Increment("errors")

	if collector.Get("latency") != 100.5 {
		t.Errorf("Get(latency) = %v, want 100.5", collector.Get("latency"))
	}

	all := collector.GetAll()
	if all["throughput"] != 1000.0 {
		t.Errorf("GetAll()[throughput] = %v, want 1000.0", all["throughput"])
	}
}

func TestMetricsCollector_RecordDuration(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordDuration("request_duration", 100*time.Millisecond)
	collector.RecordDuration("request_duration", 200*time.Millisecond)

	if collector.histograms["request_duration"] == nil {
		t.Fatal("histogram should not be nil")
	}
	if len(collector.histograms["request_duration"]) != 2 {
		t.Errorf("len(histogram) = %d, want 2", len(collector.histograms["request_duration"]))
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger("debug")

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
	if logger.level != "debug" {
		t.Errorf("level = %q, want 'debug'", logger.level)
	}
}

func TestLogger_WithFields(t *testing.T) {
	logger := NewLogger("info")
	logger2 := logger.WithFields(map[string]any{"service": "goreact"})

	if logger2 == nil {
		t.Fatal("WithFields() returned nil")
	}
	if logger2.fields["service"] != "goreact" {
		t.Errorf("fields[service] = %v, want 'goreact'", logger2.fields["service"])
	}
}

func TestNewTokenTracker(t *testing.T) {
	tracker := NewTokenTracker()

	if tracker == nil {
		t.Fatal("NewTokenTracker() returned nil")
	}
	if tracker.byModel == nil {
		t.Error("byModel should not be nil")
	}
}

func TestTokenTracker_Track(t *testing.T) {
	tracker := NewTokenTracker()

	tracker.Track("gpt-4", 100, 50)
	tracker.Track("gpt-4", 200, 100)
	tracker.Track("gpt-3.5-turbo", 50, 25)

	total := tracker.GetTotal()
	if total.PromptTokens != 350 {
		t.Errorf("PromptTokens = %d, want 350", total.PromptTokens)
	}
	if total.CompletionTokens != 175 {
		t.Errorf("CompletionTokens = %d, want 175", total.CompletionTokens)
	}
	if total.TotalTokens != 525 {
		t.Errorf("TotalTokens = %d, want 525", total.TotalTokens)
	}

	gpt4 := tracker.GetByModel("gpt-4")
	if gpt4 == nil {
		t.Fatal("GetByModel(gpt-4) returned nil")
	}
	if gpt4.PromptTokens != 300 {
		t.Errorf("gpt-4 PromptTokens = %d, want 300", gpt4.PromptTokens)
	}
}

func TestTokenTracker_GetByModel_NotFound(t *testing.T) {
	tracker := NewTokenTracker()

	usage := tracker.GetByModel("unknown")
	if usage != nil {
		t.Errorf("GetByModel(unknown) should return nil, got %v", usage)
	}
}
