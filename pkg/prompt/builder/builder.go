package builder

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ray/goreact/pkg/prompt"
	"github.com/ray/goreact/pkg/prompt/formatter"
)

// Turn 对话轮次
type Turn struct {
	Role    string
	Content string
}

// FewShotExample Few-Shot 示例
type FewShotExample struct {
	Task       string
	Thought    string
	Action     string
	Parameters map[string]any
	Result     string
}

// FluentPromptBuilder 流式 API 的 Prompt 构建器
type FluentPromptBuilder struct {
	systemPrompt     string
	userPrompt       string
	task             string
	tools            []formatter.ToolDesc
	history          []Turn
	fewShots         []FewShotExample
	variables        map[string]any
	toolFormatter    formatter.ToolFormatter
	historyFormatter HistoryFormatter
	maxTokens        int
	tokenCounter     prompt.TokenCounter
	systemTemplate   string
	userTemplate     string
}

// New 创建新的 FluentPromptBuilder
func New() *FluentPromptBuilder {
	return &FluentPromptBuilder{
		variables:        make(map[string]any),
		toolFormatter:    formatter.NewSimpleTextFormatter(),
		historyFormatter: NewSimpleHistoryFormatter(),
		maxTokens:        4096,
		systemTemplate:   DefaultSystemTemplate,
		userTemplate:     DefaultUserTemplate,
	}
}

// WithSystemPrompt 设置系统提示词
func (b *FluentPromptBuilder) WithSystemPrompt(prompt string) *FluentPromptBuilder {
	b.systemPrompt = prompt
	return b
}

// WithSystemTemplate 设置系统提示词模板
func (b *FluentPromptBuilder) WithSystemTemplate(tmpl string) *FluentPromptBuilder {
	b.systemTemplate = tmpl
	return b
}

// WithUserTemplate 设置用户提示词模板
func (b *FluentPromptBuilder) WithUserTemplate(tmpl string) *FluentPromptBuilder {
	b.userTemplate = tmpl
	return b
}

// WithTask 设置任务
func (b *FluentPromptBuilder) WithTask(task string) *FluentPromptBuilder {
	b.task = task
	return b
}

// WithTools 设置工具列表
func (b *FluentPromptBuilder) WithTools(tools []formatter.ToolDesc) *FluentPromptBuilder {
	b.tools = tools
	return b
}

// WithHistory 设置历史记录
func (b *FluentPromptBuilder) WithHistory(history []Turn) *FluentPromptBuilder {
	b.history = history
	return b
}

// WithFewShots 设置 Few-Shot 示例
func (b *FluentPromptBuilder) WithFewShots(examples []FewShotExample) *FluentPromptBuilder {
	b.fewShots = examples
	return b
}

// WithVariable 添加自定义变量
func (b *FluentPromptBuilder) WithVariable(key string, value any) *FluentPromptBuilder {
	b.variables[key] = value
	return b
}

// WithToolFormatter 设置工具格式化器
func (b *FluentPromptBuilder) WithToolFormatter(formatter formatter.ToolFormatter) *FluentPromptBuilder {
	b.toolFormatter = formatter
	return b
}

// WithHistoryFormatter 设置历史格式化器
func (b *FluentPromptBuilder) WithHistoryFormatter(formatter HistoryFormatter) *FluentPromptBuilder {
	b.historyFormatter = formatter
	return b
}

// WithMaxTokens 设置最大 token 数
func (b *FluentPromptBuilder) WithMaxTokens(maxTokens int) *FluentPromptBuilder {
	b.maxTokens = maxTokens
	return b
}

// WithTokenCounter 设置 token 计数器
func (b *FluentPromptBuilder) WithTokenCounter(counter prompt.TokenCounter) *FluentPromptBuilder {
	b.tokenCounter = counter
	return b
}

// Build 构建 Prompt
func (b *FluentPromptBuilder) Build() *prompt.Prompt {
	// 准备变量
	vars := b.prepareVariables()

	// 渲染模板
	system := b.renderTemplate(b.systemTemplate, vars)
	user := b.renderTemplate(b.userTemplate, vars)

	// 如果设置了自定义 systemPrompt，使用它
	if b.systemPrompt != "" {
		system = b.systemPrompt
	}

	return &prompt.Prompt{
		System: system,
		User:   user,
	}
}

