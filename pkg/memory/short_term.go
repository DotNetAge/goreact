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

// ShortTermAccessor manages MemoryItem nodes
type ShortTermAccessor struct {
	BaseAccessor
}

// NewShortTermAccessor creates a new ShortTermAccessor
func NewShortTermAccessor(graphRAG pattern.GraphRAGPattern) *ShortTermAccessor {
	return &ShortTermAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeMemoryItem,
		},
	}
}

// Get retrieves a memory item by ID
func (a *ShortTermAccessor) Get(ctx context.Context, id string) (*goreactcore.MemoryItemNode, error) {
	node, err := a.BaseAccessor.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return nodeToMemoryItemNode(node), nil
}

// Add adds a memory item to a session
func (a *ShortTermAccessor) Add(ctx context.Context, sessionName string, item *goreactcore.MemoryItemNode) (*goreactcore.MemoryItemNode, error) {
	item.SessionName = sessionName

	node := &core.Node{
		ID:   item.Name,
		Type: goreactcommon.NodeTypeMemoryItem,
		Properties: map[string]any{
			"name":           item.Name,
			"node_type":      goreactcommon.NodeTypeMemoryItem,
			"session_name":   sessionName,
			"content":        item.Content,
			"type":           string(item.Type),
			"source":         string(item.Source),
			"importance":     item.Importance,
			"emphasis_level": int(item.EmphasisLevel),
			"created_at":     item.CreatedAt.Format(time.RFC3339),
		},
	}

	if err := a.graphRAG.AddNode(ctx, node); err != nil {
		return nil, err
	}

	// Create edge from session to memory item
	edge := &core.Edge{
		ID:     fmt.Sprintf("session-%s-memory-%s", sessionName, item.Name),
		Type:   "HAS_MEMORY_ITEM",
		Source: sessionName,
		Target: item.Name,
		Properties: map[string]any{
			"created_at": time.Now().Format(time.RFC3339),
		},
	}

	if err := a.graphRAG.AddEdge(ctx, edge); err != nil {
		return nil, err
	}

	return item, nil
}

// Update updates a memory item
func (a *ShortTermAccessor) Update(ctx context.Context, item *goreactcore.MemoryItemNode) error {
	node := &core.Node{
		ID:   item.Name,
		Type: goreactcommon.NodeTypeMemoryItem,
		Properties: map[string]any{
			"name":           item.Name,
			"node_type":      goreactcommon.NodeTypeMemoryItem,
			"session_name":   item.SessionName,
			"content":        item.Content,
			"type":           string(item.Type),
			"source":         string(item.Source),
			"importance":     item.Importance,
			"emphasis_level": int(item.EmphasisLevel),
			"updated_at":     time.Now().Format(time.RFC3339),
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// List lists memory items for a session
func (a *ShortTermAccessor) List(ctx context.Context, sessionName string) ([]*goreactcore.MemoryItemNode, error) {
	query := fmt.Sprintf(
		"MATCH (s:%s {id: $sessionId})-[:HAS_MEMORY_ITEM]->(m:%s) RETURN m",
		goreactcommon.NodeTypeSession, goreactcommon.NodeTypeMemoryItem,
	)

	results, err := a.graphRAG.QueryGraph(ctx, query, map[string]any{"sessionId": sessionName})
	if err != nil {
		return nil, err
	}

	items := make([]*goreactcore.MemoryItemNode, 0, len(results))
	for _, result := range results {
		if mData, ok := result["m"].(map[string]any); ok {
			item := &goreactcore.MemoryItemNode{
				SessionName:   sessionName,
				Content:       getString(mData["content"]),
				Type:          goreactcommon.MemoryItemType(getString(mData["type"])),
				Source:        goreactcommon.MemorySource(getString(mData["source"])),
				Importance:    getFloat64(mData["importance"]),
				EmphasisLevel: parseEmphasisLevel(mData["emphasis_level"]),
			}
			if name, ok := mData["name"].(string); ok {
				item.Name = name
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// Search searches for relevant memory items using semantic search
func (a *ShortTermAccessor) Search(ctx context.Context, sessionName string, query string, topK int) ([]*goreactcore.MemoryItemNode, error) {
	// Use GraphRAG's retrieve capability for semantic search
	results, err := a.graphRAG.Retrieve(ctx, []string{query}, topK)
	if err != nil {
		return nil, err
	}

	items := make([]*goreactcore.MemoryItemNode, 0, len(results))
	for _, result := range results {
		// Filter by session name and node type
		if session, ok := result.Metadata["session_name"].(string); ok && session == sessionName {
			if nodeType, ok := result.Metadata["node_type"].(string); ok && nodeType == goreactcommon.NodeTypeMemoryItem {
				// Get content from Answer or first Chunk
				content := result.Answer
				if content == "" && len(result.Chunks) > 0 {
					content = result.Chunks[0].Content
				}
				
				item := &goreactcore.MemoryItemNode{
					SessionName:   session,
					Content:       content,
					Type:          goreactcommon.MemoryItemType(getString(result.Metadata["type"])),
					Source:        goreactcommon.MemorySource(getString(result.Metadata["source"])),
					Importance:    getFloat64(result.Metadata["importance"]),
					EmphasisLevel: parseEmphasisLevel(result.Metadata["emphasis_level"]),
				}
				if name, ok := result.Metadata["name"].(string); ok {
					item.Name = name
				}
				items = append(items, item)
			}
		}
	}

	return items, nil
}

// Clear clears all memory items for a session
func (a *ShortTermAccessor) Clear(ctx context.Context, sessionName string) error {
	// Query all memory items for the session
	items, err := a.List(ctx, sessionName)
	if err != nil {
		return err
	}

	// Delete each item
	for _, item := range items {
		if err := a.graphRAG.DeleteNode(ctx, item.Name); err != nil {
			return err
		}
	}

	return nil
}
