package goreact

import (
	"context"
	"fmt"
	"sync"

	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/reactor"
)

// ---------------------------------------------------------------------------
// Default presets
// ---------------------------------------------------------------------------

// DefaultModel returns a ModelConfig pre-configured for a fast, cost-effective model.
// The default uses qwen3.5-flash which provides excellent performance-to-cost ratio.
// Override individual fields as needed (e.g., change BaseURL for a compatible API).
func DefaultModel() *core.ModelConfig {
	return &core.ModelConfig{
		Name:        "qwen3.5-flash",
		Description: "Quick and cost-effective model for general-purpose tasks",
		MaxTokens:   8192,
	}
}

// DefaultConfig returns an AgentConfig with sensible defaults for a general-purpose agent.
func DefaultConfig() *core.AgentConfig {
	return &core.AgentConfig{
		Name:        "default-agent",
		Domain:      "general",
		Description: "A general-purpose AI agent powered by GoReAct",
	}
}

// ---------------------------------------------------------------------------
// Agent — top-level facade
// ---------------------------------------------------------------------------

// Agent is the top-level facade for interacting with the ReAct agent system.
// Users should interact exclusively through this type; knowledge of the internal
// Reactor engine is NOT required.
//
// Quick start:
//
//	agent := goreact.DefaultAgent("your-api-key")
//	answer, err := agent.Ask("Hello!")
type Agent struct {
	config       *core.AgentConfig
	model        *core.ModelConfig
	memory       core.Memory
	reactor      *reactor.Reactor
	eventBus     reactor.EventBus
	lastResult   *Result
	scheduler    *core.CronScheduler
	sessionStore core.SessionStore

	interruptMu sync.Mutex
	cancelFunc  context.CancelFunc
	isRunning   bool
	snapshot    *reactor.RunSnapshot
}

// ---------------------------------------------------------------------------
// Result — what developers care about after a call
// ---------------------------------------------------------------------------

// Result holds the outcome of an Ask or AskStream call.
// Developers can query token consumption, iterations, tool usage, etc.
type Result struct {
	Answer    string `json:"answer"`
	Tokens    int    `json:"tokens"`
	Duration  string `json:"duration,omitempty"` // human-readable, e.g. "1.23s"
	Steps     int    `json:"steps"`              // total T-A-O iterations
	ToolsUsed int    `json:"tools_used"`         // number of tool invocations
}

// ---------------------------------------------------------------------------
// AgentOption — functional options for NewAgent
// ---------------------------------------------------------------------------
//
// Architecture: WithConfig and WithModel are the two primary options that define
// an Agent's identity and capabilities. All other options are supplementary.
// SystemPrompt belongs to AgentConfig, NOT as a standalone option.
//
// Only options with real extension value for developers are exposed here.
// Internal defense mechanisms use sensible defaults and are NOT exposed to avoid
// unnecessary type leakage. Advanced users can access the internal Reactor
// via Agent.Reactor() for fine-grained control.
// ---------------------------------------------------------------------------

// agentSetup holds all optional configuration collected from AgentOptions.
type agentSetup struct {
	config *core.AgentConfig
	model  *core.ModelConfig

	memory        core.Memory
	sessionID     string
	sessionTokens int64

	// Tools & Skills
	extraTools     []core.FuncTool
	skipAllBundled bool
	skipToolNames  map[string]bool
	skillDirs      []string

	// Event streaming & security
	eventBus       reactor.EventBus
	securityPolicy core.SecurityPolicy

	// Scheduler for cron-based scheduled tasks
	scheduler *core.CronScheduler

	// Session store for conversation persistence (sliding window backing store)
	sessionStore core.SessionStore
}

// AgentOption configures an Agent during creation via NewAgent.
type AgentOption func(*agentSetup)

