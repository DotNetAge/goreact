package memory

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gorag/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// Accessor interface defines common operations for all node type accessors
type Accessor interface {
	NodeType() string
	Get(ctx context.Context, id string) (*core.Node, error)
	List(ctx context.Context, opts ...ListOption) ([]*core.Node, error)
	Delete(ctx context.Context, id string) error
}

// BaseAccessor provides base accessor functionality.
// It holds a reference to GraphRAG for all storage operations.
type BaseAccessor struct {
	graphRAG pattern.GraphRAGPattern
	nodeType string
}

// NodeType returns the node type managed by this accessor
func (a *BaseAccessor) NodeType() string {
	return a.nodeType
}

// Get retrieves a node by ID from GraphRAG
func (a *BaseAccessor) Get(ctx context.Context, id string) (*core.Node, error) {
	if a.graphRAG == nil {
		return nil, fmt.Errorf("graphRAG is not initialized")
	}
	return a.graphRAG.GetNode(ctx, id)
}

// List retrieves all nodes of this type from GraphRAG
func (a *BaseAccessor) List(ctx context.Context, opts ...ListOption) ([]*core.Node, error) {
	if a.graphRAG == nil {
		return nil, fmt.Errorf("graphRAG is not initialized")
	}

	// Parse options
	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Query nodes by type using graph query
	query := fmt.Sprintf("MATCH (n:%s) RETURN n", a.nodeType)
	if options.Limit > 0 {
		query = fmt.Sprintf("MATCH (n:%s) RETURN n LIMIT %d", a.nodeType, options.Limit)
	}

	results, err := a.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	nodes := make([]*core.Node, 0, len(results))
	for _, result := range results {
		if nodeData, ok := result["n"].(map[string]any); ok {
			node := &core.Node{
				ID:         nodeData["id"].(string),
				Type:       a.nodeType,
				Properties: nodeData,
			}
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// Delete removes a node from GraphRAG
func (a *BaseAccessor) Delete(ctx context.Context, id string) error {
	if a.graphRAG == nil {
		return fmt.Errorf("graphRAG is not initialized")
	}
	return a.graphRAG.DeleteNode(ctx, id)
}

// ListOption is a function that configures list options
type ListOption func(*ListOptions)

// ListOptions contains options for listing
type ListOptions struct {
	Limit  int
	Offset int
	Filter map[string]any
	Order  string
}

// WithLimit sets the limit for list operations
func WithLimit(limit int) ListOption {
	return func(o *ListOptions) {
		o.Limit = limit
	}
}

// WithOffset sets the offset for list operations
func WithOffset(offset int) ListOption {
	return func(o *ListOptions) {
		o.Offset = offset
	}
}

// WithFilter sets the filter for list operations
func WithFilter(filter map[string]any) ListOption {
	return func(o *ListOptions) {
		o.Filter = filter
	}
}

// WithOrder sets the order for list operations
func WithOrder(order string) ListOption {
	return func(o *ListOptions) {
		o.Order = order
	}
}
