package core

import (
	"context"
	"fmt"
	"sync"
)

// MCPServerConfig holds the connection configuration for an MCP server.
type MCPServerConfig struct {
	// Name is a human-readable identifier for this MCP server.
	Name string `json:"name" yaml:"name"`

	// Transport specifies the transport type: "stdio" or "sse".
	Transport string `json:"transport" yaml:"transport"`

	// Command is the executable to launch for stdio transport.
	Command string `json:"command,omitempty" yaml:"command,omitempty"`

	// Args are arguments passed to the command.
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`

	// URL is the endpoint URL for SSE transport.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// Env are additional environment variables for the command.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

// MCPToolInfo represents a tool discovered from an MCP server.
type MCPToolInfo struct {
	// ServerName identifies which MCP server provides this tool.
	ServerName string `json:"server_name"`

	// ToolInfo is the standard tool metadata converted from MCP schema.
	ToolInfo ToolInfo
}

// MCPClient is the interface for communicating with an MCP server.
// Implementations handle tool discovery and invocation over different transports.
type MCPClient interface {
	// Connect establishes a connection to the MCP server.
	Connect(ctx context.Context) error

	// Disconnect closes the connection and releases resources.
	Disconnect() error

	// ListTools discovers all available tools from the MCP server.
	ListTools(ctx context.Context) ([]MCPToolInfo, error)

	// CallTool invokes a tool on the MCP server by name.
	CallTool(ctx context.Context, toolName string, params map[string]any) (any, error)

	// IsConnected returns whether the client is currently connected.
	IsConnected() bool
}

// MCPToolAdapter wraps an MCP server tool as a standard FuncTool.
// This allows MCP tools to be registered in the reactor's tool registry
// just like built-in tools.
type MCPToolAdapter struct {
	info  *ToolInfo
	client MCPClient
	mcpToolName string
}

// NewMCPToolAdapter creates a FuncTool that delegates execution to an MCP server.
func NewMCPToolAdapter(client MCPClient, serverName string, toolInfo ToolInfo) FuncTool {
	info := toolInfo
	info.IsReadOnly = false // MCP tools may have side effects; assume not read-only
	return &MCPToolAdapter{
		info:       &info,
		client:     client,
		mcpToolName: toolInfo.Name,
	}
}

func (a *MCPToolAdapter) Info() *ToolInfo {
	return a.info
}

func (a *MCPToolAdapter) Execute(ctx context.Context, params map[string]any) (any, error) {
	if !a.client.IsConnected() {
		return nil, fmt.Errorf("MCP server is not connected")
	}
	return a.client.CallTool(ctx, a.mcpToolName, params)
}

// MCPToolRegistry manages multiple MCP server connections and their tools.
type MCPToolRegistry struct {
	clients map[string]MCPClient
	mu      sync.Mutex
}

// NewMCPToolRegistry creates a registry for managing MCP server connections.
func NewMCPToolRegistry() *MCPToolRegistry {
	return &MCPToolRegistry{
		clients: make(map[string]MCPClient),
	}
}

// RegisterClient adds an MCP client under the given server name.
func (r *MCPToolRegistry) RegisterClient(name string, client MCPClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[name] = client
}

// DiscoverTools connects to all registered MCP servers and returns
// their tools as standard FuncTool instances.
func (r *MCPToolRegistry) DiscoverTools(ctx context.Context) ([]FuncTool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var tools []FuncTool
	for name, client := range r.clients {
		if !client.IsConnected() {
			if err := client.Connect(ctx); err != nil {
				return nil, fmt.Errorf("failed to connect to MCP server %q: %w", name, err)
			}
		}
		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tools from MCP server %q: %w", name, err)
		}
		for _, mt := range mcpTools {
			tools = append(tools, NewMCPToolAdapter(client, mt.ServerName, mt.ToolInfo))
		}
	}
	return tools, nil
}

// DisconnectAll closes all MCP server connections.
func (r *MCPToolRegistry) DisconnectAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, client := range r.clients {
		if client.IsConnected() {
			if err := client.Disconnect(); err != nil {
				errs = append(errs, fmt.Errorf("disconnect %q: %w", name, err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d errors during disconnect: %v", len(errs), errs[0])
	}
	return nil
}
