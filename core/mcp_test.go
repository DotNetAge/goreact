package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type mockMCPClient struct {
	connected   atomic.Bool
	tools       []MCPToolInfo
	callErr     error
	connectErr  error
	disconnectErr error
}

func (m *mockMCPClient) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected.Store(true)
	return nil
}

func (m *mockMCPClient) Disconnect() error {
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	m.connected.Store(false)
	return nil
}

func (m *mockMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	return m.tools, nil
}

func (m *mockMCPClient) CallTool(ctx context.Context, toolName string, params map[string]any) (any, error) {
	if m.callErr != nil {
		return nil, m.callErr
	}
	return map[string]any{"result": "ok", "tool": toolName}, nil
}

func (m *mockMCPClient) IsConnected() bool {
	return m.connected.Load()
}

func TestNewMCPToolAdapter(t *testing.T) {
	client := &mockMCPClient{}
	info := ToolInfo{
		Name:        "test_tool",
		Description: "A test MCP tool",
	}

	adapter := NewMCPToolAdapter(client, "test-server", info)
	if adapter.Info().Name != "test_tool" {
		t.Errorf("expected name=test_tool, got %s", adapter.Info().Name)
	}
	if adapter.Info().IsReadOnly {
		t.Error("MCP tools should default to IsReadOnly=false")
	}
}

func TestMCPToolAdapter_Execute_NotConnected(t *testing.T) {
	client := &mockMCPClient{}
	adapter := NewMCPToolAdapter(client, "srv", ToolInfo{Name: "t"})

	_, err := adapter.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestMCPToolAdapter_Execute_Connected(t *testing.T) {
	client := &mockMCPClient{connected: atomic.Bool{}}
	client.connected.Store(true)
	adapter := NewMCPToolAdapter(client, "srv", ToolInfo{Name: "my_tool"})

	result, err := adapter.Execute(context.Background(), map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["tool"] != "my_tool" {
		t.Errorf("expected tool=my_tool, got %v", m["tool"])
	}
}

func TestMCPToolAdapter_Execute_CallError(t *testing.T) {
	client := &mockMCPClient{callErr: errors.New("call failed"), connected: atomic.Bool{}}
	client.connected.Store(true)
	adapter := NewMCPToolAdapter(client, "srv", ToolInfo{Name: "t"})

	_, err := adapter.Execute(context.Background(), nil)
	if err == nil || err.Error() != "call failed" {
		t.Errorf("expected 'call failed' error, got %v", err)
	}
}

func TestMCPToolRegistry_RegisterAndDiscover(t *testing.T) {
	registry := NewMCPToolRegistry()
	client := &mockMCPClient{
		tools: []MCPToolInfo{
			{ServerName: "server1", ToolInfo: ToolInfo{Name: "tool_a", Description: "Tool A"}},
			{ServerName: "server1", ToolInfo: ToolInfo{Name: "tool_b", Description: "Tool B"}},
		},
	}

	registry.RegisterClient("server1", client)

	ctx := context.Background()
	tools, err := registry.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("DiscoverTools failed: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestMCPToolRegistry_DiscoverTools_ConnectsAutomatically(t *testing.T) {
	registry := NewMCPToolRegistry()
	client := &mockMCPClient{
		tools: []MCPToolInfo{
			{ServerName: "s1", ToolInfo: ToolInfo{Name: "t1"}},
		},
	}

	registry.RegisterClient("s1", client)

	if client.IsConnected() {
		t.Error("client should not be connected before DiscoverTools")
	}

	_, err := registry.DiscoverTools(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !client.IsConnected() {
		t.Error("client should be connected after DiscoverTools")
	}
}

func TestMCPToolRegistry_DiscoverTools_ConnectError(t *testing.T) {
	registry := NewMCPToolRegistry()
	client := &mockMCPClient{connectErr: errors.New("connection refused")}
	registry.RegisterClient("bad", client)

	_, err := registry.DiscoverTools(context.Background())
	if err == nil {
		t.Fatal("expected error for failed connection")
	}
}

func TestMCPToolRegistry_DisconnectAll(t *testing.T) {
	registry := NewMCPToolRegistry()
	c1 := &mockMCPClient{connected: atomic.Bool{}}
	c1.connected.Store(true)
	c2 := &mockMCPClient{connected: atomic.Bool{}}
	c2.connected.Store(true)

	registry.RegisterClient("s1", c1)
	registry.RegisterClient("s2", c2)

	err := registry.DisconnectAll()
	if err != nil {
		t.Fatalf("DisconnectAll failed: %v", err)
	}
	if c1.IsConnected() || c2.IsConnected() {
		t.Error("all clients should be disconnected")
	}
}

func TestMCPToolRegistry_DisconnectAll_WithErrors(t *testing.T) {
	registry := NewMCPToolRegistry()
	badClient := &mockMCPClient{
		connected:      atomic.Bool{},
		disconnectErr:  errors.New("disconnect fail"),
	}
	badClient.connected.Store(true)
	registry.RegisterClient("bad", badClient)

	err := registry.DisconnectAll()
	if err == nil {
		t.Fatal("expected error when disconnect fails")
	}
}

func TestMCPServerConfig_Fields(t *testing.T) {
	cfg := MCPServerConfig{
		Name:       "test-server",
		Transport:  "stdio",
		Command:    "/usr/bin/tool",
		Args:       []string{"--port", "3000"},
		URL:        "",
		Env:        map[string]string{"DEBUG": "1"},
	}
	if cfg.Name != "test-server" {
		t.Errorf("expected Name=test-server, got %s", cfg.Name)
	}
	if len(cfg.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(cfg.Args))
	}
}
