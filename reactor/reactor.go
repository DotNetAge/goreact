package reactor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	gochat "github.com/DotNetAge/gochat"
	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/tools"
)

// MaxHistoryTurns limits conversation history turns sent to LLM.
const MaxHistoryTurns = 10

// ReactorConfig holds the configuration for creating a Reactor.
type ReactorConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	ClientType  gochat.ClientType
	Temperature float64
	MaxTokens   int

	SystemPrompt  string
	MaxIterations int

	IsLocal bool
}

// RunResult holds the complete output of a Run invocation.
type RunResult struct {
	Answer                string        `json:"answer" yaml:"answer"`
	Intent                *Intent       `json:"intent,omitempty" yaml:"intent,omitempty"`
	Steps                 []Step        `json:"steps,omitempty" yaml:"steps,omitempty"`
	TotalIterations       int           `json:"total_iterations" yaml:"total_iterations"`
	TerminationReason     string        `json:"termination_reason,omitempty" yaml:"termination_reason,omitempty"`
	Confidence            float64       `json:"confidence" yaml:"confidence"`
	ClarificationNeeded   bool          `json:"clarification_needed" yaml:"clarification_needed"`
	ClarificationQuestion string        `json:"clarification_question,omitempty" yaml:"clarification_question,omitempty"`
	TokensUsed            int           `json:"tokens_used,omitempty" yaml:"tokens_used,omitempty"`
	TotalDuration         time.Duration `json:"total_duration_ms,omitempty" yaml:"total_duration_ms,omitempty"`
}

// ReActor is the public interface for the T-A-O reactor.
// External consumers only need Run/RunFromSnapshot for task execution.
type ReActor interface {
	Run(ctx context.Context, input string, history ConversationHistory) (*RunResult, error)
	RunFromSnapshot(ctx context.Context, snapshot *RunSnapshot, newInput string) (*RunResult, error)
}

// ReActorInternal extends ReActor with individual T-A-O phase access.
// Used by test code and internal orchestration that needs fine-grained control.
type ReActorInternal interface {
	ReActor
	Think(ctx *ReactContext) (int, error)
	Act(ctx *ReactContext) error
	Observe(ctx *ReactContext) error
	CheckTermination(ctx *ReactContext) (bool, string)
}

var _ ReActorInternal = (*Reactor)(nil)

type Reactor struct {
	config         ReactorConfig
	intentRegistry IntentRegistry
	toolRegistry   core.ToolRegistryInterface
	skillRegistry  core.SkillRegistry
	taskManager    core.TaskManager
	llmClient      gochat.ClientBuilder

	memory         core.Memory
	tokenEstimator core.TokenEstimator

	askUser       *tools.AskUser
	askPermission *tools.AskPermission
	eventBus      EventBus
	messageBus    *core.AgentMessageBus

	pendingTasks   map[string]chan any
	pendingTasksMu sync.RWMutex

	scheduler *core.CronScheduler

	sessionStore  core.SessionStore
	contextWindow *core.ContextWindow
	slideConfig   core.SlideConfig

	mockLLM func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error)

	pauseRequested bool
	pauseMu        sync.Mutex

	snapshotHolder struct {
		sync.RWMutex
		snap *RunSnapshot
	}
}

func (r *Reactor) EventBus() EventBus { return r.eventBus }

func (r *Reactor) MessageBus() *core.AgentMessageBus { return r.messageBus }

func (r *Reactor) AskUser() *tools.AskUser { return r.askUser }

func (r *Reactor) SetAskUser(t *tools.AskUser) { r.askUser = t }

func (r *Reactor) AskPermission() *tools.AskPermission { return r.askPermission }

func (r *Reactor) Memory() core.Memory { return r.memory }

func (r *Reactor) SetPauseRequested() {
	r.pauseMu.Lock()
	defer r.pauseMu.Unlock()
	r.pauseRequested = true
}

func (r *Reactor) TakeSnapshot() *RunSnapshot {
	r.pauseMu.Lock()
	defer r.pauseMu.Unlock()
	r.pauseRequested = false
	return r.snapshotHolder.snap
}

func (r *Reactor) setSnapshot(snap *RunSnapshot) {
	r.snapshotHolder.Lock()
	defer r.snapshotHolder.Unlock()
	r.snapshotHolder.snap = snap
}

