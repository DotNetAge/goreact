package core

import (
	"context"
	"testing"
	"time"

	chatcore "github.com/DotNetAge/gochat/pkg/core"
)

func TestNewPipelineContext(t *testing.T) {
	ctx := context.Background()
	sessionID := "test-session"
	input := "test input"

	pctx := NewPipelineContext(ctx, sessionID, input)

	if pctx.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, pctx.SessionID)
	}
	if pctx.Input != input {
		t.Errorf("Expected Input %s, got %s", input, pctx.Input)
	}
	if pctx.CurrentStep != 1 {
		t.Errorf("Expected CurrentStep 1, got %d", pctx.CurrentStep)
	}
	if pctx.IsFinished {
		t.Error("Expected IsFinished to be false")
	}
	if pctx.MaxSteps != 10 {
		t.Errorf("Expected MaxSteps 10, got %d", pctx.MaxSteps)
	}
	if pctx.Logger == nil {
		t.Error("Expected Logger to be set")
	}
	if pctx.Metrics == nil {
		t.Error("Expected Metrics to be set")
	}
	if pctx.TotalTokens == nil {
		t.Error("Expected TotalTokens to be set")
	}
	if pctx.Traces == nil {
		t.Error("Expected Traces to be initialized")
	}
}

func TestPipelineContext_WithAttachments(t *testing.T) {
	ctx := context.Background()
	pctx := NewPipelineContext(ctx, "test", "input")

	attachment := chatcore.Attachment{URL: "test.jpg"}
	pctx = NewPipelineContext(ctx, "test", "input", WithAttachments(attachment))

	if len(pctx.Attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(pctx.Attachments))
	}
}

func TestPipelineContext_WithMaxSteps(t *testing.T) {
	ctx := context.Background()
	pctx := NewPipelineContext(ctx, "test", "input", WithMaxSteps(20))

	if pctx.MaxSteps != 20 {
		t.Errorf("Expected MaxSteps 20, got %d", pctx.MaxSteps)
	}
}

func TestPipelineContext_WithLogger(t *testing.T) {
	ctx := context.Background()
	customLogger := &noopLogger{}
	pctx := NewPipelineContext(ctx, "test", "input", WithLogger(customLogger))

	if pctx.Logger != customLogger {
		t.Error("Expected custom logger to be set")
	}
}

func TestPipelineContext_WithMetrics(t *testing.T) {
	ctx := context.Background()
	customMetrics := &noopMetrics{}
	pctx := NewPipelineContext(ctx, "test", "input", WithMetrics(customMetrics))

	if pctx.Metrics != customMetrics {
		t.Error("Expected custom metrics to be set")
	}
}

func TestPipelineContext_WithThoughtStream(t *testing.T) {
	ctx := context.Background()
	called := false
	hook := func(s string) { called = true }
	pctx := NewPipelineContext(ctx, "test", "input", WithThoughtStream(hook))

	if pctx.OnThoughtStream == nil {
		t.Error("Expected OnThoughtStream to be set")
	}
	pctx.OnThoughtStream("test")
	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestPipelineContext_AppendTrace(t *testing.T) {
	ctx := context.Background()
	pctx := NewPipelineContext(ctx, "test", "input")

	trace := &Trace{
		Step:    1,
		Thought: "test thought",
	}
	pctx.AppendTrace(trace)

	if len(pctx.Traces) != 1 {
		t.Errorf("Expected 1 trace, got %d", len(pctx.Traces))
	}
	if pctx.CurrentStep != 2 {
		t.Errorf("Expected CurrentStep 2, got %d", pctx.CurrentStep)
	}
}

func TestPipelineContext_LastTrace(t *testing.T) {
	ctx := context.Background()
	pctx := NewPipelineContext(ctx, "test", "input")

	if pctx.LastTrace() != nil {
		t.Error("Expected nil for empty traces")
	}

	trace := &Trace{Step: 1, Thought: "first"}
	pctx.AppendTrace(trace)

	trace2 := &Trace{Step: 2, Thought: "second"}
	pctx.AppendTrace(trace2)

	last := pctx.LastTrace()
	if last.Thought != "second" {
		t.Errorf("Expected 'second', got '%s'", last.Thought)
	}
}

func TestPipelineContext_GetSet(t *testing.T) {
	ctx := context.Background()
	pctx := NewPipelineContext(ctx, "test", "input")

	pctx.Set("key1", "value1")
	val, ok := pctx.Get("key1")
	if !ok {
		t.Error("Expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got '%v'", val)
	}

	pctx.Set("key1", nil)
	_, ok = pctx.Get("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}
}

func TestPipelineContext_Get_EmptyState(t *testing.T) {
	ctx := context.Background()
	pctx := NewPipelineContext(ctx, "test", "input")
	pctx.state = nil

	_, ok := pctx.Get("any")
	if ok {
		t.Error("Expected false when state is nil")
	}
}

func TestPipelineContext_ToLLMMessages(t *testing.T) {
	ctx := context.Background()
	pctx := NewPipelineContext(ctx, "test", "input")

	messages := pctx.ToLLMMessages()
	if messages != nil {
		t.Error("Expected nil (deprecated)")
	}
}

func TestTokenUsage_Add(t *testing.T) {
	usage := &TokenUsage{}

	usage.Add(100, 50)
	if usage.PromptTokens != 100 {
		t.Errorf("Expected PromptTokens 100, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("Expected CompletionTokens 50, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens 150, got %d", usage.TotalTokens)
	}

	usage.Add(50, 30)
	if usage.PromptTokens != 150 {
		t.Errorf("Expected PromptTokens 150, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 80 {
		t.Errorf("Expected CompletionTokens 80, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 230 {
		t.Errorf("Expected TotalTokens 230, got %d", usage.TotalTokens)
	}
}

func TestNoopLogger(t *testing.T) {
	logger := DefaultLogger()

	logger.Debug("debug msg", "key", "value")
	logger.Info("info msg", "key", "value")
	logger.Warn("warn msg", "key", "value")
	logger.Error(nil, "error msg", "key", "value")

	child := logger.With("key", "value")
	if child == nil {
		t.Error("Expected child logger")
	}
}

func TestNoopMetrics(t *testing.T) {
	metrics := DefaultMetrics()

	metrics.IncCounter("test", 1.0, nil)
	metrics.RecordTimer("test", time.Second, nil)
	metrics.RecordGauge("test", 1.0, nil)
}