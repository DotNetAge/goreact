package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// Task Planner Implementation
// =============================================================================

// TaskPlannerImpl implements TaskPlanner
type TaskPlannerImpl struct {
	decomposer         Decomposer
	dependencyAnalyzer DependencyAnalyzer
	optimizer          PlanOptimizer
	config             *PlannerConfig
}

// PlannerConfig represents planner configuration
type PlannerConfig struct {
	Strategy           DecompositionStrategy `json:"strategy"`
	MaxSubTasks        int                   `json:"max_sub_tasks"`
	MaxDepth           int                   `json:"max_depth"`
	EnableOptimization bool                  `json:"enable_optimization"`
}

// DefaultPlannerConfig returns default planner config
func DefaultPlannerConfig() *PlannerConfig {
	return &PlannerConfig{
		Strategy:           DecompositionHybrid,
		MaxSubTasks:        10,
		MaxDepth:           3,
		EnableOptimization: true,
	}
}

// NewTaskPlanner creates a new task planner
func NewTaskPlanner(config *PlannerConfig) *TaskPlannerImpl {
	if config == nil {
		config = DefaultPlannerConfig()
	}
	return &TaskPlannerImpl{
		decomposer:         NewDefaultDecomposer(config.Strategy),
		dependencyAnalyzer: NewDefaultDependencyAnalyzer(),
		optimizer:          NewDefaultOptimizer(),
		config:             config,
	}
}

// Plan generates a complete execution plan
func (p *TaskPlannerImpl) Plan(task *Task) (*OrchestrationPlan, error) {
	// Decompose task into sub-tasks
	subTasks, err := p.Decompose(task)
	if err != nil {
		return nil, NewOrchestrationError(ErrorPlanningFailed, "failed to decompose task", err)
	}

	// Analyze dependencies
	graph, err := p.dependencyAnalyzer.Analyze(subTasks)
	if err != nil {
		return nil, NewOrchestrationError(ErrorPlanningFailed, "failed to analyze dependencies", err)
	}

	// Check for cycles
	if cycle, err := p.dependencyAnalyzer.DetectCycle(graph); err != nil && len(cycle) > 0 {
		return nil, NewOrchestrationError(ErrorDependencyViolation,
			fmt.Sprintf("cycle detected: %v", cycle), err)
	}

	// Topological sort to get execution order
	executionOrder, err := p.dependencyAnalyzer.TopologicalSort(graph)
	if err != nil {
		return nil, NewOrchestrationError(ErrorPlanningFailed, "failed to sort tasks", err)
	}

	// Create plan
	plan := &OrchestrationPlan{
		Name:            fmt.Sprintf("plan-%s", task.ID),
		TaskName:        task.Name,
		SubTasks:        subTasks,
		DependencyGraph: graph,
		ExecutionOrder:  executionOrder,
	}

	// Optimize plan
	if p.config.EnableOptimization {
		optimized, err := p.Optimize(plan)
		if err == nil {
			plan = optimized
		}
	}

	// Calculate estimated duration
	plan.EstimatedDuration = p.estimateDuration(plan)

	return plan, nil
}

// Decompose decomposes a task into sub-tasks
func (p *TaskPlannerImpl) Decompose(task *Task) ([]*SubTask, error) {
	return p.decomposer.Decompose(task)
}

// Optimize optimizes the execution plan
func (p *TaskPlannerImpl) Optimize(plan *OrchestrationPlan) (*OrchestrationPlan, error) {
	return p.optimizer.Optimize(plan)
}

// estimateDuration estimates total execution duration
func (p *TaskPlannerImpl) estimateDuration(plan *OrchestrationPlan) time.Duration {
	var total time.Duration
	for _, layer := range plan.ExecutionOrder {
		var maxInLayer time.Duration
		for _, taskName := range layer {
			for _, st := range plan.SubTasks {
				if st.Name == taskName && st.Timeout > maxInLayer {
					maxInLayer = st.Timeout
				}
			}
		}
		total += maxInLayer
	}
	return total
}

// =============================================================================
// Decomposer Implementations
// =============================================================================

// DefaultDecomposer implements Decomposer with configurable strategy
type DefaultDecomposer struct {
	strategy    DecompositionStrategy
	ruleBased   *RuleBasedDecomposer
	llmBased    *LLMBasedDecomposer
}

