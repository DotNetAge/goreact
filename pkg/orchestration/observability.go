package orchestration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Metrics Collector Implementation
// =============================================================================

// DefaultMetricsCollector implements MetricsCollector
type DefaultMetricsCollector struct {
	metrics   *Metrics
	tags      map[string]string
	counters  map[string]*int64
	gauges    map[string]*float64
	histograms map[string]*Histogram
	mu        sync.RWMutex
	startTime time.Time
}

// Histogram tracks value distribution
type Histogram struct {
	Count   int64
	Sum     float64
	Min     float64
	Max     float64
	Buckets []int64
}

// NewDefaultMetricsCollector creates a new metrics collector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		metrics: &Metrics{},
		tags:    make(map[string]string),
		counters: make(map[string]*int64),
		gauges:   make(map[string]*float64),
		histograms: make(map[string]*Histogram),
		startTime: time.Now(),
	}
}

// RecordMetric records a metric
func (c *DefaultMetricsCollector) RecordMetric(name string, value float64, tags map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	switch name {
	case "orchestration_latency":
		c.metrics.OrchestrationLatency = time.Duration(value)
	case "execution_time":
		c.metrics.ExecutionTime = time.Duration(value)
	case "sub_task_success_rate":
		c.metrics.SubTaskSuccessRate = value
	case "agent_utilization":
		c.metrics.AgentUtilization = value
	case "parallelism_efficiency":
		c.metrics.ParallelismEfficiency = value
	case "active_agents":
		c.metrics.ActiveAgents = int(value)
	case "queued_tasks":
		c.metrics.QueuedTasks = int(value)
	case "avg_execution_time":
		c.metrics.AvgExecutionTime = time.Duration(value)
	case "concurrency_utilization":
		c.metrics.ConcurrencyUtilization = value
	case "rate_limit_hits":
		c.metrics.RateLimitHits = int(value)
	}
	
	// Store as gauge
	if _, exists := c.gauges[name]; !exists {
		c.gauges[name] = new(float64)
	}
	*c.gauges[name] = value
	
	// Update histogram
	if _, exists := c.histograms[name]; !exists {
		c.histograms[name] = &Histogram{
			Buckets: make([]int64, 10),
		}
	}
	h := c.histograms[name]
	h.Count++
	h.Sum += value
	if h.Count == 1 || value < h.Min {
		h.Min = value
	}
	if value > h.Max {
		h.Max = value
	}
}

// IncrementCounter increments a counter metric
func (c *DefaultMetricsCollector) IncrementCounter(name string, delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.counters[name]; !exists {
		c.counters[name] = new(int64)
	}
	atomic.AddInt64(c.counters[name], delta)
}

// GetMetrics returns current metrics
func (c *DefaultMetricsCollector) GetMetrics() *Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Update derived metrics
	c.metrics.OrchestrationLatency = time.Since(c.startTime)
	
	return c.metrics
}

// Reset resets the metrics
func (c *DefaultMetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.metrics = &Metrics{}
	c.counters = make(map[string]*int64)
	c.gauges = make(map[string]*float64)
	c.histograms = make(map[string]*Histogram)
	c.startTime = time.Now()
}

// GetCounter returns a counter value
func (c *DefaultMetricsCollector) GetCounter(name string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if counter, exists := c.counters[name]; exists {
		return atomic.LoadInt64(counter)
	}
	return 0
}

// GetGauge returns a gauge value
func (c *DefaultMetricsCollector) GetGauge(name string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if gauge, exists := c.gauges[name]; exists {
		return *gauge
	}
	return 0
}

// =============================================================================
// Alert Manager Implementation
// =============================================================================

// DefaultAlertManager implements AlertManager
type DefaultAlertManager struct {
	rules  []*AlertRule
	alerts []*Alert
	mu     sync.RWMutex
}

// AlertRule defines an alert rule
type AlertRule struct {
	Name      string
	Condition func(*Metrics) bool
	Level     AlertLevel
	Message   string
}

