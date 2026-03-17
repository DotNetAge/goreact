package steps

import (
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

// ReActData 承载 ReAct 循环的强类型数据
type ReActData struct {
	Task         string                 // 当前任务
	Thought      *types.Thought         // 当前思考
	ExecResult   *types.ExecutionResult // 执行结果
	Feedback     *types.Feedback        // 观察反馈
	HistorySteps []HistoryStep          // 历史记录步骤
}

// HistoryStep 历史记录步骤
type HistoryStep struct {
	Action   string `json:"action"`
	Result   string `json:"result"`
	Feedback string `json:"feedback"`
}

// ReActState 承载 ReAct 循环的状态和数据（强类型化）
type ReActState struct {
	Task         string                 // 当前任务
	Iteration    int                    // 当前迭代次数
	Result       *types.Result          // 执行结果
	ExecCtx      *core.Context          // 执行上下文
	ShouldStop   bool                   // 是否应该停止循环
	LastThought  *types.Thought         // 最后一次思考
	LastResult   *types.ExecutionResult // 最后一次执行结果
	LastFeedback *types.Feedback        // 最后一次观察反馈
	Data         *ReActData             // 强类型数据容器
}

// NewReActState 创建新的 ReAct 状态
func NewReActState(task string) *ReActState {
	data := &ReActData{
		Task:         task,
		HistorySteps: make([]HistoryStep, 0),
	}

	state := &ReActState{
		Task:       task,
		ExecCtx:    core.NewContext(),
		Iteration:  0,
		ShouldStop: false,
		Data:       data,
		Result: &types.Result{
			Trace:     make([]types.TraceStep, 0),
			Metadata:  make(map[string]interface{}),
			StartTime: time.Now(),
		},
	}

	return state
}
