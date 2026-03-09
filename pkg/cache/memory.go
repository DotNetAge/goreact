package cache

import (
	"context"
	"runtime"
	"sync"
	"time"
)

const (
	DefaultMaxSize         = 1000
	DefaultTTL             = 1 * time.Hour
	DefaultCleanupInterval = 5 * time.Minute
)

// cacheItem 缓存项
type cacheItem struct {
	value      any
	expiration time.Time
	lastAccess time.Time // 最后访问时间，用于 LRU 驱逐
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	mu         sync.RWMutex
	items      map[string]*cacheItem
	maxSize    int
	defaultTTL time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
}

// MemoryOption 内存缓存配置选项
type MemoryOption func(*MemoryCache)

// WithMaxSize 设置最大缓存数量
func WithMaxSize(size int) MemoryOption {
	return func(c *MemoryCache) {
		c.maxSize = size
	}
}

// WithDefaultTTL 设置默认 TTL
func WithDefaultTTL(ttl time.Duration) MemoryOption {
	return func(c *MemoryCache) {
		c.defaultTTL = ttl
	}
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(options ...MemoryOption) *MemoryCache {
	ctx, cancel := context.WithCancel(context.Background())

	cache := &MemoryCache{
		items:      make(map[string]*cacheItem),
		maxSize:    DefaultMaxSize,
		defaultTTL: DefaultTTL,
		ctx:        ctx,
		cancel:     cancel,
	}

	for _, opt := range options {
		opt(cache)
	}

	// 启动清理协程
	go cache.cleanupExpired()

	// 设置 finalizer 确保 goroutine 被清理
	runtime.SetFinalizer(cache, func(c *MemoryCache) {
		_ = c.Close() // 忽略错误，因为在 finalizer 中无法处理
	})

	return cache
}

// Get 获取缓存值
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(item.expiration) {
		delete(c.items, key)
		return nil, false
	}

	// 更新最后访问时间（LRU）
	item.lastAccess = time.Now()

	return item.value, true
}

// Set 设置缓存值
func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果达到最大容量，删除最旧的项
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	// 使用默认 TTL
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	c.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
		lastAccess: time.Now(),
	}
}

// Delete 删除缓存值
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// Size 获取缓存大小
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Close 关闭缓存，停止清理协程
func (c *MemoryCache) Close() error {
	c.cancel()
	return nil
}

// evictOldest 删除最旧的项（LRU 驱逐策略）
func (c *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range c.items {
		if first || item.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.lastAccess
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanupExpired 定期清理过期项
func (c *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(DefaultCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, item := range c.items {
				if now.After(item.expiration) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		}
	}
}
