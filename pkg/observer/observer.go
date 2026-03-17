package observer

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// Observer 观察模块接口
type Observer interface {
	Observe(result *types.ExecutionResult, context *core.Context) (*types.Feedback, error)
}
