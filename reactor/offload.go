package reactor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// offloadThreshold is the maximum result size (in bytes) kept in context.
// Results exceeding this threshold are written to disk and replaced with a reference.
const offloadThreshold = 30 * 1024 // 30K chars

// offloadDirName is the directory under which offloaded files are stored.
const offloadDirName = ".goreact" + string(os.PathSeparator) + "offload"

// offloadPrefix is the marker prefix for offload reference text in message content.
const offloadPrefix = "[offload:"

// offloadSuffix is the marker suffix for offload reference text.
const offloadSuffix = "]"

// resultExceedsThreshold checks if a tool result string exceeds the offload threshold.
func resultExceedsThreshold(result string) bool {
	return len(result) > offloadThreshold
}

// offloadResult writes a large tool result to disk and returns a structured reference
// string that can be injected into the context in place of the full result.
//
// Reference format: [offload:/path/to/file:size:preview]
func offloadResult(ctx context.Context, sessionID, toolName, result string) string {
	offloadDir := offloadPath(sessionID)
	if err := os.MkdirAll(offloadDir, 0755); err != nil {
		// If we can't create the offload directory, keep the result in context
		return result
	}

	filename := fmt.Sprintf("%s_%d.out", toolName, time.Now().UnixNano())
	filePath := filepath.Join(offloadDir, filename)

	if err := os.WriteFile(filePath, []byte(result), 0644); err != nil {
		return result
	}

	preview := result
	if len(preview) > 200 {
		preview = preview[:200]
	}

	ref := fmt.Sprintf("[offload:%s:%d:%s]", filePath, len(result), preview)
	return ref
}

// isOffloadReference checks if a string is an offload reference marker.
func isOffloadReference(content string) bool {
	return strings.HasPrefix(content, offloadPrefix) && strings.HasSuffix(content, offloadSuffix)
}

// restoreOffloadResult reads the full result from an offload reference.
// Returns the restored content and true on success, or the original reference and false on error.
func restoreOffloadResult(ref string) (string, bool) {
	// Format: [offload:path:size:preview]
	inner := ref[len(offloadPrefix) : len(ref)-len(offloadSuffix)]
	parts := strings.SplitN(inner, ":", 3)
	if len(parts) < 2 {
		return ref, false
	}

	filePath := parts[0]
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ref, false
	}

	return string(data), true
}

// offloadPath returns the directory path for offloaded files for a session.
func offloadPath(sessionID string) string {
	return filepath.Join(offloadDirName, sessionID)
}

// offloadLargeResults checks all actions in the context for large results and offloads them.
// Called from persistStep after Act completes.
func (r *Reactor) offloadLargeResults(ctx *ReactContext) {
	if ctx.LastAction == nil || ctx.LastAction.Result == "" {
		return
	}
	if !resultExceedsThreshold(ctx.LastAction.Result) {
		return
	}

	sessionID := r.resolveSessionID(ctx)
	ref := offloadResult(ctx.Ctx(), sessionID, ctx.LastAction.Target, ctx.LastAction.Result)
	if ref != ctx.LastAction.Result {
		// Offload succeeded, update the action result with the reference
		ctx.LastAction.Result = ref
	}
}

// restoreOffloadedResults scans the conversation history for offload references
// and replaces them with the full file content. Called at the start of each Think phase.
func (r *Reactor) restoreOffloadedResults(ctx *ReactContext) {
	for i := range ctx.ConversationHistory {
		msg := &ctx.ConversationHistory[i]
		if isOffloadReference(msg.Content) {
			if restored, ok := restoreOffloadResult(msg.Content); ok {
				msg.Content = restored
			}
		}
	}
}

// resolveSessionID returns a session identifier for offload directory naming.
func (r *Reactor) resolveSessionID(ctx *ReactContext) string {
	if ctx.SessionID != "" {
		return ctx.SessionID
	}
	if cw := r.llmCaller.ContextWindow(); cw != nil && cw.SessionID != "" {
		return cw.SessionID
	}
	return ctx.TaskID
}
