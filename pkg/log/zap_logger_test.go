package log

import (
	"testing"
)

func TestZapLogger_ImplementsInterface(t *testing.T) {
	logger, err := NewDefaultZapLogger()
	if err != nil {
		t.Skipf("Skipping zap logger tests: %v", err)
	}

	var _ Logger = logger
}

func TestNewZapLogger(t *testing.T) {
	logger, err := NewDefaultZapLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
}

func TestNewDefaultZapLogger(t *testing.T) {
	logger, err := NewDefaultZapLogger()
	if err != nil {
		t.Fatalf("Failed to create default logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}

	logger.Info("test message", String("key", "value"))
}

func TestNewDevelopmentZapLogger(t *testing.T) {
	logger, err := NewDevelopmentZapLogger()
	if err != nil {
		t.Fatalf("Failed to create development logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}

	logger.Debug("debug message", Int("count", 42))
	logger.Info("info message", Float64("rate", 3.14))
	logger.Warn("warn message", Bool("flag", true))
}

func TestZapLogger_LogLevels(t *testing.T) {
	logger, err := NewDefaultZapLogger()
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}

	logger.Debug("debug", String("k", "v"))
	logger.Info("info", Int("num", 123))
	logger.Warn("warn", Err(&testError{msg: "test"}))
	logger.Error("error occurred", Duration("elapsed", 1000))
}

func TestZapLogger_With(t *testing.T) {
	logger, err := NewDefaultZapLogger()
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}

	child := logger.With(String("session", "abc123"))
	if child == nil {
		t.Fatal("Expected non-nil child logger")
	}

	child.Info("message from child")
}

func TestZapLogger_ConvertFields(t *testing.T) {
	logger, err := NewDefaultZapLogger()
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}

	logger.Info("test",
		String("str", "value"),
		Int("int", 42),
		Float64("float", 3.14),
		Bool("bool", true),
		Any("any", []int{1, 2, 3}),
		Err(nil),
		Duration("dur", 1000),
	)
}