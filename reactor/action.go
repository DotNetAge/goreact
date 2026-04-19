package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ActionType identifies the kind of action taken.
type ActionType string

// ActionType constants.
const (
	ActionTypeToolCall ActionType = "tool_call"
	ActionTypeAnswer   ActionType = "answer"
	ActionTypeClarify  ActionType = "clarify"
)

// Action represents the output of the Act phase.
type Action struct {
	Type      ActionType     `json:"type" yaml:"type"`
	Target    string         `json:"target" yaml:"target"`       // Tool name (for tool_call)
	Params    map[string]any `json:"params" yaml:"params"`       // Tool parameters
	Result    string         `json:"result" yaml:"result"`       // Execution result
	Error     error          `json:"-" yaml:"-"`                 // Execution error (not serialized)
	ErrorMsg  string         `json:"error,omitempty" yaml:"error,omitempty"` // Serialized error message
	Duration  time.Duration  `json:"duration" yaml:"duration"`   // Execution duration
	Timestamp time.Time      `json:"timestamp" yaml:"timestamp"` // When the action was taken
}

// ToolRegistry manages available core.FuncTool instances. It is safe for concurrent use.
// This registry wraps core.FuncTool (which returns (any, error)) and normalizes the
// result to (string, error) for the reactor's consumption.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]core.FuncTool
}

// NewToolRegistry creates an empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]core.FuncTool)}
}

// Register adds a core.FuncTool. Returns error if a tool with the same name exists.
func (r *ToolRegistry) Register(tool core.FuncTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := tool.Info().Name
	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// Get returns a tool by name.
func (r *ToolRegistry) Get(name string) (core.FuncTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools.
func (r *ToolRegistry) All() []core.FuncTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]core.FuncTool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// ToToolInfos extracts core.ToolInfo from all registered tools for prompt building.
func (r *ToolRegistry) ToToolInfos() []core.ToolInfo {
	tools := r.All()
	infos := make([]core.ToolInfo, len(tools))
	for i, t := range tools {
		infos[i] = *t.Info()
	}
	return infos
}

// ExecuteTool runs a tool by name with the given parameters.
// It normalizes the core.FuncTool result (any) to a string via JSON marshaling.
func (r *ToolRegistry) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, time.Duration, error) {
	tool, ok := r.Get(name)
	if !ok {
		return "", 0, fmt.Errorf("tool %q not found", name)
	}
	start := time.Now()
	result, err := tool.Execute(ctx, params)
	duration := time.Since(start)
	if err != nil {
		return "", duration, err
	}
	// Normalize any result to string
	str, ok := result.(string)
	if !ok {
		b, err := json.Marshal(result)
		if err != nil {
			return fmt.Sprintf("%v", result), duration, nil
		}
		str = string(b)
	}
	return str, duration, nil
}
