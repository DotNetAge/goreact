package model

import (
	"fmt"
	"strings"
)

const (
	DefaultTemperature = 0.7
	DefaultMaxTokens   = 4096
	DefaultTimeout     = 30
)

// ModelFeatures 描述模型支持的特性能力
type ModelFeatures struct {
	Vision         bool // 支持图像输入
	Audio          bool // 支持音频输入
	Video          bool // 支持视频输入
	ToolCalling    bool // 支持原生 Function/Tool Calling
	Streaming      bool // 支持流式响应
	Thinking       bool // 支持深度思考（如 DeepSeek R1, Qwen Reasoning）
	FileAttachment bool // 支持直接阅读上传的文件内容
}

// Model 模型配置（纯配置,不持有运行时对象）
// 包含调用 LLM 所需的所有配置参数
type Model struct {
	Name        string            // 模型名称（唯一标识）
	Provider    string            // 提供商（openai, anthropic, ollama）
	ModelID     string            // 模型 ID（gpt-4, claude-3-opus, qwen3:8b）
	APIKey      string            // API 密钥（可选，Ollama 不需要）
	BaseURL     string            // API 基础 URL（可选）
	Temperature float64           // 温度参数
	MaxTokens   int               // 最大 token 数
	Timeout     int               // 超时时间（秒）
	Features    ModelFeatures     // 模型能力标识（支持什么功能）
	Metadata    map[string]string // 其他元数据
}

// NewModel 创建新的模型配置
func NewModel(name, provider, modelID string) (*Model, error) {
	// 验证必需参数
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}
	if strings.TrimSpace(provider) == "" {
		return nil, fmt.Errorf("provider cannot be empty")
	}
	if strings.TrimSpace(modelID) == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	return &Model{
		Name:        name,
		Provider:    provider,
		ModelID:     modelID,
		Temperature: DefaultTemperature,
		MaxTokens:   DefaultMaxTokens,
		Timeout:     DefaultTimeout,
		Metadata:    make(map[string]string),
	}, nil
}

// WithAPIKey 设置 API 密钥
func (m *Model) WithAPIKey(apiKey string) *Model {
	m.APIKey = apiKey
	return m
}

// WithBaseURL 设置基础 URL
func (m *Model) WithBaseURL(baseURL string) *Model {
	m.BaseURL = baseURL
	return m
}

// WithFeatureVision 启用或禁用视觉支持
func (m *Model) WithFeatureVision(supports bool) *Model {
	m.Features.Vision = supports
	return m
}

// WithFeatureToolCalling 启用或禁用原生工具调用支持
func (m *Model) WithFeatureToolCalling(supports bool) *Model {
	m.Features.ToolCalling = supports
	return m
}

// WithFeatureStreaming 启用或禁用流式输出
func (m *Model) WithFeatureStreaming(supports bool) *Model {
	m.Features.Streaming = supports
	return m
}

// WithFeatureThinking 启用或禁用深度思考能力
func (m *Model) WithFeatureThinking(supports bool) *Model {
	m.Features.Thinking = supports
	return m
}

// WithFeatureFileAttachment 启用或禁用文件附件阅读能力
func (m *Model) WithFeatureFileAttachment(supports bool) *Model {
	m.Features.FileAttachment = supports
	return m
}

// WithTemperature 设置温度
func (m *Model) WithTemperature(temperature float64) (*Model, error) {
	if temperature < 0.0 || temperature > 2.0 {
		return nil, fmt.Errorf("temperature must be between 0.0 and 2.0, got: %f", temperature)
	}
	m.Temperature = temperature
	return m, nil
}

// WithMaxTokens 设置最大 token 数
func (m *Model) WithMaxTokens(maxTokens int) (*Model, error) {
	if maxTokens <= 0 {
		return nil, fmt.Errorf("max tokens must be positive, got: %d", maxTokens)
	}
	m.MaxTokens = maxTokens
	return m, nil
}

// WithTimeout 设置超时时间
func (m *Model) WithTimeout(timeout int) (*Model, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive, got: %d", timeout)
	}
	m.Timeout = timeout
	return m, nil
}
