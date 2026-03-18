package memory

import (
	"context"
)

// Manager defines the interface for long-term semantic memory storage and retrieval.
// Advanced clients can implement this using Vector Databases (RAG) to provide
// intelligent context recall based on the user's current intent or query.
type Manager interface {
	// Store saves a specific piece of information or preference to the user's long-term memory.
	Store(ctx context.Context, sessionID string, key string, value any) error

	// Retrieve gets a specific key-value pair from memory (useful for exact matches/state).
	Retrieve(ctx context.Context, sessionID string, key string) (any, error)

	// Recall dynamically searches and retrieves relevant memory fragments based on
	// the current semantic context/intent (e.g., "What did the user say about their diet last week?").
	// Returns a string summarizing the relevant memories, ready to be injected into the LLM prompt.
	Recall(ctx context.Context, sessionID string, intent string) (string, error)

	// Update modifies an existing memory entry's weight or content.
	// Used by the ReAct pipeline (Observer/Terminator) to reinforce successful behaviors
	// (increase weight) or penalize hallucinations/failures (decrease weight, naturally decaying to 0).
	Update(ctx context.Context, sessionID string, key string, deltaWeight float64) error

	// Compress summarizes or prunes older/less relevant memories to save space.
	Compress(ctx context.Context, sessionID string) error

	// Persist forces an immediate write to the underlying storage mechanism.
	Persist(ctx context.Context, sessionID string) error

	// Load pulls the latest state from the underlying storage.
	Load(ctx context.Context, sessionID string) error
}
