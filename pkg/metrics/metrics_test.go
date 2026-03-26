package metrics

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	c := NewCounter()

	t.Run("initial value is zero", func(t *testing.T) {
		if c.Get() != 0 {
			t.Errorf("Expected 0, got %d", c.Get())
		}
	})

	t.Run("increment", func(t *testing.T) {
		c.Increment()
		if c.Get() != 1 {
			t.Errorf("Expected 1, got %d", c.Get())
		}
	})

	t.Run("multiple increments", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			c.Increment()
		}
		if c.Get() != 6 {
			t.Errorf("Expected 6, got %d", c.Get())
		}
	})

	t.Run("reset", func(t *testing.T) {
		c.Reset()
		if c.Get() != 0 {
			t.Errorf("Expected 0 after reset, got %d", c.Get())
		}
	})

	t.Run("concurrent increments", func(t *testing.T) {
		c.Reset()
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.Increment()
			}()
		}
		wg.Wait()
		if c.Get() != 100 {
			t.Errorf("Expected 100, got %d", c.Get())
		}
	})
}

func TestTimer(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		timer := NewTimer()
		metrics := timer.GetMetrics()
		if metrics["count"].(int64) != 0 {
			t.Errorf("Expected count 0, got %v", metrics["count"])
		}
	})

	t.Run("single record", func(t *testing.T) {
		timer := NewTimer()
		timer.Record(100 * time.Millisecond)
		metrics := timer.GetMetrics()
		if metrics["count"].(int64) != 1 {
			t.Errorf("Expected count 1, got %v", metrics["count"])
		}
		if metrics["sum"].(time.Duration) != 100*time.Millisecond {
			t.Errorf("Expected sum 100ms, got %v", metrics["sum"])
		}
		if metrics["min"].(time.Duration) != 100*time.Millisecond {
			t.Errorf("Expected min 100ms, got %v", metrics["min"])
		}
		if metrics["max"].(time.Duration) != 100*time.Millisecond {
			t.Errorf("Expected max 100ms, got %v", metrics["max"])
		}
	})

	t.Run("multiple records calculates avg", func(t *testing.T) {
		timer := NewTimer()
		timer.Record(100 * time.Millisecond)
		timer.Record(200 * time.Millisecond)
		metrics := timer.GetMetrics()
		if metrics["avg"].(time.Duration) != 150*time.Millisecond {
			t.Errorf("Expected avg 150ms, got %v", metrics["avg"])
		}
		if metrics["max"].(time.Duration) != 200*time.Millisecond {
			t.Errorf("Expected max 200ms, got %v", metrics["max"])
		}
		if metrics["min"].(time.Duration) != 100*time.Millisecond {
			t.Errorf("Expected min 100ms, got %v", metrics["min"])
		}
	})

	t.Run("reset", func(t *testing.T) {
		timer := NewTimer()
		timer.Record(time.Second)
		timer.Reset()
		metrics := timer.GetMetrics()
		if metrics["count"].(int64) != 0 {
			t.Errorf("Expected count 0, got %v", metrics["count"])
		}
		if metrics["sum"].(time.Duration) != 0 {
			t.Errorf("Expected sum 0, got %v", metrics["sum"])
		}
	})
}

func TestTokenCounter(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		tc := NewTokenCounter()
		metrics := tc.GetMetrics()
		if metrics["total_tokens"].(int64) != 0 {
			t.Errorf("Expected 0 total tokens, got %v", metrics["total_tokens"])
		}
		if metrics["call_count"].(int64) != 0 {
			t.Errorf("Expected 0 call count, got %v", metrics["call_count"])
		}
	})

	t.Run("record tokens", func(t *testing.T) {
		tc := NewTokenCounter()
		tc.Record(100, 50, 150)
		metrics := tc.GetMetrics()
		if metrics["prompt_tokens"].(int64) != 100 {
			t.Errorf("Expected 100 prompt tokens, got %v", metrics["prompt_tokens"])
		}
		if metrics["completion_tokens"].(int64) != 50 {
			t.Errorf("Expected 50 completion tokens, got %v", metrics["completion_tokens"])
		}
		if metrics["total_tokens"].(int64) != 150 {
			t.Errorf("Expected 150 total tokens, got %v", metrics["total_tokens"])
		}
	})

	t.Run("multiple records", func(t *testing.T) {
		tc := NewTokenCounter()
		tc.Record(100, 50, 150)
		tc.Record(200, 100, 300)
		metrics := tc.GetMetrics()
		if metrics["total_tokens"].(int64) != 450 {
			t.Errorf("Expected 450 total tokens, got %v", metrics["total_tokens"])
		}
		if metrics["call_count"].(int64) != 2 {
			t.Errorf("Expected 2 calls, got %v", metrics["call_count"])
		}
		if metrics["avg_prompt_tokens"].(int64) != 150 {
			t.Errorf("Expected 150 avg prompt tokens, got %v", metrics["avg_prompt_tokens"])
		}
	})

	t.Run("reset", func(t *testing.T) {
		tc := NewTokenCounter()
		tc.Record(100, 50, 150)
		tc.Reset()
		metrics := tc.GetMetrics()
		if metrics["total_tokens"].(int64) != 0 {
			t.Errorf("Expected 0 after reset, got %v", metrics["total_tokens"])
		}
	})
}

