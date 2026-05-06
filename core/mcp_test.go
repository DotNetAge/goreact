package core

import (
	"context"
	"testing"
)

func TestMCPClientLifecycle(t *testing.T) {
	client := &mockMCPClient{}

	if client.IsConnected() {
		t.Error("should not be connected initially")
	}

	if err := client.Connect(nil); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if !client.IsConnected() {
		t.Error("should be connected after Connect")
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}

	if client.IsConnected() {
		t.Error("should not be connected after Disconnect")
	}
}

func TestMCPToolAdapter(t *testing.T) {
	client := &mockMCPClient{
		connected: true,
		tools: []MCPToolInfo{{
			ServerName: "test-server",
			ToolInfo: ToolInfo{
				Name:        "weather",
				Description: "Get weather",
			},
		}},
	}

	toolInfo := ToolInfo{
		Name:        "weather",
		Description: "Get weather",
	}

	adapter := NewMCPToolAdapter(client, "test-server", toolInfo)

	info := adapter.Info()
	if info.Name != "weather" {
		t.Errorf("expected name 'weather', got %q", info.Name)
	}

	_, err := adapter.Execute(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.callLog) != 1 {
		t.Errorf("expected 1 call, got %d", len(client.callLog))
	}
}

func TestMCPAdapter_NotConnected(t *testing.T) {
	client := &mockMCPClient{connected: false}
	toolInfo := ToolInfo{Name: "test", Description: "test"}
	adapter := NewMCPToolAdapter(client, "test", toolInfo)

	_, err := adapter.Execute(nil, nil)
	if err == nil {
		t.Fatal("expected error when MCP server is not connected")
	}
}

func TestMCPRegistry_DiscoverTools(t *testing.T) {
	registry := NewMCPToolRegistry()

	client := &mockMCPClient{
		connected: true,
		tools: []MCPToolInfo{
			{ServerName: "s1", ToolInfo: ToolInfo{Name: "tool1", Description: "desc1"}},
			{ServerName: "s1", ToolInfo: ToolInfo{Name: "tool2", Description: "desc2"}},
		},
	}
	registry.RegisterClient("s1", client)

	ctx := context.Background()
	tools, err := registry.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("DiscoverTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestMCPRegistry_DisconnectAll(t *testing.T) {
	registry := NewMCPToolRegistry()

	client := &mockMCPClient{connected: true}
	registry.RegisterClient("s1", client)

	err := registry.DisconnectAll()
	if err != nil {
		t.Fatalf("DisconnectAll failed: %v", err)
	}

	if client.IsConnected() {
		t.Error("should be disconnected after DisconnectAll")
	}
}

func TestMCPRegistry_EmptyRegistry(t *testing.T) {
	registry := NewMCPToolRegistry()

	ctx := context.Background()
	tools, err := registry.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("DiscoverTools failed: %v", err)
	}

	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}

	err = registry.DisconnectAll()
	if err != nil {
		t.Fatalf("DisconnectAll failed on empty registry: %v", err)
	}
}

type mockMCPClient struct {
	connected bool
	tools     []MCPToolInfo
	callLog   []struct {
		name   string
		params map[string]any
	}
}

func (c *mockMCPClient) Connect(ctx context.Context) error {
	c.connected = true
	return nil
}

func (c *mockMCPClient) Disconnect() error {
	c.connected = false
	return nil
}

func (c *mockMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	return c.tools, nil
}

func (c *mockMCPClient) CallTool(ctx context.Context, toolName string, params map[string]any) (any, error) {
	c.callLog = append(c.callLog, struct {
		name   string
		params map[string]any
	}{toolName, params})
	return "mock result", nil
}

func (c *mockMCPClient) IsConnected() bool {
	return c.connected
}
