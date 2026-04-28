package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ToolExecutionResult struct {
	Result      string
	Duration    time.Duration
	Error       error
	ToolName    string
	Interaction *InteractionRequest // non-nil when tool requests human interaction
}

type ToolExecutor interface {
	Execute(ctx context.Context, name string, params map[string]any) (*ToolExecutionResult, error)
	ResetCycle()
}

type toolExecutorConfig struct {
	registry          ToolRegistry
	permissionChecker ToolPermissionChecker
	preHooks          []Hook
	postHooks         []Hook
	resultLimits      ToolResultLimits
	eventEmitter      func(ReactEvent)
	maxPersistChars   int
	persistDir        string
}

type ExecutorOption func(*toolExecutorConfig)

func WithPermissionChecker(checker ToolPermissionChecker) ExecutorOption {
	return func(c *toolExecutorConfig) { c.permissionChecker = checker }
}

func WithPreHooks(hooks ...Hook) ExecutorOption {
	return func(c *toolExecutorConfig) { c.preHooks = hooks }
}

func WithPostHooks(hooks ...Hook) ExecutorOption {
	return func(c *toolExecutorConfig) { c.postHooks = hooks }
}

func WithResultLimits(limits ToolResultLimits) ExecutorOption {
	return func(c *toolExecutorConfig) { c.resultLimits = limits }
}

func WithEventEmitter(emitter func(ReactEvent)) ExecutorOption {
	return func(c *toolExecutorConfig) { c.eventEmitter = emitter }
}

func WithMaxPersistChars(n int) ExecutorOption {
	return func(c *toolExecutorConfig) { c.maxPersistChars = n }
}

func WithPersistDir(dir string) ExecutorOption {
	return func(c *toolExecutorConfig) { c.persistDir = dir }
}

func NewToolExecutor(registry ToolRegistry, opts ...ExecutorOption) ToolExecutor {
	cfg := &toolExecutorConfig{registry: registry}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.maxPersistChars == 0 {
		cfg.maxPersistChars = DefaultToolResultLimits().MaxResultSizeChars
	}
	if cfg.persistDir == "" {
		cfg.persistDir = defaultPersistDir()
	}
	return &defaultToolExecutor{
		cfg:       cfg,
		charsUsed: 0,
	}
}

type defaultToolExecutor struct {
	cfg       *toolExecutorConfig
	charsUsed int
	mu        sync.RWMutex
}

func (e *defaultToolExecutor) ResetCycle() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.charsUsed = 0
}