// NewDefaultDecomposer creates a default decomposer
func NewDefaultDecomposer(strategy DecompositionStrategy) *DefaultDecomposer {
	return &DefaultDecomposer{
		strategy:    strategy,
		ruleBased:   NewRuleBasedDecomposer(),
		llmBased:    NewLLMBasedDecomposer(nil),
	}
}

// Decompose decomposes task based on strategy
func (d *DefaultDecomposer) Decompose(task *Task) ([]*SubTask, error) {
	switch d.strategy {
	case DecompositionRule:
		return d.ruleBased.Decompose(task)
	case DecompositionLLM:
		return d.llmBased.Decompose(task)
	case DecompositionHybrid:
		// Try rule-based first, then LLM
		subTasks, err := d.ruleBased.Decompose(task)
		if err == nil && len(subTasks) > 0 {
			return subTasks, nil
		}
		return d.llmBased.Decompose(task)
	default:
		return nil, fmt.Errorf("unknown decomposition strategy: %s", d.strategy)
	}
}

// Strategy returns the decomposition strategy
func (d *DefaultDecomposer) Strategy() DecompositionStrategy {
	return d.strategy
}

// RuleBasedDecomposer decomposes tasks based on predefined rules
type RuleBasedDecomposer struct {
	rules map[string]*DecompositionRuleSpec
}

// NewRuleBasedDecomposer creates a rule-based decomposer
func NewRuleBasedDecomposer() *RuleBasedDecomposer {
	return &RuleBasedDecomposer{
		rules: getDefaultDecompositionRules(),
	}
}

// Decompose decomposes task using rules
func (d *RuleBasedDecomposer) Decompose(task *Task) ([]*SubTask, error) {
	taskType := d.detectTaskType(task)
	rule, exists := d.rules[taskType]
	if !exists {
		return nil, fmt.Errorf("no rule found for task type: %s", taskType)
	}

	subTasks := make([]*SubTask, len(rule.SubTaskDefs))
	for i, def := range rule.SubTaskDefs {
		subTasks[i] = &SubTask{
			Name:                fmt.Sprintf("%s-%s", task.ID, def.Name),
			ParentName:          task.Name,
			Description:         def.Description,
			RequiredCapabilities: def.Capabilities,
			Dependencies:        def.Dependencies,
			Priority:            task.Priority,
			Timeout:             task.Timeout / time.Duration(len(rule.SubTaskDefs)),
			Input:               task.Input,
		}
	}

	return subTasks, nil
}

// Strategy returns the decomposition strategy
func (d *RuleBasedDecomposer) Strategy() DecompositionStrategy {
	return DecompositionRule
}

// detectTaskType detects the type of task
func (d *RuleBasedDecomposer) detectTaskType(task *Task) string {
	desc := strings.ToLower(task.Description)
	
	if strings.Contains(desc, "code review") || strings.Contains(desc, "代码审查") {
		return "code_review"
	}
	if strings.Contains(desc, "document") || strings.Contains(desc, "文档") {
		return "documentation"
	}
	if strings.Contains(desc, "data") || strings.Contains(desc, "数据") {
		return "data_analysis"
	}
	if strings.Contains(desc, "test") || strings.Contains(desc, "测试") {
		return "testing"
	}
	
	return "general"
}

