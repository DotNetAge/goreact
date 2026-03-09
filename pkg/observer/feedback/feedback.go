package feedback

import (
	"fmt"
	"strings"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// SmartGenerator 智能反馈生成器
type SmartGenerator struct{}

// NewSmartGenerator 创建智能反馈生成器
func NewSmartGenerator() *SmartGenerator {
	return &SmartGenerator{}
}

// Generate 生成反馈
func (g *SmartGenerator) Generate(result *types.ExecutionResult, ctx *core.Context) string {
	toolName := g.getToolName(result)

	if !result.Success {
		return g.generateFailureFeedback(result, toolName)
	}

	outputStr := fmt.Sprintf("%v", result.Output)

	// 检测空结果
	if g.isEmpty(outputStr) {
		return g.generateEmptyFeedback(toolName)
	}

	// 检测 HTTP 错误状态码
	if g.containsHTTPError(outputStr) {
		return g.generateHTTPErrorFeedback(outputStr, toolName)
	}

	// 正常成功
	return g.generateSuccessFeedback(outputStr, toolName)
}

func (g *SmartGenerator) generateSuccessFeedback(output, toolName string) string {
	// 截断长输出
	display := output
	if len(display) > 200 {
		display = display[:200] + "..."
	}

	return fmt.Sprintf("✅ Tool '%s' executed successfully. Result: %s", toolName, display)
}

func (g *SmartGenerator) generateFailureFeedback(result *types.ExecutionResult, toolName string) string {
	errMsg := ""
	if result.Error != nil {
		errMsg = result.Error.Error()
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "❌ Tool '%s' failed: %s\n", toolName, errMsg)

	// 根据错误类型提供建议
	if strings.Contains(errMsg, "connection refused") {
		sb.WriteString("The service appears to be down. Suggestions:\n")
		sb.WriteString("1. Try a different endpoint\n")
		sb.WriteString("2. Check if the service is running\n")
		sb.WriteString("3. Use a different approach")
	} else if strings.Contains(errMsg, "timeout") {
		sb.WriteString("The operation timed out. Suggestions:\n")
		sb.WriteString("1. Try again later\n")
		sb.WriteString("2. Use a simpler query\n")
		sb.WriteString("3. Try a different approach")
	} else if strings.Contains(errMsg, "not found") {
		sb.WriteString("The resource was not found. Suggestions:\n")
		sb.WriteString("1. Check the name or path\n")
		sb.WriteString("2. List available resources first\n")
		sb.WriteString("3. Try a different search term")
	} else {
		sb.WriteString("Please try a different approach or adjust the parameters.")
	}

	return sb.String()
}

func (g *SmartGenerator) generateHTTPErrorFeedback(output, toolName string) string {
	var sb strings.Builder

	if strings.Contains(output, "404") {
		fmt.Fprintf(&sb, "⚠️ Tool '%s' completed but returned 404 Not Found.\n", toolName)
		sb.WriteString("The requested resource does not exist. Suggestions:\n")
		sb.WriteString("1. Check the URL path\n")
		sb.WriteString("2. List available resources first\n")
		sb.WriteString("3. Try a different endpoint")
	} else if strings.Contains(output, "500") {
		fmt.Fprintf(&sb, "⚠️ Tool '%s' completed but returned 500 Server Error.\n", toolName)
		sb.WriteString("The server encountered an internal error. Suggestions:\n")
		sb.WriteString("1. Try again later\n")
		sb.WriteString("2. Use a different endpoint")
	} else if strings.Contains(output, "401") || strings.Contains(output, "403") {
		fmt.Fprintf(&sb, "⚠️ Tool '%s' completed but returned an authentication error.\n", toolName)
		sb.WriteString("Suggestions:\n")
		sb.WriteString("1. Check your API key or credentials\n")
		sb.WriteString("2. Verify you have permission to access this resource")
	} else {
		fmt.Fprintf(&sb, "⚠️ Tool '%s' completed but the response may contain errors.\n", toolName)
		sb.WriteString("Please review the output carefully.")
	}

	return sb.String()
}

func (g *SmartGenerator) generateEmptyFeedback(toolName string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "⚠️ Tool '%s' returned an empty result.\n", toolName)
	sb.WriteString("Suggestions:\n")
	sb.WriteString("1. Try broader search terms\n")
	sb.WriteString("2. Check if the input parameters are correct\n")
	sb.WriteString("3. Try a different approach")
	return sb.String()
}

func (g *SmartGenerator) getToolName(result *types.ExecutionResult) string {
	if result.Metadata != nil {
		if name, ok := result.Metadata["tool_name"].(string); ok {
			return name
		}
	}
	return "unknown"
}

func (g *SmartGenerator) isEmpty(output string) bool {
	trimmed := strings.TrimSpace(output)
	return trimmed == "" || trimmed == "[]" || trimmed == "{}" || trimmed == "null" || trimmed == "<nil>"
}

func (g *SmartGenerator) containsHTTPError(output string) bool {
	errorCodes := []string{`"status": 4`, `"status": 5`, `"status":4`, `"status":5`}
	for _, code := range errorCodes {
		if strings.Contains(output, code) {
			return true
		}
	}
	return false
}
