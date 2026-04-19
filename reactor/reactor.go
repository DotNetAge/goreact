package reactor

import (
	"context"
	"encoding/json"
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
	// LLM configuration
	APIKey      string
	BaseURL     string
	Model       string
	ClientType  gochat.ClientType
	Temperature float64
	MaxTokens   int

	// Agent configuration
	SystemPrompt  string
	MaxIterations int
}

// DefaultReactorConfig returns a config with sensible defaults.
// APIKey must be set before use.
func DefaultReactorConfig() ReactorConfig {
	return ReactorConfig{
		Model:         core.DefaultModel,
		ClientType:    gochat.OpenAIClient,
		Temperature:   core.DefaultTemperature,
		MaxTokens:     core.DefaultMaxTokens,
		MaxIterations: core.DefaultMaxSteps,
		SystemPrompt: "You are a helpful AI assistant powered by a T-A-O (Think-Act-Observe) agent system. " +
			"你是一个由 T-A-O（思考-行动-观察）智能体系统驱动的 AI 助手。",
	}
}

// RunResult holds the complete output of a Run invocation.
type RunResult struct {
	Answer                string  `json:"answer" yaml:"answer"`
	Intent                *Intent `json:"intent,omitempty" yaml:"intent,omitempty"`
	Steps                 []Step  `json:"steps,omitempty" yaml:"steps,omitempty"`
	TotalIterations       int     `json:"total_iterations" yaml:"total_iterations"`
	TerminationReason     string  `json:"termination_reason,omitempty" yaml:"termination_reason,omitempty"`
	Confidence            float64 `json:"confidence" yaml:"confidence"`
	ClarificationNeeded   bool    `json:"clarification_needed" yaml:"clarification_needed"`
	ClarificationQuestion string  `json:"clarification_question,omitempty" yaml:"clarification_question,omitempty"`
	TokensUsed            int     `json:"tokens_used,omitempty" yaml:"tokens_used,omitempty"`
	TotalDuration         time.Duration `json:"total_duration_ms,omitempty" yaml:"total_duration_ms,omitempty"`
}

// ReActor is the core interface for the T-A-O reactor.
type ReActor interface {
	// Run executes the full T-A-O loop for a single user input.
	Run(ctx context.Context, input string, history ConversationHistory) (*RunResult, error)
	// Think, Act, Observe, CheckTermination are the individual T-A-O phases.
	Think(ctx *ReactContext) (int, error)
	Act(ctx *ReactContext) error
	Observe(ctx *ReactContext) error
	CheckTermination(ctx *ReactContext) (bool, string)
}

// Reactor is the standard T-A-O reactor implementation.
type Reactor struct {
	config         ReactorConfig
	intentRegistry *IntentRegistry
	toolRegistry   *ToolRegistry
	skillRegistry  core.SkillRegistry
	taskManager    core.TaskManager
	llmClient      gochat.ClientBuilder // pre-configured LLM client builder

	// Context defense (three-layer strategy)
	compactor       core.ContextCompactor // optional: third layer (compact)
	compactorConfig core.CompactorConfig
	tokenEstimator  core.TokenEstimator

	// AskUser tool for interactive clarification (interrupt-resume pattern).
	// The LLM calls ask_user tool when it needs user input; the tool blocks
	// until the external caller calls AskUser().Respond().
	askUser *tools.AskUser

	// AskPermission handles tool authorization (interrupt-resume pattern).
	// High-risk tools require user approval before execution.
	// The external caller calls AskPermission().Respond() to approve/deny.
	askPermission *tools.AskPermission

	// EventBus for streaming agent-level events to external consumers.
	// Shared across main reactor and all subagent tasks.
	eventBus EventBus

	// pendingTasks tracks in-flight subagent tasks (Issue #2: instance-bound, not global)
	pendingTasks   map[string]chan *RunResult
	pendingTasksMu sync.RWMutex
}

// EventBus returns the reactor's event bus for subscribing to agent events.
func (r *Reactor) EventBus() EventBus {
	return r.eventBus
}

