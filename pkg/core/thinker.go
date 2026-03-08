package core

import "github.com/ray/goreact/pkg/types"

// Thinker 思考模块接口
type Thinker interface {
	Think(task string, context *Context) (*types.Thought, error)
}
