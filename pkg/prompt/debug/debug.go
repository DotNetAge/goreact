package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/pkg/prompt"
)

// Logger 日志接口
type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	IsDebug() bool
}

// PromptDebugger Prompt 调试器
type PromptDebugger struct {
	enabled bool
	logger  Logger
	tracker *TokenTracker
}

// NewPromptDebugger 创建调试器
func NewPromptDebugger(enabled bool, logger Logger) *PromptDebugger {
	return &PromptDebugger{
		enabled: enabled,
		logger:  logger,
		tracker: NewTokenTracker(),
	}
}

// LogPrompt 记录 Prompt 信息
func (d *PromptDebugger) LogPrompt(p *prompt.Prompt, metadata map[string]any) {
	if !d.enabled || d.logger == nil {
		return
	}

	counter := d.getCounter(metadata)
	systemTokens := counter.Count(p.System)
	userTokens := counter.Count(p.User)
	totalTokens := systemTokens + userTokens

	// 更新追踪器
	d.tracker.SystemTokens = systemTokens
	d.tracker.UserTokens = userTokens
	d.tracker.TotalTokens = totalTokens

	// 从 metadata 提取信息
	toolsCount := d.getInt(metadata, "tools_count")
	historyTurns := d.getInt(metadata, "history_turns")
	fewShotsCount := d.getInt(metadata, "few_shots_count")

	d.logger.Info("Prompt Built",
		"system_tokens", systemTokens,
		"user_tokens", userTokens,
		"total_tokens", totalTokens,
		"tools_count", toolsCount,
		"history_turns", historyTurns,
		"few_shots_count", fewShotsCount,
	)

	// Debug 模式下输出完整内容
	if d.logger.IsDebug() {
		d.logger.Debug("Prompt Content",
			"system", d.truncate(p.System, 200),
			"user", d.truncate(p.User, 200),
		)
	}
}

// LogBuildTime 记录构建耗时
func (d *PromptDebugger) LogBuildTime(duration time.Duration) {
	if !d.enabled || d.logger == nil {
		return
	}

	d.logger.Info("Prompt Build Time",
		"duration_ms", duration.Milliseconds(),
	)
}

// GetTracker 获取 Token 追踪器
func (d *PromptDebugger) GetTracker() *TokenTracker {
	return d.tracker
}

func (d *PromptDebugger) getCounter(metadata map[string]any) prompt.TokenCounter {
	if counter, ok := metadata["token_counter"].(prompt.TokenCounter); ok {
		return counter
	}
	// 默认使用简单估算
	return &simpleCounter{}
}

func (d *PromptDebugger) getInt(metadata map[string]any, key string) int {
	if val, ok := metadata[key].(int); ok {
		return val
	}
	return 0
}

func (d *PromptDebugger) truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// TokenTracker Token 使用追踪器
type TokenTracker struct {
	SystemTokens   int
	UserTokens     int
	HistoryTokens  int
	ToolsTokens    int
	FewShotsTokens int
	TotalTokens    int
}

// NewTokenTracker 创建追踪器
func NewTokenTracker() *TokenTracker {
	return &TokenTracker{}
}

// Report 生成报告
func (t *TokenTracker) Report() string {
	var sb strings.Builder
	sb.WriteString("Token Usage Report:\n")
	sb.WriteString(fmt.Sprintf("  System Prompt: %d (%.1f%%)\n",
		t.SystemTokens, t.percentage(t.SystemTokens)))
	sb.WriteString(fmt.Sprintf("  User Prompt: %d (%.1f%%)\n",
		t.UserTokens, t.percentage(t.UserTokens)))
	sb.WriteString(fmt.Sprintf("  History: %d (%.1f%%)\n",
		t.HistoryTokens, t.percentage(t.HistoryTokens)))
	sb.WriteString(fmt.Sprintf("  Tools: %d (%.1f%%)\n",
		t.ToolsTokens, t.percentage(t.ToolsTokens)))
	sb.WriteString(fmt.Sprintf("  Few-Shots: %d (%.1f%%)\n",
		t.FewShotsTokens, t.percentage(t.FewShotsTokens)))
	sb.WriteString(fmt.Sprintf("  Total: %d\n", t.TotalTokens))
	return sb.String()
}

func (t *TokenTracker) percentage(tokens int) float64 {
	if t.TotalTokens == 0 {
		return 0
	}
	return float64(tokens) / float64(t.TotalTokens) * 100
}

// SimpleLogger 简单日志实现
type SimpleLogger struct {
	debug bool
}

// NewSimpleLogger 创建简单日志器
func NewSimpleLogger(debug bool) *SimpleLogger {
	return &SimpleLogger{debug: debug}
}

func (l *SimpleLogger) Info(msg string, args ...any) {
	fmt.Printf("[INFO] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Debug(msg string, args ...any) {
	if !l.debug {
		return
	}
	fmt.Printf("[DEBUG] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) IsDebug() bool {
	return l.debug
}

// simpleCounter 简单计数器（用于默认情况）
type simpleCounter struct{}

func (c *simpleCounter) Count(text string) int {
	return len(text) / 4
}
