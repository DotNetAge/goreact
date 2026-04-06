package orchestration

import (
	"context"
	"encoding/json"
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

// mergeLLM merges using LLM
func (m *DefaultResultMerger) mergeLLM(results []*SubResult) map[string]any {
	// LLM-based merging requires an LLM client
	// Fall back to structured merge if no LLM client is available
	// This method is kept for interface compatibility
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
	if m.llmClient == nil || len(results) == 0 {
		return make(map[string]any)
	}
	
	// Build merge prompt
	prompt := m.buildMergePrompt(results)
	
	// Call LLM for merging
	ctx := context.Background()
	response, err := m.llmClient.Generate(ctx, prompt)
	if err != nil {
		// Fallback to structured merge on error
		return m.fallbackMerge(results)
	}
	
	// Parse LLM response
	merged, err := m.parseMergeResponse(response)
	if err != nil {
		return m.fallbackMerge(results)
	}
	
	return merged
}

// MergeToString merges results to string using LLM
func (m *LLMMerger) MergeToString(results []*SubResult) string {
	if m.llmClient == nil || len(results) == 0 {
		return ""
	}
	
	// Build summary prompt
	prompt := m.buildSummaryPrompt(results)
	
	// Call LLM for summary
	ctx := context.Background()
	response, err := m.llmClient.Generate(ctx, prompt)
	if err != nil {
		// Fallback to simple concatenation
		return m.fallbackMergeToString(results)
	}
	
	return response
}

// buildMergePrompt builds the prompt for LLM-based merging
func (m *LLMMerger) buildMergePrompt(results []*SubResult) string {
	var sb strings.Builder
	
	sb.WriteString(m.prompt)
	sb.WriteString("\n\n## Results to Merge:\n\n")
	
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("### Result %d: %s\n", i+1, r.SubTaskName))
		sb.WriteString(fmt.Sprintf("- Agent: %s\n", r.AgentName))
		sb.WriteString(fmt.Sprintf("- Success: %v\n", r.Success))
		sb.WriteString(fmt.Sprintf("- Duration: %v\n", r.Duration))
		sb.WriteString("- Output:\n")
		for k, v := range r.Output {
			sb.WriteString(fmt.Sprintf("  - %s: %v\n", k, v))
		}
		sb.WriteString("\n")
	}
	
	sb.WriteString("\n## Instructions:\n")
	sb.WriteString("Merge these results into a single coherent output in JSON format.\n")
	sb.WriteString("Resolve any conflicts and synthesize the information.\n")
	
	return sb.String()
}

// buildSummaryPrompt builds the prompt for LLM-based summarization
func (m *LLMMerger) buildSummaryPrompt(results []*SubResult) string {
	var sb strings.Builder
	
	sb.WriteString("Summarize the following task execution results into a concise summary:\n\n")
	
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s (by %s): ", i+1, r.SubTaskName, r.AgentName))
		if r.Success {
			sb.WriteString("Success\n")
			for k, v := range r.Output {
				sb.WriteString(fmt.Sprintf("   - %s: %v\n", k, v))
			}
		} else {
			sb.WriteString(fmt.Sprintf("Failed: %v\n", r.Error))
		}
	}
	
	sb.WriteString("\nProvide a brief summary of what was accomplished.\n")
	
	return sb.String()
}

// parseMergeResponse parses the LLM response into a map
func (m *LLMMerger) parseMergeResponse(response string) (map[string]any, error) {
	// Try to extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	jsonStr := response[jsonStart : jsonEnd+1]
	
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return result, nil
}

// fallbackMerge provides fallback merging when LLM fails
func (m *LLMMerger) fallbackMerge(results []*SubResult) map[string]any {
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

// fallbackMergeToString provides fallback string merging
func (m *LLMMerger) fallbackMergeToString(results []*SubResult) string {
	var sb strings.Builder
	for _, r := range results {
		if r.Success {
			sb.WriteString(fmt.Sprintf("%s: completed successfully\n", r.SubTaskName))
		} else {
			sb.WriteString(fmt.Sprintf("%s: failed - %v\n", r.SubTaskName, r.Error))
		}
	}
	return sb.String()
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
