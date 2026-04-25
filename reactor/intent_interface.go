package reactor

// IntentRegistry defines the contract for intent type management.
// Implementations can override classification behavior, e.g., using LLM-based
// semantic matching instead of the default keyword-based approach.
type IntentRegistry interface {
	// Register adds a new intent definition. Returns error if type already exists.
	Register(def IntentDefinition) error

	// Unregister removes an intent definition by type.
	Unregister(typ string)

	// All returns a copy of all registered intent definitions.
	All() []IntentDefinition

	// FormatPromptSection renders intents into the classification prompt.
	FormatPromptSection() string
}
