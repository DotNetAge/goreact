package cost

import (
	"fmt"
	"sync"
)

// Pricing 定价配置
type Pricing struct {
	InputTokenPrice  float64 // 每 1K tokens 的价格
	OutputTokenPrice float64 // 每 1K tokens 的价格
}

// Tracker 成本追踪器
type Tracker struct {
	mu                sync.Mutex
	pricing           Pricing
	totalInputTokens  int
	totalOutputTokens int
}

// NewTracker 创建成本追踪器
func NewTracker(pricing Pricing) *Tracker {
	return &Tracker{
		pricing: pricing,
	}
}

// RecordTokens 记录 token 使用量
func (t *Tracker) RecordTokens(inputTokens, outputTokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalInputTokens += inputTokens
	t.totalOutputTokens += outputTokens
}

// TotalCost 计算总成本
func (t *Tracker) TotalCost() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	inputCost := float64(t.totalInputTokens) / 1000.0 * t.pricing.InputTokenPrice
	outputCost := float64(t.totalOutputTokens) / 1000.0 * t.pricing.OutputTokenPrice

	return inputCost + outputCost
}

// TotalInputTokens 返回总输入 tokens
func (t *Tracker) TotalInputTokens() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalInputTokens
}

// TotalOutputTokens 返回总输出 tokens
func (t *Tracker) TotalOutputTokens() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalOutputTokens
}

// ExceedsLimit 检查是否超过成本限制
func (t *Tracker) ExceedsLimit(limit float64) bool {
	return t.TotalCost() > limit
}

// Report 生成成本报告
func (t *Tracker) Report() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	inputCost := float64(t.totalInputTokens) / 1000.0 * t.pricing.InputTokenPrice
	outputCost := float64(t.totalOutputTokens) / 1000.0 * t.pricing.OutputTokenPrice
	totalCost := inputCost + outputCost

	return fmt.Sprintf(`=== Cost Report ===
Total Input Tokens:  %d
Total Output Tokens: %d
Total Cost:          $%.4f

Breakdown:
- Input:  %d tokens × $%.2f/1K = $%.4f
- Output: %d tokens × $%.2f/1K = $%.4f`,
		t.totalInputTokens,
		t.totalOutputTokens,
		totalCost,
		t.totalInputTokens,
		t.pricing.InputTokenPrice,
		inputCost,
		t.totalOutputTokens,
		t.pricing.OutputTokenPrice,
		outputCost,
	)
}

// Reset 重置追踪器
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalInputTokens = 0
	t.totalOutputTokens = 0
}
