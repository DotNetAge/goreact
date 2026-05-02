package reactor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Decision constants for Thought.Decision
const (
	DecisionAct         = "act"
	DecisionAnswer      = "answer"
	DecisionClarify     = "clarify"
	DecisionDelegate    = "delegate"
	DecisionCoordinate  = "coordinate" // WBS decomposed → enter Coordinator mode to dispatch & monitor sub-tasks
)

// Thought represents the output of the Think phase.
// In two-phase thinking, Phase 1 produces SelectedSkill only,
// and Phase 2 produces the full decision/action/answer fields.
type Thought struct {
	IdentID     string  `json:"ident_id,omitempty" yaml:"ident_id"`
	Content     string  `json:"content,omitempty" yaml:"content"`
	Reasoning   string  `json:"reasoning" yaml:"reasoning"`
	Decision    string  `json:"decision" yaml:"decision"`
	Confidence  float64 `json:"confidence" yaml:"confidence"`
	IsFinal     bool    `json:"is_final" yaml:"is_final"`
	FinalAnswer string  `json:"final_answer,omitempty" yaml:"final_answer"`

	// ActionTarget + ActionParams: single tool call (legacy path, backward compatible)
	ActionTarget string         `json:"action_target,omitempty" yaml:"action_target"`
	ActionParams map[string]any `json:"action_params,omitempty" yaml:"action_params"`

	// ToolCalls holds multiple tool calls for batch parallel execution (v2).
	// Map key = tool name, value = parameter map.
	// When set, Act executes all tools in parallel: sync tools wait for result,
	// async tools (IsAsync=true) run in goroutines and return {task_id, status: "running"}.
	ToolCalls map[string]map[string]any `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`

	ClarificationQuestion string `json:"clarification_question,omitempty" yaml:"clarification_question"`

	DelegateTarget string `json:"delegate_target,omitempty" yaml:"delegate_target"`
	DelegatePrompt string `json:"delegate_prompt,omitempty" yaml:"delegate_prompt"`

	SelectedSkill string    `json:"selected_skill,omitempty" yaml:"selected_skill"`
	Timestamp     time.Time `json:"timestamp" yaml:"timestamp"`
}

// jsonBlockRegex matches ```json ... ``` code blocks.
var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*\n?(.*?)\n?\\s*```")

// stripJSONWrappers removes markdown code fences and leading/trailing whitespace from LLM output.
func stripJSONWrappers(s string) string {
	s = strings.TrimSpace(s)
	if m := jsonBlockRegex.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	return s
}

// ParseThinkResponse parses an LLM response string into a Thought struct.
func ParseThinkResponse(content string) (*Thought, error) {
	content = stripJSONWrappers(content)

	var thought Thought
	if err := json.Unmarshal([]byte(content), &thought); err != nil {
		return nil, fmt.Errorf("failed to parse thought JSON: %w\nraw: %s", err, truncate(content, 200))
	}

	// Normalize decision
	thought.Decision = strings.ToLower(strings.TrimSpace(thought.Decision))
	switch thought.Decision {
	case DecisionAct, DecisionAnswer, DecisionClarify, DecisionDelegate:
		// valid
	default:
		thought.Decision = DecisionAnswer
	}

	if thought.Timestamp.IsZero() {
		thought.Timestamp = time.Now()
	}

	return &thought, nil
}

// truncate shortens a string to maxLen runes for error messages.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
