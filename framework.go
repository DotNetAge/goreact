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

// Framework is the main framework facade.
// Memory is the single source of truth for all resources.
// Tools and agents are accessed through Memory's accessors.
type Framework struct {
	memory      *memory.Memory
	resources   *resource.ResourceManager
	orchestrator any
}

// New creates a new Framework instance
func New() *Framework {
	return &Framework{
		memory:    memory.NewMemory(memory.NewMockGraphRAG()),
		resources: resource.NewResourceManager(),
	}
}

// RegisterAgent registers an agent configuration to resources
func (f *Framework) RegisterAgent(name string, agentConfig any) error {
	return f.resources.RegisterAgent(name, agentConfig)
}

// GetAgent retrieves an agent node by name from memory
func (f *Framework) GetAgent(ctx context.Context, name string) (*memory.AgentAccessor, error) {
	return f.memory.Agents(), nil
}

// RegisterTool registers a tool to the tool factory
func (f *Framework) RegisterTool(name string, constructor tool.ToolConstructor) error {
	return tool.GetToolFactory().Register(name, constructor)
}

// GetToolFactory returns the global tool factory
func (f *Framework) GetToolFactory() *tool.ToolFactory {
	return tool.GetToolFactory()
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
	// Get agent from memory
	agentNode, err := f.memory.Agents().Get(ctx, "assistant")
	if err != nil {
		return nil, errors.New("agent not found")
	}
	_ = agentNode // Would create agent instance from node
	return nil, errors.New("not implemented: agent instantiation from memory")
}

// Execute executes a task using the reactor
func (f *Framework) Execute(ctx context.Context, input string, opts ...reactor.Option) (*reactor.Result, error) {
	r := reactor.NewReactor()
	return r.Execute(ctx, input, opts...)
}

// Default creates a framework with default configuration
func Default() *Framework {
	fw := New()
	
	// Initialize built-in tools
	tool.InitBuiltins()
	
	// Load resources
	fw.memory.Load(context.Background(), fw.resources)
	
	return fw
}
