package goreact

import (
	"context"
	"errors"

	"github.com/DotNetAge/goreact/pkg/agent"
	"github.com/DotNetAge/goreact/pkg/memory"
	"github.com/DotNetAge/goreact/pkg/reactor"
	"github.com/DotNetAge/goreact/pkg/resource"
	"github.com/DotNetAge/goreact/pkg/tool"
)

// Version is the framework version
const Version = "0.1.0"

// Framework is the main framework facade
type Framework struct {
	agents      *agent.Registry
	tools       *tool.Registry
	memory      *memory.Memory
	resources   *resource.ResourceManager
	orchestrator any
}

// New creates a new Framework instance
func New() *Framework {
	return &Framework{
		agents:    agent.NewRegistry(),
		tools:     tool.NewRegistry(),
		memory:    memory.NewMemory(),
		resources: resource.NewResourceManager(),
	}
}

// RegisterAgent registers an agent
func (f *Framework) RegisterAgent(a agent.Agent) error {
	return f.agents.Register(a)
}

// GetAgent retrieves an agent by name
func (f *Framework) GetAgent(name string) (agent.Agent, bool) {
	return f.agents.Get(name)
}

// RegisterTool registers a tool
func (f *Framework) RegisterTool(t tool.Tool) error {
	return f.tools.Register(t)
}

// GetTool retrieves a tool by name
func (f *Framework) GetTool(name string) (tool.Tool, bool) {
	return f.tools.Get(name)
}

// Memory returns the memory instance
func (f *Framework) Memory() *memory.Memory {
	return f.memory
}

// Resources returns the resource manager
func (f *Framework) Resources() *resource.ResourceManager {
	return f.resources
}

// Ask executes a question with the default agent
func (f *Framework) Ask(ctx context.Context, question string) (*agent.Result, error) {
	a, exists := f.agents.Get("assistant")
	if !exists {
		return nil, errors.New("agent not found")
	}
	return a.Ask(ctx, question)
}

// Execute executes a task using the reactor
func (f *Framework) Execute(ctx context.Context, input string, opts ...reactor.Option) (*reactor.Result, error) {
	r := reactor.NewReactor()
	return r.Execute(ctx, input, opts...)
}

// Default creates a framework with default configuration
func Default() *Framework {
	fw := New()
	
	// Register built-in tools
	tool.RegisterBuiltins()
	
	// Load resources
	fw.memory.Load(context.Background(), fw.resources)
	
	return fw
}
