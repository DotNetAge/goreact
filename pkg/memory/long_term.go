package memory

import (
	"context"

	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// LongTermAccessor manages long-term memory through GraphRAG's document indexing
type LongTermAccessor struct {
	BaseAccessor
}

// NewLongTermAccessor creates a new LongTermAccessor
func NewLongTermAccessor(graphRAG pattern.GraphRAGPattern) *LongTermAccessor {
	return &LongTermAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: "Document",
		},
	}
}

// Search searches for relevant documents using semantic search
func (a *LongTermAccessor) Search(ctx context.Context, content string, topK int) ([]goreactcore.Node, error) {
	results, err := a.graphRAG.Retrieve(ctx, []string{content}, topK)
	if err != nil {
		return nil, err
	}

	nodes := make([]goreactcore.Node, 0, len(results))
	for _, result := range results {
		// Get content from Answer or first Chunk
		desc := result.Answer
		if desc == "" && len(result.Chunks) > 0 {
			desc = result.Chunks[0].Content
		}
		
		node := &goreactcore.BaseNode{
			Name:        result.ID,
			NodeType:    "Document",
			Description: desc,
			Metadata:    result.Metadata,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// SearchByType searches for documents by type
func (a *LongTermAccessor) SearchByType(ctx context.Context, content string, nodeType string, topK int) ([]goreactcore.Node, error) {
	results, err := a.graphRAG.Retrieve(ctx, []string{content}, topK)
	if err != nil {
		return nil, err
	}

	nodes := make([]goreactcore.Node, 0, len(results))
	for _, result := range results {
		if typ, ok := result.Metadata["node_type"].(string); ok && typ == nodeType {
			// Get content from Answer or first Chunk
			desc := result.Answer
			if desc == "" && len(result.Chunks) > 0 {
				desc = result.Chunks[0].Content
			}
			
			node := &goreactcore.BaseNode{
				Name:        result.ID,
				NodeType:    nodeType,
				Description: desc,
				Metadata:    result.Metadata,
			}
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// IndexDirectory indexes a directory of documents through GraphRAG
func (a *LongTermAccessor) IndexDirectory(ctx context.Context, path string, recursive bool) error {
	return a.graphRAG.IndexDirectory(ctx, path, recursive)
}

// IndexFile indexes a single file through GraphRAG
func (a *LongTermAccessor) IndexFile(ctx context.Context, filePath string) error {
	return a.graphRAG.IndexFile(ctx, filePath)
}
