package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
)

// Executor executes tools with validation and error handling
type Executor struct {
	registry     *Registry
	whitelist    *Whitelist
	maxRetries   int
	timeout      time.Duration
	allowedLevels []common.SecurityLevel
}

// ExecutorOption is a function that configures the Executor
type ExecutorOption func(*Executor)

// NewExecutor creates a new Executor
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		registry:      globalRegistry,
		whitelist:     NewWhitelist(),
		maxRetries:    common.DefaultActorMaxRetries,
		timeout:       common.DefaultActorTimeout,
		allowedLevels: []common.SecurityLevel{common.LevelSafe, common.LevelSensitive},
	}
	
	for _, opt := range opts {
		opt(e)
	}
	
	return e
}

// WithRegistry sets the registry
func WithRegistry(registry *Registry) ExecutorOption {
	return func(e *Executor) {
		e.registry = registry
	}
}

// WithWhitelist sets the whitelist
func WithWhitelist(whitelist *Whitelist) ExecutorOption {
	return func(e *Executor) {
		e.whitelist = whitelist
	}
}

// WithMaxRetries sets the max retries
func WithMaxRetries(maxRetries int) ExecutorOption {
	return func(e *Executor) {
		e.maxRetries = maxRetries
	}
}

// WithTimeout sets the timeout
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = timeout
	}
}

// WithAllowedLevels sets the allowed security levels
func WithAllowedLevels(levels []common.SecurityLevel) ExecutorOption {
	return func(e *Executor) {
		e.allowedLevels = levels
	}
}

// Execute executes a tool with the given parameters
func (e *Executor) Execute(ctx context.Context, name string, params map[string]any) (*core.ActionResult, error) {
	startTime := time.Now()
	
	// Get the tool
	t, exists := e.registry.Get(name)
	if !exists {
		return nil, common.NewError(common.ErrCodeToolNotFound, fmt.Sprintf("tool %s not found", name), nil)
	}
	
	// Check security level
	if !e.isLevelAllowed(t.SecurityLevel()) {
		return nil, common.NewError(common.ErrCodeToolUnauthorized, 
			fmt.Sprintf("tool %s has security level %s which is not allowed", name, t.SecurityLevel()), nil)
	}
	
	// Check whitelist for sensitive tools
	if t.SecurityLevel() >= common.LevelSensitive && !e.whitelist.IsAllowed(name) {
		return nil, common.NewError(common.ErrCodeToolUnauthorized,
			fmt.Sprintf("tool %s requires authorization", name), nil)
	}
	
	// Create timeout context
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}
	
	// Execute with retry
	var result any
	var err error
	
	for i := 0; i <= e.maxRetries; i++ {
		result, err = t.Run(ctx, params)
		if err == nil {
			break
		}
		
		// Check if error is retryable
		if !isRetryableError(err) {
			break
		}
		
		// Wait before retry
		if i < e.maxRetries {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
	}
	
	// Build result
	duration := time.Since(startTime)
	actionResult := &core.ActionResult{
		Duration: duration,
		Metadata: make(map[string]any),
	}
	
	if err != nil {
		actionResult.Success = false
		actionResult.Error = err.Error()
		return actionResult, err
	}
	
	actionResult.Success = true
	actionResult.Result = result
	actionResult.WithTool(name)
	
	return actionResult, nil
}

// ValidateParams validates tool parameters
func (e *Executor) ValidateParams(name string, params map[string]any) error {
	info, exists := e.registry.GetInfo(name)
	if !exists {
		return common.NewError(common.ErrCodeToolNotFound, fmt.Sprintf("tool %s not found", name), nil)
	}
	
	// Check required parameters
	for _, param := range info.Parameters {
		if param.Required {
			value, exists := params[param.Name]
			if !exists || value == nil {
				return common.NewError(common.ErrCodeToolValidation,
					fmt.Sprintf("required parameter %s is missing", param.Name), nil)
			}
		}
	}
	
	return nil
}

// Authorize authorizes a tool for execution
func (e *Executor) Authorize(name string, permanent bool) error {
	return e.whitelist.Add(name, permanent)
}

// Revoke revokes tool authorization
func (e *Executor) Revoke(name string) {
	e.whitelist.Remove(name)
}

// isLevelAllowed checks if a security level is allowed
func (e *Executor) isLevelAllowed(level common.SecurityLevel) bool {
	for _, allowed := range e.allowedLevels {
		if allowed == level {
			return true
		}
	}
	return false
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	// Timeout errors are retryable
	if common.IsTimeoutError(err) {
		return true
	}
	// Context canceled is not retryable
	if err == context.Canceled {
		return false
	}
	return false
}
