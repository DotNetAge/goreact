package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/engine"
	"github.com/DotNetAge/goreact/pkg/memory"
	"github.com/DotNetAge/goreact/pkg/skill"
)

// Agent 智能体实体（Rich Domain Model）
// Agent = 角色设定(Config) + 大脑与身体(Reactor/Engine) + 记忆体系(MemoryBank)
type Agent struct {
	// --- 静态配置与描述 (Config & Identity) ---
	AgentName    string            // 智能体名称
	AgentDesc    string            // 智能体描述（用于选择匹配）
	SystemPrompt string            // 系统提示词（定义 Agent 的角色和行为）
	ModelName    string            // 使用的模型名称（引用 Model 配置）
	Metadata     map[string]string // 元数据（可选）

	// --- 运行时状态机引擎 (Runtime Engine) ---
	// 引擎是 Agent 的核心执行器，负责 ReAct 循环流转。
	reactor *engine.Reactor

	// --- 技能与演化 (Skill & Evolution) ---
	skillManager skill.Manager

	// --- 记忆体系 (Memory Architecture) ---
	// 专属记忆体，包含短期工作记忆、语义知识库与肌肉记忆。
	memoryBank memory.MemoryBank
}

// NewAgent 创建一个新的智能体模版/配置
func NewAgent(name, description, systemPrompt, modelName string) *Agent {
	return &Agent{
		AgentName:    name,
		AgentDesc:    description,
		SystemPrompt: systemPrompt,
		ModelName:    modelName,
		Metadata:     make(map[string]string),
	}
}

// =========================================================================
// 应用层入口 (Application API)
// =========================================================================

// Chat 执行多轮对话流任务，内部启动 ReAct 引擎循环
func (a *Agent) Chat(ctx context.Context, sessionID, task string, opts ...core.ContextOption) (string, error) {
	if a.reactor == nil {
		return "", fmt.Errorf("AgentNotAssembled: The agent has not been assembled by a Builder. Missing reactor engine.")
	}

	// 调用底层 Reactor 执行 ReAct 循环
	reactCtx, err := a.reactor.Run(ctx, sessionID, task, opts...)

	// 记录 Skill 执行数据 (如果命中)
	if a.skillManager != nil && reactCtx != nil {
		if activeSkill, ok := reactCtx.Get("active_skill"); ok {
			if skillName, isStr := activeSkill.(string); isStr {
				success := err == nil && reactCtx.Error == nil
				tokens := 0
				if reactCtx.TotalTokens != nil {
					tokens = reactCtx.TotalTokens.TotalTokens
				}

				// 质量打分策略: 成功则给 1.0, 失败给 0.0
				score := 0.0
				if success {
					score = 1.0
				}

				_ = a.skillManager.RecordExecution(
					skillName,
					success,
					time.Since(reactCtx.StartTime),
					tokens,
					score,
				)
			}
		}
	}

	if err != nil {
		return "", err
	}

	// 提取最终输出
	if reactCtx.FinalResult != "" {
		return reactCtx.FinalResult, nil
	}
	return "No final answer produced.", nil
}
