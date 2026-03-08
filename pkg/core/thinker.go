package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/types"
)

// Thinker 思考模块接口
type Thinker interface {
	Think(task string, context *Context) (*types.Thought, error)
}

// DefaultThinker 默认思考模块实现
type DefaultThinker struct {
	llmClient llm.Client
	toolDesc  string
}

// NewDefaultThinker 创建默认思考模块
func NewDefaultThinker(llmClient llm.Client, toolDesc string) *DefaultThinker {
	return &DefaultThinker{
		llmClient: llmClient,
		toolDesc:  toolDesc,
	}
}

// Think 执行思考
func (t *DefaultThinker) Think(task string, context *Context) (*types.Thought, error) {
	// 从 Context 中获取历史信息
	var history string
	if lastAction, ok := context.Get("last_action"); ok {
		if action, ok := lastAction.(*types.Action); ok {
			history += fmt.Sprintf("Action: %s with params %v\n", action.ToolName, action.Parameters)
		}
	}
	if lastResult, ok := context.Get("last_result"); ok {
		if result, ok := lastResult.(*types.ExecutionResult); ok {
			history += fmt.Sprintf("Result: %v (Success: %v)\n", result.Output, result.Success)
		}
	}
	if lastFeedback, ok := context.Get("last_feedback"); ok {
		if feedback, ok := lastFeedback.(*types.Feedback); ok {
			history += fmt.Sprintf("Feedback: %s\n", feedback.Message)
		}
	}

	// 如果有历史信息，添加到上下文
	if history != "" {
		context.Set("history", history)
	}

	// 构建 prompt
	prompt := t.buildPrompt(task, context)

	// 调用 LLM
	response, err := t.llmClient.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// 解析响应
	thought := t.parseResponse(response)
	return thought, nil
}

// buildPrompt 构建 LLM prompt
func (t *DefaultThinker) buildPrompt(task string, context *Context) string {
	var sb strings.Builder

	sb.WriteString("You are a helpful AI assistant that uses tools to solve tasks.\n\n")
	sb.WriteString(t.toolDesc)
	sb.WriteString("\n\n")
	sb.WriteString("Task: " + task + "\n\n")

	// 添加历史记录
	if history, ok := context.Get("history"); ok {
		if historyStr, ok := history.(string); ok && historyStr != "" {
			sb.WriteString("Previous steps:\n")
			sb.WriteString(historyStr)
			sb.WriteString("\n\n")
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
func (t *DefaultThinker) parseResponse(response string) *types.Thought {
	thought := &types.Thought{
		Metadata: make(map[string]interface{}),
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

// parseSimpleJSON 简单的 JSON 解析（仅用于演示）
func (t *DefaultThinker) parseSimpleJSON(jsonStr string) map[string]interface{} {
	params := make(map[string]interface{})

	// 移除花括号
	jsonStr = strings.Trim(jsonStr, "{}")

	// 分割键值对
	pairs := strings.Split(jsonStr, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, ":")
		if len(kv) == 2 {
			key := strings.Trim(strings.TrimSpace(kv[0]), "\"")
			value := strings.TrimSpace(kv[1])

			// 移除引号
			value = strings.Trim(value, "\"")

			// 尝试转换为数字
			if num, err := parseNumber(value); err == nil {
				params[key] = num
			} else {
				params[key] = value
			}
		}
	}

	return params
}

// parseNumber 解析数字
func parseNumber(s string) (interface{}, error) {
	// 尝试解析为整数
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
		return i, nil
	}

	// 尝试解析为浮点数
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return f, nil
	}

	return nil, fmt.Errorf("not a number")
}
