package agent

import (
	"fmt"
	"sync"
)

// Registry manages agent registration
type Registry struct {
	mu     sync.RWMutex
	agents map[string]Agent
}

// NewRegistry creates a new Registry
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

// Register registers an agent
func (r *Registry) Register(agent Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := agent.Name()
	if _, exists := r.agents[name]; exists {
		return fmt.Errorf("agent %s already registered", name)
	}
	
	r.agents[name] = agent
	return nil
}

// Unregister unregisters an agent
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

// Get retrieves an agent by name
func (r *Registry) Get(name string) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, exists := r.agents[name]
	return agent, exists
}

// List lists all registered agents
func (r *Registry) List() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	agents := make([]Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

// ListByDomain lists agents by domain
func (r *Registry) ListByDomain(domain string) []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	agents := []Agent{}
	for _, agent := range r.agents {
		if agent.Domain() == domain {
			agents = append(agents, agent)
		}
	}
	return agents
}

// Clear clears all registered agents
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents = make(map[string]Agent)
}

// Note: Global registry and package-level functions have been removed.
// Agents should be accessed through Memory's AgentAccessor.
// See memory/agent.go for the Memory-based approach.
