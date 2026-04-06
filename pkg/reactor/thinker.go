package reactor

import (
	"context"
	"fmt"

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
	
	// Fallback: Simplified implementation without LLM
	thought := goreactcore.NewThought(
		"Analyzing the current situation...",
		"Need to determine the next action",
		"act",
		0.8,
	)
	
	// Check if we should terminate
	if state.IsComplete() {
		thought.Decision = "answer"
		thought.FinalAnswer = "Task completed based on available information."
		thought.Confidence = 0.7
	}
	
	return thought, nil
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
	prompt := fmt.Sprintf(`
You are an intelligent reasoning engine. Analyze the current state and make a decision.

## Current Task
%s

## Current Step
%d / %d

## History
`, state.Input, state.CurrentStep, state.MaxSteps)
	
	for i, ctx := range context {
		if i >= t.config.MaxHistorySteps {
			break
		}
		prompt += fmt.Sprintf("- %s\n", ctx)
	}
	
	if len(reflections) > 0 {
		prompt += "\n## Relevant Reflections\n"
		for _, r := range reflections {
			prompt += fmt.Sprintf("- %s\n", r)
		}
	}
	
	return prompt
}

// parseResponse parses the LLM response
func (t *BaseThinker) parseResponse(response string) (*goreactcore.Thought, error) {
	// Would parse JSON response from LLM
	// Simplified implementation
	return goreactcore.NewThought(response, "", "act", 0.5), nil
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
	
	// Simplified intent classification
	// Would use LLM for actual classification
	
	intent := goreactcommon.IntentTask
	confidence := 0.7
	
	// Simple heuristic-based classification
	if len(input) < 20 {
		intent = goreactcommon.IntentChat
		confidence = 0.6
	}
	
	return &goreactcore.IntentResult{
		Type:       string(intent),
		Confidence: confidence,
		Reasoning:  "Heuristic-based classification",
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
