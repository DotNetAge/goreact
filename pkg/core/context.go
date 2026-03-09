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

// Clone 克隆上下文
func (c *Context) Clone() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	newCtx := &Context{
		data: make(map[string]any, len(c.data)),
		ctx:  c.ctx,
	}
	for k, v := range c.data {
		newCtx.data[k] = v
	}
	return newCtx
}
