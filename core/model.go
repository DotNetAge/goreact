package core

import "github.com/DotNetAge/gochat/core"

type ModelConfig struct {
	// ID          string `json:"id" yaml:"id"`
	Name              string  `json:"name" yaml:"name"`                             // 模型名称
	Description       string  `json:"description" yaml:"description"`               // 模型描述
	BaseURL           string  `json:"base_url" yaml:"base_url"`                     // 模型基地址
	APIKey            string  `json:"api_key" yaml:"api_key"`                       // API密钥
	AuthToken         string  `json:"auth_token" yaml:"auth_token"`                 // 认证令牌
	MaxTokens         int64   `json:"max_tokens" yaml:"max_tokens"`                 // 最大token数
	IsLocal           bool    `json:"is_local" yaml:"is_local"`                     // 是否本地模型
	FuncCalling       bool    `json:"func_calling" yaml:"func_calling"`             // 是否支持函数调用
	Structuring       bool    `json:"structuring" yaml:"structuring"`               // 是否支持结构化
	WebSearching      bool    `json:"web_searching" yaml:"web_searching"`           // 是否支持网页搜索
	PrefixCon         bool    `json:"prefix_con" yaml:"prefix_con"`                 // 是否支持前缀续写
	ContextCache      bool    `json:"context_cache" yaml:"context_cache"`           // 是否支持上下文缓存
	TopP              float64 `json:"top_p" yaml:"top_p"`                           // top p
	TopK              float64 `json:"top_k" yaml:"top_k"`                           // top k
	Temperature       float64 `json:"temperature" yaml:"temperature"`               // 温度
	RepetitionPenalty float64 `json:"repetition_penalty" yaml:"repetition_penalty"` // 重复惩罚
}

func (m *ModelConfig) Config() *core.Config {
	return &core.Config{
		BaseURL:   m.BaseURL,
		APIKey:    m.APIKey,
		AuthToken: m.AuthToken,
	}
}
