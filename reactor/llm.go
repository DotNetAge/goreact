package reactor

import (
	"encoding/json"
	"fmt"
	"strings"

	gochat "github.com/DotNetAge/gochat"
	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// callLLMWithHistory makes an LLM call using the reactor's cached client and conversation history.
// If a mockLLM function is configured (for testing), it delegates to the mock instead.
func (r *Reactor) callLLMWithHistory(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int) (*gochatcore.Response, error) {
	if r.mockLLM != nil {
		return r.mockLLM(systemPrompt, userMessage, history)
	}
	builder := r.buildLLMBuilder(systemPrompt, userMessage, history, maxHistoryTurns, nil, "")
	return builder.GetResponseFor(r.config.ClientType)
}

// callLLMStream makes a streaming LLM call, emitting ThinkingDelta events via EventBus
// as content arrives, then returns the complete response content and token usage.
// If mockLLM is configured, it delegates to the mock (non-streaming).
//
// llmTools are passed to the LLM for native function calling.
// skillsSection is injected as an additional SystemMessage (capabilities layer).
func (r *Reactor) callLLMStream(reactCtx *ReactContext, systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int, llmTools []gochatcore.Tool, skillsSection string) (string, int, error) {
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

	builder := r.buildLLMBuilder(systemPrompt, userMessage, history, maxHistoryTurns, llmTools, skillsSection)

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
			reactCtx.EmitEvent(core.ThinkingDelta, event.Content)

		case gochatcore.EventError:
			return contentBuf.String(), 0, event.Err

		case gochatcore.EventDone:
		}
	}

	tokens := 0
	if usage := stream.Usage(); usage != nil && usage.TotalTokens > 0 {
		tokens = usage.TotalTokens
	}

	return contentBuf.String(), tokens, nil
}

// buildLLMBuilder creates a pre-configured ClientBuilder with system prompt, history, user message,
// native tools, and skills section.
//
// Layer 1 (System Messages): Base identity + optional skills/capabilities
// Layer 2 (User Message): Phase instruction (Think/Intent) + user input
// Layer 3 (History): Conversation history trimmed by token budget
// Layer 4 (Tools): Native function calling definitions
func (r *Reactor) buildLLMBuilder(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int, llmTools []gochatcore.Tool, skillsSection string) gochat.ClientBuilder {
	builder := r.llmClient.
		Model(r.config.Model).
		Temperature(r.config.Temperature).
		MaxTokens(r.config.MaxTokens)

	if r.config.TopP > 0 {
		builder.TopP(r.config.TopP)
	}
	if r.config.TopK > 0 {
		builder.TopK(r.config.TopK)
	}
	if r.config.PresencePenalty != 0 {
		builder.PresencePenalty(r.config.PresencePenalty)
	}
	if r.config.FrequencyPenalty != 0 {
		builder.FrequencyPenalty(r.config.FrequencyPenalty)
	}

	if r.config.SystemPrompt != "" {
		builder.SystemMessage(r.config.SystemPrompt)
	}

	if skillsSection != "" {
		builder.SystemMessage(skillsSection)
	}

	if systemPrompt != "" {
		builder.SystemMessage(systemPrompt)
	}

	maxTokensForHistory := int64(float64(r.config.MaxTokens) * historyTokenBudgetRatio)

	var chatMessages []gochatcore.Message
	messages := history

	if maxHistoryTurns > 0 && len(messages) > maxHistoryTurns {
		messages = messages[len(messages)-maxHistoryTurns:]
	}

	estimateFn := r.tokenEstimator.Estimate
	var selectedMessages []core.Message
	var usedTokens int64

	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := int64(estimateFn(messages[i].Content))
		if usedTokens+msgTokens > maxTokensForHistory {
			break
		}
		selectedMessages = append(selectedMessages, messages[i])
		usedTokens += msgTokens
	}

	for i, j := 0, len(selectedMessages)-1; i < j; i, j = i+1, j-1 {
		selectedMessages[i], selectedMessages[j] = selectedMessages[j], selectedMessages[i]
	}

	for _, m := range selectedMessages {
		chatMessages = append(chatMessages, gochatcore.NewTextMessage(m.Role, m.Content))
	}
	builder.Messages(chatMessages...)
	builder.UserMessage(userMessage)

	if len(llmTools) > 0 {
		builder.Tools(llmTools...)
	}

	return builder
}

