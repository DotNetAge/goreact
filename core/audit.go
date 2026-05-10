package core

import (
	"context"
	"time"
)

type AuditEventType string

const (
	AuditToolCall       AuditEventType = "tool_call"
	AuditPermissionDeny  AuditEventType = "permission_deny"
	AuditPermissionGrant AuditEventType = "permission_grant"
	AuditSessionCreate   AuditEventType = "session_create"
	AuditSessionEnd      AuditEventType = "session_end"
	AuditAgentSpawn     AuditEventType = "agent_spawn"
	AuditConfigChange   AuditEventType = "config_change"
	AuditError          AuditEventType = "error"
)

type AuditEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	EventType AuditEventType `json:"event_type"`
	SessionID string         `json:"session_id,omitempty"`
	TaskID    string         `json:"task_id,omitempty"`
	Principal string         `json:"principal,omitempty"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource,omitempty"`
	Result    string         `json:"result,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// AuditLogger provides an audit trail for security-sensitive operations.
// Implementations can write to files, databases, or external log services.
type AuditLogger interface {
	Log(ctx context.Context, entry AuditEntry) error
	Close() error
}
