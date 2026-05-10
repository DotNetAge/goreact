package core

import (
	"context"
	"encoding/json"
)

type consoleAuditLogger struct {
	logger Logger
}

func NewConsoleAuditLogger() AuditLogger {
	return &consoleAuditLogger{
		logger: DefaultLogger(),
	}
}

func (l *consoleAuditLogger) Log(ctx context.Context, entry AuditEntry) error {
	data, _ := json.Marshal(entry)
	l.logger.Info("audit",
		"entry", string(data),
	)
	return nil
}

func (l *consoleAuditLogger) Close() error {
	return nil
}
