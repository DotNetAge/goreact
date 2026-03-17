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

// noOpLogger 是一个空操作 logger，当默认 logger 创建失败时使用
type noOpLogger struct{}

func (n *noOpLogger) Debug(msg string, fields ...log.Field) {}
func (n *noOpLogger) Info(msg string, fields ...log.Field)  {}
func (n *noOpLogger) Warn(msg string, fields ...log.Field)  {}
func (n *noOpLogger) Error(msg string, fields ...log.Field) {}
func (n *noOpLogger) With(fields ...log.Field) log.Logger   { return n }

// reactor ReAct 引擎实现（仅保留核心依赖）
type reactor struct {
	logger         log.Logger
	metrics        metrics.Metrics
	llmClient      gochatcore.Client
	modelManager   *model.Manager
	toolManager    *tools.Manager
	agentManager   *agent.Manager
	skillManager   skill.Manager
	cache          cache.Cache
	thinker        thinker.Thinker
	actor          actor.Actor
	observer       observer.Observer
	loopController terminator.Terminator
	maxRetries     int
	retryInterval  time.Duration
	maxTraceSize   int
	thinkerCache   map[string]thinker.Thinker
}

// RegisterTool 注册单个工具
func (r *reactor) RegisterTool(t tools.Tool) {
	r.toolManager.RegisterTool(t)
	// 清空 Thinker 缓存以使用新的工具描述
	r.thinkerCache = make(map[string]thinker.Thinker)
	r.thinker = r.getOrCreateThinker(r.llmClient, "")
}

// RegisterTools 注册多个工具
func (r *reactor) RegisterTools(ts ...tools.Tool) {
	for _, t := range ts {
		r.toolManager.RegisterTool(t)
	}
	// 清空 Thinker 缓存以使用新的工具描述
	r.thinkerCache = make(map[string]thinker.Thinker)
	r.thinker = r.getOrCreateThinker(r.llmClient, "")
}
