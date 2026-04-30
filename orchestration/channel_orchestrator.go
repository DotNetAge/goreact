package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/DotNetAge/goreact"
	"github.com/DotNetAge/goreact/core"
)

// ChannelOrchestrator is the default Orchestrator implementation using Go channels
// as the Actor-model message inbox. It implements all four roles:
//   - 编排引擎: Actor loop processing Delegate/Query/Cancel/Result messages
//   - Agent 工厂: GetAgent with Registry cache + Model resolution + dynamic creation (§12)
//   - 事件聚合器: Unified event stream from all agents
//   - Model 分配器: Automatic model lookup per agent config
//
// 智能路由能力（Design §6.3）：
//
//	当 LLM Router 可用时，支持语义匹配任务→Agent 的自动路由。
//	当 AgentFactory 可用时，支持按需动态创建新 Agent。
//	当 ScoreTracker 可用时，支持基于历史绩效的优选排序。
type ChannelOrchestrator struct {
	// === Configuration (immutable after construction) ===
	modelRegistry  core.ModelRegistry
	registry       *goreact.AgentRegistry
	spawnFunc      SpawnFunction
	maxConcurrent  int
	defaultTimeout time.Duration
	inboxSize      int
	config         OrchestratorConfig // P2-1: centralized config for external exposure

	// === Intelligent Routing Components (Design §6 / §8 / §12) ===
	router       Router        // 智能路由引擎 (nil = 降级为关键词匹配)
	factory      *AgentFactory // 动态 Agent 创建工厂 (nil = 不支持动态创建)
	scoreTracker *ScoreTracker // 绩效追踪器 (nil = 不记录绩效)

	// === Agent Factory Cache ===
	agentCache   map[string]*goreact.Agent // name -> cached instance
	agentCacheMu sync.RWMutex

	// === Runtime State Tracking ===
	runtimeDir *core.RuntimeDirectory // Agent runtime state (idle/busy/coordinating/error, scores)

	// === Task Management ===
	store TaskStore

	// === Channel Actor Loop (P0-2: per-agent inbox architecture) ===
	agentInboxes   map[string]chan Message // agentName → buffered inbox (Design §7.2)
	agentInboxesMu sync.RWMutex            // Protects agentInboxes map
	controlCh      chan Message            // Global control messages (delegate/query/cancel/broadcast)
	done           chan struct{}           // Closed when Stop() completes
	started        bool
	startOnce      sync.Once
	state          OrchestratorState // P2-4: full state machine (Initializing/Running/Draining/Stopped)
	stateMu        sync.RWMutex

	// === Event Aggregation ===
	eventOut    chan core.ReactEvent
	eventSubsMu sync.RWMutex
	eventSubs   map[chan<- core.ReactEvent]func(core.ReactEvent) bool
	eventDone   chan struct{} // Closed when Stop() completes

	// logger
	logger *slog.Logger

	// === P1-5: Heartbeat tracking ===
	heartbeatMu    sync.RWMutex
	heartbeats     map[string]time.Time // agentName → last heartbeat timestamp
	heartbeatCheck *time.Ticker         // Periodic heartbeat checker

	// === P1-4: Idle agent cleanup ===
	idleCleanupConfig IdleCleanupConfig
	idleCleanupTicker *time.Ticker

	// === CoordinatorPool — manages active Coordinator instances ===
	coordinators   map[string]*Coordinator
	coordinatorsMu sync.RWMutex
}

