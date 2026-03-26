package compression

import (
	"testing"
)

type mockCounter struct {
	counts map[string]int
}

func newMockCounter() *mockCounter {
	return &mockCounter{counts: make(map[string]int)}
}

func (m *mockCounter) Count(text string) int {
	if m.counts == nil {
		m.counts = make(map[string]int)
	}
	if _, ok := m.counts[text]; !ok {
		m.counts[text] = len(text) / 4
	}
	return m.counts[text]
}

func (m *mockCounter) SetCount(text string, count int) {
	if m.counts == nil {
		m.counts = make(map[string]int)
	}
	m.counts[text] = count
}

func TestTruncateStrategy_Compress(t *testing.T) {
	counter := newMockCounter()
	counter.SetCount("short", 10)
	counter.SetCount("long message", 100)

	t.Run("empty turns", func(t *testing.T) {
		s := NewTruncateStrategy()
		result, err := s.Compress([]Turn{}, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected 0, got %d", len(result))
		}
	})

	t.Run("single turn", func(t *testing.T) {
		s := NewTruncateStrategy()
		result, err := s.Compress([]Turn{{Role: "user", Content: "test"}}, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("Expected 1, got %d", len(result))
		}
	})

	t.Run("under limit", func(t *testing.T) {
		s := NewTruncateStrategy()
		turns := []Turn{
			{Role: "user", Content: "short"},
		}
		result, err := s.Compress(turns, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("Expected 1, got %d", len(result))
		}
	})

	t.Run("over limit triggers truncation", func(t *testing.T) {
		s := NewTruncateStrategy()
		turns := []Turn{
			{Role: "user", Content: "first"},
			{Role: "user", Content: "second"},
			{Role: "user", Content: "third"},
			{Role: "user", Content: "fourth"},
		}
		counter.SetCount("first", 50)
		counter.SetCount("second", 50)
		counter.SetCount("third", 50)
		counter.SetCount("fourth", 50)

		result, err := s.Compress(turns, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) >= len(turns) {
			t.Errorf("Expected truncation, got %d turns", len(result))
		}
	})
}

func TestTruncateStrategy_countTurns(t *testing.T) {
	counter := newMockCounter()
	counter.SetCount("hello", 10)

	s := &TruncateStrategy{}
	total := s.countTurns([]Turn{{Content: "hello"}}, counter)
	if total != 10 {
		t.Errorf("Expected 10, got %d", total)
	}
}

func TestSlidingWindowStrategy_Compress(t *testing.T) {
	counter := newMockCounter()

	t.Run("window larger than turns", func(t *testing.T) {
		s := NewSlidingWindowStrategy(10)
		turns := []Turn{
			{Role: "user", Content: "first"},
			{Role: "user", Content: "second"},
		}
		result, err := s.Compress(turns, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2, got %d", len(result))
		}
	})

	t.Run("window smaller than turns", func(t *testing.T) {
		s := NewSlidingWindowStrategy(2)
		turns := []Turn{
			{Role: "user", Content: "first"},
			{Role: "user", Content: "second"},
			{Role: "user", Content: "third"},
		}
		result, err := s.Compress(turns, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2, got %d", len(result))
		}
		if result[0].Content != "second" {
			t.Errorf("Expected 'second', got %q", result[0].Content)
		}
		if result[1].Content != "third" {
			t.Errorf("Expected 'third', got %q", result[1].Content)
		}
	})

	t.Run("zero window size defaults to 10", func(t *testing.T) {
		s := NewSlidingWindowStrategy(0)
		if s.WindowSize != 10 {
			t.Errorf("Expected 10, got %d", s.WindowSize)
		}
	})

	t.Run("negative window size defaults to 10", func(t *testing.T) {
		s := NewSlidingWindowStrategy(-5)
		if s.WindowSize != 10 {
			t.Errorf("Expected 10, got %d", s.WindowSize)
		}
	})
}

func TestPriorityStrategy_Compress(t *testing.T) {
	counter := newMockCounter()
	counter.SetCount("first", 20)
	counter.SetCount("second", 20)
	counter.SetCount("third", 20)

	t.Run("empty turns", func(t *testing.T) {
		s := NewPriorityStrategy(nil)
		result, err := s.Compress([]Turn{}, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected 0, got %d", len(result))
		}
	})

	t.Run("under limit", func(t *testing.T) {
		s := NewPriorityStrategy(nil)
		turns := []Turn{
			{Role: "user", Content: "first"},
		}
		result, err := s.Compress(turns, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("Expected 1, got %d", len(result))
		}
	})

	t.Run("keeps system message first", func(t *testing.T) {
		s := NewPriorityStrategy(nil)
		turns := []Turn{
			{Role: "system", Content: "system msg"},
			{Role: "user", Content: "first"},
		}
		result, err := s.Compress(turns, 10, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		foundSystem := false
		for _, turn := range result {
			if turn.Role == "system" {
				foundSystem = true
				break
			}
		}
		if !foundSystem {
			t.Error("Expected system message to be kept")
		}
	})

	t.Run("keeps recent messages", func(t *testing.T) {
		s := NewPriorityStrategy(nil)
		turns := []Turn{
			{Role: "user", Content: "first"},
			{Role: "user", Content: "second"},
			{Role: "user", Content: "third"},
		}
		result, err := s.Compress(turns, 10, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		foundThird := false
		for _, turn := range result {
			if turn.Content == "third" {
				foundThird = true
				break
			}
		}
		if !foundThird {
			t.Error("Expected recent message to be kept")
		}
	})
}

func TestPriorityStrategy_countTurns(t *testing.T) {
	counter := newMockCounter()
	counter.SetCount("hello", 10)

	s := &PriorityStrategy{}
	total := s.countTurns([]Turn{{Content: "hello"}}, counter)
	if total != 10 {
		t.Errorf("Expected 10, got %d", total)
	}
}

func TestHybridStrategy_Compress(t *testing.T) {
	counter := newMockCounter()
	counter.SetCount("first", 50)
	counter.SetCount("second", 50)
	counter.SetCount("third", 50)

	t.Run("single strategy", func(t *testing.T) {
		s := NewHybridStrategy(NewSlidingWindowStrategy(2))
		turns := []Turn{
			{Role: "user", Content: "first"},
			{Role: "user", Content: "second"},
			{Role: "user", Content: "third"},
		}
		result, err := s.Compress(turns, 100, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2, got %d", len(result))
		}
	})

	t.Run("multiple strategies", func(t *testing.T) {
		s := NewHybridStrategy(
			NewTruncateStrategy(),
			NewSlidingWindowStrategy(1),
		)
		turns := []Turn{
			{Role: "user", Content: "first"},
			{Role: "user", Content: "second"},
		}
		result, err := s.Compress(turns, 10, counter)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) > 2 {
			t.Errorf("Expected at most 2, got %d", len(result))
		}
	})
}

func TestHybridStrategy_countTurns(t *testing.T) {
	counter := newMockCounter()
	counter.SetCount("hello", 10)

	s := &HybridStrategy{}
	total := s.countTurns([]Turn{{Content: "hello"}}, counter)
	if total != 10 {
		t.Errorf("Expected 10, got %d", total)
	}
}