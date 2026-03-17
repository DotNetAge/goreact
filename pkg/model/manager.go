package model

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/DotNetAge/gochat/pkg/client/anthropic"
	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/ollama"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	gochatcore "github.com/DotNetAge/gochat/pkg/core"
)

const (
	DefaultOllamaBaseURL = "http://localhost:11434"
)

// Manager 模型管理器
type Manager struct {
	models map[string]*Model
	mu     sync.RWMutex // 保护并发访问
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
	m.mu.Lock()
	m.models[model.Name] = model
	m.mu.Unlock()
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

	m.mu.RLock()
	model, exists := m.models[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("model not found: %s", name)
	}
	return model, nil
}

// ListModels 列出所有模型
func (m *Manager) ListModels() []*Model {
	m.mu.RLock()
	models := make([]*Model, 0, len(m.models))
	for _, model := range m.models {
		models = append(models, model)
	}
	m.mu.RUnlock()
	return models
}

// CreateLLMClient 根据模型配置创建 LLM 客户端
func (m *Manager) CreateLLMClient(modelName string) (gochatcore.Client, error) {
	model, err := m.GetModel(modelName)
	if err != nil {
		return nil, err
	}

	return m.createClientFromModel(model)
}

// createClientFromModel 根据模型配置创建客户端
func (m *Manager) createClientFromModel(model *Model) (gochatcore.Client, error) {
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
func (m *Manager) createOpenAIClient(model *Model) (gochatcore.Client, error) {
	if model.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	cfg := openai.Config{
		Config: base.Config{
			APIKey:      model.APIKey,
			Model:       model.ModelID,
			BaseURL:     model.BaseURL,
			Temperature: float64(model.Temperature),
		},
	}

	c, err := openai.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// createAnthropicClient 创建 Anthropic 客户端
func (m *Manager) createAnthropicClient(model *Model) (gochatcore.Client, error) {
	if model.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	cfg := anthropic.Config{
		Config: base.Config{
			APIKey:      model.APIKey,
			Model:       model.ModelID,
			BaseURL:     model.BaseURL,
			Temperature: float64(model.Temperature),
		},
	}

	c, err := anthropic.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// createOllamaClient 创建 Ollama 客户端
func (m *Manager) createOllamaClient(model *Model) (gochatcore.Client, error) {
	baseURL := model.BaseURL
	if baseURL == "" {
		baseURL = DefaultOllamaBaseURL
	}

	cfg := ollama.Config{
		Config: base.Config{
			Model:       model.ModelID,
			BaseURL:     baseURL,
			Temperature: float64(model.Temperature),
		},
	}

	c, err := ollama.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}
