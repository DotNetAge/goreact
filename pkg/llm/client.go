package llm

// Client LLM 客户端接口
type Client interface {
	// Generate 生成文本
	Generate(prompt string) (string, error)
}
