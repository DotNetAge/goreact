package skill

import (
	"context"
	"fmt"
	"sync"

	"github.com/DotNetAge/gochat/pkg/core"
	"github.com/DotNetAge/goreact/pkg/common"
)

// Registry manages skill registration and retrieval
type Registry struct {
	mu     sync.RWMutex
	skills map[string]*Skill
	plans  map[string]*SkillExecutionPlan
}

// NewRegistry creates a new Registry
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*Skill),
		plans:  make(map[string]*SkillExecutionPlan),
	}
}

// Register registers a skill
func (r *Registry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.skills[skill.Name]; exists {
		return fmt.Errorf("skill %s already registered", skill.Name)
	}
	
	r.skills[skill.Name] = skill
	return nil
}

// Unregister unregisters a skill
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.skills, name)
	delete(r.plans, name)
}

// Get retrieves a skill by name
func (r *Registry) Get(name string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	skill, exists := r.skills[name]
	return skill, exists
}

// GetPlan retrieves a compiled execution plan
func (r *Registry) GetPlan(name string) (*SkillExecutionPlan, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plan, exists := r.plans[name]
	return plan, exists
}

// SetPlan sets a compiled execution plan
func (r *Registry) SetPlan(plan *SkillExecutionPlan) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.plans[plan.SkillName] = plan
}

// List lists all registered skills
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	return skills
}

// Search performs semantic search on skills (simplified implementation)
func (r *Registry) Search(query string, topK int) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Simplified implementation - in production would use vector similarity
	// For now, return skills matching by name or description
	skills := make([]*Skill, 0)
	for _, skill := range r.skills {
		// Simple keyword matching
		if containsIgnoreCase(skill.Name, query) || containsIgnoreCase(skill.Description, query) {
			skills = append(skills, skill)
			if len(skills) >= topK {
				break
			}
		}
	}
	return skills
}

// containsIgnoreCase checks if s contains substr (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	// Simple implementation
	return len(s) >= len(substr) && 
		(s == substr || len(substr) == 0 || 
		 (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring finds substr in s (case insensitive)
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			subc := substr[j]
			// Convert to lowercase for comparison
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// ListByAgent lists skills by agent
func (r *Registry) ListByAgent(agent string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	skills := []*Skill{}
	for _, skill := range r.skills {
		if skill.Agent == agent {
			skills = append(skills, skill)
		}
	}
	return skills
}

// Clear clears all registered skills
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.skills = make(map[string]*Skill)
	r.plans = make(map[string]*SkillExecutionPlan)
}

// Note: Global registry and package-level functions have been removed.
// Use SkillFactory for on-demand skill instantiation instead.
// See factory.go for the new approach.

// Executor executes skills.
// It uses SkillFactory and SkillInfoAccessor (from Memory) for on-demand skill access.
type Executor struct {
	factory      *SkillFactory
	infoAccessor SkillInfoAccessor
	compiler     *Compiler
	resolver     *RuntimeContextResolver
	toolExecutor any // Would be tool.Executor
	llmClient   core.Client
}

// ExecutorOption is a function that configures the Executor
type ExecutorOption func(*Executor)

// NewExecutor creates a new Executor
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		factory:  globalSkillFactory,
		compiler: NewCompiler(),
	}
	
	for _, opt := range opts {
		opt(e)
	}
	
	// Initialize resolver with LLM client
	e.resolver = NewRuntimeContextResolver(e.llmClient)
	
	return e
}

// Execute executes a skill by name
func (e *Executor) Execute(ctx context.Context, name string, params map[string]any, state map[string]any) (any, error) {
	// Check for cached plan from memory
	var plan *SkillExecutionPlan
	var err error
	
	if e.infoAccessor != nil {
		plan, err = e.infoAccessor.GetExecutionPlan(ctx, name)
		if err != nil {
			plan = nil
		}
	}
	
	if plan == nil {
		// Get skill from factory or memory
		skill, ok := e.factory.Create(name)
		if !ok && e.infoAccessor != nil {
			skill, err = e.infoAccessor.Get(ctx, name)
			if err != nil {
				return nil, common.NewError(common.ErrCodeSkillNotFound, fmt.Sprintf("skill %s not found", name), nil)
			}
			ok = true
		}
		
		if !ok {
			return nil, common.NewError(common.ErrCodeSkillNotFound, fmt.Sprintf("skill %s not found", name), nil)
		}
		
		// Compile skill
		plan, err = e.compiler.Compile(ctx, skill)
		if err != nil {
			return nil, common.NewError(common.ErrCodeSkillCompilation, fmt.Sprintf("failed to compile skill %s", name), err)
		}
		
		// Cache the plan in memory
		if e.infoAccessor != nil {
			_ = e.infoAccessor.StoreExecutionPlan(ctx, plan)
		}
	}
	
	// Resolve parameters
	resolvedParams, err := e.resolver.Resolve(ctx, plan, "", state)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve parameters: %w", err)
	}
	
	// Merge with provided params
	for k, v := range params {
		resolvedParams[k] = v
	}
	
	// Execute steps
	results := []any{}
	context := map[string]any{
		"params": resolvedParams,
		"steps":  results,
	}
	
	for _, step := range plan.Steps {
		// Render parameters
		renderedParams, err := e.compiler.RenderParams(step.ParamsTemplate, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render step params: %w", err)
		}
		
		// Check condition
		if step.Condition != "" {
			// Evaluate condition
			// Simplified - would use expression evaluator
		}
		
		// Execute tool
		// result, err := e.toolExecutor.Execute(ctx, step.ToolName, renderedParams)
		// Simplified - just collect the step
		results = append(results, map[string]any{
			"tool":   step.ToolName,
			"params": renderedParams,
		})
	}
	
	// Update execution stats
	plan.IncrementExecution(true)
	
	return results, nil
}

// WithFactory sets the skill factory
func WithFactory(factory *SkillFactory) ExecutorOption {
	return func(e *Executor) {
		e.factory = factory
	}
}

// WithSkillInfoAccessor sets the skill info accessor (from Memory)
func WithSkillInfoAccessor(accessor SkillInfoAccessor) ExecutorOption {
	return func(e *Executor) {
		e.infoAccessor = accessor
	}
}

// WithLLM sets the LLM client
func WithLLM(llm core.Client) ExecutorOption {
	return func(e *Executor) {
		e.llmClient = llm
	}
}
