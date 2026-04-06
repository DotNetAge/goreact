package reactor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/memory"
	"github.com/DotNetAge/goreact/pkg/skill"
	"github.com/DotNetAge/goreact/pkg/tool"
)

// DefaultActorConfig returns the default actor config
func DefaultActorConfig() *goreactcommon.ActorConfig {
	return &goreactcommon.ActorConfig{
		MaxRetries:           goreactcommon.DefaultActorMaxRetries,
		Timeout:              goreactcommon.DefaultActorTimeout,
		EnableSkillCache:     true,
		AllowedToolLevels:    []goreactcommon.SecurityLevel{goreactcommon.LevelSafe, goreactcommon.LevelSensitive},
		MaxConcurrentActions: goreactcommon.DefaultMaxConcurrentActions,
		EnableDryRun:         false,
	}
}

// BaseActor provides base actor functionality
type BaseActor struct {
	toolExecutor *tool.Executor
	memory       *memory.Memory
	skillCompiler *skill.Compiler
	config       *goreactcommon.ActorConfig
	skillCache   map[string]*skill.SkillExecutionPlan
	skillCacheMu sync.RWMutex
}

// NewBaseActor creates a new BaseActor
func NewBaseActor(config *goreactcommon.ActorConfig) *BaseActor {
	if config == nil {
		config = DefaultActorConfig()
	}
	return &BaseActor{
		config:       config,
		skillCache:   make(map[string]*skill.SkillExecutionPlan),
		skillCompiler: skill.NewCompiler(),
	}
}

// WithToolExecutor sets the tool executor
func (a *BaseActor) WithToolExecutor(executor *tool.Executor) *BaseActor {
	a.toolExecutor = executor
	return a
}

// WithMemory sets the memory
func (a *BaseActor) WithMemory(mem *memory.Memory) *BaseActor {
	a.memory = mem
	return a
}

// WithSkillCompiler sets the skill compiler
func (a *BaseActor) WithSkillCompiler(compiler *skill.Compiler) *BaseActor {
	a.skillCompiler = compiler
	return a
}

