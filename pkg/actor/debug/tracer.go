package debug

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ray/goreact/pkg/tools"
)

// ExecutionRecord 执行记录
type ExecutionRecord struct {
	ToolName   string
	Parameters map[string]any
	Output     any
	Error      error
	Duration   time.Duration
	Timestamp  time.Time
}

// ExecutionTracer 执行追踪器
type ExecutionTracer struct {
	enabled bool
	mu      sync.RWMutex
	records []ExecutionRecord
}

// NewExecutionTracer 创建执行追踪器
func NewExecutionTracer(enabled bool) *ExecutionTracer {
	return &ExecutionTracer{
		enabled: enabled,
		records: make([]ExecutionRecord, 0),
	}
}

// Record 记录执行
func (t *ExecutionTracer) Record(record ExecutionRecord) {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.records = append(t.records, record)
}

// GetRecords 获取所有记录
func (t *ExecutionTracer) GetRecords() []ExecutionRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]ExecutionRecord{}, t.records...)
}

// Clear 清空记录
func (t *ExecutionTracer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records = make([]ExecutionRecord, 0)
}

// Report 生成报告
func (t *ExecutionTracer) Report() string {
	records := t.GetRecords()
	if len(records) == 0 {
		return "No executions recorded"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Total Executions: %d\n\n", len(records)))

	for i, record := range records {
		sb.WriteString(fmt.Sprintf("Execution #%d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Tool: %s\n", record.ToolName))
		sb.WriteString(fmt.Sprintf("  Duration: %v\n", record.Duration))
		sb.WriteString(fmt.Sprintf("  Timestamp: %s\n", record.Timestamp.Format("15:04:05.000")))

		if record.Error != nil {
			sb.WriteString(fmt.Sprintf("  Status: ❌ Failed\n"))
			sb.WriteString(fmt.Sprintf("  Error: %v\n", record.Error))
		} else {
			sb.WriteString(fmt.Sprintf("  Status: ✅ Success\n"))
			outputStr := fmt.Sprintf("%v", record.Output)
			if len(outputStr) > 50 {
				outputStr = outputStr[:50] + "..."
			}
			sb.WriteString(fmt.Sprintf("  Output: %s\n", outputStr))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// TracingWrapper 追踪包装器
type TracingWrapper struct {
	tracer *ExecutionTracer
}

// WithTracing 创建追踪包装器
func WithTracing(tracer *ExecutionTracer) *TracingWrapper {
	return &TracingWrapper{tracer: tracer}
}

func (w *TracingWrapper) Wrap(t tools.Tool) tools.Tool {
	return &tracingTool{
		base:   t,
		tracer: w.tracer,
	}
}

type tracingTool struct {
	base   tools.Tool
	tracer *ExecutionTracer
}

func (t *tracingTool) Name() string {
	return t.base.Name()
}

func (t *tracingTool) Description() string {
	return t.base.Description()
}

func (t *tracingTool) Execute(params map[string]any) (any, error) {
	start := time.Now()

	output, err := t.base.Execute(params)

	duration := time.Since(start)

	t.tracer.Record(ExecutionRecord{
		ToolName:   t.base.Name(),
		Parameters: params,
		Output:     output,
		Error:      err,
		Duration:   duration,
		Timestamp:  start,
	})

	return output, err
}

// PerformanceProfiler 性能分析器
type PerformanceProfiler struct {
	mu    sync.RWMutex
	stats map[string]*ToolStats
}

// ToolStats 工具统计
type ToolStats struct {
	TotalCalls    int
	SuccessCalls  int
	FailedCalls   int
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
}

// NewPerformanceProfiler 创建性能分析器
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		stats: make(map[string]*ToolStats),
	}
}

// Record 记录执行
func (p *PerformanceProfiler) Record(toolName string, duration time.Duration, success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats, ok := p.stats[toolName]
	if !ok {
		stats = &ToolStats{
			MinDuration: duration,
			MaxDuration: duration,
		}
		p.stats[toolName] = stats
	}

	stats.TotalCalls++
	if success {
		stats.SuccessCalls++
	} else {
		stats.FailedCalls++
	}

	stats.TotalDuration += duration

	if duration < stats.MinDuration {
		stats.MinDuration = duration
	}
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}
}

// Report 生成报告
func (p *PerformanceProfiler) Report() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.stats) == 0 {
		return "No performance data collected"
	}

	var sb strings.Builder
	sb.WriteString("Performance Report:\n\n")

	for toolName, stats := range p.stats {
		avgDuration := stats.TotalDuration / time.Duration(stats.TotalCalls)
		successRate := float64(stats.SuccessCalls) / float64(stats.TotalCalls) * 100

		sb.WriteString(fmt.Sprintf("Tool: %s\n", toolName))
		sb.WriteString(fmt.Sprintf("  Total Calls: %d\n", stats.TotalCalls))
		sb.WriteString(fmt.Sprintf("  Success Rate: %.1f%%\n", successRate))
		sb.WriteString(fmt.Sprintf("  Avg Duration: %v\n", avgDuration))
		sb.WriteString(fmt.Sprintf("  Min Duration: %v\n", stats.MinDuration))
		sb.WriteString(fmt.Sprintf("  Max Duration: %v\n", stats.MaxDuration))
		sb.WriteString("\n")
	}

	return sb.String()
}

// ProfilingWrapper 性能分析包装器
type ProfilingWrapper struct {
	profiler *PerformanceProfiler
}

// WithProfiling 创建性能分析包装器
func WithProfiling(profiler *PerformanceProfiler) *ProfilingWrapper {
	return &ProfilingWrapper{profiler: profiler}
}

func (w *ProfilingWrapper) Wrap(t tools.Tool) tools.Tool {
	return &profilingTool{
		base:     t,
		profiler: w.profiler,
	}
}

type profilingTool struct {
	base     tools.Tool
	profiler *PerformanceProfiler
}

func (t *profilingTool) Name() string {
	return t.base.Name()
}

func (t *profilingTool) Description() string {
	return t.base.Description()
}

func (t *profilingTool) Execute(params map[string]any) (any, error) {
	start := time.Now()

	output, err := t.base.Execute(params)

	duration := time.Since(start)
	t.profiler.Record(t.base.Name(), duration, err == nil)

	return output, err
}
