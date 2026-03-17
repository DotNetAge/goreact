package middlewares

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// IntentCache 意图缓存接口
type IntentCache interface {
	// Get 获取缓存的意图
	Get(key string) (*CachedIntent, bool)
	// Set 设置缓存
	Set(key string, intent *CachedIntent)
}

// CachedIntent 缓存的意图
type CachedIntent struct {
	Task       string         // 原始任务
	Thought    *types.Thought // 缓存的 Thought
	Similarity float64        // 相似度（用于匹配）
}

// MemoryIntentCache 内存实现的意图缓存
type MemoryIntentCache struct {
	cache map[string]*CachedIntent
}

// NewMemoryIntentCache 创建内存意图缓存
func NewMemoryIntentCache() *MemoryIntentCache {
	return &MemoryIntentCache{
		cache: make(map[string]*CachedIntent),
	}
}

func (c *MemoryIntentCache) Get(key string) (*CachedIntent, bool) {
	intent, ok := c.cache[key]
	return intent, ok
}

func (c *MemoryIntentCache) Set(key string, intent *CachedIntent) {
	c.cache[key] = intent
}

// IntentCacheMiddleware 意图缓存中间件
// 如果找到相似的历史意图，直接返回（短路）
func IntentCacheMiddleware(cache IntentCache, similarityThreshold float64) thinker.ThinkMiddleware {
	if cache == nil {
		cache = NewMemoryIntentCache()
	}

	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 生成缓存键
			cacheKey := generateCacheKey(task)

			// 查找缓存
			if cached, ok := cache.Get(cacheKey); ok {
				// 计算相似度（这里简化为精确匹配）
				similarity := 1.0
				if cached.Task != task {
					similarity = calculateSimilarity(task, cached.Task)
				}

				// 如果相似度足够高，返回缓存结果
				if similarity >= similarityThreshold {
					// 克隆 Thought 避免修改缓存
					thought := cloneThought(cached.Thought)
					if thought.Metadata == nil {
						thought.Metadata = make(map[string]interface{})
					}
					thought.Metadata["cache_hit"] = true
					thought.Metadata["cache_similarity"] = similarity

					// 将相似度存入 context（供其他中间件使用）
					ctx.Set("cache_hit", true)
					ctx.Set("cache_similarity", similarity)

					return thought, nil
				}
			}

			// 缓存未命中，执行实际 Think
			thought, err := next(task, ctx)

			// 成功则缓存结果
			if err == nil && thought != nil {
				cache.Set(cacheKey, &CachedIntent{
					Task:       task,
					Thought:    thought,
					Similarity: 1.0,
				})
			}

			return thought, err
		}
	}
}

// generateCacheKey 生成缓存键
func generateCacheKey(task string) string {
	hash := sha256.Sum256([]byte(task))
	return hex.EncodeToString(hash[:])
}

// calculateSimilarity 计算相似度（简化实现）
func calculateSimilarity(task1, task2 string) float64 {
	if task1 == task2 {
		return 1.0
	}
	// 简化：基于长度和公共前缀
	// 实际应用中应使用更复杂的算法（如编辑距离、向量相似度）
	minLen := len(task1)
	if len(task2) < minLen {
		minLen = len(task2)
	}

	commonPrefix := 0
	for i := 0; i < minLen; i++ {
		if task1[i] == task2[i] {
			commonPrefix++
		} else {
			break
		}
	}

	maxLen := len(task1)
	if len(task2) > maxLen {
		maxLen = len(task2)
	}

	return float64(commonPrefix) / float64(maxLen)
}

// cloneThought 克隆 Thought
func cloneThought(original *types.Thought) *types.Thought {
	clone := &types.Thought{
		Reasoning:    original.Reasoning,
		ShouldFinish: original.ShouldFinish,
		FinalAnswer:  original.FinalAnswer,
		Metadata:     make(map[string]interface{}),
	}

	// 复制 metadata
	for k, v := range original.Metadata {
		clone.Metadata[k] = v
	}

	// 复制 Action
	if original.Action != nil {
		clone.Action = &types.Action{
			ToolName:   original.Action.ToolName,
			Parameters: make(map[string]interface{}),
			Reasoning:  original.Action.Reasoning,
		}
		for k, v := range original.Action.Parameters {
			clone.Action.Parameters[k] = v
		}
	}

	return clone
}
