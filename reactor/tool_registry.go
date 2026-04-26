package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// DefaultToolRegistry manages available core.FuncTool instances. It is safe for concurrent use.
// This registry wraps core.FuncTool (which returns (any, error)) and normalizes the
// result to (string, error) for the reactor's consumption.
//
// The registry supports a multi-layer permission pipeline:
//  1. SecurityPolicy (legacy) — simple allow/deny
//  2. PreToolUse hooks — can block, modify input, or make permission decisions
//  3. ToolPermissionChecker — full authorization with allow/deny/ask semantics
//  4. PostToolUse hooks — observe/modify results after execution
type DefaultToolRegistry struct {
	mu               sync.RWMutex
	tools            map[string]core.FuncTool
	securityPolicy   core.SecurityPolicy
	resultStorage    core.ToolResultStorage // nil = no persistence (inline only)
	resultLimits     core.ToolResultLimits
	messageCharsUsed int // tracks total chars used in current message cycle

	// Permission pipeline (new)
	permissionChecker core.ToolPermissionChecker
	hooks             map[core.HookEventType][]core.Hook // keyed by event type
	eventEmitter      func(core.ReactEvent)              // for emitting PermissionRequest events

	// Memory for semantic tool search (reflexive memory fallback).
	// When set, SemanticSearch uses Memory.Retrieve to find tools by intent semantics.
	memory core.Memory
}

// ToolRegistry is an alias for DefaultToolRegistry for backward compatibility.
type ToolRegistry = DefaultToolRegistry

// NewDefaultToolRegistry creates an empty tool registry.
func NewDefaultToolRegistry() *DefaultToolRegistry {
	return &DefaultToolRegistry{
		tools:        make(map[string]core.FuncTool),
		resultLimits: core.DefaultToolResultLimits(),
		hooks:        make(map[core.HookEventType][]core.Hook),
	}
}

// Deprecated: Use NewDefaultToolRegistry instead.
func NewToolRegistry() *DefaultToolRegistry {
	return NewDefaultToolRegistry()
}

// Compile-time interface check
var _ core.ToolRegistryInterface = (*DefaultToolRegistry)(nil)

// SetResultStorage sets the tool result persistence storage.
// When set, tool results exceeding the size threshold will be persisted to disk
// and only a preview will be returned inline. This is the second layer of defense
// against context explosion.
func (r *DefaultToolRegistry) SetResultStorage(storage core.ToolResultStorage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resultStorage = storage
}

// SetResultLimits configures per-result and per-message total size limits.
func (r *DefaultToolRegistry) SetResultLimits(limits core.ToolResultLimits) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resultLimits = limits
}

// ResetMessageCharCounter resets the per-message character counter.
// Should be called at the start of each T-A-O cycle.
func (r *DefaultToolRegistry) ResetMessageCharCounter() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageCharsUsed = 0
}

