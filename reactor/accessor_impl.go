package reactor

import (
	"context"

	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/tools"
)

// Ensure Reactor implements tools.OrchestrationAccessor (v2).
var _ tools.OrchestrationAccessor = (*Reactor)(nil)

// Orchestrator returns the reactor's orchestrator for tool-layer orchestration access.
// Implements tools.OrchestrationAccessor v2 interface.
func (r *Reactor) Orchestrator() core.AgentOrchestrator {
	return r.orchestrator
}

// EventEmitter returns a function to emit ReactEvents via the event bus.
func (r *Reactor) EventEmitter() func(core.ReactEvent) {
	if r.eventBus == nil {
		return nil
	}
	return r.eventBus.Emit
}

// RunInline executes a synchronous inline task using the same reactor context.
// Used by TaskCreateTool for plan→execute sequential workflow.
// This is the ReactorAccessor.Run(ctx, prompt) implementation.
func (r *Reactor) RunInline(ctx context.Context, prompt string) (answer string, err error) {
	result, runErr := r.Run(ctx, prompt, nil)
	if runErr != nil {
		return "", runErr
	}
	return result.Answer, nil
}

// Config returns the reactor's configuration.
func (r *Reactor) Config() tools.ReactorConfig {
	return tools.ReactorConfig{
		APIKey:        r.config.APIKey,
		BaseURL:       r.config.BaseURL,
		Model:         r.config.Model,
		SystemPrompt:  r.config.SystemPrompt,
		Temperature:   r.config.Temperature,
		MaxTokens:     r.config.MaxTokens,
		MaxIterations: r.config.MaxIterations,
		ClientType:    r.config.ClientType,
		IsLocal:       r.config.IsLocal,
	}
}

// registerOrchestrationTools creates and registers all orchestration tools
// (task, subagent, team) with the reactor as their accessor.
func (r *Reactor) registerOrchestrationTools() {
	// Task tools
	taskCreate := tools.NewTaskCreateTool()
	taskCreate.SetAccessor(r)
	_ = r.RegisterTool(taskCreate)

	taskResult := tools.NewTaskResultTool()
	taskResult.SetAccessor(r)
	_ = r.RegisterTool(taskResult)

	taskList := tools.NewTaskListTool()
	taskList.SetAccessor(r)
	_ = r.RegisterTool(taskList)

	// Skill tools
	skillCreate := tools.NewSkillCreateTool()
	_ = r.RegisterTool(skillCreate)

	skillList := tools.NewSkillListTool()
	_ = r.RegisterTool(skillList)
}

