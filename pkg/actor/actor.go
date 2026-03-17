package actor

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// Actor 行动模块接口
type Actor interface {
	Act(action *types.Action, context *core.Context) (*types.ExecutionResult, error)
}
