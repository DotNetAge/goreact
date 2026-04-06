package reactor

import (
	"regexp"
	"strings"

	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// InsightExtractor extracts insights from action results
type InsightExtractor interface {
	Extract(result any, context *goreactcore.ObservationContext) []string
}

// PatternExtractor extracts insights by pattern matching
type PatternExtractor struct {
	patterns []*InsightPattern
}

// InsightPattern represents a pattern for insight extraction
type InsightPattern struct {
	Name     string
	Pattern  *regexp.Regexp
	Type     goreactcommon.InsightType
	Priority int
}

// NewPatternExtractor creates a new PatternExtractor
func NewPatternExtractor() *PatternExtractor {
	return &PatternExtractor{
		patterns: getDefaultPatterns(),
	}
}

// Extract extracts insights by matching patterns
func (e *PatternExtractor) Extract(result any, context *goreactcore.ObservationContext) []string {
	insights := []string{}
	content := fmt.Sprintf("%v", result)

	for _, pattern := range e.patterns {
		if matches := pattern.Pattern.FindAllString(content, -1); len(matches) > 0 {
			for _, match := range matches {
				insight := fmt.Sprintf("[%s] %s: %s", pattern.Type, pattern.Name, match)
				insights = append(insights, insight)
			}
		}
	}

	return insights
}

// AddPattern adds a pattern to the extractor
func (e *PatternExtractor) AddPattern(pattern *InsightPattern) {
	e.patterns = append(e.patterns, pattern)
}

// getDefaultPatterns returns default insight patterns
func getDefaultPatterns() []*InsightPattern {
	return []*InsightPattern{
		{
			Name:     "Error Pattern",
			Pattern:  regexp.MustCompile(`(?i)(error|exception|failed|failure):\s*(.+)`),
			Type:     goreactcommon.InsightTypeAnomaly,
			Priority: 1,
		},
		{
			Name:     "File Found",
			Pattern:  regexp.MustCompile(`(?i)(found|created|deleted|modified)\s+(file|directory):\s*(.+)`),
			Type:     goreactcommon.InsightTypeKeyFinding,
			Priority: 2,
		},
		{
			Name:     "Timeout",
			Pattern:  regexp.MustCompile(`(?i)(timeout|timed out|deadline exceeded)`),
			Type:     goreactcommon.InsightTypeAnomaly,
			Priority: 1,
		},
		{
			Name:     "Success Pattern",
			Pattern:  regexp.MustCompile(`(?i)(successfully|completed|done)`),
			Type:     goreactcommon.InsightTypeKeyFinding,
			Priority: 3,
		},
	}
}

// KeywordExtractor extracts insights by keyword matching
type KeywordExtractor struct {
	keywords map[string]string
}

// NewKeywordExtractor creates a new KeywordExtractor
func NewKeywordExtractor() *KeywordExtractor {
	return &KeywordExtractor{
		keywords: getDefaultKeywords(),
	}
}

// Extract extracts insights by matching keywords
func (e *KeywordExtractor) Extract(result any, context *goreactcore.ObservationContext) []string {
	insights := []string{}
	content := strings.ToLower(fmt.Sprintf("%v", result))

	for keyword, category := range e.keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			insight := fmt.Sprintf("[keyword] Found '%s' - Category: %s", keyword, category)
			insights = append(insights, insight)
		}
	}

	return insights
}

// AddKeyword adds a keyword to the extractor
func (e *KeywordExtractor) AddKeyword(keyword, category string) {
	e.keywords[keyword] = category
}

// getDefaultKeywords returns default keywords
func getDefaultKeywords() map[string]string {
	return map[string]string{
		"error":     "error",
		"warning":   "warning",
		"success":   "status",
		"failed":    "error",
		"timeout":   "error",
		"created":   "action",
		"deleted":   "action",
		"modified":  "action",
		"found":     "status",
		"config":    "file",
		"yaml":      "file",
		"json":      "file",
		"test":      "testing",
		"passed":    "testing",
		"coverage":  "testing",
	}
}

// AnomalyDetector detects anomalies in results
type AnomalyDetector struct {
	threshold float64
}

// NewAnomalyDetector creates a new AnomalyDetector
func NewAnomalyDetector(threshold float64) *AnomalyDetector {
	if threshold <= 0 {
		threshold = 3.0 // Default: 3x deviation
	}
	return &AnomalyDetector{threshold: threshold}
}

// Extract extracts anomalies from the result
func (d *AnomalyDetector) Extract(result any, context *goreactcore.ObservationContext) []string {
	insights := []string{}
	content := fmt.Sprintf("%v", result)

	// Check for unusual patterns
	anomalyPatterns := []struct {
		name    string
		check   func(string) bool
		message string
	}{
		{
			name: "long_output",
			check: func(s string) bool {
				return len(s) > 10000
			},
			message: "Unusually long output detected",
		},
		{
			name: "many_errors",
			check: func(s string) bool {
				return strings.Count(strings.ToLower(s), "error") > 5
			},
			message: "Multiple errors detected in output",
		},
		{
			name: "repeated_pattern",
			check: func(s string) bool {
				// Check for repeated lines
				lines := strings.Split(s, "\n")
				if len(lines) < 3 {
					return false
				}
				count := make(map[string]int)
				for _, line := range lines {
					count[line]++
					if count[line] > 5 {
						return true
					}
				}
				return false
			},
			message: "Repeated pattern detected in output",
		},
	}

	for _, ap := range anomalyPatterns {
		if ap.check(content) {
			insights = append(insights, fmt.Sprintf("[anomaly] %s", ap.message))
		}
	}

	return insights
}

// CompositeExtractor combines multiple extractors
type CompositeExtractor struct {
	extractors []InsightExtractor
	maxInsights int
}

// NewCompositeExtractor creates a new CompositeExtractor
func NewCompositeExtractor(maxInsights int) *CompositeExtractor {
	if maxInsights <= 0 {
		maxInsights = 5
	}
	return &CompositeExtractor{
		extractors: []InsightExtractor{
			NewPatternExtractor(),
			NewKeywordExtractor(),
			NewAnomalyDetector(0),
		},
		maxInsights: maxInsights,
	}
}

// Extract extracts insights using all extractors
func (e *CompositeExtractor) Extract(result any, context *goreactcore.ObservationContext) []string {
	allInsights := []string{}

	for _, extractor := range e.extractors {
		insights := extractor.Extract(result, context)
		allInsights = append(allInsights, insights...)
	}

	// Deduplicate and limit
	seen := make(map[string]bool)
	unique := []string{}
	for _, insight := range allInsights {
		if !seen[insight] {
			seen[insight] = true
			unique = append(unique, insight)
			if len(unique) >= e.maxInsights {
				break
			}
		}
	}

	return unique
}

// AddExtractor adds an extractor to the composite
func (e *CompositeExtractor) AddExtractor(extractor InsightExtractor) {
	e.extractors = append(e.extractors, extractor)
}
