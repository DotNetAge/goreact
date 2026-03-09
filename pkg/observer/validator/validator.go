package validator

import (
	"fmt"
	"strings"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid     bool
	Issues      []string
	Suggestions []string
}

// Rule 验证规则接口
type Rule interface {
	Check(result *types.ExecutionResult, ctx *core.Context) (issues []string, suggestions []string)
}

// Validator 结果验证器
type Validator struct {
	rules []Rule
}

// Option 配置选项
type Option func(*Validator)

// New 创建验证器
func New(opts ...Option) *Validator {
	v := &Validator{}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// WithHTTPStatusRule 添加 HTTP 状态码规则
func WithHTTPStatusRule() Option {
	return func(v *Validator) { v.rules = append(v.rules, &httpStatusRule{}) }
}

// WithErrorPatternRule 添加错误模式规则
func WithErrorPatternRule() Option {
	return func(v *Validator) { v.rules = append(v.rules, &errorPatternRule{}) }
}

// WithEmptyResultRule 添加空结果规则
func WithEmptyResultRule() Option {
	return func(v *Validator) { v.rules = append(v.rules, &emptyResultRule{}) }
}

// Validate 验证结果
func (v *Validator) Validate(result *types.ExecutionResult, ctx *core.Context) ValidationResult {
	vr := ValidationResult{IsValid: true}

	// 执行失败直接标记无效
	if !result.Success {
		vr.IsValid = false
		vr.Issues = append(vr.Issues, "execution failed")
		return vr
	}

	// 运行所有规则
	for _, rule := range v.rules {
		issues, suggestions := rule.Check(result, ctx)
		if len(issues) > 0 {
			vr.IsValid = false
			vr.Issues = append(vr.Issues, issues...)
		}
		vr.Suggestions = append(vr.Suggestions, suggestions...)
	}

	return vr
}

// === HTTP Status Rule ===

type httpStatusRule struct{}

func (r *httpStatusRule) Check(result *types.ExecutionResult, ctx *core.Context) ([]string, []string) {
	output := fmt.Sprintf("%v", result.Output)

	// 检测 4xx/5xx 状态码
	errorPatterns := []struct {
		pattern string
		message string
		suggest string
	}{
		{`"status": 404`, "HTTP 404 Not Found", "Check the URL path or list available resources"},
		{`"status":404`, "HTTP 404 Not Found", "Check the URL path or list available resources"},
		{`"status": 500`, "HTTP 500 Internal Server Error", "Try again later or use a different endpoint"},
		{`"status":500`, "HTTP 500 Internal Server Error", "Try again later or use a different endpoint"},
		{`"status": 401`, "HTTP 401 Unauthorized", "Check your API key or credentials"},
		{`"status":401`, "HTTP 401 Unauthorized", "Check your API key or credentials"},
		{`"status": 403`, "HTTP 403 Forbidden", "You don't have permission to access this resource"},
		{`"status":403`, "HTTP 403 Forbidden", "You don't have permission to access this resource"},
		{`"status": 429`, "HTTP 429 Too Many Requests", "Wait before retrying, you've been rate limited"},
		{`"status":429`, "HTTP 429 Too Many Requests", "Wait before retrying, you've been rate limited"},
		{`"status": 502`, "HTTP 502 Bad Gateway", "The upstream server is unavailable, try again later"},
		{`"status":502`, "HTTP 502 Bad Gateway", "The upstream server is unavailable, try again later"},
		{`"status": 503`, "HTTP 503 Service Unavailable", "The service is temporarily unavailable"},
		{`"status":503`, "HTTP 503 Service Unavailable", "The service is temporarily unavailable"},
	}

	var issues, suggestions []string
	for _, ep := range errorPatterns {
		if strings.Contains(output, ep.pattern) {
			issues = append(issues, ep.message)
			suggestions = append(suggestions, ep.suggest)
		}
	}

	return issues, suggestions
}

// === Error Pattern Rule ===

type errorPatternRule struct{}

func (r *errorPatternRule) Check(result *types.ExecutionResult, ctx *core.Context) ([]string, []string) {
	output := fmt.Sprintf("%v", result.Output)
	lower := strings.ToLower(output)

	patterns := []string{`"error"`, `"error":`, `"ERROR"`, `"FAILED"`}

	var issues []string
	for _, p := range patterns {
		if strings.Contains(output, p) || strings.Contains(lower, strings.ToLower(p)) {
			issues = append(issues, fmt.Sprintf("output contains error pattern: %s", p))
			break // 只报告一次
		}
	}

	return issues, nil
}

// === Empty Result Rule ===

type emptyResultRule struct{}

func (r *emptyResultRule) Check(result *types.ExecutionResult, ctx *core.Context) ([]string, []string) {
	if result.Output == nil {
		return nil, []string{"Result is empty, try different parameters"}
	}

	output := strings.TrimSpace(fmt.Sprintf("%v", result.Output))

	if output == "" || output == "[]" || output == "{}" || output == "null" || output == "<nil>" {
		return nil, []string{"Result is empty, try broader search terms or different parameters"}
	}

	return nil, nil
}
