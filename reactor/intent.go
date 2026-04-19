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
	Entities map[string]string `json:"entities" yaml:"entities"`

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

// BuildIntentPrompt builds the intent classification prompt.
// registry controls which intent types appear in the prompt; if nil, defaults are used.
func BuildIntentPrompt(input string, context string, registry *IntentRegistry) string {
	if registry == nil {
		registry = NewIntentRegistry()
	}

	intentTypesSection := registry.FormatPromptSection()

	return fmt.Sprintf(`You are an intelligent intent classifier for a T-A-O (Think-Act-Observe) agent system.

<instructions>
Analyze the user's input and produce a structured intent classification.
CRITICAL: Your output content (summary, clarification_question, entity values) MUST be in the same language as the user input.
Focus on semantic understanding, not keyword matching.
</instructions>

<intent_types>
%s</intent_types>

<key_principles>
- **Primary signal is semantic intent**, not surface keywords. "I was wondering if you could..." is a Task, not Chat.
- **Compound inputs**: If the input contains both a new request AND a reference to previous context, classify as the more actionable intent (prefer task > follow_up).
- **Implicit clarification**: If the conversation history shows a pending clarification question and the user's input naturally answers it, classify as clarification even without explicit acknowledgment.
- **Edge cases**: Short inputs like "continue" or "and then?" should be follow_up. "OK" or "yes" in response to a system question should be clarification.
- **Ambiguity handling**: If the intent is clear but critical parameters are missing, classify the intent normally AND set requires_clarification to true with specific missing_slots.
</key_principles>

<entity_extraction_rules>
Extract key entities from the user's input as key-value pairs:
- Entity keys should be descriptive and lowercase (e.g., "location", "time_range", "target_name", "query_keyword")
- Only extract entities that are explicitly mentioned or clearly implied
- If no extractable entities exist, return an empty object {}
- Do NOT hallucinate entities that are not supported by the input
</entity_extraction_rules>

<examples>
<example>
<input>Search for flights from Beijing to Shanghai</input>
<output>{"type":"task","confidence":0.95,"entities":{"departure":"Beijing","destination":"Shanghai","subject":"flights"},"summary":"Search for flights from Beijing to Shanghai","topic":"travel","requires_clarification":false,"missing_slots":[],"clarification_question":"","reference_id":"","reference_type":""}</output>
</example>

<example>
<input>Hey, how have you been?</input>
<output>{"type":"chat","confidence":0.98,"entities":{},"summary":"Casual greeting and small talk","topic":"greeting","requires_clarification":false,"missing_slots":[],"clarification_question":"","reference_id":"","reference_type":""}</output>
</example>

<example>
<input>Use SF Express please</input>
<conversation_context>System asked: "Which courier would you like to use for shipping?"</conversation_context>
<output>{"type":"clarification","confidence":0.97,"entities":{"courier":"SF Express"},"summary":"Choose SF Express as the shipping method","topic":"order_management","requires_clarification":false,"missing_slots":[],"clarification_question":"","reference_id":"","reference_type":""}</output>
</example>

<example>
<input>Can you elaborate on that last result?</input>
<output>{"type":"follow_up","confidence":0.92,"entities":{},"summary":"Request for more details on the previous result","topic":"","requires_clarification":false,"missing_slots":[],"clarification_question":"","reference_id":"","reference_type":"previous_result"}</output>
</example>

<example>
<input>No, I meant yesterday's data not today's</input>
<output>{"type":"feedback","confidence":0.90,"entities":{"correction_target":"time range","corrected_value":"yesterday"},"summary":"Correcting the time range from today to yesterday","topic":"","requires_clarification":false,"missing_slots":[],"clarification_question":"","reference_id":"","reference_type":"previous_result"}</output>
</example>

<example>
<input>Send an email</input>
<output>{"type":"task","confidence":0.90,"entities":{"action":"send email"},"summary":"Send an email but recipient and content are missing","topic":"communication","requires_clarification":true,"missing_slots":["recipient","content"],"clarification_question":"Sure, who should I send it to and what should the content be?","reference_id":"","reference_type":""}</output>
</example>
</examples>

<current_input>
User input: %s
Conversation context: %s
</current_input>

<output_format>
Return ONLY a valid JSON object, no markdown, no code blocks, no explanation:
{"type":"<intent_type>","confidence":<0.0-1.0>,"entities":{...},"summary":"<one sentence>","topic":"<topic tag>","requires_clarification":<bool>,"missing_slots":[...],"clarification_question":"<question or empty>","reference_id":"<id or empty>","reference_type":"<type or empty>"}
</output_format>`, intentTypesSection, input, context)
}
