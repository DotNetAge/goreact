package presets

import (
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/actor/debug"
	"github.com/ray/goreact/pkg/actor/resultfmt"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/tool"
	"github.com/ray/goreact/pkg/types"
)

// Option 配置选项
type Option func(*config)

type config struct {
	timeout      time.Duration
	maxRetry     int
	retryDelay   time.Duration
	allowedTools map[string]bool
	maxOutput    int
}

func defaultConfig() *config {
	return &config{
		timeout:    10 * time.Second,
		maxRetry:   3,
		retryDelay: 500 * time.Millisecond,
		maxOutput:  2000,
	}
}

// WithTimeout 设置超时
func WithTimeout(timeout time.Duration) Option {
	return func(c *config) { c.timeout = timeout }
}

// WithRetry 设置重试
func WithRetry(maxRetry int, delay time.Duration) Option {
	return func(c *config) {
		c.maxRetry = maxRetry
		c.retryDelay = delay
	}
}

// WithAllowedTools 设置工具白名单
func WithAllowedTools(tools ...string) Option {
	return func(c *config) {
		c.allowedTools = make(map[string]bool)
		for _, t := range tools {
			c.allowedTools[t] = true
		}
	}
}

// WithMaxOutput 设置最大输出长度
func WithMaxOutput(length int) Option {
	return func(c *config) { c.maxOutput = length }
}

// ============================================================
// ResilientActor - 弹性模式
// 自带：超时 + 重试 + 结果格式化
// ============================================================

// ResilientActor 弹性 Actor
type ResilientActor struct {
	toolManager *tool.Manager
	cfg         *config
	formatter   *resultfmt.Formatter
}

// NewResilientActor 创建弹性 Actor
func NewResilientActor(tm *tool.Manager, opts ...Option) *ResilientActor {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &ResilientActor{
		toolManager: tm,
		cfg:         cfg,
		formatter:   resultfmt.New(resultfmt.WithMaxLength(cfg.maxOutput)),
	}
}

func (a *ResilientActor) Act(action *types.Action, ctx *core.Context) (*types.ExecutionResult, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}

	var output any
	var lastErr error

	for attempt := 1; attempt <= a.cfg.maxRetry; attempt++ {
		// 带超时执行
		done := make(chan struct{})
		go func() {
			output, lastErr = a.toolManager.ExecuteTool(action.ToolName, action.Parameters)
			close(done)
		}()

		select {
		case <-done:
			if lastErr == nil {
				return a.buildResult(action, output, nil), nil
			}
			// 最后一次不等待
			if attempt < a.cfg.maxRetry {
				time.Sleep(a.cfg.retryDelay)
			}
		case <-time.After(a.cfg.timeout):
			lastErr = fmt.Errorf("timeout after %v", a.cfg.timeout)
			if attempt < a.cfg.maxRetry {
				time.Sleep(a.cfg.retryDelay)
			}
		}
	}

	return a.buildResult(action, nil, lastErr), nil
}

func (a *ResilientActor) buildResult(action *types.Action, output any, err error) *types.ExecutionResult {
	result := &types.ExecutionResult{
		Success:  err == nil,
		Output:   output,
		Metadata: make(map[string]any),
	}

	result.Metadata["tool_name"] = action.ToolName
	result.Metadata["parameters"] = action.Parameters

	if err != nil {
		result.Error = err
	}

	// 格式化输出
	if output != nil {
		result.Output = a.formatter.Format(output)
	}

	return result
}

// ============================================================
// DebugActor - 调试模式
// 自带：完整追踪 + 性能分析
// ============================================================

// DebugActor 调试 Actor
type DebugActor struct {
	toolManager *tool.Manager
	tracer      *debug.ExecutionTracer
	profiler    *debug.PerformanceProfiler
}

// NewDebugActor 创建调试 Actor
func NewDebugActor(tm *tool.Manager, tracer *debug.ExecutionTracer, profiler *debug.PerformanceProfiler) *DebugActor {
	return &DebugActor{
		toolManager: tm,
		tracer:      tracer,
		profiler:    profiler,
	}
}

