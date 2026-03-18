package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool represents a single functional capability that the Agent can execute.
type Tool interface {
	// Name must return the exact unique string that the Thinker uses to invoke this tool.
	Name() string

	// Description should provide a clear semantic explanation of what the tool does,
	// and what its inputs/outputs are. (Useful for the Thinker's System Prompt).
	Description() string

	// Execute performs the actual work. It takes a raw map of inputs
	// (unmarshaled from the LLM's JSON ActionInput) and returns a raw result or error.
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

// Manager defines the interface for tool discovery and retrieval.
// Advanced clients can implement this using Vector Databases for Tool RAG (Semantic Matching).
type Manager interface {
	// GetTool retrieves a specific tool by its exact name (used by the Actor).
	GetTool(name string) (Tool, bool)

	// ListAvailableTools returns a list of tools relevant to the current context.
	// In a simple setup, this returns all tools.
	// In a RAG setup, it dynamically fetches the Top-K relevant tools based on the user intent.
	ListAvailableTools(ctx context.Context, intent string) ([]Tool, error)
}

// MapTool is a helper implementation of Tool that wraps a simple function.
type MapTool struct {
	ToolName        string
	ToolDescription string
	ExecuteFunc     func(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

func (t *MapTool) Name() string        { return t.ToolName }
func (t *MapTool) Description() string { return t.ToolDescription }
func (t *MapTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	if t.ExecuteFunc == nil {
		return nil, fmt.Errorf("tool %q has no execution logic", t.ToolName)
	}
	return t.ExecuteFunc(ctx, input)
}

// ExtractInput is a generic utility to help Tools parse their incoming map into a strict struct.
func ExtractInput(input map[string]interface{}, target interface{}) error {
	b, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal map: %w", err)
	}
	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("failed to unmarshal into strict struct: %w", err)
	}
	return nil
}

// =======================
// Default Basic Manager
// =======================

// SimpleManager holds an in-memory map of tools. It does not perform semantic RAG.
type SimpleManager struct {
	registry map[string]Tool
}

func NewSimpleManager() *SimpleManager {
	return &SimpleManager{
		registry: make(map[string]Tool),
	}
}

func (m *SimpleManager) Register(t ...Tool) {
	for _, t := range t {
		m.registry[t.Name()] = t
	}
}

func (m *SimpleManager) GetTool(name string) (Tool, bool) {
	t, ok := m.registry[name]
	return t, ok
}

func (m *SimpleManager) ListAvailableTools(ctx context.Context, intent string) ([]Tool, error) {
	var list []Tool
	for _, t := range m.registry {
		list = append(list, t)
	}
	return list, nil
}
