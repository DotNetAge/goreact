package types

import (
	"errors"
	"time"
)

// 错误定义
var (
	ErrExecutorNotSet = errors.New("executor not set for agent")
)

// Result 任务执行结果
type Result struct {
	Success   bool           // 是否成功
	Output    string         // 输出结果
	Error     error          // 错误信息
	Trace     []TraceStep    // 执行轨迹
	Metadata  map[string]any // 元数据
	StartTime time.Time      // 开始时间
	EndTime   time.Time      // 结束时间
}

// TraceStep 执行轨迹的一步
type TraceStep struct {
	Step      int       // 步骤编号
	Type      string    // 类型：think/act/observe
	Content   string    // 内容
	Timestamp time.Time // 时间戳
}

// Thought 思考结果
type Thought struct {
	Reasoning    string         // 推理过程
	Action       *Action        // 要执行的动作（可能为空）
	ShouldFinish bool           // 是否应该结束
	FinalAnswer  string         // 最终答案（如果应该结束）
	Metadata     map[string]any // 元数据
}

// Action 要执行的动作
type Action struct {
	ToolName   string         // 工具名称
	Parameters map[string]any // 参数
	Reasoning  string         // 为什么要执行这个动作
}

// ExecutionResult 动作执行结果
type ExecutionResult struct {
	Success  bool           // 是否成功
	Output   any            // 输出结果
	Error    error          // 错误信息
	Metadata map[string]any // 元数据
}

// Feedback 观察反馈
type Feedback struct {
	ShouldContinue bool           // 是否应该继续
	Message        string         // 反馈消息
	Metadata       map[string]any // 元数据
}

// LoopState 循环状态
type LoopState struct {
	Iteration    int              // 当前迭代次数
	Task         string           // 任务描述
	LastThought  *Thought         // 上一次思考
	LastResult   *ExecutionResult // 上一次执行结果
	LastFeedback *Feedback        // 上一次反馈
}

// LoopAction 循环控制动作
type LoopAction struct {
	ShouldContinue bool   // 是否继续循环
	Reason         string // 原因
}
