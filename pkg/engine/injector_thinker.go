package engine

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// simpleThinker 简单思考器实现
type simpleThinker struct {
	fn func(string, *core.Context) (*types.Thought, error)
}

func (s *simpleThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
	return s.fn(task, ctx)
}
