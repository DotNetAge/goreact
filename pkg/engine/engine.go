package engine

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ray/goreact/pkg/agent"
	"github.com/ray/goreact/pkg/cache"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/llm/mock"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/model"
	"github.com/ray/goreact/pkg/skill"
	"github.com/ray/goreact/pkg/tool"
	"github.com/ray/goreact/pkg/tool/provider"
	"github.com/ray/goreact/pkg/types"
)

const (
	DefaultMaxIterations   = 10
	DefaultMaxRetries      = 3
	DefaultRetryInterval   = 1 * time.Second
	DefaultMaxTraceSize    = 1000
	DefaultFallbackMessage = "LLM unavailable, please try again later"
)

// noOpLogger 是一个空操作 logger，当默认 logger 创建失败时使用
type noOpLogger struct{}

func (n *noOpLogger) Debug(msg string, fields ...log.Field) {}
func (n *noOpLogger) Info(msg string, fields ...log.Field)  {}
func (n *noOpLogger) Warn(msg string, fields ...log.Field)  {}
func (n *noOpLogger) Error(msg string, fields ...log.Field) {}
func (n *noOpLogger) With(fields ...log.Field) log.Logger   { return n }

// Engine ReAct 引擎核心
type Engine struct {
	thinker          core.Thinker
	actor            core.Actor
	observer         core.Observer
	loopController   core.LoopController
	toolManager      *tool.Manager
	skillManager     skill.Manager
	agentManager     *agent.Manager
	modelManager     *model.Manager
	providerRegistry *provider.Registry
	llmClient        llm.Client
	cache            cache.Cache
	maxRetries       int
	retryInterval    time.Duration
	maxTraceSize     int // 最大 Trace 条目数，0 表示无限制
	metrics          metrics.Metrics
	logger           log.Logger
	thinkerCache     map[string]core.Thinker // Thinker 缓存，key: cacheKey
}

// New 创建新的引擎实例
func New(options ...Option) *Engine {
	// 创建默认 logger，如果失败则使用 nil（会在后续检查）
	defaultLogger, err := log.NewDefaultZapLogger()
	if err != nil {
		// 如果 logger 创建失败，使用 nil，后续会检查
		defaultLogger = nil
	}

	engine := &Engine{
		toolManager:      tool.NewManager(),
		providerRegistry: provider.NewRegistry(),
		llmClient:        mock.NewMockClient([]string{}),
		loopController:   core.NewDefaultLoopController(DefaultMaxIterations),
		maxRetries:       DefaultMaxRetries,
		retryInterval:    DefaultRetryInterval,
		maxTraceSize:     DefaultMaxTraceSize,
		metrics:          metrics.NewDefaultMetrics(),
		logger:           defaultLogger,
		thinkerCache:     make(map[string]core.Thinker),
	}

	// 应用选项
	for _, opt := range options {
		opt(engine)
	}

	// 如果 logger 仍然是 nil，创建一个 no-op logger
	if engine.logger == nil {
		engine.logger = &noOpLogger{}
	}

	// 如果没有设置核心模块，使用默认实现
	if engine.thinker == nil {
		engine.thinker = engine.getOrCreateThinker(engine.llmClient, "")
	}
	if engine.actor == nil {
		engine.actor = core.NewDefaultActor(engine.toolManager)
	}
	if engine.observer == nil {
		engine.observer = core.NewDefaultObserver()
	}

	// 从 Provider Registry 自动发现和注册工具
	if engine.providerRegistry != nil {
		if tools, err := engine.providerRegistry.DiscoverAllTools(); err == nil {
			for _, t := range tools {
				engine.toolManager.RegisterTool(t)
			}
			// 更新 Thinker（清空缓存以使用新的工具描述）
			engine.thinkerCache = make(map[string]core.Thinker)
			engine.thinker = engine.getOrCreateThinker(engine.llmClient, "")
		}
	}

	return engine
}

