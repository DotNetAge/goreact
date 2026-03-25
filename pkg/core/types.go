package core

// JSONSchema is a map representation of a JSON Schema object.
type JSONSchema map[string]any

// Action represents a specific tool call or plan decided by the Thinker.
type Action struct {
	// Name is the registered tool or skill name (e.g., "Calculator", "SearchEngine").
	Name string
	// Input holds the structured parameters for the action, often unmarshaled from JSON.
	Input map[string]any
}

// Observation represents the standardized, noise-reduced result after Observer processing.
type Observation struct {
	// Data is the cleaned, truncated, and summarized text ready to be fed back to the LLM.
	Data string
	// Raw holds the original, unprocessed output from the Actor (for potential debugging or multimodal uses).
	Raw any
	// Error stores any system or tool-level error that occurred during execution.
	// If present, the Observer usually translates this into a semantic string within Data.
	Error error
	// IsSuccess indicates whether the Action was executed without technical failure.
	IsSuccess bool
}

// Trace represents a single complete iteration step in the ReAct loop.
// It acts as the "Scratchpad" or Short-Term Memory for the Agent.
type Trace struct {
	// Step is the iteration number in the current session.
	Step int
	// Thought is the explicit reasoning output from the LLM (Chain of Thought).
	Thought string
	// Action is the specific task/tool the Agent decided to execute.
	Action *Action
	// Observation is the sensory feedback generated after executing the Action.
	// It is nil until the Observer processes the Actor's results.
	Observation *Observation
}
