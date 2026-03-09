package model

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/llm/anthropic"
	"github.com/ray/goreact/pkg/llm/ollama"
	"github.com/ray/goreact/pkg/llm/openai"
)

const (
	DefaultOllamaBaseURL = "http://localhost:11434"
)

// Manager 模型管理器
type Manager struct {
	models map[string]*Model
}

// NewManager 创建模型管理器
func NewManager() *Manager {
	return &Manager{
		models: make(map[string]*Model),
	}
}

// RegisterModel 注册模型配置
func (m *Manager) RegisterModel(model *Model) error {
	if err := validateModel(model); err != nil {
		return fmt.Errorf("invalid model: %w", err)
	}
	m.models[model.Name] = model
	return nil
}

// validateModel 验证模型配置
func validateModel(model *Model) error {
	if model == nil {
		return fmt.Errorf("model cannot be nil")
	}

	// 验证名称
	if strings.TrimSpace(model.Name) == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// 验证提供商
	validProviders := map[string]bool{
		"openai":    true,
		"anthropic": true,
		"ollama":    true,
	}
	if !validProviders[model.Provider] {
		return fmt.Errorf("invalid provider: %s (must be one of: openai, anthropic, ollama)", model.Provider)
	}

	// 验证模型 ID
	if strings.TrimSpace(model.ModelID) == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	// 验证温度参数（通常在 0.0 到 2.0 之间）
	if model.Temperature < 0.0 || model.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0, got: %f", model.Temperature)
	}

	// 验证最大 token 数
	if model.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive, got: %d", model.MaxTokens)
	}

	// 验证超时时间
	if model.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got: %d", model.Timeout)
	}

	// 验证 BaseURL 格式（如果提供）
	if model.BaseURL != "" {
		if _, err := url.Parse(model.BaseURL); err != nil {
			return fmt.Errorf("invalid base URL: %w", err)
		}
	}

	// 验证 API Key（OpenAI 和 Anthropic 需要）
	if (model.Provider == "openai" || model.Provider == "anthropic") && strings.TrimSpace(model.APIKey) == "" {
		return fmt.Errorf("%s provider requires API key", model.Provider)
	}

	return nil
}

// GetModel 获取模型配置
func (m *Manager) GetModel(name string) (*Model, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	model, exists := m.models[name]
	if !exists {
		return nil, fmt.Errorf("model not found: %s", name)
	}
	return model, nil
}

// ListModels 列出所有模型
func (m *Manager) ListModels() []*Model {
	models := make([]*Model, 0, len(m.models))
	for _, model := range m.models {
		models = append(models, model)
	}
	return models
}

// CreateLLMClient 根据模型配置创建 LLM 客户端
func (m *Manager) CreateLLMClient(modelName string) (llm.Client, error) {
	model, err := m.GetModel(modelName)
	if err != nil {
		return nil, err
	}

	return m.createClientFromModel(model)
}

// createClientFromModel 根据模型配置创建客户端
func (m *Manager) createClientFromModel(model *Model) (llm.Client, error) {
	switch model.Provider {
	case "openai":
		return m.createOpenAIClient(model)
	case "anthropic":
		return m.createAnthropicClient(model)
	case "ollama":
		return m.createOllamaClient(model)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}

// createOpenAIClient 创建 OpenAI 客户端
func (m *Manager) createOpenAIClient(model *Model) (llm.Client, error) {
	if model.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	opts := []openai.Option{
		openai.WithModel(model.ModelID),
	}

	if model.BaseURL != "" {
		opts = append(opts, openai.WithBaseURL(model.BaseURL))
	}

	if model.Timeout > 0 {
		opts = append(opts, openai.WithTimeout(time.Duration(model.Timeout)*time.Second))
	}

	return openai.NewOpenAIClient(model.APIKey, opts...), nil
}

// createAnthropicClient 创建 Anthropic 客户端
func (m *Manager) createAnthropicClient(model *Model) (llm.Client, error) {
	if model.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	opts := []anthropic.Option{
		anthropic.WithModel(model.ModelID),
	}

	if model.BaseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(model.BaseURL))
	}

	if model.Timeout > 0 {
		opts = append(opts, anthropic.WithTimeout(time.Duration(model.Timeout)*time.Second))
	}

	return anthropic.NewAnthropicClient(model.APIKey, opts...), nil
}

// createOllamaClient 创建 Ollama 客户端
func (m *Manager) createOllamaClient(model *Model) (llm.Client, error) {
	baseURL := model.BaseURL
	if baseURL == "" {
		baseURL = DefaultOllamaBaseURL
	}

	opts := []ollama.Option{
		ollama.WithModel(model.ModelID),
		ollama.WithBaseURL(baseURL),
	}

	if model.Temperature > 0 {
		opts = append(opts, ollama.WithTemperature(model.Temperature))
	}

	if model.Timeout > 0 {
		opts = append(opts, ollama.WithTimeout(time.Duration(model.Timeout)*time.Second))
	}

	return ollama.NewOllamaClient(opts...), nil
}
