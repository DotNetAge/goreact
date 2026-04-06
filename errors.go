package goreact

import "errors"

// Agent errors
var (
	ErrAgentNotFound    = errors.New("agent not found")
	ErrAgentNotActive   = errors.New("agent not active")
	ErrAgentMaxSteps    = errors.New("max steps exceeded")
)

// Tool errors
var (
	ErrToolNotFound     = errors.New("tool not found")
	ErrToolExecution    = errors.New("tool execution failed")
	ErrToolValidation   = errors.New("tool validation failed")
	ErrToolUnauthorized = errors.New("tool unauthorized")
)

// Skill errors
var (
	ErrSkillNotFound    = errors.New("skill not found")
	ErrSkillExecution   = errors.New("skill execution failed")
	ErrSkillCompilation = errors.New("skill compilation failed")
)

// Memory errors
var (
	ErrMemoryNotFound   = errors.New("memory not found")
	ErrMemoryStorage    = errors.New("memory storage failed")
	ErrMemoryRetrieval  = errors.New("memory retrieval failed")
)
