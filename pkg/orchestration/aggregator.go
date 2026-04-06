package orchestration

import (
	"fmt"
	"strings"
	"sync"
)

// =============================================================================
// Result Aggregator Implementation
// =============================================================================

// ResultAggregator implements Aggregator interface
type ResultAggregator struct {
	validator ResultValidator
	merger    ResultMerger
	config    *AggregatorConfig
}

// AggregatorConfig represents aggregator configuration
type AggregatorConfig struct {
	MergeStrategy MergeStrategy `json:"merge_strategy"`
	ValidateResults bool        `json:"validate_results"`
}

// DefaultAggregatorConfig returns default aggregator config
func DefaultAggregatorConfig() *AggregatorConfig {
	return &AggregatorConfig{
		MergeStrategy:   MergeStrategyConcat,
		ValidateResults: true,
	}
}

// NewResultAggregator creates a new result aggregator
func NewResultAggregator(config *AggregatorConfig) *ResultAggregator {
	if config == nil {
		config = DefaultAggregatorConfig()
	}
	return &ResultAggregator{
		validator: NewDefaultResultValidator(),
		merger:    NewDefaultResultMerger(config.MergeStrategy),
		config:    config,
	}
}

// Aggregate aggregates sub-task results into final result
func (a *ResultAggregator) Aggregate(results []*SubResult) (*Result, error) {
	if len(results) == 0 {
		return &Result{Success: true}, nil
	}

	// Validate results if enabled
	if a.config.ValidateResults {
		if err := a.validator.Validate(results); err != nil {
			return nil, NewOrchestrationError(ErrorExecutionFailed, "result validation failed", err)
		}
	}

	// Merge results
	mergedOutput := a.merger.Merge(results)

	// Calculate overall success
	success := true
	for _, r := range results {
		if !r.Success {
			success = false
			break
		}
	}

	return &Result{
		SubResults:  results,
		FinalOutput: mergedOutput,
		Success:     success,
	}, nil
}

// Merge merges multiple results into a string
func (a *ResultAggregator) Merge(results []*SubResult) string {
	return a.merger.MergeToString(results)
}

// Validate validates the results
func (a *ResultAggregator) Validate(results []*SubResult) error {
	return a.validator.Validate(results)
}

// =============================================================================
// Result Merger Interface and Implementation
// =============================================================================

// ResultMerger merges results
type ResultMerger interface {
	Merge(results []*SubResult) map[string]any
	MergeToString(results []*SubResult) string
}

// DefaultResultMerger implements ResultMerger
type DefaultResultMerger struct {
	strategy MergeStrategy
}

// NewDefaultResultMerger creates a default result merger
func NewDefaultResultMerger(strategy MergeStrategy) *DefaultResultMerger {
	return &DefaultResultMerger{strategy: strategy}
}

// Merge merges results based on strategy
func (m *DefaultResultMerger) Merge(results []*SubResult) map[string]any {
	switch m.strategy {
	case MergeStrategyConcat:
		return m.mergeConcat(results)
	case MergeStrategyStructured:
		return m.mergeStructured(results)
	case MergeStrategyLLM:
		return m.mergeLLM(results)
	case MergeStrategyVoting:
		return m.mergeVoting(results)
	default:
		return m.mergeConcat(results)
	}
}

// MergeToString merges results to string
func (m *DefaultResultMerger) MergeToString(results []*SubResult) string {
	var builder strings.Builder
	for _, r := range results {
		if r.Success {
			for k, v := range r.Output {
				builder.WriteString(fmt.Sprintf("%s: %v\n", k, v))
			}
		}
	}
	return builder.String()
}

// mergeConcat merges by simple concatenation
func (m *DefaultResultMerger) mergeConcat(results []*SubResult) map[string]any {
	merged := make(map[string]any)
	for _, r := range results {
		if r.Success {
			for k, v := range r.Output {
				merged[r.SubTaskName+"_"+k] = v
			}
		}
	}
	return merged
}

// mergeStructured merges by structured grouping
func (m *DefaultResultMerger) mergeStructured(results []*SubResult) map[string]any {
	merged := make(map[string]any)
	merged["results"] = make([]map[string]any, 0)
	
	for _, r := range results {
		if r.Success {
			result := map[string]any{
				"sub_task": r.SubTaskName,
				"agent":    r.AgentName,
				"output":   r.Output,
				"duration": r.Duration.String(),
			}
			merged["results"] = append(merged["results"].([]map[string]any), result)
		}
	}
	
	return merged
}

// mergeLLM merges using LLM (placeholder)
func (m *DefaultResultMerger) mergeLLM(results []*SubResult) map[string]any {
	// TODO: Implement LLM-based merging
	// For now, fall back to structured merge
	return m.mergeStructured(results)
}

// mergeVoting merges by voting
func (m *DefaultResultMerger) mergeVoting(results []*SubResult) map[string]any {
	// Collect votes for each key
	votes := make(map[string][]any)
	for _, r := range results {
		if r.Success {
			for k, v := range r.Output {
				votes[k] = append(votes[k], v)
			}
		}
	}
	
	// Select most common value for each key
	merged := make(map[string]any)
	for k, vals := range votes {
		merged[k] = m.selectMostCommon(vals)
	}
	
	return merged
}