// getDefaultDecompositionRules returns default decomposition rules
func getDefaultDecompositionRules() map[string]*DecompositionRuleSpec {
	return map[string]*DecompositionRuleSpec{
		"code_review": {
			TaskType: "code_review",
			SubTaskDefs: []SubTaskDef{
				{Name: "syntax", Description: "Check code syntax", Capabilities: []string{"code_analysis", "syntax"}},
				{Name: "style", Description: "Check code style", Capabilities: []string{"code_analysis", "style"}},
				{Name: "security", Description: "Check security issues", Capabilities: []string{"security", "code_analysis"}},
			},
		},
		"documentation": {
			TaskType: "documentation",
			SubTaskDefs: []SubTaskDef{
				{Name: "api", Description: "Generate API documentation", Capabilities: []string{"documentation", "api"}},
				{Name: "user_guide", Description: "Generate user guide", Capabilities: []string{"documentation", "writing"}},
				{Name: "changelog", Description: "Generate changelog", Capabilities: []string{"documentation"}},
			},
		},
		"data_analysis": {
			TaskType: "data_analysis",
			SubTaskDefs: []SubTaskDef{
				{Name: "clean", Description: "Clean data", Capabilities: []string{"data_processing"}},
				{Name: "analyze", Description: "Analyze data", Capabilities: []string{"data_analysis", "statistics"}},
				{Name: "report", Description: "Generate report", Capabilities: []string{"reporting"}},
			},
		},
		"testing": {
			TaskType: "testing",
			SubTaskDefs: []SubTaskDef{
				{Name: "unit", Description: "Unit tests", Capabilities: []string{"testing", "unit_test"}},
				{Name: "integration", Description: "Integration tests", Capabilities: []string{"testing", "integration"}},
				{Name: "coverage", Description: "Coverage analysis", Capabilities: []string{"testing", "coverage"}},
			},
		},
		"general": {
			TaskType: "general",
			SubTaskDefs: []SubTaskDef{
				{Name: "main", Description: "Execute main task", Capabilities: []string{"general"}},
			},
		},
	}
}

// LLMBasedDecomposer decomposes tasks using LLM
type LLMBasedDecomposer struct {
	llmClient LLMClient
}

// LLMClient represents an LLM client interface
type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// NewLLMBasedDecomposer creates an LLM-based decomposer
func NewLLMBasedDecomposer(client LLMClient) *LLMBasedDecomposer {
	return &LLMBasedDecomposer{llmClient: client}
}

// Decompose decomposes task using LLM
func (d *LLMBasedDecomposer) Decompose(task *Task) ([]*SubTask, error) {
	if d.llmClient == nil {
		// Fallback to simple decomposition
		return []*SubTask{{
			Name:                fmt.Sprintf("%s-main", task.ID),
			ParentName:          task.Name,
			Description:         task.Description,
			RequiredCapabilities: []string{"general"},
			Priority:            task.Priority,
			Timeout:             task.Timeout,
			Input:               task.Input,
		}}, nil
	}

	// Build decomposition prompt
	prompt := d.buildDecompositionPrompt(task)
	
	// Call LLM for decomposition
	ctx := context.Background()
	response, err := d.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM decomposition failed: %w", err)
	}
	
	// Parse LLM response into sub-tasks
	subTasks, err := d.parseDecompositionResponse(response, task)
	if err != nil {
		// Fallback to simple decomposition on parse error
		return []*SubTask{{
			Name:                fmt.Sprintf("%s-main", task.ID),
			ParentName:          task.Name,
			Description:         task.Description,
			RequiredCapabilities: []string{"general"},
			Priority:            task.Priority,
			Timeout:             task.Timeout,
			Input:               task.Input,
		}}, nil
	}
	
	return subTasks, nil
}

// buildDecompositionPrompt builds the prompt for LLM-based task decomposition
func (d *LLMBasedDecomposer) buildDecompositionPrompt(task *Task) string {
	return fmt.Sprintf(`You are a task decomposition expert. Decompose the following task into sub-tasks.

## Task
- ID: %s
- Name: %s
- Description: %s
- Priority: %d

## Decomposition Requirements
1. Each sub-task should be independent and executable
2. Define clear dependencies between sub-tasks
3. Specify required capabilities for each sub-task
4. Ensure sub-tasks can be parallelized where possible

## Output Format
Respond in JSON format:
{
  "sub_tasks": [
    {
      "name": "sub-task-name",
      "description": "what this sub-task does",
      "capabilities": ["capability1", "capability2"],
      "dependencies": ["dependency-task-name"],
      "estimated_duration": "duration string"
    }
  ]
}
`, task.ID, task.Name, task.Description, task.Priority)
}

