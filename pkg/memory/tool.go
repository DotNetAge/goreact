package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreacttool "github.com/DotNetAge/goreact/pkg/tool"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// ToolAccessor manages Tool and GeneratedTool nodes
type ToolAccessor struct {
	BaseAccessor
}

// NewToolAccessor creates a new ToolAccessor
func NewToolAccessor(graphRAG pattern.GraphRAGPattern) *ToolAccessor {
	return &ToolAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeTool,
		},
	}
}

// Get retrieves a tool by name
func (a *ToolAccessor) Get(ctx context.Context, toolName string) (any, error) {
	node, err := a.BaseAccessor.Get(ctx, toolName)
	if err != nil {
		return nil, err
	}
	return nodeToTool(node), nil
}

// List lists all tools
func (a *ToolAccessor) List(ctx context.Context) ([]any, error) {
	nodes, err := a.BaseAccessor.List(ctx)
	if err != nil {
		return nil, err
	}

	tools := make([]any, 0, len(nodes))
	for _, node := range nodes {
		tools = append(tools, nodeToTool(node))
	}

	return tools, nil
}

// Add adds a tool
func (a *ToolAccessor) Add(ctx context.Context, tool any) error {
	t, ok := tool.(*goreacttool.ToolNode)
	if !ok {
		return fmt.Errorf("invalid tool type")
	}

	node := &core.Node{
		ID:   t.Name,
		Type: goreactcommon.NodeTypeTool,
		Properties: map[string]any{
			"name":            t.Name,
			"node_type":       goreactcommon.NodeTypeTool,
			"description":     t.Description,
			"type":            string(t.Type),
			"schema":          t.Schema,
			"endpoint":        t.Endpoint,
			"security_level":  t.SecurityLevel.String(),
			"is_idempotent":   t.IsIdempotent,
			"execution_count": t.ExecutionCount,
			"success_rate":    t.SuccessRate,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// Delete deletes a tool
func (a *ToolAccessor) Delete(ctx context.Context, toolName string) error {
	return a.BaseAccessor.Delete(ctx, toolName)
}

// GetGenerated retrieves a generated tool
func (a *ToolAccessor) GetGenerated(ctx context.Context, toolName string) (any, error) {
	node, err := a.graphRAG.GetNode(ctx, "generated-"+toolName)
	if err != nil {
		return nil, err
	}
	return nodeToGeneratedTool(node), nil
}

// ListGenerated lists all generated tools
func (a *ToolAccessor) ListGenerated(ctx context.Context) ([]any, error) {
	query := fmt.Sprintf("MATCH (n:%s) RETURN n", "GeneratedTool")
	results, err := a.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	tools := make([]any, 0, len(results))
	for _, result := range results {
		if nData, ok := result["n"].(map[string]any); ok {
			tools = append(tools, mapToGeneratedTool(nData))
		}
	}

	return tools, nil
}

// ApproveGenerated approves a generated tool
func (a *ToolAccessor) ApproveGenerated(ctx context.Context, toolName string) error {
	node, err := a.graphRAG.GetNode(ctx, "generated-"+toolName)
	if err != nil {
		return err
	}

	node.Properties["status"] = string(goreactcommon.GeneratedStatusApproved)
	return a.graphRAG.AddNode(ctx, node)
}
