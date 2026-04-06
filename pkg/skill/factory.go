package skill

import (
	"context"
	"fmt"
	"sync"
)

// SkillFactory creates Skill instances on demand.
// Skills are typically loaded from SKILL.md files and stored in Memory.
// This factory provides a way to register and retrieve skill constructors.
type SkillFactory struct {
	mu           sync.RWMutex
	constructors map[string]SkillConstructor
}

// SkillConstructor is a function that creates a Skill instance
type SkillConstructor func() *Skill

// NewSkillFactory creates a new SkillFactory
func NewSkillFactory() *SkillFactory {
	return &SkillFactory{
		constructors: make(map[string]SkillConstructor),
	}
}

// Register registers a skill constructor
func (f *SkillFactory) Register(name string, constructor SkillConstructor) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.constructors[name]; exists {
		return fmt.Errorf("skill %s already registered", name)
	}

	f.constructors[name] = constructor
	return nil
}

// MustRegister registers a skill constructor, panics on error
func (f *SkillFactory) MustRegister(name string, constructor SkillConstructor) {
	if err := f.Register(name, constructor); err != nil {
		panic(err)
	}
}

// Create creates a Skill instance by name
func (f *SkillFactory) Create(name string) (*Skill, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	constructor, exists := f.constructors[name]
	if !exists {
		return nil, false
	}

	return constructor(), true
}

// List returns all registered skill names
func (f *SkillFactory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.constructors))
	for name := range f.constructors {
		names = append(names, name)
	}
	return names
}

// SkillInfoAccessor provides access to skill metadata from Memory
type SkillInfoAccessor interface {
	Get(ctx context.Context, skillName string) (*Skill, error)
	List(ctx context.Context) ([]*Skill, error)
	GetExecutionPlan(ctx context.Context, skillName string) (*SkillExecutionPlan, error)
	StoreExecutionPlan(ctx context.Context, plan *SkillExecutionPlan) error
}

// HybridSkillFactory creates skills from both registered constructors and memory metadata
type HybridSkillFactory struct {
	factory      *SkillFactory
	infoAccessor SkillInfoAccessor
}

// NewHybridSkillFactory creates a new HybridSkillFactory
func NewHybridSkillFactory(factory *SkillFactory, accessor SkillInfoAccessor) *HybridSkillFactory {
	return &HybridSkillFactory{
		factory:      factory,
		infoAccessor: accessor,
	}
}

// GetOrCreate gets a skill from factory or retrieves from memory
func (h *HybridSkillFactory) GetOrCreate(ctx context.Context, name string) (*Skill, error) {
	// First, try to create from registered constructor
	if skill, ok := h.factory.Create(name); ok {
		return skill, nil
	}

	// If not found, try to get from memory metadata
	if h.infoAccessor != nil {
		skill, err := h.infoAccessor.Get(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("skill %s not found: %w", name, err)
		}
		return skill, nil
	}

	return nil, fmt.Errorf("skill %s not found", name)
}

// GetExecutionPlan retrieves or creates an execution plan for a skill
func (h *HybridSkillFactory) GetExecutionPlan(ctx context.Context, skillName string) (*SkillExecutionPlan, error) {
	if h.infoAccessor != nil {
		plan, err := h.infoAccessor.GetExecutionPlan(ctx, skillName)
		if err == nil {
			return plan, nil
		}
	}
	return nil, fmt.Errorf("execution plan for skill %s not found", skillName)
}

// StoreExecutionPlan stores an execution plan in memory
func (h *HybridSkillFactory) StoreExecutionPlan(ctx context.Context, plan *SkillExecutionPlan) error {
	if h.infoAccessor != nil {
		return h.infoAccessor.StoreExecutionPlan(ctx, plan)
	}
	return fmt.Errorf("no skill info accessor configured")
}

// Global skill factory instance
var globalSkillFactory = NewSkillFactory()

// GetSkillFactory returns the global skill factory
func GetSkillFactory() *SkillFactory {
	return globalSkillFactory
}
