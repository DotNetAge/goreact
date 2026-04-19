package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// AskUser is a special tool that the LLM can call when it needs user clarification.
// Unlike normal tools, this tool BLOCKS until the user provides an answer via a channel.
// The user's answer is then returned as the tool result, and the T-A-O loop continues naturally.
//
// Usage pattern:
//
//	askTool := NewAskUserTool()
//	// Set the event callback so the tool can emit ClarifyNeeded events
//	askTool.SetEventEmitter(func(e core.ReactEvent) { eventBus.Emit(e) })
//	reactor.RegisterTool(askTool)
//	// ... later, user provides answer:
//	askTool.Respond("user's answer")
type AskUser struct {
	info         *core.ToolInfo
	eventEmitter func(core.ReactEvent) // optional: set by reactor to emit events

	mu      sync.Mutex
	pending *askRequest
}

type askRequest struct {
	question string
	answer   string
	done     chan struct{}
	err      error
}

// NewAskUserTool creates a new AskUser tool.
func NewAskUserTool() core.FuncTool {
	return &AskUser{
		info: &core.ToolInfo{
			Name:        "ask_user",
			Description: "Ask the user a clarifying question when information is missing or ambiguous. The tool will block until the user provides an answer. Use this when you are uncertain about the user's intent, need more details, or want to confirm your understanding before proceeding. Prefer concise, specific questions.",
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

// SetEventEmitter sets a callback for emitting ReactEvents (e.g., ClarifyNeeded).
// Called by the reactor during tool registration.
func (t *AskUser) SetEventEmitter(fn func(core.ReactEvent)) {
	t.eventEmitter = fn
}

func (t *AskUser) Info() *core.ToolInfo {
	return t.info
}

// Execute blocks until the user provides a response via Respond().
//
// Interrupt-resume flow:
//  1. LLM calls ask_user tool
//  2. Tool emits ClarifyNeeded event, then blocks on channel
//  3. Client receives event, shows question to user
//  4. External code calls Respond() with user's answer
//  5. Tool unblocks, returns answer as tool_result
//  6. T-A-O loop continues with the answer in context
func (t *AskUser) Execute(ctx context.Context, params map[string]any) (any, error) {
	question, ok := params["question"].(string)
	if !ok || question == "" {
		return "", fmt.Errorf("missing required parameter: question")
	}

	t.mu.Lock()
	if t.pending != nil {
		t.mu.Unlock()
		return "", fmt.Errorf("ask_user is already waiting for a response")
	}

	req := &askRequest{
		question: question,
		done:     make(chan struct{}),
	}
	t.pending = req
	t.mu.Unlock()

	// Emit ClarifyNeeded event so clients know to prompt the user
	if t.eventEmitter != nil {
		t.eventEmitter(core.NewReactEvent("", "main", "", core.ClarifyNeeded, question))
	}

	// Block until the user responds, context is cancelled, or timeout
	select {
	case <-req.done:
		t.mu.Lock()
		t.pending = nil
		t.mu.Unlock()
		if req.err != nil {
			return "", req.err
		}
		return fmt.Sprintf("User answered: %s", req.answer), nil

	case <-ctx.Done():
		t.mu.Lock()
		t.pending = nil
		t.mu.Unlock()
		return "", fmt.Errorf("ask_user cancelled: %w", ctx.Err())
	}
}

// Respond delivers the user's answer to a waiting AskUser execution.
// Returns an error if the tool is not currently waiting for input.
func (t *AskUser) Respond(answer string) error {
	t.mu.Lock()
	req := t.pending
	if req == nil {
		t.mu.Unlock()
		return fmt.Errorf("ask_user is not currently waiting for input")
	}
	req.answer = answer
	close(req.done)
	t.mu.Unlock()
	return nil
}

// RespondError delivers an error to a waiting AskUser execution.
func (t *AskUser) RespondError(err error) error {
	t.mu.Lock()
	req := t.pending
	if req == nil {
		t.mu.Unlock()
		return fmt.Errorf("ask_user is not currently waiting for input")
	}
	req.err = err
	close(req.done)
	t.mu.Unlock()
	return nil
}

// IsWaiting returns true if the tool is currently blocking, waiting for user input.
func (t *AskUser) IsWaiting() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pending != nil
}

// WaitWithTimeout waits for the tool to become ready (not waiting).
func (t *AskUser) WaitWithTimeout(timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		t.mu.Lock()
		waiting := t.pending != nil
		t.mu.Unlock()
		if !waiting {
			return nil
		}
		select {
		case <-deadline:
			return fmt.Errorf("timed out waiting for ask_user to complete")
		case <-time.After(50 * time.Millisecond):
		}
	}
}