// AskUser returns the reactor's ask_user tool for responding to clarification requests.
// The external caller should call .Respond(answer) when the user provides input.
func (r *Reactor) AskUser() *tools.AskUser {
	return r.askUser
}

// SetAskUser sets a custom AskUser tool instance.
func (r *Reactor) SetAskUser(t *tools.AskUser) {
	r.askUser = t
}

// AskPermission returns the reactor's permission checker for responding to authorization requests.
// The external caller should call .Respond(result) when the user approves or denies.
func (r *Reactor) AskPermission() *tools.AskPermission {
	return r.askPermission
}

// SetAskPermission sets a custom AskPermission instance.
func (r *Reactor) SetAskPermission(p *tools.AskPermission) {
	r.askPermission = p
}

// registerPendingTask adds a pending subagent task to this reactor instance.
func (r *Reactor) registerPendingTask(taskID string, ch chan *RunResult) {
	r.pendingTasksMu.Lock()
	defer r.pendingTasksMu.Unlock()
	if r.pendingTasks == nil {
		r.pendingTasks = make(map[string]chan *RunResult)
	}
	r.pendingTasks[taskID] = ch
}

// getPendingTask retrieves the channel for a pending subagent task.
func (r *Reactor) getPendingTask(taskID string) (chan *RunResult, bool) {
	r.pendingTasksMu.RLock()
	defer r.pendingTasksMu.RUnlock()
	ch, ok := r.pendingTasks[taskID]
	return ch, ok
}

// removePendingTask removes a completed pending task.
func (r *Reactor) removePendingTask(taskID string) {
	r.pendingTasksMu.Lock()
	defer r.pendingTasksMu.Unlock()
	delete(r.pendingTasks, taskID)
}

// reactorSetup holds options applied before tool registration.
type reactorSetup struct {
	skipTools        map[string]bool
	skipAllBundled   bool
	extraTools       []core.FuncTool
	securityPolicy   SecurityPolicy
	resultStorage    core.ToolResultStorage
	resultLimits     core.ToolResultLimits
	compactor        core.ContextCompactor
	compactorConfig  core.CompactorConfig
	tokenEstimator   core.TokenEstimator
	eventBus         EventBus
}

// ReactorOption configures a Reactor during creation.
type ReactorOption func(*reactorSetup)

// WithExtraTools adds additional tools to the reactor beyond the bundled ones.
func WithExtraTools(tools ...core.FuncTool) ReactorOption {
	return func(s *reactorSetup) {
		s.extraTools = append(s.extraTools, tools...)
	}
}

// WithoutBundledTools skips registration of all built-in tools (orchestration tools are still registered).
func WithoutBundledTools() ReactorOption {
	return func(s *reactorSetup) {
		s.skipAllBundled = true
	}
}

// WithoutTool skips registration of a specific built-in tool by name.
func WithoutTool(name string) ReactorOption {
	return func(s *reactorSetup) {
		if s.skipTools == nil {
			s.skipTools = make(map[string]bool)
		}
		s.skipTools[name] = true
	}
}

// WithSecurityPolicy sets a custom security policy for tool execution.
// The policy is called before executing a tool; return false to block execution.
func WithSecurityPolicy(policy SecurityPolicy) ReactorOption {
	return func(s *reactorSetup) {
		s.securityPolicy = policy
	}
}

// WithResultStorage enables tool result persistence (second layer defense).
// Large tool results will be saved to disk and only a preview kept in context.
func WithResultStorage(storage core.ToolResultStorage) ReactorOption {
	return func(s *reactorSetup) {
		s.resultStorage = storage
	}
}

// WithResultLimits configures tool result size thresholds (second layer defense).
func WithResultLimits(limits core.ToolResultLimits) ReactorOption {
	return func(s *reactorSetup) {
		s.resultLimits = limits
	}
}

// WithCompactor enables automatic context compaction (third layer defense).
// When the context window approaches its token limit, older messages will be
// summarized/compressed to free space.
func WithCompactor(compactor core.ContextCompactor) ReactorOption {
	return func(s *reactorSetup) {
		s.compactor = compactor
	}
}