// Register adds a core.FuncTool. Returns error if a tool with the same name exists.
func (r *DefaultToolRegistry) Register(tool core.FuncTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := tool.Info().Name
	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// Get returns a tool by name.
func (r *DefaultToolRegistry) Get(name string) (core.FuncTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools.
func (r *DefaultToolRegistry) All() []core.FuncTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]core.FuncTool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// SetMemory injects a Memory instance for semantic tool search (reflexive memory).
// When set, GetWithSemantic will fallback to Memory.Retrieve for intent-based
// tool discovery if exact name matching fails.
func (r *DefaultToolRegistry) SetMemory(mem core.Memory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memory = mem
}

// GetWithSemantic first tries exact name matching, then falls back to
// semantic search via Memory if configured. This is the reflexive memory
// integration: tools indexed in Memory can be found by intent semantics.
//
// Returns the matched tool and true on success, nil and false otherwise.
func (r *DefaultToolRegistry) GetWithSemantic(ctx context.Context, name string, intent string) (core.FuncTool, bool) {
	// Phase 1: exact name match
	if tool, ok := r.Get(name); ok {
		return tool, true
	}

	// Phase 2: semantic search via Memory (reflexive memory fallback)
	if r.memory == nil || intent == "" {
		return nil, false
	}

	r.mu.RLock()
	mem := r.memory
	r.mu.RUnlock()

	records, err := mem.Retrieve(ctx, intent,
		core.WithMemoryTypes(core.MemoryTypeReflexive),
		core.WithMemoryLimit(5),
	)
	if err != nil || len(records) == 0 {
		return nil, false
	}

	// Phase 3: use returned tool names for exact lookup in map
	for _, rec := range records {
		if tool, ok := r.Get(rec.Title); ok {
			return tool, true
		}
	}

	return nil, false
}

// ToToolInfos extracts core.ToolInfo from all registered tools for prompt building.
func (r *DefaultToolRegistry) ToToolInfos() []core.ToolInfo {
	tools := r.All()
	infos := make([]core.ToolInfo, len(tools))
	for i, t := range tools {
		infos[i] = *t.Info()
	}
	return infos
}

// SetSecurityPolicy sets a custom security policy for tool execution.
// The policy is invoked before every tool execution; return false to block.
//
// Deprecated: Use SetPermissionChecker for the full permission pipeline.
func (r *DefaultToolRegistry) SetSecurityPolicy(policy core.SecurityPolicy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.securityPolicy = policy
}

// SetPermissionChecker sets the permission checker for fine-grained authorization.
// When set, it is consulted after PreToolUse hooks and before tool execution.
// The checker can return PermissionAsk to block execution until the user responds.
func (r *DefaultToolRegistry) SetPermissionChecker(checker core.ToolPermissionChecker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.permissionChecker = checker
}

// AddHook registers a lifecycle hook. Hooks are executed in registration order.
// Multiple hooks for the same event type are supported and run sequentially.
func (r *DefaultToolRegistry) AddHook(hook core.Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	eventType := hook.EventType()
	r.hooks[eventType] = append(r.hooks[eventType], hook)
}

// SetEventEmitter sets a callback for emitting ReactEvents (e.g., PermissionRequest).
// Called by the reactor during initialization.
func (r *DefaultToolRegistry) SetEventEmitter(fn func(core.ReactEvent)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventEmitter = fn
}

// ExecuteTool runs a tool by name with the given parameters.
// It normalizes the core.FuncTool result (any) to a string via JSON marshaling.
// This method implements the second layer of context defense: tool result persistence.
//
// Permission pipeline (executed in order):
//  1. Legacy SecurityPolicy check (allow/deny)
//  2. PreToolUse hooks (can block, modify input, or decide permission)
//  3. ToolPermissionChecker (allow/deny/ask)
//  4. If ask: emit PermissionRequest, block until user responds via PermissionResponder
//  5. Execute tool with (possibly modified) params
//  6. PostToolUse hooks
func (r *DefaultToolRegistry) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, time.Duration, error) {
	tool, ok := r.Get(name)
	if !ok {
		return "", 0, fmt.Errorf("tool %q not found", name)
	}

	r.mu.RLock()
	policy := r.securityPolicy
	storage := r.resultStorage
	limits := r.resultLimits
	checker := r.permissionChecker
	preHooks := r.hooks[core.HookPreToolUse]
	postHooks := r.hooks[core.HookPostToolUse]
	emitter := r.eventEmitter
	r.mu.RUnlock()

	toolInfo := tool.Info()

	// === Layer 1: Legacy SecurityPolicy ===
	if policy != nil && !policy(name, toolInfo.SecurityLevel) {
		return "", 0, fmt.Errorf("tool %q (security level: %v) blocked by security policy", name, toolInfo.SecurityLevel)
	}

	// === Layer 2: PreToolUse Hooks ===
	useCtx := &core.ToolUseContext{
		ToolName: name,
		ToolInfo: toolInfo,
		Params:   params,
		Ctx:      ctx,
	}

	for _, hook := range preHooks {
		hr := hook.Execute(useCtx)
		if hr.PreventContinuation {
			return "", 0, fmt.Errorf("tool %q blocked by pre-tool-use hook: %s", name, hr.Message)
		}
		if hr.PermissionResult != nil {
			if hr.PermissionResult.Behavior == core.PermissionDeny {
				return "", 0, fmt.Errorf("tool %q denied by hook: %s", name, hr.PermissionResult.Message)
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

	// === Layer 3: Permission Checker ===
	if checker != nil {
		permResult := checker.CheckPermissions(useCtx)

		switch permResult.Behavior {
		case core.PermissionDeny:
			if emitter != nil {
				emitter(core.NewReactEvent(useCtx.SessionID, useCtx.TaskID, "", core.PermissionDenied, permResult.Message))
			}
			return "", 0, fmt.Errorf("tool %q denied: %s", name, permResult.Message)

		case core.PermissionAsk:
			// Emit permission request event to client
			if emitter != nil {
				emitter(core.NewReactEvent(useCtx.SessionID, useCtx.TaskID, "", core.PermissionRequest, core.PermissionRequestData{
					ToolName:      name,
					Params:        params,
					Reason:        permResult.Message,
					SecurityLevel: toolInfo.SecurityLevel,
				}))
			}

			// Block until user responds via PermissionResponder.BlockAndWait()
			if responder, ok := checker.(core.PermissionResponder); ok {
				finalResult := responder.BlockAndWait(useCtx)
				switch finalResult.Behavior {
				case core.PermissionDeny:
					if emitter != nil {
						emitter(core.NewReactEvent(useCtx.SessionID, useCtx.TaskID, "", core.PermissionDenied, finalResult.Message))
					}
					return "", 0, fmt.Errorf("tool %q denied by user: %s", name, finalResult.Message)
				case core.PermissionAllow:
					if finalResult.UpdatedInput != nil {
						params = finalResult.UpdatedInput
						useCtx.Params = params
					}
				}
			}
			// If checker doesn't implement PermissionResponder, ask just blocks forever
			// (the caller should provide a proper implementation). In practice this
			// shouldn't happen — the standard AskPermission checker handles blocking.

		case core.PermissionAllow:
			if permResult.UpdatedInput != nil {
				params = permResult.UpdatedInput
				useCtx.Params = params
			}
		}
	}

	// === Execute the tool ===
	start := time.Now()
	result, err := tool.Execute(ctx, params)
	duration := time.Since(start)

	// === Layer 4: PostToolUse Hooks ===
	if len(postHooks) > 0 {
		var resultStr string
		if err == nil {
			resultStr, _ = result.(string)
			if resultStr == "" {
				b, _ := json.Marshal(result)
				resultStr = string(b)
			}
		}
		postCtx := &core.PostToolUseContext{
			ToolUseContext: useCtx,
			Result:         resultStr,
			Err:            err,
			Duration:       duration.Milliseconds(),
		}
		for _, hook := range postHooks {
			hook.Execute(postCtx)
		}
	}

	if err != nil {
		return "", duration, err
	}

	// Normalize any result to string
	str, ok := result.(string)
	if !ok {
		b, err := json.Marshal(result)
		if err != nil {
			str = fmt.Sprintf("%v", result)
		} else {
			str = string(b)
		}
	}

	// === Second Layer Defense: Tool Result Persistence ===

	// Check per-tool override: if MaxResultSizeChars == -1, skip persistence
	toolMaxChars := toolInfo.MaxResultSizeChars
	shouldPersist := true
	if toolMaxChars == -1 {
		shouldPersist = false
	}

	// Check per-message total budget
	r.mu.Lock()
	r.messageCharsUsed += len([]rune(str))
	currentMessageChars := r.messageCharsUsed
	r.mu.Unlock()

	if currentMessageChars > limits.MaxToolResultsPerMessageChars {
		// We've exceeded the per-message budget, force truncation
		targetChars := limits.MaxResultSizeChars
		if targetChars <= 0 {
			targetChars = 5000
		}
		runes := []rune(str)
		if len(runes) > targetChars {
			str = string(runes[:targetChars]) +
				fmt.Sprintf("\n... [context budget exceeded: total tool output in this cycle is %d chars, limit is %d] ...",
					currentMessageChars, limits.MaxToolResultsPerMessageChars)
		}
		return str, duration, nil
	}

	// Persist to disk if storage is configured and result is large enough
	if shouldPersist && storage != nil {
		persisted := storage.Persist(name, str)
		if persisted != nil {
			str = core.PersistedResultTag(persisted)
		}
	}

	return str, duration, nil
}
