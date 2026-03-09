package core

import (
	"context"
	"sync"
)

// Context 执行上下文，用于在 ReAct 循环中传递数据
type Context struct {
	mu   sync.RWMutex
	data map[string]any
	ctx  context.Context // 底层 context.Context
}

// NewContext 创建新的上下文
func NewContext() *Context {
	return &Context{
		data: make(map[string]any),
		ctx:  context.Background(),
	}
}

// NewContextWithContext 使用指定的 context.Context 创建上下文
func NewContextWithContext(ctx context.Context) *Context {
	return &Context{
		data: make(map[string]any),
		ctx:  ctx,
	}
}

// Context 获取底层的 context.Context
func (c *Context) Context() context.Context {
	return c.ctx
}

// Get 获取上下文中的值
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

// Set 设置上下文中的值
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// GetAll 获取所有上下文数据
func (c *Context) GetAll() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]any, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// Clone 克隆上下文（深拷贝）
func (c *Context) Clone() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	newCtx := &Context{
		data: make(map[string]any, len(c.data)),
		ctx:  c.ctx,
	}
	for k, v := range c.data {
		newCtx.data[k] = deepCopyValue(v)
	}
	return newCtx
}

// deepCopyValue 深拷贝值（处理常见的引用类型）
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []map[string]string:
		// 拷贝 []map[string]string（用于 history_steps）
		copied := make([]map[string]string, len(val))
		for i, m := range val {
			copiedMap := make(map[string]string, len(m))
			for mk, mv := range m {
				copiedMap[mk] = mv
			}
			copied[i] = copiedMap
		}
		return copied

	case map[string]any:
		// 拷贝 map[string]any
		copied := make(map[string]any, len(val))
		for mk, mv := range val {
			copied[mk] = deepCopyValue(mv)
		}
		return copied

	case map[string]string:
		// 拷贝 map[string]string
		copied := make(map[string]string, len(val))
		for mk, mv := range val {
			copied[mk] = mv
		}
		return copied

	case []string:
		// 拷贝 []string
		copied := make([]string, len(val))
		copy(copied, val)
		return copied

	case []any:
		// 拷贝 []any
		copied := make([]any, len(val))
		for i, item := range val {
			copied[i] = deepCopyValue(item)
		}
		return copied

	default:
		// 对于基本类型（string, int, bool 等）和不可变类型，直接返回
		// 注意：如果存储了指针或其他复杂类型，需要额外处理
		return v
	}
}
