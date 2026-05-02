package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// AskUser is a pure-data tool that the LLM calls when it needs user clarification.
// Unlike the previous blocking design, this tool returns an InteractionRequest
// immediately without blocking. The Reactor's Act phase detects the interaction
// request and manages the pause → wait → resume lifecycle.
//
// This design decouples the tool from Reactor internals (no SetEventEmitter,
// no channels, no blocking). The tool is fully testable as a pure function.
type AskUser struct {
	info *core.ToolInfo
}

// NewAskUserTool creates a new AskUser tool.
func NewAskUserTool() core.FuncTool {
	return &AskUser{
		info: &core.ToolInfo{
			Name:        "ask_user",
			Description: "Asks the user multiple choice questions to gather information, clarify ambiguity, understand preferences, make decisions or offer them choices.",
			Prompt: `Use this tool when you need to ask the user questions during execution. This allows you to:
1. Gather user preferences or requirements
2. Clarify ambiguous instructions
3. Get decisions on implementation choices as you work
4. Offer choices to the user about what direction to take.

Usage notes:
- Users will always be able to select "Other" to provide custom text input
- Use multiSelect: true to allow multiple answers to be selected for a question
- If you recommend a specific option, make that the first option in the list and add "(Recommended)" at the end of the label`,
			Tags:        []string{"interaction", "question", "clarify", "human"},
			IsReadOnly:  true,
			Parameters: []core.Parameter{
				{
					Name:        "question",
					Type:        "string",
					Description: "The clarifying question to ask the user. Be specific and concise.",
					Required:    true,
				},
			},
		},
	}
}

func (t *AskUser) Info() *core.ToolInfo {
	return t.info
}

// Execute returns an InteractionRequest immediately.
// The Reactor's Act phase will detect this and manage human interaction.
func (t *AskUser) Execute(ctx context.Context, params map[string]any) (any, error) {
	question, ok := params["question"].(string)
	if !ok || question == "" {
		return nil, fmt.Errorf("missing required parameter: question")
	}

	return map[string]any{
		"_interaction": &core.InteractionRequest{
			Type:     core.InteractionAskUser,
			Question: question,
			ToolName: "ask_user",
		},
		"status": "waiting_for_user",
	}, nil
}
