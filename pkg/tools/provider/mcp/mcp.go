package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"context"

	"github.com/DotNetAge/goreact/pkg/tools"
)

// MCPProvider Model Context Protocol 提供者
// 实现 MCP 协议以集成外部工具服务器
type MCPProvider struct {
	name       string
	serverURL  string
	apiKey     string
	httpClient *http.Client
	tools      map[string]*MCPTool
	healthy    bool
}

// MCPTool MCP 工具包装器
type MCPTool struct {
	name        string
	description string
	schema      map[string]any
	provider    *MCPProvider
}

// MCPConfig MCP 提供者配置
type MCPConfig struct {
	ServerURL string `json:"server_url"`
	APIKey    string `json:"api_key"`
	Timeout   int    `json:"timeout"` // 秒
}

// NewMCPProvider 创建新的 MCP 提供者
func NewMCPProvider(name string) *MCPProvider {
	return &MCPProvider{
		name:  name,
		tools: make(map[string]*MCPTool),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name 返回提供者名称
func (p *MCPProvider) Name() string {
	return p.name
}

// Initialize 初始化 MCP 提供者
func (p *MCPProvider) Initialize(config map[string]any) error {
	// 解析配置
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	var mcpConfig MCPConfig
	if err := json.Unmarshal(configJSON, &mcpConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if mcpConfig.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}

	p.serverURL = mcpConfig.ServerURL
	p.apiKey = mcpConfig.APIKey

	if mcpConfig.Timeout > 0 {
		p.httpClient.Timeout = time.Duration(mcpConfig.Timeout) * time.Second
	}

	// 测试连接
	if err := p.ping(); err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	p.healthy = true
	return nil
}

// ping 测试 MCP 服务器连接
func (p *MCPProvider) ping() error {
	req, err := http.NewRequest("GET", p.serverURL+"/health", nil)
	if err != nil {
		return err
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// DiscoverTools 从 MCP 服务器发现工具
func (p *MCPProvider) DiscoverTools() ([]tools.Tool, error) {
	req, err := http.NewRequest("GET", p.serverURL+"/tools", nil)
	if err != nil {
		return nil, err
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.healthy = false
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.healthy = false
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析工具列表
	var toolsResponse struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Schema      map[string]any `json:"schema"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(body, &toolsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse tools response: %w", err)
	}

	// 创建工具包装器
	toolsList := make([]tools.Tool, 0, len(toolsResponse.Tools))
	for _, t := range toolsResponse.Tools {
		mcpTool := &MCPTool{
			name:        t.Name,
			description: t.Description,
			schema:      t.Schema,
			provider:    p,
		}
		p.tools[t.Name] = mcpTool
		toolsList = append(toolsList, mcpTool)
	}

	return toolsList, nil
}

// GetTool 获取指定的工具
func (p *MCPProvider) GetTool(name string) (tools.Tool, error) {
	t, exists := p.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return t, nil
}

// Close 关闭 MCP 提供者
func (p *MCPProvider) Close() error {
	p.healthy = false
	p.httpClient.CloseIdleConnections()
	return nil
}

// IsHealthy 检查提供者健康状态
func (p *MCPProvider) IsHealthy() bool {
	return p.healthy
}

// Name 返回工具名称
func (t *MCPTool) Name() string {
	return t.name
}

// Description 返回工具描述
// SecurityLevel returns the tool's security risk level
func (t *MCPTool) SecurityLevel() tools.SecurityLevel {
	return tools.LevelHighRisk // MCP tools are external, treat as high risk by default
}

func (t *MCPTool) Description() string {
	return t.description
}

// Execute 执行 MCP 工具
func (t *MCPTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	// 构建请求
	requestBody := map[string]any{
		"tool":   t.name,
		"params": params,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", t.provider.serverURL+"/execute", bytes.NewBuffer(bodyJSON))
	if err != nil {
		return nil, err
	}

	if t.provider.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.provider.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.provider.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var result struct {
		Success bool   `json:"success"`
		Result  any    `json:"result"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("tool execution failed: %s", result.Error)
	}

	return result.Result, nil
}
