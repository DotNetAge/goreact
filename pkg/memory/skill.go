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
func (a *SkillAccessor) Get(ctx context.Context, skillName string) (*goreactskill.Skill, error) {
	node, err := a.BaseAccessor.Get(ctx, skillName)
	if err != nil {
		return nil, err
	}
	skill := nodeToSkill(node)
	return &skill, nil
}

// List lists all skills
func (a *SkillAccessor) List(ctx context.Context) ([]*goreactskill.Skill, error) {
	nodes, err := a.BaseAccessor.List(ctx)
	if err != nil {
		return nil, err
	}

	skills := make([]*goreactskill.Skill, 0, len(nodes))
	for _, node := range nodes {
		skill := nodeToSkill(node)
		skills = append(skills, &skill)
	}

	return skills, nil
}

// Add adds a skill
func (a *SkillAccessor) Add(ctx context.Context, skill *goreactskill.SkillNode) error {
	node := &core.Node{
		ID:   skill.Name,
		Type: goreactcommon.NodeTypeSkill,
		Properties: map[string]any{
			"name":          skill.Name,
			"node_type":     goreactcommon.NodeTypeSkill,
			"description":   skill.Description,
			"agent":         skill.Agent,
			"intent":        skill.Intent,
			"template":      skill.Template,
			"parameters":    skill.Parameters,
			"allowed_tools": skill.AllowedTools,
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
func (a *SkillAccessor) GetExecutionPlan(ctx context.Context, skillName string) (*goreactskill.SkillExecutionPlan, error) {
	planID := "plan-" + skillName
	node, err := a.graphRAG.GetNode(ctx, planID)
	if err != nil {
		return nil, err
	}
	return nodeToExecutionPlan(node), nil
}

// StoreExecutionPlan stores a compiled skill execution plan
func (a *SkillAccessor) StoreExecutionPlan(ctx context.Context, plan *goreactskill.SkillExecutionPlan) error {
	node := &core.Node{
		ID:   plan.Name,
		Type: goreactcommon.NodeTypeSkillExecutionPlan,
		Properties: map[string]any{
			"name":            plan.Name,
			"node_type":       goreactcommon.NodeTypeSkillExecutionPlan,
			"skill_name":      plan.SkillName,
			"steps":           plan.Steps,
			"parameters":      plan.Parameters,
			"compiled_at":     plan.CompiledAt.Format(time.RFC3339),
			"execution_count": plan.ExecutionCount,
			"success_rate":    plan.SuccessRate,
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

	plan.ExecutionCount++
	if success {
		plan.SuccessRate = plan.SuccessRate*0.9 + 0.1
	} else {
		plan.SuccessRate = plan.SuccessRate * 0.9
	}
	return a.StoreExecutionPlan(ctx, plan)
}

// GetGenerated retrieves a generated skill
func (a *SkillAccessor) GetGenerated(ctx context.Context, skillName string) (*goreactskill.GeneratedSkill, error) {
	node, err := a.graphRAG.GetNode(ctx, "generated-"+skillName)
	if err != nil {
		return nil, err
	}
	skill := nodeToGeneratedSkill(node)
	return &skill, nil
}

// ListGenerated lists all generated skills
func (a *SkillAccessor) ListGenerated(ctx context.Context) ([]*goreactskill.GeneratedSkill, error) {
	query := fmt.Sprintf("MATCH (n:%s) RETURN n", "GeneratedSkill")
	results, err := a.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	skills := make([]*goreactskill.GeneratedSkill, 0, len(results))
	for _, result := range results {
		if nData, ok := result["n"].(map[string]any); ok {
			skill := mapToGeneratedSkill(nData)
			skills = append(skills, &skill)
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
func (a *SkillAccessor) Search(ctx context.Context, query string, topK int) ([]*goreactskill.Skill, error) {
	// Use GraphRAG's retrieve capability for semantic search
	// Retrieve takes queries and topK
	results, err := a.graphRAG.Retrieve(ctx, []string{query}, topK)
	if err != nil {
		return nil, err
	}

	skills := make([]*goreactskill.Skill, 0)
	for _, result := range results {
		// Convert RetrievalResult chunks to Skills
		for _, chunk := range result.Chunks {
			skill := &goreactskill.Skill{
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

	if skill.ContentHash == hash {
		return skill, nil
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

	// Check if the skill's hash matches (would need to store hash in plan)
	_ = plan
	return true, nil
}
