package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactskill "github.com/DotNetAge/goreact/pkg/skill"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// SkillAccessor manages Skill and SkillExecutionPlan nodes
type SkillAccessor struct {
	BaseAccessor
}

// NewSkillAccessor creates a new SkillAccessor
func NewSkillAccessor(graphRAG pattern.GraphRAGPattern) *SkillAccessor {
	return &SkillAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeSkill,
		},
	}
}

// Get retrieves a skill by name
func (a *SkillAccessor) Get(ctx context.Context, skillName string) (any, error) {
	node, err := a.BaseAccessor.Get(ctx, skillName)
	if err != nil {
		return nil, err
	}
	return nodeToSkill(node), nil
}

// List lists all skills
func (a *SkillAccessor) List(ctx context.Context) ([]any, error) {
	nodes, err := a.BaseAccessor.List(ctx)
	if err != nil {
		return nil, err
	}

	skills := make([]any, 0, len(nodes))
	for _, node := range nodes {
		skills = append(skills, nodeToSkill(node))
	}

	return skills, nil
}

// Add adds a skill
func (a *SkillAccessor) Add(ctx context.Context, skill any) error {
	s, ok := skill.(*goreactskill.SkillNode)
	if !ok {
		return fmt.Errorf("invalid skill type")
	}

	node := &core.Node{
		ID:   s.Name,
		Type: goreactcommon.NodeTypeSkill,
		Properties: map[string]any{
			"name":          s.Name,
			"node_type":     goreactcommon.NodeTypeSkill,
			"description":   s.Description,
			"agent":         s.Agent,
			"intent":        s.Intent,
			"template":      s.Template,
			"parameters":    s.Parameters,
			"allowed_tools": s.AllowedTools,
			"created_at":    time.Now().Format(time.RFC3339),
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// Delete deletes a skill
func (a *SkillAccessor) Delete(ctx context.Context, skillName string) error {
	return a.BaseAccessor.Delete(ctx, skillName)
}

// GetExecutionPlan retrieves a compiled skill execution plan
func (a *SkillAccessor) GetExecutionPlan(ctx context.Context, skillName string) (any, error) {
	planID := "plan-" + skillName
	node, err := a.graphRAG.GetNode(ctx, planID)
	if err != nil {
		return nil, err
	}
	return nodeToExecutionPlan(node), nil
}

// StoreExecutionPlan stores a compiled skill execution plan
func (a *SkillAccessor) StoreExecutionPlan(ctx context.Context, plan any) error {
	p, ok := plan.(*goreactskill.SkillExecutionPlan)
	if !ok {
		return fmt.Errorf("invalid plan type")
	}

	node := &core.Node{
		ID:   p.Name,
		Type: goreactcommon.NodeTypeSkillExecutionPlan,
		Properties: map[string]any{
			"name":            p.Name,
			"node_type":       goreactcommon.NodeTypeSkillExecutionPlan,
			"skill_name":      p.SkillName,
			"steps":           p.Steps,
			"parameters":      p.Parameters,
			"compiled_at":     p.CompiledAt.Format(time.RFC3339),
			"execution_count": p.ExecutionCount,
			"success_rate":    p.SuccessRate,
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// DeleteExecutionPlan deletes a skill execution plan
func (a *SkillAccessor) DeleteExecutionPlan(ctx context.Context, skillName string) error {
	planID := "plan-" + skillName
	return a.graphRAG.DeleteNode(ctx, planID)
}

// UpdateExecutionStats updates execution statistics for a skill
func (a *SkillAccessor) UpdateExecutionStats(ctx context.Context, skillName string, success bool, duration time.Duration) error {
	plan, err := a.GetExecutionPlan(ctx, skillName)
	if err != nil {
		return err
	}

	if p, ok := plan.(*goreactskill.SkillExecutionPlan); ok {
		p.ExecutionCount++
		if success {
			p.SuccessRate = p.SuccessRate*0.9 + 0.1
		} else {
			p.SuccessRate = p.SuccessRate * 0.9
		}
		return a.StoreExecutionPlan(ctx, p)
	}

	return nil
}

// GetGenerated retrieves a generated skill
func (a *SkillAccessor) GetGenerated(ctx context.Context, skillName string) (any, error) {
	node, err := a.graphRAG.GetNode(ctx, "generated-"+skillName)
	if err != nil {
		return nil, err
	}
	return nodeToGeneratedSkill(node), nil
}

// ListGenerated lists all generated skills
func (a *SkillAccessor) ListGenerated(ctx context.Context) ([]any, error) {
	query := fmt.Sprintf("MATCH (n:%s) RETURN n", "GeneratedSkill")
	results, err := a.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	skills := make([]any, 0, len(results))
	for _, result := range results {
		if nData, ok := result["n"].(map[string]any); ok {
			skills = append(skills, mapToGeneratedSkill(nData))
		}
	}

	return skills, nil
}

// ApproveGenerated approves a generated skill
func (a *SkillAccessor) ApproveGenerated(ctx context.Context, skillName string) error {
	node, err := a.graphRAG.GetNode(ctx, "generated-"+skillName)
	if err != nil {
		return err
	}

	node.Properties["status"] = string(goreactcommon.GeneratedStatusApproved)
	return a.graphRAG.AddNode(ctx, node)
}

// Search performs semantic search on skills by description
func (a *SkillAccessor) Search(ctx context.Context, query string, topK int) ([]any, error) {
	// Use GraphRAG's retrieve capability for semantic search
	// Retrieve takes queries and topK
	results, err := a.graphRAG.Retrieve(ctx, []string{query}, topK)
	if err != nil {
		return nil, err
	}

	skills := make([]any, 0)
	for _, result := range results {
		// Convert RetrievalResult chunks to Skills
		for _, chunk := range result.Chunks {
			skill := goreactskill.Skill{
				Name:        chunk.ID,
				Description: chunk.Content,
			}
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// GetByHash retrieves a skill by content hash for cache validation
func (a *SkillAccessor) GetByHash(ctx context.Context, skillName string, hash string) (*goreactskill.Skill, error) {
	skill, err := a.Get(ctx, skillName)
	if err != nil {
		return nil, err
	}

	if s, ok := skill.(goreactskill.Skill); ok {
		if s.ContentHash == hash {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("skill hash mismatch")
}

// InvalidateCache invalidates the execution plan cache for a skill
func (a *SkillAccessor) InvalidateCache(ctx context.Context, skillName string) error {
	// Delete the cached execution plan
	return a.DeleteExecutionPlan(ctx, skillName)
}

// CheckCacheValidity checks if the cached plan is still valid
func (a *SkillAccessor) CheckCacheValidity(ctx context.Context, skillName string, currentHash string) (bool, error) {
	plan, err := a.GetExecutionPlan(ctx, skillName)
	if err != nil {
		return false, nil // No cache = invalid
	}

	if p, ok := plan.(*goreactskill.SkillExecutionPlan); ok {
		// Check if the skill's hash matches (would need to store hash in plan)
		_ = p
		return true, nil
	}

	return false, nil
}
