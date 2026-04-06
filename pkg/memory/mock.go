package memory

import (
	"context"
	"sync"

	"github.com/DotNetAge/gochat/pkg/embedding"
	goragcore "github.com/DotNetAge/gorag/pkg/core"
	"github.com/DotNetAge/gorag/pkg/indexer"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// MockGraphRAG is an in-memory mock implementation of GraphRAGPattern for testing and demos.
// It stores nodes and edges in memory without persistence.
type MockGraphRAG struct {
	mu    sync.RWMutex
	nodes map[string]*goragcore.Node
	edges map[string]*goragcore.Edge
	texts []string
	docs  map[string]*goragcore.Document
}

// NewMockGraphRAG creates a new mock GraphRAG instance
func NewMockGraphRAG() *MockGraphRAG {
	return &MockGraphRAG{
		nodes: make(map[string]*goragcore.Node),
		edges: make(map[string]*goragcore.Edge),
		texts: make([]string, 0),
		docs:  make(map[string]*goragcore.Document),
	}
}

// Indexer returns a mock indexer
func (m *MockGraphRAG) Indexer() indexer.Indexer {
	return &mockIndexer{mock: m}
}

// Retriever returns a mock retriever
func (m *MockGraphRAG) Retriever() goragcore.Retriever {
	return &mockRetriever{mock: m}
}

// Repository returns a mock repository
func (m *MockGraphRAG) Repository() goragcore.Repository {
	return &mockRepository{mock: m}
}

// GraphRepository returns a mock graph repository
func (m *MockGraphRAG) GraphRepository() goragcore.GraphRepository {
	return &mockGraphRepository{mock: m}
}

// IndexFile mocks file indexing
func (m *MockGraphRAG) IndexFile(ctx context.Context, filePath string) error {
	return nil
}

// IndexDirectory mocks directory indexing
func (m *MockGraphRAG) IndexDirectory(ctx context.Context, dirPath string, recursive bool) error {
	return nil
}

// Retrieve mocks retrieval
func (m *MockGraphRAG) Retrieve(ctx context.Context, queries []string, topK int) ([]*goragcore.RetrievalResult, error) {
	results := make([]*goragcore.RetrievalResult, len(queries))
	for i := range queries {
		results[i] = &goragcore.RetrievalResult{
			Query: queries[i],
		}
	}
	return results, nil
}

// IndexText mocks text indexing
func (m *MockGraphRAG) IndexText(ctx context.Context, text string, metadata ...map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.texts = append(m.texts, text)
	return nil
}

// IndexTexts mocks batch text indexing
func (m *MockGraphRAG) IndexTexts(ctx context.Context, texts []string, metadata ...map[string]any) error {
	for _, text := range texts {
		if err := m.IndexText(ctx, text, metadata...); err != nil {
			return err
		}
	}
	return nil
}

// Delete mocks deletion
func (m *MockGraphRAG) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.nodes, id)
	delete(m.edges, id)
	delete(m.docs, id)
	return nil
}

// AddNode adds a node to the mock graph
func (m *MockGraphRAG) AddNode(ctx context.Context, node *goragcore.Node) error {
	if node == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes[node.ID] = node
	return nil
}

// AddNodes adds multiple nodes to the mock graph
func (m *MockGraphRAG) AddNodes(ctx context.Context, nodes []*goragcore.Node) error {
	for _, node := range nodes {
		if err := m.AddNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

// GetNode retrieves a node by ID
func (m *MockGraphRAG) GetNode(ctx context.Context, nodeID string) (*goragcore.Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	node, ok := m.nodes[nodeID]
	if !ok {
		return nil, nil
	}
	return node, nil
}

// DeleteNode removes a node from the mock graph
func (m *MockGraphRAG) DeleteNode(ctx context.Context, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.nodes, nodeID)
	return nil
}

// AddEdge adds an edge to the mock graph
func (m *MockGraphRAG) AddEdge(ctx context.Context, edge *goragcore.Edge) error {
	if edge == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.edges[edge.ID] = edge
	return nil
}

// AddEdges adds multiple edges to the mock graph
func (m *MockGraphRAG) AddEdges(ctx context.Context, edges []*goragcore.Edge) error {
	for _, edge := range edges {
		if err := m.AddEdge(ctx, edge); err != nil {
			return err
		}
	}
	return nil
}

// DeleteEdge removes an edge from the mock graph
func (m *MockGraphRAG) DeleteEdge(ctx context.Context, edgeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.edges, edgeID)
	return nil
}

