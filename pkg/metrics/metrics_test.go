package metrics

import (
	"testing"
	"time"
)

func TestMetrics_Collection(t *testing.T) {
	// 创建指标收集器
	metrics := NewDefaultMetrics()

	// 记录延迟
	metrics.RecordLatency("test.operation", 100*time.Millisecond)
	metrics.RecordLatency("test.operation", 200*time.Millisecond)
	metrics.RecordLatency("test.operation", 150*time.Millisecond)

	// 记录成功
	metrics.RecordSuccess("test.operation")
	metrics.RecordSuccess("test.operation")

	// 记录错误
	metrics.RecordError("test.operation", nil)

	// 获取指标
	allMetrics := metrics.GetMetrics()

	// 验证延迟指标
	if latencies, ok := allMetrics["latencies"].(map[string]interface{}); ok {
		if testMetrics, ok := latencies["test.operation"].(map[string]interface{}); ok {
			if count, ok := testMetrics["count"].(int64); !ok || count != 3 {
				t.Errorf("Expected latency count 3, got %v", testMetrics["count"])
			}
		}
	}

	// 验证成功指标
	if successes, ok := allMetrics["successes"].(map[string]int64); ok {
		if count, ok := successes["test.operation"]; !ok || count != 2 {
			t.Errorf("Expected success count 2, got %v", count)
		}
	}

	// 验证错误指标
	if errors, ok := allMetrics["errors"].(map[string]int64); ok {
		if count, ok := errors["test.operation"]; !ok || count != 1 {
			t.Errorf("Expected error count 1, got %v", count)
		}
	}

	// 测试重置
	metrics.Reset()
	resetMetrics := metrics.GetMetrics()

	if latencies, ok := resetMetrics["latencies"].(map[string]interface{}); ok {
		if testMetrics, ok := latencies["test.operation"].(map[string]interface{}); ok {
			if count, ok := testMetrics["count"].(int64); !ok || count != 0 {
				t.Errorf("Expected latency count 0 after reset, got %v", testMetrics["count"])
			}
		}
	}
}
