package orchestration

import (
	"context"
	"fmt"
	"time"
)

// =============================================================================
// Orchestration Error
// =============================================================================

// OrchestrationError represents an orchestration error
type OrchestrationError struct {
	Type         ErrorType `json:"type"`
	SubTaskName  string    `json:"sub_task_name,omitempty"`
	AgentName    string    `json:"agent_name,omitempty"`
	Original     error     `json:"original,omitempty"`
	Recoverable  bool      `json:"recoverable"`
	Message      string    `json:"message"`
	RetryCount   int       `json:"retry_count"`
	MaxRetries   int       `json:"max_retries"`
}

// Error implements the error interface
func (e *OrchestrationError) Error() string {
	if e.Original != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Original)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *OrchestrationError) Unwrap() error {
	return e.Original
}

// CanRetry returns whether the error can be retried
func (e *OrchestrationError) CanRetry() bool {
	return e.Recoverable && e.RetryCount < e.MaxRetries
}

// NewOrchestrationError creates a new orchestration error
func NewOrchestrationError(typ ErrorType, message string, original error) *OrchestrationError {
	return &OrchestrationError{
		Type:        typ,
		Message:     message,
		Original:    original,
		Recoverable: isErrorRecoverable(typ),
		MaxRetries:  3,
	}
}

// isErrorRecoverable determines if an error type is recoverable
func isErrorRecoverable(typ ErrorType) bool {
	switch typ {
	case ErrorPlanningFailed, ErrorExecutionFailed, ErrorTimeout:
		return true
	case ErrorAgentSelectionFailed:
		return true // Can select alternative agent
	case ErrorResourceExhausted:
		return true // Can wait for resources
	case ErrorDependencyViolation:
		return false // Need to replan
	default:
		return false
	}
}

// =============================================================================
// Error Handler
// =============================================================================

// ErrorHandler handles orchestration errors
type ErrorHandler struct {
	config *ErrorHandlerConfig
}

