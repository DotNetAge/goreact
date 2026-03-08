package model

import (
	"github.com/ray/goreact/pkg/llm"
)

// LLMAdapter LLM客户端适配器
type LLMAdapter struct {
	name     string
	modelType string
	client   llm.Client
	capabilities []Capability
}

// NewLLMAdapter 创建LLM客户端适配器
func NewLLMAdapter(name, modelType string, client llm.Client, capabilities []Capability) *LLMAdapter {
	return &LLMAdapter{
		name:     name,
		modelType: modelType,
		client:   client,
		capabilities: capabilities,
	}
}

// Name 返回模型名称
func (a *LLMAdapter) Name() string {
	return a.name
}

// Type 返回模型类型
func (a *LLMAdapter) Type() string {
	return a.modelType
}

// Execute 执行模型推理
func (a *LLMAdapter) Execute(prompt string, options map[string]interface{}) (string, error) {
	// 忽略options，直接调用LLM客户端
	return a.client.Generate(prompt)
}

// GetCapabilities 返回模型能力
func (a *LLMAdapter) GetCapabilities() []Capability {
	return a.capabilities
}
