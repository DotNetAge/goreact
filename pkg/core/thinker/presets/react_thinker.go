package presets

import (
	"fmt"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/core/thinker/components"
	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/types"
)

// ReActThinker 标准 ReAct 模式的 Thinker
type ReActThinker struct {
	llmClient      llm.Client
	tools          []thinker.ToolDesc
	promptBuilder  *components.DefaultPromptBuilder
	contextManager *components.DefaultContextManager
	responseParser *components.ReActParser
}

// NewReActThinker 创建 ReAct Thinker
func NewReActThinker(llmClient llm.Client, toolDescriptions string) *ReActThinker {
	return &ReActThinker{
		llmClient:      llmClient,
		tools:          parseToolDescriptions(toolDescriptions),
		promptBuilder:  components.NewPromptBuilder(components.ReActTemplate),
		contextManager: components.NewContextManager(4096), // 默认 4k tokens
		responseParser: components.NewReActParser(),
	}
}

// NewReActThinkerWithTools 使用工具描述列表创建 ReAct Thinker
func NewReActThinkerWithTools(llmClient llm.Client, tools []thinker.ToolDesc) *ReActThinker {
	return &ReActThinker{
		llmClient:      llmClient,
		tools:          tools,
		promptBuilder:  components.NewPromptBuilder(components.ReActTemplate),
		contextManager: components.NewContextManager(4096),
		responseParser: components.NewReActParser(),
	}
}

// Think 执行思考
func (t *ReActThinker) Think(task string, context *core.Context) (*types.Thought, error) {
	// 1. 构建 prompt
	prompt := t.promptBuilder.Build(task, context, t.tools)

	// 2. 调用 LLM
	response, err := t.llmClient.Generate(prompt.String())
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// 3. 解析响应
	thought, err := t.responseParser.Parse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 4. 更新上下文管理器
	t.contextManager.AddTurn(&thinker.Turn{
		Role:    "assistant",
		Content: response,
	})

	return thought, nil
}

// SetTemplate 设置自定义模板
func (t *ReActThinker) SetTemplate(template *thinker.PromptTemplate) {
	t.promptBuilder.SetTemplate(template)
}

// SetMaxTokens 设置最大 token 数
func (t *ReActThinker) SetMaxTokens(maxTokens int) {
	t.contextManager = components.NewContextManager(maxTokens)
}

// parseToolDescriptions 解析工具描述字符串为 ToolDesc 列表
func parseToolDescriptions(toolDesc string) []thinker.ToolDesc {
	// 简单实现：将整个描述作为一个工具
	// 实际使用中，应该从 ToolManager 获取结构化的工具列表
	return []thinker.ToolDesc{
		{
			Name:        "tools",
			Description: toolDesc,
		},
	}
}
