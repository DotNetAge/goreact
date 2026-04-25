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

	// IsLocal indicates whether this reactor uses a local model.
	// When true, SubAgent spawning is forced to run synchronously (serial execution)
	// instead of the default asynchronous goroutine mode. This prevents concurrent
	// LLM calls that local models typically cannot handle.
	// SubAgents with their own model override (IsLocal=false) can still run async.
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

// ReActor is the core interface for the T-A-O reactor.
type ReActor interface {
	// Run executes the full T-A-O loop for a single user input.
	Run(ctx context.Context, input string, history ConversationHistory) (*RunResult, error)
	// RunFromSnapshot resumes a T-A-O execution from a previously saved snapshot.
	RunFromSnapshot(ctx context.Context, snapshot *RunSnapshot, newInput string) (*RunResult, error)
	// Think, Act, Observe, CheckTermination are the individual T-A-O phases.
	Think(ctx *ReactContext) (int, error)
	Act(ctx *ReactContext) error
	Observe(ctx *ReactContext) error
	CheckTermination(ctx *ReactContext) (bool, string)
}

// Reactor is the standard T-A-O reactor implementation.
type Reactor struct {
	config         ReactorConfig
	intentRegistry IntentRegistry
	toolRegistry   core.ToolRegistryInterface
	skillRegistry  core.SkillRegistry
	taskManager    core.TaskManager
	llmClient      gochat.ClientBuilder // pre-configured LLM client builder

	// Memory for knowledge retrieval (suppresses hallucination via RAG).
	// If nil, Think phase operates without memory augmentation.
	memory core.Memory

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

	// MessageBus enables inter-agent communication within teams.
	// Shared across main reactor and all subagent tasks so that
	// team members can send/receive messages via channel-based mailboxes.
	messageBus *core.AgentMessageBus

	// pendingTasks tracks in-flight subagent tasks (Issue #2: instance-bound, not global)
	pendingTasks   map[string]chan any
	pendingTasksMu sync.RWMutex

	// scheduler manages cron-based scheduled tasks.
	// When configured, the cron tool can register/list/remove scheduled tasks
	// and the scheduler will trigger agent execution at the specified times.
	scheduler *core.CronScheduler

	// mockLLM replaces the real LLM call when set (non-nil).
	// Used for deterministic end-to-end testing without real API calls.
	// When set, callLLMWithHistory delegates to this function instead of the gochat client.
	mockLLM func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error)

	// pauseRequested is set to true by Pause() to indicate the current Run should
	// save its state before returning. The runTAOLoop checks this after detecting
	// context cancellation and saves a snapshot if true.
	pauseRequested bool
	pauseMu        sync.Mutex

	// snapshotHolder stores the latest snapshot from a paused Run.
	// Instance-scoped (not global) so multiple Reactor instances are safe for concurrent use.
	snapshotHolder struct {
		sync.RWMutex
		snap *RunSnapshot
	}
}

// EventBus returns the reactor's event bus for subscribing to agent events.
func (r *Reactor) EventBus() EventBus {
	return r.eventBus
}

