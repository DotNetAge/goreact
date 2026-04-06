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
func (a *ToolAccessor) Get(ctx context.Context, toolName string) (*goreacttool.ToolNode, error) {
	node, err := a.BaseAccessor.Get(ctx, toolName)
	if err != nil {
		return nil, err
	}
	tool := nodeToTool(node)
	return &tool, nil
}

// List lists all tools
func (a *ToolAccessor) List(ctx context.Context) ([]*goreacttool.ToolNode, error) {
	nodes, err := a.BaseAccessor.List(ctx)
	if err != nil {
		return nil, err
	}

	tools := make([]*goreacttool.ToolNode, 0, len(nodes))
	for _, node := range nodes {
		tool := nodeToTool(node)
		tools = append(tools, &tool)
	}

	return tools, nil
}

// Add adds a tool
func (a *ToolAccessor) Add(ctx context.Context, tool *goreacttool.ToolNode) error {
	node := &core.Node{
		ID:   tool.Name,
		Type: goreactcommon.NodeTypeTool,
		Properties: map[string]any{
			"name":            tool.Name,
			"node_type":       goreactcommon.NodeTypeTool,
			"description":     tool.Description,
			"type":            string(tool.Type),
			"schema":          tool.Schema,
			"endpoint":        tool.Endpoint,
			"security_level":  tool.SecurityLevel.String(),
			"is_idempotent":   tool.IsIdempotent,
			"execution_count": tool.ExecutionCount,
			"success_rate":    tool.SuccessRate,
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
func (a *ToolAccessor) GetGenerated(ctx context.Context, toolName string) (*goreacttool.GeneratedTool, error) {
	node, err := a.graphRAG.GetNode(ctx, "generated-"+toolName)
	if err != nil {
		return nil, err
	}
	tool := nodeToGeneratedTool(node)
	return &tool, nil
}

// ListGenerated lists all generated tools
func (a *ToolAccessor) ListGenerated(ctx context.Context) ([]*goreacttool.GeneratedTool, error) {
	query := fmt.Sprintf("MATCH (n:%s) RETURN n", "GeneratedTool")
	results, err := a.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	tools := make([]*goreacttool.GeneratedTool, 0, len(results))
	for _, result := range results {
		if nData, ok := result["n"].(map[string]any); ok {
			tool := mapToGeneratedTool(nData)
			tools = append(tools, &tool)
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
