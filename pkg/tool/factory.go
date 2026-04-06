package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/DotNetAge/goreact/pkg/common"
)

// ToolFactory creates Tool instances on demand.
// It replaces the global registry pattern with on-demand instantiation.
type ToolFactory struct {
	mu          sync.RWMutex
	constructors map[string]ToolConstructor
}

// ToolConstructor is a function that creates a Tool instance
type ToolConstructor func() Tool

// NewToolFactory creates a new ToolFactory
func NewToolFactory() *ToolFactory {
	return &ToolFactory{
		constructors: make(map[string]ToolConstructor),
	}
}

// Register registers a tool constructor
func (f *ToolFactory) Register(name string, constructor ToolConstructor) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.constructors[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	f.constructors[name] = constructor
	return nil
}

// MustRegister registers a tool constructor, panics on error
func (f *ToolFactory) MustRegister(name string, constructor ToolConstructor) {
	if err := f.Register(name, constructor); err != nil {
		panic(err)
	}
}

// Create creates a Tool instance by name
func (f *ToolFactory) Create(name string) (Tool, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	constructor, exists := f.constructors[name]
	if !exists {
		return nil, false
	}

	return constructor(), true
}

// GetConstructor returns the constructor for a tool
func (f *ToolFactory) GetConstructor(name string) (ToolConstructor, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	constructor, exists := f.constructors[name]
	return constructor, exists
}

// List returns all registered tool names
func (f *ToolFactory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.constructors))
	for name := range f.constructors {
		names = append(names, name)
	}
	return names
}

// RegisterBuiltins registers built-in tools to the factory
func (f *ToolFactory) RegisterBuiltins() {
	// File system tools
	f.MustRegister("bash", func() Tool { return NewBashTool("") })
	f.MustRegister("read", func() Tool { return NewReadTool() })
	f.MustRegister("write", func() Tool { return NewWriteTool() })
	f.MustRegister("glob", func() Tool { return NewGlobTool() })
	f.MustRegister("list", func() Tool { return NewListTool() })
	f.MustRegister("delete", func() Tool { return NewDeleteTool() })
}

// ToolInfoAccessor provides access to tool metadata from Memory
type ToolInfoAccessor interface {
	Get(ctx context.Context, toolName string) (*ToolNode, error)
	List(ctx context.Context) ([]*ToolNode, error)
}

// HybridToolFactory creates tools from both registered constructors and memory metadata
type HybridToolFactory struct {
	factory     *ToolFactory
	infoAccessor ToolInfoAccessor
}

// NewHybridToolFactory creates a new HybridToolFactory
func NewHybridToolFactory(factory *ToolFactory, accessor ToolInfoAccessor) *HybridToolFactory {
	return &HybridToolFactory{
		factory:      factory,
		infoAccessor: accessor,
	}
}

// GetOrCreate gets a tool from factory or creates from memory metadata
func (h *HybridToolFactory) GetOrCreate(ctx context.Context, name string) (Tool, error) {
	// First, try to create from registered constructor
	if tool, ok := h.factory.Create(name); ok {
		return tool, nil
	}

	// If not found, try to create from memory metadata
	if h.infoAccessor != nil {
		toolNode, err := h.infoAccessor.Get(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("tool %s not found: %w", name, err)
		}

		// Create tool from node metadata
		return h.createFromNode(toolNode)
	}

	return nil, fmt.Errorf("tool %s not found", name)
}

// createFromNode creates a Tool instance from ToolNode metadata
func (h *HybridToolFactory) createFromNode(node *ToolNode) (Tool, error) {
	// For dynamic tools defined in memory, we create a DynamicTool
	// that wraps the metadata and executes based on the tool type
	if node == nil {
		return nil, fmt.Errorf("nil tool node")
	}

	return &DynamicTool{
		node: node,
	}, nil
}

// DynamicTool represents a tool defined dynamically in memory
type DynamicTool struct {
	node *ToolNode
}

// Name returns the tool name
func (t *DynamicTool) Name() string {
	return t.node.Name
}

// Type returns the node type
func (t *DynamicTool) Type() string {
	return t.node.NodeType
}

// Properties returns the node properties
func (t *DynamicTool) Properties() map[string]any {
	return map[string]any{
		"description":     t.node.Description,
		"type":            string(t.node.Type),
		"security_level":  t.node.SecurityLevel.String(),
		"is_idempotent":   t.node.IsIdempotent,
	}
}

// Description returns the tool description
func (t *DynamicTool) Description() string {
	return t.node.Description
}

// SecurityLevel returns the security level
func (t *DynamicTool) SecurityLevel() common.SecurityLevel {
	return t.node.SecurityLevel
}

// IsIdempotent returns whether the tool is idempotent
func (t *DynamicTool) IsIdempotent() bool {
	return t.node.IsIdempotent
}

// Run executes the dynamic tool
// Note: Dynamic tools need an executor to be set via SetExecutor
func (t *DynamicTool) Run(ctx context.Context, params map[string]any) (any, error) {
	// Dynamic tools require an external executor (e.g., HTTP client, script runner)
	// This is a placeholder - actual execution depends on the tool type
	return nil, fmt.Errorf("dynamic tool %s requires an executor", t.node.Name)
}

// Global tool factory instance
var globalToolFactory = NewToolFactory()

// GetToolFactory returns the global tool factory
func GetToolFactory() *ToolFactory {
	return globalToolFactory
}

// InitBuiltins initializes built-in tools in the global factory
func InitBuiltins() {
	globalToolFactory.RegisterBuiltins()
}
