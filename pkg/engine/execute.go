package engine

import (
	"context"
	"fmt"
	"time"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/ray/goreact/pkg/agent"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/skill"
	"github.com/ray/goreact/pkg/types"
)

func (r *reactor) Execute(ctx context.Context, task string) *types.Result {
	if ctx == nil {
		ctx = context.Background()
	}

	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		r.metrics.RecordLatency("engine.execute", latency)
	}()

	r.logger.Info("Task execution started",
		log.String("task", task),
		log.Any("start_time", startTime),
	)

	// 1. Agent 选择
	var selectedAgent *agent.Agent
	var systemPrompt string
	if r.agentManager != nil {
		if selResult, err := r.agentManager.SelectAgentWithResult(task); err == nil && selResult != nil {
			selectedAgent = selResult.Agent
			systemPrompt = selResult.Agent.SystemPrompt

			r.logger.Info("Agent selected",
				log.String("agent_name", selResult.Agent.Name),
				log.String("model_name", selResult.Agent.ModelName),
				log.Float64("selection_score", selResult.Score),
			)
		}
	}

	// 2. Model 选择和 LLM Client 构建
	var llmClient gochatcore.Client
	if selectedAgent != nil && selectedAgent.ModelName != "" && r.modelManager != nil {
		if client, err := r.modelManager.CreateLLMClient(selectedAgent.ModelName); err == nil {
			llmClient = client
			r.logger.Info("LLM client created", log.String("model_name", selectedAgent.ModelName))
		} else {
			llmClient = r.llmClient
			r.logger.Warn("Failed to create LLM client, using default", log.Err(err))
		}
	} else {
		llmClient = r.llmClient
	}

	// 3. 重新创建 Thinker（使用选定的 LLM Client 和 System Prompt）
	currentThinker := r.thinker
	if llmClient != r.llmClient || systemPrompt != "" {
		currentThinker = r.getOrCreateThinker(llmClient, systemPrompt)
	}

	// 4. Skill 选择和注入
	var selectedSkill *skill.Skill
	originalTask := task

	if r.skillManager != nil {
		if sk, err := r.skillManager.SelectSkill(task); err == nil && sk != nil {
			selectedSkill = sk
			r.logger.Info("Skill selected", log.String("skill_name", sk.Name))

			task = fmt.Sprintf(`%s

# Skill Instructions

You are now using the skill: **%s**

Description: %s

Please follow these instructions carefully:

%s

---

Now, complete the original task: %s`,
				task,
				selectedSkill.Name,
				selectedSkill.Description,
				selectedSkill.Instructions,
				originalTask)
		}
	}

	// 5. 创建 ReAct 状态
	state := NewReActState(task)
	state.Result.StartTime = startTime

	if selectedSkill != nil {
		state.Result.Metadata["selected_skill"] = selectedSkill.Name
		state.Result.Metadata["skill_description"] = selectedSkill.Description
	}

	// 6. 检查缓存
	if r.cache != nil {
		cacheKey := r.generateCacheKey(task)
		if cached, ok := r.cache.Get(cacheKey); ok {
			if result, ok := cached.(*types.Result); ok {
				result.Metadata["cached"] = true
				result.Metadata["cache_hit_time"] = time.Now()
				r.logger.Info("Cache hit", log.String("cache_key", cacheKey))
				return result
			}
		}
	}

	// 7. 构建并执行 Pipeline
	pipeline := BuildReActPipeline(
		currentThinker, r.actor, r.observer, r.loopController,
		r.logger, r.metrics,
		r.maxRetries, r.retryInterval,
		DefaultMaxIterations,
	)

	for !state.ShouldStop {
		select {
		case <-ctx.Done():
			state.Result.Success = false
			state.Result.Error = fmt.Errorf("execution cancelled: %w", ctx.Err())
			state.Result.EndTime = time.Now()
			r.logger.Warn("Task execution cancelled", log.Int("iterations", state.Iteration))
			return state.Result
		default:
		}

		if err := pipeline.Execute(ctx, state); err != nil {
			state.Result.Success = false
			state.Result.Error = err
			state.Result.EndTime = time.Now()
			r.logger.Error("Pipeline execution failed", log.Err(err))
			break
		}
	}

	// 8. 缓存结果
	if r.cache != nil && state.Result.Success {
		cacheKey := r.generateCacheKey(task)
		r.cache.Set(cacheKey, state.Result, 0) // 使用默认 TTL
		r.logger.Debug("Result cached", log.String("cache_key", cacheKey))
	}

	state.Result.EndTime = time.Now()
	r.logger.Info("Task execution completed",
		log.Bool("success", state.Result.Success),
		log.Int("iterations", state.Iteration),
	)

	return state.Result
}
