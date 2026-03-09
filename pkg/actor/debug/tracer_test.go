package debug

import (
	"fmt"
	"testing"
	"time"

	"github.com/ray/goreact/pkg/actor/schema"
)

func makeTestTool() *schema.Tool {
	return schema.NewTool(
		"test_tool",
		"A test tool",
		schema.Define(
			schema.Param("input", schema.String, "Input").Required(),
		),
		func(p schema.ValidatedParams) (any, error) {
			return "result: " + p.GetString("input"), nil
		},
	)
}

func makeFailingTool() *schema.Tool {
	return schema.NewTool(
		"failing_tool",
		"A tool that fails",
		schema.Define(
			schema.Param("input", schema.String, "Input").Required(),
		),
		func(p schema.ValidatedParams) (any, error) {
			return nil, fmt.Errorf("intentional failure")
		},
	)
}

func TestExecutionTracer(t *testing.T) {
	tracer := NewExecutionTracer(true)
	tool := WithTracing(tracer).Wrap(makeTestTool())

	// 执行
	result, err := tool.Execute(map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result: hello" {
		t.Errorf("expected 'result: hello', got '%v'", result)
	}

	// 检查记录
	records := tracer.GetRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	record := records[0]
	if record.ToolName != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", record.ToolName)
	}
	if record.Error != nil {
		t.Errorf("expected no error, got %v", record.Error)
	}
	if record.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestExecutionTracerFailure(t *testing.T) {
	tracer := NewExecutionTracer(true)
	tool := WithTracing(tracer).Wrap(makeFailingTool())

	_, err := tool.Execute(map[string]any{"input": "hello"})
	if err == nil {
		t.Fatal("expected error")
	}

	records := tracer.GetRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Error == nil {
		t.Error("expected error in record")
	}
}

func TestExecutionTracerDisabled(t *testing.T) {
	tracer := NewExecutionTracer(false)
	tool := WithTracing(tracer).Wrap(makeTestTool())

	tool.Execute(map[string]any{"input": "hello"})

	records := tracer.GetRecords()
	if len(records) != 0 {
		t.Errorf("expected 0 records when disabled, got %d", len(records))
	}
}

func TestExecutionTracerClear(t *testing.T) {
	tracer := NewExecutionTracer(true)
	tool := WithTracing(tracer).Wrap(makeTestTool())

	tool.Execute(map[string]any{"input": "hello"})
	tool.Execute(map[string]any{"input": "world"})

	if len(tracer.GetRecords()) != 2 {
		t.Fatal("expected 2 records")
	}

	tracer.Clear()

	if len(tracer.GetRecords()) != 0 {
		t.Error("expected 0 records after clear")
	}
}

func TestExecutionTracerReport(t *testing.T) {
	tracer := NewExecutionTracer(true)
	tool := WithTracing(tracer).Wrap(makeTestTool())

	tool.Execute(map[string]any{"input": "hello"})

	report := tracer.Report()
	if report == "" {
		t.Error("report should not be empty")
	}
	if !containsStr(report, "test_tool") {
		t.Error("report should contain tool name")
	}
	if !containsStr(report, "Success") {
		t.Error("report should contain success status")
	}
}

func TestPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler()
	tool := WithProfiling(profiler).Wrap(makeTestTool())

	for i := 0; i < 5; i++ {
		tool.Execute(map[string]any{"input": fmt.Sprintf("test_%d", i)})
	}

	report := profiler.Report()
	if !containsStr(report, "test_tool") {
		t.Error("report should contain tool name")
	}
	if !containsStr(report, "Total Calls: 5") {
		t.Errorf("report should show 5 calls: %s", report)
	}
	if !containsStr(report, "100.0%") {
		t.Error("report should show 100% success rate")
	}
}

func TestPerformanceProfilerMixed(t *testing.T) {
	profiler := NewPerformanceProfiler()

	successTool := WithProfiling(profiler).Wrap(makeTestTool())
	failTool := WithProfiling(profiler).Wrap(makeFailingTool())

	successTool.Execute(map[string]any{"input": "ok"})
	successTool.Execute(map[string]any{"input": "ok"})
	failTool.Execute(map[string]any{"input": "fail"})

	report := profiler.Report()
	if !containsStr(report, "test_tool") {
		t.Error("report should contain test_tool")
	}
	if !containsStr(report, "failing_tool") {
		t.Error("report should contain failing_tool")
	}
}

func TestPerformanceProfilerMinMax(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// 手动记录不同耗时
	profiler.Record("slow_tool", 100*time.Millisecond, true)
	profiler.Record("slow_tool", 200*time.Millisecond, true)
	profiler.Record("slow_tool", 50*time.Millisecond, true)

	report := profiler.Report()
	if !containsStr(report, "Total Calls: 3") {
		t.Errorf("expected 3 calls: %s", report)
	}
}

func TestPerformanceProfilerEmpty(t *testing.T) {
	profiler := NewPerformanceProfiler()
	report := profiler.Report()
	if !containsStr(report, "No performance data") {
		t.Errorf("expected empty report message: %s", report)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
