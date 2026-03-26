package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/DotNetAge/goreact/pkg/tools"
)

type mockProvider struct {
	name      string
	healthy   bool
	tools     []tools.Tool
	err      error
	closeErr  error
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Initialize(config map[string]any) error {
	return m.err
}

func (m *mockProvider) DiscoverTools() ([]tools.Tool, error) {
	return m.tools, m.err
}

func (m *mockProvider) GetTool(name string) (tools.Tool, error) {
	return nil, m.err
}

func (m *mockProvider) Close() error {
	return m.closeErr
}

func (m *mockProvider) IsHealthy() bool {
	return m.healthy
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("Expected non-nil registry")
	}
	if r.providers == nil {
		t.Error("Expected providers map to be initialized")
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		r := NewRegistry()
		provider := &mockProvider{name: "test", healthy: true}
		err := r.Register(provider)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("nil provider", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(nil)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		r := NewRegistry()
		provider := &mockProvider{name: "", healthy: true}
		err := r.Register(provider)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		r := NewRegistry()
		provider := &mockProvider{name: "test", healthy: true}
		r.Register(provider)
		err := r.Register(provider)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Run("existing provider", func(t *testing.T) {
		r := NewRegistry()
		provider := &mockProvider{name: "test", healthy: true}
		r.Register(provider)

		retrieved, err := r.Get("test")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retrieved.Name() != "test" {
			t.Errorf("Expected 'test', got %q", retrieved.Name())
		}
	})

	t.Run("non-existing provider", func(t *testing.T) {
		r := NewRegistry()
		_, err := r.Get("nonexistent")
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "a", healthy: true})
	r.Register(&mockProvider{name: "b", healthy: true})

	names := r.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(names))
	}
}

func TestRegistry_DiscoverAllTools(t *testing.T) {
	t.Run("no providers", func(t *testing.T) {
		r := NewRegistry()
		allTools, err := r.DiscoverAllTools()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(allTools) != 0 {
			t.Errorf("Expected 0 tools, got %d", len(allTools))
		}
	})

	t.Run("healthy provider returns tools", func(t *testing.T) {
		r := NewRegistry()
		tool := &mockTool{name: "test"}
		provider := &mockProvider{name: "test", healthy: true, tools: []tools.Tool{tool}}
		r.Register(provider)

		allTools, err := r.DiscoverAllTools()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(allTools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(allTools))
		}
	})

	t.Run("unhealthy provider skipped", func(t *testing.T) {
		r := NewRegistry()
		provider := &mockProvider{name: "test", healthy: false}
		r.Register(provider)

		allTools, err := r.DiscoverAllTools()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(allTools) != 0 {
			t.Errorf("Expected 0 tools, got %d", len(allTools))
		}
	})

	t.Run("provider with error continues", func(t *testing.T) {
		r := NewRegistry()
		provider := &mockProvider{name: "test", healthy: true, err: errors.New("discover error")}
		r.Register(provider)

		allTools, err := r.DiscoverAllTools()
		if err != nil {
			t.Errorf("Expected no error (error logged), got %v", err)
		}
		if len(allTools) != 0 {
			t.Errorf("Expected 0 tools, got %d", len(allTools))
		}
	})
}

func TestRegistry_Close(t *testing.T) {
	t.Run("no providers", func(t *testing.T) {
		r := NewRegistry()
		err := r.Close()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("single provider closes", func(t *testing.T) {
		r := NewRegistry()
		provider := &mockProvider{name: "test", closeErr: nil}
		r.Register(provider)

		err := r.Close()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("multiple providers last error returned", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockProvider{name: "a", closeErr: nil})
		r.Register(&mockProvider{name: "b", closeErr: errors.New("close error")})

		err := r.Close()
		if err == nil {
			t.Error("Expected error")
		}
	})
}

type mockTool struct {
	name string
}

func (m *mockTool) Name() string                                         { return m.name }
func (m *mockTool) Description() string                                  { return "mock tool" }
func (m *mockTool) SecurityLevel() tools.SecurityLevel                   { return tools.LevelSafe }
func (m *mockTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	return nil, nil
}