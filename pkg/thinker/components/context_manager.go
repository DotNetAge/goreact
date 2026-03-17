package components

import (
	"fmt"
	"strings"

	"github.com/ray/goreact/pkg/core/thinker"
)

// ContextManager 上下文管理器接口
type ContextManager interface {
	// AddTurn 添加对话轮次
	AddTurn(turn *thinker.Turn)

	// GetHistory 获取历史（格式化）
	GetHistory(maxTurns int) string

	// Compress 压缩上下文
	Compress(strategy thinker.CompressionStrategy) error

	// EstimateTokens 估算 token 数
	EstimateTokens() int

	// Clear 清空历史
	Clear()
}

// DefaultContextManager 默认上下文管理器
type DefaultContextManager struct {
	turns     []*thinker.Turn
	maxTokens int
}

// NewContextManager 创建上下文管理器
func NewContextManager(maxTokens int) *DefaultContextManager {
	return &DefaultContextManager{
		turns:     make([]*thinker.Turn, 0),
		maxTokens: maxTokens,
	}
}

// AddTurn 添加对话轮次
func (m *DefaultContextManager) AddTurn(turn *thinker.Turn) {
	m.turns = append(m.turns, turn)

	// 自动压缩（如果超过限制）
	if m.EstimateTokens() > m.maxTokens {
		m.Compress(thinker.StrategyTruncate)
	}
}

// GetHistory 获取历史（格式化）
func (m *DefaultContextManager) GetHistory(maxTurns int) string {
	if len(m.turns) == 0 {
		return ""
	}

	// 限制返回的轮次数
	start := 0
	if maxTurns > 0 && len(m.turns) > maxTurns {
		start = len(m.turns) - maxTurns
	}

	var sb strings.Builder
	for _, turn := range m.turns[start:] {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", turn.Role, turn.Content))
	}

	return sb.String()
}

// Compress 压缩上下文
func (m *DefaultContextManager) Compress(strategy thinker.CompressionStrategy) error {
	switch strategy {
	case thinker.StrategyTruncate:
		return m.compressTruncate()
	case thinker.StrategySlidingWindow:
		return m.compressSlidingWindow()
	case thinker.StrategySummarize:
		// 摘要策略需要 LLM，暂不实现
		return fmt.Errorf("summarize strategy not implemented yet")
	default:
		return fmt.Errorf("unknown compression strategy: %s", strategy)
	}
}

// compressTruncate 截断压缩（移除最早的轮次）
func (m *DefaultContextManager) compressTruncate() error {
	if len(m.turns) <= 1 {
		return nil
	}

	// 移除最早的 25% 轮次
	removeCount := len(m.turns) / 4
	if removeCount < 1 {
		removeCount = 1
	}

	m.turns = m.turns[removeCount:]
	return nil
}

// compressSlidingWindow 滑动窗口压缩（保留最近的 N 轮）
func (m *DefaultContextManager) compressSlidingWindow() error {
	windowSize := m.maxTokens / 100 // 假设每轮平均 100 tokens
	if windowSize < 5 {
		windowSize = 5
	}

	if len(m.turns) > windowSize {
		m.turns = m.turns[len(m.turns)-windowSize:]
	}

	return nil
}

// EstimateTokens 估算 token 数（简单估算：1 token ≈ 4 字符）
func (m *DefaultContextManager) EstimateTokens() int {
	totalChars := 0
	for _, turn := range m.turns {
		totalChars += len(turn.Content)
	}
	return totalChars / 4
}

// Clear 清空历史
func (m *DefaultContextManager) Clear() {
	m.turns = make([]*thinker.Turn, 0)
}