// New creates a new ChannelOrchestrator with the given options.
// Call Start(ctx) to begin processing messages.
func New(opts ...OrchestratorOption) (*ChannelOrchestrator, error) {
	setup := &orchestratorSetup{}
	for _, opt := range opts {
		opt(setup)
	}

	o := &ChannelOrchestrator{
		modelRegistry:     setup.modelRegistry,
		registry:          setup.registry,
		spawnFunc:         setup.spawnFunc,
		maxConcurrent:     setup.maxConcurrent,
		inboxSize:         setup.inboxSize,
		router:            setup.llmRouter,
		factory:           setup.agentFactory,
		scoreTracker:      setup.scoreTracker,
		agentCache:        make(map[string]*goreact.Agent),
		agentInboxes:      make(map[string]chan Message),
		controlCh:         make(chan Message, setup.inboxSize),
		runtimeDir:        core.NewRuntimeDirectory(0), // unlimited
		store:             NewInMemoryTaskStore(),
		done:              make(chan struct{}),
		eventOut:          make(chan core.ReactEvent, 256),
		eventSubs:         make(map[chan<- core.ReactEvent]func(core.ReactEvent) bool),
		eventDone:         make(chan struct{}),
		logger:            slog.Default(),
		heartbeats:        make(map[string]time.Time),
		idleCleanupConfig: IdleCleanupConfig{},
		coordinators:      make(map[string]*Coordinator),
	}

	// Resolve default model if provided via WithDefaultModel
	if o.modelRegistry == nil && setup.defaultModel != nil {
		o.modelRegistry = core.NewInMemoryModelRegistry()
		if err := o.modelRegistry.Register("default", setup.defaultModel); err != nil {
			return nil, fmt.Errorf("failed to register default model: %w", err)
		}
	}

	// Set defaults
	if o.inboxSize <= 0 {
		o.inboxSize = 256
	}
	if o.maxConcurrent <= 0 {
		o.maxConcurrent = 0 // unlimited
	}
	if o.defaultTimeout <= 0 {
		o.defaultTimeout = 5 * time.Minute
	}
	if o.spawnFunc == nil {
		o.spawnFunc = defaultSpawnFunction(o)
	}

	// Auto-create LLMRouter when a ModelConfig is available but no Router was explicitly injected.
	// This enables RouteTask() intelligent routing out-of-the-box with just WithDefaultModel().
	if o.router == nil && setup.defaultModel != nil && setup.defaultModel.APIKey != "" {
		router, err := NewLLMRouter(setup.defaultModel)
		if err != nil {
			return nil, fmt.Errorf("failed to create LLM router from default model: %w", err)
		}
		o.router = router

		// Also log that the router was auto-created
		o.logger.Info("LLM Router auto-created from default model",
			"model", setup.defaultModel.Name,
		)
	}

	// If Router exists but no Factory, create a default Factory backed by the AgentRegistry.
	if o.router != nil && o.factory == nil {
		// router is always *LLMRouter at this point
		llmRouter, _ := o.router.(*LLMRouter)
		var adapter goreactRegistryAdapter = &registryAdapterImpl{reg: o.registry}
		o.factory = NewAgentFactory(llmRouter, &adapter)
	}

	// Always create a ScoreTracker if none provided
	if o.scoreTracker == nil {
		o.scoreTracker = NewScoreTracker()
	}

	return o, nil
}

// defaultSpawnFunction creates a sub-agent using goreact.NewAgent internally.
// This is the default SpawnFunction used when none is provided via WithSpawnFunction.
func defaultSpawnFunction(orch *ChannelOrchestrator) SpawnFunction {
	return func(
		ctx context.Context,
		agentConfig *core.AgentConfig,
		modelConfig *core.ModelConfig,
		taskPrompt string,
		taskID string,
		resultCh chan<- any,
	) error {
		// Build the sub-agent using goreact.NewAgent with resolved config and model
		agent, err := goreact.NewAgent(
			goreact.WithConfig(agentConfig),
			goreact.WithModel(modelConfig),
		)
		if err != nil {
			return fmt.Errorf("failed to create sub-agent for task %q: %w", taskID, err)
		}

		// Run the T-A-O loop in a goroutine and send result to channel
		go func() {
			result, runErr := agent.Ask(taskID, taskPrompt)
			if runErr != nil {
				select {
				case resultCh <- fmt.Errorf("task %q failed: %w", taskID, runErr):
				case <-ctx.Done():
				}
				return
			}
			select {
			case resultCh <- result.Answer:
			case <-ctx.Done():
			}
		}()

		return nil
	}
}

// --- Lifecycle ---

func (o *ChannelOrchestrator) Start(ctx context.Context) error {
	var startErr error
	o.startOnce.Do(func() {
		o.setState(OrchestratorInitializing)
		o.started = true

		// Start the Actor loop goroutine
		go o.runLoop(ctx)

		// Start event dispatcher goroutine
		go o.runEventDispatcher(ctx)

		o.setState(OrchestratorRunning)
		o.logger.Info("orchestrator started",
			"max_concurrent", o.maxConcurrent,
			"default_timeout", o.defaultTimeout,
		)
	})
	return startErr
}