// selectMostCommon selects the most common value
func (m *DefaultResultMerger) selectMostCommon(vals []any) any {
	counts := make(map[string]int)
	for _, v := range vals {
		key := fmt.Sprintf("%v", v)
		counts[key]++
	}
	
	maxCount := 0
	var result string
	for k, c := range counts {
		if c > maxCount {
			maxCount = c
			result = k
		}
	}
	
	return result
}

// =============================================================================
// Result Validator Implementation
// =============================================================================

// DefaultResultValidator implements ResultValidator
type DefaultResultValidator struct {
	rules []*ValidationRule
	mu    sync.RWMutex
}

// NewDefaultResultValidator creates a default result validator
func NewDefaultResultValidator() *DefaultResultValidator {
	v := &DefaultResultValidator{
		rules: []*ValidationRule{},
	}
	
	// Add default rules
	v.AddRule(&ValidationRule{
		Name:     "completeness",
		Check:    v.checkCompleteness,
		Severity: SeverityError,
	})
	v.AddRule(&ValidationRule{
		Name:     "consistency",
		Check:    v.checkConsistency,
		Severity: SeverityWarning,
	})
	v.AddRule(&ValidationRule{
		Name:     "quality",
		Check:    v.checkQuality,
		Severity: SeverityInfo,
	})
	
	return v
}

// Validate validates the results
func (v *DefaultResultValidator) Validate(results []*SubResult) error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	for _, rule := range v.rules {
		for _, result := range results {
			if err := rule.Check(result); err != nil {
				if rule.Severity == SeverityError {
					return fmt.Errorf("validation rule '%s' failed: %w", rule.Name, err)
				}
				// Log warning/info
			}
		}
	}
	
	return nil
}

// AddRule adds a validation rule
func (v *DefaultResultValidator) AddRule(rule *ValidationRule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules = append(v.rules, rule)
}

// checkCompleteness checks if result is complete
func (v *DefaultResultValidator) checkCompleteness(result *SubResult) error {
	if result == nil {
		return fmt.Errorf("result is nil")
	}
	if result.SubTaskName == "" {
		return fmt.Errorf("sub-task name is empty")
	}
	return nil
}

// checkConsistency checks result consistency
func (v *DefaultResultValidator) checkConsistency(result *SubResult) error {
	// Check if success status matches error presence
	if result.Success && result.Error != nil {
		return fmt.Errorf("result marked as success but has error")
	}
	if !result.Success && result.Error == nil {
		return fmt.Errorf("result marked as failed but has no error")
	}
	return nil
}

// checkQuality checks result quality
func (v *DefaultResultValidator) checkQuality(result *SubResult) error {
	if !result.Success {
		return nil // Skip quality check for failed results
	}
	
	// Basic quality checks
	if result.Output == nil || len(result.Output) == 0 {
		return fmt.Errorf("successful result has empty output")
	}
	
	return nil
}

// =============================================================================
// LLM-Based Merger (Advanced)
// =============================================================================

// LLMMerger uses LLM to intelligently merge results
type LLMMerger struct {
	llmClient LLMClient
	prompt    string
}

// NewLLMMerger creates an LLM-based merger
func NewLLMMerger(client LLMClient) *LLMMerger {
	return &LLMMerger{
		llmClient: client,
		prompt:    getDefaultMergePrompt(),
	}
}

// Merge merges results using LLM
func (m *LLMMerger) Merge(results []*SubResult) map[string]any {
	// TODO: Implement LLM-based merging
	// This would send results to LLM with merge prompt
	// and parse the response
	return make(map[string]any)
}

// MergeToString merges results to string using LLM
func (m *LLMMerger) MergeToString(results []*SubResult) string {
	// TODO: Implement
	return ""
}

func getDefaultMergePrompt() string {
	return `You are a result aggregator. Merge the following sub-task results into a coherent final result.

Results:
{{range .Results}}
- Task: {{.SubTaskName}}
  Agent: {{.AgentName}}
  Output: {{.Output}}
{{end}}

Provide a consolidated summary that captures all key information.`
}

// =============================================================================
// Validation Types
// =============================================================================

// ValidationResult represents validation result
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationError   `json:"warnings,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// =============================================================================
// Result Collector
// =============================================================================

// ResultCollector collects results from multiple sources
type ResultCollector struct {
	results []*SubResult
	mu      sync.Mutex
}

// NewResultCollector creates a new result collector
func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		results: make([]*SubResult, 0),
	}
}

// Add adds a result
func (c *ResultCollector) Add(result *SubResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = append(c.results, result)
}

// AddAll adds multiple results
func (c *ResultCollector) AddAll(results []*SubResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = append(c.results, results...)
}

// Get returns all collected results
func (c *ResultCollector) Get() []*SubResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.results
}

// Clear clears collected results
func (c *ResultCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = make([]*SubResult, 0)
}

// Count returns the count of results
func (c *ResultCollector) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.results)
}

// Successful returns only successful results
func (c *ResultCollector) Successful() []*SubResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	successful := make([]*SubResult, 0)
	for _, r := range c.results {
		if r.Success {
			successful = append(successful, r)
		}
	}
	return successful
}

// Failed returns only failed results
func (c *ResultCollector) Failed() []*SubResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	failed := make([]*SubResult, 0)
	for _, r := range c.results {
		if !r.Success {
			failed = append(failed, r)
		}
	}
	return failed
}
