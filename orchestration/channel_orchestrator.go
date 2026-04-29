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
//   - Agent 工厂: GetAgent with Registry cache + Model resolution
//   - 事件聚合器: Unified event stream from all agents
//   - Model 分配器: Automatic model lookup per agent config
type ChannelOrchestrator struct {
	// === Configuration (immutable after construction) ===
	modelRegistry core.ModelRegistry
	registry     *goreact.AgentRegistry
	spawnFunc    SpawnFunction
	maxConcurrent int
	defaultTimeout time.Duration
	inboxSize     int

	// === Agent Factory Cache ===
	agentCache   map[string]*goreact.Agent // name -> cached instance
	agentCacheMu sync.RWMutex

	// === Task Management ===
	store TaskStore

	// === Channel Actor Loop ===
	inbox    chan Message
	done     chan struct{} // Closed when Stop() completes
	started  bool
	startOnce sync.Once

	// === Event Aggregation ===
	eventOut      chan core.ReactEvent
	eventSubsMu   sync.RWMutex
	eventSubs     map[chan<- core.ReactEvent]func(core.ReactEvent) bool
	eventDone     chan struct{} // Closed when Stop() completes

	// logger
	logger *slog.Logger
}

// New creates a new ChannelOrchestrator with the given options.
// Call Start(ctx) to begin processing messages.
func New(opts ...OrchestratorOption) (*ChannelOrchestrator, error) {
	setup := &orchestratorSetup{}
	for _, opt := range opts {
		opt(setup)
	}

	o := &ChannelOrchestrator{
		modelRegistry: setup.modelRegistry,
		registry:      setup.registry,
		spawnFunc:    setup.spawnFunc,
		maxConcurrent: setup.maxConcurrent,
		inboxSize:     setup.inboxSize,
		agentCache:    make(map[string]*goreact.Agent),
		store:         NewInMemoryTaskStore(),
		inbox:         make(chan Message, setup.inboxSize),
		done:          make(chan struct{}),
		eventOut:      make(chan core.ReactEvent, 256),
		eventSubs:     make(map[chan<- core.ReactEvent]func(core.ReactEvent) bool),
		eventDone:      make(chan struct{}),
		logger:        slog.Default(),
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
		o.started = true
		// Load agents from directory if configured and no registry yet
		// (handled by WithAgentsDir in New or pre-built registry)

		// Start the Actor loop goroutine
		go o.runLoop(ctx)

		// Start event dispatcher goroutine
		go o.runEventDispatcher(ctx)

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
	// Close inbox to signal runLoop to drain and exit
	close(o.inbox)

	// Wait for runLoop to finish
	select {
	case <-o.done:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Close event aggregator
	close(o.eventDone)

	o.started = false
	o.logger.Info("orchestrator stopped")
	return nil
}

// --- Actor Loop ---

func (o *ChannelOrchestrator) runLoop(ctx context.Context) {
	defer close(o.done)
	for {
		select {
		case msg, ok := <-o.inbox:
			if !ok {
				return // Inbox closed, shutdown
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

	o.emitEvent(core.ReactEvent{
		Type: core.SubtaskCompleted,
		Data: core.SubtaskResult{
			TaskID:  task.ID,
			Success: success,
			Answer:  answer,
			Error:   errMsg,
		},
	})
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
		TaskID:    "", // Assigned by handleDelegate
		From:      "api", // Programmatic access
		Payload:   DelegateRequest{AgentName: agentName, TaskPrompt: taskPrompt, ParentID: parentID, Metadata: metadata},
		ReplyCh:   replyCh,
		Timestamp: time.Now().UnixMilli(),
	}
	select {
	case o.inbox <- msg:
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
	case o.inbox <- msg:
	case <-replyCh:
			// Will be processed by handleCancel
	}
	err := o.store.CancelTask(taskID)
	return err
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
	default:
		// Event buffer full — non-blocking to avoid blocking Actor loop
	}
}

// --- Low-Level Access ---

func (o *ChannelOrchestrator) Send(msg Message) <-chan Response {
	replyCh := make(chan Response, 1)
	msg.ReplyCh = replyCh
	msg.Timestamp = time.Now().UnixMilli()
	select {
	case o.inbox <- msg:
		return replyCh
	case <-time.After(5 * time.Second):
		return make(chan Response)
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
