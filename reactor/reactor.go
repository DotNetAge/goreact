package reactor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	gochat "github.com/DotNetAge/gochat"
	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/tools"
)

var logger = slog.Default()

const (
	historyTokenBudgetRatio = 0.7
	StreamChannelBufferSize = 256

	defaultMaxHistoryTurns = 8
	tokensPerTurnEstimate  = 500
	minMaxHistoryTurns     = 3
	maxMaxHistoryTurns     = 20
)

func maxHistoryTurnsForConfig(maxTokens int) int {
	if maxTokens <= 0 {
		return defaultMaxHistoryTurns
	}
	estimated := int(float64(maxTokens) * historyTokenBudgetRatio / float64(tokensPerTurnEstimate))
	if estimated < minMaxHistoryTurns {
		return minMaxHistoryTurns
	}
	if estimated > maxMaxHistoryTurns {
		return maxMaxHistoryTurns
	}
	return estimated
}

// ReactorConfig holds the configuration for creating a Reactor.
// Generation parameters are aligned with core.ModelConfig for full LLM control.
type ReactorConfig struct {
	APIKey     string
	BaseURL    string
	AuthToken  string
	Model      string
	ClientType gochat.ClientType

	Temperature      float64
	TopP             float64
	TopK             int
	PresencePenalty  float64
	FrequencyPenalty float64
	MaxTokens        int

	SystemPrompt  string
	MaxIterations int

	IsLocal bool
}

