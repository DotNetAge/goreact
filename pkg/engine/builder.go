package engine

import (
	"github.com/ray/goreact/pkg/core/thinker"
)

// NewReactor 创建 ReAct 引擎实例
func NewReactor(opts ...ReactorOption) Reactor {
	options := DefaultReactorOptions()
	for _, opt := range opts {
		opt(options)
	}

	// 如果 logger 为 nil，使用 no-op logger
	if options.Logger == nil {
		options.Logger = &noOpLogger{}
	}

	return &reactor{
		logger:         options.Logger,
		metrics:        options.Metrics,
		llmClient:      options.LLMClient,
		modelManager:   options.ModelManager,
		toolManager:    options.ToolManager,
		agentManager:   options.AgentManager,
		skillManager:   options.SkillManager,
		cache:          options.Cache,
		thinker:        options.Thinker,
		actor:          options.Actor,
		observer:       options.Observer,
		loopController: options.LoopController,
		maxRetries:     options.MaxRetries,
		retryInterval:  options.RetryInterval,
		maxTraceSize:   options.MaxTraceSize,
		thinkerCache:   make(map[string]thinker.Thinker),
	}
}
