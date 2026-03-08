package cache

import (
	"sync"
	"time"
)

// cacheItem 缓存项
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	mu         sync.RWMutex
	items      map[string]*cacheItem
	maxSize    int
	defaultTTL time.Duration
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
	cache := &MemoryCache{
		items:      make(map[string]*cacheItem),
		maxSize:    1000,
		defaultTTL: 1 * time.Hour,
	}

	for _, opt := range options {
		opt(cache)
	}

	// 启动清理协程
	go cache.cleanupExpired()

	return cache
}

// Get 获取缓存值
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

// Set 设置缓存值
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
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

// evictOldest 删除最旧的项（简单实现，删除第一个找到的）
func (c *MemoryCache) evictOldest() {
	for key := range c.items {
		delete(c.items, key)
		return
	}
}

// cleanupExpired 定期清理过期项
func (c *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
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
