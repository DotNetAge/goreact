package core

import "sync"

// Context 执行上下文，用于在 ReAct 循环中传递数据
type Context struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewContext 创建新的上下文
func NewContext() *Context {
	return &Context{
		data: make(map[string]interface{}),
	}
}

// Get 获取上下文中的值
func (c *Context) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

// Set 设置上下文中的值
func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// GetAll 获取所有上下文数据
func (c *Context) GetAll() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]interface{}, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// Clone 克隆上下文
func (c *Context) Clone() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	newCtx := NewContext()
	for k, v := range c.data {
		newCtx.data[k] = v
	}
	return newCtx
}