// WithCompactorConfig configures the compaction thresholds (third layer defense).
func WithCompactorConfig(config core.CompactorConfig) ReactorOption {
	return func(s *reactorSetup) {
		s.compactorConfig = config
	}
}

// WithTokenEstimator sets a custom token estimator for budget tracking.
func WithTokenEstimator(estimator core.TokenEstimator) ReactorOption {
	return func(s *reactorSetup) {
		s.tokenEstimator = estimator
	}
}

// WithEventBus sets the event bus for streaming agent events.
// If not set, a new InProcessEventBus is created automatically.
func WithEventBus(bus EventBus) ReactorOption {
	return func(s *reactorSetup) {
		s.eventBus = bus
	}
}

// NewReactor creates a new Reactor with the given configuration and options.
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

	// Process options
	setup := &reactorSetup{skipTools: make(map[string]bool)}
	for _, opt := range opts {
		opt(setup)
	}

	r := &Reactor{
		config:          config,
		intentRegistry:  NewIntentRegistry(),
		toolRegistry:    NewToolRegistry(),
		skillRegistry:   NewSkillRegistry(),
		taskManager:     core.NewInMemoryTaskManager(),
		compactorConfig: core.DefaultCompactorConfig(),
		tokenEstimator:  core.NewDefaultTokenEstimator(3.0),
	}

	// Apply event bus (create default if not provided via option)
	if setup.eventBus != nil {
		r.eventBus = setup.eventBus
	} else {
		r.eventBus = NewEventBus()
	}

	// Pre-configure LLM client builder (Issue #13: reuse client across calls)
	r.llmClient = gochat.Client().Config(
		gochat.WithAPIKey(config.APIKey),
		gochat.WithBaseURL(config.BaseURL),
	)

	// Register bundled skills
	RegisterBundledSkills(r.skillRegistry)

	// Register orchestration tools (always registered)
	_ = r.RegisterTool(NewTaskCreateTool(r))
	_ = r.RegisterTool(NewTaskResultTool(r))
	_ = r.RegisterTool(NewTaskListTool(r))

	// Register ask_user tool for interactive clarification (interrupt-resume)
	r.askUser = tools.NewAskUserTool().(*tools.AskUser)
	r.askUser.SetEventEmitter(func(e core.ReactEvent) {
		if r.eventBus != nil {
			r.eventBus.Emit(e)
		}
	})
	_ = r.RegisterTool(r.askUser)

	// Set up tool permission checker for high-risk tool authorization
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

	// Register built-in core tools (unless skipped by options)
	if !setup.skipAllBundled {
		bundledTools := []struct {
			name string
			tool core.FuncTool
		}{
			{"grep", tools.NewGrepTool()},
			{"glob", tools.NewGlobTool()},
			{"bash", tools.NewBashTool()},
			{"web_fetch", tools.NewWebFetchTool()},
			{"todo_write", tools.NewTodoWriteTool()},
			{"todo_read", tools.NewTodoReadTool()},
			{"todo_execute", tools.NewTodoExecuteTool()},
			{"file_edit", tools.NewFileEditTool()},
			{"repl", tools.NewREPLTool()},
			{"read", tools.NewReadTool()},
			{"write", tools.NewWriteTool()},
			{"ls", tools.NewLSTool()},
			{"echo", tools.NewEchoTool()},
			{"calculator", tools.NewCalculatorTool()},
			{"replace", tools.NewReplaceTool()},
			{"cron", tools.NewCronTool()},
		}
		for _, bt := range bundledTools {
			if !setup.skipTools[bt.name] {
				_ = r.RegisterTool(bt.tool)
			}
		}
	}

	// Register extra tools from options
	for _, tool := range setup.extraTools {
		_ = r.RegisterTool(tool)
	}

	// Apply security policy
	if setup.securityPolicy != nil {
		r.toolRegistry.SetSecurityPolicy(setup.securityPolicy)
	}

	// Apply result storage (second layer defense)
	if setup.resultStorage != nil {
		r.toolRegistry.SetResultStorage(setup.resultStorage)
	}
	if setup.resultLimits.MaxResultSizeChars > 0 {
		r.toolRegistry.SetResultLimits(setup.resultLimits)
	}

	// Apply compactor config (third layer defense)
	if setup.compactorConfig.CompactThresholdRatio > 0 {
		r.compactorConfig = setup.compactorConfig
	}
	if setup.compactor != nil {
		r.compactor = setup.compactor
	}
	if setup.tokenEstimator != nil {
		r.tokenEstimator = setup.tokenEstimator
	}

	return r
}