// RunResult holds the complete output of a Run invocation.
type RunResult struct {
	Answer                string               `json:"answer" yaml:"answer"`
	Intent                *Intent              `json:"intent,omitempty" yaml:"intent,omitempty"`
	Steps                 []Step               `json:"steps,omitempty" yaml:"steps,omitempty"`
	TotalIterations       int                  `json:"total_iterations" yaml:"total_iterations"`
	TerminationReason     string               `json:"termination_reason,omitempty" yaml:"termination_reason,omitempty"`
	Confidence            float64              `json:"confidence" yaml:"confidence"`
	ClarificationNeeded   bool                 `json:"clarification_needed" yaml:"clarification_needed"`
	ClarificationQuestion string               `json:"clarification_question,omitempty" yaml:"clarification_question,omitempty"`
	TokensUsed            int                  `json:"tokens_used,omitempty" yaml:"tokens_used,omitempty"`
	TotalDuration         time.Duration        `json:"total_duration_ms,omitempty" yaml:"total_duration_ms,omitempty"`
	Experience            *core.ExperienceData `json:"experience,omitempty" yaml:"experience,omitempty"`
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
	toolRegistry   core.ToolRegistry
	toolExecutor   core.ToolExecutor
	skillRegistry  core.SkillRegistry
	ruleRegistry   core.RuleRegistry
	taskManager    core.TaskManager
	llmClient      gochat.ClientBuilder

	memory         core.Memory
	tokenEstimator core.TokenEstimator

	interactionHandler HumanInteractionHandler
	askPermission      *tools.AskPermission
	eventBus           EventBus
	messageBus         *core.AgentMessageBus

	pendingTasks   map[string]chan any
	pendingTasksMu sync.RWMutex

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

func (r *Reactor) InteractionHandler() HumanInteractionHandler { return r.interactionHandler }

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
	resultLimits      core.ToolResultLimits
	tokenEstimator    core.TokenEstimator
	eventBus          EventBus
	mcpRegistry       *core.MCPToolRegistry
	skillDirs         []string
	skipBundledSkills bool
	messageBus        *core.AgentMessageBus
	memory            core.Memory
	mockLLM           func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error)
	sessionStore      core.SessionStore
	intentRegistry    IntentRegistry
	toolRegistry      core.ToolRegistry
	skillRegistry     core.SkillRegistry
	ruleRegistry      core.RuleRegistry
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
		tokenEstimator: core.NewTokenEstimator(),
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
	if setup.ruleRegistry != nil {
		r.ruleRegistry = setup.ruleRegistry
	} else {
		r.ruleRegistry = NewDefaultRuleRegistry()
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

	r.llmClient = gochat.Client().Config(
		gochat.WithAPIKey(config.APIKey),
		gochat.WithBaseURL(config.BaseURL),
	)

	if !setup.skipBundledSkills {
		if err := RegisterBundledSkills(r.skillRegistry); err != nil {
			logger.Warn("failed to register bundled skills", "error", err)
		}
	}

	for _, dir := range setup.skillDirs {
		loader := core.NewFileSystemSkillLoader(dir)
		skills, err := loader.Load()
		if err != nil {
			logger.Warn("failed to load skills", "dir", dir, "error", err)
			continue
		}
		for _, skill := range skills {
			if err := r.skillRegistry.RegisterSkill(skill); err != nil {
				logger.Warn("failed to register skill", "name", skill.Name, "error", err)
			}
		}
	}

	r.registerOrchestrationTools()

	r.interactionHandler = NewDefaultInteractionHandler(func(e core.ReactEvent) {
		if r.eventBus != nil {
			r.eventBus.Emit(e)
		}
	})
	if err := r.RegisterTool(tools.NewAskUserTool()); err != nil {
		logger.Warn("failed to register ask_user tool", "error", err)
	}

	r.askPermission = tools.NewAskPermission()
	r.askPermission.SetEventEmitter(func(e core.ReactEvent) {
		if r.eventBus != nil {
			r.eventBus.Emit(e)
		}
	})
	r.toolExecutor = core.NewToolExecutor(
		r.toolRegistry,
		core.WithPermissionChecker(r.askPermission),
		core.WithResultLimits(setup.resultLimits),
		core.WithEventEmitter(func(e core.ReactEvent) {
			if r.eventBus != nil {
				r.eventBus.Emit(e)
			}
		}),
	)

	if !setup.skipAllBundled {
		bundledTools := []struct {
			name string
			tool core.FuncTool
		}{
			// --- File operations ---
			{"grep", tools.NewGrepTool()},
			{"glob", tools.NewGlobTool()},
			{"read", tools.NewReadTool()},
			{"write", tools.NewWriteTool()},
			{"file_edit", tools.NewFileEditTool()},

			// --- Execution ---
			{"bash", tools.NewBashTool()},
			{"run_script", tools.NewRunScriptTool()},

			// --- Network ---
			{"web_search", tools.NewWebSearchTool()},
			{"web_fetch", tools.NewWebFetchTool()},

			// --- Knowledge & Memory ---
			{"memory_save", tools.NewMemorySaveTool()},
			{"memory_search", tools.NewMemorySearchTool()},

			// --- Task management ---
			{"todo_write", tools.NewTodoWriteTool()},
			{"todo_read", tools.NewTodoReadTool()},
			{"todo_execute", tools.NewTodoExecuteTool()},

			// --- Communication ---
			{"email", tools.NewEmailTool(tools.EmailConfig{})},
			{"ask_user", tools.NewAskUserTool()},
		}
		for _, bt := range bundledTools {
			if !setup.skipTools[bt.name] {
				if err := r.RegisterTool(bt.tool); err != nil {
					logger.Warn("failed to register bundled tool", "name", bt.name, "error", err)
				}
			}
		}

		r.registerOrchestrationTools()
	}

	for _, tool := range setup.extraTools {
		if err := r.RegisterTool(tool); err != nil {
			logger.Warn("failed to register extra tool", "error", err)
		}
	}

	if setup.tokenEstimator != nil {
		r.tokenEstimator = setup.tokenEstimator
	}

	if r.memory != nil {
		tools.SetMemory(r.memory)
	}

	return r
}

func (r *Reactor) SkillRegistry() core.SkillRegistry         { return r.skillRegistry }
func (r *Reactor) IntentRegistry() IntentRegistry            { return r.intentRegistry }
func (r *Reactor) ToolRegistry() core.ToolRegistry           { return r.toolRegistry }
func (r *Reactor) ToolExecutor() core.ToolExecutor           { return r.toolExecutor }
func (r *Reactor) RuleRegistry() core.RuleRegistry           { return r.ruleRegistry }
func (r *Reactor) TaskManager() core.TaskManager             { return r.taskManager }
func (r *Reactor) SessionStore() core.SessionStore           { return r.sessionStore }
func (r *Reactor) ContextWindow() *core.ContextWindow        { return r.contextWindow }
func (r *Reactor) SetContextWindow(cw *core.ContextWindow)   { r.contextWindow = cw }
func (r *Reactor) RegisterTool(tool core.FuncTool) error     { return r.toolRegistry.Register(tool) }
func (r *Reactor) RegisterIntent(def IntentDefinition) error { return r.intentRegistry.Register(def) }
func (r *Reactor) maxHistoryTurns() int                      { return maxHistoryTurnsForConfig(r.config.MaxTokens) }

