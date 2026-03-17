package wrapper

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/tools"
)

// Wrapper 包装器接口
type Wrapper interface {
	Wrap(tool tools.Tool) tools.Tool
}

// TimeoutWrapper 超时包装器
type TimeoutWrapper struct {
	timeout time.Duration
}

// WithTimeout 创建超时包装器
func WithTimeout(timeout time.Duration) *TimeoutWrapper {
	return &TimeoutWrapper{timeout: timeout}
}

func (w *TimeoutWrapper) Wrap(t tools.Tool) tools.Tool {
	return &timeoutTool{
		base:    t,
		timeout: w.timeout,
	}
}

type timeoutTool struct {
	base    tools.Tool
	timeout time.Duration
}

func (t *timeoutTool) Name() string {
	return t.base.Name()
}

func (t *timeoutTool) Description() string {
	return t.base.Description()
}

func (t *timeoutTool) Execute(params map[string]any) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	resultCh := make(chan result, 1)
	go func() {
		output, err := t.base.Execute(params)
		resultCh <- result{output: output, err: err}
	}()

	select {
	case res := <-resultCh:
		return res.output, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("tool execution timeout after %v", t.timeout)
	}
}

type result struct {
	output any
	err    error
}

// RetryWrapper 重试包装器
type RetryWrapper struct {
	maxAttempts int
	interval    time.Duration
	retryIf     func(error) bool
}

// WithRetry 创建重试包装器
func WithRetry(maxAttempts int, interval time.Duration) *RetryWrapper {
	return &RetryWrapper{
		maxAttempts: maxAttempts,
		interval:    interval,
		retryIf:     func(err error) bool { return err != nil }, // 默认所有错误都重试
	}
}

// WithRetryIf 设置重试条件
func (w *RetryWrapper) WithRetryIf(fn func(error) bool) *RetryWrapper {
	w.retryIf = fn
	return w
}

func (w *RetryWrapper) Wrap(t tools.Tool) tools.Tool {
	return &retryTool{
		base:        t,
		maxAttempts: w.maxAttempts,
		interval:    w.interval,
		retryIf:     w.retryIf,
	}
}

type retryTool struct {
	base        tools.Tool
	maxAttempts int
	interval    time.Duration
	retryIf     func(error) bool
}

func (t *retryTool) Name() string {
	return t.base.Name()
}

func (t *retryTool) Description() string {
	return t.base.Description()
}

func (t *retryTool) Execute(params map[string]any) (any, error) {
	var lastErr error

	for attempt := 1; attempt <= t.maxAttempts; attempt++ {
		output, err := t.base.Execute(params)

		if err == nil {
			return output, nil
		}

		lastErr = err

		// 检查是否应该重试
		if !t.retryIf(err) {
			return nil, err
		}

		// 最后一次尝试不需要等待
		if attempt < t.maxAttempts {
			time.Sleep(t.interval * time.Duration(attempt))
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", t.maxAttempts, lastErr)
}

// Wrap 组合多个包装器
func Wrap(t tools.Tool, wrappers ...Wrapper) tools.Tool {
	result := t
	for _, w := range wrappers {
		result = w.Wrap(result)
	}
	return result
}
