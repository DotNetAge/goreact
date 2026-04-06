package prompt

import (
	"strings"
)

// ContextManager manages prompt context and token limits
type ContextManager struct {
	maxTokens            int
	minRelevance         float64
	compressionThreshold float64
}

// NewContextManager creates a new ContextManager
func NewContextManager(maxTokens int) *ContextManager {
	return &ContextManager{
		maxTokens:            maxTokens,
		minRelevance:         0.5,
		compressionThreshold: 0.8,
	}
}

// Manage manages the prompt to fit within token limits
func (m *ContextManager) Manage(prompt *Prompt) error {
	currentTokens := m.CountTokens(prompt.String())

	if currentTokens <= m.maxTokens {
		return nil
	}

	targetTokens := int(float64(m.maxTokens) * m.compressionThreshold)

	for m.CountTokens(prompt.String()) > targetTokens {
		if !m.compressOne(prompt) {
			break
		}
	}

	return nil
}

// compressOne compresses one item from the prompt
func (m *ContextManager) compressOne(prompt *Prompt) bool {
	// Try to remove from RAG context first
	if prompt.RAGContext != nil && len(prompt.RAGContext.Documents) > 0 {
		prompt.RAGContext.Documents = prompt.RAGContext.Documents[:len(prompt.RAGContext.Documents)-1]
		return true
	}

	// Try to remove examples
	if len(prompt.Examples) > 1 {
		prompt.Examples = prompt.Examples[:len(prompt.Examples)-1]
		return true
	}

	// Try to remove sections with lowest priority
	if len(prompt.Sections) > 0 {
		// Find section with lowest priority
		minPriority := prompt.Sections[0].Priority
		minIdx := 0
		for i, section := range prompt.Sections {
			if section.Priority < minPriority && section.Type != "system" && section.Type != "question" {
				minPriority = section.Priority
				minIdx = i
			}
		}

		// Remove it if it's not essential
		if prompt.Sections[minIdx].Type != "system" && prompt.Sections[minIdx].Type != "question" {
			prompt.Sections = append(prompt.Sections[:minIdx], prompt.Sections[minIdx+1:]...)
			return true
		}
	}

	return false
}

// CountTokens estimates token count for a string
func (m *ContextManager) CountTokens(text string) int {
	// Simple estimation: ~4 characters per token
	return len(text) / 4
}

// Optimize optimizes the prompt for the token limit
func (m *ContextManager) Optimize(prompt *Prompt) *Prompt {
	// Filter by relevance
	if prompt.RAGContext != nil {
		m.filterByRelevance(prompt.RAGContext)
	}

	// Deduplicate
	m.deduplicate(prompt)

	// Prioritize sections
	m.prioritize(prompt)

	// Truncate by tokens
	m.truncateByTokens(prompt)

	return prompt
}

// filterByRelevance filters RAG context by relevance
func (m *ContextManager) filterByRelevance(context *RAGContext) {
	filtered := make([]*Document, 0)
	for _, doc := range context.Documents {
		if doc.Score >= m.minRelevance {
			filtered = append(filtered, doc)
		}
	}
	context.Documents = filtered
}

// deduplicate removes duplicate content
func (m *ContextManager) deduplicate(prompt *Prompt) {
	// Deduplicate examples
	if len(prompt.Examples) > 0 {
		seen := make(map[string]bool)
		filtered := make([]*Example, 0)
		for _, ex := range prompt.Examples {
			if !seen[ex.ID] {
				seen[ex.ID] = true
				filtered = append(filtered, ex)
			}
		}
		prompt.Examples = filtered
	}

	// Deduplicate RAG documents
	if prompt.RAGContext != nil && len(prompt.RAGContext.Documents) > 0 {
		seen := make(map[string]bool)
		filtered := make([]*Document, 0)
		for _, doc := range prompt.RAGContext.Documents {
			if !seen[doc.ID] {
				seen[doc.ID] = true
				filtered = append(filtered, doc)
			}
		}
		prompt.RAGContext.Documents = filtered
	}
}

// prioritize sorts sections by priority
func (m *ContextManager) prioritize(prompt *Prompt) {
	// Sort sections by priority (higher first)
	for i := 0; i < len(prompt.Sections); i++ {
		for j := i + 1; j < len(prompt.Sections); j++ {
			if prompt.Sections[j].Priority > prompt.Sections[i].Priority {
				prompt.Sections[i], prompt.Sections[j] = prompt.Sections[j], prompt.Sections[i]
			}
		}
	}
}

// truncateByTokens truncates content to fit token limit
func (m *ContextManager) truncateByTokens(prompt *Prompt) {
	for m.CountTokens(prompt.String()) > m.maxTokens {
		if !m.compressOne(prompt) {
			break
		}
	}
}

// SummarizeHistory summarizes history entries
func (m *ContextManager) SummarizeHistory(history []string) string {
	if len(history) <= 3 {
		return strings.Join(history, "\n")
	}

	recentHistory := history[len(history)-3:]
	oldHistory := history[:len(history)-3]

	// Simple summary - would use LLM in production
	summary := "历史摘要: " + itoa(len(oldHistory)) + " 步骤已执行\n\n最近操作:\n"
	summary += strings.Join(recentHistory, "\n")

	return summary
}

// SetMaxTokens sets the maximum token limit
func (m *ContextManager) SetMaxTokens(maxTokens int) {
	m.maxTokens = maxTokens
}

// SetMinRelevance sets the minimum relevance threshold
func (m *ContextManager) SetMinRelevance(minRelevance float64) {
	m.minRelevance = minRelevance
}
