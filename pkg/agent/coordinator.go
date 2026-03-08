package agent

import (
	"fmt"
	"strings"
	"time"
)

// Executor 执行器接口（用于执行 Agent 的任务）
type Executor interface {
	Execute(prompt string, context interface{}) *Result
}

// Result 执行结果
type Result struct {
	Success  bool
	Output   string
	Error    error
	Trace    []TraceStep
	Metadata map[string]interface{}
	EndTime  time.Time
}

// TraceStep 执行轨迹
type TraceStep struct {
	Step      int
	Type      string
	Content   string
	Timestamp time.Time
}

// Coordinator 智能体协调器
type Coordinator struct {
	agents         []*Agent
	executor       Executor       // 执行器（注入）
	taskDecomposer TaskDecomposer // 任务拆解器（注入）
}

// NewCoordinator 创建协调器
func NewCoordinator(executor Executor) *Coordinator {
	return &Coordinator{
		agents:         make([]*Agent, 0),
		executor:       executor,
		taskDecomposer: NewDefaultTaskDecomposer(),
	}
}

// RegisterAgent 注册智能体
func (c *Coordinator) RegisterAgent(agent *Agent) {
	c.agents = append(c.agents, agent)
}

// SetTaskDecomposer 设置任务拆解器（外部接入点）
func (c *Coordinator) SetTaskDecomposer(decomposer TaskDecomposer) {
	c.taskDecomposer = decomposer
}

// ExecuteTask 执行任务，自动选择合适的智能体
func (c *Coordinator) ExecuteTask(task string, context interface{}) *Result {
	// 1. 使用任务拆解器拆解任务
	subTasks, err := c.taskDecomposer.Decompose(task, context)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Errorf("failed to decompose task: %w", err),
			EndTime: time.Now(),
		}
	}

	// 2. 为每个子任务选择合适的 Agent 并执行
	result := &Result{
		Trace:    make([]TraceStep, 0),
		Metadata: make(map[string]interface{}),
		Success:  true,
	}

	subTaskResults := make(map[string]*Result)

	for _, subTask := range subTasks {
		// 检查依赖
		for _, depID := range subTask.Dependencies {
			if depResult, ok := subTaskResults[depID]; !ok || !depResult.Success {
				result.Success = false
				result.Error = fmt.Errorf("dependency %s failed", depID)
				return result
			}
		}

		// 选择最合适的 Agent
		selectedAgent := c.selectAgent(subTask.Description)
		if selectedAgent == nil {
			result.Success = false
			result.Error = fmt.Errorf("no suitable agent found for: %s", subTask.Description)
			return result
		}

		// 构建完整提示词：System Prompt + Task
		fullPrompt := selectedAgent.SystemPrompt + "\n\nTask: " + subTask.Description

		// 使用 Executor 执行
		subResult := c.executor.Execute(fullPrompt, context)
		subTaskResults[subTask.ID] = subResult

		// 记录轨迹
		result.Trace = append(result.Trace, TraceStep{
			Step:      len(result.Trace) + 1,
			Type:      "subtask",
			Content:   fmt.Sprintf("Agent %s executed: %s", selectedAgent.Name, subTask.Description),
			Timestamp: time.Now(),
		})

		if !subResult.Success {
			result.Success = false
			result.Error = subResult.Error
			return result
		}

		// 合并输出
		if result.Output == "" {
			result.Output = subResult.Output
		} else {
			result.Output += "\n" + subResult.Output
		}
	}

	result.EndTime = time.Now()
	return result
}

// selectAgent 根据任务选择合适的智能体（简单的关键词匹配）
func (c *Coordinator) selectAgent(task string) *Agent {
	taskLower := strings.ToLower(task)
	var bestAgent *Agent
	bestScore := 0

	for _, agent := range c.agents {
		score := 0
		promptLower := strings.ToLower(agent.SystemPrompt)

		// 简单的关键词匹配
		words := strings.Fields(promptLower)
		for _, word := range words {
			if len(word) > 3 && strings.Contains(taskLower, word) {
				score++
			}
		}

		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	// 如果没有匹配，返回第一个
	if bestAgent == nil && len(c.agents) > 0 {
		bestAgent = c.agents[0]
	}

	return bestAgent
}
