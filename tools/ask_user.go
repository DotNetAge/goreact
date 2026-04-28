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
			Description: "Ask the user a clarifying question when information is missing or ambiguous. The tool will return a structured question request and the system will wait for the user's response before continuing.",
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
