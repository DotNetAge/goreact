package memory

import (
	"testing"

	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

func TestSessionOptions(t *testing.T) {
	opts := &SessionOptions{}

	// Test WithMaxTurns
	WithMaxTurns(10)(opts)
	if opts.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, want 10", opts.MaxTurns)
	}

	// Test WithIncludeReflections
	WithIncludeReflections(true)(opts)
	if !opts.IncludeReflections {
		t.Error("IncludeReflections should be true")
	}

	// Test WithIncludePlans
	WithIncludePlans(true)(opts)
	if !opts.IncludePlans {
		t.Error("IncludePlans should be true")
	}
}

func TestSessionHistory(t *testing.T) {
	history := &SessionHistory{
		SessionName: "session-123",
		Messages: []*goreactcore.MessageNode{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		TotalTurns: 2,
		Session: &goreactcore.SessionNode{
			BaseNode: goreactcore.BaseNode{
				Name: "session-123",
			},
			UserName: "user-1",
		},
	}

	if history.SessionName != "session-123" {
		t.Errorf("SessionName = %q, want 'session-123'", history.SessionName)
	}
	if len(history.Messages) != 2 {
		t.Errorf("len(Messages) = %d, want 2", len(history.Messages))
	}
	if history.TotalTurns != 2 {
		t.Errorf("TotalTurns = %d, want 2", history.TotalTurns)
	}
}

func TestBaseAccessor_NodeType(t *testing.T) {
	accessor := &BaseAccessor{
		nodeType: goreactcommon.NodeTypeSession,
	}

	if accessor.nodeType != goreactcommon.NodeTypeSession {
		t.Errorf("nodeType = %q, want 'Session'", accessor.nodeType)
	}
}

func TestNodeToSessionNode(t *testing.T) {
	// Test nil node
	result := nodeToSessionNode(nil)
	if result != nil {
		t.Error("nodeToSessionNode(nil) should return nil")
	}
}

func TestParseTime(t *testing.T) {
	// Test empty string
	result := parseTime("")
	if !result.IsZero() {
		t.Error("parseTime('') should return zero time")
	}

	// Test invalid format
	result = parseTime("invalid")
	if !result.IsZero() {
		t.Error("parseTime('invalid') should return zero time")
	}
}