func (r *Reactor) getSnapshot() *RunSnapshot {
	r.snapshotHolder.Lock()
	defer r.snapshotHolder.Unlock()
	snap := r.snapshotHolder.snap
	r.snapshotHolder.snap = nil
	return snap
}

func (r *Reactor) clearSnapshot() {
	r.snapshotHolder.Lock()
	defer r.snapshotHolder.Unlock()
	r.snapshotHolder.snap = nil
}

func (r *Reactor) ConsumeSnapshot() *RunSnapshot { return r.getSnapshot() }

func (r *Reactor) PeekSnapshot() *RunSnapshot {
	r.snapshotHolder.RLock()
	defer r.snapshotHolder.RUnlock()
	return r.snapshotHolder.snap
}

func (r *Reactor) SetAskPermission(p *tools.AskPermission) { r.askPermission = p }

type reactorSetup struct {
	systemPrompt      string
	skipTools         map[string]bool
	skipAllBundled    bool
	extraTools        []core.FuncTool
	securityPolicy    core.SecurityPolicy
	resultStorage     core.ToolResultStorage
	resultLimits      core.ToolResultLimits
	tokenEstimator    core.TokenEstimator
	eventBus          EventBus
	mcpRegistry       *core.MCPToolRegistry
	skillDirs         []string
	skipBundledSkills bool
	messageBus        *core.AgentMessageBus
	memory            core.Memory
	mockLLM           func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error)
	scheduler         *core.CronScheduler
	sessionStore      core.SessionStore
	intentRegistry    IntentRegistry
	toolRegistry      core.ToolRegistryInterface
	skillRegistry     core.SkillRegistry
}

