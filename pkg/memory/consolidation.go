package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// Consolidator handles memory consolidation from short-term to long-term
type Consolidator struct {
	memory *Memory
	llm    core.Client
	config *ConsolidationConfig
}

// NewConsolidator creates a new Consolidator
func NewConsolidator(memory *Memory, llm core.Client, config *ConsolidationConfig) *Consolidator {
	if config == nil {
		config = DefaultConsolidationConfig()
	}
	return &Consolidator{
		memory: memory,
		llm:    llm,
		config: config,
	}
}

// Consolidate performs memory consolidation for a session
func (c *Consolidator) Consolidate(ctx context.Context, sessionName string) (*ConsolidationResult, error) {
	result := &ConsolidationResult{
		SessionName:    sessionName,
		ConsolidatedAt: time.Now(),
	}

	// Get short-term memory items
	items, err := c.memory.ShortTerms().List(ctx, sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get short-term memory: %w", err)
	}

	if len(items) == 0 {
		return result, nil
	}

	// Filter important items
	importantItems := c.filterImportantItems(items)

	// Classify items
	classified := c.classifyItems(ctx, importantItems)

	// Store to long-term memory
	for category, categoryItems := range classified {
		docPath := c.generateDocumentPath(sessionName, category)
		content := c.buildDocumentContent(categoryItems)

		// Store to GraphRAG
		if c.memory.graphRAG != nil {
			err := c.memory.graphRAG.IndexText(ctx, content, map[string]any{
				"session":         sessionName,
				"category":        string(category),
				"doc_path":        docPath,
				"consolidated_at": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
		}

		result.Categories = append(result.Categories, ConsolidatedCategory{
			Category: string(category),
			Count:    len(categoryItems),
			DocPath:  docPath,
		})
	}

	result.TotalItems = len(importantItems)
	result.Success = true
	return result, nil
}

// filterImportantItems filters items above importance threshold
func (c *Consolidator) filterImportantItems(items []*goreactcore.MemoryItemNode) []*goreactcore.MemoryItemNode {
	important := []*goreactcore.MemoryItemNode{}
	for _, item := range items {
		if item.Importance >= c.config.ImportanceThreshold {
			important = append(important, item)
		}
	}
	return important
}

// classifyItems classifies items into categories
func (c *Consolidator) classifyItems(ctx context.Context, items []*goreactcore.MemoryItemNode) map[CategoryClassifier][]*goreactcore.MemoryItemNode {
	classified := make(map[CategoryClassifier][]*goreactcore.MemoryItemNode)

	for _, item := range items {
		category := c.classifyItem(ctx, item)
		classified[category] = append(classified[category], item)
	}

	return classified
}

// classifyItem classifies a single item
func (c *Consolidator) classifyItem(ctx context.Context, item *goreactcore.MemoryItemNode) CategoryClassifier {
	// If LLM is available, use it for classification
	if c.llm != nil {
		return c.classifyWithLLM(ctx, item)
	}

	// Simple rule-based classification
	content := strings.ToLower(item.Content)

	if strings.Contains(content, "rule:") || strings.Contains(content, "must") || strings.Contains(content, "always") {
		return CategoryRule
	}

	if strings.Contains(content, "fact:") || strings.Contains(content, "is located") || strings.Contains(content, "was born") {
		return CategoryFact
	}

	if strings.Contains(content, "how to") || strings.Contains(content, "step") || strings.Contains(content, "procedure") {
		return CategoryTask
	}

	return CategoryKnowledge
}

// classifyWithLLM classifies content using LLM
func (c *Consolidator) classifyWithLLM(ctx context.Context, item *goreactcore.MemoryItemNode) CategoryClassifier {
	prompt := fmt.Sprintf(`Classify this content into one of these categories: rule, knowledge, chat, fact, task.

Content: %s

Respond with only the category name.`, item.Content)

	resp, err := c.llm.Chat(ctx, []core.Message{
		core.NewUserMessage(prompt),
	})
	if err != nil {
		return CategoryKnowledge
	}

	category := strings.TrimSpace(strings.ToLower(resp.Content))
	switch CategoryClassifier(category) {
	case CategoryRule, CategoryKnowledge, CategoryChat, CategoryFact, CategoryTask:
		return CategoryClassifier(category)
	default:
		return CategoryKnowledge
	}
}

// generateDocumentPath generates the document path for storage
func (c *Consolidator) generateDocumentPath(sessionName string, category CategoryClassifier) string {
	template := c.config.DocumentPathTemplate
	if template == "" {
		template = "memory/{{.Category}}/{{.Date}}.md"
	}

	// Simple template replacement
	path := strings.ReplaceAll(template, "{{.Category}}", string(category))
	path = strings.ReplaceAll(path, "{{.Date}}", time.Now().Format("2006-01-02"))
	path = strings.ReplaceAll(path, "{{.SessionName}}", sessionName)

	return path
}

// buildDocumentContent builds the document content from items
func (c *Consolidator) buildDocumentContent(items []*goreactcore.MemoryItemNode) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Memory Consolidation\n\n"))
	builder.WriteString(fmt.Sprintf("Consolidated at: %s\n\n", time.Now().Format(time.RFC3339)))
	builder.WriteString("---\n\n")

	for _, item := range items {
		builder.WriteString(fmt.Sprintf("- %s\n", item.Content))
		if item.Description != "" {
			builder.WriteString(fmt.Sprintf("  - %s\n", item.Description))
		}
	}

	return builder.String()
}

// ConsolidationResult represents the result of consolidation
type ConsolidationResult struct {
	SessionName    string                 `json:"session_name"`
	ConsolidatedAt time.Time              `json:"consolidated_at"`
	TotalItems     int                    `json:"total_items"`
	Categories     []ConsolidatedCategory `json:"categories"`
	Success        bool                   `json:"success"`
	Errors         []string               `json:"errors,omitempty"`
}

// ConsolidatedCategory represents a consolidated category
type ConsolidatedCategory struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
	DocPath  string `json:"doc_path"`
}

