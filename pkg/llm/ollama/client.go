package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient Ollama LLM 客户端
type OllamaClient struct {
	baseURL     string
	model       string
	temperature float64
	httpClient  *http.Client
}

// Option Ollama 客户端配置选项
type Option func(*OllamaClient)

// WithModel 设置模型名称
func WithModel(model string) Option {
	return func(c *OllamaClient) {
		c.model = model
	}
}

// WithTemperature 设置温度参数
func WithTemperature(temp float64) Option {
	return func(c *OllamaClient) {
		c.temperature = temp
	}
}

// WithBaseURL 设置 Ollama 服务地址
func WithBaseURL(url string) Option {
	return func(c *OllamaClient) {
		c.baseURL = url
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *OllamaClient) {
		c.httpClient.Timeout = timeout
	}
}

// NewOllamaClient 创建 Ollama 客户端
func NewOllamaClient(options ...Option) *OllamaClient {
	client := &OllamaClient{
		baseURL:     "http://localhost:11434",
		model:       "qwen3:0.6b",
		temperature: 0.7,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	for _, opt := range options {
		opt(client)
	}

	return client
}

// Generate 生成文本
func (c *OllamaClient) Generate(prompt string) (string, error) {
	// 构建请求
	reqBody := GenerateRequest{
		Model:       c.model,
		Prompt:      prompt,
		Stream:      false,
		Temperature: c.temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	url := c.baseURL + "/api/generate"
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("Ollama error: %s", errResp.Error)
		}
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var genResp GenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !genResp.Done {
		return "", fmt.Errorf("generation not completed")
	}

	return genResp.Response, nil
}

// GetModel 获取当前使用的模型
func (c *OllamaClient) GetModel() string {
	return c.model
}

// GetBaseURL 获取 Ollama 服务地址
func (c *OllamaClient) GetBaseURL() string {
	return c.baseURL
}