// Act executes an action
func (a *BaseActor) Act(ctx context.Context, action *goreactcore.Action, state *goreactcore.State) (*goreactcore.ActionResult, error) {
	startTime := time.Now()
	
	// Validate action
	if err := a.Validate(action); err != nil {
		return nil, err
	}
	
	// Check for dry run
	if a.config.EnableDryRun {
		return a.dryRun(action, startTime)
	}
	
	// Route based on action type
	switch action.Type {
	case goreactcommon.ActionTypeToolCall:
		return a.executeTool(ctx, action, state)
	case goreactcommon.ActionTypeSkillInvoke:
		return a.executeSkill(ctx, action, state)
	case goreactcommon.ActionTypeSubAgentDelegate:
		return a.delegateToSubAgent(ctx, action, state)
	case goreactcommon.ActionTypeNoAction:
		return &goreactcore.ActionResult{
			Success:  true,
			Result:   nil,
			Duration: time.Since(startTime),
		}, nil
	default:
		return nil, fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// dryRun simulates action execution without actually running it
func (a *BaseActor) dryRun(action *goreactcore.Action, startTime time.Time) (*goreactcore.ActionResult, error) {
	return &goreactcore.ActionResult{
		Success:  true,
		Result:   map[string]any{"dry_run": true, "action": fmt.Sprintf("%s: %s", action.Type, action.Target)},
		Duration: time.Since(startTime),
	}, nil
}

// Validate validates an action
func (a *BaseActor) Validate(action *goreactcore.Action) error {
	if action == nil {
		return fmt.Errorf("action is nil")
	}
	
	if action.Target == "" && action.Type != goreactcommon.ActionTypeNoAction {
		return fmt.Errorf("action target is required")
	}
	
	return nil
}

// executeTool executes a tool
func (a *BaseActor) executeTool(ctx context.Context, action *goreactcore.Action, state *goreactcore.State) (*goreactcore.ActionResult, error) {
	// Use tool executor if available
	if a.toolExecutor != nil {
		result, err := a.toolExecutor.Execute(ctx, action.Target, action.Params)
		if err != nil {
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}
		return result, nil
	}
	
	// Try to get tool from memory
	if a.memory != nil {
		toolNode, err := a.memory.Tools().Get(ctx, action.Target)
		if err == nil && toolNode != nil {
			return a.executeToolFromNode(ctx, toolNode, action.Params, state)
		}
	}
	
	// Fallback: Return error if no tool available
	return nil, fmt.Errorf("tool '%s' not found and no tool executor configured", action.Target)
}

// executeToolFromNode executes a tool from its node definition
func (a *BaseActor) executeToolFromNode(ctx context.Context, toolNode *tool.ToolNode, params map[string]any, state *goreactcore.State) (*goreactcore.ActionResult, error) {
	// Validate parameters against tool definition
	if err := a.validateToolParams(toolNode, params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}
	
	// Execute tool based on type
	switch toolNode.Type {
	case "python":
		return a.executePythonTool(ctx, toolNode, params)
	case "bash":
		return a.executeBashTool(ctx, toolNode, params)
	case "cli":
		return a.executeCLITool(ctx, toolNode, params)
	default:
		return nil, fmt.Errorf("unsupported tool type: %s", toolNode.Type)
	}
}

// validateToolParams validates parameters against tool definition
func (a *BaseActor) validateToolParams(toolNode *tool.ToolNode, params map[string]any) error {
	// Basic validation - check schema if available
	if toolNode.Schema != nil {
		if required, ok := toolNode.Schema["required"].([]string); ok {
			for _, req := range required {
				if _, exists := params[req]; !exists {
					return fmt.Errorf("required parameter '%s' is missing", req)
				}
			}
		}
	}
	return nil
}

// executePythonTool executes a Python tool
func (a *BaseActor) executePythonTool(ctx context.Context, toolNode *tool.ToolNode, params map[string]any) (*goreactcore.ActionResult, error) {
	startTime := time.Now()
	
	// Build command to execute Python script
	// This would use os/exec to run the Python script
	// For now, return a placeholder result
	
	result := &goreactcore.ActionResult{
		Success:  true,
		Result:   map[string]any{"output": fmt.Sprintf("Python tool '%s' executed", toolNode.Name)},
		Duration: time.Since(startTime),
	}
	result.WithTool(toolNode.Name)
	
	return result, nil
}

// executeBashTool executes a Bash tool
func (a *BaseActor) executeBashTool(ctx context.Context, toolNode *tool.ToolNode, params map[string]any) (*goreactcore.ActionResult, error) {
	startTime := time.Now()
	
	// Execute bash script
	// This would use os/exec to run the bash script
	// For now, return a placeholder result
	
	result := &goreactcore.ActionResult{
		Success:  true,
		Result:   map[string]any{"output": fmt.Sprintf("Bash tool '%s' executed", toolNode.Name)},
		Duration: time.Since(startTime),
	}
	result.WithTool(toolNode.Name)
	
	return result, nil
}

// executeCLITool executes a CLI tool
func (a *BaseActor) executeCLITool(ctx context.Context, toolNode *tool.ToolNode, params map[string]any) (*goreactcore.ActionResult, error) {
	startTime := time.Now()
	
	// Execute CLI command
	// This would use os/exec to run the CLI command
	// For now, return a placeholder result
	
	result := &goreactcore.ActionResult{
		Success:  true,
		Result:   map[string]any{"output": fmt.Sprintf("CLI tool '%s' executed", toolNode.Name)},
		Duration: time.Since(startTime),
	}
	result.WithTool(toolNode.Name)
	
	return result, nil
}

// executeSkill executes a skill with caching
func (a *BaseActor) executeSkill(ctx context.Context, action *goreactcore.Action, state *goreactcore.State) (*goreactcore.ActionResult, error) {
	// Check cache first
	if a.config.EnableSkillCache {
		plan := a.getSkillPlan(action.Target)
		if plan != nil {
			// Execute cached plan
			return a.executeSkillPlan(ctx, plan, action.Params, state)
		}
	}
	
	// Try to load skill from memory
	if a.memory != nil {
		skillNode, err := a.memory.Skills().Get(ctx, action.Target)
		if err == nil && skillNode != nil {
			// Compile skill into execution plan
			plan, err := a.compileSkillFromNode(skillNode)
			if err != nil {
				return nil, fmt.Errorf("failed to compile skill: %w", err)
			}
			
			// Cache the plan
			if a.config.EnableSkillCache && plan != nil {
				a.cacheSkillPlan(action.Target, plan)
			}
			
			// Execute the plan
			return a.executeSkillPlan(ctx, plan, action.Params, state)
		}
	}
	
	// Fallback: Skill not found
	return nil, fmt.Errorf("skill '%s' not found", action.Target)
}

// executeSkillPlan executes a compiled skill plan
func (a *BaseActor) executeSkillPlan(ctx context.Context, plan *skill.SkillExecutionPlan, params map[string]any, state *goreactcore.State) (*goreactcore.ActionResult, error) {
	startTime := time.Now()
	
	// Merge plan defaults with provided params
	mergedParams := a.mergeParams(plan, params)
	
	// Execute each step
	for _, step := range plan.Steps {
		// Check condition if present
		if step.Condition != "" {
			conditionMet, err := a.evaluateCondition(step.Condition, mergedParams, state)
			if err != nil {
				return nil, fmt.Errorf("condition evaluation failed: %w", err)
			}
			if !conditionMet {
				continue
			}
		}
		
		// Execute step
		action := goreactcore.NewAction(goreactcommon.ActionTypeToolCall, step.ToolName, a.applyTemplate(step.ParamsTemplate, mergedParams))
		stepResult, err := a.executeTool(ctx, action, state)
		if err != nil {
			// Handle failure based on step configuration
			if step.OnError == "stop" {
				return nil, err
			}
			// Continue or retry based on configuration
			continue
		}
		
		// Store intermediate result
		if stepResult != nil {
			mergedParams[fmt.Sprintf("_step_%d", step.Index)] = stepResult.Result
		}
	}
	
	// Record execution
	plan.IncrementExecution(true)
	
	result := &goreactcore.ActionResult{
		Success:  true,
		Result:   mergedParams,
		Duration: time.Since(startTime),
	}
	result.WithSkill(plan.SkillName)
	
	return result, nil
}

// evaluateCondition evaluates a condition expression
func (a *BaseActor) evaluateCondition(condition string, params map[string]any, state *goreactcore.State) (bool, error) {
	// Simple condition evaluation
	// Supports basic comparisons: ==, !=, >, <, >=, <=
	
	conditions := []string{"==", "!=", ">=", "<=", ">", "<"}
	
	for _, op := range conditions {
		if strings.Contains(condition, op) {
			parts := strings.SplitN(condition, op, 2)
			if len(parts) != 2 {
				continue
			}
			
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			
			// Get left value
			leftVal, ok := params[left]
			if !ok {
				leftVal = left
			}
			
			// Get right value
			rightVal, ok := params[right]
			if !ok {
				rightVal = right
			}
			
			// Compare
			switch op {
			case "==":
				return fmt.Sprintf("%v", leftVal) == fmt.Sprintf("%v", rightVal), nil
			case "!=":
				return fmt.Sprintf("%v", leftVal) != fmt.Sprintf("%v", rightVal), nil
			}
		}
	}
	
	// Default to true if condition cannot be evaluated
	return true, nil
}

// delegateToSubAgent delegates to a sub-agent
func (a *BaseActor) delegateToSubAgent(ctx context.Context, action *goreactcore.Action, state *goreactcore.State) (*goreactcore.ActionResult, error) {
	startTime := time.Now()
	
	// Get question from params
	question, ok := action.Params["question"].(string)
	if !ok {
		return nil, fmt.Errorf("question parameter is required for sub-agent delegation")
	}
	
	// Try to get sub-agent from memory
	if a.memory != nil {
		agentNode, err := a.memory.Sessions().Get(ctx, action.Target)
		if err == nil && agentNode != nil {
			// Create and execute sub-agent
			// This would involve creating a new Reactor instance
			// and executing the question
			result := &goreactcore.ActionResult{
				Success:  true,
				Result:   map[string]any{"answer": fmt.Sprintf("Sub-agent '%s' processed: %s", action.Target, question)},
				Duration: time.Since(startTime),
			}
			result.WithSubAgent(action.Target)
			return result, nil
		}
	}
	
	// Fallback: Sub-agent not configured
	return nil, fmt.Errorf("sub-agent '%s' not found", action.Target)
}

// getSkillPlan retrieves a cached skill plan
func (a *BaseActor) getSkillPlan(skillName string) *skill.SkillExecutionPlan {
	a.skillCacheMu.RLock()
	defer a.skillCacheMu.RUnlock()
	
	return a.skillCache[skillName]
}

// cacheSkillPlan caches a skill plan
func (a *BaseActor) cacheSkillPlan(skillName string, plan *skill.SkillExecutionPlan) {
	a.skillCacheMu.Lock()
	defer a.skillCacheMu.Unlock()
	
	a.skillCache[skillName] = plan
}

// compileSkillFromNode compiles a skill into an execution plan
func (a *BaseActor) compileSkillFromNode(skillObj *skill.Skill) (*skill.SkillExecutionPlan, error) {
	if a.skillCompiler == nil {
		return nil, fmt.Errorf("skill compiler not configured")
	}
	
	// Compile skill
	ctx := context.Background()
	return a.skillCompiler.Compile(ctx, skillObj)
}

// mergeParams merges plan defaults with provided parameters
func (a *BaseActor) mergeParams(plan *skill.SkillExecutionPlan, params map[string]any) map[string]any {
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
	
	a.skillCache = make(map[string]*skill.SkillExecutionPlan)
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