func NewReactor(config ReactorConfig, opts ...ReactorOption) *Reactor {
	if config.MaxIterations <= 0 {
		config.MaxIterations = core.DefaultMaxSteps
	}
	if config.Temperature <= 0 {
		config.Temperature = core.DefaultTemperature
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = core.DefaultMaxTokens
	}

	setup := &reactorSetup{skipTools: make(map[string]bool)}
	for _, opt := range opts {
		opt(setup)
	}

	if setup.systemPrompt != "" {
		config.SystemPrompt = setup.systemPrompt
	}

	r := &Reactor{
		config:         config,
		taskManager:    core.NewInMemoryTaskManager(),
		tokenEstimator: core.NewDefaultTokenEstimator(3.0),
		memory:         setup.memory,
		mockLLM:        setup.mockLLM,
		sessionStore:   setup.sessionStore,
		slideConfig:    core.DefaultSlideConfig,
	}

	if setup.intentRegistry != nil {
		r.intentRegistry = setup.intentRegistry
	} else {
		r.intentRegistry = NewDefaultIntentRegistry()
	}
	if setup.toolRegistry != nil {
		r.toolRegistry = setup.toolRegistry
	} else {
		r.toolRegistry = NewDefaultToolRegistry()
	}
	if setup.skillRegistry != nil {
		r.skillRegistry = setup.skillRegistry
	} else {
		r.skillRegistry = NewDefaultSkillRegistry()
	}

	if setup.eventBus != nil {
		r.eventBus = setup.eventBus
	} else {
		r.eventBus = NewEventBus()
	}

	if setup.messageBus != nil {
		r.messageBus = setup.messageBus
	} else {
		r.messageBus = core.NewAgentMessageBus()
	}

	if setup.scheduler != nil {
		r.scheduler = setup.scheduler
	}

	r.llmClient = gochat.Client().Config(
		gochat.WithAPIKey(config.APIKey),
		gochat.WithBaseURL(config.BaseURL),
	)

	if !setup.skipBundledSkills {
		if err := RegisterBundledSkills(r.skillRegistry); err != nil {
			fmt.Printf("[goreact] warning: failed to register bundled skills: %v\n", err)
		}
	}

	for _, dir := range setup.skillDirs {
		loader := core.NewFileSystemSkillLoader(dir)
		skills, err := loader.Load()
		if err != nil {
			fmt.Printf("[goreact] warning: failed to load skills from %q: %v\n", dir, err)
			continue
		}
		for _, skill := range skills {
			if err := r.skillRegistry.RegisterSkill(skill); err != nil {
				fmt.Printf("[goreact] warning: failed to register skill %q: %v\n", skill.Name, err)
			}
		}
	}

	r.registerOrchestrationTools()

	r.askUser = tools.NewAskUserTool().(*tools.AskUser)
	r.askUser.SetEventEmitter(func(e core.ReactEvent) {
		if r.eventBus != nil {
			r.eventBus.Emit(e)
		}
	})
	_ = r.RegisterTool(r.askUser)

	r.askPermission = tools.NewAskPermission()
	r.askPermission.SetEventEmitter(func(e core.ReactEvent) {
		if r.eventBus != nil {
			r.eventBus.Emit(e)
		}
	})
	r.toolRegistry.SetPermissionChecker(r.askPermission)
	r.toolRegistry.SetEventEmitter(func(e core.ReactEvent) {
		if r.eventBus != nil {
			r.eventBus.Emit(e)
		}
	})

	if !setup.skipAllBundled {
		bundledTools := []struct {
			name string
			tool core.FuncTool
		}{
			{"grep", tools.NewGrepTool()},
			{"glob", tools.NewGlobTool()},
			{"bash", tools.NewBashTool()},
			{"web_search", tools.NewWebSearchTool()},
			{"web_fetch", tools.NewWebFetchTool()},
			{"todo_write", tools.NewTodoWriteTool()},
			{"todo_read", tools.NewTodoReadTool()},
			{"todo_execute", tools.NewTodoExecuteTool()},
			{"file_edit", tools.NewFileEditTool()},
			{"repl", tools.NewREPLTool()},
			{"read", tools.NewReadTool()},
			{"write", tools.NewWriteTool()},
			{"ls", tools.NewLsTool()},
			{"echo", tools.NewEchoTool()},
			{"calculator", tools.NewCalculatorTool()},
			{"replace", tools.NewReplaceTool()},
			{"cron", tools.NewCronTool()},
			{"memory_save", tools.NewMemorySaveTool()},
			{"memory_search", tools.NewMemorySearchTool()},
			{"email", tools.NewEmailTool(tools.EmailConfig{})},
			{"ask_user", tools.NewAskUserTool()},
		}
		for _, bt := range bundledTools {
			if !setup.skipTools[bt.name] {
				_ = r.RegisterTool(bt.tool)
			}
		}

		if cronTool, ok := r.toolRegistry.Get("cron"); ok {
			if ct, ok := cronTool.(*tools.Cron); ok {
				ct.SetAccessor(r)
			}
		}

		r.registerOrchestrationTools()
	}

	for _, tool := range setup.extraTools {
		_ = r.RegisterTool(tool)
	}

	if setup.securityPolicy != nil {
		r.toolRegistry.SetSecurityPolicy(setup.securityPolicy)
	}

	if setup.resultStorage != nil {
		r.toolRegistry.SetResultStorage(setup.resultStorage)
	}
	if setup.resultLimits.MaxResultSizeChars > 0 {
		r.toolRegistry.SetResultLimits(setup.resultLimits)
	}
	if setup.tokenEstimator != nil {
		r.tokenEstimator = setup.tokenEstimator
	}

	if r.memory != nil {
		tools.SetMemory(r.memory)
		r.toolRegistry.SetMemory(r.memory)
	}

	return r
}

func (r *Reactor) SkillRegistry() core.SkillRegistry         { return r.skillRegistry }
func (r *Reactor) IntentRegistry() IntentRegistry            { return r.intentRegistry }
func (r *Reactor) ToolRegistry() core.ToolRegistryInterface  { return r.toolRegistry }
func (r *Reactor) TaskManager() core.TaskManager             { return r.taskManager }
func (r *Reactor) Scheduler() *core.CronScheduler            { return r.scheduler }
func (r *Reactor) SessionStore() core.SessionStore           { return r.sessionStore }
func (r *Reactor) ContextWindow() *core.ContextWindow        { return r.contextWindow }
func (r *Reactor) SetContextWindow(cw *core.ContextWindow)   { r.contextWindow = cw }
func (r *Reactor) RegisterTool(tool core.FuncTool) error     { return r.toolRegistry.Register(tool) }
func (r *Reactor) RegisterIntent(def IntentDefinition) error { return r.intentRegistry.Register(def) }