// SkillRegistry returns the reactor's skill registry.
func (r *Reactor) SkillRegistry() core.SkillRegistry {
	return r.skillRegistry
}

// IntentRegistry returns the reactor's intent registry for dynamic intent management.
func (r *Reactor) IntentRegistry() *IntentRegistry {
	return r.intentRegistry
}

// ToolRegistry returns the reactor's tool registry for dynamic tool management.
func (r *Reactor) ToolRegistry() *ToolRegistry {
	return r.toolRegistry
}

// TaskManager returns the reactor's task manager.
func (r *Reactor) TaskManager() core.TaskManager {
	return r.taskManager
}

// RegisterTool is a convenience method to register a core.FuncTool.
func (r *Reactor) RegisterTool(tool core.FuncTool) error {
	return r.toolRegistry.Register(tool)
}

// RegisterIntent is a convenience method to register an intent type.
func (r *Reactor) RegisterIntent(def IntentDefinition) error {
	return r.intentRegistry.Register(def)
}

// callLLMWithHistory makes an LLM call using the reactor's cached client and conversation history.
func (r *Reactor) callLLMWithHistory(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) (*gochatcore.Response, error) {
	builder := r.buildLLMBuilder(systemPrompt, userMessage, history, maxHistoryTurns)
	return builder.GetResponseFor(r.config.ClientType)
}

// callLLMStream makes a streaming LLM call, emitting ThinkingDelta events via EventBus
// as content arrives, then returns the complete response content and token usage.
// The ctx parameter is the ReactContext (not context.Context) used for event emission.
func (r *Reactor) callLLMStream(reactCtx *ReactContext, systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) (string, int, error) {
	builder := r.buildLLMBuilder(systemPrompt, userMessage, history, maxHistoryTurns)

	stream, err := builder.GetStreamFor(r.config.ClientType)
	if err != nil {
		return "", 0, fmt.Errorf("stream LLM call failed: %w", err)
	}
	defer stream.Close()

	var contentBuf strings.Builder
	for stream.Next() {
		event := stream.Event()
		if event.Err != nil {
			return contentBuf.String(), 0, event.Err
		}

		switch event.Type {
		case gochatcore.EventContent:
			contentBuf.WriteString(event.Content)
			// Emit thinking delta for real-time streaming to clients
			reactCtx.EmitEvent(core.ThinkingDelta, event.Content)

		case gochatcore.EventError:
			return contentBuf.String(), 0, event.Err

		case gochatcore.EventDone:
			// Stream completed normally
		}
	}

	// Extract token usage
	tokens := 0
	if usage := stream.Usage(); usage != nil && usage.TotalTokens > 0 {
		tokens = usage.TotalTokens
	}

	return contentBuf.String(), tokens, nil
}

// buildLLMBuilder creates a pre-configured ClientBuilder with system prompt, history, and user message.
// This is shared by both streaming and non-streaming call paths.
func (r *Reactor) buildLLMBuilder(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) gochat.ClientBuilder {
	builder := r.llmClient.
		Model(r.config.Model).
		Temperature(r.config.Temperature).
		MaxTokens(r.config.MaxTokens)

	if r.config.SystemPrompt != "" {
		builder.SystemMessage(r.config.SystemPrompt)
	}
	if systemPrompt != "" {
		builder.SystemMessage(systemPrompt)
	}

	var chatMessages []gochatcore.Message
	messages := history
	if maxHistoryTurns > 0 && len(messages) > maxHistoryTurns {
		messages = messages[len(messages)-maxHistoryTurns:]
	}
	for _, m := range messages {
		chatMessages = append(chatMessages, gochatcore.NewTextMessage(m.Role, m.Content))
	}
	builder.Messages(chatMessages...)
	builder.UserMessage(userMessage)

	return builder
}

