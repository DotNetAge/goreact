package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// MigrationService provides data migration between GraphRAG instances
type MigrationService interface {
	Plan(ctx context.Context, config *MigrationConfig) (*MigrationPlan, error)
	Execute(ctx context.Context, plan *MigrationPlan) (*MigrationResult, error)
	Validate(ctx context.Context, result *MigrationResult) (*ValidationResult, error)
	Rollback(ctx context.Context, result *MigrationResult) error
	GetProgress(ctx context.Context, migrationID string) (*MigrationProgress, error)
	Cancel(ctx context.Context, migrationID string) error
}

// MigrationConfig contains migration configuration
type MigrationConfig struct {
	SourceGraphRAG *GraphRAGConfig `json:"source_graph_rag" yaml:"source_graph_rag"`
	TargetGraphRAG *GraphRAGConfig `json:"target_graph_rag" yaml:"target_graph_rag"`
	Options        *MigrationOptions `json:"options" yaml:"options"`
}

// GraphRAGConfig contains GraphRAG connection configuration
type GraphRAGConfig struct {
	Name     string       `json:"name" yaml:"name"`
	Embedding string      `json:"embedding" yaml:"embedding"`
	Graph    *GraphConfig `json:"graph" yaml:"graph"`
	Vector   *VectorConfig `json:"vector" yaml:"vector"`
}

