package reactor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ConversationHistory is a typed slice of core.Message for conversation history.
type ConversationHistory []core.Message

// Format renders the conversation history into a structured text block.
// maxTurns limits the number of recent messages included (0 = all).
func (h ConversationHistory) Format(maxTurns int) string {
	if len(h) == 0 {
		return "(no conversation history)"
	}
	messages := h
	if maxTurns > 0 && len(messages) > maxTurns {
		messages = messages[len(messages)-maxTurns:]
	}
	var sb strings.Builder
	for i, msg := range messages {
		ts := ""
		if msg.Timestamp > 0 {
			ts = time.Unix(msg.Timestamp, 0).Format(" 15:04:05")
		}
		fmt.Fprintf(&sb, "  [%d] %s%s: %s\n", i+1, msg.Role, ts, msg.Content)
	}
	return sb.String()
}

// ReactContext holds the shared state for a single Run invocation.
// It is created at the start of Run and mutated throughout the T-A-O loop.
type ReactContext struct {
	// Lifecycle
	ctx              context.Context
	cancel           context.CancelFunc
	CurrentIteration int
	MaxIterations    int

	// Input
	Input string
	ConversationHistory
	Intent *Intent

	// Last cycle results
	LastThought     *Thought
	LastAction      *Action
	LastObservation *Observation
	History         []Step

	// Termination
	IsTerminated      bool
	TerminationReason string

	// Thread safety for concurrent read access
	mu sync.RWMutex
}

// NewReactContext creates a new ReactContext for a Run invocation.
// If ctx is nil, context.Background() is used.
func NewReactContext(ctx context.Context, input string, history ConversationHistory, maxIter int) *ReactContext {
	if maxIter <= 0 {
		maxIter = core.DefaultMaxSteps
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	return &ReactContext{
		ctx:                 ctx,
		cancel:              cancel,
		Input:               input,
		ConversationHistory: history,
		MaxIterations:       maxIter,
		History:             make([]Step, 0, maxIter),
	}
}

// Ctx returns the context.Context for this run.
func (c *ReactContext) Ctx() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

// Cancel cancels the run context.
func (c *ReactContext) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// AppendHistory adds a completed step to the history.
func (c *ReactContext) AppendHistory(step Step) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.History = append(c.History, step)
}

// AddMessage appends a message to the conversation history.
func (c *ReactContext) AddMessage(role, content string) {
	c.ConversationHistory = append(c.ConversationHistory, core.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	})
}

// FormatToolDescriptions renders a slice of core.ToolInfo into text for prompt injection.
func FormatToolDescriptions(tools []core.ToolInfo) string {
	if len(tools) == 0 {
		return "(no tools available)"
	}
	var sb strings.Builder
	for i, t := range tools {
		fmt.Fprintf(&sb, "%d. **%s**: %s\n", i+1, t.Name, t.Description)
	}
	return sb.String()
}
