package core

type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// ContextWindow 多轮对话上下文窗口
type ContextWindow struct {
	SessionID       string    // 会话ID
	Messages        []Message // 会话消息列表
	TokensRemaining int64     // 剩余Token数量
	MaxTokens       int64     // 最大Token数量
}
