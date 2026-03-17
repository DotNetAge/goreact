package engine

import (
	"context"

	"github.com/ray/goreact/pkg/types"
)

// Run 执行任务（简化版）
func (r *reactor) Run(task string) *types.Result {
	return r.Execute(context.Background(), task)
}
