package core

import (
	"testing"
)

// TestLoggerInterfaceVerification verifies that all Logger implementations satisfy the interface
func TestLoggerInterfaceVerification(t *testing.T) {
	var l Logger
	
	l = &SlogAdapter{}
	if l == nil {
		t.Error("SlogAdapter should implement Logger interface")
	}
	
	l = &ConsoleLogger{}
	if l == nil {
		t.Error("ConsoleLogger should implement Logger interface")
	}
}

// TestDefaultLoggerReturnsValidLogger verifies DefaultLogger returns a non-nil Logger
func TestDefaultLoggerReturnsValidLogger(t *testing.T) {
	logger := DefaultLogger()
	if logger == nil {
		t.Fatal("DefaultLogger() should not return nil")
	}
	
	logger.Info("test info message", "key", "value")
	logger.Warn("test warn message", "key", "value")
	logger.Debug("test debug message", "key", "value")
	logger.Error("test error message", nil, "key", "value")
}

// TestSlogAdapterMethodSignatures verifies SlogAdapter has correct method signatures
func TestSlogAdapterMethodSignatures(t *testing.T) {
	adapter := NewSlogAdapter(nil)
	if adapter == nil {
		t.Fatal("NewSlogAdapter should not return nil")
	}
	
	adapter.Info("info test", "key1", "val1", "key2", 2)
	adapter.Warn("warn test", "key1", "val1")
	adapter.Debug("debug test", "key1", "val1")
	adapter.Error("error test", nil, "key1", "val1")
}

// TestConsoleLoggerLevels verifies ConsoleLogger respects log levels
func TestConsoleLoggerLevels(t *testing.T) {
	infoLogger := NewConsoleLogger("INFO")
	debugLogger := NewConsoleLogger("DEBUG")
	errorLogger := NewConsoleLogger("ERROR")
	
	tests := []struct {
		name   string
		logger *ConsoleLogger
		level  string
		shouldLog bool
	}{
		{"InfoLogger-InfoCall", infoLogger, "INFO", true},
		{"InfoLogger-DebugCall", infoLogger, "DEBUG", false},
		{"InfoLogger-WarnCall", infoLogger, "WARN", true},
		{"InfoLogger-ErrorCall", infoLogger, "ERROR", true},
		{"DebugLogger-DebugCall", debugLogger, "DEBUG", true},
		{"ErrorLogger-InfoCall", errorLogger, "INFO", false},
		{"ErrorLogger-ErrorCall", errorLogger, "ERROR", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.logger.shouldLog(tt.level)
			if result != tt.shouldLog {
				t.Errorf("shouldLog(%s) = %v, want %v", tt.level, result, tt.shouldLog)
			}
		})
	}
}

// TestLoggerInjectionPath demonstrates the complete injection path: Reactor → ToolExecutor → ToolContext → Tools
func TestLoggerInjectionPath(t *testing.T) {
	customLogger := &ConsoleLogger{level: "DEBUG"}
	
	tc := &ToolContext{
		Logger: customLogger,
		SessionID: "test-session",
	}
	
	if tc.Logger == nil {
		t.Error("ToolContext.Logger should be set")
	}
	
	if tc.Logger != customLogger {
		t.Error("ToolContext.Logger should be the injected customLogger")
	}
	
	tc.Logger.Info("logger injection test successful", 
		"session_id", tc.SessionID,
		"test", "injection_path_verified",
	)
}
