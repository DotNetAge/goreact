// Package memory provides memory management for the goreact framework.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
	"github.com/DotNetAge/goreact/pkg/resource"
)

// Memory is a facade that holds a GraphRAG instance and all accessors.
// It does not expose operation methods directly - operations are done through accessors.
type Memory struct {
	mu             sync.RWMutex
	graphRAG       pattern.GraphRAGPattern

	// Accessors
	sessions       *SessionAccessor
	shortTerms     *ShortTermAccessor
	longTerms      *LongTermAccessor
	agents         *AgentAccessor
	skills         *SkillAccessor
	tools          *ToolAccessor
	reflections    *ReflectionAccessor
	plans          *PlanAccessor
	trajectories   *TrajectoryAccessor
	frozenSessions *FrozenSessionAccessor
}

// NewMemory creates a new Memory instance with the given GraphRAG.
// All accessors share the same GraphRAG instance.
func NewMemory(graphRAG pattern.GraphRAGPattern) *Memory {
	m := &Memory{
		graphRAG: graphRAG,
	}

	// Initialize all accessors with the same GraphRAG instance
	m.sessions = NewSessionAccessor(graphRAG)
	m.shortTerms = NewShortTermAccessor(graphRAG)
	m.longTerms = NewLongTermAccessor(graphRAG)
	m.agents = NewAgentAccessor(graphRAG)
	m.skills = NewSkillAccessor(graphRAG)
	m.tools = NewToolAccessor(graphRAG)
	m.reflections = NewReflectionAccessor(graphRAG)
	m.plans = NewPlanAccessor(graphRAG)
	m.trajectories = NewTrajectoryAccessor(graphRAG)
	m.frozenSessions = NewFrozenSessionAccessor(graphRAG)

	return m
}

// Load loads resources from the resource manager into GraphRAG.
// It indexes all agents, skills, tools, and models as nodes in the graph.
func (m *Memory) Load(ctx context.Context, rm *resource.ResourceManager) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.graphRAG == nil {
		return fmt.Errorf("graphRAG is not initialized")
	}

	// Index all Agents
	for name, agentAny := range rm.GetAgents() {
		node, err := anyToAgentNode(name, agentAny)
		if err != nil {
			return fmt.Errorf("failed to convert agent %s: %w", name, err)
		}
		if err := m.graphRAG.AddNode(ctx, node); err != nil {
			return fmt.Errorf("failed to index agent %s: %w", name, err)
		}

		// Add edges for agent's skills (if we can access them)
		if skills := getAgentSkills(agentAny); len(skills) > 0 {
			for _, skillName := range skills {
				edge := &core.Edge{
					ID:     fmt.Sprintf("agent-%s-skill-%s", name, skillName),
					Type:   "HAS_SKILL",
					Source: name,
					Target: skillName,
					Properties: map[string]any{
						"created_at": time.Now().Format(time.RFC3339),
					},
				}
				if err := m.graphRAG.AddEdge(ctx, edge); err != nil {
					return fmt.Errorf("failed to add skill edge for agent %s: %w", name, err)
				}
			}
		}
	}

	// Index all Skills
	for name, skillAny := range rm.GetSkills() {
		node, err := anyToSkillNode(name, skillAny)
		if err != nil {
			return fmt.Errorf("failed to convert skill %s: %w", name, err)
		}
		if err := m.graphRAG.AddNode(ctx, node); err != nil {
			return fmt.Errorf("failed to index skill %s: %w", name, err)
		}
	}

	// Index all Tools
	for name, toolAny := range rm.GetTools() {
		node, err := anyToToolNode(name, toolAny)
		if err != nil {
			return fmt.Errorf("failed to convert tool %s: %w", name, err)
		}
		if err := m.graphRAG.AddNode(ctx, node); err != nil {
			return fmt.Errorf("failed to index tool %s: %w", name, err)
		}
	}

	// Index all Models
	for name, modelAny := range rm.GetModels() {
		node, err := anyToModelNode(name, modelAny)
		if err != nil {
			return fmt.Errorf("failed to convert model %s: %w", name, err)
		}
		if err := m.graphRAG.AddNode(ctx, node); err != nil {
			return fmt.Errorf("failed to index model %s: %w", name, err)
		}
	}

	return nil
}

// Sessions returns the session accessor
func (m *Memory) Sessions() *SessionAccessor {
	return m.sessions
}

// ShortTerms returns the short-term memory accessor
func (m *Memory) ShortTerms() *ShortTermAccessor {
	return m.shortTerms
}

// LongTerms returns the long-term memory accessor
func (m *Memory) LongTerms() *LongTermAccessor {
	return m.longTerms
}

// Agents returns the agent accessor
func (m *Memory) Agents() *AgentAccessor {
	return m.agents
}

// Skills returns the skill accessor
func (m *Memory) Skills() *SkillAccessor {
	return m.skills
}

// Tools returns the tool accessor
func (m *Memory) Tools() *ToolAccessor {
	return m.tools
}

// Reflections returns the reflection accessor
func (m *Memory) Reflections() *ReflectionAccessor {
	return m.reflections
}

// Plans returns the plan accessor
func (m *Memory) Plans() *PlanAccessor {
	return m.plans
}

// Trajectories returns the trajectory accessor
func (m *Memory) Trajectories() *TrajectoryAccessor {
	return m.trajectories
}

// FrozenSessions returns the frozen session accessor
func (m *Memory) FrozenSessions() *FrozenSessionAccessor {
	return m.frozenSessions
}

// GetGraphRAG returns the GraphRAG instance
func (m *Memory) GetGraphRAG() pattern.GraphRAGPattern {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.graphRAG
}