// prepareVariables 准备模板变量
func (b *FluentPromptBuilder) prepareVariables() map[string]any {
	vars := make(map[string]any)

	// 复制自定义变量
	for k, v := range b.variables {
		vars[k] = v
	}

	// 添加标准变量
	vars["task"] = b.task
	vars["tools"] = b.formatTools()
	vars["history"] = b.formatHistory()
	vars["few_shots"] = b.formatFewShots()

	// 添加统计信息
	vars["tools_count"] = len(b.tools)
	vars["history_count"] = len(b.history)
	vars["few_shots_count"] = len(b.fewShots)

	return vars
}

// formatTools 格式化工具列表
func (b *FluentPromptBuilder) formatTools() string {
	if len(b.tools) == 0 {
		return "No tools available"
	}
	return b.toolFormatter.Format(b.tools)
}

// formatHistory 格式化历史记录
func (b *FluentPromptBuilder) formatHistory() string {
	if len(b.history) == 0 {
		return ""
	}
	return b.historyFormatter.Format(b.history)
}

// formatFewShots 格式化 Few-Shot 示例
func (b *FluentPromptBuilder) formatFewShots() string {
	if len(b.fewShots) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, example := range b.fewShots {
		sb.WriteString(fmt.Sprintf("Example %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("Task: %s\n", example.Task))
		sb.WriteString(fmt.Sprintf("Thought: %s\n", example.Thought))
		sb.WriteString(fmt.Sprintf("Action: %s\n", example.Action))
		if len(example.Parameters) > 0 {
			sb.WriteString(fmt.Sprintf("Parameters: %v\n", example.Parameters))
		}
		sb.WriteString(fmt.Sprintf("Result: %s\n\n", example.Result))
	}

	return sb.String()
}

// renderTemplate 渲染模板
func (b *FluentPromptBuilder) renderTemplate(tmpl string, vars map[string]any) string {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return tmpl
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return tmpl
	}

	return buf.String()
}

// HistoryFormatter 历史记录格式化器接口
type HistoryFormatter interface {
	Format(history []Turn) string
}

// SimpleHistoryFormatter 简单历史格式化器
type SimpleHistoryFormatter struct{}

func NewSimpleHistoryFormatter() *SimpleHistoryFormatter {
	return &SimpleHistoryFormatter{}
}

func (f *SimpleHistoryFormatter) Format(history []Turn) string {
	if len(history) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, turn := range history {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", turn.Role, turn.Content))
	}
	return sb.String()
}

// ConversationalFormatter 对话式格式化器
type ConversationalFormatter struct{}

func NewConversationalFormatter() *ConversationalFormatter {
	return &ConversationalFormatter{}
}

func (f *ConversationalFormatter) Format(history []Turn) string {
	if len(history) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, turn := range history {
		role := strings.Title(turn.Role)
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", role, turn.Content))
	}
	return sb.String()
}

// MarkdownHistoryFormatter Markdown 格式化器
type MarkdownHistoryFormatter struct{}

func NewMarkdownHistoryFormatter() *MarkdownHistoryFormatter {
	return &MarkdownHistoryFormatter{}
}

func (f *MarkdownHistoryFormatter) Format(history []Turn) string {
	if len(history) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, turn := range history {
		role := strings.Title(turn.Role)
		sb.WriteString(fmt.Sprintf("**%s:** %s\n\n", role, turn.Content))
	}
	return sb.String()
}

// 默认模板

// DefaultSystemTemplate 默认系统模板
const DefaultSystemTemplate = `You are a helpful AI assistant that uses tools to solve tasks.

Available tools:
{{.tools}}

Follow the ReAct pattern:
1. Thought: Analyze the task and decide what to do
2. Action: Choose a tool and specify parameters
3. Observation: Review the result from the tool
4. Repeat until the task is complete

When you have the final answer, respond with:
Thought: [your final reasoning]
Final Answer: [the answer]`

// DefaultUserTemplate 默认用户模板
const DefaultUserTemplate = `Task: {{.task}}
{{if .history}}
Previous conversation:
{{.history}}
{{end}}
{{if .few_shots}}
Here are some examples:
{{.few_shots}}
{{end}}
Please respond in this format:
Thought: [your reasoning]
Action: [tool_name]
Parameters: {"key": "value"}
Reasoning: [why you chose this action]

OR if the task is complete:
Thought: [your final reasoning]
Final Answer: [your answer]`
