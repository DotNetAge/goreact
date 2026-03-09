package llm

import "context"

// Client LLM 客户端接口
type Client interface {
	// Generate 生成文本
	Generate(ctx context.Context, prompt string) (string, error)
}

// TokenUsage Token 使用量
type TokenUsage struct {
	PromptTokens     int // 输入 Token 数
	CompletionTokens int // 输出 Token 数
	TotalTokens      int // 总 Token 数
}

// TokenReporter 可选接口，LLM Client 实现此接口以报告 Token 消耗
// Engine 在调用 Generate 后检查 Client 是否实现了此接口
type TokenReporter interface {
	// LastTokenUsage 返回最近一次 Generate 调用的 Token 使用量
	LastTokenUsage() *TokenUsage
}