// RegisterTool 注册单个工具
func (e *Engine) RegisterTool(t tool.Tool) {
	e.toolManager.RegisterTool(t)
	// 清空 Thinker 缓存以使用新的工具描述
	e.thinkerCache = make(map[string]core.Thinker)
	e.thinker = e.getOrCreateThinker(e.llmClient, "")
}

// RegisterTools 注册多个工具
func (e *Engine) RegisterTools(tools ...tool.Tool) {
	for _, t := range tools {
		e.toolManager.RegisterTool(t)
	}
	// 清空 Thinker 缓存以使用新的工具描述
	e.thinkerCache = make(map[string]core.Thinker)
	e.thinker = e.getOrCreateThinker(e.llmClient, "")
}

// Execute 执行任务
// ctx: 用于取消和超时控制的 context
// task: 要执行的任务描述
// execCtx: 执行上下文，用于存储执行过程中的数据
func (e *Engine) Execute(ctx context.Context, task string, execCtx *core.Context) *types.Result {
	// 如果没有提供 context，使用 background context
	if ctx == nil {
		ctx = context.Background()
	}

	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.metrics.RecordLatency("engine.execute", latency)
	}()

	// 初始化执行上下文
	if execCtx == nil {
		execCtx = core.NewContext()
	}
	execCtx.Set("task", task)

	// 记录任务开始
	e.logger.Info("Task execution started",
		log.String("task", task),
		log.Any("start_time", startTime),
	)

	// 1. Agent 选择（如果有 AgentManager）
	var selectedAgent *agent.Agent
	var systemPrompt string
	if e.agentManager != nil {
		if selResult, err := e.agentManager.SelectAgentWithResult(task); err == nil && selResult != nil {
			selectedAgent = selResult.Agent
			systemPrompt = selResult.Agent.SystemPrompt
			execCtx.Set("selected_agent", selResult.Agent.Name)
			execCtx.Set("system_prompt", systemPrompt)
			execCtx.Set("selection_method", string(selResult.Method))

			e.logger.Info("Agent selected",
				log.String("agent_name", selResult.Agent.Name),
				log.String("model_name", selResult.Agent.ModelName),
				log.String("description", selResult.Agent.Description),
				log.String("selection_method", string(selResult.Method)),
				log.Float64("selection_score", selResult.Score),
			)

			if selResult.Method == agent.SelectionFallback {
				e.logger.Warn("No matching agent found, using fallback",
					log.String("agent_name", selResult.Agent.Name),
					log.String("task", task),
				)
			}
		} else if err != nil {
			e.logger.Warn("Agent selection failed",
				log.Err(err),
			)
		}
	}

	// 2. Model 选择和 LLM Client 构建
	var llmClient llm.Client
	if selectedAgent != nil && selectedAgent.ModelName != "" && e.modelManager != nil {
		// 根据 Agent 指定的 Model 创建 LLM Client
		if client, err := e.modelManager.CreateLLMClient(selectedAgent.ModelName); err == nil {
			llmClient = client
			execCtx.Set("selected_model", selectedAgent.ModelName)

			e.logger.Info("LLM client created",
				log.String("model_name", selectedAgent.ModelName),
			)
		} else {
			// 如果创建失败，使用默认 LLM Client
			llmClient = e.llmClient

			e.logger.Warn("Failed to create LLM client, using default",
				log.String("model_name", selectedAgent.ModelName),
				log.Err(err),
			)
		}
	} else {
		// 使用默认 LLM Client
		llmClient = e.llmClient

		e.logger.Debug("Using default LLM client")
	}

	// 3. 重新创建 Thinker（使用选定的 LLM Client 和 System Prompt）
	currentThinker := e.thinker
	if llmClient != e.llmClient || systemPrompt != "" {
		// 使用缓存的 Thinker
		currentThinker = e.getOrCreateThinker(llmClient, systemPrompt)
	}

	// 4. Skill 选择和注入
	var selectedSkill *skill.Skill
	var skillStartTime time.Time
	originalTask := task

	if e.skillManager != nil {
		skillStartTime = time.Now()
		// 尝试选择合适的 Skill
		if sk, err := e.skillManager.SelectSkill(task); err == nil && sk != nil {
			selectedSkill = sk

			e.logger.Info("Skill selected",
				log.String("skill_name", sk.Name),
				log.String("description", sk.Description),
			)

			// 将 Skill 指令注入到任务中
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

			// 将 Skill 信息存入上下文
			execCtx.Set("selected_skill", selectedSkill.Name)
			execCtx.Set("skill_instructions", selectedSkill.Instructions)

			// 如果有脚本，也存入上下文供后续使用
			if len(selectedSkill.Scripts) > 0 {
				execCtx.Set("skill_scripts", selectedSkill.Scripts)
			}

			// 如果有参考文档，也存入上下文
			if len(selectedSkill.References) > 0 {
				execCtx.Set("skill_references", selectedSkill.References)
			}
		}
	}

	// 检查缓存
	if e.cache != nil {
		cacheKey := e.generateCacheKey(task)
		if cached, ok := e.cache.Get(cacheKey); ok {
			if result, ok := cached.(*types.Result); ok {
				// 返回缓存的结果（添加缓存标记）
				result.Metadata["cached"] = true
				result.Metadata["cache_hit_time"] = time.Now()

				e.logger.Info("Cache hit",
					log.String("cache_key", cacheKey),
				)

				return result
			}
		}
	}

	result := &types.Result{
		Trace:     make([]types.TraceStep, 0),
		Metadata:  make(map[string]interface{}),
		StartTime: startTime,
	}

	// 如果选中了 Skill，记录到结果元数据中
	if selectedSkill != nil {
		result.Metadata["selected_skill"] = selectedSkill.Name
		result.Metadata["skill_description"] = selectedSkill.Description
	}

	// 执行 ReAct 循环
	state := &types.LoopState{
		Iteration: 0,
		Task:      task,
	}

	for {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			result.Success = false
			result.Error = fmt.Errorf("execution cancelled: %w", ctx.Err())
			result.EndTime = time.Now()

			e.logger.Warn("Task execution cancelled",
				log.Int("iterations", state.Iteration),
				log.Err(ctx.Err()),
			)

			return result
		default:
		}

		state.Iteration++

		e.logger.Debug("Starting iteration",
			log.Int("iteration", state.Iteration),
		)

		// 添加轨迹：开始新的迭代
		e.addTrace(result, types.TraceStep{
			Step:      state.Iteration,
			Type:      "iteration_start",
			Content:   fmt.Sprintf("Starting iteration %d", state.Iteration),
			Timestamp: time.Now(),
		})

		// 1. 思考
		var thought *types.Thought
		var err error
		thinkStartTime := time.Now()

		// 记录思考前的资源使用
		resourceMonitor := metrics.NewResourceMonitor()
		beforeSnapshot := resourceMonitor.Snapshot()

		for retry := 0; retry <= e.maxRetries; retry++ {
			// 使用选定的 Thinker（可能包含 Agent 的 System Prompt）
			thought, err = currentThinker.Think(task, execCtx)
			if err == nil {
				thinkDuration := time.Since(thinkStartTime)
				e.logger.Debug("Thinking completed",
					log.Int("iteration", state.Iteration),
					log.Duration("duration", thinkDuration),
					log.String("reasoning", thought.Reasoning),
				)

				// 记录思考后的资源使用
				afterSnapshot := resourceMonitor.Snapshot()
				delta := afterSnapshot.Delta(beforeSnapshot)

				// 记录资源使用到 Metrics
				// CPU 使用率估算：基于 Goroutine 数量变化和 CPU 核心数
				cpuPercent := 0.0
				if delta.NumGoroutines > 0 {
					cpuPercent = float64(delta.NumGoroutines) / float64(afterSnapshot.NumCPU) * 100
				}

				e.metrics.RecordResourceUsage(
					"llm.think",
					cpuPercent,                  // CPU 使用率（估算）
					afterSnapshot.MemoryAllocMB, // 当前内存使用 (MB)
					0,                           // GPU 使用率（本地 LLM 可能需要）
					0,                           // GPU 内存使用 (MB)
				)

				e.logger.Debug("Resource usage recorded",
					log.Int("iteration", state.Iteration),
					log.Float64("memory_alloc_mb", afterSnapshot.MemoryAllocMB),
					log.Float64("memory_delta_mb", delta.MemoryAllocMB),
					log.Int("goroutines", afterSnapshot.NumGoroutines),
					log.Int("gc_count", delta.GCCount),
				)

				// 记录 Token 使用量（如果 LLM Client 支持）
				if tokenReporter, ok := llmClient.(llm.TokenReporter); ok {
					if usage := tokenReporter.LastTokenUsage(); usage != nil {
						e.metrics.RecordTokenUsage("llm.think", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
						e.logger.Debug("Token usage recorded",
							log.Int("iteration", state.Iteration),
							log.Int("prompt_tokens", usage.PromptTokens),
							log.Int("completion_tokens", usage.CompletionTokens),
							log.Int("total_tokens", usage.TotalTokens),
						)
					}
				}

				break
			}

			e.logger.Warn("Thinking failed",
				log.Int("iteration", state.Iteration),
				log.Int("retry", retry+1),
				log.Int("max_retries", e.maxRetries),
				log.Err(err),
			)

			// 添加轨迹：思考失败
			e.addTrace(result, types.TraceStep{
				Step:      state.Iteration,
				Type:      "think_error",
				Content:   fmt.Sprintf("Thinking failed: %v (Retry %d/%d)", err, retry+1, e.maxRetries),
				Timestamp: time.Now(),
			})

			// 如果达到最大重试次数，使用优雅降级
			if retry >= e.maxRetries {
				e.logger.Warn("Max retries reached, using graceful degradation",
					log.Int("iteration", state.Iteration),
				)

				// 添加轨迹：优雅降级
				e.addTrace(result, types.TraceStep{
					Step:      state.Iteration,
					Type:      "graceful_degradation",
					Content:   "LLM unavailable, using simplified mode",
					Timestamp: time.Now(),
				})

				// 简化模式：尝试直接使用工具
				thought = e.simplifiedMode(task)
				if thought == nil {
					result.Success = false
					result.Error = fmt.Errorf("LLM unavailable and no simplified mode available for task")
					result.EndTime = time.Now()
					// 记录错误指标
					e.metrics.RecordError("engine.execute", result.Error)

					e.logger.Error("Task execution failed",
						log.Err(result.Error),
						log.Int("iterations", state.Iteration),
					)

					return result
				}

				break
			}

			// 等待重试间隔
			time.Sleep(e.retryInterval)
		}
		state.LastThought = thought

		// 添加轨迹：思考
		e.addTrace(result, types.TraceStep{
			Step:      state.Iteration,
			Type:      "think",
			Content:   fmt.Sprintf("Reasoning: %s", thought.Reasoning),
			Timestamp: time.Now(),
		})

		// 检查是否应该结束
		if thought.ShouldFinish {
			result.Success = true
			result.Output = thought.FinalAnswer
			result.EndTime = time.Now()

			e.addTrace(result, types.TraceStep{
				Step:      state.Iteration,
				Type:      "finish",
				Content:   fmt.Sprintf("Task completed: %s", thought.FinalAnswer),
				Timestamp: time.Now(),
			})

			e.logger.Info("Task execution completed",
				log.Bool("success", true),
				log.Int("iterations", state.Iteration),
				log.Duration("total_duration", time.Since(startTime)),
				log.String("output", thought.FinalAnswer),
			)

			// 缓存成功的结果
			if e.cache != nil {
				cacheKey := e.generateCacheKey(state.Task)
				e.cache.Set(cacheKey, result, 0) // 使用默认 TTL
			}

			// 记录成功指标
			e.metrics.RecordSuccess("engine.execute")

			// 记录 Skill 执行统计
			if selectedSkill != nil && e.skillManager != nil {
				executionTime := time.Since(skillStartTime)
				// 简单的质量评分：成功完成给 1.0
				qualityScore := 1.0
				// 计算 token 消耗（简化处理，实际应该从 LLM 响应中获取）
				tokenConsumed := len(task) + len(thought.FinalAnswer)
				if err := e.skillManager.RecordExecution(selectedSkill.Name, true, executionTime, tokenConsumed, qualityScore); err != nil {
					e.logger.Warn("Failed to record skill execution",
						log.String("skill_name", selectedSkill.Name),
						log.Err(err),
					)
				}
			}

			return result
		}

		// 2. 行动（如果有动作）
		if thought.Action != nil {
			e.logger.Debug("Executing action",
				log.Int("iteration", state.Iteration),
				log.String("tool_name", thought.Action.ToolName),
				log.Any("parameters", thought.Action.Parameters),
			)

			// 添加轨迹：行动
			e.addTrace(result, types.TraceStep{
				Step:      state.Iteration,
				Type:      "act",
				Content:   fmt.Sprintf("Action: %s with params %v", thought.Action.ToolName, thought.Action.Parameters),
				Timestamp: time.Now(),
			})

			var execResult *types.ExecutionResult
			var err error
			actStartTime := time.Now()
			for retry := 0; retry <= e.maxRetries; retry++ {
				// 执行动作
				execResult, err = e.actor.Act(thought.Action, execCtx)
				if err == nil {
					actDuration := time.Since(actStartTime)
					e.logger.Debug("Action executed",
						log.Int("iteration", state.Iteration),
						log.String("tool_name", thought.Action.ToolName),
						log.Bool("success", execResult.Success),
						log.Duration("duration", actDuration),
					)
					break
				}

				e.logger.Warn("Action execution failed",
					log.Int("iteration", state.Iteration),
					log.String("tool_name", thought.Action.ToolName),
					log.Int("retry", retry+1),
					log.Int("max_retries", e.maxRetries),
					log.Err(err),
				)

				// 添加轨迹：行动失败
				e.addTrace(result, types.TraceStep{
					Step:      state.Iteration,
					Type:      "act_error",
					Content:   fmt.Sprintf("Action failed: %v (Retry %d/%d)", err, retry+1, e.maxRetries),
					Timestamp: time.Now(),
				})

				// 如果达到最大重试次数，返回错误
				if retry >= e.maxRetries {
					result.Success = false
					result.Error = fmt.Errorf("action failed after %d retries: %w", e.maxRetries, err)
					result.EndTime = time.Now()
					// 记录错误指标
					e.metrics.RecordError("engine.act", err)

					e.logger.Error("Action execution failed after max retries",
						log.Int("iteration", state.Iteration),
						log.String("tool_name", thought.Action.ToolName),
						log.Err(err),
					)

					return result
				}

				// 等待重试间隔
				time.Sleep(e.retryInterval)
			}
			state.LastResult = execResult

			// 添加轨迹：执行结果
			e.addTrace(result, types.TraceStep{
				Step:      state.Iteration,
				Type:      "result",
				Content:   fmt.Sprintf("Result: %v (Success: %v)", execResult.Output, execResult.Success),
				Timestamp: time.Now(),
			})

			// 3. 观察
			var feedback *types.Feedback
			var observeErr error
			for retry := 0; retry <= e.maxRetries; retry++ {
				feedback, observeErr = e.observer.Observe(execResult, execCtx)
				if observeErr == nil {
					break
				}

				// 添加轨迹：观察失败
				e.addTrace(result, types.TraceStep{
					Step:      state.Iteration,
					Type:      "observe_error",
					Content:   fmt.Sprintf("Observation failed: %v (Retry %d/%d)", observeErr, retry+1, e.maxRetries),
					Timestamp: time.Now(),
				})

				// 如果达到最大重试次数，返回错误
				if retry >= e.maxRetries {
					result.Success = false
					result.Error = fmt.Errorf("observation failed after %d retries: %w", e.maxRetries, observeErr)
					result.EndTime = time.Now()
					// 记录错误指标
					e.metrics.RecordError("engine.observe", observeErr)
					return result
				}

				// 等待重试间隔
				time.Sleep(e.retryInterval)
			}
			state.LastFeedback = feedback

			// 添加轨迹：观察
			e.addTrace(result, types.TraceStep{
				Step:      state.Iteration,
				Type:      "observe",
				Content:   fmt.Sprintf("Feedback: %s", feedback.Message),
				Timestamp: time.Now(),
			})

			// 更新 Context 中的上下文信息（累积历史记录）
			execCtx.Set("last_action", thought.Action)
			execCtx.Set("last_result", execResult)
			execCtx.Set("last_feedback", feedback)

			// 追加到历史步骤列表
			step := map[string]string{
				"action":   fmt.Sprintf("%s with params %v", thought.Action.ToolName, thought.Action.Parameters),
				"result":   fmt.Sprintf("%v (Success: %v)", execResult.Output, execResult.Success),
				"feedback": feedback.Message,
			}

			var steps []map[string]string
			if existing, ok := execCtx.Get("history_steps"); ok {
				if s, ok := existing.([]map[string]string); ok {
					steps = s
				}
			}
			steps = append(steps, step)
			execCtx.Set("history_steps", steps)
		}

		// 4. 循环控制
		action := e.loopController.Control(state)
		if !action.ShouldContinue {
			// 如果是因为达到最大迭代次数而停止，标记为失败
			if state.Iteration >= DefaultMaxIterations {
				result.Success = false
				result.Error = fmt.Errorf("max iterations reached without completion")
				// 记录错误指标
				e.metrics.RecordError("engine.loop", result.Error)
			} else {
				result.Success = true
				// 记录成功指标
				e.metrics.RecordSuccess("engine.execute")
			}

			result.Output = action.Reason
			result.EndTime = time.Now()

			e.addTrace(result, types.TraceStep{
				Step:      state.Iteration,
				Type:      "stop",
				Content:   fmt.Sprintf("Stopping: %s", action.Reason),
				Timestamp: time.Now(),
			})

			// 缓存结果
			if e.cache != nil {
				cacheKey := e.generateCacheKey(state.Task)
				e.cache.Set(cacheKey, result, 0) // 使用默认 TTL
			}

			// 记录 Skill 执行统计（失败或达到最大迭代次数）
			if selectedSkill != nil && e.skillManager != nil {
				executionTime := time.Since(skillStartTime)
				success := result.Success
				// 失败的质量评分较低
				qualityScore := 0.3
				if success {
					qualityScore = 0.7 // 部分成功
				}
				tokenConsumed := len(task) + len(action.Reason)
				if err := e.skillManager.RecordExecution(selectedSkill.Name, success, executionTime, tokenConsumed, qualityScore); err != nil {
					e.logger.Warn("Failed to record skill execution",
						log.String("skill_name", selectedSkill.Name),
						log.Err(err),
					)
				}
			}

			return result
		}
	}
}

