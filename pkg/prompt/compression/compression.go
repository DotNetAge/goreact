package compression

import (
	"fmt"
	"sort"
)

// Turn 对话轮次
type Turn struct {
	Role    string
	Content string
}

// CompressionStrategy 压缩策略接口
type CompressionStrategy interface {
	Compress(turns []Turn, maxTokens int, counter TokenCounter) ([]Turn, error)
}

// TokenCounter Token 计数器接口
type TokenCounter interface {
	Count(text string) int
}

// TruncateStrategy 截断策略（移除最早的轮次）
type TruncateStrategy struct {
	RemoveRatio float64 // 移除比例（默认 0.25）
}

func NewTruncateStrategy() *TruncateStrategy {
	return &TruncateStrategy{RemoveRatio: 0.25}
}

func (s *TruncateStrategy) Compress(turns []Turn, maxTokens int, counter TokenCounter) ([]Turn, error) {
	if len(turns) <= 1 {
		return turns, nil
	}

	currentTokens := s.countTurns(turns, counter)
	if currentTokens <= maxTokens {
		return turns, nil
	}

	// 移除最早的 N% 轮次
	removeCount := int(float64(len(turns)) * s.RemoveRatio)
	if removeCount < 1 {
		removeCount = 1
	}

	return turns[removeCount:], nil
}

func (s *TruncateStrategy) countTurns(turns []Turn, counter TokenCounter) int {
	total := 0
	for _, turn := range turns {
		total += counter.Count(turn.Content)
	}
	return total
}

// SlidingWindowStrategy 滑动窗口策略（保留最近的 N 轮）
type SlidingWindowStrategy struct {
	WindowSize int // 窗口大小（轮次数）
}

func NewSlidingWindowStrategy(windowSize int) *SlidingWindowStrategy {
	if windowSize <= 0 {
		windowSize = 10
	}
	return &SlidingWindowStrategy{WindowSize: windowSize}
}

func (s *SlidingWindowStrategy) Compress(turns []Turn, maxTokens int, counter TokenCounter) ([]Turn, error) {
	if len(turns) <= s.WindowSize {
		return turns, nil
	}

	return turns[len(turns)-s.WindowSize:], nil
}

// PriorityStrategy 优先级压缩策略
type PriorityStrategy struct {
	Priorities      map[string]int // role -> priority
	KeepRecent      int            // 保留最近的 N 轮（不管优先级）
	KeepSystemFirst bool           // 保留第一条 system 消息
}

func NewPriorityStrategy(priorities map[string]int) *PriorityStrategy {
	if priorities == nil {
		priorities = map[string]int{
			"system":    100,
			"user":      80,
			"assistant": 60,
		}
	}
	return &PriorityStrategy{
		Priorities:      priorities,
		KeepRecent:      3,
		KeepSystemFirst: true,
	}
}

func (s *PriorityStrategy) Compress(turns []Turn, maxTokens int, counter TokenCounter) ([]Turn, error) {
	if len(turns) == 0 {
		return turns, nil
	}

	currentTokens := s.countTurns(turns, counter)
	if currentTokens <= maxTokens {
		return turns, nil
	}

	// 1. 标记必须保留的消息
	mustKeep := make(map[int]bool)

	// 保留第一条 system 消息
	if s.KeepSystemFirst {
		for i, turn := range turns {
			if turn.Role == "system" {
				mustKeep[i] = true
				break
			}
		}
	}

	// 保留最近的 N 轮
	recentStart := len(turns) - s.KeepRecent
	if recentStart < 0 {
		recentStart = 0
	}
	for i := recentStart; i < len(turns); i++ {
		mustKeep[i] = true
	}

	// 2. 为每个消息计算分数（优先级 * 位置权重）
	type scoredTurn struct {
		index  int
		turn   Turn
		score  float64
		tokens int
	}

	var scored []scoredTurn
	for i, turn := range turns {
		if mustKeep[i] {
			continue // 跳过必须保留的
		}

		priority := s.Priorities[turn.Role]
		if priority == 0 {
			priority = 50 // 默认优先级
		}

		// 位置权重：越新的消息权重越高
		positionWeight := float64(i+1) / float64(len(turns))

		score := float64(priority) * (0.5 + 0.5*positionWeight)
		tokens := counter.Count(turn.Content)

		scored = append(scored, scoredTurn{
			index:  i,
			turn:   turn,
			score:  score,
			tokens: tokens,
		})
	}

	// 3. 按分数排序（降序）
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 4. 贪心选择：按分数选择，直到达到 token 限制
	selected := make(map[int]bool)
	for k := range mustKeep {
		selected[k] = true
	}

	usedTokens := 0
	for k := range mustKeep {
		usedTokens += counter.Count(turns[k].Content)
	}

	for _, st := range scored {
		if usedTokens+st.tokens <= maxTokens {
			selected[st.index] = true
			usedTokens += st.tokens
		}
	}

	// 5. 按原始顺序重建
	var result []Turn
	for i, turn := range turns {
		if selected[i] {
			result = append(result, turn)
		}
	}

	return result, nil
}

func (s *PriorityStrategy) countTurns(turns []Turn, counter TokenCounter) int {
	total := 0
	for _, turn := range turns {
		total += counter.Count(turn.Content)
	}
	return total
}

// HybridStrategy 混合策略（组合多种策略）
type HybridStrategy struct {
	Strategies []CompressionStrategy
}

func NewHybridStrategy(strategies ...CompressionStrategy) *HybridStrategy {
	return &HybridStrategy{Strategies: strategies}
}

func (s *HybridStrategy) Compress(turns []Turn, maxTokens int, counter TokenCounter) ([]Turn, error) {
	result := turns
	var err error

	for _, strategy := range s.Strategies {
		result, err = strategy.Compress(result, maxTokens, counter)
		if err != nil {
			return nil, fmt.Errorf("strategy failed: %w", err)
		}

		// 如果已经满足要求，提前退出
		currentTokens := s.countTurns(result, counter)
		if currentTokens <= maxTokens {
			break
		}
	}

	return result, nil
}

func (s *HybridStrategy) countTurns(turns []Turn, counter TokenCounter) int {
	total := 0
	for _, turn := range turns {
		total += counter.Count(turn.Content)
	}
	return total
}