// classifyIntent runs intent classification on the user's input.
func (r *Reactor) classifyIntent(ctx *ReactContext) (*Intent, int, error) {
	instructions := BuildIntentPrompt(ctx.Input, "", r.intentRegistry)

	resp, err := r.callLLMWithHistory(instructions, ctx.Input, ctx.ConversationHistory, MaxHistoryTurns)
	if err != nil {
		return nil, 0, fmt.Errorf("intent classification LLM call failed: %w", err)
	}

	tokens := 0
	if resp.Usage != nil && resp.Usage.TotalTokens > 0 {
		tokens = resp.Usage.TotalTokens
	}

	intent, err := parseIntentResponse(resp.Content)
	if err != nil {
		return nil, tokens, fmt.Errorf("intent classification parse failed: %w", err)
	}

	return intent, tokens, nil
}

// parseIntentResponse parses an LLM response into an Intent struct.
func parseIntentResponse(content string) (*Intent, error) {
	content = stripJSONWrappers(content)
	var intent Intent
	if err := json.Unmarshal([]byte(content), &intent); err != nil {
		return nil, fmt.Errorf("failed to parse intent JSON: %w", err)
	}
	return &intent, nil
}

// Think asks the LLM to decide the next action based on the current context.
// Uses streaming to emit ThinkingDelta events in real-time via EventBus.
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
	tools := r.toolRegistry.ToToolInfos()

	// Discover applicable skills based on current context/intent
	skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)
	
	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, tools, skills)

	// Use streaming call — emits ThinkingDelta events as content arrives
	content, tokens, err := r.callLLMStream(ctx, instructions, ctx.Input, ctx.ConversationHistory, MaxHistoryTurns)
	if err != nil {
		return tokens, fmt.Errorf("think LLM call failed: %w", err)
	}

	thought, err := ParseThinkResponse(content)
	if err != nil {
		return tokens, fmt.Errorf("think parse failed: %w", err)
	}

	ctx.LastThought = thought
	return tokens, nil
}

// Act executes the decision from the Think phase.
func (r *Reactor) Act(ctx *ReactContext) error {
	thought := ctx.LastThought
	if thought == nil {
		return fmt.Errorf("act called without a thought")
	}

	start := time.Now()
	action := Action{
		Timestamp: start,
	}

	switch thought.Decision {
	case DecisionAnswer:
		action.Type = ActionTypeAnswer
		action.Result = thought.FinalAnswer
		if action.Result == "" {
			action.Result = thought.Reasoning
		}

	case DecisionClarify:
		action.Type = ActionTypeClarify
		question := thought.ClarificationQuestion
		if question == "" {
			question = "Could you provide more details so I can better assist you?"
		}
		action.Result = question

	case DecisionAct:
		action.Type = ActionTypeToolCall
		action.Target = thought.ActionTarget
		action.Params = thought.ActionParams

		if action.Target == "" {
			action.Type = ActionTypeAnswer
			action.Result = thought.FinalAnswer
			if action.Result == "" {
				action.Result = "Sorry, I cannot determine which tool to use for your request."
			}
			break
		}

		result, duration, err := r.toolRegistry.ExecuteTool(ctx.Ctx(), action.Target, action.Params)
		if err != nil {
			action.Error = err
			action.ErrorMsg = err.Error()
		} else {
			action.Result = result
		}
		action.Duration = duration

	default:
		action.Type = ActionTypeAnswer
		action.Result = thought.FinalAnswer
		if action.Result == "" {
			action.Result = thought.Reasoning
		}
	}

	ctx.LastAction = &action
	return nil
}

