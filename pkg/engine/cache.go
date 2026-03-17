package engine

import (
	"crypto/sha256"
	"fmt"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
)

// generateCacheKey 生成缓存键
func (r *reactor) generateCacheKey(task string) string {
	data := fmt.Sprintf("task:%s|tools:%s",
		task,
		r.toolManager.GetToolDescriptions(),
	)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// getOrCreateThinker 获取或创建 Thinker（带缓存）
func (r *reactor) getOrCreateThinker(llmClient gochatcore.Client, systemPrompt string) core.Thinker {
	cacheKey := fmt.Sprintf("%p:%s", llmClient, systemPrompt)

	if cached, ok := r.thinkerCache[cacheKey]; ok {
		return cached
	}

	var newThinker core.Thinker
	toolDesc := r.toolManager.GetToolDescriptions()

	if systemPrompt != "" {
		newThinker = thinker.NewSimpleThinkerWithSystemPrompt(llmClient, toolDesc, systemPrompt)
	} else {
		newThinker = thinker.NewSimpleThinker(llmClient, toolDesc)
	}

	r.thinkerCache[cacheKey] = newThinker
	return newThinker
}