// ConsolidationService provides consolidation service methods
type ConsolidationService interface {
	Consolidate(ctx context.Context, sessionName string) (*ConsolidationResult, error)
	ConsolidateBatch(ctx context.Context, sessionNames []string) ([]*ConsolidationResult, error)
	GetConsolidationHistory(ctx context.Context, sessionName string) (*ConsolidationRecord, error)
}

// ConsolidationRecord represents a consolidation record
type ConsolidationRecord struct {
	SessionName    string    `json:"session_name"`
	ConsolidatedAt time.Time `json:"consolidated_at"`
	TotalItems     int       `json:"total_items"`
	Categories     []string  `json:"categories"`
	DocumentPaths  []string  `json:"document_paths"`
}

// ConsolidateBatch consolidates multiple sessions
func (c *Consolidator) ConsolidateBatch(ctx context.Context, sessionNames []string) ([]*ConsolidationResult, error) {
	results := make([]*ConsolidationResult, 0, len(sessionNames))
	for _, name := range sessionNames {
		result, err := c.Consolidate(ctx, name)
		if err != nil {
			results = append(results, &ConsolidationResult{
				SessionName: name,
				Success:     false,
				Errors:      []string{err.Error()},
			})
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

// GetConsolidationHistory gets consolidation history for a session
func (c *Consolidator) GetConsolidationHistory(ctx context.Context, sessionName string) (*ConsolidationRecord, error) {
	if c.memory == nil {
		return nil, fmt.Errorf("memory is not initialized")
	}

	// Query from GraphRAG for consolidation records
	if c.memory.graphRAG != nil {
		results, err := c.memory.graphRAG.QueryGraph(ctx, fmt.Sprintf(
			"MATCH (c:ConsolidationRecord {session_name: '%s'}) RETURN c ORDER BY c.consolidated_at DESC LIMIT 1",
			sessionName), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query consolidation history: %w", err)
		}

		if len(results) > 0 {
			// Parse result from map[string]any
			if recordData, ok := results[0]["c"].(map[string]any); ok {
				record := &ConsolidationRecord{}
				if sessionNameVal, ok := recordData["session_name"].(string); ok {
					record.SessionName = sessionNameVal
				}
				return record, nil
			}
		}
	}

	// Fallback: Query from file system (use default path)
	recordPath := filepath.Join("./documents", "consolidation", sessionName+".json")
	if _, err := os.Stat(recordPath); err == nil {
		data, err := os.ReadFile(recordPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read consolidation record: %w", err)
		}

		var record ConsolidationRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("failed to parse consolidation record: %w", err)
		}

		return &record, nil
	}

	return nil, fmt.Errorf("no consolidation history found for session: %s", sessionName)
}