// Observe evaluates the result of the Act phase.
func (r *Reactor) Observe(ctx *ReactContext) error {
	action := ctx.LastAction
	if action == nil {
		return fmt.Errorf("observe called without an action")
	}

	var obs *Observation

	switch action.Type {
	case ActionTypeToolCall:
		if action.Error != nil {
			obs = NewErrorObservation(action.Error, false)
			obs.Insights = []string{fmt.Sprintf("Tool %q execution failed", action.Target)}
		} else {
			insights := analyzeActionResult(action.Result)
			obs = NewSuccessObservation(action.Result, insights...)
		}

	case ActionTypeAnswer:
		obs = NewSuccessObservation(action.Result, "direct answer generated")

	case ActionTypeClarify:
		obs = NewSuccessObservation(action.Result, "clarification question generated")

	default:
		obs = NewSuccessObservation(action.Result)
	}

	ctx.LastObservation = obs
	return nil
}

// CheckTermination evaluates whether the T-A-O loop should stop.
func (r *Reactor) CheckTermination(ctx *ReactContext) (bool, string) {
	// Hard constraints
	if ctx.CurrentIteration >= ctx.MaxIterations {
		return true, "reached max iterations"
	}

	if ctx.Ctx().Err() != nil {
		return true, "request cancelled"
	}

	if ctx.LastObservation != nil && ctx.LastObservation.Error != "" && !ctx.LastObservation.ShouldRetry {
		if isToolErrorIrrecoverable(ctx.LastObservation) {
			return true, "tool error: irrecoverable"
		}
	}

	// Soft constraints
	if ctx.LastThought != nil && ctx.LastThought.IsFinal {
		return true, "thinker produced final answer"
	}

	if ctx.LastAction != nil && ctx.LastAction.Type == ActionTypeAnswer {
		return true, "direct answer produced"
	}

	if ctx.LastAction != nil && ctx.LastAction.Type == ActionTypeClarify {
		// NOTE: clarify no longer terminates the loop.
		// The LLM now uses the "ask_user" tool for interactive clarification.
		// When the LLM's Think phase produces DecisionClarify, it means the LLM
		// chose to ask a question directly (not via tool). In that case, we still
		// terminate — but the preferred path is the ask_user tool which allows
		// the T-A-O loop to resume after the user answers.
		// Only terminate if this is a direct clarify (not from ask_user tool).
		if ctx.LastAction.Target == "" {
			return true, "clarification needed"
		}
		// If Target is set (ask_user tool result), continue the loop
	}

	if isResultConverged(ctx.History) {
		return true, "result converged"
	}

	if isDuplicateAction(ctx.History) {
		return true, "duplicate action detected"
	}

	return false, ""
}

