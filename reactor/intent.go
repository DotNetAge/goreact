package reactor

import (
	"fmt"
	"strings"
	"sync"

	"github.com/DotNetAge/goreact/core"
)

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

// IntentRegistry manages dynamically registered intent types.
// It is safe for concurrent use.
type IntentRegistry struct {
	mu      sync.RWMutex
	defs    []IntentDefinition
	builtIn bool
}

// IntentDefinition describes a single intent type that can be used for classification.
type IntentDefinition struct {
	Type          string `json:"type" yaml:"type"`                     // Intent identifier (e.g. "task", "chat")
	Description   string `json:"description" yaml:"description"`       // English description
	DescriptionCN string `json:"description_cn" yaml:"description_cn"` // Chinese description
}

// DefaultIntentDefinitions returns the 5 built-in intent types with bilingual descriptions.
func DefaultIntentDefinitions() []IntentDefinition {
	return []IntentDefinition{
		{
			Type:          "chat",
			Description:   "Casual conversation, greetings, emotional expression, general knowledge questions with no actionable goal",
			DescriptionCN: "日常闲聊、问候寒暄、情感表达、无具体可执行目标的通用知识提问",
		},
		{
			Type:          "task",
			Description:   "The user wants the system to perform a concrete action: query data, execute an operation, create/modify/delete something, compute something",
			DescriptionCN: "用户希望系统执行具体操作：查询数据、执行操作、创建/修改/删除内容、进行计算",
		},
		{
			Type:          "clarification",
			Description:   "The user is directly answering a previous question asked by the system. This is NOT general dialogue - it's a specific response to a system prompt.",
			DescriptionCN: "用户直接回答了系统之前提出的问题。这不是普通对话——而是对系统提问的明确回应。",
		},
		{
			Type:          "follow_up",
			Description:   "The user wants to drill deeper into, refine, or get more details about a previous result or topic. They reference something discussed earlier.",
			DescriptionCN: "用户希望对之前的结果或话题进行深入探讨、细化或获取更多细节，引用了之前讨论过的内容。",
		},
		{
			Type:          "feedback",
			Description:   "The user is evaluating a previous result: expressing satisfaction, dissatisfaction, or requesting corrections.",
			DescriptionCN: "用户对之前的结果进行评价：表达满意、不满意或请求修正。",
		},
	}
}

// NewIntentRegistry creates a registry pre-loaded with the 5 built-in intent definitions.
func NewIntentRegistry() *IntentRegistry {
	r := &IntentRegistry{builtIn: true}
	r.defs = DefaultIntentDefinitions()
	return r
}

// Register adds a new intent definition to the registry.
// It returns an error if an intent with the same type already exists.
func (r *IntentRegistry) Register(def IntentDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.defs {
		if d.Type == def.Type {
			return fmt.Errorf("intent type %q already registered", def.Type)
		}
	}
	r.defs = append(r.defs, def)
	return nil
}

// Unregister removes an intent definition by type.
// Built-in intents can be unregistered to allow full customization.
func (r *IntentRegistry) Unregister(typ string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, d := range r.defs {
		if d.Type == typ {
			r.defs = append(r.defs[:i], r.defs[i+1:]...)
			return
		}
	}
}

// All returns a copy of all registered intent definitions.
func (r *IntentRegistry) All() []IntentDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]IntentDefinition, len(r.defs))
	copy(out, r.defs)
	return out
}

// FormatPromptSection renders all registered intents into the prompt XML block.
func (r *IntentRegistry) FormatPromptSection() string {
	defs := r.All()
	var sb strings.Builder
	for i, d := range defs {
		fmt.Fprintf(&sb, "%d. **%s** - %s\n", i+1, d.Type, d.Description)
	}
	return sb.String()
}

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

// BuildIntentPrompt builds the intent classification prompt using Go template.
// registry controls which intent types appear in the prompt; if nil, defaults are used.
func BuildIntentPrompt(input string, context string, registry *IntentRegistry) string {
	if registry == nil {
		registry = NewIntentRegistry()
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
