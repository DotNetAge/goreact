package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ray/goreact/pkg/llm"
)

const (
	DefaultModel      = "claude-3-opus-20240229"
	DefaultBaseURL    = "https://api.anthropic.com/v1"
	DefaultTimeout    = 60 * time.Second
	DefaultMaxTokens  = 1000
	DefaultAPIVersion = "2023-06-01"
)

// Client Anthropic客户端
type Client struct {
	apiKey     llm.SecureString
	model      string
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

// Option Anthropic客户端选项
type Option func(*Client)

// WithModel 设置模型
func WithModel(model string) Option {
	return func(c *Client) {
		c.model = model
	}
}

// WithBaseURL 设置基础URL
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTimeout 设置超时
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithHTTPClient 设置HTTP客户端
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// NewAnthropicClient 创建新的Anthropic客户端
func NewAnthropicClient(apiKey string, options ...Option) *Client {
	client := &Client{
		apiKey:  llm.NewSecureString(apiKey),
		model:   DefaultModel,
		baseURL: DefaultBaseURL,
		timeout: DefaultTimeout,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
		},
	}

	for _, opt := range options {
		opt(client)
	}

	// 如果用户通过 WithTimeout 修改了超时，同步到 httpClient
	client.httpClient.Timeout = client.timeout

	return client
}

// Generate 生成文本
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	// 构建请求体 - 使用 Messages API 格式
	reqBody := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": DefaultMaxTokens,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 构建请求 - 使用正确的 Messages API 端点
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey.Value())
	req.Header.Set("anthropic-version", DefaultAPIVersion)

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	// 解析响应 - Messages API 格式
	var respBody struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// 提取文本内容
	if len(respBody.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return respBody.Content[0].Text, nil
}