func (a *DebugActor) Act(action *types.Action, ctx *core.Context) (*types.ExecutionResult, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}

	start := time.Now()
	output, execErr := a.toolManager.ExecuteTool(action.ToolName, action.Parameters)
	duration := time.Since(start)

	// 记录追踪
	a.tracer.Record(debug.ExecutionRecord{
		ToolName:   action.ToolName,
		Parameters: action.Parameters,
		Output:     output,
		Error:      execErr,
		Duration:   duration,
		Timestamp:  start,
	})

	// 记录性能
	a.profiler.Record(action.ToolName, duration, execErr == nil)

	result := &types.ExecutionResult{
		Success:  execErr == nil,
		Output:   output,
		Metadata: make(map[string]any),
	}

	result.Metadata["tool_name"] = action.ToolName
	result.Metadata["parameters"] = action.Parameters
	result.Metadata["duration"] = duration.String()

	if execErr != nil {
		result.Error = execErr
	}

	return result, nil
}

// ============================================================
// SafeActor - 安全模式
// 自带：工具白名单 + 结果格式化
// ============================================================

// SafeActor 安全 Actor
type SafeActor struct {
	toolManager *tool.Manager
	cfg         *config
	formatter   *resultfmt.Formatter
}

// NewSafeActor 创建安全 Actor
func NewSafeActor(tm *tool.Manager, opts ...Option) *SafeActor {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &SafeActor{
		toolManager: tm,
		cfg:         cfg,
		formatter:   resultfmt.New(resultfmt.WithMaxLength(cfg.maxOutput)),
	}
}

func (a *SafeActor) Act(action *types.Action, ctx *core.Context) (*types.ExecutionResult, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}

	// 检查工具白名单
	if a.cfg.allowedTools != nil && !a.cfg.allowedTools[action.ToolName] {
		return &types.ExecutionResult{
			Success:  false,
			Metadata: map[string]any{"tool_name": action.ToolName},
			Error: fmt.Errorf("tool '%s' is not allowed. Allowed tools: %v",
				action.ToolName, a.allowedToolNames()),
		}, nil
	}

	output, execErr := a.toolManager.ExecuteTool(action.ToolName, action.Parameters)

	result := &types.ExecutionResult{
		Success:  execErr == nil,
		Output:   output,
		Metadata: make(map[string]any),
	}

	result.Metadata["tool_name"] = action.ToolName

	if execErr != nil {
		result.Error = execErr
	}

	if output != nil {
		result.Output = a.formatter.Format(output)
	}

	return result, nil
}

func (a *SafeActor) allowedToolNames() []string {
	names := make([]string, 0, len(a.cfg.allowedTools))
	for name := range a.cfg.allowedTools {
		names = append(names, name)
	}
	return names
}

// ============================================================
// ProductionActor - 生产模式（全部最佳实践）
// 自带：超时 + 重试 + 结果格式化 + 追踪 + 性能分析
// ============================================================

// ProductionActor 生产 Actor
type ProductionActor struct {
	resilient *ResilientActor
	tracer    *debug.ExecutionTracer
	profiler  *debug.PerformanceProfiler
}

// NewProductionActor 创建生产 Actor
func NewProductionActor(tm *tool.Manager, opts ...Option) *ProductionActor {
	return &ProductionActor{
		resilient: NewResilientActor(tm, opts...),
		tracer:    debug.NewExecutionTracer(true),
		profiler:  debug.NewPerformanceProfiler(),
	}
}

func (a *ProductionActor) Act(action *types.Action, ctx *core.Context) (*types.ExecutionResult, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}

	start := time.Now()
	result, err := a.resilient.Act(action, ctx)
	duration := time.Since(start)

	if err != nil {
		return nil, err
	}

	// 追踪
	a.tracer.Record(debug.ExecutionRecord{
		ToolName:   action.ToolName,
		Parameters: action.Parameters,
		Output:     result.Output,
		Error:      result.Error,
		Duration:   duration,
		Timestamp:  start,
	})

	// 性能
	a.profiler.Record(action.ToolName, duration, result.Success)

	// 补充 metadata
	result.Metadata["duration"] = duration.String()

	return result, nil
}

// GetTracer 获取追踪器（用于调试）
func (a *ProductionActor) GetTracer() *debug.ExecutionTracer {
	return a.tracer
}

// GetProfiler 获取性能分析器
func (a *ProductionActor) GetProfiler() *debug.PerformanceProfiler {
	return a.profiler
}
