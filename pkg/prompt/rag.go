package prompt

import (
	"context"
	"strings"
)

// RAGInjector injects RAG context into prompts
type RAGInjector struct {
	contextManager *ContextManager
}

// NewRAGInjector creates a new RAGInjector
func NewRAGInjector(contextManager *ContextManager) *RAGInjector {
	return &RAGInjector{
		contextManager: contextManager,
	}
}

// Inject injects RAG context into the prompt
func (i *RAGInjector) Inject(prompt *Prompt, context *RAGContext, strategy InjectionStrategy) {
	if context == nil || len(context.Documents) == 0 {
		return
	}

	ragSection := i.FormatContext(context)

	section := &PromptSection{
		Type:     "rag",
		Content:  ragSection,
		Priority: 50, // Middle priority
	}

	switch strategy {
	case InjectionPrefix:
		// Insert at beginning
		prompt.Sections = append([]*PromptSection{section}, prompt.Sections...)
	case InjectionInfix:
		// Insert before examples
		i.insertBeforeExamples(prompt, section)
	case InjectionSuffix:
		// Insert before question
		i.insertBeforeQuestion(prompt, section)
	case InjectionDynamic:
		// Smart injection based on context
		i.smartInject(prompt, context, section)
	}

	prompt.RAGContext = context
}

// FormatContext formats RAG context for injection
func (i *RAGInjector) FormatContext(context *RAGContext) string {
	var sb strings.Builder

	sb.WriteString("# RAG Context\n\n")

	// Documents section
	if len(context.Documents) > 0 {
		sb.WriteString("## 相关文档\n")
		for j, doc := range context.Documents {
			sb.WriteString(formatDocument(doc, j+1))
		}
		sb.WriteString("\n")
	}

	// Graph context section
	if context.GraphContext != nil && len(context.GraphContext.Nodes) > 0 {
		sb.WriteString("## 相关实体关系\n")
		for _, edge := range context.GraphContext.Edges {
			sb.WriteString("- " + edge.Source + " --" + edge.Relation + "--> " + edge.Target + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatDocument formats a single document
func formatDocument(doc *Document, index int) string {
	return formatDocumentEntry(index, doc.Content, doc.Source, doc.Score)
}

// formatDocumentEntry creates a document entry string
func formatDocumentEntry(index int, content, source string, score float64) string {
	result := "[" + itoa(index) + "] " + content
	if source != "" {
		result += " (来源: " + source
		if score > 0 {
			result += ", 相关度: " + ftoa(score, 2)
		}
		result += ")"
	}
	return result + "\n"
}

// insertBeforeExamples inserts section before examples
func (i *RAGInjector) insertBeforeExamples(prompt *Prompt, section *PromptSection) {
	idx := 0
	for j, s := range prompt.Sections {
		if s.Type == "examples" {
			idx = j
			break
		}
		idx = j + 1
	}

	// Insert at found position
	prompt.Sections = append(prompt.Sections[:idx], append([]*PromptSection{section}, prompt.Sections[idx:]...)...)
}

// insertBeforeQuestion inserts section before question
func (i *RAGInjector) insertBeforeQuestion(prompt *Prompt, section *PromptSection) {
	idx := len(prompt.Sections)
	for j, s := range prompt.Sections {
		if s.Type == "question" {
			idx = j
			break
		}
	}

	prompt.Sections = append(prompt.Sections[:idx], append([]*PromptSection{section}, prompt.Sections[idx:]...)...)
}

// smartInject intelligently chooses injection position
func (i *RAGInjector) smartInject(prompt *Prompt, context *RAGContext, section *PromptSection) {
	// Default to suffix for now
	// In production, would analyze context size, question type, etc.
	i.insertBeforeQuestion(prompt, section)
}

// OptimizeContext optimizes RAG context for token limits
func (i *RAGInjector) OptimizeContext(context *RAGContext, maxTokens int) *RAGContext {
	if i.contextManager == nil {
		return context
	}

	// Filter by relevance
	context = i.filterByRelevance(context, 0.5)

	// Deduplicate
	context = i.deduplicate(context)

	// Truncate by tokens
	context = i.truncateByTokens(context, maxTokens)

	return context
}

// filterByRelevance filters documents by relevance score
func (i *RAGInjector) filterByRelevance(context *RAGContext, minScore float64) *RAGContext {
	filtered := make([]*Document, 0)
	for _, doc := range context.Documents {
		if doc.Score >= minScore {
			filtered = append(filtered, doc)
		}
	}
	context.Documents = filtered
	return context
}

// deduplicate removes duplicate documents
func (i *RAGInjector) deduplicate(context *RAGContext) *RAGContext {
	seen := make(map[string]bool)
	filtered := make([]*Document, 0)

	for _, doc := range context.Documents {
		if !seen[doc.ID] {
			seen[doc.ID] = true
			filtered = append(filtered, doc)
		}
	}
	context.Documents = filtered
	return context
}

// truncateByTokens truncates documents to fit token limit
func (i *RAGInjector) truncateByTokens(context *RAGContext, maxTokens int) *RAGContext {
	// Simple character-based estimation
	maxChars := maxTokens * 4 // Rough estimate

	totalChars := 0
	filtered := make([]*Document, 0)

	for _, doc := range context.Documents {
		docLen := len(doc.Content)
		if totalChars+docLen <= maxChars {
			filtered = append(filtered, doc)
			totalChars += docLen
		}
	}
	context.Documents = filtered
	return context
}

// RetrieveFromMemory retrieves RAG context from memory
func (i *RAGInjector) RetrieveFromMemory(ctx context.Context, query string, topK int) (*RAGContext, error) {
	// This would be implemented with actual memory access
	// For now, return empty context
	return &RAGContext{
		Query:     query,
		Mode:      RAGModeHybrid,
		Documents: []*Document{},
	}, nil
}

// MergeContexts merges multiple RAG contexts
func (i *RAGInjector) MergeContexts(contexts ...*RAGContext) *RAGContext {
	if len(contexts) == 0 {
		return nil
	}

	merged := &RAGContext{
		Mode:      RAGModeHybrid,
		Documents: make([]*Document, 0),
	}

	seen := make(map[string]bool)
	for _, ctx := range contexts {
		for _, doc := range ctx.Documents {
			if !seen[doc.ID] {
				seen[doc.ID] = true
				merged.Documents = append(merged.Documents, doc)
			}
		}
	}

	return merged
}

// Helper functions
func itoa(i int) string {
	// Simple integer to string conversion
	if i == 0 {
		return "0"
	}

	var result []byte
	negative := false
	if i < 0 {
		negative = true
		i = -i
	}

	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}

func ftoa(f float64, precision int) string {
	// Simple float to string conversion
	intPart := int(f)
	fracPart := f - float64(intPart)

	result := itoa(intPart)

	if precision > 0 {
		result += "."
		for j := 0; j < precision; j++ {
			fracPart *= 10
			result += itoa(int(fracPart) % 10)
		}
	}

	return result
}
