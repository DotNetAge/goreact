package reactor

import (
	"context"
	"time"

	"github.com/DotNetAge/goreact/core"
	"github.com/google/uuid"
)

// ensureContextWindow returns the active ContextWindow, creating one lazily if needed.
func (r *Reactor) ensureContextWindow(sessionID string) *core.ContextWindow {
	if r.contextWindow == nil {
		r.contextWindow = core.NewContextWindow(sessionID, int64(r.config.MaxTokens))
	}
	return r.contextWindow
}

// persistMessage writes a message to both ContextWindow and SessionStore.
//
// This is called during T-A-O loop execution for each intermediate assistant message
// (e.g., step summaries). It is distinct from Agent-layer persistMessage which handles
// user questions and final answers at session lifecycle boundaries.
//
// Both layers coexist because:
//   - Agent layer: owns session identity, manages user input / final output persistence
//   - Reactor layer: owns execution-loop intermediate step persistence and sliding
func (r *Reactor) persistMessage(ctx context.Context, role, content string) {
	if r.sessionStore == nil {
		return
	}

	sessionID := uuid.NewString()
	if r.contextWindow != nil {
		sessionID = r.contextWindow.SessionID
	}
	cw := r.ensureContextWindow(sessionID)
	msg := core.Message{Role: role, Content: content, Timestamp: time.Now().Unix()}
	cw.AddMessageWithTimestamp(role, content, msg.Timestamp)

	agentName := ""
	if r.contextWindow != nil {
		agentName = r.contextWindow.Role
	}
	r.sessionStore.Append(ctx, cw.SessionID, agentName, msg)

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
		r.sessionStore.SetSlideHandler(func(ctx context.Context, e core.SlideEvent) {
			logger.Debug("context window slid",
				"sessionID", e.SessionID,
				"slidedCount", len(e.Slided),
				"remaining", e.Remaining)
		})
		core.EmitSlideEvent(nil, ctx, event)
	}
}
