package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// BaseThinker provides base thinker functionality
type BaseThinker struct {
	llmClient     core.Client
	promptBuilder PromptBuilder
	memory        MemoryAccessor
	config        *goreactcommon.ThinkerConfig
}

// PromptBuilder interface for building prompts
type PromptBuilder interface {
	BuildThinkPrompt(state *goreactcore.State) string
	BuildIntentPrompt(input string) string
}

// MemoryAccessor interface for memory access
type MemoryAccessor interface {
	GetRecentHistory(sessionName string, limit int) []string
	GetRelevantReflections(taskType string) []string
}

// NewBaseThinker creates a new BaseThinker
func NewBaseThinker(llmClient core.Client, config *goreactcommon.ThinkerConfig) *BaseThinker {
	if config == nil {
		config = &goreactcommon.ThinkerConfig{
			MaxTokens:                  goreactcommon.DefaultMaxTokens,
			Temperature:                goreactcommon.DefaultTemperature,
			EnableReflectionInjection:  true,
			EnablePlanContext:          true,
			MaxHistorySteps:            goreactcommon.DefaultMaxHistorySteps,
			ConfidenceThreshold:        goreactcommon.DefaultConfidenceThreshold,
		}
	}
	return &BaseThinker{llmClient: llmClient, config: config}
}

// WithPromptBuilder sets the prompt builder
func (t *BaseThinker) WithPromptBuilder(pb PromptBuilder) *BaseThinker {
	t.promptBuilder = pb
	return t
}

// WithMemory sets the memory accessor
func (t *BaseThinker) WithMemory(m MemoryAccessor) *BaseThinker {
	t.memory = m
	return t
}

// Think performs thinking
func (t *BaseThinker) Think(ctx context.Context, state *goreactcore.State) (*goreactcore.Thought, error) {
	// 1. Retrieve context from memory
	var contextHistory []string
	if t.memory != nil {
		contextHistory = t.memory.GetRecentHistory(state.SessionName, t.config.MaxHistorySteps)
	}
	
	// 2. Retrieve reflections
	var reflections []string
	if t.memory != nil {
		reflections = t.memory.GetRelevantReflections("task")
	}
	
	// 3. Build prompt
	prompt := t.buildPrompt(state, contextHistory, reflections)
	
	// 4. Call LLM if available
	if t.llmClient != nil {
		response, err := t.callLLM(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}
		// 5. Parse response
		return t.parseResponse(response)
	}
	
	// Fallback: Analyze state without LLM
	return t.analyzeStateWithoutLLM(state)
}

// callLLM calls the LLM client
func (t *BaseThinker) callLLM(ctx context.Context, prompt string) (string, error) {
	resp, err := t.llmClient.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// buildPrompt builds the think prompt
func (t *BaseThinker) buildPrompt(state *goreactcore.State, context []string, reflections []string) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf(`You are an intelligent reasoning engine. Analyze the current state and make a decision.

## Current Task
%s

## Current Step
%d / %d

## Current Plan
`, state.Input, state.CurrentStep, state.MaxSteps))
	
	if state.Plan != nil {
		sb.WriteString(fmt.Sprintf("Goal: %s\n", state.Plan.Goal))
		sb.WriteString("Steps:\n")
		for i, step := range state.Plan.Steps {
			status := "pending"
			if i < state.Plan.CurrentStepIndex {
				status = "completed"
			} else if i == state.Plan.CurrentStepIndex {
				status = "current"
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, status, step.Description))
		}
	}
	
	sb.WriteString("\n## History\n")
	for i, ctx := range context {
		if i >= t.config.MaxHistorySteps {
			break
		}
		sb.WriteString(fmt.Sprintf("- %s\n", ctx))
	}
	
	if len(reflections) > 0 {
		sb.WriteString("\n## Relevant Reflections\n")
		for _, r := range reflections {
			sb.WriteString(fmt.Sprintf("- %s\n", r))
		}
	}
	
	sb.WriteString(`
## Decision Required
Decide what to do next:
1. "act" - Execute an action (tool call, skill, or sub-agent delegation)
2. "answer" - Provide the final answer

## Output Format (JSON)
{
  "thought": "Your reasoning process",
  "decision": "act|answer",
  "action_type": "tool_call|skill_invoke|sub_agent_delegate|none",
  "action_target": "tool_or_skill_name",
  "action_params": {"key": "value"},
  "final_answer": "answer if decision is answer",
  "confidence": 0.0-1.0
}`)
	
	return sb.String()
}

