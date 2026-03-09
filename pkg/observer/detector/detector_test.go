package detector

import (
	"testing"
)

func TestLoopDetectorNoLoop(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(2))

	pattern := ld.Record("http", map[string]any{"url": "a"}, false)
	if pattern.Detected {
		t.Error("should not detect loop on first call")
	}
}

func TestLoopDetectorRepeatedFailure(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(2))

	ld.Record("http", map[string]any{"url": "https://api.example.com"}, false)
	pattern := ld.Record("http", map[string]any{"url": "https://api.example.com"}, false)

	if !pattern.Detected {
		t.Error("should detect loop on repeated failure")
	}
	if pattern.Type != "repeated_failure" {
		t.Errorf("expected type 'repeated_failure', got '%s'", pattern.Type)
	}
	if pattern.RepeatCount != 2 {
		t.Errorf("expected repeat count 2, got %d", pattern.RepeatCount)
	}
	if pattern.Suggestion == "" {
		t.Error("should have suggestion")
	}
}

func TestLoopDetectorDifferentParams(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(2))

	ld.Record("http", map[string]any{"url": "a"}, false)
	pattern := ld.Record("http", map[string]any{"url": "b"}, false)

	if pattern.Detected {
		t.Error("different params should not trigger loop detection")
	}
}

func TestLoopDetectorDifferentTools(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(2))

	ld.Record("http", map[string]any{"url": "a"}, false)
	pattern := ld.Record("bash", map[string]any{"url": "a"}, false)

	if pattern.Detected {
		t.Error("different tools should not trigger loop detection")
	}
}

func TestLoopDetectorSuccessNotDetected(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(2))

	ld.Record("calculator", map[string]any{"a": 1}, true)
	pattern := ld.Record("calculator", map[string]any{"a": 1}, true)

	if pattern.Detected {
		t.Error("repeated success should not trigger loop detection")
	}
}

func TestLoopDetectorThreeRepeats(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(3))

	ld.Record("http", map[string]any{"url": "a"}, false)
	pattern := ld.Record("http", map[string]any{"url": "a"}, false)
	if pattern.Detected {
		t.Error("should not detect at 2 repeats when max is 3")
	}

	pattern = ld.Record("http", map[string]any{"url": "a"}, false)
	if !pattern.Detected {
		t.Error("should detect at 3 repeats")
	}
	if pattern.RepeatCount != 3 {
		t.Errorf("expected repeat count 3, got %d", pattern.RepeatCount)
	}
}

func TestLoopDetectorWindowSize(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(2), WithWindowSize(3))

	// 填满窗口
	ld.Record("http", map[string]any{"url": "a"}, false)
	ld.Record("bash", map[string]any{"cmd": "ls"}, true)
	ld.Record("search", map[string]any{"q": "test"}, true)

	// 第一次 "a" 已经滑出窗口
	pattern := ld.Record("http", map[string]any{"url": "a"}, false)
	if pattern.Detected {
		t.Error("should not detect when first occurrence is outside window")
	}
}

func TestLoopDetectorClear(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(2))

	ld.Record("http", map[string]any{"url": "a"}, false)
	ld.Clear()
	pattern := ld.Record("http", map[string]any{"url": "a"}, false)

	if pattern.Detected {
		t.Error("should not detect after clear")
	}
}

func TestNoActionDetection(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(3))

	ld.RecordNoAction()
	ld.RecordNoAction()
	pattern := ld.RecordNoAction()

	if !pattern.Detected {
		t.Error("should detect no-action stagnation")
	}
	if pattern.Type != "no_action" {
		t.Errorf("expected type 'no_action', got '%s'", pattern.Type)
	}
}

func TestNoActionReset(t *testing.T) {
	ld := NewLoopDetector(WithMaxRepeats(3))

	ld.RecordNoAction()
	ld.RecordNoAction()
	// 一次正常操作打断连续无行动
	ld.Record("calculator", map[string]any{"a": 1}, true)
	pattern := ld.RecordNoAction()

	if pattern.Detected {
		t.Error("should not detect after action breaks the streak")
	}
}
