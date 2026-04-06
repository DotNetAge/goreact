package tool

import (
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

// Note: Global registry and package-level functions have been removed.
// Use ToolFactory for on-demand tool instantiation instead.
// See factory.go for the new approach.
