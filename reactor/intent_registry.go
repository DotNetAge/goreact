package reactor

import (
	"fmt"
	"strings"
	"sync"
)

// DefaultIntentRegistry manages dynamically registered intent types.
// It is safe for concurrent use.
type DefaultIntentRegistry struct {
	mu      sync.RWMutex
	defs    []IntentDefinition
	builtIn bool
}

// IntentRegistry is an alias for DefaultIntentRegistry for backward compatibility.
// type IntentRegistry = DefaultIntentRegistry

// NewDefaultIntentRegistry creates a registry pre-loaded with the 5 built-in intent definitions.
func NewDefaultIntentRegistry() *DefaultIntentRegistry {
	r := &DefaultIntentRegistry{builtIn: true}
	r.defs = DefaultIntentDefinitions()
	return r
}

// Deprecated: Use NewDefaultIntentRegistry instead.
func NewIntentRegistry() *DefaultIntentRegistry {
	return NewDefaultIntentRegistry()
}

// Compile-time interface check
var _ IntentRegistry = (*DefaultIntentRegistry)(nil)

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

// Register adds a new intent definition to the registry.
// It returns an error if an intent with the same type already exists.
func (r *DefaultIntentRegistry) Register(def IntentDefinition) error {
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
func (r *DefaultIntentRegistry) Unregister(typ string) {
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
func (r *DefaultIntentRegistry) All() []IntentDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]IntentDefinition, len(r.defs))
	copy(out, r.defs)
	return out
}

// FormatPromptSection renders all registered intents into the prompt XML block.
func (r *DefaultIntentRegistry) FormatPromptSection() string {
	defs := r.All()
	var sb strings.Builder
	for i, d := range defs {
		fmt.Fprintf(&sb, "%d. **%s** - %s\n", i+1, d.Type, d.Description)
	}
	return sb.String()
}
