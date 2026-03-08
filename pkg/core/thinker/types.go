package thinker

import "time"

// IntentType 意图类型
type IntentType string

const (
	IntentTypeToolCall  IntentType = "tool_call"  // 工具调用
	IntentTypeQuestion  IntentType = "question"   // 问题
	IntentTypeChat      IntentType = "chat"       // 闲聊
	IntentTypeMultiStep IntentType = "multi_step" // 多步骤任务
)

// Intent 意图
type Intent struct {
	Type       IntentType             // 意图类型
	ToolName   string                 // 工具名称（如果是 tool_call）
	Parameters map[string]interface{} // 参数
	Missing    []string               // 缺失的参数
	Confidence float64                // 置信度 (0-1)
	SubIntents []*Intent              // 子意图（如果是 multi_step）
}

// Turn 对话轮次
type Turn struct {
	Role      string    // user, assistant, system
	Content   string    // 内容
	Timestamp time.Time // 时间戳
}

// PromptTemplate 提示词模板
type PromptTemplate struct {
	System string // System Prompt 模板
	User   string // User Prompt 模板
}

// Prompt 完整的提示词
type Prompt struct {
	System string // System Prompt
	User   string // User Prompt
}

// String 返回完整的 prompt 字符串
func (p *Prompt) String() string {
	if p.System != "" {
		return p.System + "\n\n" + p.User
	}
	return p.User
}

// ToolDesc 工具描述
type ToolDesc struct {
	Name        string // 工具名称
	Description string // 工具描述
}

// CompressionStrategy 压缩策略
type CompressionStrategy string

const (
	StrategyTruncate      CompressionStrategy = "truncate"       // 截断最早的
	StrategySlidingWindow CompressionStrategy = "sliding_window" // 滑动窗口
	StrategySummarize     CompressionStrategy = "summarize"      // 摘要（需要 LLM）
)