// WithConfig sets the AgentConfig that defines the agent's identity:
// name, domain, description, and system prompt.
// If not set, DefaultConfig() is used.
//
//	config := &core.AgentConfig{
//	    Name:        "code-reviewer",
//	    Domain:      "software-engineering",
//	    Description: "A code review assistant",
//	    SystemPrompt: "You are a senior code reviewer...",
//	}
func WithConfig(config *core.AgentConfig) AgentOption {
	return func(s *agentSetup) {
		s.config = config
	}
}

// WithModel sets the ModelConfig that defines the LLM backend:
// model name, API key, base URL, etc.
// If not set, DefaultModel() is used (API key must be set separately or via this option).
//
//	model := goreact.DefaultModel()
//	model.APIKey = "your-api-key"
//	model.BaseURL = "https://api.example.com/v1"
func WithModel(model *core.ModelConfig) AgentOption {
	return func(s *agentSetup) {
		s.model = model
	}
}

// WithMemory sets a Memory implementation for knowledge retrieval and hallucination suppression.
// If not set, the agent operates without memory augmentation.
func WithMemory(mem core.Memory) AgentOption {
	return func(s *agentSetup) {
		s.memory = mem
	}
}

// WithSession starts a conversation session immediately upon creation.
// sessionID identifies the session; maxTokens sets the token budget (0 = default 8192).
func WithSession(sessionID string, maxTokens int64) AgentOption {
	return func(s *agentSetup) {
		s.sessionID = sessionID
		s.sessionTokens = maxTokens
	}
}

// WithExtraTools adds custom tools to the agent.
// Each tool must implement core.FuncTool (Info() + Execute()).
func WithExtraTools(tools ...core.FuncTool) AgentOption {
	return func(s *agentSetup) {
		s.extraTools = append(s.extraTools, tools...)
	}
}

// WithoutBundledTools skips registration of all built-in tools (orchestration tools are still registered).
func WithoutBundledTools() AgentOption {
	return func(s *agentSetup) {
		s.skipAllBundled = true
	}
}

// WithoutTool skips registration of a specific built-in tool by name.
func WithoutTool(name string) AgentOption {
	return func(s *agentSetup) {
		if s.skipToolNames == nil {
			s.skipToolNames = make(map[string]bool)
		}
		s.skipToolNames[name] = true
	}
}

// WithSkillDir loads additional skills from the given directory.
// Each subdirectory should contain a SKILL.md file defining the skill.
// May be called multiple times to load from multiple directories.
func WithSkillDir(dir string) AgentOption {
	return func(s *agentSetup) {
		s.skillDirs = append(s.skillDirs, dir)
	}
}

// WithEventBus sets the event bus for streaming agent-level events (thinking, actions, etc.).
// If not set, an in-process bus is created automatically.
func WithEventBus(bus reactor.EventBus) AgentOption {
	return func(s *agentSetup) {
		s.eventBus = bus
	}
}

// WithSecurityPolicy sets a custom security policy for tool execution.
// The policy is a function that receives (toolName, securityLevel) and returns
// true to allow or false to block execution.
//
//	goreact.WithSecurityPolicy(func(name string, level core.SecurityLevel) bool {
//	    return name != "bash" // block bash tool
//	})
func WithSecurityPolicy(policy core.SecurityPolicy) AgentOption {
	return func(s *agentSetup) {
		s.securityPolicy = policy
	}
}

// WithScheduler enables cron-based scheduled task management.
// The scheduler allows the agent to register, list, and manage cron-based tasks.
// When a scheduled task fires, the provided callback is invoked to trigger agent execution.
//
// Usage:
//
//	scheduler := core.NewCronScheduler()
//	agent := goreact.NewAgent(
//	    goreact.WithModel(model),
//	    goreact.WithScheduler(scheduler),
//	)
//	scheduler.Start(context.Background())
//	defer scheduler.Stop()
func WithScheduler(scheduler *core.CronScheduler) AgentOption {
	return func(s *agentSetup) {
		s.scheduler = scheduler
	}
}

