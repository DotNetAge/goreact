package reactor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// Decision constants for Thought.Decision
const (
	DecisionAct     = "act"
	DecisionAnswer  = "answer"
	DecisionClarify = "clarify"
)

// Thought represents the output of the Think phase.
type Thought struct {
	IdentID     string         `json:"ident_id,omitempty" yaml:"ident_id"`
	Content     string         `json:"content,omitempty" yaml:"content"`
	Reasoning   string         `json:"reasoning" yaml:"reasoning"`
	Decision    string         `json:"decision" yaml:"decision"`
	Confidence  float64        `json:"confidence" yaml:"confidence"`
	IsFinal     bool           `json:"is_final" yaml:"is_final"`
	FinalAnswer string         `json:"final_answer,omitempty" yaml:"final_answer"`

	// Action fields (used when Decision == "act")
	ActionTarget string         `json:"action_target,omitempty" yaml:"action_target"`
	ActionParams map[string]any `json:"action_params,omitempty" yaml:"action_params"`

	// Clarification (used when Decision == "clarify")
	ClarificationQuestion string `json:"clarification_question,omitempty" yaml:"clarification_question"`

	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// BuildThinkPrompt constructs the Think phase prompt.
// It includes the classified intent, available tools, applicable skills, and user input.
func BuildThinkPrompt(input string, intent *Intent, tools []core.ToolInfo, skills []*core.Skill) string {
	intentSection := "(no intent)"
	if intent != nil {
		b, _ := json.Marshal(intent)
		intentSection = string(b)
	}

	toolSection := FormatToolDescriptions(tools)

	skillSection := ""
	if len(skills) > 0 {
		skillSection = "\n<activated_skills>\n"
		for _, s := range skills {
			skillSection += fmt.Sprintf("## Skill: %s\n%s\n", s.Name, s.Instructions)
		}
		skillSection += "</activated_skills>\n"
	}

	return fmt.Sprintf(`You are the Thinker in a T-A-O (Think-Act-Observe) agent system.

<role>
Based on the user's input, classified intent, and available tools, decide the next action.
CRITICAL: Your output content (reasoning, final_answer, clarification_question) MUST be in the same language as the user input.
Your decision must be one of:
- "act": Invoke a tool to fulfill the user's request
- "answer": Provide a direct answer from your knowledge
- "clarify": Ask the user for missing information
</role>

<rules>
1. If the intent is "chat" or "feedback", prefer "answer" unless tools would significantly enhance the response.
2. If the intent is "task", check if any tool can fulfill it -> "act", otherwise -> "answer".
3. If the intent is "clarification", extract the user's answer and -> "act" with the previously pending task.
4. If the intent is "follow_up", refer to conversation history and -> "act" or "answer" as appropriate.
5. Extract specific, well-typed parameters for tool calls from the user's input. Do NOT hallucinate parameters.
6. If required parameters are missing, -> "clarify" with specific questions about what's needed.
7. Set is_final to true ONLY when you have a complete, satisfactory answer to return to the user.
8. Always provide reasoning in the same language as the user input.
</rules>
%s
<intent>
%s
</intent>

<available_tools>
%s
</available_tools>

<current_input>
User input: %s
</current_input>

<output_format>
Return ONLY a valid JSON object, no markdown, no code blocks, no explanation:
{"decision":"act|answer|clarify","reasoning":"<reasoning process>","confidence":<0.0-1.0>,"action_target":"<tool_name or empty>","action_params":{...},"final_answer":"<answer or empty>","clarification_question":"<question or empty>","is_final":<bool>}
</output_format>`, skillSection, intentSection, toolSection, input)
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
	case DecisionAct, DecisionAnswer, DecisionClarify:
		// valid
	default:
		thought.Decision = DecisionAnswer
	}

	if thought.Timestamp.IsZero() {
		thought.Timestamp = time.Now()
	}

	return &thought, nil
}

// truncate shortens a string to maxLen characters for error messages.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
