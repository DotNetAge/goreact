package reactor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
	"github.com/DotNetAge/goreact/pkg/core"
)

// ActorConfig represents actor configuration
type ActorConfig struct {
	MaxRetries           int
	Timeout              time.Duration
	EnableSkillCache     bool
	AllowedToolLevels    []common.SecurityLevel
	MaxConcurrentActions int
}

// DefaultActorConfig returns the default actor config
func DefaultActorConfig() *ActorConfig {
	return &ActorConfig{
		MaxRetries:           common.DefaultActorMaxRetries,
		Timeout:              common.DefaultActorTimeout,
		EnableSkillCache:     true,
		AllowedToolLevels:    []common.SecurityLevel{common.LevelSafe, common.LevelSensitive},
		MaxConcurrentActions: common.DefaultMaxConcurrentActions,
	}
}

// BaseActor provides base actor functionality
type BaseActor struct {
	toolExecutor   any // Would be tool.Executor
	memory         any // Would be Memory
	config         *ActorConfig
	skillCache     map[string]*core.SkillExecutionPlan
	skillCacheMu   sync.RWMutex
}

// NewBaseActor creates a new BaseActor
func NewBaseActor(config *ActorConfig) *BaseActor {
	if config == nil {
		config = DefaultActorConfig()
	}
	return &BaseActor{
		config:     config,
		skillCache: make(map[string]*core.SkillExecutionPlan),
	}
}

// WithToolExecutor sets the tool executor
func (a *BaseActor) WithToolExecutor(executor any) *BaseActor {
	a.toolExecutor = executor
	return a
}

// WithMemory sets the memory
func (a *BaseActor) WithMemory(memory any) *BaseActor {
	a.memory = memory
	return a
}

