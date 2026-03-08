package thinker

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// Thinker 思考模块接口
type Thinker interface {
	Think(task string, context *core.Context) (*types.Thought, error)
}
