package debug

import (
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/prompt"
)

func TestNewPromptDebugger(t *testing.T) {
	logger := NewSimpleLogger(false)
	d := NewPromptDebugger(true, logger)

	if !d.enabled {
		t.Error("Expected enabled to be true")
	}
	if d.logger == nil {
		t.Error("Expected logger to be set")
	}
	if d.tracker == nil {
		t.Error("Expected tracker to be set")
	}
}

func TestPromptDebugger_LogPrompt_Disabled(t *testing.T) {
	d := NewPromptDebugger(false, nil)
	p := &prompt.Prompt{System: "sys", User: "user"}

	d.LogPrompt(p, nil)
}

func TestPromptDebugger_LogPrompt_WithNilLogger(t *testing.T) {
	d := NewPromptDebugger(true, nil)
	p := &prompt.Prompt{System: "sys", User: "user"}

	d.LogPrompt(p, nil)
}

func TestPromptDebugger_LogPrompt(t *testing.T) {
	logger := NewSimpleLogger(false)
	d := NewPromptDebugger(true, logger)
	p := &prompt.Prompt{System: "system prompt", User: "user prompt"}
	metadata := map[string]any{
		"tools_count":    2,
		"history_turns":   3,
		"few_shots_count": 1,
		"token_counter":   &simpleCounter{},
	}

	d.LogPrompt(p, metadata)

	tracker := d.GetTracker()
	if tracker == nil {
		t.Error("Expected tracker")
	}
	if tracker.TotalTokens == 0 {
		t.Error("Expected total tokens to be set")
	}
}

func TestPromptDebugger_LogBuildTime_Disabled(t *testing.T) {
	d := NewPromptDebugger(false, nil)
	d.LogBuildTime(time.Second)
}

func TestPromptDebugger_LogBuildTime(t *testing.T) {
	logger := NewSimpleLogger(false)
	d := NewPromptDebugger(true, logger)
	d.LogBuildTime(time.Second)
}

func TestPromptDebugger_GetTracker(t *testing.T) {
	logger := NewSimpleLogger(false)
	d := NewPromptDebugger(true, logger)

	tracker := d.GetTracker()
	if tracker == nil {
		t.Error("Expected tracker")
	}
}

func TestPromptDebugger_getCounter(t *testing.T) {
	d := &PromptDebugger{}

	t.Run("from metadata", func(t *testing.T) {
		counter := &simpleCounter{}
		metadata := map[string]any{"token_counter": counter}
		result := d.getCounter(metadata)
		if result != counter {
			t.Error("Expected counter from metadata")
		}
	})

	t.Run("default counter", func(t *testing.T) {
		metadata := map[string]any{}
		result := d.getCounter(metadata)
		if result == nil {
			t.Error("Expected default counter")
		}
	})
}

func TestPromptDebugger_getInt(t *testing.T) {
	d := &PromptDebugger{}

	t.Run("existing key", func(t *testing.T) {
		metadata := map[string]any{"count": 42}
		result := d.getInt(metadata, "count")
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		metadata := map[string]any{}
		result := d.getInt(metadata, "missing")
		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		metadata := map[string]any{"count": "not an int"}
		result := d.getInt(metadata, "count")
		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})
}

func TestPromptDebugger_truncate(t *testing.T) {
	d := &PromptDebugger{}

	if d.truncate("short", 10) != "short" {
		t.Error("Expected 'short' unchanged")
	}

	result := d.truncate("this is a long text", 10)
	if result != "this is a ..." {
		t.Errorf("Expected 'this is a ...', got %q", result)
	}
}

func TestNewTokenTracker(t *testing.T) {
	tracker := NewTokenTracker()
	if tracker == nil {
		t.Error("Expected tracker")
	}
}

func TestTokenTracker_Report(t *testing.T) {
	tracker := &TokenTracker{
		SystemTokens:  100,
		UserTokens:    200,
		HistoryTokens: 50,
		ToolsTokens:   30,
		FewShotsTokens: 20,
		TotalTokens:   400,
	}

	report := tracker.Report()
	if report == "" {
		t.Error("Expected non-empty report")
	}
}

func TestTokenTracker_percentage(t *testing.T) {
	tracker := &TokenTracker{TotalTokens: 100}

	if p := tracker.percentage(25); p != 25.0 {
		t.Errorf("Expected 25.0, got %f", p)
	}

	tracker.TotalTokens = 0
	if p := tracker.percentage(25); p != 0 {
		t.Errorf("Expected 0, got %f", p)
	}
}

func TestSimpleLogger_Info(t *testing.T) {
	logger := NewSimpleLogger(false)
	logger.Info("test message", "key", "value", "another", 123)
}

func TestSimpleLogger_Debug_Enabled(t *testing.T) {
	logger := NewSimpleLogger(true)
	logger.Debug("debug message", "key", "value")
}

func TestSimpleLogger_Debug_Disabled(t *testing.T) {
	logger := NewSimpleLogger(false)
	logger.Debug("debug message", "key", "value")
}

func TestSimpleLogger_IsDebug(t *testing.T) {
	logger := NewSimpleLogger(true)
	if !logger.IsDebug() {
		t.Error("Expected debug to be enabled")
	}

	logger = NewSimpleLogger(false)
	if logger.IsDebug() {
		t.Error("Expected debug to be disabled")
	}
}

func TestSimpleCounter_Count(t *testing.T) {
	counter := &simpleCounter{}
	text := "hellohello" // 10 chars
	count := counter.Count(text)
	if count != 2 { // len/4 = 10/4 = 2 (integer division)
		t.Errorf("Expected 2, got %d", count)
	}
}