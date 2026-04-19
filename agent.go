package goreact

import (
	"context"

	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/reactor"
)

// Agent is the top-level facade for interacting with the ReAct agent system.
type Agent struct {
	config        *core.AgentConfig
	model         *core.ModelConfig
	memory        *core.Memory
	reactor       reactor.ReActor
	contextWindow *core.ContextWindow
}

// NewAgent creates a new Agent with the given configuration.
func NewAgent(config *core.AgentConfig,
	model *core.ModelConfig,
	memory *core.Memory,
	reactor reactor.ReActor) *Agent {
	return &Agent{
		config:  config,
		model:   model,
		memory:  memory,
		reactor: reactor,
	}
}

// NewAgentWithSession creates an Agent with a pre-initialized ContextWindow.
func NewAgentWithSession(config *core.AgentConfig,
	model *core.ModelConfig,
	memory *core.Memory,
	reactor reactor.ReActor,
	sessionID string,
	maxTokens int64) *Agent {
	return &Agent{
		config:        config,
		model:         model,
		memory:        memory,
		reactor:       reactor,
		contextWindow: core.NewContextWindow(sessionID, maxTokens),
	}
}

func (a *Agent) Name() string {
	return a.config.Name
}

func (a *Agent) Domain() string {
	return a.config.Domain
}

func (a *Agent) Description() string {
	return a.config.Description
}

// ContextWindow returns the agent's context window, or nil if no session is active.
func (a *Agent) ContextWindow() *core.ContextWindow {
	return a.contextWindow
}

// NewSession starts a new conversation session, replacing any existing one.
// The previous context is discarded.
func (a *Agent) NewSession(sessionID string, maxTokens int64) {
	a.contextWindow = core.NewContextWindow(sessionID, maxTokens)
}

// SessionID returns the current session ID, or empty string if no session.
func (a *Agent) SessionID() string {
	if a.contextWindow == nil {
		return ""
	}
	return a.contextWindow.SessionID
}

// Ask sends a question to the Agent and returns the answer.
// If a ContextWindow is active, the conversation history is automatically
// managed: user input and assistant response are appended to the window,
// and the history is pruned if it exceeds the token budget.
func (a *Agent) Ask(question string) (string, error) {
	// Build conversation history from ContextWindow if available
	var history reactor.ConversationHistory
	if a.contextWindow != nil {
		a.contextWindow.AddMessage("user", question)
		msgs := a.contextWindow.RecentMessages(0)
		history = make(reactor.ConversationHistory, len(msgs))
		for i, m := range msgs {
			history[i] = core.Message{Role: m.Role, Content: m.Content, Timestamp: m.Timestamp}
		}
	}

	// Delegate to the reactor for full T-A-O processing
	result, err := a.reactor.Run(context.TODO(), question, history)
	if err != nil {
		return "", err
	}

	// Record assistant response and token usage in ContextWindow
	if a.contextWindow != nil {
		if result.Answer != "" {
			a.contextWindow.AddMessage("assistant", result.Answer)
		}
		a.contextWindow.AddTokens(int64(result.TokensUsed))
		// Prune if over budget
		if a.contextWindow.TokensRemaining() <= 0 {
			a.contextWindow.Prune(nil)
		}
	}

	return result.Answer, nil
}

// AskWithContext is like Ask but accepts an explicit context.Context for cancellation.
func (a *Agent) AskWithContext(ctx context.Context, question string) (string, error) {
	var history reactor.ConversationHistory
	if a.contextWindow != nil {
		a.contextWindow.AddMessage("user", question)
		msgs := a.contextWindow.RecentMessages(0)
		history = make(reactor.ConversationHistory, len(msgs))
		for i, m := range msgs {
			history[i] = core.Message{Role: m.Role, Content: m.Content, Timestamp: m.Timestamp}
		}
	}

	result, err := a.reactor.Run(ctx, question, history)
	if err != nil {
		return "", err
	}

	if a.contextWindow != nil {
		if result.Answer != "" {
			a.contextWindow.AddMessage("assistant", result.Answer)
		}
		a.contextWindow.AddTokens(int64(result.TokensUsed))
		if a.contextWindow.TokensRemaining() <= 0 {
			a.contextWindow.Prune(nil)
		}
	}

	return result.Answer, nil
}

// Recognize identifies the user's intent from the given text.
func (a *Agent) Recognize(text string) (*reactor.Intent, error) {
	return nil, nil // TODO: implement using reactor's intent classification
}
