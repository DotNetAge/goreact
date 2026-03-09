package detector

import (
	"fmt"
	"strings"
	"sync"
)

// LoopPattern 循环模式
type LoopPattern struct {
	Detected    bool
	Type        string // "repeated_failure", "no_action"
	RepeatCount int
	Suggestion  string
}

// ActionRecord 操作记录
type ActionRecord struct {
	ToolName   string
	Parameters map[string]any
	Success    bool
	IsNoAction bool
}

// LoopDetector 循环检测器
type LoopDetector struct {
	mu         sync.Mutex
	history    []ActionRecord
	maxRepeats int
	windowSize int
}

// DetectorOption 配置选项
type DetectorOption func(*LoopDetector)

// WithMaxRepeats 设置最大重复次数
func WithMaxRepeats(n int) DetectorOption {
	return func(d *LoopDetector) { d.maxRepeats = n }
}

// WithWindowSize 设置检测窗口大小
func WithWindowSize(n int) DetectorOption {
	return func(d *LoopDetector) { d.windowSize = n }
}

// NewLoopDetector 创建循环检测器
func NewLoopDetector(opts ...DetectorOption) *LoopDetector {
	d := &LoopDetector{
		maxRepeats: 3,
		windowSize: 10,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Record 记录一次操作并检测循环
func (d *LoopDetector) Record(toolName string, params map[string]any, success bool) LoopPattern {
	d.mu.Lock()
	defer d.mu.Unlock()

	record := ActionRecord{
		ToolName:   toolName,
		Parameters: params,
		Success:    success,
	}

	d.history = append(d.history, record)
	d.trimHistory()

	// 成功的操作不检测循环
	if success {
		return LoopPattern{}
	}

	return d.detectRepeatedFailure(record)
}

// RecordNoAction 记录一次无行动
func (d *LoopDetector) RecordNoAction() LoopPattern {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.history = append(d.history, ActionRecord{IsNoAction: true})
	d.trimHistory()

	return d.detectNoAction()
}

// Clear 清空历史
func (d *LoopDetector) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.history = nil
}

// detectRepeatedFailure 检测重复失败
func (d *LoopDetector) detectRepeatedFailure(current ActionRecord) LoopPattern {
	count := 0
	for i := len(d.history) - 1; i >= 0; i-- {
		r := d.history[i]
		if r.IsNoAction {
			continue
		}
		if r.ToolName == current.ToolName && !r.Success && d.paramsEqual(r.Parameters, current.Parameters) {
			count++
		} else {
			break
		}
	}

	if count >= d.maxRepeats {
		return LoopPattern{
			Detected:    true,
			Type:        "repeated_failure",
			RepeatCount: count,
			Suggestion: fmt.Sprintf(
				"You've tried '%s' with the same parameters %d times, all failed. "+
					"STOP retrying and try a different approach.",
				current.ToolName, count),
		}
	}

	return LoopPattern{}
}

// detectNoAction 检测连续无行动
func (d *LoopDetector) detectNoAction() LoopPattern {
	count := 0
	for i := len(d.history) - 1; i >= 0; i-- {
		if d.history[i].IsNoAction {
			count++
		} else {
			break
		}
	}

	if count >= d.maxRepeats {
		return LoopPattern{
			Detected:    true,
			Type:        "no_action",
			RepeatCount: count,
			Suggestion: fmt.Sprintf(
				"You've been thinking for %d iterations without taking action. "+
					"Please use a tool to make progress.",
				count),
		}
	}

	return LoopPattern{}
}

// trimHistory 修剪历史到窗口大小
func (d *LoopDetector) trimHistory() {
	if len(d.history) > d.windowSize {
		d.history = d.history[len(d.history)-d.windowSize:]
	}
}

// paramsEqual 比较参数是否相同
func (d *LoopDetector) paramsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", bv) {
			return false
		}
	}
	return true
}

// FormatPattern 格式化循环模式为反馈消息
func FormatPattern(pattern LoopPattern) string {
	if !pattern.Detected {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "🔄 Loop detected (%s): ", pattern.Type)
	sb.WriteString(pattern.Suggestion)
	return sb.String()
}
