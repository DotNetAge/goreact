package reactor

import (
	"encoding/json"
	"fmt"
	"strings"

	goreactcore "github.com/DotNetAge/goreact/pkg/core"
)

// ResultProcessor processes action results into string content
type ResultProcessor interface {
	Process(result any) (string, error)
	CanHandle(result any) bool
}

// StringProcessor handles string results
type StringProcessor struct {
	maxLen int
}

// NewStringProcessor creates a new StringProcessor
func NewStringProcessor(maxLen int) *StringProcessor {
	if maxLen <= 0 {
		maxLen = 1048576 // 1MB default
	}
	return &StringProcessor{maxLen: maxLen}
}

// Process processes a string result
func (p *StringProcessor) Process(result any) (string, error) {
	s, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("result is not a string")
	}
	return p.truncate(s), nil
}

// CanHandle checks if the processor can handle the result
func (p *StringProcessor) CanHandle(result any) bool {
	_, ok := result.(string)
	return ok
}

// truncate truncates a string if it exceeds max length
func (p *StringProcessor) truncate(s string) string {
	if len(s) <= p.maxLen {
		return s
	}
	return s[:p.maxLen] + "...[truncated]"
}

// StructProcessor handles struct/map results
type StructProcessor struct {
	maxLen int
}

// NewStructProcessor creates a new StructProcessor
func NewStructProcessor(maxLen int) *StructProcessor {
	if maxLen <= 0 {
		maxLen = 1048576
	}
	return &StructProcessor{maxLen: maxLen}
}

// Process processes a struct/map result
func (p *StructProcessor) Process(result any) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize result: %w", err)
	}
	s := string(data)
	if len(s) > p.maxLen {
		s = s[:p.maxLen] + "...[truncated]"
	}
	return s, nil
}

// CanHandle checks if the processor can handle the result
func (p *StructProcessor) CanHandle(result any) bool {
	switch result.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

// ArrayProcessor handles array results
type ArrayProcessor struct {
	maxLen int
}

// NewArrayProcessor creates a new ArrayProcessor
func NewArrayProcessor(maxLen int) *ArrayProcessor {
	if maxLen <= 0 {
		maxLen = 1048576
	}
	return &ArrayProcessor{maxLen: maxLen}
}

// Process processes an array result
func (p *ArrayProcessor) Process(result any) (string, error) {
	arr, ok := result.([]any)
	if !ok {
		return "", fmt.Errorf("result is not an array")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Array[%d]:\n", len(arr)))
	for i, item := range arr {
		itemStr := fmt.Sprintf("%v", item)
		if len(itemStr) > 100 {
			itemStr = itemStr[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", i, itemStr))
	}

	s := sb.String()
	if len(s) > p.maxLen {
		s = s[:p.maxLen] + "...[truncated]"
	}
	return s, nil
}

// CanHandle checks if the processor can handle the result
func (p *ArrayProcessor) CanHandle(result any) bool {
	_, ok := result.([]any)
	return ok
}

// ErrorProcessor handles error results
type ErrorProcessor struct{}

// NewErrorProcessor creates a new ErrorProcessor
func NewErrorProcessor() *ErrorProcessor {
	return &ErrorProcessor{}
}

// Process processes an error result
func (p *ErrorProcessor) Process(result any) (string, error) {
	err, ok := result.(error)
	if !ok {
		return "", fmt.Errorf("result is not an error")
	}
	return fmt.Sprintf("Error: %s", err.Error()), nil
}

// CanHandle checks if the processor can handle the result
func (p *ErrorProcessor) CanHandle(result any) bool {
	_, ok := result.(error)
	return ok
}

// ResultProcessorChain chains multiple processors
type ResultProcessorChain struct {
	processors []ResultProcessor
}

// NewResultProcessorChain creates a new processor chain
func NewResultProcessorChain() *ResultProcessorChain {
	return &ResultProcessorChain{
		processors: []ResultProcessor{
			NewErrorProcessor(),
			NewStringProcessor(0),
			NewArrayProcessor(0),
			NewStructProcessor(0),
		},
	}
}

// Process processes a result using the first matching processor
func (c *ResultProcessorChain) Process(result any) (string, error) {
	for _, p := range c.processors {
		if p.CanHandle(result) {
			return p.Process(result)
		}
	}
	// Fallback to string conversion
	return fmt.Sprintf("%v", result), nil
}

// AddProcessor adds a processor to the chain
func (c *ResultProcessorChain) AddProcessor(p ResultProcessor) {
	c.processors = append(c.processors, p)
}
