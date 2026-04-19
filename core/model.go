package core

import "github.com/DotNetAge/gochat/core"

type ModelConfig struct {
	Name        string `json:"name" yaml:"name"`               // 模型名称
	Description string `json:"description" yaml:"description"` // 模型描述
	BaseURL     string `json:"base_url" yaml:"base_url"`       // 模型基地址
	APIKey      string `json:"api_key" yaml:"api_key"`         // API密钥
	AuthToken   string `json:"auth_token" yaml:"auth_token"`   // 认证令牌
	MaxTokens   int64  `json:"max_tokens" yaml:"max_tokens"`   // 最大token数
}

func (m *ModelConfig) Config() *core.Config {
	return &core.Config{
		BaseURL:   m.BaseURL,
		APIKey:    m.APIKey,
		AuthToken: m.AuthToken,
	}
}