func TestResourceUsageCounter(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		rc := NewResourceUsageCounter()
		metrics := rc.GetMetrics()
		if metrics["call_count"].(int64) != 0 {
			t.Errorf("Expected 0 calls, got %v", metrics["call_count"])
		}
	})

	t.Run("record usage", func(t *testing.T) {
		rc := NewResourceUsageCounter()
		rc.Record(50.0, 1024.0, 0.0, 0.0)
		metrics := rc.GetMetrics()
		if metrics["avg_cpu_percent"].(float64) != 50.0 {
			t.Errorf("Expected 50.0 avg cpu, got %v", metrics["avg_cpu_percent"])
		}
		if metrics["max_cpu_percent"].(float64) != 50.0 {
			t.Errorf("Expected 50.0 max cpu, got %v", metrics["max_cpu_percent"])
		}
	})

	t.Run("track max values", func(t *testing.T) {
		rc := NewResourceUsageCounter()
		rc.Record(30.0, 500.0, 0.0, 0.0)
		rc.Record(80.0, 2000.0, 0.0, 0.0)
		metrics := rc.GetMetrics()
		if metrics["max_cpu_percent"].(float64) != 80.0 {
			t.Errorf("Expected 80.0 max cpu, got %v", metrics["max_cpu_percent"])
		}
		if metrics["max_memory_mb"].(float64) != 2000.0 {
			t.Errorf("Expected 2000.0 max memory, got %v", metrics["max_memory_mb"])
		}
	})

	t.Run("reset", func(t *testing.T) {
		rc := NewResourceUsageCounter()
		rc.Record(50.0, 1024.0, 0.0, 0.0)
		rc.Reset()
		metrics := rc.GetMetrics()
		if metrics["call_count"].(int64) != 0 {
			t.Errorf("Expected 0 after reset, got %v", metrics["call_count"])
		}
	})
}

func TestDefaultMetrics(t *testing.T) {
	m := NewDefaultMetrics()

	t.Run("RecordLatency", func(t *testing.T) {
		m.RecordLatency("test_op", 100*time.Millisecond)
		metrics := m.GetMetrics()
		latencies := metrics["latencies"].(map[string]any)
		if latencies["test_op"] == nil {
			t.Error("Expected test_op latency to be recorded")
		}
	})

	t.Run("RecordError", func(t *testing.T) {
		m.RecordError("test_op", errors.New("test error"))
		metrics := m.GetMetrics()
		errors := metrics["errors"].(map[string]int64)
		if errors["test_op"] != 1 {
			t.Errorf("Expected 1 error, got %d", errors["test_op"])
		}
	})

	t.Run("RecordSuccess", func(t *testing.T) {
		m.RecordSuccess("test_op")
		metrics := m.GetMetrics()
		successes := metrics["successes"].(map[string]int64)
		if successes["test_op"] != 1 {
			t.Errorf("Expected 1 success, got %d", successes["test_op"])
		}
	})

	t.Run("RecordTokenUsage", func(t *testing.T) {
		m.RecordTokenUsage("test_op", 100, 50, 150)
		metrics := m.GetMetrics()
		tokenUsage := metrics["token_usage"].(map[string]any)
		if tokenUsage["test_op"] == nil {
			t.Error("Expected test_op token usage to be recorded")
		}
	})

	t.Run("RecordResourceUsage", func(t *testing.T) {
		m.RecordResourceUsage("test_op", 50.0, 1024.0, 0.0, 0.0)
		metrics := m.GetMetrics()
		resourceUsage := metrics["resource_usage"].(map[string]any)
		if resourceUsage["test_op"] == nil {
			t.Error("Expected test_op resource usage to be recorded")
		}
	})

	t.Run("Reset clears values but not map keys", func(t *testing.T) {
		m := NewDefaultMetrics()
		m.RecordLatency("test_op", 100*time.Millisecond)
		m.RecordError("test_op", errors.New("test error"))
		m.RecordSuccess("test_op")
		m.RecordTokenUsage("test_op", 100, 50, 150)
		m.RecordResourceUsage("test_op", 50.0, 1024.0, 0.0, 0.0)

		m.Reset()

		metrics := m.GetMetrics()
		latencies := metrics["latencies"].(map[string]any)
		timerMetrics := latencies["test_op"].(map[string]any)
		if timerMetrics["count"].(int64) != 0 {
			t.Errorf("Expected timer count 0 after reset, got %v", timerMetrics["count"])
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := m.Close()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestDefaultMetrics_Interface(t *testing.T) {
	m := NewDefaultMetrics()
	var _ Metrics = m
}

func TestResourceMonitor(t *testing.T) {
	monitor := NewResourceMonitor()

	t.Run("Snapshot returns valid data", func(t *testing.T) {
		snapshot := monitor.Snapshot()
		if snapshot == nil {
			t.Fatal("Expected non-nil snapshot")
		}
		if snapshot.NumGoroutines < 1 {
			t.Errorf("Expected at least 1 goroutine, got %d", snapshot.NumGoroutines)
		}
		if snapshot.NumCPU < 1 {
			t.Errorf("Expected at least 1 CPU, got %d", snapshot.NumCPU)
		}
	})

	t.Run("Delta between snapshots", func(t *testing.T) {
		before := monitor.Snapshot()
		time.Sleep(10 * time.Millisecond)
		after := monitor.Snapshot()

		delta := after.Delta(before)
		if delta.Duration < 0 {
			t.Errorf("Expected positive duration, got %v", delta.Duration)
		}
		if delta.NumGoroutines < -1000 || delta.NumGoroutines > 1000 {
			t.Errorf("Goroutine count delta seems invalid: %d", delta.NumGoroutines)
		}
	})
}