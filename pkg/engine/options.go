package engine

import (
	"time"

	"github.com/ray/goreact/pkg/agent"
	"github.com/ray/goreact/pkg/cache"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/model"
	"github.com/ray/goreact/pkg/skill"
	"github.com/ray/goreact/pkg/tool/provider"
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

// WithSkillManager 设置技能管理器
func WithSkillManager(sm skill.Manager) Option {
	return func(e *Engine) {
		e.skillManager = sm
	}
}

// WithProviderRegistry 设置工具提供者注册表
func WithProviderRegistry(registry *provider.Registry) Option {
	return func(e *Engine) {
		e.providerRegistry = registry
	}
}

// WithProvider 注册单个工具提供者
func WithProvider(p provider.Provider) Option {
	return func(e *Engine) {
		if e.providerRegistry == nil {
			e.providerRegistry = provider.NewRegistry()
		}
		if err := e.providerRegistry.Register(p); err != nil {
			// 记录错误但不中断初始化
			if e.logger != nil {
				e.logger.Warn("Failed to register provider", log.Err(err))
			}
		}
	}
}

// WithAgentManager 设置 Agent 管理器
func WithAgentManager(am *agent.Manager) Option {
	return func(e *Engine) {
		e.agentManager = am
	}
}

// WithModelManager 设置 Model 管理器
func WithModelManager(mm *model.Manager) Option {
	return func(e *Engine) {
		e.modelManager = mm
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(e *Engine) {
		e.logger = logger
	}
}

// WithMaxTraceSize 设置最大 Trace 大小
func WithMaxTraceSize(size int) Option {
	return func(e *Engine) {
		if size > 0 {
			e.maxTraceSize = size
		}
	}
}
