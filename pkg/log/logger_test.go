package log

import (
	"testing"
)

func TestString(t *testing.T) {
	f := String("key", "value")
	if f.Key != "key" {
		t.Errorf("Expected 'key', got %q", f.Key)
	}
	if f.Value != "value" {
		t.Errorf("Expected 'value', got %v", f.Value)
	}
}

func TestInt(t *testing.T) {
	f := Int("count", 42)
	if f.Key != "count" {
		t.Errorf("Expected 'count', got %q", f.Key)
	}
	if f.Value != 42 {
		t.Errorf("Expected 42, got %v", f.Value)
	}
}

func TestFloat64(t *testing.T) {
	f := Float64("rate", 3.14)
	if f.Key != "rate" {
		t.Errorf("Expected 'rate', got %q", f.Key)
	}
	if f.Value != 3.14 {
		t.Errorf("Expected 3.14, got %v", f.Value)
	}
}

func TestBool(t *testing.T) {
	f := Bool("enabled", true)
	if f.Key != "enabled" {
		t.Errorf("Expected 'enabled', got %q", f.Key)
	}
	if f.Value != true {
		t.Errorf("Expected true, got %v", f.Value)
	}
}

func TestAny(t *testing.T) {
	f := Any("data", []int{1, 2, 3})
	if f.Key != "data" {
		t.Errorf("Expected 'data', got %q", f.Key)
	}
}

func TestErr(t *testing.T) {
	err := &testError{msg: "test error"}
	f := Err(err)
	if f.Key != "error" {
		t.Errorf("Expected 'error', got %q", f.Key)
	}
	if f.Value != err {
		t.Errorf("Expected error, got %v", f.Value)
	}
}

func TestDuration(t *testing.T) {
	f := Duration("elapsed", 1000)
	if f.Key != "elapsed" {
		t.Errorf("Expected 'elapsed', got %q", f.Key)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}