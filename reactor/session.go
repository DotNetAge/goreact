package reactor

import (
	"context"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ensureContextWindow returns the active ContextWindow, creating one lazily if needed.
func (r *Reactor) ensureContextWindow(sessionID string) *core.ContextWindow {
	if r.contextWindow == nil {
		r.contextWindow = core.NewContextWindow(sessionID, int64(r.config.MaxTokens))
	}
	return r.contextWindow
}

// persistMessage writes a message to both ContextWindow and SessionStore.
// This is called for every user/assistant message to maintain the sliding window.
// Also accounts for the message's token cost in ContextWindow.
func (r *Reactor) persistMessage(ctx context.Context, role, content string) {
	if r.sessionStore == nil {
		return
	}

	sessionID := "default"
	if r.contextWindow != nil {
		sessionID = r.contextWindow.SessionID
	}
	cw := r.ensureContextWindow(sessionID)
	msg := core.Message{Role: role, Content: content, Timestamp: time.Now().Unix()}
	cw.AddMessageWithTimestamp(role, content, msg.Timestamp)
	r.sessionStore.Append(ctx, cw.SessionID, msg)

	tokens := int64(r.tokenEstimator.Estimate(content))
	if tokens > 0 {
		cw.AddTokens(tokens)
	}
}

// checkSlide checks whether the context window needs sliding and triggers it.
// Slid messages are forwarded to the SessionStore's SlideHandler for RAG/Memory processing.
func (r *Reactor) checkSlide(ctx context.Context) {
	if r.sessionStore == nil || r.contextWindow == nil {
		return
	}

	if !r.contextWindow.SlideTriggered(r.slideConfig) {
		return
	}

	estimateFn := func(s string) int { return r.tokenEstimator.Estimate(s) }
	slided := r.contextWindow.Slide(r.slideConfig, estimateFn)

	if len(slided.Messages) > 0 {
		event := core.SlideEvent{
			SessionID: r.contextWindow.SessionID,
			Slided:    slided.Messages,
			Remaining: r.contextWindow.MessageCount(),
			Timestamp: time.Now().Unix(),
		}
		core.EmitSlideEvent(nil, ctx, event)
	}
}
