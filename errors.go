package goreact

import (
	"errors"

	"github.com/DotNetAge/goreact/core"
)

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

// Skill errors (defined in core package, aliased here for backward compatibility)
var (
	ErrSkillNotFound    = core.ErrSkillNotFound
	ErrSkillExecution   = core.ErrSkillExecution
	ErrSkillCompilation = core.ErrSkillCompilation
)

// Memory errors (defined in core package, aliased here for backward compatibility)
var (
	ErrMemoryNotFound   = core.ErrMemoryNotFound
	ErrMemoryStorage    = core.ErrMemoryStorage
	ErrMemoryRetrieval  = core.ErrMemoryRetrieval
)
