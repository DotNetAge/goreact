package engine

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// Think 设置思考函数（方法注入）
func (r *reactor) Think(fn func(string, *core.Context) (*types.Thought, error)) *reactor {
	r.thinker = &simpleThinker{fn: fn}
	return r
}

// Act 设置行动函数（方法注入）
func (r *reactor) Act(fn func(*types.Action, *core.Context) (*types.ExecutionResult, error)) *reactor {
	r.actor = &simpleActor{fn: fn}
	return r
}

// Observe 设置观察函数（方法注入）
func (r *reactor) Observe(fn func(*types.ExecutionResult, *core.Context) (*types.Feedback, error)) *reactor {
	r.observer = &simpleObserver{fn: fn}
	return r
}

// Loop 设置循环控制函数（方法注入）
func (r *reactor) Loop(fn func(*types.LoopState) *types.LoopAction) *reactor {
	r.loopController = &simpleLoopController{fn: fn}
	return r
}
