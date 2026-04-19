package goreact

import (
	"github.com/DotNetAge/gochat"
	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/reactor"
)

type Agent struct {
	config  *core.AgentConfig
	model   *core.ModelConfig
	memory  *core.Memory
	reactor reactor.ReActor
	context *core.ContextWindow
}

func NewAgent(config *core.AgentConfig,
	model *core.ModelConfig,
	memory *core.Memory,
	reactor reactor.ReActor) *Agent {
	return &Agent{
		config:  config,
		model:   model,
		memory:  memory,
		reactor: reactor,
		context: &core.ContextWindow{},
	}
}

func (a *Agent) Name() string {
	return a.config.Name
}

func (a *Agent) Domain() string {
	return a.config.Domain
}

func (a *Agent) Description() string {
	return a.config.Description
}

// Ask 方法用于向 Agent 发送问题并获取回答
func (a *Agent) Ask(question string) (string, error) {
	// 多轮会话
	// TODO: 解析响应并返回
	return "", nil
}

func (a *Agent) Recognize(text string) (core.Intent, error) {
	// 识别用户意图
	recognizedPrompt := ""

	_, err := gochat.Client().
		Init(*a.model.Config()).
		SystemMessage(recognizedPrompt).
		UserMessage(text).
		GetResponse(gochat.OpenAIClient)

	if err != nil {
		return core.Intent{}, err
	}

	return core.Intent{}, nil
}
