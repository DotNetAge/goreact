package stagnation

import (
	"sync"

	"github.com/ray/goreact/pkg/types"
)

// StagnationResult 停滞检测结果
type StagnationResult struct {
	IsStagnant bool
	Type       string // "no_action", "repeated_failure", "no_progress"
	Suggestion string
}

// Detector 停滞检测器
type Detector struct {
	mu                   sync.Mutex
	noProgressLimit      int
	repeatedFailureLimit int
	noActionCount        int
	lastFailedAction     *types.Action
	failureCount         int
}

// DetectorOption 配置选项
type DetectorOption func(*Detector)

// WithNoProgressLimit 设置无进展限制
func WithNoProgressLimit(limit int) DetectorOption {
	return func(d *Detector) {
		d.noProgressLimit = limit
	}
}

// WithRepeatedFailureLimit 设置重复失败限制
func WithRepeatedFailureLimit(limit int) DetectorOption {
	return func(d *Detector) {
		d.repeatedFailureLimit = limit
	}
}

// NewDetector 创建停滞检测器
func NewDetector(opts ...DetectorOption) *Detector {
	d := &Detector{
		noProgressLimit:      3,
		repeatedFailureLimit: 2,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Check 检查是否停滞
func (d *Detector) Check(state *types.LoopState) StagnationResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查无行动
	if state.LastThought != nil && state.LastThought.Action == nil {
		d.noActionCount++
		d.failureCount = 0
		d.lastFailedAction = nil

		if d.noActionCount >= d.noProgressLimit {
			return StagnationResult{
				IsStagnant: true,
				Type:       "no_action",
				Suggestion: "The agent has been thinking without taking action. Consider forcing an action or providing more specific guidance.",
			}
		}
		return StagnationResult{}
	}

	// 有行动，重置无行动计数
	d.noActionCount = 0

	// 检查重复失败
	if state.LastResult != nil && !state.LastResult.Success {
		currentAction := state.LastThought.Action

		// 检查是否与上次失败的操作相同
		if d.lastFailedAction != nil && d.isSameAction(d.lastFailedAction, currentAction) {
			d.failureCount++

			if d.failureCount >= d.repeatedFailureLimit {
				return StagnationResult{
					IsStagnant: true,
					Type:       "repeated_failure",
					Suggestion: "The same action has failed multiple times. Try a different tool or different parameters.",
				}
			}
		} else {
			// 不同的失败操作，重置计数
			d.failureCount = 1
			d.lastFailedAction = currentAction
		}
	} else {
		// 成功了，重置失败计数
		d.failureCount = 0
		d.lastFailedAction = nil
	}

	return StagnationResult{}
}

// Reset 重置检测器
func (d *Detector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.noActionCount = 0
	d.failureCount = 0
	d.lastFailedAction = nil
}

// isSameAction 检查两个 Action 是否相同
func (d *Detector) isSameAction(a1, a2 *types.Action) bool {
	if a1 == nil || a2 == nil {
		return false
	}

	if a1.ToolName != a2.ToolName {
		return false
	}

	// 简单比较：只比较工具名称
	return true
}
