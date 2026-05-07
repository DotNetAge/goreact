package core

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
)

type consoleAuditLogger struct {
	logger *slog.Logger
}

func NewConsoleAuditLogger() AuditLogger {
	return &consoleAuditLogger{
		logger: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

func (l *consoleAuditLogger) Log(ctx context.Context, entry AuditEntry) error {
	data, _ := json.Marshal(entry)
	l.logger.Info("audit", "entry", string(data))
	return nil
}

func (l *consoleAuditLogger) Close() error {
	return nil
}
