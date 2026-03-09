package cost

import (
	"strings"
	"testing"
)

func TestTrackerBasic(t *testing.T) {
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.01,
		OutputTokenPrice: 0.03,
	})

	// 初始状态
	if tracker.TotalInputTokens() != 0 {
		t.Errorf("Expected 0 input tokens, got %d", tracker.TotalInputTokens())
	}
	if tracker.TotalOutputTokens() != 0 {
		t.Errorf("Expected 0 output tokens, got %d", tracker.TotalOutputTokens())
	}
	if tracker.TotalCost() != 0 {
		t.Errorf("Expected 0 cost, got %f", tracker.TotalCost())
	}

	// 记录 tokens
	tracker.RecordTokens(1000, 500)

	if tracker.TotalInputTokens() != 1000 {
		t.Errorf("Expected 1000 input tokens, got %d", tracker.TotalInputTokens())
	}
	if tracker.TotalOutputTokens() != 500 {
		t.Errorf("Expected 500 output tokens, got %d", tracker.TotalOutputTokens())
	}

	// 计算成本：1000 * 0.01/1000 + 500 * 0.03/1000 = 0.01 + 0.015 = 0.025
	expectedCost := 0.025
	if tracker.TotalCost() != expectedCost {
		t.Errorf("Expected cost %f, got %f", expectedCost, tracker.TotalCost())
	}
}

func TestTrackerMultipleRecords(t *testing.T) {
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.01,
		OutputTokenPrice: 0.03,
	})

	// 记录多次
	tracker.RecordTokens(500, 200)
	tracker.RecordTokens(800, 300)
	tracker.RecordTokens(600, 250)

	expectedInput := 500 + 800 + 600
	expectedOutput := 200 + 300 + 250

	if tracker.TotalInputTokens() != expectedInput {
		t.Errorf("Expected %d input tokens, got %d", expectedInput, tracker.TotalInputTokens())
	}
	if tracker.TotalOutputTokens() != expectedOutput {
		t.Errorf("Expected %d output tokens, got %d", expectedOutput, tracker.TotalOutputTokens())
	}

	// 计算成本：1900 * 0.01/1000 + 750 * 0.03/1000 = 0.019 + 0.0225 = 0.0415
	expectedCost := 0.0415
	actualCost := tracker.TotalCost()
	if actualCost < expectedCost-0.0001 || actualCost > expectedCost+0.0001 {
		t.Errorf("Expected cost ~%f, got %f", expectedCost, actualCost)
	}
}

func TestTrackerExceedsLimit(t *testing.T) {
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.01,
		OutputTokenPrice: 0.03,
	})

	tracker.RecordTokens(1000, 500) // 成本 = 0.025

	// 未超过限制
	if tracker.ExceedsLimit(0.03) {
		t.Errorf("Expected not to exceed limit 0.03, but got true")
	}

	// 刚好等于限制
	if tracker.ExceedsLimit(0.025) {
		t.Errorf("Expected not to exceed limit 0.025 (equal), but got true")
	}

	// 超过限制
	if !tracker.ExceedsLimit(0.02) {
		t.Errorf("Expected to exceed limit 0.02, but got false")
	}
}

func TestTrackerReport(t *testing.T) {
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.01,
		OutputTokenPrice: 0.03,
	})

	tracker.RecordTokens(1900, 750)

	report := tracker.Report()

	// 检查报告包含关键信息
	expectedStrings := []string{
		"Cost Report",
		"1900",    // 输入 tokens
		"750",     // 输出 tokens
		"$0.0415", // 总成本
		"$0.0190", // 输入成本
		"$0.0225", // 输出成本
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(report, expected) {
			t.Errorf("Expected report to contain '%s', but it doesn't.\nReport:\n%s", expected, report)
		}
	}
}

func TestTrackerReset(t *testing.T) {
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.01,
		OutputTokenPrice: 0.03,
	})

	// 记录一些数据
	tracker.RecordTokens(1000, 500)

	if tracker.TotalInputTokens() == 0 {
		t.Errorf("Expected non-zero tokens before reset")
	}

	// 重置
	tracker.Reset()

	// 检查是否已重置
	if tracker.TotalInputTokens() != 0 {
		t.Errorf("Expected 0 input tokens after reset, got %d", tracker.TotalInputTokens())
	}
	if tracker.TotalOutputTokens() != 0 {
		t.Errorf("Expected 0 output tokens after reset, got %d", tracker.TotalOutputTokens())
	}
	if tracker.TotalCost() != 0 {
		t.Errorf("Expected 0 cost after reset, got %f", tracker.TotalCost())
	}
}

func TestTrackerDifferentPricing(t *testing.T) {
	// 测试不同的定价
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.005, // 更便宜
		OutputTokenPrice: 0.015,
	})

	tracker.RecordTokens(1000, 1000)

	// 成本：1000 * 0.005/1000 + 1000 * 0.015/1000 = 0.005 + 0.015 = 0.02
	expectedCost := 0.02
	if tracker.TotalCost() != expectedCost {
		t.Errorf("Expected cost %f, got %f", expectedCost, tracker.TotalCost())
	}
}

func TestTrackerZeroTokens(t *testing.T) {
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.01,
		OutputTokenPrice: 0.03,
	})

	// 记录 0 tokens
	tracker.RecordTokens(0, 0)

	if tracker.TotalCost() != 0 {
		t.Errorf("Expected 0 cost for 0 tokens, got %f", tracker.TotalCost())
	}
}

func TestTrackerConcurrency(t *testing.T) {
	tracker := NewTracker(Pricing{
		InputTokenPrice:  0.01,
		OutputTokenPrice: 0.03,
	})

	// 并发记录
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tracker.RecordTokens(10, 5)
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 检查总数
	expectedInput := 10 * 100 * 10 // 10,000
	expectedOutput := 10 * 100 * 5 // 5,000

	if tracker.TotalInputTokens() != expectedInput {
		t.Errorf("Expected %d input tokens, got %d", expectedInput, tracker.TotalInputTokens())
	}
	if tracker.TotalOutputTokens() != expectedOutput {
		t.Errorf("Expected %d output tokens, got %d", expectedOutput, tracker.TotalOutputTokens())
	}
}