// Act executes an action
func (a *BaseActor) Act(ctx context.Context, action *core.Action, state *core.State) (*core.ActionResult, error) {
	startTime := time.Now()
	
	// Validate action
	if err := a.Validate(action); err != nil {
		return nil, err
	}
	
	// Route based on action type
	switch action.Type {
	case common.ActionTypeToolCall:
		return a.executeTool(ctx, action, state)
	case common.ActionTypeSkillInvoke:
		return a.executeSkill(ctx, action, state)
	case common.ActionTypeSubAgentDelegate:
		return a.delegateToSubAgent(ctx, action, state)
	case common.ActionTypeNoAction:
		return &core.ActionResult{
			Success:  true,
			Result:   nil,
			Duration: time.Since(startTime),
		}, nil
	default:
		return nil, fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// Validate validates an action
func (a *BaseActor) Validate(action *core.Action) error {
	if action == nil {
		return fmt.Errorf("action is nil")
	}
	
	if action.Target == "" && action.Type != common.ActionTypeNoAction {
		return fmt.Errorf("action target is required")
	}
	
	return nil
}

// executeTool executes a tool
func (a *BaseActor) executeTool(ctx context.Context, action *core.Action, state *core.State) (*core.ActionResult, error) {
	startTime := time.Now()
	
	// Would use tool executor
	// result, err := a.toolExecutor.Execute(ctx, action.Target, action.Params)
	
	// Simplified implementation
	result := &core.ActionResult{
		Success:  true,
		Result:   map[string]any{"output": fmt.Sprintf("Tool '%s' executed successfully", action.Target)},
		Duration: time.Since(startTime),
	}
	result.WithTool(action.Target)
	
	return result, nil
}

// executeSkill executes a skill with caching
func (a *BaseActor) executeSkill(ctx context.Context, action *core.Action, state *core.State) (*core.ActionResult, error) {
	startTime := time.Now()
	
	// Check cache first
	if a.config.EnableSkillCache {
		plan := a.getSkillPlan(action.Target)
		if plan != nil {
			// Execute cached plan
			return a.executeSkillPlan(ctx, plan, action.Params, state)
		}
	}
	
	// Would compile skill from memory
	// skill := a.memory.GetNode(action.Target, NodeTypeSkill)
	// plan := a.compileSkill(skill)
	
	// Simplified implementation
	result := &core.ActionResult{
		Success:  true,
		Result:   map[string]any{"output": fmt.Sprintf("Skill '%s' executed successfully", action.Target)},
		Duration: time.Since(startTime),
	}
	result.WithSkill(action.Target)
	
	return result, nil
}

// executeSkillPlan executes a compiled skill plan
func (a *BaseActor) executeSkillPlan(ctx context.Context, plan *core.SkillExecutionPlan, params map[string]any, state *core.State) (*core.ActionResult, error) {
	startTime := time.Now()
	
	// Merge plan defaults with provided params
	mergedParams := a.mergeParams(plan, params)
	
	// Execute each step
	for _, step := range plan.Steps {
		// Check condition
		if step.Condition != "" {
			// Would evaluate condition
			// if !evaluateCondition(step.Condition, mergedParams, state) {
			//     continue
			// }
		}
		
		// Execute step
		action := core.NewAction(common.ActionTypeToolCall, step.ToolName, a.applyTemplate(step.ParamsTemplate, mergedParams))
		stepResult, err := a.executeTool(ctx, action, state)
		if err != nil {
			// Handle failure
			if step.OnFailure == "abort" {
				return nil, err
			}
			// Continue or retry
		}
		
		// Store intermediate result
		if stepResult != nil {
			mergedParams["_step_"+fmt.Sprintf("%d", step.Index)] = stepResult.Result
		}
	}
	
	// Record execution
	plan.RecordExecution(true)
	
	result := &core.ActionResult{
		Success:  true,
		Result:   mergedParams,
		Duration: time.Since(startTime),
	}
	result.WithSkill(plan.SkillName)
	
	return result, nil
}

// delegateToSubAgent delegates to a sub-agent
func (a *BaseActor) delegateToSubAgent(ctx context.Context, action *core.Action, state *core.State) (*core.ActionResult, error) {
	startTime := time.Now()
	
	// Would create sub-agent and execute
	// subAgent := a.createSubAgent(action.Target)
	// result, err := subAgent.Ask(ctx, action.Params["question"].(string))
	
	// Simplified implementation
	result := &core.ActionResult{
		Success:  true,
		Result:   map[string]any{"output": fmt.Sprintf("Delegated to sub-agent '%s' successfully", action.Target)},
		Duration: time.Since(startTime),
	}
	result.WithSubAgent(action.Target)
	
	return result, nil
}

// getSkillPlan retrieves a cached skill plan
func (a *BaseActor) getSkillPlan(skillName string) *core.SkillExecutionPlan {
	a.skillCacheMu.RLock()
	defer a.skillCacheMu.RUnlock()
	
	return a.skillCache[skillName]
}

// cacheSkillPlan caches a skill plan
func (a *BaseActor) cacheSkillPlan(skillName string, plan *core.SkillExecutionPlan) {
	a.skillCacheMu.Lock()
	defer a.skillCacheMu.Unlock()
	
	a.skillCache[skillName] = plan
}

// compileSkill compiles a skill into an execution plan
func (a *BaseActor) compileSkill(skill any) *core.SkillExecutionPlan {
	// Would parse skill definition and create execution plan
	// Simplified implementation
	plan := core.NewSkillExecutionPlan("unknown")
	return plan
}

// mergeParams merges plan defaults with provided parameters
func (a *BaseActor) mergeParams(plan *core.SkillExecutionPlan, params map[string]any) map[string]any {
	merged := make(map[string]any)
	
	// Add defaults
	for _, param := range plan.Parameters {
		if param.Default != nil {
			merged[param.Name] = param.Default
		}
	}
	
	// Override with provided
	for k, v := range params {
		merged[k] = v
	}
	
	return merged
}

// applyTemplate applies parameter templates
func (a *BaseActor) applyTemplate(template, params map[string]any) map[string]any {
	result := make(map[string]any)
	
	for k, v := range template {
		// Check if value is a template reference
		if str, ok := v.(string); ok {
			// Simple template substitution
			if val, exists := params[str]; exists {
				result[k] = val
			} else {
				result[k] = str
			}
		} else {
			result[k] = v
		}
	}
	
	return result
}

// ClearSkillCache clears the skill cache
func (a *BaseActor) ClearSkillCache() {
	a.skillCacheMu.Lock()
	defer a.skillCacheMu.Unlock()
	
	a.skillCache = make(map[string]*core.SkillExecutionPlan)
}

// GetCachedSkills returns the list of cached skill names
func (a *BaseActor) GetCachedSkills() []string {
	a.skillCacheMu.RLock()
	defer a.skillCacheMu.RUnlock()
	
	names := make([]string, 0, len(a.skillCache))
	for name := range a.skillCache {
		names = append(names, name)
	}
	return names
}