// Run executes the full T-A-O loop for a single user input.
// This is the main entry point for using the reactor.
func (r *Reactor) Run(ctx context.Context, input string, history ConversationHistory) (*RunResult, error) {
	reactCtx := NewReactContext(ctx, input, history, r.config.MaxIterations)

	// Inject event emission callback into context.
	// All T-A-O phases will use reactCtx.EmitEvent() to publish to EventBus.
	if r.eventBus != nil {
		reactCtx.emitEvent = r.eventBus.Emit
	}

	totalTokens := 0
	runStart := time.Now()

	// Phase 1: Classify intent
	intent, tokens, err := r.classifyIntent(reactCtx)
	if err != nil {
		reactCtx.EmitEvent(core.Error, fmt.Sprintf("intent classification: %v", err))
		return nil, fmt.Errorf("intent classification: %w", err)
	}
	totalTokens += tokens
	reactCtx.Intent = intent

	// Apply confidence threshold
	ApplyConfidenceThreshold(intent, 0)

	// Early return for clarification needed from intent
	if intent.RequiresClarification {
		reactCtx.EmitEvent(core.ClarifyNeeded, intent.ClarificationQuestion)
		return &RunResult{
			Intent:                intent,
			ClarificationNeeded:   true,
			ClarificationQuestion: intent.ClarificationQuestion,
			Confidence:            intent.Confidence,
			TokensUsed:            totalTokens,
		}, nil
	}

	// Phase 2: T-A-O loop
	for reactCtx.CurrentIteration < reactCtx.MaxIterations {
		// Check termination before each cycle
		if terminated, reason := r.CheckTermination(reactCtx); terminated {
			reactCtx.IsTerminated = true
			reactCtx.TerminationReason = reason
			break
		}

		cycleStart := time.Now()

		// Reset per-message character counter at the start of each cycle
		r.toolRegistry.ResetMessageCharCounter()

		// Think
		tokens, err := r.Think(reactCtx)
		totalTokens += tokens
		if err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("think error: %v", err)
			reactCtx.EmitEvent(core.Error, reactCtx.TerminationReason)
			break
		}
		reactCtx.EmitEvent(core.ThinkingDone, reactCtx.LastThought)

		// Act
		if err := r.Act(reactCtx); err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("act error: %v", err)
			reactCtx.EmitEvent(core.Error, reactCtx.TerminationReason)
			break
		}

		// Emit action events
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

		// Observe
		if err := r.Observe(reactCtx); err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("observe error: %v", err)
			reactCtx.EmitEvent(core.Error, reactCtx.TerminationReason)
			break
		}
		reactCtx.EmitEvent(core.ObservationDone, reactCtx.LastObservation)

		// === Third Layer Defense: Context Compaction ===
		r.maybeCompact(reactCtx)

		// Record step
		step := Step{
			Iteration:   reactCtx.CurrentIteration + 1,
			Thought:     *reactCtx.LastThought,
			Action:      *reactCtx.LastAction,
			Observation: *reactCtx.LastObservation,
			Timestamp:   time.Now(),
			Duration:    time.Since(cycleStart),
		}
		reactCtx.AppendHistory(step)

		// Emit cycle end event
		reactCtx.EmitEvent(core.CycleEnd, core.CycleInfo{
			Iteration:        reactCtx.CurrentIteration + 1,
			Duration:         time.Since(cycleStart),
		})

		// Feed back to conversation history to maintain context
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
		reactCtx.AddMessage("assistant", stepSummary.String())

		reactCtx.CurrentIteration++
	}

	// Phase 3: Build result
	result := &RunResult{
		Intent:            intent,
		Steps:             reactCtx.History,
		TotalIterations:   reactCtx.CurrentIteration,
		TerminationReason: reactCtx.TerminationReason,
		Confidence:        intent.Confidence,
		TokensUsed:        totalTokens,
	}

	// Extract the final answer from the last action or thought
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

	// Emit final answer event
	if result.Answer != "" {
		reactCtx.EmitEvent(core.FinalAnswer, result.Answer)
	}

	// Build and emit execution summary
	totalDuration := time.Since(runStart)
	result.TotalDuration = totalDuration

	summary := core.ExecutionSummaryData{
		TotalIterations:   result.TotalIterations,
		TotalDuration:     totalDuration,
		TokensUsed:        totalTokens,
		TerminationReason: result.TerminationReason,
	}
	// Aggregate tool usage from steps
	seenTools := make(map[string]bool)
	for _, step := range reactCtx.History {
		if step.Action.Type == ActionTypeToolCall && step.Action.Target != "" {
			summary.ToolCalls++
			if !seenTools[step.Action.Target] {
				seenTools[step.Action.Target] = true
				summary.ToolsUsed = append(summary.ToolsUsed, step.Action.Target)
			}
		}
	}
	reactCtx.EmitEvent(core.ExecutionSummary, summary)

	return result, nil
}

// --- Termination helper functions ---