// parseDecompositionResponse parses the LLM response into sub-tasks
func (d *LLMBasedDecomposer) parseDecompositionResponse(response string, task *Task) ([]*SubTask, error) {
	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	jsonStr := response[jsonStart : jsonEnd+1]
	
	var parsed struct {
		SubTasks []struct {
			Name              string   `json:"name"`
			Description       string   `json:"description"`
			Capabilities      []string `json:"capabilities"`
			Dependencies      []string `json:"dependencies"`
			EstimatedDuration string   `json:"estimated_duration"`
		} `json:"sub_tasks"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Convert to SubTask slice
	subTasks := make([]*SubTask, len(parsed.SubTasks))
	for i, st := range parsed.SubTasks {
		timeout := task.Timeout
		if st.EstimatedDuration != "" {
			if d, err := time.ParseDuration(st.EstimatedDuration); err == nil {
				timeout = d
			}
		}
		
		subTasks[i] = &SubTask{
			Name:                 fmt.Sprintf("%s-%s", task.ID, st.Name),
			ParentName:           task.Name,
			Description:          st.Description,
			RequiredCapabilities: st.Capabilities,
			Dependencies:         st.Dependencies,
			Priority:             task.Priority,
			Timeout:              timeout,
			Input:                task.Input,
		}
	}
	
	return subTasks, nil
}

// Strategy returns the decomposition strategy
func (d *LLMBasedDecomposer) Strategy() DecompositionStrategy {
	return DecompositionLLM
}

// =============================================================================
// Dependency Analyzer Implementation
// =============================================================================

// DefaultDependencyAnalyzer implements DependencyAnalyzer
type DefaultDependencyAnalyzer struct{}

// NewDefaultDependencyAnalyzer creates a default dependency analyzer
func NewDefaultDependencyAnalyzer() *DefaultDependencyAnalyzer {
	return &DefaultDependencyAnalyzer{}
}

// Analyze analyzes dependencies between sub-tasks
func (a *DefaultDependencyAnalyzer) Analyze(subTasks []*SubTask) (*Graph, error) {
	graph := &Graph{
		Nodes: make([]string, len(subTasks)),
		Edges: []*Edge{},
	}

	// Add nodes
	for i, st := range subTasks {
		graph.Nodes[i] = st.Name
	}

	// Add edges based on explicit dependencies
	for _, st := range subTasks {
		for _, dep := range st.Dependencies {
			graph.Edges = append(graph.Edges, &Edge{
				From: dep,
				To:   st.Name,
				Type: DependencySequential,
			})
		}
	}

	return graph, nil
}

// TopologicalSort performs Kahn's algorithm for topological sort
// Returns tasks grouped by execution layer (all tasks in a layer can be executed in parallel)
func (a *DefaultDependencyAnalyzer) TopologicalSort(graph *Graph) ([][]string, error) {
	// Build adjacency list and in-degree map
	adjList := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize
	for _, node := range graph.Nodes {
		adjList[node] = []string{}
		inDegree[node] = 0
	}

	// Build adjacency list
	for _, edge := range graph.Edges {
		adjList[edge.From] = append(adjList[edge.From], edge.To)
		inDegree[edge.To]++
	}

	// Kahn's algorithm with layer grouping
	var layers [][]string
	currentLayer := []string{}

	// Find all nodes with in-degree 0
	for node, degree := range inDegree {
		if degree == 0 {
			currentLayer = append(currentLayer, node)
		}
	}

	for len(currentLayer) > 0 {
		layers = append(layers, currentLayer)
		nextLayer := []string{}

		for _, node := range currentLayer {
			for _, neighbor := range adjList[node] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					nextLayer = append(nextLayer, neighbor)
				}
			}
		}

		currentLayer = nextLayer
	}

	// Check if all nodes are processed
	totalNodes := 0
	for _, layer := range layers {
		totalNodes += len(layer)
	}

	if totalNodes != len(graph.Nodes) {
		return nil, fmt.Errorf("graph contains a cycle")
	}

	return layers, nil
}

// DetectCycle detects cycles in the dependency graph using DFS
func (a *DefaultDependencyAnalyzer) DetectCycle(graph *Graph) ([]string, error) {
	// Build adjacency list
	adjList := make(map[string][]string)
	for _, node := range graph.Nodes {
		adjList[node] = []string{}
	}
	for _, edge := range graph.Edges {
		adjList[edge.From] = append(adjList[edge.From], edge.To)
	}

	// DFS with cycle detection
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cycle []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		inStack[node] = true

		for _, neighbor := range adjList[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					cycle = append(cycle, node)
					return true
				}
			} else if inStack[neighbor] {
				cycle = append(cycle, node, neighbor)
				return true
			}
		}

		inStack[node] = false
		return false
	}

	for _, node := range graph.Nodes {
		if !visited[node] {
			if dfs(node) {
				// Reverse to get correct order
				for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
					cycle[i], cycle[j] = cycle[j], cycle[i]
				}
				return cycle, fmt.Errorf("cycle detected")
			}
		}
	}

	return nil, nil
}

// =============================================================================
// Plan Optimizer
// =============================================================================

// PlanOptimizer optimizes execution plans
type PlanOptimizer interface {
	Optimize(plan *OrchestrationPlan) (*OrchestrationPlan, error)
}

// DefaultOptimizer implements PlanOptimizer
type DefaultOptimizer struct{}

// NewDefaultOptimizer creates a default optimizer
func NewDefaultOptimizer() *DefaultOptimizer {
	return &DefaultOptimizer{}
}

// Optimize optimizes the execution plan
func (o *DefaultOptimizer) Optimize(plan *OrchestrationPlan) (*OrchestrationPlan, error) {
	// Create a copy of the plan
	optimized := &OrchestrationPlan{
		Name:              plan.Name + "-optimized",
		TaskName:          plan.TaskName,
		SubTasks:          make([]*SubTask, len(plan.SubTasks)),
		DependencyGraph:   plan.DependencyGraph,
		ExecutionOrder:    plan.ExecutionOrder,
		EstimatedDuration: plan.EstimatedDuration,
	}

	// Copy sub-tasks
	for i, st := range plan.SubTasks {
		optimized.SubTasks[i] = &SubTask{
			Name:                 st.Name,
			ParentName:           st.ParentName,
			Description:          st.Description,
			RequiredCapabilities: st.RequiredCapabilities,
			Dependencies:         st.Dependencies,
			Priority:             st.Priority,
			Timeout:              st.Timeout,
			Input:                st.Input,
		}
	}

	// Apply optimizations
	o.balanceLoad(optimized)
	o.optimizeTimeouts(optimized)

	return optimized, nil
}

// balanceLoad balances load across execution layers
func (o *DefaultOptimizer) balanceLoad(plan *OrchestrationPlan) {
	// Group tasks by capability to enable better parallelization
	capabilityGroups := make(map[string][]*SubTask)
	for _, st := range plan.SubTasks {
		for _, cap := range st.RequiredCapabilities {
			capabilityGroups[cap] = append(capabilityGroups[cap], st)
		}
	}
	
	// Reorder execution order to maximize parallelization
	// Tasks with the same capability can often be parallelized
	newOrder := make([][]string, 0)
	processed := make(map[string]bool)
	
	for _, layer := range plan.ExecutionOrder {
		newLayer := make([]string, 0)
		for _, taskName := range layer {
			if !processed[taskName] {
				newLayer = append(newLayer, taskName)
				processed[taskName] = true
			}
		}
		if len(newLayer) > 0 {
			newOrder = append(newOrder, newLayer)
		}
	}
	
	// Add any unprocessed tasks
	for _, st := range plan.SubTasks {
		if !processed[st.Name] {
			// Find the earliest layer where dependencies are satisfied
			added := false
			for i, layer := range newOrder {
				if o.canAddToLayer(st, layer, processed) {
					newOrder[i] = append(newOrder[i], st.Name)
					processed[st.Name] = true
					added = true
					break
				}
			}
			if !added {
				newOrder = append(newOrder, []string{st.Name})
				processed[st.Name] = true
			}
		}
	}
	
	plan.ExecutionOrder = newOrder
}

// canAddToLayer checks if a task can be added to a layer
func (o *DefaultOptimizer) canAddToLayer(task *SubTask, layer []string, processed map[string]bool) bool {
	// Check if all dependencies are satisfied in previous layers
	for _, dep := range task.Dependencies {
		if !processed[dep] {
			return false
		}
		// Check that dependency is not in current layer
		for _, name := range layer {
			if name == dep {
				return false
			}
		}
	}
	return true
}

// optimizeTimeouts adjusts timeouts based on task complexity
func (o *DefaultOptimizer) optimizeTimeouts(plan *OrchestrationPlan) {
	for _, st := range plan.SubTasks {
		if st.Timeout == 0 {
			// Set default timeout based on complexity
			st.Timeout = 5 * time.Minute
		}
	}
}