// WithSessionStore sets a SessionStore for conversation persistence.
// If not set, NewAgent falls back to MemorySessionStore (in-memory, no persistence).
func WithSessionStore(store core.SessionStore) AgentOption {
	return func(s *agentSetup) {
		s.sessionStore = store
	}
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// DefaultAgent creates a ready-to-use Agent with sensible defaults.
// It only requires an API key to start working. The agent uses qwen3.5-flash
// by default with a standard T-A-O reactor and a session context window of 8192 tokens.
//
// Usage:
//
//	agent := goreact.DefaultAgent("your-api-key")
//	answer, err := agent.Ask("Hello, how are you?")
func DefaultAgent(apiKey string) (*Agent, error) {
	model := DefaultModel()
	model.APIKey = apiKey

	return NewAgent(
		WithModel(model),
		WithSession("default", 8192),
	)
}

// NewAgent creates an Agent configured entirely through options.
// WithConfig and WithModel define the agent's core identity and LLM backend;
// all other options are supplementary.
//
// Minimal usage:
//
//	agent := goreact.NewAgent(
//	    goreact.WithConfig(&core.AgentConfig{
//	        Name:        "my-agent",
//	        Domain:      "general",
//	        SystemPrompt: "You are a helpful assistant.",
//	    }),
//	    goreact.WithModel(model),
//	)
//
// One-liner with defaults:
//
//	model := goreact.DefaultModel()
//	model.APIKey = "your-api-key"
//	agent := goreact.NewAgent(goreact.WithModel(model))
//
// Full-featured:
//
//	agent := goreact.NewAgent(
//	    goreact.WithConfig(config),
//	    goreact.WithModel(model),
//	    goreact.WithMemory(mem),
//	    goreact.WithExtraTools(myTool),
//	    goreact.WithSkillDir("/path/to/skills"),
//	    goreact.WithSession("s1", 16384),
//	    goreact.WithSecurityPolicy(policy),
//	)
func NewAgent(opts ...AgentOption) (*Agent, error) {
	setup := &agentSetup{}
	for _, opt := range opts {
		opt(setup)
	}

	// Apply defaults if not provided
	if setup.config == nil {
		setup.config = DefaultConfig()
	}
	if setup.model == nil {
		setup.model = DefaultModel()
	}

	config := setup.config
	model := setup.model

	// Resolve system prompt: AgentConfig.SystemPrompt > template render > fallback
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		if rendered, err := reactor.RenderDefaultSystemPrompt(
			config.Name, config.Domain, config.Description,
		); err == nil {
			systemPrompt = rendered
		} else {
			systemPrompt = "You are a helpful AI assistant powered by a T-A-O agent system."
		}
	}
	config.SystemPrompt = systemPrompt

	// Validate required fields
	if model.APIKey == "" {
		return nil, fmt.Errorf("goreact: ModelConfig.APIKey is required, got empty. Use goreact.WithModel(model) where model.APIKey is set")
	}

	// Build reactor options — only forward options with real extension value
	var reactorOpts []reactor.ReactorOption
	if setup.memory != nil {
		reactorOpts = append(reactorOpts, reactor.WithMemory(setup.memory))
	}
	if setup.eventBus != nil {
		reactorOpts = append(reactorOpts, reactor.WithEventBus(setup.eventBus))
	}
	if len(setup.extraTools) > 0 {
		reactorOpts = append(reactorOpts, reactor.WithExtraTools(setup.extraTools...))
	}
	if setup.skipAllBundled {
		reactorOpts = append(reactorOpts, reactor.WithoutBundledTools())
	}
	for name := range setup.skipToolNames {
		reactorOpts = append(reactorOpts, reactor.WithoutTool(name))
	}
	if setup.securityPolicy != nil {
		reactorOpts = append(reactorOpts, reactor.WithSecurityPolicy(setup.securityPolicy))
	}
	if setup.scheduler != nil {
		reactorOpts = append(reactorOpts, reactor.WithScheduler(setup.scheduler))
	}
	for _, dir := range setup.skillDirs {
		reactorOpts = append(reactorOpts, reactor.WithSkillDir(dir))
	}

	if setup.sessionStore == nil {
		setup.sessionStore = core.NewMemorySessionStore()
	}
	reactorOpts = append(reactorOpts, reactor.WithSessionStore(setup.sessionStore))

	// Build ReactorConfig from ModelConfig
	reactorConfig := reactor.ReactorConfig{
		APIKey:       model.APIKey,
		BaseURL:      model.BaseURL,
		Model:        model.Name,
		SystemPrompt: systemPrompt,
		IsLocal:      model.IsLocal,
	}

	r := reactor.NewReactor(reactorConfig, reactorOpts...)

	a := &Agent{
		config:       config,
		model:        model,
		memory:       setup.memory,
		reactor:      r,
		eventBus:     r.EventBus(),
		scheduler:    setup.scheduler,
		sessionStore: setup.sessionStore,
	}

	if setup.sessionID != "" {
		maxTokens := setup.sessionTokens
		if maxTokens <= 0 {
			maxTokens = 8192
		}
		a.reactor.SetContextWindow(core.NewContextWindow(setup.sessionID, maxTokens))
	}

	return a, nil
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

// Config returns the agent's configuration.
func (a *Agent) Config() *core.AgentConfig {
	return a.config
}

// Model returns the agent's model configuration.
func (a *Agent) Model() *core.ModelConfig {
	return a.model
}

func (a *Agent) Name() string {
	return a.config.Name
}

func (a *Agent) Domain() string {
	return a.config.Domain
}

func (a *Agent) Description() string {
	return a.config.Description
}

// Memory returns the agent's memory instance, or nil if not configured.
func (a *Agent) Memory() core.Memory {
	return a.memory
}

// ContextWindow returns the agent's context window (delegated to Reactor).
func (a *Agent) ContextWindow() *core.ContextWindow {
	return a.reactor.ContextWindow()
}

// Reactor returns the internal Reactor for advanced use cases.
// Most users should NOT need this; it is exposed for scenarios that require
// direct Reactor access (e.g., registering tools at runtime, accessing internal
// registries, or fine-tuning defense mechanisms).
func (a *Agent) Reactor() *reactor.Reactor {
	return a.reactor
}

// Scheduler returns the agent's CronScheduler, or nil if not configured.
// Use WithScheduler() during agent creation to enable scheduled tasks.
func (a *Agent) Scheduler() *core.CronScheduler {
	return a.scheduler
}

// SessionStore returns the agent's session store, or nil if not configured.
func (a *Agent) SessionStore() core.SessionStore {
	return a.sessionStore
}

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

// NewSession starts a new conversation session, replacing any existing one.
// The previous context is discarded.
func (a *Agent) NewSession(sessionID string, maxTokens int64) {
	a.reactor.SetContextWindow(core.NewContextWindow(sessionID, maxTokens))
}

// SessionID returns the current session ID, or empty string if no session.
func (a *Agent) SessionID() string {
	if cw := a.reactor.ContextWindow(); cw != nil {
		return cw.SessionID
	}
	return ""
}

// ---------------------------------------------------------------------------
// Conversation — the two core methods developers use
// ---------------------------------------------------------------------------

// Ask sends a question to the Agent and returns a Result with the answer,
// token usage, and execution statistics.
//
// If a ContextWindow is active, the conversation history is automatically
// managed: user input and assistant response are appended to the window,
// and the history is pruned if it exceeds the token budget.
//
// Usage:
//
//	agent := goreact.DefaultAgent("your-api-key")
//	result, err := agent.Ask("What is the capital of France?")
//	if err != nil { ... }
//	fmt.Printf("Answer: %s\nTokens: %d\n", result.Answer, result.Tokens)
func (a *Agent) Ask(question string) (*Result, error) {
	return a.AskWithContext(context.TODO(), question)
}

// AskWithContext is like Ask but accepts an explicit context.Context for cancellation
// and timeout control. Most users should use Ask; this is for advanced scenarios
// such as request-scoped deadlines or graceful shutdown.
func (a *Agent) AskWithContext(ctx context.Context, question string) (*Result, error) {
	runCtx, cancel := context.WithCancel(ctx)

	a.interruptMu.Lock()
	a.cancelFunc = cancel
	a.isRunning = true
	a.interruptMu.Unlock()

	defer func() {
		cancel()
		a.interruptMu.Lock()
		a.cancelFunc = nil
		a.isRunning = false
		a.interruptMu.Unlock()
	}()

	runResult, err := a.reactor.Run(runCtx, question, nil)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Answer:    runResult.Answer,
		Tokens:    runResult.TokensUsed,
		Duration:  runResult.TotalDuration.String(),
		Steps:     runResult.TotalIterations,
		ToolsUsed: len(runResult.Steps),
	}
	a.lastResult = result
	return result, nil
}

