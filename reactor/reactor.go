package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gochat "github.com/DotNetAge/gochat"
	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
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

// gochatConfig converts ReactorConfig to gochat's core.Config.
func (c ReactorConfig) gochatConfig() gochatcore.Config {
	return gochatcore.Config{
		APIKey:    c.APIKey,
		BaseURL:   c.BaseURL,
		Model:     c.Model,
		MaxTokens: c.MaxTokens,
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
}

// ReActor is the core interface for the T-A-O reactor.
type ReActor interface {
	Think(ctx *ReactContext) (int, error)
	Act(ctx *ReactContext) error
	Observe(ctx *ReactContext) error
	CheckTermination(ctx *ReactContext) (bool, string)
}

// defaultReactor is the standard T-A-O reactor implementation.
type defaultReactor struct {
	config         ReactorConfig
	intentRegistry *IntentRegistry
	toolRegistry   *ToolRegistry
}

// NewReactor creates a new Reactor with the given configuration.
func NewReactor(config ReactorConfig) *defaultReactor {
	if config.MaxIterations <= 0 {
		config.MaxIterations = core.DefaultMaxSteps
	}
	if config.Temperature <= 0 {
		config.Temperature = core.DefaultTemperature
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = core.DefaultMaxTokens
	}
	return &defaultReactor{
		config:         config,
		intentRegistry: NewIntentRegistry(),
		toolRegistry:   NewToolRegistry(),
	}
}

// IntentRegistry returns the reactor's intent registry for dynamic intent management.
func (r *defaultReactor) IntentRegistry() *IntentRegistry {
	return r.intentRegistry
}

// ToolRegistry returns the reactor's tool registry for dynamic tool management.
func (r *defaultReactor) ToolRegistry() *ToolRegistry {
	return r.toolRegistry
}

// RegisterTool is a convenience method to register a core.FuncTool.
func (r *defaultReactor) RegisterTool(tool core.FuncTool) error {
	return r.toolRegistry.Register(tool)
}

// RegisterIntent is a convenience method to register an intent type.
func (r *defaultReactor) RegisterIntent(def IntentDefinition) error {
	return r.intentRegistry.Register(def)
}

// callLLMWithHistory makes an LLM call using the reactor's configuration and conversation history.
func (r *defaultReactor) callLLMWithHistory(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) (*gochatcore.Response, error) {
	builder := gochat.Client().
		Config(
			gochat.WithAPIKey(r.config.APIKey),
			gochat.WithBaseURL(r.config.BaseURL),
		).
		Model(r.config.Model).
		Temperature(r.config.Temperature).
		MaxTokens(r.config.MaxTokens)

	if r.config.SystemPrompt != "" {
		builder.SystemMessage(r.config.SystemPrompt)
	}
	if systemPrompt != "" {
		builder.SystemMessage(systemPrompt)
	}

	// Inject history messages
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

	return builder.GetResponseFor(r.config.ClientType)
}

// classifyIntent runs intent classification on the user's input.
func (r *defaultReactor) classifyIntent(ctx *ReactContext) (*Intent, int, error) {
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
func (r *defaultReactor) Think(ctx *ReactContext) (int, error) {
	tools := r.toolRegistry.ToToolInfos()
	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, tools)

	resp, err := r.callLLMWithHistory(instructions, ctx.Input, ctx.ConversationHistory, MaxHistoryTurns)
	if err != nil {
		return 0, fmt.Errorf("think LLM call failed: %w", err)
	}

	tokens := 0
	if resp.Usage != nil && resp.Usage.TotalTokens > 0 {
		tokens = resp.Usage.TotalTokens
	}

	thought, err := ParseThinkResponse(resp.Content)
	if err != nil {
		return tokens, fmt.Errorf("think parse failed: %w", err)
	}

	ctx.LastThought = thought
	return tokens, nil
}

// Act executes the decision from the Think phase.
func (r *defaultReactor) Act(ctx *ReactContext) error {
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
func (r *defaultReactor) Observe(ctx *ReactContext) error {
	action := ctx.LastAction
	if action == nil {
		return fmt.Errorf("observe called without an action")
	}

	var obs *Observation

	switch action.Type {
	case ActionTypeToolCall:
		if action.Error != nil {
			obs = NewErrorObservation(action.ErrorMsg, false)
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
func (r *defaultReactor) CheckTermination(ctx *ReactContext) (bool, string) {
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
		return true, "clarification needed"
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
func (r *defaultReactor) Run(ctx context.Context, input string, history ConversationHistory) (*RunResult, error) {
	reactCtx := NewReactContext(ctx, input, history, r.config.MaxIterations)
	totalTokens := 0

	// Phase 1: Classify intent
	intent, tokens, err := r.classifyIntent(reactCtx)
	if err != nil {
		return nil, fmt.Errorf("intent classification: %w", err)
	}
	totalTokens += tokens
	reactCtx.Intent = intent

	// Apply confidence threshold
	ApplyConfidenceThreshold(intent, 0)

	// Early return for clarification needed from intent
	if intent.RequiresClarification {
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

		// Think
		tokens, err := r.Think(reactCtx)
		totalTokens += tokens
		if err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("think error: %v", err)
			break
		}

		// Act
		if err := r.Act(reactCtx); err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("act error: %v", err)
			break
		}

		// Observe
		if err := r.Observe(reactCtx); err != nil {
			reactCtx.TerminationReason = fmt.Sprintf("observe error: %v", err)
			break
		}

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