// GraphConfig contains graph database configuration
type GraphConfig struct {
	Type     string `json:"type" yaml:"type"`
	URI      string `json:"uri" yaml:"uri"`
	Database string `json:"database" yaml:"database"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// VectorConfig contains vector store configuration
type VectorConfig struct {
	Type       string `json:"type" yaml:"type"`
	Address    string `json:"address" yaml:"address"`
	Collection string `json:"collection" yaml:"collection"`
	Dimension  int    `json:"dimension" yaml:"dimension"`
}

// MigrationOptions contains migration options
type MigrationOptions struct {
	BatchSize        int               `json:"batch_size" yaml:"batch_size"`
	ParallelWorkers  int               `json:"parallel_workers" yaml:"parallel_workers"`
	DryRun           bool              `json:"dry_run" yaml:"dry_run"`
	SkipValidation   bool              `json:"skip_validation" yaml:"skip_validation"`
	Overwrite        bool              `json:"overwrite" yaml:"overwrite"`
	ContinueOnError  bool              `json:"continue_on_error" yaml:"continue_on_error"`
	Filters          []MigrationFilter `json:"filters" yaml:"filters"`
}

// MigrationFilter contains filter criteria for migration
type MigrationFilter struct {
	Type     FilterType `json:"type" yaml:"type"`
	Field    string     `json:"field" yaml:"field"`
	Operator string     `json:"operator" yaml:"operator"`
	Value    any        `json:"value" yaml:"value"`
}

// FilterType represents the type of migration filter
type FilterType string

const (
	FilterTypeNodeType  FilterType = "node_type"
	FilterTypeEdgeType  FilterType = "edge_type"
	FilterTypeDateRange FilterType = "date_range"
	FilterTypeCustom    FilterType = "custom"
)

// MigrationPlan represents a migration plan
type MigrationPlan struct {
	ID            string          `json:"id" yaml:"id"`
	Config        *MigrationConfig `json:"config" yaml:"config"`
	Steps         []MigrationStep `json:"steps" yaml:"steps"`
	EstimatedTime time.Duration   `json:"estimated_time" yaml:"estimated_time"`
	TotalRecords  int64           `json:"total_records" yaml:"total_records"`
	CreatedAt     time.Time       `json:"created_at" yaml:"created_at"`
}

// MigrationStep represents a step in the migration plan
type MigrationStep struct {
	Name         string   `json:"name" yaml:"name"`
	Type         StepType `json:"type" yaml:"type"`
	SourceQuery  string   `json:"source_query" yaml:"source_query"`
	TargetQuery  string   `json:"target_query" yaml:"target_query"`
	RecordCount  int64    `json:"record_count" yaml:"record_count"`
	Order        int      `json:"order" yaml:"order"`
}

// StepType represents the type of migration step
type StepType string

const (
	StepTypeNodes       StepType = "nodes"
	StepTypeEdges       StepType = "edges"
	StepTypeVectors     StepType = "vectors"
	StepTypeIndexes     StepType = "indexes"
	StepTypeConstraints StepType = "constraints"
)

// MigrationResult represents the result of a migration
type MigrationResult struct {
	ID              string           `json:"id" yaml:"id"`
	PlanID          string           `json:"plan_id" yaml:"plan_id"`
	Status          MigrationStatus  `json:"status" yaml:"status"`
	StartTime       time.Time        `json:"start_time" yaml:"start_time"`
	EndTime         time.Time        `json:"end_time" yaml:"end_time"`
	RecordsMigrated int64            `json:"records_migrated" yaml:"records_migrated"`
	RecordsFailed   int64            `json:"records_failed" yaml:"records_failed"`
	Errors          []MigrationError `json:"errors" yaml:"errors"`
}

// MigrationStatus represents the status of a migration
type MigrationStatus string

const (
	MigrationStatusPending    MigrationStatus = "pending"
	MigrationStatusRunning    MigrationStatus = "running"
	MigrationStatusCompleted  MigrationStatus = "completed"
	MigrationStatusFailed     MigrationStatus = "failed"
	MigrationStatusCancelled  MigrationStatus = "cancelled"
	MigrationStatusRolledBack MigrationStatus = "rolled_back"
)

// MigrationError represents an error during migration
type MigrationError struct {
	Step      string    `json:"step" yaml:"step"`
	RecordID  string    `json:"record_id" yaml:"record_id"`
	Error     string    `json:"error" yaml:"error"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// MigrationProgress represents the progress of a migration
type MigrationProgress struct {
	MigrationID        string          `json:"migration_id" yaml:"migration_id"`
	CurrentStep        string          `json:"current_step" yaml:"current_step"`
	RecordsProcessed   int64           `json:"records_processed" yaml:"records_processed"`
	TotalRecords       int64           `json:"total_records" yaml:"total_records"`
	Percentage         float64         `json:"percentage" yaml:"percentage"`
	EstimatedRemaining time.Duration   `json:"estimated_remaining" yaml:"estimated_remaining"`
	Status             MigrationStatus `json:"status" yaml:"status"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid    bool              `json:"valid" yaml:"valid"`
	Errors   []ValidationError `json:"errors" yaml:"errors"`
	Warnings []ValidationWarning `json:"warnings" yaml:"warnings"`
	CheckedAt time.Time        `json:"checked_at" yaml:"checked_at"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Type    string `json:"type" yaml:"type"`
	ID      string `json:"id" yaml:"id"`
	Field   string `json:"field" yaml:"field"`
	Message string `json:"message" yaml:"message"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Type    string `json:"type" yaml:"type"`
	ID      string `json:"id" yaml:"id"`
	Message string `json:"message" yaml:"message"`
}

// DefaultMigrationOptions returns default migration options
func DefaultMigrationOptions() *MigrationOptions {
	return &MigrationOptions{
		BatchSize:       1000,
		ParallelWorkers: 4,
		DryRun:          false,
		SkipValidation:  false,
		Overwrite:       false,
		ContinueOnError: true,
		Filters:         []MigrationFilter{},
	}
}

// Migrator implements MigrationService
type Migrator struct {
	source    pattern.GraphRAGPattern
	target    pattern.GraphRAGPattern
	config    *MigrationConfig
	extractor DataExtractor
	transformer DataTransformer
	validator DataValidator
	progress  map[string]*MigrationProgress
}

// NewMigrator creates a new Migrator
func NewMigrator(source, target pattern.GraphRAGPattern, config *MigrationConfig) *Migrator {
	if config.Options == nil {
		config.Options = DefaultMigrationOptions()
	}
	
	m := &Migrator{
		source:   source,
		target:   target,
		config:   config,
		progress: make(map[string]*MigrationProgress),
	}
	
	m.extractor = NewGraphRAGExtractor(source)
	m.transformer = NewDefaultTransformer()
	m.validator = NewDefaultValidator()
	
	return m
}

// Plan creates a migration plan
func (m *Migrator) Plan(ctx context.Context, config *MigrationConfig) (*MigrationPlan, error) {
	plan := &MigrationPlan{
		ID:        generateMigrationID(),
		Config:    config,
		Steps:     []MigrationStep{},
		CreatedAt: time.Now(),
	}
	
	// Count nodes
	nodeCount, err := m.extractor.CountNodes(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to count nodes: %w", err)
	}
	
	plan.Steps = append(plan.Steps, MigrationStep{
		Name:        "Migrate Nodes",
		Type:        StepTypeNodes,
		SourceQuery: "MATCH (n) RETURN n",
		RecordCount: nodeCount,
		Order:       1,
	})
	
	// Count edges
	edgeCount, err := m.extractor.CountEdges(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to count edges: %w", err)
	}
	
	plan.Steps = append(plan.Steps, MigrationStep{
		Name:        "Migrate Edges",
		Type:        StepTypeEdges,
		SourceQuery: "MATCH ()-[r]->() RETURN r",
		RecordCount: edgeCount,
		Order:       2,
	})
	
	plan.TotalRecords = nodeCount + edgeCount
	plan.EstimatedTime = time.Duration(plan.TotalRecords/int64(m.config.Options.BatchSize)) * time.Second
	
	return plan, nil
}

// Execute executes a migration plan
func (m *Migrator) Execute(ctx context.Context, plan *MigrationPlan) (*MigrationResult, error) {
	result := &MigrationResult{
		ID:        generateMigrationID(),
		PlanID:    plan.ID,
		Status:    MigrationStatusRunning,
		StartTime: time.Now(),
		Errors:    []MigrationError{},
	}
	
	// Initialize progress
	progress := &MigrationProgress{
		MigrationID:      result.ID,
		RecordsProcessed: 0,
		TotalRecords:     plan.TotalRecords,
		Status:           MigrationStatusRunning,
	}
	m.progress[result.ID] = progress
	
	// Execute each step
	for _, step := range plan.Steps {
		progress.CurrentStep = step.Name
		
		switch step.Type {
		case StepTypeNodes:
			migrated, failed, err := m.migrateNodes(ctx, step)
			if err != nil {
				result.Errors = append(result.Errors, MigrationError{
					Step:      step.Name,
					Error:     err.Error(),
					Timestamp: time.Now(),
				})
				if !m.config.Options.ContinueOnError {
					result.Status = MigrationStatusFailed
					return result, err
				}
			}
			result.RecordsMigrated += migrated
			result.RecordsFailed += failed
			
		case StepTypeEdges:
			migrated, failed, err := m.migrateEdges(ctx, step)
			if err != nil {
				result.Errors = append(result.Errors, MigrationError{
					Step:      step.Name,
					Error:     err.Error(),
					Timestamp: time.Now(),
				})
				if !m.config.Options.ContinueOnError {
					result.Status = MigrationStatusFailed
					return result, err
				}
			}
			result.RecordsMigrated += migrated
			result.RecordsFailed += failed
		}
		
		progress.RecordsProcessed += step.RecordCount
		progress.Percentage = float64(progress.RecordsProcessed) / float64(progress.TotalRecords) * 100
	}
	
	result.Status = MigrationStatusCompleted
	result.EndTime = time.Now()
	progress.Status = MigrationStatusCompleted
	
	return result, nil
}

// migrateNodes migrates nodes from source to target
func (m *Migrator) migrateNodes(ctx context.Context, step MigrationStep) (int64, int64, error) {
	var migrated, failed int64
	
	batchSize := m.config.Options.BatchSize
	offset := 0
	
	for {
		nodes, err := m.extractor.ExtractNodes(ctx, "", 
			WithLimit(batchSize),
			WithOffset(offset),
		)
		if err != nil {
			return migrated, failed, err
		}
		
		if len(nodes) == 0 {
			break
		}
		
		for _, node := range nodes {
			// Transform node
			transformed, err := m.transformer.TransformNode(node)
			if err != nil {
				failed++
				continue
			}
			
			// Validate node
			if err := m.validator.ValidateNode(transformed); err != nil {
				failed++
				continue
			}
			
			// Add to target
			if !m.config.Options.DryRun {
				if err := m.target.AddNode(ctx, transformed); err != nil {
					failed++
					continue
				}
			}
			migrated++
		}
		
		offset += batchSize
	}
	
	return migrated, failed, nil
}

// migrateEdges migrates edges from source to target
func (m *Migrator) migrateEdges(ctx context.Context, step MigrationStep) (int64, int64, error) {
	var migrated, failed int64
	
	batchSize := m.config.Options.BatchSize
	offset := 0
	
	for {
		edges, err := m.extractor.ExtractEdges(ctx, "",
			WithLimit(batchSize),
			WithOffset(offset),
		)
		if err != nil {
			return migrated, failed, err
		}
		
		if len(edges) == 0 {
			break
		}
		
		for _, edge := range edges {
			// Transform edge
			transformed, err := m.transformer.TransformEdge(edge)
			if err != nil {
				failed++
				continue
			}
			
			// Validate edge
			if err := m.validator.ValidateEdge(transformed); err != nil {
				failed++
				continue
			}
			
			// Add to target
			if !m.config.Options.DryRun {
				if err := m.target.AddEdge(ctx, transformed); err != nil {
					failed++
					continue
				}
			}
			migrated++
		}
		
		offset += batchSize
	}
	
	return migrated, failed, nil
}

// Validate validates a migration result
func (m *Migrator) Validate(ctx context.Context, result *MigrationResult) (*ValidationResult, error) {
	validation := &ValidationResult{
		Valid:     true,
		Errors:    []ValidationError{},
		Warnings:  []ValidationWarning{},
		CheckedAt: time.Now(),
	}
	
	// Validate integrity
	if err := m.validator.ValidateIntegrity(ctx, result); err != nil {
		validation.Valid = false
		validation.Errors = append(validation.Errors, ValidationError{
			Type:    "integrity",
			Message: err.Error(),
		})
	}
	
	return validation, nil
}

// Rollback rolls back a migration
func (m *Migrator) Rollback(ctx context.Context, result *MigrationResult) error {
	// Mark as rolled back
	result.Status = MigrationStatusRolledBack
	
	if progress, ok := m.progress[result.ID]; ok {
		progress.Status = MigrationStatusRolledBack
	}
	
	return nil
}

// GetProgress gets migration progress
func (m *Migrator) GetProgress(ctx context.Context, migrationID string) (*MigrationProgress, error) {
	progress, ok := m.progress[migrationID]
	if !ok {
		return nil, fmt.Errorf("migration not found: %s", migrationID)
	}
	return progress, nil
}

// Cancel cancels a migration
func (m *Migrator) Cancel(ctx context.Context, migrationID string) error {
	progress, ok := m.progress[migrationID]
	if !ok {
		return fmt.Errorf("migration not found: %s", migrationID)
	}
	
	progress.Status = MigrationStatusCancelled
	return nil
}

// DataExtractor extracts data from GraphRAG
type DataExtractor interface {
	ExtractNodes(ctx context.Context, nodeType string, opts ...ListOption) ([]*core.Node, error)
	ExtractEdges(ctx context.Context, edgeType string, opts ...ListOption) ([]*core.Edge, error)
	CountNodes(ctx context.Context, nodeType string) (int64, error)
	CountEdges(ctx context.Context, edgeType string) (int64, error)
}

// GraphRAGExtractor implements DataExtractor
type GraphRAGExtractor struct {
	graphRAG pattern.GraphRAGPattern
}

// NewGraphRAGExtractor creates a new GraphRAGExtractor
func NewGraphRAGExtractor(graphRAG pattern.GraphRAGPattern) *GraphRAGExtractor {
	return &GraphRAGExtractor{graphRAG: graphRAG}
}

// ExtractNodes extracts nodes from GraphRAG
func (e *GraphRAGExtractor) ExtractNodes(ctx context.Context, nodeType string, opts ...ListOption) ([]*core.Node, error) {
	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}
	
	query := "MATCH (n"
	if nodeType != "" {
		query = fmt.Sprintf("MATCH (n:%s", nodeType)
	}
	query += ") RETURN n"
	
	if options.Limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, options.Limit)
	}
	if options.Offset > 0 {
		query = fmt.Sprintf("%s SKIP %d", query, options.Offset)
	}
	
	results, err := e.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	
	nodes := make([]*core.Node, 0, len(results))
	for _, result := range results {
		if nData, ok := result["n"].(map[string]any); ok {
			node := &core.Node{
				Properties: nData,
			}
			if id, ok := nData["id"].(string); ok {
				node.ID = id
			}
			if t, ok := nData["node_type"].(string); ok {
				node.Type = t
			}
			nodes = append(nodes, node)
		}
	}
	
	return nodes, nil
}

// ExtractEdges extracts edges from GraphRAG
func (e *GraphRAGExtractor) ExtractEdges(ctx context.Context, edgeType string, opts ...ListOption) ([]*core.Edge, error) {
	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}
	
	query := "MATCH ()-[r"
	if edgeType != "" {
		query = fmt.Sprintf("MATCH ()-[r:%s", edgeType)
	}
	query += "]->() RETURN r"
	
	if options.Limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, options.Limit)
	}
	if options.Offset > 0 {
		query = fmt.Sprintf("%s SKIP %d", query, options.Offset)
	}
	
	results, err := e.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	
	edges := make([]*core.Edge, 0, len(results))
	for _, result := range results {
		if rData, ok := result["r"].(map[string]any); ok {
			edge := &core.Edge{
				Properties: rData,
			}
			if id, ok := rData["id"].(string); ok {
				edge.ID = id
			}
			if t, ok := rData["type"].(string); ok {
				edge.Type = t
			}
			if src, ok := rData["source"].(string); ok {
				edge.Source = src
			}
			if tgt, ok := rData["target"].(string); ok {
				edge.Target = tgt
			}
			edges = append(edges, edge)
		}
	}
	
	return edges, nil
}

// CountNodes counts nodes in GraphRAG
func (e *GraphRAGExtractor) CountNodes(ctx context.Context, nodeType string) (int64, error) {
	query := "MATCH (n"
	if nodeType != "" {
		query = fmt.Sprintf("MATCH (n:%s", nodeType)
	}
	query += ") RETURN count(n) as count"
	
	results, err := e.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return 0, err
	}
	
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			return count, nil
		}
	}
	
	return 0, nil
}

// CountEdges counts edges in GraphRAG
func (e *GraphRAGExtractor) CountEdges(ctx context.Context, edgeType string) (int64, error) {
	query := "MATCH ()-[r"
	if edgeType != "" {
		query = fmt.Sprintf("MATCH ()-[r:%s", edgeType)
	}
	query += "]->() RETURN count(r) as count"
	
	results, err := e.graphRAG.QueryGraph(ctx, query, nil)
	if err != nil {
		return 0, err
	}
	
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			return count, nil
		}
	}
	
	return 0, nil
}

// DataTransformer transforms data during migration
type DataTransformer interface {
	TransformNode(node *core.Node) (*core.Node, error)
	TransformEdge(edge *core.Edge) (*core.Edge, error)
	RegisterNodeTransformer(nodeType string, transformer NodeTransformer)
	RegisterEdgeTransformer(edgeType string, transformer EdgeTransformer)
}

// NodeTransformer transforms a node
type NodeTransformer func(node *core.Node) (*core.Node, error)

// EdgeTransformer transforms an edge
type EdgeTransformer func(edge *core.Edge) (*core.Edge, error)

// DefaultTransformer implements DataTransformer
type DefaultTransformer struct {
	nodeTransformers map[string]NodeTransformer
	edgeTransformers map[string]EdgeTransformer
}

// NewDefaultTransformer creates a new DefaultTransformer
func NewDefaultTransformer() *DefaultTransformer {
	return &DefaultTransformer{
		nodeTransformers: make(map[string]NodeTransformer),
		edgeTransformers: make(map[string]EdgeTransformer),
	}
}

// TransformNode transforms a node
func (t *DefaultTransformer) TransformNode(node *core.Node) (*core.Node, error) {
	if transformer, ok := t.nodeTransformers[node.Type]; ok {
		return transformer(node)
	}
	return node, nil
}

// TransformEdge transforms an edge
func (t *DefaultTransformer) TransformEdge(edge *core.Edge) (*core.Edge, error) {
	if transformer, ok := t.edgeTransformers[edge.Type]; ok {
		return transformer(edge)
	}
	return edge, nil
}

// RegisterNodeTransformer registers a node transformer
func (t *DefaultTransformer) RegisterNodeTransformer(nodeType string, transformer NodeTransformer) {
	t.nodeTransformers[nodeType] = transformer
}

// RegisterEdgeTransformer registers an edge transformer
func (t *DefaultTransformer) RegisterEdgeTransformer(edgeType string, transformer EdgeTransformer) {
	t.edgeTransformers[edgeType] = transformer
}

// DataValidator validates data during migration
type DataValidator interface {
	ValidateNode(node *core.Node) error
	ValidateEdge(edge *core.Edge) error
	ValidateIntegrity(ctx context.Context, result *MigrationResult) error
	RegisterNodeValidator(nodeType string, validator NodeValidator)
}

// NodeValidator validates a node
type NodeValidator func(node *core.Node) error

// DefaultValidator implements DataValidator
type DefaultValidator struct {
	nodeValidators map[string]NodeValidator
}

// NewDefaultValidator creates a new DefaultValidator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		nodeValidators: make(map[string]NodeValidator),
	}
}

// ValidateNode validates a node
func (v *DefaultValidator) ValidateNode(node *core.Node) error {
	if node.ID == "" {
		return fmt.Errorf("node ID is required")
	}
	if node.Type == "" {
		return fmt.Errorf("node type is required")
	}
	
	if validator, ok := v.nodeValidators[node.Type]; ok {
		return validator(node)
	}
	
	return nil
}

// ValidateEdge validates an edge
func (v *DefaultValidator) ValidateEdge(edge *core.Edge) error {
	if edge.ID == "" {
		return fmt.Errorf("edge ID is required")
	}
	if edge.Source == "" {
		return fmt.Errorf("edge source is required")
	}
	if edge.Target == "" {
		return fmt.Errorf("edge target is required")
	}
	return nil
}

// ValidateIntegrity validates migration integrity
func (v *DefaultValidator) ValidateIntegrity(ctx context.Context, result *MigrationResult) error {
	// Basic integrity check
	if result.RecordsMigrated == 0 {
		return fmt.Errorf("no records migrated")
	}
	return nil
}

// RegisterNodeValidator registers a node validator
func (v *DefaultValidator) RegisterNodeValidator(nodeType string, validator NodeValidator) {
	v.nodeValidators[nodeType] = validator
}

func generateMigrationID() string {
	return "migration-" + time.Now().Format("20060102150405")
}
