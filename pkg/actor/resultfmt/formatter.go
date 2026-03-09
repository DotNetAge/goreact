package resultfmt

import (
	"fmt"
	"strings"
)

// Formatter 结果格式化器
type Formatter struct {
	maxLength int
}

// New 创建格式化器
func New(opts ...Option) *Formatter {
	f := &Formatter{
		maxLength: 1000, // 默认 1000 字符
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Option 配置选项
type Option func(*Formatter)

// WithMaxLength 设置最大长度
func WithMaxLength(length int) Option {
	return func(f *Formatter) {
		f.maxLength = length
	}
}

// Format 格式化结果
func (f *Formatter) Format(result any) string {
	str := fmt.Sprintf("%v", result)

	if len(str) <= f.maxLength {
		return str
	}

	// 截断并添加提示
	truncated := str[:f.maxLength]
	return fmt.Sprintf("%s... (truncated, showing first %d of %d chars)",
		truncated, f.maxLength, len(str))
}

// ErrorFormatter 错误格式化器
type ErrorFormatter struct{}

// NewErrorFormatter 创建错误格式化器
func NewErrorFormatter() *ErrorFormatter {
	return &ErrorFormatter{}
}

// Format 格式化错误为 LLM 友好的消息
func (f *ErrorFormatter) Format(err error, toolName string, params map[string]any) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// 检测常见错误模式并转换为友好消息
	if strings.Contains(errMsg, "connection refused") {
		return f.formatConnectionRefused(toolName, params)
	}

	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return f.formatTimeout(toolName, params)
	}

	if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "dns") {
		return f.formatDNSError(toolName, params)
	}

	if strings.Contains(errMsg, "permission denied") {
		return f.formatPermissionDenied(toolName, params)
	}

	// 默认格式化
	return fmt.Sprintf("Tool '%s' failed: %s\n\nSuggestion: Check the parameters and try again.", toolName, errMsg)
}

func (f *ErrorFormatter) formatConnectionRefused(toolName string, params map[string]any) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("❌ Connection refused when executing '%s'.\n\n", toolName))
	sb.WriteString("This usually means:\n")
	sb.WriteString("1. The service is not running\n")
	sb.WriteString("2. The port number is incorrect\n")
	sb.WriteString("3. A firewall is blocking the connection\n\n")
	sb.WriteString("Suggestions:\n")
	sb.WriteString("- Check if the service is running\n")
	sb.WriteString("- Verify the URL/port is correct\n")
	sb.WriteString("- Try a different endpoint\n")

	if url, ok := params["url"]; ok {
		sb.WriteString(fmt.Sprintf("\nAttempted URL: %v", url))
	}

	return sb.String()
}

func (f *ErrorFormatter) formatTimeout(toolName string, params map[string]any) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⏱️ Timeout when executing '%s'.\n\n", toolName))
	sb.WriteString("The operation took too long to complete.\n\n")
	sb.WriteString("Suggestions:\n")
	sb.WriteString("- The service might be slow or overloaded\n")
	sb.WriteString("- Try again later\n")
	sb.WriteString("- Use a different endpoint if available\n")
	return sb.String()
}

func (f *ErrorFormatter) formatDNSError(toolName string, params map[string]any) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌐 DNS resolution failed for '%s'.\n\n", toolName))
	sb.WriteString("The hostname could not be resolved.\n\n")
	sb.WriteString("Suggestions:\n")
	sb.WriteString("- Check if the hostname is correct\n")
	sb.WriteString("- Verify your internet connection\n")
	sb.WriteString("- The service might be down\n")

	if url, ok := params["url"]; ok {
		sb.WriteString(fmt.Sprintf("\nAttempted URL: %v", url))
	}

	return sb.String()
}

func (f *ErrorFormatter) formatPermissionDenied(toolName string, params map[string]any) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔒 Permission denied when executing '%s'.\n\n", toolName))
	sb.WriteString("You don't have permission to perform this operation.\n\n")
	sb.WriteString("Suggestions:\n")
	sb.WriteString("- Check if you have the necessary permissions\n")
	sb.WriteString("- Try a different approach that doesn't require elevated privileges\n")
	sb.WriteString("- Ask the user for permission\n")
	return sb.String()
}