// AskStream sends a question and returns a channel that streams text fragments
// as they are produced by the reactor. The channel is closed when the reactor
// finishes. Use Events() to receive structured event data (thinking, tool calls, etc.)
// alongside the text stream.
//
// Usage:
//
//	ch, cancel, err := agent.AskStream("Explain quantum computing")
//	if err != nil { ... }
//	defer cancel()
//	for text := range ch {
//	    fmt.Print(text)
//	}
//	result := agent.LastResult()
//	fmt.Printf("\nTokens: %d\n", result.Tokens)
func (a *Agent) AskStream(question string) (<-chan string, func(), error) {
	return a.AskStreamWithContext(context.TODO(), question)
}

// AskStreamWithContext is like AskStream but accepts an explicit context.Context.
func (a *Agent) AskStreamWithContext(ctx context.Context, question string) (<-chan string, func(), error) {
	// Subscribe to thinking_delta events for text streaming
	ch, cancel := a.eventBus.SubscribeFiltered(func(e core.ReactEvent) bool {
		return e.Type == core.ThinkingDelta || e.Type == core.FinalAnswer
	})

	textCh := make(chan string, 256)

	// Run reactor in background, forwarding text fragments
	go func() {
		defer cancel()
		defer close(textCh)

		result, err := a.AskWithContext(ctx, question)
		if err != nil {
			textCh <- fmt.Sprintf("[error] %v", err)
			return
		}
		// If nothing was streamed (e.g. short answer without streaming), push final answer
		select {
		case <-ctx.Done():
			return
		default:
			if result.Answer != "" {
				textCh <- result.Answer
			}
		}
	}()

	// Forward event text to the output channel
	go func() {
		for event := range ch {
			switch data := event.Data.(type) {
			case string:
				select {
				case textCh <- data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return textCh, func() { cancel() }, nil
}

// ---------------------------------------------------------------------------
// Interruption — Cancel, Pause, Resume
// ---------------------------------------------------------------------------

// Cancel interrupts the currently running Ask/AskStream call.
// The Run will return with partial results and TerminationReason "request cancelled".
// If no Run is in progress, this is a no-op.
// The state is NOT saved — use Pause() if you want to resume later.
func (a *Agent) Cancel() {
	a.interruptMu.Lock()
	defer a.interruptMu.Unlock()
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
}

// Pause interrupts the currently running Ask/AskStream call and saves the
// execution state (snapshot) so it can be resumed later with Resume().
// The Run returns with partial results.
// If no Run is in progress, this is a no-op.
// Calling Pause() replaces any previously saved snapshot.
func (a *Agent) Pause() {
	a.interruptMu.Lock()
	defer a.interruptMu.Unlock()
	if !a.isRunning {
		return
	}
	// Signal the reactor to save a snapshot when the Run loop detects cancellation
	a.reactor.SetPauseRequested()
	// Cancel the context to interrupt the Run
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
}

// Resume continues a previously paused task. If newInput is non-empty, it is
// appended to the conversation history before resuming (useful for redirect scenarios).
// Returns the result of the resumed execution.
// Returns an error if no snapshot is available (nothing was paused).
func (a *Agent) Resume(newInput ...string) (*Result, error) {
	a.interruptMu.Lock()
	snap := a.snapshot
	a.interruptMu.Unlock()

	if snap == nil {
		// Try the reactor-level snapshot holder (set by runTAOLoop on pause)
		snap = a.reactor.ConsumeSnapshot()
	}
	if snap == nil {
		return nil, fmt.Errorf("goreact: cannot Resume — no paused snapshot available. Call Pause() first while a Run is in progress")
	}

	input := ""
	if len(newInput) > 0 {
		input = newInput[0]
	}

	ctx := context.Background()

	runResult, err := a.reactor.RunFromSnapshot(ctx, snap, input)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Answer:    runResult.Answer,
		Tokens:    runResult.TokensUsed,
		Duration:  runResult.TotalDuration.String(),
		Steps:     runResult.TotalIterations,
		ToolsUsed: len(runResult.Steps),
	}
	a.lastResult = result

	a.interruptMu.Lock()
	a.snapshot = nil
	a.interruptMu.Unlock()

	return result, nil
}

// Snapshot returns the current saved RunSnapshot (from Pause), or nil if none.
func (a *Agent) Snapshot() *reactor.RunSnapshot {
	a.interruptMu.Lock()
	defer a.interruptMu.Unlock()
	if a.snapshot != nil {
		return a.snapshot
	}
	return a.reactor.PeekSnapshot()
}

// IsRunning returns true if an Ask/AskStream call is currently in progress.
func (a *Agent) IsRunning() bool {
	a.interruptMu.Lock()
	defer a.interruptMu.Unlock()
	return a.isRunning
}

// LastResult returns the Result from the most recent Ask/AskStream call, or nil.
// This is the primary way to inspect token usage, step count, and duration
// after a call completes.
func (a *Agent) LastResult() *Result {
	return a.lastResult
}

// Events subscribes to all agent events (thinking, tool calls, errors, etc.)
// and returns a read-only channel and a cancel function.
//
// Call this BEFORE Ask/AskStream to receive events from the next call.
// Each event is a core.ReactEvent with a Type field for routing:
//
//   - ThinkingDelta: text fragment (streaming thought)
//   - ThinkingDone: completed thought
//   - ActionStart / ActionResult: tool execution
//   - FinalAnswer: the complete answer
//   - Error: reactor-level errors
//   - ExecutionSummary: iteration count, tool usage, token stats
//
// Usage:
//
//	ch, cancel := agent.Events()
//	defer cancel()
//	result, _ := agent.Ask("Summarize this article")
//	for event := range ch {
//	    fmt.Printf("[%s] %v\n", event.Type, event.Data)
//	}
//	fmt.Printf("Total tokens: %d\n", result.Tokens)
func (a *Agent) Events() (<-chan core.ReactEvent, func()) {
	return a.eventBus.Subscribe()
}

// EventsFiltered subscribes to events matching the given filter.
// This is useful when you only care about specific event types.
//
// Usage — only receive thinking and tool action events:
//
//	ch, cancel := agent.EventsFiltered(func(e core.ReactEvent) bool {
//	    return e.Type == core.ThinkingDelta || e.Type == core.ActionStart
//	})
//	defer cancel()
func (a *Agent) EventsFiltered(filter func(core.ReactEvent) bool) (<-chan core.ReactEvent, func()) {
	return a.eventBus.SubscribeFiltered(filter)
}
