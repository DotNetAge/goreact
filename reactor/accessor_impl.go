package reactor

import (
	"context"

	"github.com/DotNetAge/goreact/core"
	"github.com/DotNetAge/goreact/tools"
)

// Ensure Reactor implements tools.ReactorAccessor.
var _ tools.ReactorAccessor = (*Reactor)(nil)

// TaskManager returns the reactor's task manager.
// (Already defined in reactor.go — this file adds the accessor bridge methods only)
//
// Note: TaskManager() and MessageBus() are already defined in reactor.go
// with the correct signatures. They satisfy the ReactorAccessor interface.

// EventEmitter returns a function to emit ReactEvents via the event bus.
func (r *Reactor) EventEmitter() func(core.ReactEvent) {
	if r.eventBus == nil {
		return nil
	}
	return r.eventBus.Emit
}

// RegisterPendingTask stores a pending task result channel.
func (r *Reactor) RegisterPendingTask(taskID string, resultCh chan any) {
	r.pendingTasksMu.Lock()
	defer r.pendingTasksMu.Unlock()
	if r.pendingTasks == nil {
		r.pendingTasks = make(map[string]chan any)
	}
	r.pendingTasks[taskID] = resultCh
}

// GetPendingTask retrieves the channel for a pending subagent task.
func (r *Reactor) GetPendingTask(taskID string) (<-chan any, bool) {
	r.pendingTasksMu.RLock()
	defer r.pendingTasksMu.RUnlock()
	ch, ok := r.pendingTasks[taskID]
	if !ok {
		return nil, false
	}
	return ch, true
}

// RemovePendingTask removes a completed pending task.
func (r *Reactor) RemovePendingTask(taskID string) {
	r.pendingTasksMu.Lock()
	defer r.pendingTasksMu.Unlock()
	delete(r.pendingTasks, taskID)
}

// RunSubAgent spawns an independent agent asynchronously in a goroutine.
// It creates a sub-reactor with the specified system prompt and model,
// executes the task, and sends the result to resultCh.
//
// If the parent reactor has IsLocal=true (indicating a local model that cannot
// handle concurrent requests), the subagent runs synchronously instead of in a
// goroutine, unless the subagent specifies a different model.
func (r *Reactor) RunSubAgent(ctx context.Context, taskID string, systemPrompt, prompt string, model string, resultCh chan<- any) {
	// Determine if this subagent should run synchronously.
	// Synchronous when: parent IsLocal=true AND no model override (or same model).
	forceSync := r.config.IsLocal && (model == "" || model == r.config.Model)

	// Build sub-reactor config inheriting parent's settings with overrides
	subConfig := r.config
	if model != "" {
		subConfig.Model = model
		// If a different model is specified, the subagent may support concurrency
		if model != r.config.Model {
			forceSync = false
		}
	}
	if systemPrompt != "" {
		subConfig.SystemPrompt = systemPrompt
	}

	if forceSync {
		// Synchronous execution: run inline and send result directly
		r.runSubAgentSync(ctx, taskID, subConfig, prompt, resultCh)
	} else {
		// Asynchronous execution: run in a goroutine
		r.runSubAgentAsync(ctx, taskID, subConfig, prompt, resultCh)
	}
}

// runSubAgentSync runs a subagent synchronously (blocking the caller).
// Used when the parent reactor uses a local model (IsLocal=true).
func (r *Reactor) runSubAgentSync(ctx context.Context, taskID string, subConfig ReactorConfig, prompt string, resultCh chan<- any) {
	// Create a dedicated sub-reactor for this task
	subReactor := NewReactor(subConfig,
		WithMemory(r.memory),
		WithMessageBus(r.messageBus),
		WithEventBus(r.eventBus),
	)

	tm := r.taskManager
	result, runErr := subReactor.Run(ctx, prompt, nil)

	if runErr != nil {
		_ = tm.UpdateTaskStatus(taskID, core.TaskStatusFailed, "", runErr.Error())
		select {
		case resultCh <- runErr.Error():
		default:
		}
	} else {
		_ = tm.UpdateTaskStatus(taskID, core.TaskStatusCompleted, result.Answer, "")
		select {
		case resultCh <- result.Answer:
		default:
		}
	}

	r.RemovePendingTask(taskID)
}

// runSubAgentAsync runs a subagent asynchronously in a goroutine.
func (r *Reactor) runSubAgentAsync(ctx context.Context, taskID string, subConfig ReactorConfig, prompt string, resultCh chan<- any) {
	go func() {
		// Create a dedicated sub-reactor for this task
		subReactor := NewReactor(subConfig,
			WithMemory(r.memory),
			WithMessageBus(r.messageBus),
			WithEventBus(r.eventBus),
		)

		tm := r.taskManager
		result, runErr := subReactor.Run(ctx, prompt, nil)

		if runErr != nil {
			_ = tm.UpdateTaskStatus(taskID, core.TaskStatusFailed, "", runErr.Error())
			select {
			case resultCh <- runErr.Error():
			default:
			}
		} else {
			_ = tm.UpdateTaskStatus(taskID, core.TaskStatusCompleted, result.Answer, "")
			select {
			case resultCh <- result.Answer:
			default:
			}
		}

		r.RemovePendingTask(taskID)
	}()
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

	// SubAgent tools
	subagentTool := tools.NewSubAgentTool()
	subagentTool.SetAccessor(r)
	_ = r.RegisterTool(subagentTool)

	subagentResult := tools.NewSubAgentResultTool()
	subagentResult.SetAccessor(r)
	_ = r.RegisterTool(subagentResult)

	subagentList := tools.NewSubAgentListTool()
	subagentList.SetAccessor(r)
	_ = r.RegisterTool(subagentList)

	// Team tools
	teamCreate := tools.NewTeamCreateTool()
	teamCreate.SetAccessor(r)
	_ = r.RegisterTool(teamCreate)

	sendMsg := tools.NewSendMessageTool()
	sendMsg.SetAccessor(r)
	_ = r.RegisterTool(sendMsg)

	recvMsg := tools.NewReceiveMessagesTool()
	recvMsg.SetAccessor(r)
	_ = r.RegisterTool(recvMsg)

	teamStatus := tools.NewTeamStatusTool()
	teamStatus.SetAccessor(r)
	_ = r.RegisterTool(teamStatus)

	teamDelete := tools.NewTeamDeleteTool()
	teamDelete.SetAccessor(r)
	_ = r.RegisterTool(teamDelete)

	waitTeam := tools.NewWaitTeamTool()
	waitTeam.SetAccessor(r)
	_ = r.RegisterTool(waitTeam)

	// Skill tools
	skillCreate := tools.NewSkillCreateTool()
	_ = r.RegisterTool(skillCreate)

	skillList := tools.NewSkillListTool()
	_ = r.RegisterTool(skillList)
}

