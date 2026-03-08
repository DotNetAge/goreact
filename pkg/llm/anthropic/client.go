package anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client Anthropic客户端
type Client struct {
	apiKey      string
	model       string
	baseURL     string
	timeout     time.Duration
	httpClient  *http.Client
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
		apiKey:     apiKey,
		model:      "claude-3-opus-20240229",
		baseURL:    "https://api.anthropic.com/v1",
		timeout:    60 * time.Second,
		httpClient: &http.Client{},
	}

	for _, opt := range options {
		opt(client)
	}

	client.httpClient.Timeout = client.timeout

	return client
}

// Generate 生成文本
func (c *Client) Generate(prompt string) (string, error) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model":     c.model,
		"prompt":    prompt,
		"max_tokens": 1000,
		"temperature": 0.7,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 构建请求
	req, err := http.NewRequest("POST", c.baseURL+"/completions", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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

	// 解析响应
	var respBody struct {
		Completion string `json:"completion"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return respBody.Completion, nil
}