// estimateInputTokens calculates the estimated total input tokens for an LLM call.
// Covers all layers sent via buildLLMBuilder:
//   - Layer 1: Base SystemPrompt + Skills section + Phase-specific prompt/instructions
//   - Layer 2: User message
//   - Layer 3: Conversation history (trimmed by maxHistoryTurns then by token budget)
//   - Layer 4: Native function calling tool definitions
//
// This ensures TokenUsed in RunResult reflects actual API consumption,
// and ContextWindow sliding decisions are based on accurate data.
func (r *Reactor) estimateInputTokens(systemPrompt, userMessage string, history ConversationHistory, maxHistoryTurns int, llmTools []gochatcore.Tool, skillsSection string) int {
	estimateFn := r.tokenEstimator.Estimate
	total := 0

	// Layer 1a: Base identity SystemPrompt (always sent first in buildLLMBuilder)
	if r.config.SystemPrompt != "" {
		total += estimateFn(r.config.SystemPrompt)
	}

	// Layer 1b: Skills/capabilities section
	if skillsSection != "" {
		total += estimateFn(skillsSection)
	}

	// Layer 1c: Phase-specific prompt or instructions
	if systemPrompt != "" {
		total += estimateFn(systemPrompt)
	}

	// Layer 2: User message
	if userMessage != "" {
		total += estimateFn(userMessage)
	}

	// Layer 3: Conversation history — replicate exact trim logic from buildLLMBuilder
	messages := history
	if maxHistoryTurns > 0 && len(messages) > maxHistoryTurns {
		messages = messages[len(messages)-maxHistoryTurns:]
	}
	maxTokensForHistory := int64(float64(r.config.MaxTokens) * historyTokenBudgetRatio)
	var usedTokens int64
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := int64(estimateFn(messages[i].Content))
		if usedTokens+msgTokens > maxTokensForHistory {
			break
		}
		total += int(msgTokens)
		usedTokens += msgTokens
	}

	// Layer 4: Native function calling tool definitions
	if len(llmTools) > 0 {
		total += int(EstimateTokensForTools(llmTools, estimateFn))
	}

	return total
}

// formatConversationContext extracts a compact context summary from conversation history
// for injection into prompts that benefit from conversational awareness (e.g., intent classification).
// maxTurns controls how many recent messages to include; 0 means all.
func formatConversationContext(history ConversationHistory, maxTurns int) string {
	if len(history) == 0 {
		return ""
	}
	messages := history
	if maxTurns > 0 && len(messages) > maxTurns {
		messages = messages[len(messages)-maxTurns:]
	}
	var sb strings.Builder
	for _, msg := range messages {
		fmt.Fprintf(&sb, "[%s] %s\n", msg.Role, msg.Content)
	}
	return sb.String()
}

// classifyIntent runs intent classification on the user's input.
// Returns the classified Intent, total token count (input + output), and error.
func (r *Reactor) classifyIntent(ctx *ReactContext) (*Intent, int, error) {
	// FIX(P0-#1): Format conversation history into context string so the intent classifier
	// has awareness of prior dialogue, instead of always passing empty context.
	contextStr := formatConversationContext(ctx.ConversationHistory, 3)

	instructions := BuildIntentPrompt(ctx.Input, contextStr, r.intentRegistry)

	resp, err := r.callLLMWithHistory(instructions, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns())
	if err != nil {
		inputTokens := r.estimateInputTokens(instructions, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), nil, "")
		return nil, inputTokens, fmt.Errorf("intent classification LLM call failed: %w", err)
	}

	// FIX(P0-#2): Calculate input tokens explicitly so totalTokens = input + output,
	// not just the LLM-reported TotalTokens (which may be output-only for some providers).
	inputTokens := r.estimateInputTokens(instructions, ctx.Input, ctx.ConversationHistory, r.maxHistoryTurns(), nil, "")

	outputTokens := 0
	if resp.Usage != nil && resp.Usage.TotalTokens > 0 {
		outputTokens = resp.Usage.TotalTokens
	}

	intent, err := parseIntentResponse(resp.Content)
	if err != nil {
		return intent, inputTokens + outputTokens, fmt.Errorf("intent classification parse failed: %w", err)
	}

	return intent, inputTokens + outputTokens, nil
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