// isToolErrorIrrecoverable checks if a tool error cannot be recovered by retry.
func isToolErrorIrrecoverable(obs *Observation) bool {
	if obs == nil || obs.Error == "" {
		return false
	}
	irrecoverablePatterns := []string{
		"not found",
		"permission denied",
		"unauthorized",
		"invalid api key",
		"authentication",
	}
	lower := strings.ToLower(obs.Error)
	for _, p := range irrecoverablePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func isResultConverged(history []Step) bool {
	if len(history) < 3 {
		return false
	}
	last3 := history[len(history)-3:]
	if last3[0].Action.Result == "" || last3[1].Action.Result == "" || last3[2].Action.Result == "" {
		return false
	}
	return last3[0].Action.Result == last3[1].Action.Result && last3[1].Action.Result == last3[2].Action.Result
}

func isDuplicateAction(history []Step) bool {
	if len(history) < 2 {
		return false
	}
	last := history[len(history)-1]
	prev := history[len(history)-2]
	if last.Action.Type != ActionTypeToolCall || prev.Action.Type != ActionTypeToolCall {
		return false
	}
	return last.Action.Target == prev.Action.Target && last.Action.Result == prev.Action.Result
}

// analyzeActionResult generates insights from a tool execution result.
func analyzeActionResult(result string) []string {
	var insights []string
	if len(result) > 1000 {
		insights = append(insights, "large result truncated for context")
	}
	if strings.Contains(strings.ToLower(result), "error") {
		insights = append(insights, "result may contain error information")
	}
	return insights
}

// --- Context Defense: Third Layer (Compact) ---

// maybeCompact checks whether the conversation history has exceeded the context budget
// and applies micro-compact or full compact (via LLM) as appropriate.
// This is called after each T-A-O cycle's observation phase.
func (r *Reactor) maybeCompact(ctx *ReactContext) {
	estimateFn := func(s string) int {
		return r.tokenEstimator.Estimate(s)
	}

	// Calculate total token usage across conversation history
	var totalTokens int64
	for _, m := range ctx.ConversationHistory {
		totalTokens += int64(estimateFn(m.Content))
	}

	budget := core.CalculateBudget(
		int64(r.config.MaxTokens),
		totalTokens,
		r.compactorConfig.CompactThresholdRatio,
	)

	if !budget.NeedCompact {
		return
	}

	// Decide between micro-compact and full compact
	if budget.UsageRatio >= r.compactorConfig.MicroCompactThreshold &&
		(r.compactor == nil || budget.UsageRatio < r.compactorConfig.CompactThresholdRatio) {
		// Micro-compact: fast, non-LLM, just truncate large messages
		targetTokens := int64(float64(r.config.MaxTokens) * r.compactorConfig.MicroCompactThreshold)
		ctx.ConversationHistory = core.MicroCompact(
			ctx.ConversationHistory, estimateFn, targetTokens,
		)
		return
	}

	// Full compact: use LLM to summarize
	if r.compactor != nil {
		req := core.CompactRequest{
			Messages:     ctx.ConversationHistory,
			PreserveLastN: r.compactorConfig.PreserveLastN,
			MaxTokens:    int64(r.config.MaxTokens),
		}
		compacted, err := r.compactor.Compact(ctx.Ctx(), req)
		if err == nil && compacted != nil {
			// Insert a boundary message
			boundary := core.SummaryMessage(
				len(ctx.ConversationHistory),
				len(compacted.CompactedMessages),
				fmt.Sprintf("context reduced from %d to %d tokens",
					compacted.OriginalTokenCount, compacted.CompactedTokenCount),
			)
			ctx.ConversationHistory = compacted.CompactedMessages
			ctx.ConversationHistory = append(
				[]core.Message{boundary},
				ctx.ConversationHistory...,
			)
		} else {
			// Fallback to micro-compact if LLM compaction fails
			targetTokens := int64(float64(r.config.MaxTokens) * 0.6)
			ctx.ConversationHistory = core.MicroCompact(
				ctx.ConversationHistory, estimateFn, targetTokens,
			)
		}
	} else {
		// No compactor configured, use micro-compact
		targetTokens := int64(float64(r.config.MaxTokens) * 0.6)
		ctx.ConversationHistory = core.MicroCompact(
			ctx.ConversationHistory, estimateFn, targetTokens,
		)
	}
}
