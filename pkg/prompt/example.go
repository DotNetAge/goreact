package prompt

import (
	"strings"
)

// ExampleSelector selects few-shot examples
type ExampleSelector struct {
	store       *ExampleStore
	maxExamples int
}

// ExampleStore stores few-shot examples
type ExampleStore struct {
	examples []*Example
}

// NewExampleSelector creates a new ExampleSelector
func NewExampleSelector(maxExamples int) *ExampleSelector {
	return &ExampleSelector{
		store:       NewExampleStore(),
		maxExamples: maxExamples,
	}
}

// NewExampleStore creates a new ExampleStore
func NewExampleStore() *ExampleStore {
	s := &ExampleStore{
		examples: make([]*Example, 0),
	}

	// Initialize with default examples
	s.examples = append(s.examples, DefaultExamples()...)

	return s
}

// Add adds an example to the store
func (s *ExampleStore) Add(example *Example) {
	s.examples = append(s.examples, example)
}

// GetAll returns all examples
func (s *ExampleStore) GetAll() []*Example {
	return s.examples
}

// Select selects examples based on the query and options
func (s *ExampleSelector) Select(query string, opts SelectOptions) []*Example {
	// Get candidates from store
	candidates := s.store.GetAll()

	// Filter by tags if specified
	if len(opts.Tags) > 0 {
		candidates = s.filterByTags(candidates, opts.Tags)
	}

	// Filter by difficulty if specified
	if opts.Difficulty > 0 {
		candidates = s.filterByDifficulty(candidates, opts.Difficulty)
	}

	// Ensure diversity
	candidates = s.ensureDiversity(candidates)

	// Limit to max examples
	if len(candidates) > s.maxExamples {
		candidates = candidates[:s.maxExamples]
	}

	return candidates
}

// SelectOptions defines options for example selection
type SelectOptions struct {
	Tags       []string
	Difficulty int
	QueryType  QuestionType
}

// filterByTags filters examples by tags
func (s *ExampleSelector) filterByTags(examples []*Example, tags []string) []*Example {
	result := make([]*Example, 0)
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	for _, ex := range examples {
		for _, exTag := range ex.Tags {
			if tagSet[exTag] {
				result = append(result, ex)
				break
			}
		}
	}
	return result
}

// filterByDifficulty filters examples by difficulty
func (s *ExampleSelector) filterByDifficulty(examples []*Example, maxDifficulty int) []*Example {
	result := make([]*Example, 0)
	for _, ex := range examples {
		if ex.Difficulty <= maxDifficulty {
			result = append(result, ex)
		}
	}
	return result
}

// ensureDiversity ensures examples cover different scenarios
func (s *ExampleSelector) ensureDiversity(examples []*Example) []*Example {
	result := make([]*Example, 0)
	seen := make(map[string]bool)

	for _, ex := range examples {
		// Use tags as diversity key
		key := strings.Join(ex.Tags, ",")
		if !seen[key] {
			result = append(result, ex)
			seen[key] = true
		}
	}

	return result
}

// FormatExamples formats examples for prompt injection
func FormatExamples(examples []*Example) string {
	if len(examples) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Few-shot Examples\n\n")

	for i, ex := range examples {
		sb.WriteString(formatExample(ex, i+1))
	}

	return sb.String()
}

// formatExample formats a single example
func formatExample(ex *Example, index int) string {
	var sb strings.Builder

	sb.WriteString("## Example ")
	sb.WriteString(itoa(index))
	sb.WriteString("\n")
	sb.WriteString("Question: ")
	sb.WriteString(ex.Question)
	sb.WriteString("\n")

	// Write T-A-O sequence
	maxLen := len(ex.Thoughts)
	if len(ex.Actions) > maxLen {
		maxLen = len(ex.Actions)
	}
	if len(ex.Observations) > maxLen {
		maxLen = len(ex.Observations)
	}
	for j := 0; j < maxLen; j++ {
		if j < len(ex.Thoughts) {
			sb.WriteString("Thought: ")
			sb.WriteString(ex.Thoughts[j])
			sb.WriteString("\n")
		}
		if j < len(ex.Actions) {
			sb.WriteString("Action: ")
			sb.WriteString(ex.Actions[j])
			sb.WriteString("\n")
		}
		if j < len(ex.Observations) {
			sb.WriteString("Observation: ")
			sb.WriteString(ex.Observations[j])
			sb.WriteString("\n")
		}
	}

	if ex.FinalAnswer != "" {
		sb.WriteString("Final Answer: ")
		sb.WriteString(ex.FinalAnswer)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ExampleManager manages examples in memory
type ExampleManager struct {
	store *ExampleStore
}

// NewExampleManager creates a new ExampleManager
func NewExampleManager() *ExampleManager {
	return &ExampleManager{
		store: NewExampleStore(),
	}
}

// AddExample adds an example
func (m *ExampleManager) AddExample(example *Example) {
	m.store.Add(example)
}

// SearchExamples searches for similar examples
func (m *ExampleManager) SearchExamples(query string, k int) []*Example {
	// Simple implementation - would use vector search in production
	return m.store.GetAll()[:min(k, len(m.store.GetAll()))]
}

// GetExamplesByTags retrieves examples by tags
func (m *ExampleManager) GetExamplesByTags(tags []string) []*Example {
	selector := NewExampleSelector(10)
	return selector.Select("", SelectOptions{Tags: tags})
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
