package cache

import "time"

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(key string) (interface{}, bool)

	// Set 设置缓存值
	Set(key string, value interface{}, ttl time.Duration)

	// Delete 删除缓存值
	Delete(key string)

	// Clear 清空所有缓存
	Clear()

	// Size 获取缓存大小
	Size() int
}
