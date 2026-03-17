package thinker

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// ThinkHandler Think 处理器函数
type ThinkHandler func(task string, ctx *core.Context) (*types.Thought, error)

// ThinkMiddleware Think 中间件
// 接收下一个处理器，返回包装后的处理器
type ThinkMiddleware func(next ThinkHandler) ThinkHandler

// MiddlewareThinker 支持中间件的 Thinker
type MiddlewareThinker struct {
	baseThinker Thinker
	middlewares []ThinkMiddleware
	handler     ThinkHandler
}

// NewMiddlewareThinker 创建支持中间件的 Thinker
func NewMiddlewareThinker(base Thinker) *MiddlewareThinker {
	mt := &MiddlewareThinker{
		baseThinker: base,
		middlewares: make([]ThinkMiddleware, 0),
	}
	mt.buildHandler()
	return mt
}

// Use 添加中间件（支持可变参数）
// 中间件按添加顺序执行（先添加的在外层）
func (t *MiddlewareThinker) Use(middlewares ...ThinkMiddleware) {
	t.middlewares = append(t.middlewares, middlewares...)
	t.buildHandler()
}

// buildHandler 构建中间件链
func (t *MiddlewareThinker) buildHandler() {
	// 最内层：调用基础 Thinker
	handler := func(task string, ctx *core.Context) (*types.Thought, error) {
		return t.baseThinker.Think(task, ctx)
	}

	// 从后向前包装中间件（洋葱模型）
	for i := len(t.middlewares) - 1; i >= 0; i-- {
		handler = t.middlewares[i](handler)
	}

	t.handler = handler
}

// Think 执行思考（通过中间件链）
func (t *MiddlewareThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
	return t.handler(task, ctx)
}

// GetMiddlewares 获取所有中间件
func (t *MiddlewareThinker) GetMiddlewares() []ThinkMiddleware {
	return t.middlewares
}

// GetBaseThinker 获取基础 Thinker
func (t *MiddlewareThinker) GetBaseThinker() Thinker {
	return t.baseThinker
}
