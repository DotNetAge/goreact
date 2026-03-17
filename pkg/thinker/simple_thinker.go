package thinker

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// SimpleThinker 最简单的 Thinker 实现（原 DefaultThinker）
// 适合快速开始和测试
type SimpleThinker struct {
	llmClient    gochatcore.Client
	toolDesc     string
	systemPrompt string // System Prompt（可选）
}

// NewSimpleThinker 创建简单 Thinker
func NewSimpleThinker(llmClient gochatcore.Client, toolDesc string) *SimpleThinker {
	return &SimpleThinker{
		llmClient: llmClient,
		toolDesc:  toolDesc,
	}
}

// NewSimpleThinkerWithSystemPrompt 创建带 System Prompt 的 Thinker
func NewSimpleThinkerWithSystemPrompt(llmClient gochatcore.Client, toolDesc, systemPrompt string) *SimpleThinker {
	return &SimpleThinker{
		llmClient:    llmClient,
		toolDesc:     toolDesc,
		systemPrompt: systemPrompt,
	}
}

// Think 执行思考
func (t *SimpleThinker) Think(task string, context *core.Context) (*types.Thought, error) {
	// 构建 prompt（直接从 context 中读取累积的历史记录）
	prompt := t.buildPrompt(task, context)

	// 调用 LLM（使用 gochat 的 Chat 接口）
	messages := []gochatcore.Message{
		gochatcore.NewUserMessage(prompt),
	}
	response, err := t.llmClient.Chat(context.Context(), messages)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// 解析响应
	thought := t.parseResponse(response.Content)
	thought.Usage = response.Usage // 原生利用 gochat 的 Usage
	return thought, nil
}

// buildPrompt 构建 LLM prompt
func (t *SimpleThinker) buildPrompt(task string, context *core.Context) string {
	var sb strings.Builder

	// 添加 System Prompt（如果有）
	if t.systemPrompt != "" {
		sb.WriteString(t.systemPrompt)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("You are a helpful AI assistant that uses tools to solve tasks.\n\n")
	}

	sb.WriteString(t.toolDesc)
	sb.WriteString("\n\n")
	sb.WriteString("Task: " + task + "\n\n")

	// 添加历史记录
	if historySteps, ok := context.Get("history_steps"); ok {
		if steps, ok := historySteps.([]map[string]string); ok && len(steps) > 0 {
			sb.WriteString("Previous steps:\n")
			for i, step := range steps {
				sb.WriteString(fmt.Sprintf("Step %d:\n", i+1))
				sb.WriteString(fmt.Sprintf("  Action: %s\n", step["action"]))
				sb.WriteString(fmt.Sprintf("  Result: %s\n", step["result"]))
				sb.WriteString(fmt.Sprintf("  Feedback: %s\n", step["feedback"]))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("Please respond in the following format:\n")
	sb.WriteString("Thought: [your reasoning]\n")
	sb.WriteString("Action: [tool_name] (or leave empty if task is complete)\n")
	sb.WriteString("Parameters: {\"key\": \"value\"} (JSON format)\n")
	sb.WriteString("Reasoning: [why you chose this action]\n")
	sb.WriteString("OR\n")
	sb.WriteString("Thought: [your reasoning]\n")
	sb.WriteString("Final Answer: [your final answer]\n")

	return sb.String()
}

// parseResponse 解析 LLM 响应
func (t *SimpleThinker) parseResponse(response string) *types.Thought {
	thought := &types.Thought{
		Metadata: make(map[string]any),
	}

	// 提取 Thought
	thoughtRegex := regexp.MustCompile(`(?i)Thought:\s*(.+?)(?:\n|$)`)
	if matches := thoughtRegex.FindStringSubmatch(response); len(matches) > 1 {
		thought.Reasoning = strings.TrimSpace(matches[1])
	}

	// 检查是否是最终答案
	finalAnswerRegex := regexp.MustCompile(`(?i)Final Answer:\s*(.+?)(?:\n|$)`)
	if matches := finalAnswerRegex.FindStringSubmatch(response); len(matches) > 1 {
		thought.ShouldFinish = true
		thought.FinalAnswer = strings.TrimSpace(matches[1])
		return thought
	}

	// 提取 Action
	actionRegex := regexp.MustCompile(`(?i)Action:\s*(.+?)(?:\n|$)`)
	if matches := actionRegex.FindStringSubmatch(response); len(matches) > 1 {
		actionName := strings.TrimSpace(matches[1])
		if actionName != "" && actionName != "none" {
			action := &types.Action{
				ToolName: actionName,
			}

			// 提取 Parameters
			paramsRegex := regexp.MustCompile(`(?i)Parameters:\s*(\{.+?\})`)
			if paramMatches := paramsRegex.FindStringSubmatch(response); len(paramMatches) > 1 {
				// 简单的 JSON 解析（实际应该用 json.Unmarshal）
				paramsStr := paramMatches[1]
				action.Parameters = t.parseSimpleJSON(paramsStr)
			}

			// 提取 Reasoning
			reasoningRegex := regexp.MustCompile(`(?i)Reasoning:\s*(.+?)(?:\n|$)`)
			if reasoningMatches := reasoningRegex.FindStringSubmatch(response); len(reasoningMatches) > 1 {
				action.Reasoning = strings.TrimSpace(reasoningMatches[1])
			}

			thought.Action = action
		}
	}

	return thought
}

// parseSimpleJSON 解析 JSON 字符串为 map
func (t *SimpleThinker) parseSimpleJSON(jsonStr string) map[string]any {
	var params map[string]any

	// 使用标准 JSON 解析
	if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
		// 如果解析失败，返回空 map，避免中断执行
		// 可以在这里记录日志，但不抛出错误
		return make(map[string]any)
	}

	return params
}