// MessageBus returns the reactor's agent message bus for team communication.
func (r *Reactor) MessageBus() *core.AgentMessageBus {
	return r.messageBus
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

// Memory returns the reactor's memory instance for external access.
// Returns nil if no memory was configured.
func (r *Reactor) Memory() core.Memory {
	return r.memory
}

// SetPauseRequested signals the reactor to save a snapshot when the current Run
// is cancelled. This is used by Agent.Pause() to enable resumable interruption.
func (r *Reactor) SetPauseRequested() {
	r.pauseMu.Lock()
	defer r.pauseMu.Unlock()
	r.pauseRequested = true
}

// TakeSnapshot captures the current execution state as a RunSnapshot.
// If a Run is in progress, this returns the state at the moment of capture.
// Returns nil if no snapshot is available (no Run has started or no pause was requested).
func (r *Reactor) TakeSnapshot() *RunSnapshot {
	r.pauseMu.Lock()
	defer r.pauseMu.Unlock()
	r.pauseRequested = false
	return nil // The actual snapshot is set by runTAOLoop when pause is detected
}

// setSnapshot stores a RunSnapshot in this reactor instance for later retrieval.
func (r *Reactor) setSnapshot(snap *RunSnapshot) {
	r.snapshotHolder.Lock()
	defer r.snapshotHolder.Unlock()
	r.snapshotHolder.snap = snap
}

// getSnapshot retrieves and consumes the stored RunSnapshot from this reactor instance.
func (r *Reactor) getSnapshot() *RunSnapshot {
	r.snapshotHolder.Lock()
	defer r.snapshotHolder.Unlock()
	snap := r.snapshotHolder.snap
	r.snapshotHolder.snap = nil
	return snap
}

// clearSnapshot resets this reactor instance's snapshot holder.
func (r *Reactor) clearSnapshot() {
	r.snapshotHolder.Lock()
	defer r.snapshotHolder.Unlock()
	r.snapshotHolder.snap = nil
}

// ConsumeSnapshot retrieves and consumes the stored RunSnapshot.
// This is the public API for Agent.Resume() to access the snapshot.
func (r *Reactor) ConsumeSnapshot() *RunSnapshot {
	return r.getSnapshot()
}

// PeekSnapshot returns the stored RunSnapshot without consuming it.
func (r *Reactor) PeekSnapshot() *RunSnapshot {
	r.snapshotHolder.RLock()
	defer r.snapshotHolder.RUnlock()
	return r.snapshotHolder.snap
}

// SetAskPermission sets a custom AskPermission instance.
func (r *Reactor) SetAskPermission(p *tools.AskPermission) {
	r.askPermission = p
}

// reactorSetup holds options applied before tool registration.
type reactorSetup struct {
	systemPrompt      string
	skipTools         map[string]bool
	skipAllBundled    bool
	extraTools        []core.FuncTool
	securityPolicy    core.SecurityPolicy
	resultStorage     core.ToolResultStorage
	resultLimits      core.ToolResultLimits
	compactor         core.ContextCompactor
	compactorConfig   core.CompactorConfig
	tokenEstimator    core.TokenEstimator
	eventBus          EventBus
	mcpRegistry       *core.MCPToolRegistry
	skillDirs         []string // External skill directories to load skills from
	skipBundledSkills bool
	messageBus        *core.AgentMessageBus // Shared message bus for team communication
	memory            core.Memory           // Optional: knowledge retrieval for hallucination suppression
	mockLLM           func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error)
	scheduler         *core.CronScheduler // Optional: cron-based scheduled task management

	// === Registry Injection (optional, nil = use default) ===
	intentRegistry IntentRegistry             // Custom intent registry (e.g., LLM-based semantic matching)
	toolRegistry   core.ToolRegistryInterface // Custom tool registry (e.g., MCP integration, dynamic discovery)
	skillRegistry  core.SkillRegistry         // Custom skill registry (e.g., embedding-based semantic matching)
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
func WithSecurityPolicy(policy core.SecurityPolicy) ReactorOption {
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

// WithMCPRegistry sets an MCP tool registry for discovering and calling
// tools from external MCP servers.
func WithMCPRegistry(registry *core.MCPToolRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.mcpRegistry = registry
	}
}

// WithSkillDir specifies external directories to load skills from.
// Each directory should contain subdirectories, each with a SKILL.md file.
// Skills loaded from these directories are registered in addition to bundled skills.
// Multiple directories can be specified by calling WithSkillDir multiple times.
func WithSkillDir(dir string) ReactorOption {
	return func(s *reactorSetup) {
		s.skillDirs = append(s.skillDirs, dir)
	}
}

// WithoutBundledSkills skips registration of all built-in bundled skills.
func WithoutBundledSkills() ReactorOption {
	return func(s *reactorSetup) {
		s.skipBundledSkills = true
	}
}

// WithMessageBus sets an AgentMessageBus for inter-agent team communication.
// SubAgents spawned with a team_name will join teams and can communicate
// via send_message/receive_messages tools. The bus is shared across the
// main reactor and all subagent tasks.
func WithMessageBus(bus *core.AgentMessageBus) ReactorOption {
	return func(s *reactorSetup) {
		s.messageBus = bus
	}
}

// WithMemory sets a Memory implementation for knowledge retrieval.
// Memory is queried during the Think phase to inject relevant knowledge
// into the LLM prompt, suppressing hallucination.
// If not set, the reactor operates without memory augmentation.
func WithMemory(mem core.Memory) ReactorOption {
	return func(s *reactorSetup) {
		s.memory = mem
	}
}

// WithScheduler enables cron-based scheduled task management.
// The scheduler runs a background loop that checks for due tasks every 30 seconds.
// When a task fires, it invokes agent.Run() with the task's prompt.
// The scheduler is started automatically with the reactor's context.
//
// Usage:
//
//	scheduler := core.NewCronScheduler()
//	scheduler.SetCallback(func(ctx context.Context, task core.ScheduledTask) {
//	    result, err := agent.AskWithContext(ctx, task.Prompt)
//	    // handle result...
//	})
//	reactor := reactor.NewReactor(config, reactor.WithScheduler(scheduler))
func WithScheduler(scheduler *core.CronScheduler) ReactorOption {
	return func(s *reactorSetup) {
		s.scheduler = scheduler
	}
}

// MockLLMFunc is the signature for a mock LLM function used in testing.
// When provided via WithMockLLM, the reactor delegates all LLM calls
// to this function instead of the real API client.
type MockLLMFunc func(systemPrompt, userMessage string, history ConversationHistory) (*gochatcore.Response, error)

// WithMockLLM replaces the real LLM client with a deterministic mock function.
// This is intended for end-to-end testing without requiring real API keys or network access.
// The mock function receives the full prompt context (system prompt, user message, history)
// and must return a complete LLM response.
func WithMockLLM(fn MockLLMFunc) ReactorOption {
	return func(s *reactorSetup) {
		s.mockLLM = fn
	}
}

func WithSystemPrompt(prompt string) ReactorOption {
	return func(rs *reactorSetup) {
		rs.systemPrompt = prompt
	}
}

// --- Registry Injection Options ---

// WithIntentRegistry sets a custom IntentRegistry implementation.
// Use this to provide LLM-based intent classification, custom intent types, etc.
// If not set, DefaultIntentRegistry with built-in definitions is used automatically.
//
// Example: embedding-enhanced semantic intent matching:
//
//	type SemanticIntentRegistry struct {
//	    *reactor.DefaultIntentRegistry
//	    embedder *embedding.Client
//	}
//	func (s *SemanticIntentRegistry) FormatPromptSection() string { /* ... */ }
//
//	r := reactor.NewReactor(config, reactor.WithIntentRegistry(&SemanticIntentRegistry{...}))
func WithIntentRegistry(reg IntentRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.intentRegistry = reg
	}
}

