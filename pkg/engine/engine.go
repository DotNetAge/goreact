package engine

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ray/goreact/pkg/agent"
	"github.com/ray/goreact/pkg/cache"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/llm/mock"
	"github.com/ray/goreact/pkg/memory"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/model"
	"github.com/ray/goreact/pkg/prompt"
	"github.com/ray/goreact/pkg/skill"
	"github.com/ray/goreact/pkg/tool"
	"github.com/ray/goreact/pkg/types"
)

// Engine ReAct 引擎核心
type Engine struct {
	toolManager    *tool.Manager
	modelManager   model.ModelManager
	agentManager   *agent.Manager
	skillManager   *skill.DefaultManager
	memoryManager  memory.MemoryManager
	promptManager  prompt.PromptManager
	thinker        core.Thinker
	actor          core.Actor
	observer       core.Observer
	loopController core.LoopController
	llmClient      llm.Client
	cache          cache.Cache
	maxRetries     int
	retryInterval  time.Duration
	metrics        metrics.Metrics
}

// New 创建新的引擎实例
func New(options ...Option) *Engine {
	// 创建默认组件
	toolManager := tool.NewManager()
	modelManager := model.NewDefaultModelManager()
	agentManager := agent.NewManager()
	skillManager := skill.NewDefaultManager()
	memoryManager := memory.NewDefaultMemoryManager("")
	promptManager := prompt.NewDefaultPromptManager()
	llmClient := mock.NewMockClient([]string{})

	engine := &Engine{
		toolManager:    toolManager,
		modelManager:   modelManager,
		agentManager:   agentManager,
		skillManager:   skillManager,
		memoryManager:  memoryManager,
		promptManager:  promptManager,
		llmClient:      llmClient,
		loopController: core.NewDefaultLoopController(10), // 默认最大 10 次迭代
		maxRetries:     3,                                 // 默认最多重试 3 次
		retryInterval:  1 * time.Second,                   // 默认重试间隔 1 秒
		metrics:        metrics.NewDefaultMetrics(),       // 默认指标收集器
	}

	// 应用选项
	for _, opt := range options {
		opt(engine)
	}

	// 如果没有设置核心模块，使用默认实现
	if engine.thinker == nil {
		engine.thinker = core.NewDefaultThinker(engine.llmClient, engine.toolManager.GetToolDescriptions(), engine.promptManager, engine.memoryManager)
	}
	if engine.actor == nil {
		engine.actor = core.NewDefaultActor(engine.toolManager)
	}
	if engine.observer == nil {
		engine.observer = core.NewDefaultObserver()
	}

	return engine
}

// RegisterTool 注册单个工具
func (e *Engine) RegisterTool(t tool.Tool) {
	e.toolManager.RegisterTool(t)
	// 更新 Thinker 的工具描述
	e.thinker = core.NewDefaultThinker(e.llmClient, e.toolManager.GetToolDescriptions(), e.promptManager, e.memoryManager)
}

// RegisterTools 注册多个工具
func (e *Engine) RegisterTools(tools ...tool.Tool) {
	for _, t := range tools {
		e.RegisterTool(t)
	}
}

// RegisterModel 注册单个模型
func (e *Engine) RegisterModel(m model.Model) {
	e.modelManager.RegisterModel(m)
}

// RegisterModels 注册多个模型
func (e *Engine) RegisterModels(models ...model.Model) {
	for _, m := range models {
		e.RegisterModel(m)
	}
}

// RegisterAgent 注册单个智能体
func (e *Engine) RegisterAgent(a *agent.Agent) {
	e.agentManager.Register(a)
}

// RegisterAgents 注册多个智能体
func (e *Engine) RegisterAgents(agents ...*agent.Agent) {
	for _, a := range agents {
		e.RegisterAgent(a)
	}
}

// RegisterSkill 注册单个技能
func (e *Engine) RegisterSkill(s *skill.Skill) {
	e.skillManager.RegisterSkill(s)
}

// RegisterSkills 注册多个技能
func (e *Engine) RegisterSkills(skills ...*skill.Skill) {
	for _, s := range skills {
		e.RegisterSkill(s)
	}
}

