package engine

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// simpleActor 简单行动器实现
type simpleActor struct {
	fn func(*types.Action, *core.Context) (*types.ExecutionResult, error)
}

func (s *simpleActor) Act(action *types.Action, ctx *core.Context) (*types.ExecutionResult, error) {
	return s.fn(action, ctx)
}
