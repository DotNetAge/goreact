package reactor

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// IntentRecognizer classifies user intent to determine processing path
type IntentRecognizer struct {
	llmClient       core.Client
	fallbackStrategy *goreactcore.IntentFallbackStrategy
}

// NewIntentRecognizer creates a new IntentRecognizer
func NewIntentRecognizer(llmClient core.Client) *IntentRecognizer {
	return &IntentRecognizer{
		llmClient:        llmClient,
		fallbackStrategy: goreactcore.DefaultIntentFallbackStrategy(),
	}
}

// WithFallbackStrategy sets the fallback strategy
func (r *IntentRecognizer) WithFallbackStrategy(strategy *goreactcore.IntentFallbackStrategy) *IntentRecognizer {
	r.fallbackStrategy = strategy
	return r
}

// Classify classifies the user intent
func (r *IntentRecognizer) Classify(ctx context.Context, input string, state *goreactcore.State) (*goreactcore.IntentResult, error) {
	// Check for pending question first
	if state.PendingQuestion != nil {
		return &goreactcore.IntentResult{
			Type:            string(goreactcommon.IntentClarification),
			Confidence:      0.95,
			Reasoning:       "User is responding to a pending clarification question",
			ExtractedAnswer: input,
		}, nil
	}

	// Build intent classification prompt
	prompt := r.buildIntentPrompt(input, state)

	// Call LLM if available
	if r.llmClient != nil {
		response, err := r.llmClient.Chat(ctx, []core.Message{
			core.NewUserMessage(prompt),
		})
		if err != nil {
			return r.fallback(input, err)
		}
		return r.parseIntentResponse(response.Content)
	}

	// Fallback to heuristic classification
	return r.heuristicClassify(input, state)
}

// buildIntentPrompt builds the intent classification prompt
func (r *IntentRecognizer) buildIntentPrompt(input string, state *goreactcore.State) string {
	return fmt.Sprintf(`
You are an intent classifier. Analyze the user input and return the most matching intent type.

## Input Analysis
- User input: %s
- Context: %v

## Intent Type Definitions
1. **Chat** - Casual conversation, greetings, general dialogue, no specific task
2. **Task** - Need to perform specific operations or solve problems
3. **Clarification** - Answering system's clarification question
4. **FollowUp** - Follow-up question about previous result
5. **Feedback** - Feedback on execution result

## Classification Rules
- If input contains action verbs (help me, please, execute), lean towards Task
- If input answers a question, lean towards Clarification
- If input starts with continue/then, lean towards FollowUp
- If input expresses satisfaction/dissatisfaction, lean towards Feedback
- If input is greeting or no clear purpose, lean towards Chat

## Output Format (strict compliance)
```json
{
  "intent": "<intent_type>",
  "confidence": <0.0-1.0>,
  "reasoning": "<reasoning>"
}
```
`, input, state.Context)
}

// parseIntentResponse parses the LLM response
func (r *IntentRecognizer) parseIntentResponse(response string) (*goreactcore.IntentResult, error) {
	// Simplified parsing - would use JSON parsing in production
	intent := "task"
	confidence := 0.8

	return &goreactcore.IntentResult{
		Type:       intent,
		Confidence: confidence,
		Reasoning:  "LLM-based classification",
	}, nil
}

// heuristicClassify performs heuristic-based classification
func (r *IntentRecognizer) heuristicClassify(input string, state *goreactcore.State) (*goreactcore.IntentResult, error) {
	intent := goreactcommon.IntentTask
	confidence := 0.7
	reasoning := "Heuristic-based classification"

	// Simple heuristics
	inputLen := len(input)
	if inputLen < 20 && !containsActionVerb(input) {
		intent = goreactcommon.IntentChat
		confidence = 0.6
		reasoning = "Short input without action verbs"
	}

	if containsFeedback(input) {
		intent = goreactcommon.IntentFeedback
		confidence = 0.75
		reasoning = "Contains feedback indicators"
	}

	if containsFollowUp(input) {
		intent = goreactcommon.IntentFollowUp
		confidence = 0.75
		reasoning = "Contains follow-up indicators"
	}

	return &goreactcore.IntentResult{
		Type:       string(intent),
		Confidence: confidence,
		Reasoning:  reasoning,
	}, nil
}

// fallback handles classification failures
func (r *IntentRecognizer) fallback(input string, err error) (*goreactcore.IntentResult, error) {
	// Use default intent when LLM fails
	return &goreactcore.IntentResult{
		Type:       r.fallbackStrategy.DefaultIntent,
		Confidence: r.fallbackStrategy.MinConfidence,
		Reasoning:  fmt.Sprintf("Fallback due to error: %v", err),
	}, nil
}

// Helper functions
func containsActionVerb(input string) bool {
	verbs := []string{"帮我", "请", "执行", "创建", "删除", "修改", "help", "please", "execute", "create", "delete", "modify"}
	for _, v := range verbs {
		if contains(input, v) {
			return true
		}
	}
	return false
}

func containsFeedback(input string) bool {
	indicators := []string{"很好", "不错", "不对", "错误", "满意", "good", "great", "wrong", "error"}
	for _, i := range indicators {
		if contains(input, i) {
			return true
		}
	}
	return false
}

func containsFollowUp(input string) bool {
	indicators := []string{"继续", "然后", "接下来", "还有", "continue", "then", "next", "more"}
	for _, i := range indicators {
		if contains(input, i) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