// CloneReactor creates a child Reactor that inherits all registries, infrastructure,
// and execution pipeline from the parent, but with an independent config, task manager,
// LLM client, and conversation context.
//
// Shared (same reference as parent):
//   - intentRegistry, toolRegistry, skillRegistry (tool/skill/intent definitions)
//   - toolExecutor (permission chain, hooks, result limits)
//   - memory, eventBus, messageBus (communication & persistence)
//   - tokenEstimator, sessionStore (context management)
//   - mockLLM (testing support)
//
// Independent (new instances for child):
//   - config (Model, SystemPrompt, Temperature, etc. — can override)
//   - taskManager (child's own task tracking)
//   - llmClient (child's own API connection)
//   - contextWindow (child's own conversation history)
//   - pendingTasks (child's own async task channels)
//   - askUser, askPermission (child's own interaction tools)
//
// Use case: SubAgent creation where the child needs parent's tools/skills/memory
// but runs its own T-A-O loop with possibly a different model or system prompt.
func (r *Reactor) CloneReactor(configOverride ReactorConfig) *Reactor {
	childConfig := r.config
	if configOverride.Model != "" {
		childConfig.Model = configOverride.Model
	}
	if configOverride.SystemPrompt != "" {
		childConfig.SystemPrompt = configOverride.SystemPrompt
	}
	if configOverride.APIKey != "" {
		childConfig.APIKey = configOverride.APIKey
	}
	if configOverride.BaseURL != "" {
		childConfig.BaseURL = configOverride.BaseURL
	}
	if configOverride.AuthToken != "" {
		childConfig.AuthToken = configOverride.AuthToken
	}
	if configOverride.Temperature > 0 {
		childConfig.Temperature = configOverride.Temperature
	}
	if configOverride.TopP > 0 {
		childConfig.TopP = configOverride.TopP
	}
	if configOverride.TopK > 0 {
		childConfig.TopK = configOverride.TopK
	}
	if configOverride.PresencePenalty != 0 {
		childConfig.PresencePenalty = configOverride.PresencePenalty
	}
	if configOverride.FrequencyPenalty != 0 {
		childConfig.FrequencyPenalty = configOverride.FrequencyPenalty
	}
	if configOverride.MaxTokens > 0 {
		childConfig.MaxTokens = configOverride.MaxTokens
	}
	if configOverride.IsLocal {
		childConfig.IsLocal = configOverride.IsLocal
	}

	child := &Reactor{
		config:         childConfig,
		intentRegistry: r.intentRegistry,
		toolRegistry:   r.toolRegistry,
		toolExecutor:   r.toolExecutor,
		skillRegistry:  r.skillRegistry,
		ruleRegistry:   r.ruleRegistry,
		memory:         r.memory,
		tokenEstimator: r.tokenEstimator,
		eventBus:       r.eventBus,
		messageBus:     r.messageBus,
		sessionStore:   r.sessionStore,
		slideConfig:    r.slideConfig,
		mockLLM:        r.mockLLM,
		taskManager:    core.NewInMemoryTaskManager(),
		pendingTasks:   make(map[string]chan any),
	}

	child.llmClient = gochat.Client().Config(
		gochat.WithAPIKey(childConfig.APIKey),
		gochat.WithBaseURL(childConfig.BaseURL),
	)

	child.interactionHandler = NewDefaultInteractionHandler(func(e core.ReactEvent) {
		if child.eventBus != nil {
			child.eventBus.Emit(e)
		}
	})

	child.askPermission = tools.NewAskPermission()
	child.askPermission.SetEventEmitter(func(e core.ReactEvent) {
		if child.eventBus != nil {
			child.eventBus.Emit(e)
		}
	})

	return child
}

func (r *Reactor) Run(ctx context.Context, input string, history ConversationHistory) (*RunResult, error) {
	r.persistMessage(ctx, "user", input)

	if r.sessionStore != nil && r.contextWindow != nil {
		maxTokensForHistory := int64(float64(r.config.MaxTokens) * historyTokenBudgetRatio)
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
		r.toolExecutor.ResetCycle()

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

	result.Experience = r.buildExperienceCandidate(reactCtx, result)

	if result.TotalIterations > 1 && result.Answer != "" &&
		!strings.Contains(result.TerminationReason, "error") {
		r.generateSummary(reactCtx, result, totalDuration)
	}

	return result, nil
}
