package core

// InteractionType identifies the kind of human interaction requested by a tool.
type InteractionType string

const (
	// InteractionAskUser indicates the agent needs to ask the user a clarifying question.
	InteractionAskUser InteractionType = "ask_user"

	// InteractionAskPermission indicates the agent needs user authorization before
	// executing a high-risk operation.
	InteractionAskPermission InteractionType = "ask_permission"

	// InteractionConfirm asks the user to confirm or cancel a proposed action.
	InteractionConfirm InteractionType = "confirm"

	// InteractionSelect presents options for the user to choose from.
	InteractionSelect InteractionType = "select"
)

// InteractionRequest is returned by interaction-type tools (ask_user, ask_permission, etc.)
// instead of blocking internally. The Reactor's Act phase detects this request and
// manages the pause → wait → resume lifecycle.
//
// Tools that produce interactions should return their result as a map containing
// the "_interaction" key with an InteractionRequest value. Example:
//
//	return map[string]any{
//	    "_interaction": InteractionRequest{Type: InteractionAskUser, Question: "Which file?"},
//	    "status": "waiting",
//	}, nil
type InteractionRequest struct {
	Type     InteractionType `json:"type"`
	Question string          `json:"question"`
	Options  []string        `json:"options,omitempty"`
	ToolName string          `json:"tool_name"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}
