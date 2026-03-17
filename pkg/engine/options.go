package engine

import (
	"time"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/agent"
	"github.com/ray/goreact/pkg/cache"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/model"
	"github.com/ray/goreact/pkg/observer"
	"github.com/ray/goreact/pkg/skill"
	"github.com/ray/goreact/pkg/terminator"
	"github.com/ray/goreact/pkg/tools"
)

const (
	DefaultMaxIterations = 10
	DefaultMaxRetries    = 3
	DefaultRetryInterval = 1 * time.Second
	DefaultMaxTraceSize  = 1000
)

// ReactorOption Reactator 配置选项
type ReactorOption func(*ReactorOptions)

// ReactorOptions Reactor 配置选项集合
type ReactorOptions struct {
	Logger         log.Logger
	Metrics        metrics.Metrics
	LLMClient      gochatcore.Client
	ModelManager   *model.Manager
	ToolManager    *tools.Manager
	AgentManager   *agent.Manager
	SkillManager   skill.Manager
	Cache          cache.Cache
	Thinker        thinker.Thinker
	Actor          actor.Actor
	Observer       observer.Observer
	LoopController terminator.Terminator
	MaxRetries     int
	RetryInterval  time.Duration
	MaxTraceSize   int
}

// DefaultReactorOptions 创建默认配置
func DefaultReactorOptions() *ReactorOptions {
	return &ReactorOptions{
		Logger:         nil,
		Metrics:        metrics.NewDefaultMetrics(),
		LLMClient:      nil,
		ModelManager:   nil,
		ToolManager:    tools.NewManager(),
		AgentManager:   nil,
		SkillManager:   nil,
		Cache:          nil,
		Thinker:        nil,
		Actor:          nil,
		Observer:       nil,
		LoopController: terminator.NewDefaultTerminator(DefaultMaxIterations),
		MaxRetries:     DefaultMaxRetries,
		RetryInterval:  DefaultRetryInterval,
		MaxTraceSize:   DefaultMaxTraceSize,
	}
}

// WithReactorLogger 设置日志记录器
func WithReactorLogger(logger log.Logger) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.Logger = logger
	}
}

// WithReactorMetrics 设置指标收集器
func WithReactorMetrics(m metrics.Metrics) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.Metrics = m
	}
}

// WithReactorLLMClient 设置 LLM 客户端
func WithReactorLLMClient(client gochatcore.Client) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.LLMClient = client
	}
}

// WithReactorModelManager 设置 Model 管理器
func WithReactorModelManager(mm *model.Manager) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.ModelManager = mm
	}
}

// WithReactorToolManager 设置工具管理器
func WithReactorToolManager(tm *tools.Manager) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.ToolManager = tm
	}
}

// WithReactorAgentManager 设置 Agent 管理器
func WithReactorAgentManager(am *agent.Manager) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.AgentManager = am
	}
}

// WithReactorSkillManager 设置技能管理器
func WithReactorSkillManager(sm skill.Manager) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.SkillManager = sm
	}
}

// WithReactorCache 设置缓存
func WithReactorCache(c cache.Cache) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.Cache = c
	}
}

// WithReactorThinker 设置思考模块
func WithReactorThinker(thinker thinker.Thinker) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.Thinker = thinker
	}
}

// WithReactorActor 设置行动模块
func WithReactorActor(actor actor.Actor) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.Actor = actor
	}
}

// WithReactorObserver 设置观察模块
func WithReactorObserver(observer observer.Observer) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.Observer = observer
	}
}

// WithReactorLoopController 设置循环控制器
func WithReactorLoopController(controller terminator.Terminator) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.LoopController = controller
	}
}

// WithReactorMaxRetries 设置最大重试次数
func WithReactorMaxRetries(max int) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.MaxRetries = max
	}
}

// WithReactorRetryInterval 设置重试间隔
func WithReactorRetryInterval(interval time.Duration) ReactorOption {
	return func(opts *ReactorOptions) {
		opts.RetryInterval = interval
	}
}

// WithReactorMaxTraceSize 设置最大 Trace 大小
func WithReactorMaxTraceSize(size int) ReactorOption {
	return func(opts *ReactorOptions) {
		if size > 0 {
			opts.MaxTraceSize = size
		}
	}
}
