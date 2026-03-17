package components

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ray/goreact/pkg/types"
)

// ResponseParser 响应解析器接口
type ResponseParser interface {
	// Parse 解析 LLM 响应
	Parse(response string) (*types.Thought, error)
}

// ReActParser ReAct 格式解析器
type ReActParser struct{}

// NewReActParser 创建 ReAct 解析器
func NewReActParser() *ReActParser {
	return &ReActParser{}
}

// Parse 解析 ReAct 格式的响应
func (p *ReActParser) Parse(response string) (*types.Thought, error) {
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
		return thought, nil
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
				paramsStr := paramMatches[1]
				params, err := parseJSON(paramsStr)
				if err == nil {
					action.Parameters = params
				} else {
					// 如果 JSON 解析失败，尝试简单解析
					action.Parameters = parseSimpleJSON(paramsStr)
				}
			}

			// 提取 Reasoning
			reasoningRegex := regexp.MustCompile(`(?i)Reasoning:\s*(.+?)(?:\n|$)`)
			if reasoningMatches := reasoningRegex.FindStringSubmatch(response); len(reasoningMatches) > 1 {
				action.Reasoning = strings.TrimSpace(reasoningMatches[1])
			}

			thought.Action = action
		}
	}

	return thought, nil
}

// JSONParser JSON 格式解析器
type JSONParser struct{}

// NewJSONParser 创建 JSON 解析器
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse 解析 JSON 格式的响应
func (p *JSONParser) Parse(response string) (*types.Thought, error) {
	var data struct {
		Thought     string                 `json:"thought"`
		Action      string                 `json:"action,omitempty"`
		Parameters  map[string]interface{} `json:"parameters,omitempty"`
		Reasoning   string                 `json:"reasoning,omitempty"`
		FinalAnswer string                 `json:"final_answer,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	thought := &types.Thought{
		Reasoning: data.Thought,
		Metadata:  make(map[string]interface{}),
	}

	if data.FinalAnswer != "" {
		thought.ShouldFinish = true
		thought.FinalAnswer = data.FinalAnswer
	} else if data.Action != "" {
		thought.Action = &types.Action{
			ToolName:   data.Action,
			Parameters: data.Parameters,
			Reasoning:  data.Reasoning,
		}
	}

	return thought, nil
}

// parseJSON 解析 JSON 字符串
func parseJSON(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

// parseSimpleJSON 简单的 JSON 解析（容错）
func parseSimpleJSON(jsonStr string) map[string]interface{} {
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
