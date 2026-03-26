package agent

import (
	"fmt"

	"github.com/DotNetAge/goreact/pkg/actor"
	"github.com/DotNetAge/goreact/pkg/engine"
	"github.com/DotNetAge/goreact/pkg/memory"
	"github.com/DotNetAge/goreact/pkg/model"
	"github.com/DotNetAge/goreact/pkg/observer"
	"github.com/DotNetAge/goreact/pkg/skill"
	"github.com/DotNetAge/goreact/pkg/terminator"
	"github.com/DotNetAge/goreact/pkg/thinker"
	"github.com/DotNetAge/goreact/pkg/tools"
)

// Builder Agent 装配工厂
// 负责将纯数据的 Agent 配置与底层的引擎组件组装成一个可执行的 Runner
type Builder struct {
	modelManager *model.Manager
	tools        []tools.Tool // 全局注册的可用工具池
	skillManager skill.Manager
	memoryBank   memory.MemoryBank
}

// NewBuilder 创建一个新的装配工厂
func NewBuilder(modelManager *model.Manager) *Builder {
	return &Builder{
		modelManager: modelManager,
		tools:        make([]tools.Tool, 0),
	}
}

// WithTools 注册当前 Agent 环境可用的工具集
func (b *Builder) WithTools(t ...tools.Tool) *Builder {
	b.tools = append(b.tools, t...)
	return b
}

// WithSkillManager 注册演化与技能管理器
func (b *Builder) WithSkillManager(sm skill.Manager) *Builder {
	b.skillManager = sm
	return b
}

// WithMemoryBank 注册记忆体系
func (b *Builder) WithMemoryBank(mb memory.MemoryBank) *Builder {
	b.memoryBank = mb
	return b
}

// Build 将 Agent 数据配置装配成可执行的 Agent 实体
func (b *Builder) Build(agentConfig *Agent) (*Agent, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}

	// 1. 获取 LLM 大脑
	llmClient, err := b.modelManager.CreateLLMClient(agentConfig.ModelName)
	if err != nil {
		return nil, fmt.Errorf("failed to create llm client for model %s: %w", agentConfig.ModelName, err)
	}

	// 2. 装配思考者 (Thinker)
	// 注入 SystemPrompt 和 工具列表
	toolManager := tools.NewSimpleManager()

	// 注册 Builder 携带的全局/环境工具
	for _, tool := range b.tools {
		toolManager.Register(tool)
	}

	// 注册 Agent 自身可能携带的工具（如果未来支持 Agent 级工具定义）
	// 目前工具主要通过 Builder 注入

	t := thinker.Default(llmClient,
		thinker.WithModel(agentConfig.ModelName),
		thinker.WithToolManager(toolManager),
		thinker.WithSystemPrompt(agentConfig.SystemPrompt), // 明确注入 SystemPrompt
		thinker.WithMemoryBank(b.memoryBank),               // 注入记忆体
	)

	// 3. 装配执行者 (Actor)
	a := actor.Default(actor.WithToolManager(toolManager))

	// 4. 装配观察者与终结者 (Observer & Terminator)
	o := observer.Default()
	term := terminator.Default()

	// 5. 将这四大组件装配进 Reactor 引擎核心
	reactor := engine.NewReactor(
		engine.WithThinker(t),
		engine.WithActor(a),
		engine.WithObserver(o),
		engine.WithTerminator(term),
	)

	// 6. 将心脏(Reactor)与其它核心管理器注入到肉体(Agent)中
	agentConfig.reactor = reactor
	agentConfig.skillManager = b.skillManager
	agentConfig.memoryBank = b.memoryBank

	return agentConfig, nil
}
