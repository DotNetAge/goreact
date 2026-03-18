package counter

import (
	"testing"
)

func TestSimpleEstimator(t *testing.T) {
	counter := NewSimpleEstimator()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "short text",
			text:     "Hello",
			expected: 1, // 5 chars / 4 = 1
		},
		{
			name:     "medium text",
			text:     "Hello, World!",
			expected: 3, // 13 chars / 4 = 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.Count(tt.text)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestUniversalEstimator(t *testing.T) {
	tests := []struct {
		name     string
		language string
		text     string
		minToken int
		maxToken int
	}{
		{
			name:     "english text",
			language: "en",
			text:     "Hello, how are you?",
			minToken: 5,
			maxToken: 10,
		},
		{
			name:     "chinese text",
			language: "zh",
			text:     "你好，世界！",
			minToken: 5,
			maxToken: 10,
		},
		{
			name:     "mixed text",
			language: "mixed",
			text:     "Calculate 100 + 200. 计算结果是 300。",
			minToken: 15,
			maxToken: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewUniversalEstimator(tt.language)
			result := counter.Count(tt.text)
			if result < tt.minToken || result > tt.maxToken {
				t.Errorf("expected between %d and %d, got %d", tt.minToken, tt.maxToken, result)
			}
		})
	}
}

func TestCachedTokenCounter(t *testing.T) {
	baseCounter := NewSimpleEstimator()
	cached := NewCachedTokenCounter(baseCounter, 10)

	text := "Hello, World!"

	// First call - should cache
	result1 := cached.Count(text)

	// Second call - should hit cache
	result2 := cached.Count(text)

	if result1 != result2 {
		t.Errorf("cached results should be equal: %d != %d", result1, result2)
	}

	// Clear cache
	cached.Clear()

	// After clear, should still work
	result3 := cached.Count(text)
	if result3 != result1 {
		t.Errorf("result after clear should equal original: %d != %d", result3, result1)
	}
}

func BenchmarkSimpleEstimator(b *testing.B) {
	counter := NewSimpleEstimator()
	text := "This is a test sentence for benchmarking the token counter."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Count(text)
	}
}

func BenchmarkUniversalEstimator(b *testing.B) {
	counter := NewUniversalEstimator("mixed")
	text := "This is a test sentence. 这是一个测试句子。"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Count(text)
	}
}

func BenchmarkCachedTokenCounter(b *testing.B) {
	baseCounter := NewUniversalEstimator("mixed")
	cached := NewCachedTokenCounter(baseCounter, 1000)
	text := "This is a test sentence. 这是一个测试句子。"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached.Count(text)
	}
}