// NewDefaultAlertManager creates a new alert manager
func NewDefaultAlertManager() *DefaultAlertManager {
	am := &DefaultAlertManager{
		rules:  make([]*AlertRule, 0),
		alerts: make([]*Alert, 0),
	}
	
	// Add default rules
	am.AddRule("task_timeout", "execution_time > 2 * estimated_time", AlertLevelWarning)
	am.AddRule("agent_unavailable", "active_agents == 0", AlertLevelCritical)
	am.AddRule("high_failure_rate", "sub_task_success_rate < 0.8", AlertLevelWarning)
	am.AddRule("resource_exhausted", "concurrency_utilization >= 1.0", AlertLevelWarning)
	
	return am
}

// Check checks for alert conditions
func (am *DefaultAlertManager) Check(metrics *Metrics) []*Alert {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alerts := make([]*Alert, 0)
	
	for _, rule := range am.rules {
		if rule.Condition(metrics) {
			alert := &Alert{
				Name:      rule.Name,
				Condition: rule.Message,
				Level:     rule.Level,
				Message:   fmt.Sprintf("Alert %s triggered", rule.Name),
				Timestamp: time.Now(),
			}
			alerts = append(alerts, alert)
			am.alerts = append(am.alerts, alert)
		}
	}
	
	return alerts
}

// AddRule adds an alert rule
func (am *DefaultAlertManager) AddRule(name string, condition string, level AlertLevel) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	rule := &AlertRule{
		Name:    name,
		Level:   level,
		Message: condition,
		Condition: func(m *Metrics) bool {
			// Parse condition and evaluate
			// Simplified implementation
			switch name {
			case "task_timeout":
				return m.ExecutionTime > 10*time.Minute
			case "agent_unavailable":
				return m.ActiveAgents == 0
			case "high_failure_rate":
				return m.SubTaskSuccessRate < 0.8
			case "resource_exhausted":
				return m.ConcurrencyUtilization >= 1.0
			default:
				return false
			}
		},
	}
	
	am.rules = append(am.rules, rule)
}

// GetAlerts returns all alerts
func (am *DefaultAlertManager) GetAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.alerts
}

// ClearAlerts clears all alerts
func (am *DefaultAlertManager) ClearAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.alerts = make([]*Alert, 0)
}

// =============================================================================
// Monitoring Dimensions
// =============================================================================

// MonitoringDimensions represents monitoring dimensions
type MonitoringDimensions struct {
	TaskMetrics    *TaskMetrics
	AgentMetrics   *AgentMetrics
	ResourceMetrics *ResourceMetrics
}

// TaskMetrics represents task monitoring metrics
type TaskMetrics struct {
	TotalTasks      int64
	CompletedTasks  int64
	FailedTasks     int64
	PendingTasks    int64
	AvgDuration     time.Duration
	SuccessRate     float64
}

// AgentMetrics represents agent monitoring metrics
type AgentMetrics struct {
	TotalAgents     int64
	HealthyAgents   int64
	UnhealthyAgents int64
	BusyAgents      int64
	IdleAgents      int64
	AvgResponseTime time.Duration
}

// ResourceMetrics represents resource monitoring metrics
type ResourceMetrics struct {
	ConcurrencyUsed    int
	ConcurrencyMax     int
	MemoryUsedMB       float64
	MemoryMaxMB        float64
	NetworkBytesIn     int64
	NetworkBytesOut    int64
}

// =============================================================================
// Metrics Aggregator
// =============================================================================

// MetricsAggregator aggregates metrics from multiple sources
type MetricsAggregator struct {
	collectors []MetricsCollector
	mu         sync.RWMutex
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator() *MetricsAggregator {
	return &MetricsAggregator{
		collectors: make([]MetricsCollector, 0),
	}
}

// AddCollector adds a metrics collector
func (a *MetricsAggregator) AddCollector(collector MetricsCollector) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.collectors = append(a.collectors, collector)
}