func (r *Reactor) Run(ctx context.Context, input string, history ConversationHistory) (*RunResult, error) {
	r.persistMessage(ctx, "user", input)

	if r.sessionStore != nil && r.contextWindow != nil {
		maxTokensForHistory := int64(float64(r.config.MaxTokens) * 0.7)
		if msgs, err := r.sessionStore.CurrentContext(ctx, r.contextWindow.SessionID, maxTokensForHistory); err == nil && len(msgs) > 0 {
			history = ConversationHistory(msgs)
		}
	}

	reactCtx := NewReactContext(ctx, input, history, r.config.MaxIterations)

	if r.eventBus != nil {
		reactCtx.emitEvent = r.eventBus.Emit
	}

	intent, tokens, err := r.classifyIntent(reactCtx)
	if err != nil {
		reactCtx.EmitEvent(core.Error, fmt.Sprintf("intent classification: %v", err))
		return nil, fmt.Errorf("intent classification: %w", err)
	}
	reactCtx.Intent = intent
	ApplyConfidenceThreshold(intent, 0)

	if intent.RequiresClarification {
		reactCtx.EmitEvent(core.ClarifyNeeded, intent.ClarificationQuestion)
		return &RunResult{
			Intent:                intent,
			ClarificationNeeded:   true,
			ClarificationQuestion: intent.ClarificationQuestion,
			Confidence:            intent.Confidence,
			TokensUsed:            tokens,
		}, nil
	}

	result, err := r.runTAOLoop(reactCtx, tokens, time.Now())
	if err == nil && result != nil && result.Answer != "" {
		r.persistMessage(ctx, "assistant", result.Answer)
		if r.contextWindow != nil {
			r.contextWindow.AddTokens(int64(result.TokensUsed))
		}
		r.checkSlide(ctx)
	}
	return result, err
}

func (r *Reactor) RunFromSnapshot(ctx context.Context, snapshot *RunSnapshot, newInput string) (*RunResult, error) {
	reactCtx := NewReactContextFromSnapshot(ctx, snapshot)

	if r.eventBus != nil {
		reactCtx.emitEvent = r.eventBus.Emit
	}

	reactCtx.IsTerminated = false
	reactCtx.TerminationReason = ""

	if newInput != "" {
		reactCtx.AddMessage("user", newInput)
	}

	return r.runTAOLoop(reactCtx, 0, time.Now())
}