// Execute 执行任务
func (e *Engine) Execute(task string, ctx *core.Context) *types.Result {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.metrics.RecordLatency("engine.execute", latency)
	}()

	// 初始化上下文
	if ctx == nil {
		ctx = core.NewContext()
	}
	ctx.Set("task", task)

	// 检查缓存
	if e.cache != nil {
		cacheKey := e.generateCacheKey(task)
		if cached, ok := e.cache.Get(cacheKey); ok {
			if result, ok := cached.(*types.Result); ok {
				// 返回缓存的结果（添加缓存标记）
				result.Metadata["cached"] = true
				result.Metadata["cache_hit_time"] = time.Now()
				return result
			}
		}
	}

	result := &types.Result{
		Trace:     make([]types.TraceStep, 0),
		Metadata:  make(map[string]interface{}),
		StartTime: startTime,
	}

	// 执行 ReAct 循环
	state := &types.LoopState{
		Iteration: 0,
		Task:      task,
	}

	for {
		state.Iteration++

		// 添加轨迹：开始新的迭代
		result.Trace = append(result.Trace, types.TraceStep{
			Step:      state.Iteration,
			Type:      "iteration_start",
			Content:   fmt.Sprintf("Starting iteration %d", state.Iteration),
			Timestamp: time.Now(),
		})

		// 1. 思考
		var thought *types.Thought
		var err error
		for retry := 0; retry <= e.maxRetries; retry++ {
			// 尝试使用模型管理器选择合适的模型
			// 这里简化处理，实际应该根据任务需求选择模型
			thought, err = e.thinker.Think(task, ctx)
			if err == nil {
				break
			}

			// 添加轨迹：思考失败
			result.Trace = append(result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "think_error",
				Content:   fmt.Sprintf("Thinking failed: %v (Retry %d/%d)", err, retry+1, e.maxRetries),
				Timestamp: time.Now(),
			})

			// 如果达到最大重试次数，使用优雅降级
			if retry >= e.maxRetries {
				// 添加轨迹：优雅降级
				result.Trace = append(result.Trace, types.TraceStep{
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
					return result
				}

				break
			}

			// 等待重试间隔
			time.Sleep(e.retryInterval)
		}
		state.LastThought = thought

		// 添加轨迹：思考
		result.Trace = append(result.Trace, types.TraceStep{
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

			result.Trace = append(result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "finish",
				Content:   fmt.Sprintf("Task completed: %s", thought.FinalAnswer),
				Timestamp: time.Now(),
			})

			// 缓存成功的结果
			if e.cache != nil {
				cacheKey := e.generateCacheKey(state.Task)
				e.cache.Set(cacheKey, result, 0) // 使用默认 TTL
			}

			// 记录成功指标
			e.metrics.RecordSuccess("engine.execute")

			return result
		}

		// 2. 行动（如果有动作）
		if thought.Action != nil {
			// 添加轨迹：行动
			result.Trace = append(result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "act",
				Content:   fmt.Sprintf("Action: %s with params %v", thought.Action.ToolName, thought.Action.Parameters),
				Timestamp: time.Now(),
			})

			var execResult *types.ExecutionResult
			var err error
			for retry := 0; retry <= e.maxRetries; retry++ {
				// 执行动作
				execResult, err = e.actor.Act(thought.Action, ctx)
				if err == nil {
					break
				}

				// 添加轨迹：行动失败
				result.Trace = append(result.Trace, types.TraceStep{
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
					return result
				}

				// 等待重试间隔
				time.Sleep(e.retryInterval)
			}
			state.LastResult = execResult

			// 添加轨迹：执行结果
			result.Trace = append(result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "result",
				Content:   fmt.Sprintf("Result: %v (Success: %v)", execResult.Output, execResult.Success),
				Timestamp: time.Now(),
			})

			// 3. 观察
			var feedback *types.Feedback
			var observeErr error
			for retry := 0; retry <= e.maxRetries; retry++ {
				feedback, observeErr = e.observer.Observe(execResult, ctx)
				if observeErr == nil {
					break
				}

				// 添加轨迹：观察失败
				result.Trace = append(result.Trace, types.TraceStep{
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
			result.Trace = append(result.Trace, types.TraceStep{
				Step:      state.Iteration,
				Type:      "observe",
				Content:   fmt.Sprintf("Feedback: %s", feedback.Message),
				Timestamp: time.Now(),
			})

			// 更新内存管理器中的上下文信息
			e.memoryManager.Store("default", "last_action", thought.Action)
			e.memoryManager.Store("default", "last_result", execResult)
			e.memoryManager.Store("default", "last_feedback", feedback)
		}

		// 4. 循环控制
		action := e.loopController.Control(state)
		if !action.ShouldContinue {
			// 如果是因为达到最大迭代次数而停止，标记为失败
			if state.Iteration >= 10 {
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

			result.Trace = append(result.Trace, types.TraceStep{
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

			return result
		}
	}
}

// generateCacheKey 生成缓存键
func (e *Engine) generateCacheKey(task string) string {
	// 使用任务和工具描述生成缓存键
	data := task + e.toolManager.GetToolDescriptions()
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
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
			Parameters: map[string]interface{}{"message": "LLM unavailable, please try again later"},
			Reasoning:  "No specific tool identified for task",
		},
		ShouldFinish: false,
	}
}
