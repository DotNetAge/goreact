package reactor

import (
	"context"
	"fmt"
	"sync"

	"github.com/DotNetAge/goreact/core"
)

// HumanInteractionHandler handles human interaction requests from tools.
// When a tool (e.g., ask_user) returns an InteractionRequest, the Reactor's
// Act phase delegates to this handler instead of the tool blocking internally.
//
// The handler is responsible for:
//  1. Emitting appropriate events (ClarifyNeeded, PermissionRequest)
//  2. Blocking until the user responds
//  3. Returning the user's answer (or error)
type HumanInteractionHandler interface {
	HandleInteraction(ctx context.Context, req *core.InteractionRequest) (string, error)
}

// DefaultInteractionHandler is the standard implementation that emits events
// and blocks on an internal channel until Respond() is called.
type DefaultInteractionHandler struct {
	mu      sync.Mutex
	pending *interactionPending
	eventEmitter func(core.ReactEvent)
}

type interactionPending struct {
	request *core.InteractionRequest
	answer  string
	err     error
	done    chan struct{}
}

// NewDefaultInteractionHandler creates a handler with optional event emitter.
func NewDefaultInteractionHandler(emitter func(core.ReactEvent)) *DefaultInteractionHandler {
	return &DefaultInteractionHandler{eventEmitter: emitter}
}

func (h *DefaultInteractionHandler) HandleInteraction(ctx context.Context, req *core.InteractionRequest) (string, error) {
	h.mu.Lock()
	if h.pending != nil {
		h.mu.Unlock()
		return "", fmt.Errorf("already waiting for an interaction response")
	}
	pending := &interactionPending{
		request: req,
		done:    make(chan struct{}),
	}
	h.pending = pending
	h.mu.Unlock()

	if h.eventEmitter != nil {
		switch req.Type {
		case core.InteractionAskUser:
			h.eventEmitter(core.NewReactEvent("", "main", "", core.ClarifyNeeded, req.Question))
		case core.InteractionAskPermission:
			h.eventEmitter(core.NewReactEvent("", "main", "", core.PermissionRequest, map[string]any{
				"tool_name": req.ToolName,
				"question":  req.Question,
			}))
		}
	}

	select {
	case <-pending.done:
		h.mu.Lock()
		h.pending = nil
		h.mu.Unlock()
		if pending.err != nil {
			return "", pending.err
		}
		return fmt.Sprintf("User answered: %s", pending.answer), nil
	case <-ctx.Done():
		h.mu.Lock()
		h.pending = nil
		h.mu.Unlock()
		return "", fmt.Errorf("interaction cancelled: %w", ctx.Err())
	}
}

// Respond delivers the user's answer to a waiting HandleInteraction call.
func (h *DefaultInteractionHandler) Respond(answer string) error {
	h.mu.Lock()
	p := h.pending
	h.mu.Unlock()
	if p == nil {
		return fmt.Errorf("not currently waiting for interaction")
	}
	p.answer = answer
	close(p.done)
	return nil
}

// RespondError delivers an error to a waiting HandleInteraction call.
func (h *DefaultInteractionHandler) RespondError(err error) error {
	h.mu.Lock()
	p := h.pending
	h.mu.Unlock()
	if p == nil {
		return fmt.Errorf("not currently waiting for interaction")
	}
	p.err = err
	close(p.done)
	return nil
}

// IsWaiting returns true if the handler is currently blocking for user input.
func (h *DefaultInteractionHandler) IsWaiting() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pending != nil
}
