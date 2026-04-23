package core

import (
	"fmt"
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

var (
	globalEncoder     *tiktoken.Tiktoken
	globalEncoderOnce sync.Once
	globalEncoderErr  error
)

const defaultModel = "gpt-4o"

func getGlobalEncoder() (*tiktoken.Tiktoken, error) {
	globalEncoderOnce.Do(func() {
		globalEncoder, globalEncoderErr = tiktoken.EncodingForModel(defaultModel)
	})
	return globalEncoder, globalEncoderErr
}

func resetGlobalEncoderForTest() {
	globalEncoder = nil
	globalEncoderErr = nil
	globalEncoderOnce = sync.Once{}
}

// CountTokens returns the exact token count for text using the tiktoken BPE tokenizer.
// It uses the o200k_base encoding (GPT-4o) by default for maximum accuracy.
//
// Error handling strategy:
//   - Empty string: returns 0, nil (no error)
//   - Encoder not yet initialized: lazily initializes on first call (sync.Once)
//   - Initialization failure: falls back to EstimateTokens heuristic and returns nil error
//   - Special characters / CJK / emoji: handled correctly by tiktoken's BPE algorithm
//   - Very long text (>1MB): processed normally, tiktoken handles large inputs efficiently
//
// This function is safe for concurrent use.
func CountTokens(text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	enc, err := getGlobalEncoder()
	if err != nil {
		return estimateTokensFallback(text), nil
	}

	tokens := enc.Encode(text, nil, nil)
	if tokens == nil {
		return estimateTokensFallback(text), fmt.Errorf("tiktoken: Encode returned nil for text of length %d", len(text))
	}

	return len(tokens), nil
}

// CountTokensMust is like CountTokens but panics on encoder initialization failure.
// Use this when the caller expects tiktoken to always be available.
func CountTokensMust(text string) int {
	if text == "" {
		return 0
	}

	enc, err := getGlobalEncoder()
	if err != nil {
		panic(fmt.Sprintf("goreact: tiktoken encoder initialization failed: %v", err))
	}

	tokens := enc.Encode(text, nil, nil)
	if tokens == nil {
		panic(fmt.Sprintf("goreact: tiktoken Encode returned nil for text of length %d", len(text)))
	}

	return len(tokens)
}

// EstimateTokens provides a fast token count estimate using tiktoken with
// automatic fallback to character-class heuristics. It never returns an error,
// making it suitable as a drop-in replacement for the old heuristic-only version.
//
// When tiktoken is available (default), it returns exact BPE token counts.
// When tiktoken fails to initialize (e.g., BPE dictionary download failure),
// it falls back to the character-class-aware heuristic that considers:
//   - ASCII letters/digits: ~4 chars per token
//   - CJK ideographs: ~1.8 chars per token
//   - Other Unicode (emoji, symbols): ~3 chars per token
func EstimateTokens(text string) int {
	count, _ := CountTokens(text)
	return count
}

func estimateTokensFallback(text string) int {
	var asciiCount, cjkCount, otherCount int
	for _, r := range text {
		if r <= 127 {
			asciiCount++
		} else if isCJK(r) {
			cjkCount++
		} else {
			otherCount++
		}
	}
	tokens := float64(asciiCount)/4.0 + float64(cjkCount)/1.8 + float64(otherCount)/3.0
	return int(tokens + 0.5)
}

// TokenCounter is the interface for counting tokens in text.
// Implementations may use different strategies (BPE, heuristic, etc.).
type TokenCounter interface {
	// Count returns the number of tokens in the text.
	// Returns an error if counting fails.
	Count(text string) (int, error)

	// Model returns the model/encoding name used by this counter.
	Model() string
}

// TiktokenTokenCounter is a TokenCounter backed by tiktoken-go's BPE tokenizer.
type TiktokenTokenCounter struct {
	encoding *tiktoken.Tiktoken
	model    string
}

// NewTiktokenTokenCounter creates a new TiktokenTokenCounter for the given model.
// Supported models include "gpt-4o" (default, uses o200k_base), "gpt-4" (cl100k_base),
// "gpt-3.5-turbo" (cl100k_base), etc.
// Returns an error if the model's encoding cannot be initialized.
func NewTiktokenTokenCounter(model string) (*TiktokenTokenCounter, error) {
	if model == "" {
		model = defaultModel
	}
	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return nil, fmt.Errorf("tiktoken: failed to get encoding for model %q: %w", model, err)
	}
	return &TiktokenTokenCounter{encoding: enc, model: model}, nil
}

func (tc *TiktokenTokenCounter) Count(text string) (int, error) {
	if text == "" {
		return 0, nil
	}
	if tc.encoding == nil {
		return 0, fmt.Errorf("tiktoken: encoding is nil")
	}
	tokens := tc.encoding.Encode(text, nil, nil)
	if tokens == nil {
		return 0, fmt.Errorf("tiktoken: Encode returned nil")
	}
	return len(tokens), nil
}

func (tc *TiktokenTokenCounter) Model() string {
	return tc.model
}

// HeuristicTokenCounter is a fallback TokenCounter using character-class estimation.
type HeuristicTokenCounter struct{}

func NewHeuristicTokenCounter() *HeuristicTokenCounter { return &HeuristicTokenCounter{} }

func (hc *HeuristicTokenCounter) Count(text string) (int, error) {
	if text == "" {
		return 0, nil
	}
	return estimateTokensFallback(text), nil
}

func (hc *HeuristicTokenCounter) Model() string { return "heuristic-v2" }