// ErrorHandlerConfig represents error handler configuration
type ErrorHandlerConfig struct {
	MaxRetries     int           `json:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	FailFast       bool          `json:"fail_fast"`
}

// DefaultErrorHandlerConfig returns default error handler config
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		MaxRetries:     3,
		InitialBackoff: time.Second,
		MaxBackoff:     30 * time.Second,
		FailFast:       false,
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config *ErrorHandlerConfig) *ErrorHandler {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}
	return &ErrorHandler{config: config}
}

// Handle handles an orchestration error
func (h *ErrorHandler) Handle(err *OrchestrationError) *ErrorAction {
	switch err.Type {
	case ErrorPlanningFailed:
		return &ErrorAction{
			Type:    ActionTypeRetry,
			Message: "Retrying planning",
		}
	case ErrorAgentSelectionFailed:
		return &ErrorAction{
			Type:    ActionTypeAlternative,
			Message: "Selecting alternative agent",
		}
	case ErrorExecutionFailed:
		if err.CanRetry() {
			return &ErrorAction{
				Type:    ActionTypeRetry,
				Message: fmt.Sprintf("Retrying execution (attempt %d/%d)", err.RetryCount+1, err.MaxRetries),
			}
		}
		return &ErrorAction{
			Type:    ActionTypeDegraded,
			Message: "Executing with degraded mode",
		}
	case ErrorTimeout:
		return &ErrorAction{
			Type:    ActionTypeRetry,
			Message: "Retrying after timeout",
		}
	case ErrorResourceExhausted:
		return &ErrorAction{
			Type:    ActionTypeWait,
			Message: "Waiting for resources",
		}
	case ErrorDependencyViolation:
		return &ErrorAction{
			Type:    ActionTypeReplan,
			Message: "Replanning due to dependency violation",
		}
	default:
		return &ErrorAction{
			Type:    ActionTypeFail,
			Message: "Unrecoverable error",
		}
	}
}

// ErrorActionType defines error action types
type ErrorActionType string

const (
	ActionTypeRetry       ErrorActionType = "retry"
	ActionTypeAlternative ErrorActionType = "alternative"
	ActionTypeDegraded    ErrorActionType = "degraded"
	ActionTypeWait        ErrorActionType = "wait"
	ActionTypeReplan      ErrorActionType = "replan"
	ActionTypeFail        ErrorActionType = "fail"
)

// ErrorAction represents an action to take for an error
type ErrorAction struct {
	Type    ErrorActionType `json:"type"`
	Message string          `json:"message"`
}

// =============================================================================
// Retry Handler Implementation
// =============================================================================

// RetryHandlerImpl implements RetryHandler
type RetryHandlerImpl struct {
	config *ErrorHandlerConfig
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(config *ErrorHandlerConfig) *RetryHandlerImpl {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}
	return &RetryHandlerImpl{config: config}
}

// Retry retries the operation with default backoff
func (h *RetryHandlerImpl) Retry(ctx context.Context, op func() error) error {
	return h.WithBackoff(ctx, op, h.config.InitialBackoff, h.config.MaxBackoff)
}

// WithBackoff executes with exponential backoff
func (h *RetryHandlerImpl) WithBackoff(ctx context.Context, op func() error, initial, max time.Duration) error {
	var lastErr error
	backoff := initial
	
	for i := 0; i < h.config.MaxRetries; i++ {
		err := op()
		if err == nil {
			return nil
		}
		
		// Check if error is permanent
		if !isRecoverableError(err) {
			return err
		}
		
		lastErr = err
		
		// Wait with backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		
		// Increase backoff exponentially
		backoff = backoff * 2
		if backoff > max {
			backoff = max
		}
	}
	
	return lastErr
}

// isRecoverableError checks if an error is recoverable
func isRecoverableError(err error) bool {
	if orchErr, ok := err.(*OrchestrationError); ok {
		return orchErr.Recoverable
	}
	return true
}

// =============================================================================
// Error Strategies
// =============================================================================

// ErrorStrategyTable maps error types to handling strategies
type ErrorStrategyTable struct {
	strategies map[ErrorType]*ErrorStrategy
}

// ErrorStrategy defines how to handle an error type
type ErrorStrategy struct {
	Type          ErrorType       `json:"type"`
	Strategy      ErrorStrategyType `json:"strategy"`
	MaxRetries    int             `json:"max_retries"`
	FallbackAction ErrorActionType `json:"fallback_action"`
}

// ErrorStrategyType defines error strategy types
type ErrorStrategyType string

const (
	StrategyRetry       ErrorStrategyType = "retry"
	StrategyAlternative ErrorStrategyType = "alternative"
	StrategyDegraded    ErrorStrategyType = "degraded"
	StrategyReplan      ErrorStrategyType = "replan"
	StrategyFail        ErrorStrategyType = "fail"
)

// NewErrorStrategyTable creates a new error strategy table
func NewErrorStrategyTable() *ErrorStrategyTable {
	return &ErrorStrategyTable{
		strategies: map[ErrorType]*ErrorStrategy{
			ErrorPlanningFailed: {
				Type:           ErrorPlanningFailed,
				Strategy:       StrategyRetry,
				MaxRetries:     3,
				FallbackAction: ActionTypeFail,
			},
			ErrorAgentSelectionFailed: {
				Type:           ErrorAgentSelectionFailed,
				Strategy:       StrategyAlternative,
				MaxRetries:     3,
				FallbackAction: ActionTypeDegraded,
			},
			ErrorExecutionFailed: {
				Type:           ErrorExecutionFailed,
				Strategy:       StrategyRetry,
				MaxRetries:     3,
				FallbackAction: ActionTypeDegraded,
			},
			ErrorTimeout: {
				Type:           ErrorTimeout,
				Strategy:       StrategyRetry,
				MaxRetries:     2,
				FallbackAction: ActionTypeFail,
			},
			ErrorResourceExhausted: {
				Type:           ErrorResourceExhausted,
				Strategy:       StrategyRetry,
				MaxRetries:     5,
				FallbackAction: ActionTypeDegraded,
			},
			ErrorDependencyViolation: {
				Type:           ErrorDependencyViolation,
				Strategy:       StrategyReplan,
				MaxRetries:     1,
				FallbackAction: ActionTypeFail,
			},
		},
	}
}

// GetStrategy returns the strategy for an error type
func (t *ErrorStrategyTable) GetStrategy(typ ErrorType) *ErrorStrategy {
	return t.strategies[typ]
}

// SetStrategy sets a strategy for an error type
func (t *ErrorStrategyTable) SetStrategy(strategy *ErrorStrategy) {
	t.strategies[strategy.Type] = strategy
}
