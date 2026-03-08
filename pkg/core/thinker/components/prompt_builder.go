package components

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
)

// PromptBuilder 提示词构建器接口
type PromptBuilder interface {
	// Build 构建完整 prompt
	Build(task string, context *core.Context, tools []thinker.ToolDesc) *thinker.Prompt

	// SetTemplate 设置模板
	SetTemplate(template *thinker.PromptTemplate)

	// AddVariable 添加变量
	AddVariable(key string, value interface{})
}

// DefaultPromptBuilder 默认提示词构建器
type DefaultPromptBuilder struct {
	template  *thinker.PromptTemplate
	variables map[string]interface{}
}

// NewPromptBuilder 创建提示词构建器
func NewPromptBuilder(tmpl *thinker.PromptTemplate) *DefaultPromptBuilder {
	if tmpl == nil {
		tmpl = ReActTemplate
	}
	return &DefaultPromptBuilder{
		template:  tmpl,
		variables: make(map[string]interface{}),
	}
}

// SetTemplate 设置模板
func (b *DefaultPromptBuilder) SetTemplate(tmpl *thinker.PromptTemplate) {
	b.template = tmpl
}

// AddVariable 添加变量
func (b *DefaultPromptBuilder) AddVariable(key string, value interface{}) {
	b.variables[key] = value
}

// Build 构建完整 prompt
func (b *DefaultPromptBuilder) Build(task string, context *core.Context, tools []thinker.ToolDesc) *thinker.Prompt {
	// 准备变量
	vars := make(map[string]interface{})
	for k, v := range b.variables {
		vars[k] = v
	}

	// 添加标准变量
	vars["task"] = task
	vars["tools"] = b.formatTools(tools)

	// 添加历史记录
	if history, ok := context.Get("history"); ok {
		if historyStr, ok := history.(string); ok && historyStr != "" {
			vars["history"] = historyStr
		}
	}

	// 渲染模板
	systemPrompt := b.renderTemplate(b.template.System, vars)
	userPrompt := b.renderTemplate(b.template.User, vars)

	return &thinker.Prompt{
		System: systemPrompt,
		User:   userPrompt,
	}
}

// renderTemplate 渲染模板
func (b *DefaultPromptBuilder) renderTemplate(tmpl string, vars map[string]interface{}) string {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		// 如果模板解析失败，返回原始字符串
		return tmpl
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return tmpl
	}

	return buf.String()
}

// formatTools 格式化工具描述
func (b *DefaultPromptBuilder) formatTools(tools []thinker.ToolDesc) string {
	if len(tools) == 0 {
		return "No tools available"
	}

	var sb strings.Builder
	for i, tool := range tools {
		sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, tool.Name, tool.Description))
	}
	return sb.String()
}

// 内置模板

// ReActTemplate 标准 ReAct 模板
var ReActTemplate = &thinker.PromptTemplate{
	System: `You are a helpful AI assistant that uses tools to solve tasks.

Available tools:
{{.tools}}

Follow the ReAct pattern:
1. Thought: Analyze the task and decide what to do
2. Action: Choose a tool and specify parameters
3. Observation: Review the result from the tool
4. Repeat until the task is complete

When you have the final answer, respond with:
Thought: [your final reasoning]
Final Answer: [the answer]`,

	User: `Task: {{.task}}
{{if .history}}
Previous steps:
{{.history}}
{{end}}
Please respond in this format:
Thought: [your reasoning]
Action: [tool_name]
Parameters: {"key": "value"}
Reasoning: [why you chose this action]

OR if the task is complete:
Thought: [your final reasoning]
Final Answer: [your answer]`,
}

// ConversationalTemplate 对话式模板
var ConversationalTemplate = &thinker.PromptTemplate{
	System: `You are a friendly and helpful AI assistant.

You can use the following tools when needed:
{{.tools}}

Guidelines:
- Be conversational and natural
- Ask clarifying questions if the user's intent is unclear
- Use tools only when necessary to accomplish the user's goal
- Explain your actions in a friendly way`,

	User: `User: {{.task}}
{{if .history}}
Conversation history:
{{.history}}
{{end}}
Please respond naturally. If you need to use a tool, format it as:
[TOOL: tool_name]
[PARAMS: {"key": "value"}]
[REASON: why you're using this tool]

Otherwise, just respond conversationally.`,
}

// PlanningTemplate 规划式模板
var PlanningTemplate = &thinker.PromptTemplate{
	System: `You are a strategic AI assistant that plans before acting.

Available tools:
{{.tools}}

Your approach:
1. First, create a plan to solve the task
2. Break down complex tasks into steps
3. Execute each step systematically
4. Verify results before proceeding`,

	User: `Task: {{.task}}
{{if .plan}}
Current plan:
{{.plan}}

Current step: {{.current_step}}
{{end}}
{{if .history}}
Execution history:
{{.history}}
{{end}}
Respond with:
Plan: [your step-by-step plan]
Current Action: [tool_name]
Parameters: {"key": "value"}

OR if planning is complete:
Final Answer: [the result]`,
}

// MinimalTemplate 最简模板（适合快速测试）
var MinimalTemplate = &thinker.PromptTemplate{
	System: `You are an AI assistant. Available tools: {{.tools}}`,
	User:   `Task: {{.task}}\n\nRespond with: Thought: [reasoning]\nAction: [tool]\nParameters: {}\nOR\nFinal Answer: [answer]`,
}
