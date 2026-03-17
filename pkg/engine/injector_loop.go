package engine

import "github.com/ray/goreact/pkg/types"

// simpleLoopController 简单循环控制器实现
type simpleLoopController struct {
	fn func(*types.LoopState) *types.LoopAction
}

func (s *simpleLoopController) Control(state *types.LoopState) *types.LoopAction {
	return s.fn(state)
}
