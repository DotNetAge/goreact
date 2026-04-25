package reactor

import (
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// Intent represents the result of intent classification
type Intent struct {
	ID string `json:"id" yaml:"id"`
	// 核心分类
	Type       string  `json:"type" yaml:"type"`
	Confidence float64 `json:"confidence" yaml:"confidence"`

	// 实体与槽位提取
	Entities map[string]any `json:"entities" yaml:"entities"`

	// 语义理解
	Summary string `json:"summary" yaml:"summary"`
	Topic   string `json:"topic" yaml:"topic"`

	// 澄清机制
	RequiresClarification bool     `json:"requires_clarification" yaml:"requires_clarification"`
	MissingSlots          []string `json:"missing_slots" yaml:"missing_slots"`
	ClarificationQuestion string   `json:"clarification_question" yaml:"clarification_question"`

	// 上下文关联
	ReferenceID   string `json:"reference_id" yaml:"reference_id"`
	ReferenceType string `json:"reference_type" yaml:"reference_type"`
}

// ApplyConfidenceThreshold post-processes an Intent result.
// When confidence is below the given threshold and the intent is not already
// requesting clarification, it automatically sets RequiresClarification to true
// and generates a confirmation question.
// Pass threshold <= 0 to use core.IntentClarifyThreshold.
func ApplyConfidenceThreshold(intent *Intent, threshold float64) {
	if threshold <= 0 {
		threshold = core.IntentClarifyThreshold
	}
	if intent.Confidence < threshold && !intent.RequiresClarification {
		intent.RequiresClarification = true
		intent.MissingSlots = append(intent.MissingSlots, "intent_confirmation")
		intent.ClarificationQuestion = fmt.Sprintf(
			"Did you mean: %s? Please confirm so I can assist you more accurately.",
			intent.Summary,
		)
	}
}

// BuildIntentPrompt builds the intent classification prompt using Go template.
// registry controls which intent types appear in the prompt; if nil, defaults are used.
func BuildIntentPrompt(input string, context string, registry IntentRegistry) string {
	if registry == nil {
		registry = NewDefaultIntentRegistry()
	}

	intentTypesSection := registry.FormatPromptSection()

	result, err := renderIntentPrompt(intentPromptData{
		IntentTypes: intentTypesSection,
		Input:       input,
		Context:     context,
	})
	if err != nil {
		// Fallback: should never happen since template is parsed at init
		return fmt.Sprintf("intent prompt render error: %v", err)
	}
	return result
}
