package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// PlanAccessor manages Plan and PlanStep nodes
type PlanAccessor struct {
	BaseAccessor
}

// NewPlanAccessor creates a new PlanAccessor
func NewPlanAccessor(graphRAG pattern.GraphRAGPattern) *PlanAccessor {
	return &PlanAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypePlan,
		},
	}
}

// Get retrieves a plan by name
func (a *PlanAccessor) Get(ctx context.Context, planName string) (*goreactcore.PlanNode, error) {
	node, err := a.BaseAccessor.Get(ctx, planName)
	if err != nil {
		return nil, err
	}
	return nodeToPlanNode(node), nil
}

// List lists plans
func (a *PlanAccessor) List(ctx context.Context, opts ...ListOption) ([]*goreactcore.PlanNode, error) {
	nodes, err := a.BaseAccessor.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	plans := make([]*goreactcore.PlanNode, 0, len(nodes))
	for _, node := range nodes {
		plans = append(plans, nodeToPlanNode(node))
	}

	return plans, nil
}

// Add adds a plan
func (a *PlanAccessor) Add(ctx context.Context, plan *goreactcore.PlanNode) error {
	node := &core.Node{
		ID:   plan.Name,
		Type: goreactcommon.NodeTypePlan,
		Properties: map[string]any{
			"name":         plan.Name,
			"node_type":    goreactcommon.NodeTypePlan,
			"session_name": plan.SessionName,
			"goal":         plan.Goal,
			"steps":        plan.Steps,
			"status":       string(plan.Status),
			"success":      plan.Success,
			"task_type":    plan.TaskType,
			"created_at":   plan.CreatedAt.Format(time.RFC3339),
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// Update updates a plan
func (a *PlanAccessor) Update(ctx context.Context, plan *goreactcore.PlanNode) error {
	return a.Add(ctx, plan)
}

// Delete deletes a plan
func (a *PlanAccessor) Delete(ctx context.Context, planName string) error {
	return a.BaseAccessor.Delete(ctx, planName)
}

// FindSimilar finds similar plans based on goal
func (a *PlanAccessor) FindSimilar(ctx context.Context, goal string, threshold float64) ([]*goreactcore.PlanNode, error) {
	// Use semantic search to find similar plans
	results, err := a.graphRAG.Retrieve(ctx, []string{goal}, 10)
	if err != nil {
		return nil, err
	}

	plans := make([]*goreactcore.PlanNode, 0, len(results))
	for _, result := range results {
		if nodeType, ok := result.Metadata["node_type"].(string); ok && nodeType == goreactcommon.NodeTypePlan {
			// Get score from the first Scores element if available
			score := float64(0)
			if len(result.Scores) > 0 {
				score = float64(result.Scores[0])
			}
			
			// Get content from Answer or first Chunk
			content := result.Answer
			if content == "" && len(result.Chunks) > 0 {
				content = result.Chunks[0].Content
			}
			
			if score >= threshold {
				plan := &goreactcore.PlanNode{
					BaseNode: goreactcore.BaseNode{
						Name:        result.ID,
						NodeType:    goreactcommon.NodeTypePlan,
						Description: content,
						Metadata:    result.Metadata,
					},
					SessionName: getString(result.Metadata["session_name"]),
					Goal:        content,
					Status:      goreactcommon.PlanStatus(getString(result.Metadata["status"])),
					Success:     getBool(result.Metadata["success"]),
					TaskType:    getString(result.Metadata["task_type"]),
				}
				plans = append(plans, plan)
			}
		}
	}

	return plans, nil
}

// UpdateStep updates a plan step
func (a *PlanAccessor) UpdateStep(ctx context.Context, planName string, stepIndex int, status string, outcome string) error {
	plan, err := a.Get(ctx, planName)
	if err != nil {
		return err
	}

	if stepIndex >= 0 && stepIndex < len(plan.Steps) {
		plan.Steps[stepIndex].Status = goreactcommon.StepStatus(status)
		plan.Steps[stepIndex].Outcome = outcome
		return a.Update(ctx, plan)
	}

	return fmt.Errorf("invalid step index: %d", stepIndex)
}
