package memory

import (
	"context"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// ReflectionAccessor manages Reflection nodes
type ReflectionAccessor struct {
	BaseAccessor
}

// NewReflectionAccessor creates a new ReflectionAccessor
func NewReflectionAccessor(graphRAG pattern.GraphRAGPattern) *ReflectionAccessor {
	return &ReflectionAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeReflection,
		},
	}
}

// Get retrieves a reflection by name
func (a *ReflectionAccessor) Get(ctx context.Context, reflectionName string) (*goreactcore.ReflectionNode, error) {
	node, err := a.BaseAccessor.Get(ctx, reflectionName)
	if err != nil {
		return nil, err
	}
	return nodeToReflectionNode(node), nil
}

// List lists reflections
func (a *ReflectionAccessor) List(ctx context.Context, opts ...ListOption) ([]*goreactcore.ReflectionNode, error) {
	nodes, err := a.BaseAccessor.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	reflections := make([]*goreactcore.ReflectionNode, 0, len(nodes))
	for _, node := range nodes {
		reflections = append(reflections, nodeToReflectionNode(node))
	}

	return reflections, nil
}

// Add adds a reflection
func (a *ReflectionAccessor) Add(ctx context.Context, reflection *goreactcore.ReflectionNode) error {
	node := &core.Node{
		ID:   reflection.Name,
		Type: goreactcommon.NodeTypeReflection,
		Properties: map[string]any{
			"name":            reflection.Name,
			"node_type":       goreactcommon.NodeTypeReflection,
			"session_name":    reflection.SessionName,
			"trajectory_name": reflection.TrajectoryName,
			"failure_reason":  reflection.FailureReason,
			"analysis":        reflection.Analysis,
			"heuristic":       reflection.Heuristic,
			"suggestions":     reflection.Suggestions,
			"score":           reflection.Score,
			"task_type":       reflection.TaskType,
			"created_at":      reflection.CreatedAt.Format(time.RFC3339),
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// Delete deletes a reflection
func (a *ReflectionAccessor) Delete(ctx context.Context, reflectionName string) error {
	return a.BaseAccessor.Delete(ctx, reflectionName)
}

// GetRelevant retrieves relevant reflections for a task
func (a *ReflectionAccessor) GetRelevant(ctx context.Context, taskType string, query string, topK int) ([]*goreactcore.ReflectionNode, error) {
	// Use semantic search to find relevant reflections
	results, err := a.graphRAG.Retrieve(ctx, []string{query}, topK)
	if err != nil {
		return nil, err
	}

	reflections := make([]*goreactcore.ReflectionNode, 0, len(results))
	for _, result := range results {
		if nodeType, ok := result.Metadata["node_type"].(string); ok && nodeType == goreactcommon.NodeTypeReflection {
			if tt, ok := result.Metadata["task_type"].(string); ok && (taskType == "" || tt == taskType) {
				// Get content from Answer or first Chunk
				content := result.Answer
				if content == "" && len(result.Chunks) > 0 {
					content = result.Chunks[0].Content
				}
				
				reflection := &goreactcore.ReflectionNode{
					BaseNode: goreactcore.BaseNode{
						Name:        result.ID,
						NodeType:    goreactcommon.NodeTypeReflection,
						Description: content,
						Metadata:    result.Metadata,
					},
					SessionName:    getString(result.Metadata["session_name"]),
					TrajectoryName: getString(result.Metadata["trajectory_name"]),
					FailureReason:  getString(result.Metadata["failure_reason"]),
					Analysis:       getString(result.Metadata["analysis"]),
					Heuristic:      getString(result.Metadata["heuristic"]),
					Score:          getFloat64(result.Metadata["score"]),
					TaskType:       tt,
				}
				if suggestions, ok := result.Metadata["suggestions"].([]string); ok {
					reflection.Suggestions = suggestions
				}
				reflections = append(reflections, reflection)
			}
		}
	}

	return reflections, nil
}