func (e *defaultToolExecutor) Execute(ctx context.Context, name string, params map[string]any) (*ToolExecutionResult, error) {
	tool, ok := e.cfg.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}

	toolInfo := tool.Info()

	useCtx := &ToolUseContext{
		ToolName: name,
		ToolInfo: toolInfo,
		Params:   params,
		Ctx:      ctx,
	}

	if e.cfg.permissionChecker != nil {
		permResult := e.cfg.permissionChecker.CheckPermissions(useCtx)
		switch permResult.Behavior {
		case PermissionDeny:
			if e.cfg.eventEmitter != nil {
				e.cfg.eventEmitter(NewReactEvent(useCtx.SessionID, useCtx.TaskID, "", PermissionDenied, permResult.Message))
			}
			return &ToolExecutionResult{ToolName: name, Error: fmt.Errorf("tool %q denied: %s", name, permResult.Message)}, nil
		case PermissionAsk:
			if e.cfg.eventEmitter != nil {
				e.cfg.eventEmitter(NewReactEvent(useCtx.SessionID, useCtx.TaskID, "", PermissionRequest, PermissionRequestData{
					ToolName:      name,
					Params:        params,
					Reason:        permResult.Message,
					SecurityLevel: toolInfo.SecurityLevel,
				}))
			}
			if responder, ok := e.cfg.permissionChecker.(PermissionResponder); ok {
				finalResult := responder.BlockAndWait(useCtx)
				switch finalResult.Behavior {
				case PermissionDeny:
					if e.cfg.eventEmitter != nil {
						e.cfg.eventEmitter(NewReactEvent(useCtx.SessionID, useCtx.TaskID, "", PermissionDenied, finalResult.Message))
					}
					return &ToolExecutionResult{ToolName: name, Error: fmt.Errorf("tool %q denied by user: %s", name, finalResult.Message)}, nil
				case PermissionAllow:
					if finalResult.UpdatedInput != nil {
						params = finalResult.UpdatedInput
						useCtx.Params = params
					}
				}
			}
		case PermissionAllow:
			if permResult.UpdatedInput != nil {
				params = permResult.UpdatedInput
				useCtx.Params = params
			}
		}
	}

	for _, hook := range e.cfg.preHooks {
		hr := hook.Execute(useCtx)
		if hr.PreventContinuation {
			return &ToolExecutionResult{ToolName: name, Error: fmt.Errorf("tool %q blocked by pre-tool-use hook: %s", name, hr.Message)}, nil
		}
		if hr.PermissionResult != nil {
			if hr.PermissionResult.Behavior == PermissionDeny {
				return &ToolExecutionResult{ToolName: name, Error: fmt.Errorf("tool %q denied by hook: %s", name, hr.PermissionResult.Message)}, nil
			}
			if hr.PermissionResult.UpdatedInput != nil {
				params = hr.PermissionResult.UpdatedInput
				useCtx.Params = params
			}
		}
		if hr.UpdatedInput != nil {
			params = hr.UpdatedInput
			useCtx.Params = params
		}
	}

	start := time.Now()
	result, err := tool.Execute(ctx, params)
	duration := time.Since(start)

	if len(e.cfg.postHooks) > 0 {
		var resultStr string
		if err == nil {
			resultStr, _ = result.(string)
			if resultStr == "" {
				b, _ := json.Marshal(result)
				resultStr = string(b)
			}
		}
		postCtx := &PostToolUseContext{
			ToolUseContext: useCtx,
			Result:         resultStr,
			Err:            err,
			Duration:       duration.Milliseconds(),
		}
		for _, hook := range e.cfg.postHooks {
			hook.Execute(postCtx)
		}
	}

	if err != nil {
		return &ToolExecutionResult{ToolName: name, Duration: duration, Error: err}, nil
	}

	var interaction *InteractionRequest
	if m, ok := result.(map[string]any); ok {
		if ir, exists := m["_interaction"]; exists {
			if req, ok := ir.(*InteractionRequest); ok && req != nil {
				interaction = req
			}
		}
	}

	str, ok := result.(string)
	if !ok {
		b, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			str = fmt.Sprintf("%v", result)
		} else {
			str = string(b)
		}
	}

	str = e.processResult(name, str, toolInfo)

	return &ToolExecutionResult{
		Result:      str,
		Duration:    duration,
		ToolName:    name,
		Interaction: interaction,
	}, nil
}

func (e *defaultToolExecutor) processResult(toolName, str string, toolInfo *ToolInfo) string {
	if toolInfo.MaxResultSizeChars == -1 {
		return str
	}

	e.mu.Lock()
	e.charsUsed += len([]rune(str))
	currentChars := e.charsUsed
	e.mu.Unlock()

	limits := DefaultToolResultLimits()
	if e.cfg.resultLimits.MaxResultSizeChars > 0 {
		limits = e.cfg.resultLimits
	}

	if currentChars > limits.MaxToolResultsPerMessageChars {
		targetChars := limits.MaxResultSizeChars
		if targetChars <= 0 {
			targetChars = 5000
		}
		runes := []rune(str)
		if len(runes) > targetChars {
			var buf strings.Builder
			buf.Grow(targetChars + 200)
			buf.WriteString(string(runes[:targetChars]))
			buf.WriteString("\n... [context budget exceeded: total tool output in this cycle is ")
			buf.WriteString(fmt.Sprintf("%d", currentChars))
			buf.WriteString(" chars, limit is ")
			buf.WriteString(fmt.Sprintf("%d", limits.MaxToolResultsPerMessageChars))
			buf.WriteString("] ...")
			str = buf.String()
		}
		return str
	}

	charCount := len([]rune(str))
	if charCount <= e.cfg.maxPersistChars {
		return str
	}

	persisted := PersistToDisk(toolName, str, e.cfg.persistDir, e.cfg.maxPersistChars, e.cfg.maxPersistChars/2)
	if persisted != nil {
		str = PersistedResultTag(persisted)
	}

	return str
}
