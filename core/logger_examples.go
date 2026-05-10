package core

import (
	"fmt"
	"os"
	"time"
)

// Example: How to inject external logger (Zap, Logrus, etc.) into ReAct library
//
// This example demonstrates the unified logging interface design.
// Any logging library that implements the Logger interface can be injected.
//
// Usage in MindX project:
//
//	import "go.uber.org/zap"
//
//	// 1. Create your Zap logger
//	zapLogger, _ := zap.NewProduction()
//	defer zapLogger.Sync()
//
//	// 2. Create adapter implementing core.Logger interface
//	zapAdapter := &ZapAdapter{logger: zapLogger}
//
//	// 3. Inject into ReactorConfig
//	config := reactor.ReactorConfig{
//	    Model: "gpt-4",
//	    Logger: zapAdapter, // ← Inject here!
//	}
//
//	// 4. All logs from ReActor → Tools will now use Zap
//	r := reactor.NewReactor(config)
//
// --- Zap Adapter Example ---

// ZapAdapter adapts uber-go/zap to implement core.Logger interface.
// This is an example of how external logging libraries can integrate.
type ZapAdapter struct {
	logger interface{ // Replace with *zap.Logger in real usage
		Info(msg string, fields ...interface{})
		Warn(msg string, fields ...interface{})
		Error(msg string, fields ...interface{})
		Debug(msg string, fields ...interface{})
	}
}

func (z *ZapAdapter) Info(msg string, keyvals ...any) {
	z.logger.Info(msg, keyvals...)
}

func (z *ZapAdapter) Error(msg string, err error, keyvals ...any) {
	args := append([]any{"error", err}, keyvals...)
	z.logger.Error(msg, args...)
}

func (z *ZapAdapter) Debug(msg string, keyvals ...any) {
	z.logger.Debug(msg, keyvals...)
}

func (z *ZapAdapter) Warn(msg string, keyvals ...any) {
	z.logger.Warn(msg, keyvals...)
}

// --- Logrus Adapter Example ---

// LogrusAdapter adapts sirupsen/logrus to implement core.Logger interface.
type LogrusAdapter struct {
	logger interface{ // Replace with *logrus.Logger in real usage
		Info(args ...interface{})
		Warn(args ...interface{})
		Error(args ...interface{})
		Debug(args ...interface{})
	}
}

func (l *LogrusAdapter) Info(msg string, keyvals ...any) {
	l.logger.Info(append([]any{msg}, keyvals...)...)
}

func (l *LogrusAdapter) Error(msg string, err error, keyvals ...any) {
	args := append([]any{msg, "error", err}, keyvals...)
	l.logger.Error(args...)
}

func (l *LogrusAdapter) Debug(msg string, keyvals ...any) {
	l.logger.Debug(append([]any{msg}, keyvals...)...)
}

func (l *LogrusAdapter) Warn(msg string, keyvals ...any) {
	l.logger.Warn(append([]any{msg}, keyvals...)...)
}

// --- Custom Console Logger Example ---

// ConsoleLogger is a simple colored console logger for development/debugging.
type ConsoleLogger struct {
	level    string // "DEBUG", "INFO", "WARN", "ERROR"
	colorize bool
}

func NewConsoleLogger(level string) *ConsoleLogger {
	return &ConsoleLogger{
		level:    level,
		colorize: true,
	}
}

func (c *ConsoleLogger) shouldLog(level string) bool {
	levels := map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3}
	return levels[level] >= levels[c.level]
}

func (c *ConsoleLogger) Info(msg string, keyvals ...any) {
	if !c.shouldLog("INFO") {
		return
	}
	c.log("INFO", msg, keyvals...)
}

func (c *ConsoleLogger) Error(msg string, err error, keyvals ...any) {
	if !c.shouldLog("ERROR") {
		return
	}
	args := append([]any{"error", err}, keyvals...)
	c.log("ERROR", msg, args)
}

func (c *ConsoleLogger) Debug(msg string, keyvals ...any) {
	if !c.shouldLog("DEBUG") {
		return
	}
	c.log("DEBUG", msg, keyvals...)
}

func (c *ConsoleLogger) Warn(msg string, keyvals ...any) {
	if !c.shouldLog("WARN") {
		return
	}
	c.log("WARN", msg, keyvals...)
}

func (c *ConsoleLogger) log(level, msg string, keyvals ...any) {
	colors := map[string]string{
		"DEBUG": "\033[36m", // Cyan
		"INFO":  "\033[32m", // Green
		"WARN":  "\033[33m", // Yellow
		"ERROR": "\033[31m", // Red
	}
	reset := "\033[0m"

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var fieldStr string
	for i := 0; i+1 < len(keyvals); i += 2 {
		fieldStr += fmt.Sprintf(" %s=%v", keyvals[i], keyvals[i+1])
	}

	if c.colorize {
		fmt.Fprintf(os.Stdout, "%s[%s]%s [%s] %s%s\n", colors[level], level, reset, timestamp, msg, fieldStr)
	} else {
		fmt.Fprintf(os.Stdout, "[%s] [%s] %s%s\n", level, timestamp, msg, fieldStr)
	}
}
