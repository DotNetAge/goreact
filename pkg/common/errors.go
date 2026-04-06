// Package common provides common types, errors, and utilities for the goreact framework.
package common

import (
	"errors"
	"fmt"
)

// Error codes for the goreact framework
const (
	// General errors
	ErrCodeUnknown         = "UNKNOWN"
	ErrCodeInvalidArgument = "INVALID_ARGUMENT"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeAlreadyExists   = "ALREADY_EXISTS"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeTimeout         = "TIMEOUT"
	ErrCodeCanceled        = "CANCELED"

	// Agent errors
	ErrCodeAgentNotFound    = "AGENT_NOT_FOUND"
	ErrCodeAgentNotActive   = "AGENT_NOT_ACTIVE"
	ErrCodeAgentMaxSteps    = "AGENT_MAX_STEPS_EXCEEDED"

	// Tool errors
	ErrCodeToolNotFound     = "TOOL_NOT_FOUND"
	ErrCodeToolExecution    = "TOOL_EXECUTION_FAILED"
	ErrCodeToolValidation   = "TOOL_VALIDATION_FAILED"
	ErrCodeToolUnauthorized = "TOOL_UNAUTHORIZED"

	// Skill errors
	ErrCodeSkillNotFound    = "SKILL_NOT_FOUND"
	ErrCodeSkillExecution   = "SKILL_EXECUTION_FAILED"
	ErrCodeSkillCompilation = "SKILL_COMPILATION_FAILED"

	// Memory errors
	ErrCodeMemoryNotFound   = "MEMORY_NOT_FOUND"
	ErrCodeMemoryStorage    = "MEMORY_STORAGE_FAILED"
	ErrCodeMemoryRetrieval  = "MEMORY_RETRIEVAL_FAILED"

	// Reactor errors
	ErrCodeReactorPaused    = "REACTOR_PAUSED"
	ErrCodeReactorStopped   = "REACTOR_STOPPED"
	ErrCodeReactorFailed    = "REACTOR_FAILED"

	// LLM errors
	ErrCodeLLMTimeout       = "LLM_TIMEOUT"
	ErrCodeLLMError         = "LLM_ERROR"
	ErrCodeLLMParseError    = "LLM_PARSE_ERROR"

	// Evolution errors
	ErrCodeEvolutionFailed  = "EVOLUTION_FAILED"
	ErrCodeEvolutionPending = "EVOLUTION_PENDING"
)

// Error represents a structured error with code and message
type Error struct {
	Code    string
	Message string
	Cause   error
	Details map[string]any
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError creates a new Error with the given code and message
func NewError(code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewErrorWithDetails creates a new Error with details
func NewErrorWithDetails(code, message string, cause error, details map[string]any) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: details,
	}
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == ErrCodeNotFound ||
			e.Code == ErrCodeAgentNotFound ||
			e.Code == ErrCodeToolNotFound ||
			e.Code == ErrCodeSkillNotFound ||
			e.Code == ErrCodeMemoryNotFound
	}
	return false
}

// IsTimeoutError checks if the error is a timeout error
func IsTimeoutError(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == ErrCodeTimeout || e.Code == ErrCodeLLMTimeout
	}
	return false
}

// IsPermissionError checks if the error is a permission error
func IsPermissionError(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == ErrCodePermissionDenied || e.Code == ErrCodeToolUnauthorized
	}
	return false
}

// Common errors
var (
	ErrNotFound         = NewError(ErrCodeNotFound, "resource not found", nil)
	ErrInvalidArgument  = NewError(ErrCodeInvalidArgument, "invalid argument", nil)
	ErrPermissionDenied = NewError(ErrCodePermissionDenied, "permission denied", nil)
	ErrTimeout          = NewError(ErrCodeTimeout, "operation timeout", nil)
	ErrCanceled         = NewError(ErrCodeCanceled, "operation canceled", nil)
)
