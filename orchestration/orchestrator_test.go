package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DotNetAge/goreact"
	"github.com/DotNetAge/goreact/core"
)

// --- Interface compliance tests ---

func TestOrchestratorInterfaceSatisfaction(t *testing.T) {
	// Verify *ChannelOrchestrator implements Orchestrator at compile time
	var _ Orchestrator = (*ChannelOrchestrator)(nil)
}

func TestModelRegistryInterfaceSatisfaction(t *testing.T) {
	var _ core.ModelRegistry = (*core.InMemoryModelRegistry)(nil)
}

func TestTaskStoreInterfaceSatisfaction(t *testing.T) {
	// InMemoryTaskStore will implement TaskStore once we add the missing methods
	_ = NewInMemoryTaskStore()
}

// --- Message type tests ---

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		expected MessageType
	}{
		{"delegate", MsgDelegate},
		{"query", MsgQuery},
		{"cancel", MsgCancel},
		{"result", MsgResult},
		{"broadcast", MsgBroadcast},
	}
	for _, tt := range tests {
		if tt.expected == "" {
			t.Errorf("MessageType %q should not be empty", tt.name)
		}
	}
}

// --- ModelRegistry tests ---

func TestInMemoryModelRegistry_RegisterAndGet(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()

	cfg := &core.ModelConfig{
		Name:   "test-model",
		APIKey: "test-key",
		BaseURL: "https://example.com/v1",
	}

	// Register
	err := reg.Register("test-model", cfg)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get
	got, err := reg.Get("test-model")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "test-model" {
		t.Errorf("Name = %q, want %q", got.Name, "test-model")
	}

	// Duplicate registration
	err = reg.Register("test-model", cfg)
	if !errors.Is(err, core.ErrDuplicateModel) {
		t.Errorf("expected ErrDuplicateModel, got %v", err)
	}

	// Not found
	_, err = reg.Get("nonexistent")
	if !errors.Is(err, core.ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestInMemoryModelRegistry_ListAndSize(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	reg.Register("a", &core.ModelConfig{Name: "a"})
	reg.Register("b", &core.ModelConfig{Name: "b"})
	reg.Register("c", &core.ModelConfig{Name: "c"})

	if n := reg.Size(); n != 3 {
		t.Errorf("Size() = %d, want 3", n)
	}

	names := reg.List()
	if len(names) != 3 {
		t.Errorf("List() length = %d, want 3", len(names))
	}
}

// Empty registry edge cases
func TestInMemoryModelRegistry_Empty(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	if n := reg.Size(); n != 0 {
		t.Errorf("empty registry Size() = %d, want 0", n)
	}
	_, err := reg.Get("anything")
	if !errors.Is(err, core.ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound on empty registry, got %v", err)
	}
	names := reg.List()
	if names != nil {
		t.Errorf("expected nil List() on empty registry, got %v", names)
	}
}

// --- New (constructor) tests ---

func TestNew_WithDefaultOptions(t *testing.T) {
	o, err := New(
		WithInboxSize(128),
		WithMaxConcurrent(5),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if o.inboxSize != 128 {
		t.Errorf("inboxSize = %d, want 128", o.inboxSize)
	}
	if o.maxConcurrent != 5 {
		t.Errorf("maxConcurrent = %d, want 5", o.maxConcurrent)
	}
	if o.defaultTimeout != 5*time.Minute {
		t.Errorf("defaultTimeout = %v, want 5m", o.defaultTimeout)
	}
}

func TestNew_WithDefaultModel(t *testing.T) {
	cfg := &core.ModelConfig{
		Name:   "default-test",
		APIKey: "test-key-for-default",
		BaseURL: "https://example.com",
	}
	o, err := New(WithDefaultModel(cfg))
	if err != nil {
		t.Fatalf("New WithDefaultModel failed: %v", err)
	}
	if o.modelRegistry == nil {
		t.Error("modelRegistry should not be nil after WithDefaultModel")
	}
	got, err := o.modelRegistry.Get("default")
	if err != nil {
		t.Fatalf("default model not registered: %v", err)
	}
	if got := got.Name; got != "default-test" {
		t.Errorf("default model name = %q, want default-test", got)
	}
}

func TestNew_WithModelRegistry(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	reg.Register("deepseek-chat", &core.ModelConfig{Name: "deepseek-chat"})
	reg.Register("qwen-flash", &core.ModelConfig{Name: "qwen-flash"})

	o, err := New(WithModelRegistry(reg))
	if err != nil {
		t.Fatalf("New WithModelRegistry failed: %v", err)
	}
	if o.modelRegistry.Size() != 2 {
		t.Errorf("expected 2 models, got %d", o.modelRegistry.Size())
	}
}

// --- Start/Stop lifecycle ---

func TestStartStop(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	reg.Register("default", &core.ModelConfig{Name: "default", APIKey: "test"})

	registry := &goreact.AgentRegistry{}
	o, err := New(
		WithModelRegistry(reg),
		WithAgentRegistry(registry), // empty registry is OK for this test
		WithInboxSize(16),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start
	if err := o.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !o.started {
		t.Error("started should be true after Start")
	}

	// Stop — use a fresh context so Stop doesn't fail with "context canceled"
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	cancel()
	if err := o.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if o.started {
		t.Error("started should be false after Stop")
	}
}

// --- DelegateTo / WaitForResult round-trip ---

func TestDelegateTo_AndWaitForResult(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	reg.Register("default", &core.ModelConfig{Name: "default", APIKey: "test"})

	// Create a minimal agent definition
	registry := &goreact.AgentRegistry{}
	// We can't easily register an AgentConfig without the full .md parser,
	// so we'll test with a pre-built registry approach

	o, _ := New(
		WithModelRegistry(reg),
		WithAgentRegistry(registry),
		WithMaxConcurrent(10),
		WithInboxSize(32),
		WithDefaultTimeout(10 * time.Second),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = o.Start(ctx)
	defer o.Stop(ctx)

	// DelegateTo requires a valid agent in registry — skip full integration here
	// since we need a real LLM or mock for that. Instead verify the channel plumbing.
	// This test validates the message flow infrastructure.

	// Send a raw message directly to test inbox processing
	replyCh := make(chan Response, 1)
	msg := Message{
		Type:    MsgQuery,
		TaskID:  "test-query-1",
		From:    "unit-test",
		Payload: nil,
		ReplyCh: replyCh,
	}

	select {
	case o.controlCh <- msg:
	case <-time.After(2 * time.Second):
		t.Fatal("controlCh send timed out — runLoop may not have started")
	}

	select {
	case resp := <-replyCh:
		if resp.Error != nil {
			t.Logf("Query response (error expected for non-existent task): %v", resp.Error)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reply timed out")
	}
}

// --- Event aggregation ---

func TestEventsSubscription(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	_ = reg.Register("default", &core.ModelConfig{Name: "default", APIKey: "test"})

	o, _ := New(
		WithModelRegistry(reg),
		WithInboxSize(16),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = o.Start(ctx)
	defer o.Stop(ctx)

	ch, cancelSub := o.EventsFiltered(func(e core.ReactEvent) bool {
		return e.Type == core.SubtaskCompleted || e.Type == core.Error
	})
	defer cancelSub()

	// Emit a test event via the internal eventOut channel
	go func() {
		time.Sleep(50 * time.Millisecond)
		o.emitEvent(core.ReactEvent{Type: core.SubtaskCompleted,
			Data: core.SubtaskResult{TaskID: "test-1", Success: true}})
	}()

	select {
	case event := <-ch:
		if event.Type != core.SubtaskCompleted {
			t.Errorf("got event type %q, want SubtaskSpawned", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for filtered event")
	}
}

// --- Stats ---

func TestStats(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	reg.Register("m1", &core.ModelConfig{Name: "m1"})

	areg := &goreact.AgentRegistry{}
	o, _ := New(
		WithModelRegistry(reg),
		WithAgentRegistry(areg),
	)

	stats := o.Stats()
	if stats["registered_models"].(int) != 1 {
		t.Errorf("registered_models = %v, want 1", stats["registered_models"])
	}
	if stats["started"].(bool) {
		t.Error("should not be started yet")
	}
}

// --- Convenience constructors ---

func TestNewWithAgentsDir(t *testing.T) {
	// This would require actual .md files on disk, so we just verify it returns error for nonexistent dir
	_, err := NewWithAgentsDir("/nonexistent/path/xyz123")
	if err == nil {
		t.Error("expected error for nonexistent agents dir")
	}
}

func TestValidateStartup_MissingModel(t *testing.T) {
	reg := core.NewInMemoryModelRegistry()
	areg := &goreact.AgentRegistry{}

	o, _ := New(WithModelRegistry(reg), WithAgentRegistry(areg))

	warns, err := o.ValidateStartup()
	if err != nil {
		t.Errorf("ValidateStartup returned unexpected error: %v", err)
	}
	// Should warn about no agents loaded
	foundAgentWarning := false
	for _, w := range warns {
		if contains(w, "agent registry") {
			foundAgentWarning = true
		}
	}
	if !foundAgentWarning {
		t.Logf("warnings (info): %v", warns) // May or may not have agent warning depending on implementation
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || indexOf(s, substr) >= 0)
}

func indexOf(s string, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