// WithToolRegistry sets a custom ToolRegistry implementation.
// Use this to add dynamic tool discovery, MCP integration, semantic filtering, etc.
// If not set, DefaultToolRegistry is used automatically.
//
// Example: MCP-integrated tool registry that merges local + remote tools:
//
//	type MCPToolRegistry struct {
//	    *reactor.DefaultToolRegistry
//	    mcpClient *mcp.Client
//	}
//	func (m *MCPToolRegistry) ToToolInfos() []core.ToolInfo { /* merge local+remote */ }
func WithToolRegistry(reg core.ToolRegistryInterface) ReactorOption {
	return func(s *reactorSetup) {
		s.toolRegistry = reg
	}
}

// WithSkillRegistry sets a custom SkillRegistry implementation.
// Use this to provide embedding-based semantic skill matching, etc.
// If not set, DefaultSkillRegistry is used automatically.
func WithSkillRegistry(reg core.SkillRegistry) ReactorOption {
	return func(s *reactorSetup) {
		s.skillRegistry = reg
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

	// Apply SystemPrompt from options (overrides ReactorConfig if set)
	if setup.systemPrompt != "" {
		config.SystemPrompt = setup.systemPrompt
	}

	r := &Reactor{
		config:          config,
		taskManager:     core.NewInMemoryTaskManager(),
		compactorConfig: core.DefaultCompactorConfig(),
		tokenEstimator:  core.NewDefaultTokenEstimator(3.0),
		memory:          setup.memory,
		mockLLM:         setup.mockLLM,
	}

	// === Registry Initialization (use injected or create defaults) ===
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

	// Apply event bus (create default if not provided via option)
	if setup.eventBus != nil {
		r.eventBus = setup.eventBus
	} else {
		r.eventBus = NewEventBus()
	}

	// Apply message bus for team communication (create default if not provided)
	if setup.messageBus != nil {
		r.messageBus = setup.messageBus
	} else {
		r.messageBus = core.NewAgentMessageBus()
	}

	// Apply scheduler for cron-based scheduled tasks
	if setup.scheduler != nil {
		r.scheduler = setup.scheduler
	}

	// Pre-configure LLM client builder (Issue #13: reuse client across calls)
	r.llmClient = gochat.Client().Config(
		gochat.WithAPIKey(config.APIKey),
		gochat.WithBaseURL(config.BaseURL),
	)

	// Register bundled skills (unless skipped)
	if !setup.skipBundledSkills {
		if err := RegisterBundledSkills(r.skillRegistry); err != nil {
			// Log but don't panic — bundled skills should always be available
			fmt.Printf("[goreact] warning: failed to register bundled skills: %v\n", err)
		}
	}

	// Load skills from external directories
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

	// Register orchestration tools (task, subagent, team, skill)
	r.registerOrchestrationTools()

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
			{"web_search", tools.NewWebSearchTool()},
			{"web_fetch", tools.NewWebFetchToolClaude()},
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
		}
		for _, bt := range bundledTools {
			if !setup.skipTools[bt.name] {
				_ = r.RegisterTool(bt.tool)
			}
		}

		// Inject reactor accessor into cron tool for scheduler access
		if cronTool, ok := r.toolRegistry.Get("cron"); ok {
			if ct, ok := cronTool.(*tools.Cron); ok {
				ct.SetAccessor(r)
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

	// Configure memory tools with the Memory instance
	if r.memory != nil {
		tools.SetMemory(r.memory)
		// Inject memory into registries for reflexive semantic search
		r.toolRegistry.SetMemory(r.memory)
	}

	return r
}

// SkillRegistry returns the reactor's skill registry.
func (r *Reactor) SkillRegistry() core.SkillRegistry {
	return r.skillRegistry
}

// IntentRegistry returns the reactor's intent registry for dynamic intent management.
func (r *Reactor) IntentRegistry() IntentRegistry {
	return r.intentRegistry
}

// ToolRegistry returns the reactor's tool registry for dynamic tool management.
func (r *Reactor) ToolRegistry() core.ToolRegistryInterface {
	return r.toolRegistry
}

// TaskManager returns the reactor's task manager.
func (r *Reactor) TaskManager() core.TaskManager {
	return r.taskManager
}

// Scheduler returns the reactor's cron scheduler, or nil if not configured.
func (r *Reactor) Scheduler() *core.CronScheduler {
	return r.scheduler
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
// If a mockLLM function is configured (for testing), it delegates to the mock instead.
func (r *Reactor) callLLMWithHistory(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) (*gochatcore.Response, error) {
	if r.mockLLM != nil {
		return r.mockLLM(systemPrompt, userMessage, history)
	}
	builder := r.buildLLMBuilder(systemPrompt, userMessage, history, maxHistoryTurns)
	return builder.GetResponseFor(r.config.ClientType)
}

// callLLMStream makes a streaming LLM call, emitting ThinkingDelta events via EventBus
// as content arrives, then returns the complete response content and token usage.
// If mockLLM is configured, it delegates to the mock (non-streaming).
func (r *Reactor) callLLMStream(reactCtx *ReactContext, systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) (string, int, error) {
	// Mock path: delegate directly without streaming
	if r.mockLLM != nil {
		resp, err := r.mockLLM(systemPrompt, userMessage, history)
		if err != nil {
			return "", 0, err
		}
		tokens := 0
		if resp.Usage != nil && resp.Usage.TotalTokens > 0 {
			tokens = resp.Usage.TotalTokens
		}
		return resp.Content, tokens, nil
	}

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
// History messages are truncated using a token-budget-aware strategy: messages are kept from
// newest to oldest (back-to-front) until the history token budget is exhausted.
func (r *Reactor) buildLLMBuilder(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) gochat.ClientBuilder {
	builder := r.llmClient.
		Model(r.config.Model).
		Temperature(r.config.Temperature).
		MaxTokens(r.config.MaxTokens)

	// Layer 1: Agent identity — always use r.config.SystemPrompt as the sole system message.
	// This defines WHO the agent is (its persona, domain expertise, behavioral constraints).
	if r.config.SystemPrompt != "" {
		builder.SystemMessage(r.config.SystemPrompt)
	}

	// Layer 2: Phase instruction (Think/Intent) — prepend to user message.
	// BuildThinkPrompt/BuildIntentPrompt returns structured instructions for the current
	// T-A-O phase. These are NOT identity definitions, so they should NOT be system messages.
	if systemPrompt != "" {
		userMessage = systemPrompt + "\n\n" + userMessage
	}

	// Three-layer token budget allocation
	// Layer 1: System prompts (fixed, not counted against history budget)
	// Layer 2: User message (fixed, not counted against history budget)
	// Layer 3: Conversation history (trimmed to fit remaining budget)
	//
	// We allocate 70% of MaxTokens to history, reserving 30% for system prompts,
	// user message, and the LLM's output.
	maxTokensForHistory := int64(float64(r.config.MaxTokens) * 0.7)

	var chatMessages []gochatcore.Message
	messages := history

	// Apply maxHistoryTurns hard limit first
	if maxHistoryTurns > 0 && len(messages) > maxHistoryTurns {
		messages = messages[len(messages)-maxHistoryTurns:]
	}

	// Token-budget-aware truncation: keep messages from newest to oldest
	estimateFn := r.tokenEstimator.Estimate
	var selectedMessages []core.Message
	var usedTokens int64

	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := int64(estimateFn(messages[i].Content))
		if usedTokens+msgTokens > maxTokensForHistory {
			break // budget exhausted
		}
		selectedMessages = append(selectedMessages, messages[i])
		usedTokens += msgTokens
	}

	// Reverse to restore chronological order
	for i, j := 0, len(selectedMessages)-1; i < j; i, j = i+1, j-1 {
		selectedMessages[i], selectedMessages[j] = selectedMessages[j], selectedMessages[i]
	}

	for _, m := range selectedMessages {
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
// If Memory is configured, relevant records are retrieved and injected into the prompt.
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
	tools := r.toolRegistry.ToToolInfos()

	// Discover applicable skills based on current context/intent
	skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)

	// Retrieve relevant memory records for hallucination suppression
	var memoryRecords []core.MemoryRecord
	if r.memory != nil {
		records, err := r.memory.Retrieve(
			ctx.Ctx(), ctx.Input,
			core.WithMemoryTypes(core.MemoryTypeLongTerm, core.MemoryTypeUser, core.MemoryTypeExperience),
			core.WithMemoryLimit(3),
		)
		if err == nil {
			memoryRecords = records
		}
		// Non-fatal: memory retrieval failure should not break the Think phase
	}

	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, tools, skills, memoryRecords)

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

	// Three-layer loop detection defense
	// Layer 1: Destructive loop — same tool call + same error, immediate termination
	if isDestructiveLoop(ctx.History) {
		return true, "destructive loop detected: same tool call and error repeated"
	}

	// Layer 2: Stuck detection — agent reasoning without tool progress
	if isAgentStuck(ctx.History) {
		return true, "agent stuck: no tool progress in recent iterations"
	}

	// Legacy checks (still useful as additional safeguards)
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
	if r.eventBus != nil {
		reactCtx.emitEvent = r.eventBus.Emit
	}

	// Phase 1: Classify intent
	intent, tokens, err := r.classifyIntent(reactCtx)
	if err != nil {
		reactCtx.EmitEvent(core.Error, fmt.Sprintf("intent classification: %v", err))
		return nil, fmt.Errorf("intent classification: %w", err)
	}
	reactCtx.Intent = intent
	ApplyConfidenceThreshold(intent, 0)

	// Early return for clarification needed from intent
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

	// Phase 2 + 3: T-A-O loop + result building
	return r.runTAOLoop(reactCtx, tokens, time.Now())
}

// RunFromSnapshot resumes a T-A-O execution from a previously saved snapshot.
// It skips intent classification (already done) and continues the T-A-O loop
// from the saved iteration. If newInput is non-empty, it is appended to the
// conversation history before resuming (useful for redirect scenarios).
//
// This is used by Agent.Resume() to continue a paused task.
func (r *Reactor) RunFromSnapshot(ctx context.Context, snapshot *RunSnapshot, newInput string) (*RunResult, error) {
	reactCtx := NewReactContextFromSnapshot(ctx, snapshot)

	// Inject event emission callback
	if r.eventBus != nil {
		reactCtx.emitEvent = r.eventBus.Emit
	}

	// Reset termination state (allow the loop to continue)
	reactCtx.IsTerminated = false
	reactCtx.TerminationReason = ""

	// Inject new user message if provided (redirect scenario)
	if newInput != "" {
		reactCtx.AddMessage("user", newInput)
	}

	// Continue from saved iteration (skip Phase 1: intent already classified)
	return r.runTAOLoop(reactCtx, 0, time.Now())
}

// runTAOLoop executes the T-A-O iteration loop (Phase 2) and builds the final result (Phase 3).
// This is the shared execution path for both Run() and RunFromSnapshot().
func (r *Reactor) runTAOLoop(reactCtx *ReactContext, initialTokens int, runStart time.Time) (*RunResult, error) {
	totalTokens := initialTokens

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
			Iteration: reactCtx.CurrentIteration + 1,
			Duration:  time.Since(cycleStart),
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
		Intent:            reactCtx.Intent,
		Steps:             reactCtx.History,
		TotalIterations:   reactCtx.CurrentIteration,
		TerminationReason: reactCtx.TerminationReason,
		Confidence:        reactCtx.Intent.Confidence,
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

	// If a pause was requested (via Agent.Pause()), save the execution snapshot
	// so the task can be resumed later. This happens before saveExperience because
	// a paused task should not be saved as completed experience.
	r.pauseMu.Lock()
	paused := r.pauseRequested
	r.pauseRequested = false
	r.pauseMu.Unlock()
	if paused {
		snap := reactCtx.ToSnapshot()
		snap.TerminationReason = "paused"
		r.setSnapshot(snap)
	}

	// Save experience to memory if the task completed successfully.
	// This records the problem description + solution so future similar
	// tasks can reuse the analysis instead of spending tokens re-analyzing.
	r.saveExperience(reactCtx, result)

	// Generate a natural-language task summary for non-trivial tasks.
	// Skip for trivially short tasks (single iteration, no tool calls) or failed tasks.
	if result.TotalIterations > 1 && result.Answer != "" &&
		!strings.Contains(result.TerminationReason, "error") {
		r.generateSummary(reactCtx, result, totalDuration)
	}

	return result, nil
}

// --- Termination helper functions (three-layer defense) ---
//
// Layer 1: Destructive loop detection — same tool call + same error repeated.
// Layer 2: Stuck detection — consecutive steps with identical reasoning but no tool progress.
// Layer 3: Hard iteration limit (enforced in CheckTermination).

// maxDestructiveLoopCount is the threshold for detecting a destructive loop.
// If the same tool+params+error appears this many times consecutively, terminate immediately.
const maxDestructiveLoopCount = 3

// maxStuckCount is the threshold for detecting a stuck agent.
// If the agent produces reasoning-only steps (no tool calls) this many times, inject a nudge.
const maxStuckCount = 4

// isToolErrorIrrecoverable checks if a tool error cannot be recovered by retry.
func isToolErrorIrrecoverable(obs *Observation) bool {
	if obs == nil || obs.Error == "" {
		return false
	}
	irrecoverablePatterns := []string{
		"permission denied",
		"unauthorized",
		"invalid api key",
		"authentication",
	}
	retrieablePatterns := []string{
		"not found",
		"team",
	}
	lower := strings.ToLower(obs.Error)
	for _, p := range irrecoverablePatterns {
		if strings.Contains(lower, p) {
			// Check if it's actually a retrieable pattern (e.g., team "not found" is retriable)
			retrieable := false
			for _, rp := range retrieablePatterns {
				if strings.Contains(lower, rp) {
					retrieable = true
					break
				}
			}
			if !retrieable {
				return true
			}
		}
	}
	return false
}

// isDestructiveLoop checks for the most dangerous pattern: the same tool call
// producing the same error repeatedly (e.g., trying to edit a file that doesn't exist).
func isDestructiveLoop(history []Step) bool {
	if len(history) < maxDestructiveLoopCount {
		return false
	}
	// Check the last N steps for identical tool calls with identical errors
	tail := history[len(history)-maxDestructiveLoopCount:]
	var target, params, errMsg string
	for i, step := range tail {
		if step.Action.Type != ActionTypeToolCall {
			return false // not a tool call, break the chain
		}
		if i == 0 {
			target = step.Action.Target
			params = fmt.Sprintf("%v", step.Action.Params)
			errMsg = step.Observation.Error
		} else {
			if step.Action.Target != target ||
				fmt.Sprintf("%v", step.Action.Params) != params ||
				step.Observation.Error != errMsg {
				return false
			}
		}
	}
	// All N steps have identical tool call + error
	return errMsg != "" // only flag if there IS an error (otherwise it's intentional repetition)
}

// isAgentStuck detects when the agent is reasoning without making progress.
// If the agent repeatedly produces Answer/Clarify decisions without tool calls,
// it's likely stuck in a reasoning loop.
func isAgentStuck(history []Step) bool {
	if len(history) < maxStuckCount {
		return false
	}
	// Count consecutive non-tool-call steps from the end
	count := 0
	for i := len(history) - 1; i >= 0 && i >= len(history)-maxStuckCount; i-- {
		if history[i].Action.Type != ActionTypeToolCall {
			count++
		} else {
			break
		}
	}
	return count >= maxStuckCount
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
//
// If Memory implements the ReNewer interface, ReNew is tried first as a semantic
// context rebuild. If ReNew fails or is not available, the traditional compact
// strategies are used as fallback.
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

	// Try ReNew first: semantic context rebuild via Memory (Phase 2)
	if r.tryReNew(ctx) {
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
			Messages:      ctx.ConversationHistory,
			PreserveLastN: r.compactorConfig.PreserveLastN,
			MaxTokens:     int64(r.config.MaxTokens),
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

// tryReNew attempts to use Memory's ReNew capability for semantic context rebuild.
// Returns true if ReNew was successfully applied, false otherwise (fallback to compact).
func (r *Reactor) tryReNew(ctx *ReactContext) bool {
	if r.memory == nil {
		return false
	}
	renewer, ok := r.memory.(core.ReNewer)
	if !ok {
		return false
	}

	// Build a summary of current intent for ReNew
	intentSummary := ""
	if ctx.Intent != nil {
		intentSummary = ctx.Intent.Topic
		if ctx.Intent.Summary != "" {
			if intentSummary != "" {
				intentSummary += " - "
			}
			intentSummary += ctx.Intent.Summary
		}
	}
	if intentSummary == "" {
		intentSummary = ctx.Input
	}

	// Convert ConversationHistory to []core.Message for ReNew
	messages := []core.Message(ctx.ConversationHistory)

	renewed, err := renewer.ReNew(ctx.Ctx(), ctx.SessionID, intentSummary, messages)
	if err != nil || len(renewed) == 0 {
		return false
	}

	// Insert a boundary message to indicate context was rebuilt via Memory
	boundary := core.Message{
		Role:    "system",
		Content: fmt.Sprintf("[Context Rebuilt via Memory] Previous %d messages were semantically rebuilt into %d messages.", len(messages), len(renewed)),
	}
	ctx.ConversationHistory = append([]core.Message{boundary}, renewed...)
	return true
}

// generateSummary produces a natural-language summary of the completed task using the LLM.
// The summary is emitted as a TaskSummary event and appended to the RunResult.
// This runs asynchronously to avoid blocking the Run return.
func (r *Reactor) generateSummary(ctx *ReactContext, result *RunResult, totalDuration time.Duration) {
	toolsUsed := BuildSummaryToolsUsed(ctx.History)
	durationStr := totalDuration.Round(time.Millisecond).String()
	answer := result.Answer
	if len(answer) > 2000 {
		answer = answer[:2000] + "... [truncated]"
	}

	prompt, err := renderSummaryPrompt(summaryPromptData{
		Input:             ctx.Input,
		Answer:            answer,
		Iterations:        result.TotalIterations,
		ToolsUsed:         toolsUsed,
		Duration:          durationStr,
		TerminationReason: result.TerminationReason,
	})
	if err != nil {
		return // non-fatal: summary is a nice-to-have
	}

	go func() {
		resp, err := r.callLLMWithHistory(prompt, "Summarize this task execution.", nil, 0)
		if err != nil || resp == nil || resp.Content == "" {
			return
		}

		summaryText := strings.TrimSpace(resp.Content)
		// Strip markdown code fences if present
		summaryText = stripJSONWrappers(summaryText)
		summaryText = strings.TrimSpace(summaryText)

		ctx.EmitEvent(core.TaskSummary, core.TaskSummaryData{Summary: summaryText})
	}()
}
