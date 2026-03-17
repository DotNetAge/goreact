package engine

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// simpleObserver 简单观察者实现
type simpleObserver struct {
	fn func(*types.ExecutionResult, *core.Context) (*types.Feedback, error)
}

func (s *simpleObserver) Observe(result *types.ExecutionResult, ctx *core.Context) (*types.Feedback, error) {
	return s.fn(result, ctx)
}
