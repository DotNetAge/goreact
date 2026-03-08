package cache

import (
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	// 创建内存缓存
	cache := NewMemoryCache(
		WithMaxSize(100),
		WithDefaultTTL(1*time.Second),
	)

	// 测试设置和获取
	cache.Set("key1", "value1", 0)
	if value, ok := cache.Get("key1"); !ok || value != "value1" {
		t.Errorf("Expected to get value1, got %v (ok: %v)", value, ok)
	}

	// 测试删除
	cache.Delete("key1")
	if _, ok := cache.Get("key1"); ok {
		t.Error("Expected key1 to be deleted")
	}

	// 测试清空
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)
	cache.Clear()
	if _, ok := cache.Get("key2"); ok {
		t.Error("Expected key2 to be cleared")
	}
	if _, ok := cache.Get("key3"); ok {
		t.Error("Expected key3 to be cleared")
	}

	// 测试大小
	cache.Set("key4", "value4", 0)
	if size := cache.Size(); size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// 测试过期
	cache.Set("key5", "value5", 100*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	if _, ok := cache.Get("key5"); ok {
		t.Error("Expected key5 to be expired")
	}
}
