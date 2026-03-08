package engine

import (
	"time"

	"github.com/ray/goreact/pkg/cache"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/metrics"
)

// Option 引擎配置选项
type Option func(*Engine)

// WithMaxIterations 设置最大迭代次数
func WithMaxIterations(max int) Option {
	return func(e *Engine) {
		e.loopController = core.NewDefaultLoopController(max)
	}
}

// WithThinker 设置思考模块
func WithThinker(thinker core.Thinker) Option {
	return func(e *Engine) {
		e.thinker = thinker
	}
}

// WithActor 设置行动模块
func WithActor(actor core.Actor) Option {
	return func(e *Engine) {
		e.actor = actor
	}
}

// WithObserver 设置观察模块
func WithObserver(observer core.Observer) Option {
	return func(e *Engine) {
		e.observer = observer
	}
}

// WithLoopController 设置循环控制器
func WithLoopController(controller core.LoopController) Option {
	return func(e *Engine) {
		e.loopController = controller
	}
}

// WithLLMClient 设置 LLM 客户端
func WithLLMClient(client llm.Client) Option {
	return func(e *Engine) {
		e.llmClient = client
	}
}

// WithCache 设置缓存
func WithCache(c cache.Cache) Option {
	return func(e *Engine) {
		e.cache = c
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(max int) Option {
	return func(e *Engine) {
		e.maxRetries = max
	}
}

// WithRetryInterval 设置重试间隔
func WithRetryInterval(interval time.Duration) Option {
	return func(e *Engine) {
		e.retryInterval = interval
	}
}

// WithMetrics 设置指标收集器
func WithMetrics(m metrics.Metrics) Option {
	return func(e *Engine) {
		e.metrics = m
	}
}
