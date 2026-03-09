package mock

import (
	"context"
	"strings"
)

// MockClient 模拟 LLM 客户端（用于测试）
type MockClient struct {
	responses []string
	callCount int
}

// NewMockClient 创建模拟客户端
func NewMockClient(responses []string) *MockClient {
	return &MockClient{
		responses: responses,
		callCount: 0,
	}
}

// Generate 生成预定义的响应
func (m *MockClient) Generate(ctx context.Context, prompt string) (string, error) {
	// 如果有预定义响应且还没用完，使用预定义响应
	if m.callCount < len(m.responses) && len(m.responses) > 0 {
		response := m.responses[m.callCount]
		m.callCount++
		return response, nil
	}

	// 否则，根据 prompt 智能生成响应
	if strings.Contains(prompt, "calculator") || strings.Contains(prompt, "calculate") || strings.Contains(prompt, "Calculate") {
		return m.generateCalculatorResponse(prompt), nil
	}

	// 默认响应
	return "Thought: I need more information to proceed.\nFinal Answer: Unable to complete the task.", nil
}

// generateCalculatorResponse 生成计算器相关的响应
func (m *MockClient) generateCalculatorResponse(prompt string) string {
	// 检测是否应该结束（已经有最终结果）
	if strings.Contains(prompt, "352") {
		return `Thought: I have calculated the final result.
Final Answer: The result of 15 * 23 + 7 is 352.`
	}

	// 检测是否是第二步（加法）- 已经有了第一步的结果 345
	if strings.Contains(prompt, "345") {
		return `Thought: Now I have 345, I need to add 7 to get the final result.
Action: calculator
Parameters: {"operation": "add", "a": 345, "b": 7}
Reasoning: Adding 7 to the previous result`
	}

	// 检测是否需要使用计算器（第一步）
	if strings.Contains(prompt, "15") && strings.Contains(prompt, "23") {
		return `Thought: I need to calculate 15 * 23 first, then add 7.
Action: calculator
Parameters: {"operation": "multiply", "a": 15, "b": 23}
Reasoning: First step is to multiply 15 by 23`
	}

	return "Thought: I need more information to proceed."
}

// Reset 重置调用计数
func (m *MockClient) Reset() {
	m.callCount = 0
}
