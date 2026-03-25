package mastersub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/thinker"
)

const MasterDecomposePrompt = `You are the Master Orchestrator for a ReAct Agent. 
Your goal is to decompose the user's high-level objective into a set of discrete, structured Tasks.

Each Task must have:
1. id: A unique short string (e.g., "t1", "fetch-data").
2. title: A short, descriptive title.
3. description: A clear instruction for a sub-agent to execute.
4. dependencies: A list of task IDs that MUST be completed before this task can start.
5. is_composite: Boolean. Set to true if the task requires complex reasoning or multiple steps. 
   Set to false if it's a simple, single-tool action.

Output ONLY a JSON array of tasks. No preamble, no markdown formatting.

Objective: %s
Available Skills Context: %s`

// Master 负责任务的全局拆解与编排
type Master struct {
	thinker thinker.Thinker
}

func NewMaster(t thinker.Thinker) *Master {
	return &Master{thinker: t}
}

// Decompose 将高层目标拆解为任务序列
func (m *Master) Decompose(ctx context.Context, goal string, skills string) ([]Task, error) {
	// 构造专门用于拆解任务的输入，增加 /json 协议前缀
	input := "/json " + fmt.Sprintf(MasterDecomposePrompt, goal, skills)
	
	// 创建一个临时的 PipelineContext 让 Master 进行推理
	// 注意：Master 的推理结果通常是 FinalResult 中的 JSON 字符串
	pctx := core.NewPipelineContext(ctx, "master-plan", input)
	
	err := m.thinker.Think(pctx)
	if err != nil {
		return nil, fmt.Errorf("master decomposition failed: %w", err)
	}

	// 解析 LLM 返回的 JSON
	var tasks []Task
	err = json.Unmarshal([]byte(pctx.FinalResult), &tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to parse master plan JSON: %w. Raw: %s", err, pctx.FinalResult)
	}

	return tasks, nil
}

// Replan 当子任务失败或环境发生重大变化时，请求 Master 重新规划
func (m *Master) Replan(ctx context.Context, currentTasks []Task, reason string) ([]Task, error) {
	fmt.Printf("[Master] 正在进行重规划，原因: %s\n", reason)
	return nil, nil
}