// Aggregate aggregates metrics from all collectors
func (a *MetricsAggregator) Aggregate() *Metrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	aggregated := &Metrics{}
	count := len(a.collectors)
	
	if count == 0 {
		return aggregated
	}
	
	for _, collector := range a.collectors {
		m := collector.GetMetrics()
		aggregated.OrchestrationLatency += m.OrchestrationLatency
		aggregated.ExecutionTime += m.ExecutionTime
		aggregated.SubTaskSuccessRate += m.SubTaskSuccessRate
		aggregated.AgentUtilization += m.AgentUtilization
		aggregated.ParallelismEfficiency += m.ParallelismEfficiency
		aggregated.ActiveAgents += m.ActiveAgents
		aggregated.QueuedTasks += m.QueuedTasks
		aggregated.AvgExecutionTime += m.AvgExecutionTime
		aggregated.ConcurrencyUtilization += m.ConcurrencyUtilization
		aggregated.RateLimitHits += m.RateLimitHits
	}
	
	// Average values
	aggregated.SubTaskSuccessRate /= float64(count)
	aggregated.AgentUtilization /= float64(count)
	aggregated.ParallelismEfficiency /= float64(count)
	aggregated.AvgExecutionTime /= time.Duration(count)
	aggregated.ConcurrencyUtilization /= float64(count)
	
	return aggregated
}

// =============================================================================
// Execution Monitor
// =============================================================================

// ExecutionMonitor monitors execution progress
type ExecutionMonitor struct {
	metrics    *DefaultMetricsCollector
	alerts     *DefaultAlertManager
	events     chan MonitorEvent
	mu         sync.RWMutex
	running    bool
}

// MonitorEvent represents a monitoring event
type MonitorEvent struct {
	Type      string
	Timestamp time.Time
	Data      any
}

// NewExecutionMonitor creates a new execution monitor
func NewExecutionMonitor() *ExecutionMonitor {
	return &ExecutionMonitor{
		metrics: NewDefaultMetricsCollector(),
		alerts:  NewDefaultAlertManager(),
		events:  make(chan MonitorEvent, 100),
	}
}

// Start starts monitoring
func (m *ExecutionMonitor) Start(ctx context.Context) {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()
	
	go m.monitorLoop(ctx)
}

// Stop stops monitoring
func (m *ExecutionMonitor) Stop() {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
}

// Record records a monitoring event
func (m *ExecutionMonitor) Record(eventType string, data any) {
	select {
	case m.events <- MonitorEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}:
	default:
		// Event channel full, drop event
	}
}

// GetMetrics returns current metrics
func (m *ExecutionMonitor) GetMetrics() *Metrics {
	return m.metrics.GetMetrics()
}

// CheckAlerts checks for alerts
func (m *ExecutionMonitor) CheckAlerts() []*Alert {
	return m.alerts.Check(m.metrics.GetMetrics())
}

// monitorLoop processes monitoring events
func (m *ExecutionMonitor) monitorLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-m.events:
			m.processEvent(event)
		}
	}
}

// processEvent processes a monitoring event
func (m *ExecutionMonitor) processEvent(event MonitorEvent) {
	switch event.Type {
	case "task_started":
		m.metrics.IncrementCounter("total_tasks", 1)
		m.metrics.RecordMetric("active_tasks", float64(m.metrics.GetCounter("active_tasks")+1), nil)
	case "task_completed":
		m.metrics.IncrementCounter("completed_tasks", 1)
		m.metrics.IncrementCounter("active_tasks", -1)
	case "task_failed":
		m.metrics.IncrementCounter("failed_tasks", 1)
		m.metrics.IncrementCounter("active_tasks", -1)
	case "agent_active":
		m.metrics.IncrementCounter("active_agents", 1)
	case "agent_idle":
		m.metrics.IncrementCounter("active_agents", -1)
	}
}

// =============================================================================
// Performance Tracker
// =============================================================================

// PerformanceTracker tracks performance metrics
type PerformanceTracker struct {
	latencies map[string][]time.Duration
	mu        sync.RWMutex
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		latencies: make(map[string][]time.Duration),
	}
}

// Record records a latency
func (t *PerformanceTracker) Record(name string, duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.latencies[name] = append(t.latencies[name], duration)
}

// GetStats returns statistics for a metric
func (t *PerformanceTracker) GetStats(name string) PerformanceStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	latencies := t.latencies[name]
	if len(latencies) == 0 {
		return PerformanceStats{}
	}
	
	var sum time.Duration
	min := latencies[0]
	max := latencies[0]
	
	for _, l := range latencies {
		sum += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}
	
	return PerformanceStats{
		Count: len(latencies),
		Avg:   sum / time.Duration(len(latencies)),
		Min:   min,
		Max:   max,
		Total: sum,
	}
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	Count int
	Avg   time.Duration
	Min   time.Duration
	Max   time.Duration
	Total time.Duration
}