// QueryGraph executes a mock query (returns empty results)
func (m *MockGraphRAG) QueryGraph(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

// GetNeighbors returns neighbors (empty in mock)
func (m *MockGraphRAG) GetNeighbors(ctx context.Context, nodeID string, depth int, limit int) ([]*goragcore.Node, []*goragcore.Edge, error) {
	return []*goragcore.Node{}, []*goragcore.Edge{}, nil
}

// Ensure MockGraphRAG implements pattern.GraphRAGPattern
var _ pattern.GraphRAGPattern = (*MockGraphRAG)(nil)

// mockIndexer implements indexing.Indexer
type mockIndexer struct {
	mock *MockGraphRAG
}

func (m *mockIndexer) IndexFile(ctx context.Context, filePath string) (*goragcore.IndexingContext, error) {
	return &goragcore.IndexingContext{}, nil
}

func (m *mockIndexer) IndexDirectory(ctx context.Context, dirPath string, recursive bool) error {
	return nil
}

func (m *mockIndexer) IndexText(ctx context.Context, text string, metadata ...map[string]any) error {
	return m.mock.IndexText(ctx, text, metadata...)
}

func (m *mockIndexer) IndexTexts(ctx context.Context, texts []string, metadata ...map[string]any) error {
	return m.mock.IndexTexts(ctx, texts, metadata...)
}

func (m *mockIndexer) IndexDocuments(ctx context.Context, docs ...*goragcore.Document) error {
	m.mock.mu.Lock()
	defer m.mock.mu.Unlock()
	for _, doc := range docs {
		m.mock.docs[doc.ID] = doc
	}
	return nil
}

func (m *mockIndexer) DeleteDocument(ctx context.Context, docID string) error {
	return m.mock.Delete(ctx, docID)
}

func (m *mockIndexer) GetDocument(ctx context.Context, docID string) (*goragcore.Document, error) {
	m.mock.mu.RLock()
	defer m.mock.mu.RUnlock()
	return m.mock.docs[docID], nil
}

func (m *mockIndexer) Init() error {
	return nil
}

func (m *mockIndexer) Start() error {
	return nil
}

func (m *mockIndexer) VectorStore() goragcore.VectorStore {
	return nil
}

func (m *mockIndexer) DocStore() goragcore.DocStore {
	return nil
}

func (m *mockIndexer) GraphStore() goragcore.GraphStore {
	return nil
}

func (m *mockIndexer) Embedder() embedding.Provider {
	return nil
}

func (m *mockIndexer) Chunker() goragcore.SemanticChunker {
	return nil
}

// mockRetriever implements goragcore.Retriever
type mockRetriever struct {
	mock *MockGraphRAG
}

func (m *mockRetriever) Retrieve(ctx context.Context, queries []string, topK int) ([]*goragcore.RetrievalResult, error) {
	return m.mock.Retrieve(ctx, queries, topK)
}

// mockRepository implements goragcore.Repository
type mockRepository struct {
	mock *MockGraphRAG
}

func (m *mockRepository) Create(ctx context.Context, collection string, entity goragcore.Entity, content string) error {
	return m.mock.IndexText(ctx, content)
}

func (m *mockRepository) Read(ctx context.Context, collection string, id string) (goragcore.Entity, error) {
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, collection string, entity goragcore.Entity, content string) error {
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, collection string, id string) error {
	return m.mock.Delete(ctx, id)
}

func (m *mockRepository) List(ctx context.Context, collection string, filter map[string]any) ([]goragcore.Entity, error) {
	return []goragcore.Entity{}, nil
}

// mockGraphRepository implements goragcore.GraphRepository
type mockGraphRepository struct {
	mock *MockGraphRAG
}

func (m *mockGraphRepository) CreateNode(ctx context.Context, node *goragcore.Node) error {
	return m.mock.AddNode(ctx, node)
}

func (m *mockGraphRepository) ReadNode(ctx context.Context, nodeID string) (*goragcore.Node, error) {
	return m.mock.GetNode(ctx, nodeID)
}

func (m *mockGraphRepository) UpdateNode(ctx context.Context, node *goragcore.Node) error {
	return m.mock.AddNode(ctx, node)
}

func (m *mockGraphRepository) DeleteNode(ctx context.Context, nodeID string) error {
	return m.mock.DeleteNode(ctx, nodeID)
}

func (m *mockGraphRepository) ListNodes(ctx context.Context, filter map[string]any) ([]*goragcore.Node, error) {
	m.mock.mu.RLock()
	defer m.mock.mu.RUnlock()
	nodes := make([]*goragcore.Node, 0, len(m.mock.nodes))
	for _, n := range m.mock.nodes {
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (m *mockGraphRepository) CreateEdge(ctx context.Context, edge *goragcore.Edge) error {
	return m.mock.AddEdge(ctx, edge)
}

func (m *mockGraphRepository) ReadEdge(ctx context.Context, edgeID string) (*goragcore.Edge, error) {
	m.mock.mu.RLock()
	defer m.mock.mu.RUnlock()
	return m.mock.edges[edgeID], nil
}

func (m *mockGraphRepository) UpdateEdge(ctx context.Context, edge *goragcore.Edge) error {
	return m.mock.AddEdge(ctx, edge)
}

func (m *mockGraphRepository) DeleteEdge(ctx context.Context, edgeID string) error {
	return m.mock.DeleteEdge(ctx, edgeID)
}

func (m *mockGraphRepository) ListEdges(ctx context.Context, filter map[string]any) ([]*goragcore.Edge, error) {
	m.mock.mu.RLock()
	defer m.mock.mu.RUnlock()
	edges := make([]*goragcore.Edge, 0, len(m.mock.edges))
	for _, e := range m.mock.edges {
		edges = append(edges, e)
	}
	return edges, nil
}

func (m *mockGraphRepository) GetNeighbors(ctx context.Context, nodeID string, depth int, limit int) ([]*goragcore.Node, []*goragcore.Edge, error) {
	return m.mock.GetNeighbors(ctx, nodeID, depth, limit)
}

func (m *mockGraphRepository) Query(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return m.mock.QueryGraph(ctx, query, params)
}