// parseResponse parses the LLM response
func (t *BaseThinker) parseResponse(response string) (*goreactcore.Thought, error) {
	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		// Return a thought with the raw response
		return goreactcore.NewThought(response, "", "act", 0.5), nil
	}
	
	jsonStr := response[jsonStart : jsonEnd+1]
	
	var parsed struct {
		Thought      string         `json:"thought"`
		Decision     string         `json:"decision"`
		ActionType   string         `json:"action_type"`
		ActionTarget string         `json:"action_target"`
		ActionParams map[string]any `json:"action_params"`
		FinalAnswer  string         `json:"final_answer"`
		Confidence   float64        `json:"confidence"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	thought := goreactcore.NewThought(parsed.Thought, parsed.FinalAnswer, parsed.Decision, parsed.Confidence)
	
	// Set action intent if decision is act
	if parsed.Decision == "act" && parsed.ActionTarget != "" {
		thought.Action = &goreactcore.ActionIntent{
			Type:   parsed.ActionType,
			Target: parsed.ActionTarget,
			Params: parsed.ActionParams,
		}
	}
	
	return thought, nil
}

// analyzeStateWithoutLLM analyzes state without LLM
func (t *BaseThinker) analyzeStateWithoutLLM(state *goreactcore.State) (*goreactcore.Thought, error) {
	thought := goreactcore.NewThought(
		"Analyzing the current situation...",
		"",
		"act",
		0.8,
	)
	
	// Check if we should terminate
	if state.IsComplete() {
		thought.Decision = "answer"
		thought.FinalAnswer = "Task completed based on available information."
		thought.Confidence = 0.7
		return thought, nil
	}
	
	// Check if we have a current plan step
	if state.Plan != nil && state.Plan.GetCurrentStep() != nil {
		currentStep := state.Plan.GetCurrentStep()
		thought.Content = fmt.Sprintf("Following plan: %s", currentStep.Description)
		
		// Determine action based on step description
		thought.Action = t.inferActionFromStep(currentStep.Description)
	}
	
	return thought, nil
}

// inferActionFromStep infers an action from a plan step description
func (t *BaseThinker) inferActionFromStep(description string) *goreactcore.ActionIntent {
	desc := strings.ToLower(description)
	
	// Check for common patterns
	if strings.Contains(desc, "analyze") || strings.Contains(desc, "分析") {
		return &goreactcore.ActionIntent{
			Type:   string(goreactcommon.ActionTypeToolCall),
			Target: "analyze",
			Params: map[string]any{},
		}
	}
	
	if strings.Contains(desc, "execute") || strings.Contains(desc, "执行") {
		return &goreactcore.ActionIntent{
			Type:   string(goreactcommon.ActionTypeSkillInvoke),
			Target: "execute",
			Params: map[string]any{},
		}
	}
	
	if strings.Contains(desc, "delegate") || strings.Contains(desc, "委托") {
		return &goreactcore.ActionIntent{
			Type:   string(goreactcommon.ActionTypeSubAgentDelegate),
			Target: "sub_agent",
			Params: map[string]any{},
		}
	}
	
	// Default to tool call
	return &goreactcore.ActionIntent{
		Type:   string(goreactcommon.ActionTypeToolCall),
		Target: "default",
		Params: map[string]any{},
	}
}

// ClassifyIntent classifies the user intent
func (t *BaseThinker) ClassifyIntent(ctx context.Context, input string, state *goreactcore.State) (*goreactcore.IntentResult, error) {
	// Check for pending question
	if state.PendingQuestion != nil {
		return &goreactcore.IntentResult{
			Type:            string(goreactcommon.IntentClarification),
			Confidence:      0.9,
			ExtractedAnswer: input,
		}, nil
	}
	
	// Use LLM for intent classification if available
	if t.llmClient != nil {
		return t.classifyIntentWithLLM(ctx, input, state)
	}
	
	// Fallback: Heuristic-based classification
	return t.classifyIntentWithHeuristics(input)
}

// classifyIntentWithLLM classifies intent using LLM
func (t *BaseThinker) classifyIntentWithLLM(ctx context.Context, input string, state *goreactcore.State) (*goreactcore.IntentResult, error) {
	prompt := fmt.Sprintf(`Classify the user intent.

Input: %s

Intent Types:
1. chat - Casual conversation, greetings, general dialogue
2. task - Need to perform specific operations or solve problems
3. clarification - Answering a question
4. follow_up - Follow-up question about previous result
5. feedback - Feedback on execution result

Output JSON format:
{
  "intent": "intent_type",
  "confidence": 0.0-1.0,
  "reasoning": "explanation"
}`, input)
	
	resp, err := t.llmClient.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return t.classifyIntentWithHeuristics(input)
	}
	
	// Parse response
	jsonStart := strings.Index(resp.Content, "{")
	jsonEnd := strings.LastIndex(resp.Content, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return t.classifyIntentWithHeuristics(input)
	}
	
	var parsed struct {
		Intent     string  `json:"intent"`
		Confidence float64 `json:"confidence"`
		Reasoning  string  `json:"reasoning"`
	}
	
	if err := json.Unmarshal([]byte(resp.Content[jsonStart:jsonEnd+1]), &parsed); err != nil {
		return t.classifyIntentWithHeuristics(input)
	}
	
	return &goreactcore.IntentResult{
		Type:       parsed.Intent,
		Confidence: parsed.Confidence,
		Reasoning:  parsed.Reasoning,
	}, nil
}

// classifyIntentWithHeuristics classifies intent using heuristics
func (t *BaseThinker) classifyIntentWithHeuristics(input string) (*goreactcore.IntentResult, error) {
	intent := goreactcommon.IntentTask
	confidence := 0.7
	reasoning := "Heuristic-based classification"
	
	inputLower := strings.ToLower(input)
	
	// Check for chat patterns
	chatPatterns := []string{"hello", "hi", "hey", "你好", "您好", "嗨"}
	for _, pattern := range chatPatterns {
		if strings.Contains(inputLower, pattern) {
			intent = goreactcommon.IntentChat
			confidence = 0.8
			reasoning = "Detected greeting pattern"
			break
		}
	}
	
	// Check for task patterns
	taskPatterns := []string{"help me", "please", "execute", "帮我", "请", "执行"}
	for _, pattern := range taskPatterns {
		if strings.Contains(inputLower, pattern) {
			intent = goreactcommon.IntentTask
			confidence = 0.9
			reasoning = "Detected task request pattern"
			break
		}
	}
	
	// Check for short input (likely chat)
	if len(input) < 20 && intent == goreactcommon.IntentTask {
		intent = goreactcommon.IntentChat
		confidence = 0.6
		reasoning = "Short input, likely chat"
	}
	
	return &goreactcore.IntentResult{
		Type:       string(intent),
		Confidence: confidence,
		Reasoning:  reasoning,
	}, nil
}

// BuildPrompt builds the think prompt (public method)
func (t *BaseThinker) BuildPrompt(state *goreactcore.State) string {
	return t.buildPrompt(state, nil, nil)
}

// ParseResponse parses the LLM response (public method)
func (t *BaseThinker) ParseResponse(response string) (*goreactcore.Thought, error) {
	return t.parseResponse(response)
}