func (o *ChannelOrchestrator) Stop(ctx context.Context) error {
	if !o.started {
		return nil
	}

	// P2-4: Graceful drain if enabled
	if o.config.EnableGracefulDrain {
		o.setState(OrchestratorDraining)
		o.logger.Info("orchestrator entering draining state")
	}

	// Close all agent inboxes to signal per-agent goroutines to exit
	o.agentInboxesMu.Lock()
	for name, ch := range o.agentInboxes {
		close(ch)
		delete(o.agentInboxes, name)
	}
	o.agentInboxesMu.Unlock()

	// Close control channel to signal runLoop to drain and exit
	close(o.controlCh)

	// Wait for runLoop to finish
	select {
	case <-o.done:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Close event aggregator
	close(o.eventDone)

	o.setState(OrchestratorStopped)
	o.started = false
	o.logger.Info("orchestrator stopped")
	return nil
}

// setState transitions the orchestrator to a new state (thread-safe, P2-4).
func (o *ChannelOrchestrator) setState(s OrchestratorState) {
	o.stateMu.Lock()
	defer o.stateMu.Unlock()
	o.state = s
}

// State returns the current orchestrator lifecycle state (P2-4).
func (o *ChannelOrchestrator) State() OrchestratorState {
	o.stateMu.RLock()
	defer o.stateMu.RUnlock()
	return o.state
}

// GetConfig returns the centralized configuration for external access (P2-1).
func (o *ChannelOrchestrator) GetConfig() OrchestratorConfig {
	return o.config
}

// --- Actor Loop ---

func (o *ChannelOrchestrator) runLoop(ctx context.Context) {
	defer close(o.done)
	for {
		select {
		case msg, ok := <-o.controlCh:
			if !ok {
				return // Control channel closed, shutdown
			}
			o.handleMessage(ctx, msg)

		case <-ctx.Done():
			o.logger.Info("orchestration runLoop: context cancelled", "error", ctx.Err())
			return
		}
	}
}

func (o *ChannelOrchestrator) handleMessage(ctx context.Context, msg Message) {
	switch msg.Type {
	case MsgDelegate:
		resp := o.handleDelegate(ctx, msg)
		o.reply(msg, resp)
	case MsgQuery:
		resp := o.handleQuery(msg)
		o.reply(msg, resp)
	case MsgCancel:
		resp := o.handleCancel(msg)
		o.reply(msg, resp)
	case MsgResult:
		o.handleResult(ctx, msg)
		// No reply needed for async results
	case MsgBroadcast:
		// Reserved for future use
		o.logger.Debug("MsgBroadcast received but not yet implemented")
	default:
		o.reply(msg, Response{Error: fmt.Errorf("unknown message type: %s", msg.Type)})
	}
}

func (o *ChannelOrchestrator) reply(msg Message, resp Response) {
	if msg.ReplyCh != nil {
		select {
		case msg.ReplyCh <- resp:
		default:
			// Reply channel full or closed — caller may have timed out
		}
	}
}

// --- Message Handlers ---

func (o *ChannelOrchestrator) handleDelegate(ctx context.Context, msg Message) Response {
	req, ok := msg.Payload.(DelegateRequest)
	if !ok {
		return Response{Error: fmt.Errorf("invalid delegate request payload")}
	}

	// Concurrency control: if max > 0 and active >= max, block or reject
	if o.maxConcurrent > 0 {
		active := o.store.ActiveTasks()
		if active >= o.maxConcurrent {
			return Response{Error: fmt.Errorf("max concurrent tasks (%d) reached, %d active", o.maxConcurrent, active)}
		}
	}

	// Create task record
	parentID := req.ParentID
	if parentID == "" {
		parentID = "root"
	}
	task, err := o.store.CreateTask(parentID, req.AgentName+": "+req.TaskPrompt, "")
	if err != nil {
		return Response{Error: fmt.Errorf("failed to create task: %w", err)}
	}

	// Create result channel
	resultCh := make(chan any, 1)
	o.store.SetResultCh(task.ID, resultCh)

	// Update status to InProgress
	_ = o.store.UpdateTaskStatus(task.ID, core.TaskStatusInProgress, "", "")

	// Emit SubtaskSpawned event
	o.emitEvent(core.ReactEvent{
		Type: core.SubtaskSpawned,
		Data: core.SubtaskInfo{TaskID: task.ID, AgentName: req.AgentName, Description: req.TaskPrompt},
	})

	// Launch sub-agent asynchronously
	go func() {
		// Ensure resultCh always receives something to prevent blocking callers
		defer func() {
			if r := recover(); r != nil {
				resultCh <- fmt.Errorf("sub-agent panic: %v", r)
				_ = o.store.UpdateTaskStatus(task.ID, core.TaskStatusFailed, "", fmt.Sprintf("panic: %v", r))
				o.emitEvent(core.ReactEvent{
					Type: core.SubtaskCompleted,
					Data: core.SubtaskResult{TaskID: task.ID, Success: false, Error: fmt.Sprintf("panic: %v", r)},
				})
			}
		}()

		// Look up agent config
		config := o.registry.Get(req.AgentName)
		if config == nil {
			resultCh <- fmt.Errorf("agent %q not found in registry", req.AgentName)
			_ = o.store.UpdateTaskStatus(task.ID, core.TaskStatusFailed, "", "agent not found")
			o.emitEvent(core.ReactEvent{
				Type: core.SubtaskCompleted,
				Data: core.SubtaskResult{TaskID: task.ID, Success: false, Error: "agent not found"},
			})
			return
		}

		// Resolve model
		modelName := config.Model
		if modelName == "" {
			modelName = "default"
		}
		modelCfg, err := o.modelRegistry.Get(modelName)
		if err != nil {
			resultCh <- fmt.Errorf("model %q not found for agent %q: %v", modelName, req.AgentName, err)
			_ = o.store.UpdateTaskStatus(task.ID, core.TaskStatusFailed, "", err.Error())
			return
		}

		// Call spawn function
		spawnErr := o.spawnFunc(ctx, config, modelCfg, req.TaskPrompt, task.ID, resultCh)
		if spawnErr != nil {
			_ = o.store.UpdateTaskStatus(task.ID, core.TaskStatusFailed, "", spawnErr.Error())
			o.emitEvent(core.ReactEvent{
				Type: core.SubtaskCompleted,
				Data: core.SubtaskResult{TaskID: task.ID, Success: false, Error: spawnErr.Error()},
			})
		}
	}()

	return Response{Data: &DelegateResult{TaskID: task.ID, ResultCh: resultCh}}
}

func (o *ChannelOrchestrator) handleQuery(msg Message) Response {
	// Look up task by ID from payload or from msg.TaskID
	taskID := msg.TaskID
	if taskID == "" {
		// List mode
		tasks, err := o.store.ListAllTasks()
		if err != nil {
			return Response{Error: err}
		}
		return Response{Data: tasks}
	}

	task, err := o.store.GetTask(taskID)
	if err != nil {
		return Response{Error: err}
	}
	return Response{Data: task}
}

func (o *ChannelOrchestrator) handleCancel(msg Message) Response {
	err := o.store.CancelTask(msg.TaskID)
	if err != nil {
		return Response{Error: err}
	}
	return Response{Data: "cancelled"}
}

func (o *ChannelOrchestrator) handleResult(ctx context.Context, msg Message) {
	// Async result from sub-agent — update store and emit event
	task, err := o.store.GetTask(msg.TaskID)
	if err != nil {
		o.logger.Error("handleResult: task not found", "task_id", msg.TaskID, "error", err)
		// Even if task not found, reply to sender to prevent blocking
		if msg.ReplyCh != nil {
			msg.ReplyCh <- Response{Error: fmt.Errorf("task not found: %s", msg.TaskID)}
		}
		return
	}

	// Determine success from payload
	success := true
	var answer, errMsg string
	if str, ok := msg.Payload.(string); ok {
		answer = str
	} else if e, ok := msg.Payload.(error); ok {
		errMsg = e.Error()
		success = false
	}

	status := core.TaskStatusCompleted
	if !success {
		status = core.TaskStatusFailed
	}
	_ = o.store.UpdateTaskStatus(task.ID, status, answer, errMsg)

	// Record performance score (Design §8.3/§8.5)
	o.recordScore(task, success)

	// Emit completion event
	o.emitEvent(core.ReactEvent{
		Type: core.SubtaskCompleted,
		Data: core.SubtaskResult{
			TaskID:  task.ID,
			Success: success,
			Answer:  answer,
			Error:   errMsg,
		},
	})

	// Reply to sender via ReplyCh to prevent blocking (data chain integrity)
	if msg.ReplyCh != nil {
		select {
		case msg.ReplyCh <- Response{Data: answer}:
		default:
			o.logger.Warn("handleResult: reply channel full or closed", "task_id", task.ID)
		}
	}
}

// recordScore 根据任务结果计算并记录绩效分数。
// 采用 Design §8.1 的 0-3 分制 + §8.3 的混合评分策略（70% 客观 + 30% 主观）。
func (o *ChannelOrchestrator) recordScore(task *core.Task, success bool) {
	if o.scoreTracker == nil || task == nil {
		return
	}

	score := 0
	if success {
		// 自动客观分 (Design §8.3):
		//   成功完成 → 基础分 2 (ScoreSuccess)
		//   如果有答案内容 → 满分 3 (ScorePerfect)
		score = ScoreSuccess
		if task.Output != "" && len(task.Output) > 20 { // 有实质输出
			score = ScorePerfect
		}
	} else {
		score = ScoreFailed
	}

	// 从任务描述中提取 Agent 名称（task.Description 格式为 "agentName: prompt"）
	agentID := extractAgentNameFromTask(task)

	if agentID != "" {
		o.scoreTracker.RecordScore(agentID, score, success, task.ID)

		// 同步更新 RuntimeDirectory 的 Score 字段
		o.runtimeDir.SetScore(agentID, float64(score))

		// 更新 TaskCount
		o.runtimeDir.IncrementTaskCount(agentID)
	}
}

// extractAgentNameFromTask 从任务的 Description 中提取 Agent 名称。
// Description 格式为 "agentName: taskPrompt"（由 handleDelegate 设置）。
func extractAgentNameFromTask(task *core.Task) string {
	if task == nil || task.Description == "" {
		return ""
	}
	for i, ch := range task.Description {
		if ch == ':' {
			return task.Description[:i]
		}
	}
	return ""
}

// --- Public Methods ---

func (o *ChannelOrchestrator) DelegateTo(
	ctx context.Context,
	agentName, taskPrompt, parentID string,
	metadata map[string]any,
) (*DelegateResult, error) {
	replyCh := make(chan Response, 1)
	msg := Message{
		Type:      MsgDelegate,
		TaskID:    "",    // Assigned by handleDelegate
		From:      "api", // Programmatic access
		Payload:   DelegateRequest{AgentName: agentName, TaskPrompt: taskPrompt, ParentID: parentID, Metadata: metadata},
		ReplyCh:   replyCh,
		Timestamp: time.Now().UnixMilli(),
	}
	select {
	case o.controlCh <- msg:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case resp := <-replyCh:
		if resp.Error != nil {
			return nil, resp.Error
		}
		dr, ok := resp.Data.(*DelegateResult)
		if !ok {
			return nil, fmt.Errorf("unexpected delegate response type")
		}
		return dr, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// RouteTask 是智能路由的公开入口方法 (Design §6.3)。
//
// 与 DelegateTo 不同，RouteTask 不要求调用者指定 Agent 名称，
// 而是通过 LLM Router 语义匹配任务描述与最合适的 Agent：
//
//	① 从 RuntimeDirectory 获取所有 Active Agent 的元数据
//	② 调用 LLM Router (或降级为关键词匹配) 进行语义匹配
//	③ 如果选中现有 Agent → 委托执行
//	④ 如果需要 __CREATE_NEW__ → 通过 AgentFactory 动态创建 → 委托执行
//	⑤ 返回结果
//
// 使用示例：
//
//	result, err := orch.RouteTask(ctx,
//	    "分析这份 PDF 财务报表并提取关键指标",
//	    "PDF 分析和财务数据处理",
//	    "",
//	    nil,
//	)
func (o *ChannelOrchestrator) RouteTask(
	ctx context.Context,
	taskDescription, desiredCapability, parentID string,
	metadata map[string]any,
) (*DelegateResult, error) {
	// Step 1: 收集候选 Agent 元数据
	candidates := o.runtimeDir.ListActive()

	// Step 2: 智能路由决策
	routeReq := RouteRequest{
		TaskDescription:   taskDescription,
		DesiredCapability: desiredCapability,
	}

	var decision *RoutingDecision
	if o.router != nil {
		var err error
		decision, err = o.router.Route(ctx, routeReq, candidates)
		if err != nil {
			return nil, fmt.Errorf("route task failed: %w", err)
		}
	} else {
		// 无 Router 时使用内置 fallback（创建临时 Router 实例来复用 fallbackRoute 逻辑）
		tempRouter, _ := NewLLMRouter(nil) // 创建无 LLM 的 Router，仅用 fallback
		decision = tempRouter.fallbackRoute(routeReq, candidates)
	}

	o.logger.Info("smart routing decision",
		"selected_agent", decision.SelectedAgent,
		"confidence", decision.Confidence,
		"reasoning", decision.Reasoning,
	)

	// Step 3: 处理路由决策
	var agentName string

	switch decision.SelectedAgent {
	case CreateNewAgent:
		// 需要动态创建新 Agent
		if o.factory == nil || !o.factory.CanCreate() {
			return nil, fmt.Errorf("agent factory unavailable or limit reached for task: %s", taskDescription[:min(80, len(taskDescription))])
		}

		newConfig, err := o.factory.Create(ctx, taskDescription, o.modelRegistry)
		if err != nil {
			return nil, fmt.Errorf("dynamic agent creation failed: %w", err)
		}

		// 注册到 Registry 和 RuntimeDirectory
		if err := o.registry.SaveTo(newConfig); err != nil {
			return nil, fmt.Errorf("failed to register new agent: %w", err)
		}
		meta := core.NewAgentRuntimeMeta(newConfig)
		meta.Score = DefaultInitialTrustScore // 冷启动初始信任分 (Design §8.4)
		if err := o.runtimeDir.Register(meta); err != nil {
			// 已存在同名 agent，忽略注册错误（可能已被并发创建）
			o.logger.Warn("runtime dir register failed for new agent", "name", newConfig.Name, "error", err)
		}

		agentName = newConfig.Name

	default:
		agentName = decision.SelectedAgent
	}

	// Step 4: 委托给选中的 Agent 执行（复用现有的 DelegateTo 路径）
	return o.DelegateTo(ctx, agentName, taskDescription, parentID, metadata)
}

func (o *ChannelOrchestrator) WaitForResult(ctx context.Context, taskID string) (*core.Task, error) {
	ch, exists := o.store.GetResultCh(taskID)
	if !exists {
		return o.store.GetTask(taskID) // May be completed already
	}

	select {
	case <-ch:
		// Block until result is written to channel
		task, err := o.store.GetTask(taskID)
		return task, err
	case <-time.After(o.defaultTimeout):
		return o.store.GetTask(taskID) // Return current state even if incomplete
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (o *ChannelOrchestrator) CancelTask(taskID string) error {
	replyCh := make(chan Response, 1)
	msg := Message{
		Type:    MsgCancel,
		TaskID:  taskID,
		From:    "api",
		ReplyCh: replyCh,
	}
	select {
	case o.controlCh <- msg:
	default:
		return fmt.Errorf("orchestrator: control channel full, cannot send cancel for task %q", taskID)
	}

	// Wait for handleCancel to process the message and return its result.
	// Do NOT call store.CancelTask again here — handleCancel already does that.
	select {
	case resp := <-replyCh:
		if resp.Error != nil {
			return resp.Error
		}
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("orchestrator: timeout waiting for cancel confirmation for task %q", taskID)
	}
}

// --- Agent Factory ---

func (o *ChannelOrchestrator) GetAgent(name string) (*goreact.Agent, error) {
	o.agentCacheMu.RLock()
	cached, ok := o.agentCache[name]
	o.agentCacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	// Not cached — build fresh
	config := o.registry.Get(name)
	if config == nil {
		return nil, fmt.Errorf("agent %q not found in registry", name)
	}

	// Resolve model from AgentConfig.Model field
	modelName := config.Model
	if modelName == "" {
		modelName = "default"
	}
	modelCfg, err := o.modelRegistry.Get(modelName)
	if err != nil {
		return nil, fmt.Errorf("model %q not registered for agent %q: %w", modelName, name, err)
	}

	// Build Agent with both Config and Model
	agent, err := goreact.NewAgent(
		goreact.WithConfig(config),
		goreact.WithModel(modelCfg),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent %q: %w", name, err)
	}

	// Cache before returning
	o.agentCacheMu.Lock()
	// Double-check: another goroutine may have cached while we were building
	if existing, ok := o.agentCache[name]; ok {
		o.agentCacheMu.Unlock()
		return existing, nil
	}
	o.agentCache[name] = agent
	o.agentCacheMu.Unlock()

	return agent, nil
}

func (o *ChannelOrchestrator) ReleaseAgent(name string) {
	o.agentCacheMu.Lock()
	delete(o.agentCache, name)
	o.agentCacheMu.Unlock()
}

// --- Event Aggregation ---

func (o *ChannelOrchestrator) Events() (<-chan core.ReactEvent, func()) {
	return o.EventsFiltered(func(e core.ReactEvent) bool { return true })
}

func (o *ChannelOrchestrator) EventsFiltered(filter func(core.ReactEvent) bool) (<-chan core.ReactEvent, func()) {
	ch := make(chan core.ReactEvent, 256)
	o.eventSubsMu.Lock()
	o.eventSubs[ch] = filter
	o.eventSubsMu.Unlock()

	cancel := func() {
		o.eventSubsMu.Lock()
		delete(o.eventSubs, ch)
		o.eventSubsMu.Unlock()
		// Drain remaining events then close
		go func() {
			for range ch {
			}
		}()
	}
	return ch, cancel
}

func (o *ChannelOrchestrator) runEventDispatcher(ctx context.Context) {
	for {
		select {
		case event, ok := <-o.eventOut:
			if !ok {
				return
			}
			o.eventSubsMu.RLock()
			for ch, filter := range o.eventSubs {
				if filter(event) {
					select {
					case ch <- event:
					default:
						// Slow consumer — drop event
					}
				}
			}
			o.eventSubsMu.RUnlock()
		case <-o.eventDone:
			// Drain remaining events
			for range o.eventOut {
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func (o *ChannelOrchestrator) emitEvent(event core.ReactEvent) {
	select {
	case o.eventOut <- event:
	case <-time.After(2 * time.Second):
		o.logger.Warn("event buffer full, dropping event",
			"task_id", event.TaskID, "event_type", event.Type)
	}
}

// --- P1-5: Heartbeat ---

// RecordHeartbeat records a heartbeat from an agent (P1-5 / Design §7.3).
// Should be called periodically by each agent to indicate liveness.
func (o *ChannelOrchestrator) RecordHeartbeat(agentName string) {
	o.heartbeatMu.Lock()
	defer o.heartbeatMu.Unlock()
	o.heartbeats[agentName] = time.Now()
}

// CheckAgentLiveness returns a list of agents that haven't sent a heartbeat
// within the given threshold duration.
func (o *ChannelOrchestrator) CheckAgentLiveness(threshold time.Duration) []string {
	o.heartbeatMu.RLock()
	defer o.heartbeatMu.RUnlock()

	now := time.Now()
	var dead []string
	for name, lastBeat := range o.heartbeats {
		if now.Sub(lastBeat) > threshold {
			dead = append(dead, name)
		}
	}
	return dead
}

// --- P1-4: Idle Cleanup ---

// SetIdleCleanupConfig configures idle agent detection and cleanup (P1-4 / Design §12.4).
func (o *ChannelOrchestrator) SetIdleCleanupConfig(cfg IdleCleanupConfig) {
	o.idleCleanupConfig = cfg
}

// RunIdleCleanup starts the periodic idle cleanup scan (P1-4).
// Should be called once after Start(). Runs until ctx is cancelled.
func (o *ChannelOrchestrator) RunIdleCleanup(ctx context.Context) {
	if o.idleCleanupConfig.CleanupInterval <= 0 || o.idleCleanupConfig.IdleTimeout <= 0 {
		return // Cleanup disabled
	}

	ticker := time.NewTicker(o.idleCleanupConfig.CleanupInterval)
	defer ticker.Stop()

	o.logger.Info("idle cleanup started",
		"idle_timeout", o.idleCleanupConfig.IdleTimeout,
		"cleanup_interval", o.idleCleanupConfig.CleanupInterval,
		"min_retained", o.idleCleanupConfig.MinRetained,
	)

	for {
		select {
		case <-ticker.C:
			o.scanAndCleanupIdleAgents()
		case <-ctx.Done():
			o.logger.Info("idle cleanup stopped")
			return
		}
	}
}

// scanAndCleanupIdleAgents scans for agents that have been idle beyond the threshold (P1-4).
func (o *ChannelOrchestrator) scanAndCleanupIdleAgents() {
	if o.idleCleanupConfig.IdleTimeout <= 0 {
		return
	}

	o.heartbeatMu.Lock()
	defer o.heartbeatMu.Unlock()

	now := time.Now()
	var idleAgents []string
	for name, lastBeat := range o.heartbeats {
		if now.Sub(lastBeat) > o.idleCleanupConfig.IdleTimeout {
			idleAgents = append(idleAgents, name)
		}
	}

	// Don't clean up below minimum retained
	if len(o.heartbeats)-len(idleAgents) < o.idleCleanupConfig.MinRetained {
		return
	}

	for _, name := range idleAgents {
		o.logger.Info("marking agent as dormant due to idle timeout", "agent_name", name)
		delete(o.heartbeats, name)
		// Note: actual agent removal should be handled by the AgentRegistry
		// This is a soft cleanup — removes heartbeat tracking only
	}
}

// --- P2-5: Unified End-to-End Responsibility Result Path ---
// Note: The existing Coordinator.WaitAndCoordinate() + TaskProgressTable.AllCompleted()
// already implements the end-to-end responsibility principle. The Coordinator collects
// all subtask results and returns a unified CoordinationResult to the parent Agent,
// which then continues execution in Executor mode.
// This ensures the caller (who received the original request) is responsible for
// returning the final answer to the user (Design §2.2).

// --- Low-Level Access (P0-2: per-agent inbox architecture) ---

// Send delivers a message to the Orchestrator's control channel (broadcast/global scope).
// For agent-specific delivery, use SendToAgent() instead.
func (o *ChannelOrchestrator) Send(msg Message) <-chan Response {
	replyCh := make(chan Response, 1)
	msg.ReplyCh = replyCh
	msg.Timestamp = time.Now().UnixMilli()
	select {
	case o.controlCh <- msg:
		return replyCh
	case <-time.After(5 * time.Second):
		return make(chan Response)
	}
}

// SendToAgent delivers a message to a specific agent's inbox (Design §7.2 P0-2).
// Returns the agent's reply channel. If the agent doesn't exist, creates a new inbox.
func (o *ChannelOrchestrator) SendToAgent(agentName string, msg Message) <-chan Response {
	replyCh := make(chan Response, 1)
	msg.ReplyCh = replyCh
	msg.Timestamp = time.Now().UnixMilli()

	// Ensure agent inbox exists
	ch := o.ensureAgentInbox(agentName)

	select {
	case ch <- msg:
		return replyCh
	case <-time.After(5 * time.Second):
		return make(chan Response)
	}
}

// ensureAgentInbox returns the buffered inbox for an agent, creating one if needed (P0-2).
func (o *ChannelOrchestrator) ensureAgentInbox(agentName string) chan Message {
	o.agentInboxesMu.RLock()
	ch, ok := o.agentInboxes[agentName]
	o.agentInboxesMu.RUnlock()
	if ok {
		return ch
	}

	// Create new inbox
	o.agentInboxesMu.Lock()
	defer o.agentInboxesMu.Unlock()

	// Double-check after lock upgrade
	if ch, ok = o.agentInboxes[agentName]; ok {
		return ch
	}

	ch = make(chan Message, o.inboxSize)
	o.agentInboxes[agentName] = ch

	// Start a goroutine to multiplex this agent's messages into the control channel
	go func() {
		for msg := range ch {
			o.handleMessage(context.Background(), msg)
		}
	}()

	o.logger.Debug("created agent inbox", "agent_name", agentName, "capacity", o.inboxSize)
	return ch
}

// UnregisterAgentInbox removes and closes an agent's inbox (P0-2).
// Used when an agent is destroyed or removed from the system.
func (o *ChannelOrchestrator) UnregisterAgentInbox(agentName string) {
	o.agentInboxesMu.Lock()
	defer o.agentInboxesMu.Unlock()
	if ch, ok := o.agentInboxes[agentName]; ok {
		close(ch)
		delete(o.agentInboxes, agentName)
		o.logger.Debug("removed agent inbox", "agent_name", agentName)
	}
}

func (o *ChannelOrchestrator) TaskStore() TaskStore { return o.store }

// ListTasks returns tasks filtered by parentID (empty string means all tasks).
// Implements reactor.AgentOrchestrator.ListTasks for tool-layer task queries.
func (o *ChannelOrchestrator) ListTasks(parentID string) ([]*core.Task, error) {
	if parentID != "" {
		return o.store.ListSubTasks(parentID)
	}
	return o.store.ListAllTasks()
}

// GetTask retrieves a single task by ID.
// Implements reactor.AgentOrchestrator.GetTask for tool-layer task queries.
func (o *ChannelOrchestrator) GetTask(taskID string) (*core.Task, error) {
	return o.store.GetTask(taskID)
}

func (o *ChannelOrchestrator) ModelRegistry() core.ModelRegistry { return o.modelRegistry }

// RuntimeDir returns the runtime state directory for agent lifecycle tracking.
// This is used by agents to register themselves and by the orchestrator to track
// agent states (idle/busy/coordinating/error) and performance scores.
func (o *ChannelOrchestrator) RuntimeDir() *core.RuntimeDirectory {
	return o.runtimeDir
}

// RegisterAgent registers an agent's runtime metadata. Delegates to RuntimeDirectory.
func (o *ChannelOrchestrator) RegisterAgent(meta *core.AgentRuntimeMeta) error {
	return o.runtimeDir.Register(meta)
}

// RegisterAgentFromConfig is a convenience method that creates AgentRuntimeMeta
// from an AgentConfig and registers it. This is the preferred way to register —
// it avoids duplicating identity fields since AgentConfig is the single source
// of truth for agent name, description, model, etc.
func (o *ChannelOrchestrator) RegisterAgentFromConfig(config *core.AgentConfig) error {
	meta := core.NewAgentRuntimeMeta(config)
	return o.runtimeDir.Register(meta)
}

// --- Internal Helpers ---

// registryAdapterImpl 将 goreact.AgentRegistry 适配为 AgentFactory 需要的
// goreactRegistryAdapter 接口。这避免了 factory 包直接依赖 goreact 包。
type registryAdapterImpl struct {
	reg *goreact.AgentRegistry
}

func (a *registryAdapterImpl) Get(name string) *core.AgentConfig {
	return a.reg.Get(name)
}

func (a *registryAdapterImpl) List() []*core.AgentConfig {
	return a.reg.List()
}

func (a *registryAdapterImpl) Register(_ string, config *core.AgentConfig) error {
	// 使用 SaveTo 来注册（写入 .md 文件 + 内存）
	return a.reg.SaveTo(config)
}