// generateCacheKey 生成缓存键
func (e *Engine) generateCacheKey(task string) string {
	// 使用任务和工具描述生成缓存键
	// 注意：不包含 LLM Client 指针，以允许跨 Engine 实例共享缓存
	// 这是一个权衡：提高缓存命中率 vs 避免配置差异导致的问题
	data := fmt.Sprintf("task:%s|tools:%s",
		task,
		e.toolManager.GetToolDescriptions(),
	)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// getOrCreateThinker 获取或创建 Thinker（带缓存）
func (e *Engine) getOrCreateThinker(llmClient llm.Client, systemPrompt string) core.Thinker {
	// 生成缓存键：llmClient 指针 + systemPrompt
	cacheKey := fmt.Sprintf("%p:%s", llmClient, systemPrompt)

	// 检查缓存
	if cached, ok := e.thinkerCache[cacheKey]; ok {
		return cached
	}

	// 创建新的 Thinker
	var newThinker core.Thinker
	toolDesc := e.toolManager.GetToolDescriptions()

	if systemPrompt != "" {
		newThinker = thinker.NewSimpleThinkerWithSystemPrompt(llmClient, toolDesc, systemPrompt)
	} else {
		newThinker = thinker.NewSimpleThinker(llmClient, toolDesc)
	}

	// 存入缓存
	e.thinkerCache[cacheKey] = newThinker
	return newThinker
}

// addTrace 添加 Trace 条目，并在超过限制时截断旧条目
func (e *Engine) addTrace(result *types.Result, step types.TraceStep) {
	result.Trace = append(result.Trace, step)

	// 如果设置了最大 Trace 大小且超过限制，移除最旧的条目
	if e.maxTraceSize > 0 && len(result.Trace) > e.maxTraceSize {
		// 保留最新的 maxTraceSize 条记录
		result.Trace = result.Trace[len(result.Trace)-e.maxTraceSize:]
	}
}

// simplifiedMode 简化模式，当LLM不可用时使用
func (e *Engine) simplifiedMode(task string) *types.Thought {
	// 简单的任务分析逻辑
	// 检查是否是计算任务
	if strings.Contains(strings.ToLower(task), "calculate") || strings.Contains(strings.ToLower(task), "+") ||
		strings.Contains(strings.ToLower(task), "-") || strings.Contains(strings.ToLower(task), "*") ||
		strings.Contains(strings.ToLower(task), "/") {
		// 尝试使用计算器工具
		return &types.Thought{
			Reasoning: "LLM unavailable, using simplified mode for calculation",
			Action: &types.Action{
				ToolName:   "calculator",
				Parameters: map[string]interface{}{"expression": task},
				Reasoning:  "Task appears to be a calculation",
			},
			ShouldFinish: false,
		}
	}

	// 检查是否是时间任务
	if strings.Contains(strings.ToLower(task), "time") || strings.Contains(strings.ToLower(task), "date") {
		// 尝试使用DateTime工具
		return &types.Thought{
			Reasoning: "LLM unavailable, using simplified mode for time query",
			Action: &types.Action{
				ToolName:   "datetime",
				Parameters: map[string]interface{}{"operation": "now"},
				Reasoning:  "Task appears to be a time/date query",
			},
			ShouldFinish: false,
		}
	}

	// 检查是否是HTTP请求
	if strings.Contains(strings.ToLower(task), "http") || strings.Contains(strings.ToLower(task), "api") ||
		strings.Contains(strings.ToLower(task), "url") || strings.Contains(strings.ToLower(task), "request") {
		// 尝试提取URL并使用HTTP工具
		// 简单的URL提取逻辑
		urlRegex := regexp.MustCompile(`https?://[^\s]+`)
		matches := urlRegex.FindStringSubmatch(task)
		if len(matches) > 0 {
			return &types.Thought{
				Reasoning: "LLM unavailable, using simplified mode for HTTP request",
				Action: &types.Action{
					ToolName:   "http",
					Parameters: map[string]interface{}{"method": "GET", "url": matches[0]},
					Reasoning:  "Task appears to be an HTTP request",
				},
				ShouldFinish: false,
			}
		}
	}

	// 对于其他任务，返回一个简单的echo响应
	return &types.Thought{
		Reasoning: "LLM unavailable, using echo as fallback",
		Action: &types.Action{
			ToolName:   "echo",
			Parameters: map[string]interface{}{"message": DefaultFallbackMessage},
			Reasoning:  "No specific tool identified for task",
		},
		ShouldFinish: false,
	}
}

// Close 优雅关闭引擎，释放所有资源
func (e *Engine) Close() error {
	e.logger.Info("Shutting down engine")

	var errs []error

	// 关闭缓存
	if e.cache != nil {
		if err := e.cache.Close(); err != nil {
			e.logger.Error("Failed to close cache", log.Err(err))
			errs = append(errs, fmt.Errorf("cache close error: %w", err))
		}
	}

	// 关闭 metrics
	if e.metrics != nil {
		if err := e.metrics.Close(); err != nil {
			e.logger.Error("Failed to close metrics", log.Err(err))
			errs = append(errs, fmt.Errorf("metrics close error: %w", err))
		}
	}

	e.logger.Info("Engine shutdown complete")

	// 如果有多个错误，返回第一个
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}