func (r *Reactor) runTAOLoop(reactCtx *ReactContext, initialTokens int, runStart time.Time) (*RunResult, error) {
	totalTokens := initialTokens

	for reactCtx.CurrentIteration < reactCtx.MaxIterations {
		if terminated, reason := r.CheckTermination(reactCtx); terminated {
			reactCtx.IsTerminated = true
			reactCtx.TerminationReason = reason
			break
		}

		cycleStart := time.Now()
		r.toolRegistry.ResetMessageCharCounter()

		tokens, err := r.Think(reactCtx)
		totalTokens += tokens
		if err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("think error: %v", err)
			reactCtx.EmitEvent(core.Error, reactCtx.TerminationReason)
			break
		}
		reactCtx.EmitEvent(core.ThinkingDone, reactCtx.LastThought)

		if err := r.Act(reactCtx); err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("act error: %v", err)
			reactCtx.EmitEvent(core.Error, reactCtx.TerminationReason)
			break
		}

		if reactCtx.LastAction.Type == ActionTypeToolCall {
			reactCtx.EmitEvent(core.ActionStart, core.ActionStartData{
				ToolName: reactCtx.LastAction.Target,
				Params:   reactCtx.LastAction.Params,
			})
			resultData := core.ActionResultData{
				ToolName: reactCtx.LastAction.Target,
				Duration: reactCtx.LastAction.Duration,
				Success:  reactCtx.LastAction.Error == nil,
			}
			if reactCtx.LastAction.Error != nil {
				resultData.Error = reactCtx.LastAction.ErrorMsg
			} else {
				resultData.Result = reactCtx.LastAction.Result
			}
			reactCtx.EmitEvent(core.ActionResult, resultData)
		}

		if err := r.Observe(reactCtx); err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("observe error: %v", err)
			reactCtx.EmitEvent(core.Error, reactCtx.TerminationReason)
			break
		}
		reactCtx.EmitEvent(core.ObservationDone, reactCtx.LastObservation)

		step := Step{
			Iteration:   reactCtx.CurrentIteration + 1,
			Thought:     *reactCtx.LastThought,
			Action:      *reactCtx.LastAction,
			Observation: *reactCtx.LastObservation,
			Timestamp:   time.Now(),
			Duration:    time.Since(cycleStart),
		}
		reactCtx.AppendHistory(step)

		reactCtx.EmitEvent(core.CycleEnd, core.CycleInfo{
			Iteration: reactCtx.CurrentIteration + 1,
			Duration:  time.Since(cycleStart),
		})

		var stepSummary strings.Builder
		fmt.Fprintf(&stepSummary, "Thought: %s", reactCtx.LastThought.Reasoning)
		if reactCtx.LastThought.Decision == DecisionAct {
			fmt.Fprintf(&stepSummary, "\nAction: %s(%v)", reactCtx.LastAction.Target, reactCtx.LastAction.Params)
		}
		if reactCtx.LastObservation.Result != "" {
			fmt.Fprintf(&stepSummary, "\nObservation: %s", reactCtx.LastObservation.Result)
		}
		if reactCtx.LastObservation.Error != "" {
			fmt.Fprintf(&stepSummary, "\nObservation Error: %s", reactCtx.LastObservation.Error)
		}
		stepSummaryStr := stepSummary.String()
		reactCtx.AddMessage("assistant", stepSummaryStr)
		r.persistMessage(reactCtx.Ctx(), "assistant", stepSummaryStr)

		reactCtx.CurrentIteration++
	}

	result := &RunResult{
		Intent:            reactCtx.Intent,
		Steps:             reactCtx.History,
		TotalIterations:   reactCtx.CurrentIteration,
		TerminationReason: reactCtx.TerminationReason,
		Confidence:        reactCtx.Intent.Confidence,
		TokensUsed:        totalTokens,
	}

	if reactCtx.LastAction != nil {
		result.Answer = reactCtx.LastAction.Result
		if reactCtx.LastAction.Type == ActionTypeClarify {
			result.ClarificationNeeded = true
			result.ClarificationQuestion = reactCtx.LastAction.Result
		}
	}
	if result.Answer == "" && reactCtx.LastThought != nil {
		result.Answer = reactCtx.LastThought.FinalAnswer
	}
	if result.Answer == "" && reactCtx.LastObservation != nil {
		result.Answer = reactCtx.LastObservation.Result
	}
	if result.Answer == "" && reactCtx.TerminationReason != "" {
		result.Answer = fmt.Sprintf("[Task terminated: %s]", reactCtx.TerminationReason)
	}

	if result.Answer != "" {
		reactCtx.EmitEvent(core.FinalAnswer, result.Answer)
	}

	totalDuration := time.Since(runStart)
	result.TotalDuration = totalDuration

	summary := core.ExecutionSummaryData{
		TotalIterations:   result.TotalIterations,
		TotalDuration:     totalDuration,
		TokensUsed:        totalTokens,
		TerminationReason: result.TerminationReason,
	}
	summary.ToolsUsed = collectUniqueToolNames(reactCtx.History)
	summary.ToolCalls = 0
	for _, step := range reactCtx.History {
		if step.Action.Type == ActionTypeToolCall && step.Action.Target != "" {
			summary.ToolCalls++
		}
	}
	reactCtx.EmitEvent(core.ExecutionSummary, summary)

	r.pauseMu.Lock()
	paused := r.pauseRequested
	r.pauseRequested = false
	r.pauseMu.Unlock()
	if paused {
		snap := reactCtx.ToSnapshot()
		snap.TerminationReason = "paused"
		r.setSnapshot(snap)
	}

	r.saveExperience(reactCtx, result)

	if result.TotalIterations > 1 && result.Answer != "" &&
		!strings.Contains(result.TerminationReason, "error") {
		r.generateSummary(reactCtx, result, totalDuration)
	}

	return result, nil
}
