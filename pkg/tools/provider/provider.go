package provider

import (
	"fmt"

	"github.com/ray/goreact/pkg/tools"
)

// Provider 工具提供者接口
// 用于集成外部工具系统（如 MCP、LangChain Tools、OpenAI Functions 等）
type Provider interface {
	// Name 返回提供者名称
	Name() string

	// Initialize 初始化提供者（连接、认证等）
	Initialize(config map[string]interface{}) error

	// DiscoverTools 发现可用的工具
	DiscoverTools() ([]tools.Tool, error)

	// GetTool 获取指定的工具
	GetTool(name string) (tools.Tool, error)

	// Close 关闭提供者连接
	Close() error

	// IsHealthy 检查提供者健康状态
	IsHealthy() bool
}

// Registry 提供者注册表
type Registry struct {
	providers map[string]Provider
}

// NewRegistry 创建新的提供者注册表
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register 注册提供者
func (r *Registry) Register(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// Get 获取提供者
func (r *Registry) Get(name string) (Provider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// List 列出所有提供者
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// DiscoverAllTools 从所有提供者发现工具
func (r *Registry) DiscoverAllTools() ([]tools.Tool, error) {
	allTools := make([]tools.Tool, 0)

	for _, provider := range r.providers {
		if !provider.IsHealthy() {
			continue
		}

		tools, err := provider.DiscoverTools()
		if err != nil {
			// 记录错误但继续处理其他提供者
			continue
		}

		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

// Close 关闭所有提供者
func (r *Registry) Close() error {
	var lastErr error
	for _, provider := range r.providers {
		if err := provider.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
