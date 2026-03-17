package engine

import (
	"context"

	"github.com/ray/goreact/pkg/tools"
	"github.com/ray/goreact/pkg/types"
)

// Reactor ReAct 引擎接口
type Reactor interface {
	Execute(ctx context.Context, task string) *types.Result
	Run(task string) *types.Result
	Close() error
	RegisterTool(t tools.Tool)
	RegisterTools(ts ...tools.Tool)
}
