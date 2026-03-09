package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ray/goreact/pkg/llm"
)

const (
	DefaultBaseURL     = "http://localhost:11434"
	DefaultModel       = "qwen3:0.6b"
	DefaultTemperature = 0.7
	DefaultTimeout     = 60 * time.Second
)

// OllamaClient Ollama LLM 客户端
type OllamaClient struct {
	baseURL        string
	model          string
	temperature    float64
	httpClient     *http.Client
	lastTokenUsage *llm.TokenUsage
	mu             sync.Mutex
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
		baseURL:     DefaultBaseURL,
		model:       DefaultModel,
		temperature: DefaultTemperature,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range options {
		opt(client)
	}

	return client
}

// Generate 生成文本
func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
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

	// 发送请求（使用带超时的 context）
	url := c.baseURL + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

	// 记录 Token 使用量
	c.mu.Lock()
	c.lastTokenUsage = &llm.TokenUsage{
		PromptTokens:     genResp.PromptEvalCount,
		CompletionTokens: genResp.EvalCount,
		TotalTokens:      genResp.PromptEvalCount + genResp.EvalCount,
	}
	c.mu.Unlock()

	return genResp.Response, nil
}

// LastTokenUsage 返回最近一次调用的 Token 使用量
func (c *OllamaClient) LastTokenUsage() *llm.TokenUsage {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lastTokenUsage == nil {
		return nil
	}

	// 返回副本，避免并发修改
	copy := *c.lastTokenUsage
	return &copy
}

// GetModel 获取当前使用的模型
func (c *OllamaClient) GetModel() string {
	return c.model
}

// GetBaseURL 获取 Ollama 服务地址
func (c *OllamaClient) GetBaseURL() string {
	return c.baseURL
}
