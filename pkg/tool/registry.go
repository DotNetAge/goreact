package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/DotNetAge/goreact/pkg/common"
)

// Registry manages tool registration and retrieval
type Registry struct {
	mu     sync.RWMutex
	tools  map[string]Tool
	infos  map[string]*ToolInfo
}

// NewRegistry creates a new Registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		infos: make(map[string]*ToolInfo),
	}
}

// Register registers a tool
func (r *Registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := t.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}
	
	r.tools[name] = t
	r.infos[name] = GetToolInfo(t)
	return nil
}

// Unregister unregisters a tool
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.tools, name)
	delete(r.infos, name)
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	t, exists := r.tools[name]
	return t, exists
}

// GetInfo retrieves tool info by name
func (r *Registry) GetInfo(name string) (*ToolInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	info, exists := r.infos[name]
	return info, exists
}

// List lists all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ListInfo lists all tool info
func (r *Registry) ListInfo() []*ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	infos := make([]*ToolInfo, 0, len(r.infos))
	for _, info := range r.infos {
		infos = append(infos, info)
	}
	return infos
}

// ListBySecurityLevel lists tools by security level
func (r *Registry) ListBySecurityLevel(level common.SecurityLevel) []*ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	infos := []*ToolInfo{}
	for _, info := range r.infos {
		if info.SecurityLevel == level {
			infos = append(infos, info)
		}
	}
	return infos
}

// Clear clears all registered tools
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.tools = make(map[string]Tool)
	r.infos = make(map[string]*ToolInfo)
}

// Global registry instance
var globalRegistry = NewRegistry()

// Register registers a tool to the global registry
func Register(t Tool) error {
	return globalRegistry.Register(t)
}

// Unregister unregisters a tool from the global registry
func Unregister(name string) {
	globalRegistry.Unregister(name)
}

// Get retrieves a tool from the global registry
func Get(name string) (Tool, bool) {
	return globalRegistry.Get(name)
}

// GetInfo retrieves tool info from the global registry
func GetInfo(name string) (*ToolInfo, bool) {
	return globalRegistry.GetInfo(name)
}

// List lists all registered tools from the global registry
func List() []Tool {
	return globalRegistry.List()
}

// ListInfo lists all tool info from the global registry
func ListInfo() []*ToolInfo {
	return globalRegistry.ListInfo()
}

// Execute executes a tool by name
func Execute(ctx context.Context, name string, params map[string]any) (any, error) {
	t, exists := globalRegistry.Get(name)
	if !exists {
		return nil, common.NewError(common.ErrCodeToolNotFound, fmt.Sprintf("tool %s not found", name), nil)
	}
	return t.Run(ctx, params)
}
