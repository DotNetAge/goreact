package presets

import (
	"fmt"
	"strings"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/observer/detector"
	"github.com/ray/goreact/pkg/observer/feedback"
	"github.com/ray/goreact/pkg/observer/validator"
	"github.com/ray/goreact/pkg/types"
)

// ============================================================
// SmartObserver - 智能反馈
// 自带：工具特定反馈 + 循环检测 + 历史更新
// ============================================================

// SmartObserver 智能 Observer
type SmartObserver struct {
	generator *feedback.SmartGenerator
	detector  *detector.LoopDetector
}

// NewSmartObserver 创建智能 Observer
func NewSmartObserver() *SmartObserver {
	return &SmartObserver{
		generator: feedback.NewSmartGenerator(),
		detector:  detector.NewLoopDetector(detector.WithMaxRepeats(3)),
	}
}

func (o *SmartObserver) Observe(result *types.ExecutionResult, ctx *core.Context) (*types.Feedback, error) {
	// 添加 nil 检查
	if result == nil {
		return nil, fmt.Errorf("execution result cannot be nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	fb := &types.Feedback{
		ShouldContinue: true,
		Metadata:       make(map[string]any),
	}

	// 生成智能反馈
	fb.Message = o.generator.Generate(result, ctx)

	// 循环检测
	toolName := getToolName(result)
	params := getParams(result)

	var pattern detector.LoopPattern
	if toolName != "" {
		pattern = o.detector.Record(toolName, params, result.Success)
	}

	if pattern.Detected {
		fb.Message += "\n\n" + detector.FormatPattern(pattern)
		fb.Metadata["loop_detected"] = true
		fb.Metadata["loop_type"] = pattern.Type
	}

	// 更新历史
	updateHistory(ctx, fb.Message)

	return fb, nil
}

// ============================================================
// StrictObserver - 严格验证
// 自带：智能反馈 + 结果验证 + 循环检测
// ============================================================

// StrictObserver 严格 Observer
type StrictObserver struct {
	generator *feedback.SmartGenerator
	validator *validator.Validator
	detector  *detector.LoopDetector
}

// NewStrictObserver 创建严格 Observer
func NewStrictObserver() *StrictObserver {
	return &StrictObserver{
		generator: feedback.NewSmartGenerator(),
		validator: validator.New(
			validator.WithHTTPStatusRule(),
			validator.WithErrorPatternRule(),
			validator.WithEmptyResultRule(),
		),
		detector: detector.NewLoopDetector(detector.WithMaxRepeats(2)),
	}
}

func (o *StrictObserver) Observe(result *types.ExecutionResult, ctx *core.Context) (*types.Feedback, error) {
	// 添加 nil 检查
	if result == nil {
		return nil, fmt.Errorf("execution result cannot be nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	fb := &types.Feedback{
		ShouldContinue: true,
		Metadata:       make(map[string]any),
	}

	// 生成智能反馈
	fb.Message = o.generator.Generate(result, ctx)

	// 严格验证
	vr := o.validator.Validate(result, ctx)
	if !vr.IsValid {
		fb.Message += fmt.Sprintf("\n\n⚠️ Validation issues: %v", vr.Issues)
		if len(vr.Suggestions) > 0 {
			fb.Message += fmt.Sprintf("\nSuggestions: %v", vr.Suggestions)
		}
		fb.Metadata["validation_failed"] = true
		fb.Metadata["issues"] = vr.Issues
	}

	// 循环检测
	toolName := getToolName(result)
	params := getParams(result)

	if toolName != "" {
		pattern := o.detector.Record(toolName, params, result.Success)
		if pattern.Detected {
			fb.Message += "\n\n" + detector.FormatPattern(pattern)
			fb.Metadata["loop_detected"] = true
		}
	}

	updateHistory(ctx, fb.Message)

	return fb, nil
}

// ============================================================
// VerboseObserver - 详细日志
// 自带：智能反馈 + 完整日志
// ============================================================

// Logger 日志接口
type Logger interface {
	Info(msg string, args ...any)
}

// VerboseObserver 详细 Observer
type VerboseObserver struct {
	generator *feedback.SmartGenerator
	detector  *detector.LoopDetector
	logger    Logger
}

// NewVerboseObserver 创建详细 Observer
func NewVerboseObserver(logger Logger) *VerboseObserver {
	return &VerboseObserver{
		generator: feedback.NewSmartGenerator(),
		detector:  detector.NewLoopDetector(detector.WithMaxRepeats(3)),
		logger:    logger,
	}
}

func (o *VerboseObserver) Observe(result *types.ExecutionResult, ctx *core.Context) (*types.Feedback, error) {
	// 添加 nil 检查
	if result == nil {
		return nil, fmt.Errorf("execution result cannot be nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	toolName := getToolName(result)

	// 详细日志
	if o.logger != nil {
		o.logger.Info("Observe",
			"tool", toolName,
			"success", result.Success,
			"output", truncate(fmt.Sprintf("%v", result.Output), 100),
		)
	}

	fb := &types.Feedback{
		ShouldContinue: true,
		Metadata:       make(map[string]any),
	}

	fb.Message = o.generator.Generate(result, ctx)

	// 循环检测
	params := getParams(result)
	if toolName != "" {
		pattern := o.detector.Record(toolName, params, result.Success)
		if pattern.Detected {
			fb.Message += "\n\n" + detector.FormatPattern(pattern)
			fb.Metadata["loop_detected"] = true
			if o.logger != nil {
				o.logger.Info("Loop detected", "type", pattern.Type, "count", pattern.RepeatCount)
			}
		}
	}

	updateHistory(ctx, fb.Message)

	return fb, nil
}

// ============================================================
// ProductionObserver - 生产模式
// 自带：智能反馈 + 结果验证 + 循环检测
// ============================================================

// NewProductionObserver 创建生产模式 Observer
func NewProductionObserver() *StrictObserver {
	return NewStrictObserver()
}

// ============================================================
// 辅助函数
// ============================================================

func getToolName(result *types.ExecutionResult) string {
	if result.Metadata != nil {
		if name, ok := result.Metadata["tool_name"].(string); ok {
			return name
		}
	}
	return ""
}

func getParams(result *types.ExecutionResult) map[string]any {
	if result.Metadata != nil {
		if params, ok := result.Metadata["parameters"].(map[string]any); ok {
			return params
		}
	}
	return nil
}

func updateHistory(ctx *core.Context, message string) {
	const maxHistorySize = 10000 // 最大历史字符数

	history := ""
	if h, ok := ctx.Get("history"); ok {
		if historyStr, ok := h.(string); ok {
			history = historyStr
		}
	}

	history += message + "\n"

	// 如果历史超过限制，保留最新的部分
	if len(history) > maxHistorySize {
		history = history[len(history)-maxHistorySize:]
		// 找到第一个换行符，从完整行开始
		if idx := strings.Index(history, "\n"); idx != -1 {
			history = history[idx+1:]
		}
	}

	ctx.Set("history", history)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
