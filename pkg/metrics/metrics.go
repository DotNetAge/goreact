package metrics

import (
	"sync"
	"time"
)

// Metrics 指标收集接口
type Metrics interface {
	// RecordLatency 记录操作延迟
	RecordLatency(operation string, latency time.Duration)

	// RecordError 记录错误
	RecordError(operation string, err error)

	// RecordSuccess 记录成功操作
	RecordSuccess(operation string)

	// RecordTokenUsage 记录 Token 消耗
	RecordTokenUsage(operation string, promptTokens, completionTokens, totalTokens int)

	// RecordResourceUsage 记录系统资源使用
	RecordResourceUsage(operation string, cpuPercent, memoryMB, gpuPercent, gpuMemoryMB float64)

	// GetMetrics 获取所有指标
	GetMetrics() map[string]any

	// Reset 重置指标
	Reset()

	// Close 关闭指标收集器，释放资源
	Close() error
}

// Counter 计数器
type Counter struct {
	value int64
	mutex sync.Mutex
}

// NewCounter 创建新的计数器
func NewCounter() *Counter {
	return &Counter{
		value: 0,
	}
}

// Increment 增加计数
func (c *Counter) Increment() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value++
}

// Get 获取当前值
func (c *Counter) Get() int64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.value
}

// Reset 重置计数器
func (c *Counter) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value = 0
}

// Timer 计时器
type Timer struct {
	count int64
	sum   time.Duration
	min   time.Duration
	max   time.Duration
	mutex sync.Mutex
}

// NewTimer 创建新的计时器
func NewTimer() *Timer {
	return &Timer{
		count: 0,
		sum:   0,
		min:   time.Hour * 24, // 初始化为一个较大的值
		max:   0,
	}
}

// Record 记录时间
func (t *Timer) Record(duration time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.count++
	t.sum += duration

	if duration < t.min {
		t.min = duration
	}

	if duration > t.max {
		t.max = duration
	}
}

// GetMetrics 获取计时器指标
func (t *Timer) GetMetrics() map[string]any {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	metrics := make(map[string]any)
	metrics["count"] = t.count
	metrics["sum"] = t.sum

	if t.count > 0 {
		metrics["avg"] = t.sum / time.Duration(t.count)
	} else {
		metrics["avg"] = time.Duration(0)
	}

	metrics["min"] = t.min
	metrics["max"] = t.max

	return metrics
}

// Reset 重置计时器
func (t *Timer) Reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.count = 0
	t.sum = 0
	t.min = time.Hour * 24
	t.max = 0
}

// TokenCounter Token 计数器
type TokenCounter struct {
	promptTokens     int64
	completionTokens int64
	totalTokens      int64
	callCount        int64
	mutex            sync.Mutex
}

// NewTokenCounter 创建新的 Token 计数器
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		promptTokens:     0,
		completionTokens: 0,
		totalTokens:      0,
		callCount:        0,
	}
}

// Record 记录 Token 使用
func (tc *TokenCounter) Record(promptTokens, completionTokens, totalTokens int) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.promptTokens += int64(promptTokens)
	tc.completionTokens += int64(completionTokens)
	tc.totalTokens += int64(totalTokens)
	tc.callCount++
}

// GetMetrics 获取 Token 指标
func (tc *TokenCounter) GetMetrics() map[string]any {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	metrics := make(map[string]any)
	metrics["prompt_tokens"] = tc.promptTokens
	metrics["completion_tokens"] = tc.completionTokens
	metrics["total_tokens"] = tc.totalTokens
	metrics["call_count"] = tc.callCount

	if tc.callCount > 0 {
		metrics["avg_prompt_tokens"] = tc.promptTokens / tc.callCount
		metrics["avg_completion_tokens"] = tc.completionTokens / tc.callCount
		metrics["avg_total_tokens"] = tc.totalTokens / tc.callCount
	} else {
		metrics["avg_prompt_tokens"] = int64(0)
		metrics["avg_completion_tokens"] = int64(0)
		metrics["avg_total_tokens"] = int64(0)
	}

	return metrics
}

// Reset 重置 Token 计数器
func (tc *TokenCounter) Reset() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.promptTokens = 0
	tc.completionTokens = 0
	tc.totalTokens = 0
	tc.callCount = 0
}

// ResourceUsageCounter 系统资源使用计数器
type ResourceUsageCounter struct {
	cpuPercent   float64 // CPU 使用率总和
	memoryMB     float64 // 内存使用量总和 (MB)
	gpuPercent   float64 // GPU 使用率总和
	gpuMemoryMB  float64 // GPU 内存使用量总和 (MB)
	callCount    int64   // 调用次数
	maxCPU       float64 // 最大 CPU 使用率
	maxMemory    float64 // 最大内存使用量
	maxGPU       float64 // 最大 GPU 使用率
	maxGPUMemory float64 // 最大 GPU 内存使用量
	mutex        sync.Mutex
}

// NewResourceUsageCounter 创建新的资源使用计数器
func NewResourceUsageCounter() *ResourceUsageCounter {
	return &ResourceUsageCounter{
		cpuPercent:   0,
		memoryMB:     0,
		gpuPercent:   0,
		gpuMemoryMB:  0,
		callCount:    0,
		maxCPU:       0,
		maxMemory:    0,
		maxGPU:       0,
		maxGPUMemory: 0,
	}
}

// Record 记录资源使用
func (rc *ResourceUsageCounter) Record(cpuPercent, memoryMB, gpuPercent, gpuMemoryMB float64) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	rc.cpuPercent += cpuPercent
	rc.memoryMB += memoryMB
	rc.gpuPercent += gpuPercent
	rc.gpuMemoryMB += gpuMemoryMB
	rc.callCount++

	// 更新最大值
	if cpuPercent > rc.maxCPU {
		rc.maxCPU = cpuPercent
	}
	if memoryMB > rc.maxMemory {
		rc.maxMemory = memoryMB
	}
	if gpuPercent > rc.maxGPU {
		rc.maxGPU = gpuPercent
	}
	if gpuMemoryMB > rc.maxGPUMemory {
		rc.maxGPUMemory = gpuMemoryMB
	}
}

// GetMetrics 获取资源使用指标
func (rc *ResourceUsageCounter) GetMetrics() map[string]any {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	metrics := make(map[string]any)
	metrics["call_count"] = rc.callCount
	metrics["total_cpu_percent"] = rc.cpuPercent
	metrics["total_memory_mb"] = rc.memoryMB
	metrics["total_gpu_percent"] = rc.gpuPercent
	metrics["total_gpu_memory_mb"] = rc.gpuMemoryMB

	if rc.callCount > 0 {
		metrics["avg_cpu_percent"] = rc.cpuPercent / float64(rc.callCount)
		metrics["avg_memory_mb"] = rc.memoryMB / float64(rc.callCount)
		metrics["avg_gpu_percent"] = rc.gpuPercent / float64(rc.callCount)
		metrics["avg_gpu_memory_mb"] = rc.gpuMemoryMB / float64(rc.callCount)
	} else {
		metrics["avg_cpu_percent"] = 0.0
		metrics["avg_memory_mb"] = 0.0
		metrics["avg_gpu_percent"] = 0.0
		metrics["avg_gpu_memory_mb"] = 0.0
	}

	metrics["max_cpu_percent"] = rc.maxCPU
	metrics["max_memory_mb"] = rc.maxMemory
	metrics["max_gpu_percent"] = rc.maxGPU
	metrics["max_gpu_memory_mb"] = rc.maxGPUMemory

	return metrics
}

// Reset 重置资源使用计数器
func (rc *ResourceUsageCounter) Reset() {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	rc.cpuPercent = 0
	rc.memoryMB = 0
	rc.gpuPercent = 0
	rc.gpuMemoryMB = 0
	rc.callCount = 0
	rc.maxCPU = 0
	rc.maxMemory = 0
	rc.maxGPU = 0
	rc.maxGPUMemory = 0
}

// DefaultMetrics 默认指标收集实现
type DefaultMetrics struct {
	latencies      map[string]*Timer
	successes      map[string]*Counter
	errors         map[string]*Counter
	tokenUsages    map[string]*TokenCounter
	resourceUsages map[string]*ResourceUsageCounter
	mutex          sync.Mutex
}

// NewDefaultMetrics 创建新的默认指标收集器
func NewDefaultMetrics() *DefaultMetrics {
	return &DefaultMetrics{
		latencies:      make(map[string]*Timer),
		successes:      make(map[string]*Counter),
		errors:         make(map[string]*Counter),
		tokenUsages:    make(map[string]*TokenCounter),
		resourceUsages: make(map[string]*ResourceUsageCounter),
	}
}

// RecordLatency 记录操作延迟
func (m *DefaultMetrics) RecordLatency(operation string, latency time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.latencies[operation]; !ok {
		m.latencies[operation] = NewTimer()
	}

	m.latencies[operation].Record(latency)
}

// RecordError 记录错误
func (m *DefaultMetrics) RecordError(operation string, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.errors[operation]; !ok {
		m.errors[operation] = NewCounter()
	}

	m.errors[operation].Increment()
}

// RecordSuccess 记录成功操作
func (m *DefaultMetrics) RecordSuccess(operation string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.successes[operation]; !ok {
		m.successes[operation] = NewCounter()
	}

	m.successes[operation].Increment()
}

// RecordTokenUsage 记录 Token 消耗
func (m *DefaultMetrics) RecordTokenUsage(operation string, promptTokens, completionTokens, totalTokens int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.tokenUsages[operation]; !ok {
		m.tokenUsages[operation] = NewTokenCounter()
	}

	m.tokenUsages[operation].Record(promptTokens, completionTokens, totalTokens)
}

// RecordResourceUsage 记录系统资源使用
func (m *DefaultMetrics) RecordResourceUsage(operation string, cpuPercent, memoryMB, gpuPercent, gpuMemoryMB float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.resourceUsages[operation]; !ok {
		m.resourceUsages[operation] = NewResourceUsageCounter()
	}

	m.resourceUsages[operation].Record(cpuPercent, memoryMB, gpuPercent, gpuMemoryMB)
}

// GetMetrics 获取所有指标
func (m *DefaultMetrics) GetMetrics() map[string]any {
	m.mutex.Lock()

	// 先复制所有引用，避免在持有锁时调用嵌套方法
	latenciesCopy := make(map[string]*Timer, len(m.latencies))
	for k, v := range m.latencies {
		latenciesCopy[k] = v
	}

	successesCopy := make(map[string]*Counter, len(m.successes))
	for k, v := range m.successes {
		successesCopy[k] = v
	}

	errorsCopy := make(map[string]*Counter, len(m.errors))
	for k, v := range m.errors {
		errorsCopy[k] = v
	}

	tokenUsagesCopy := make(map[string]*TokenCounter, len(m.tokenUsages))
	for k, v := range m.tokenUsages {
		tokenUsagesCopy[k] = v
	}

	resourceUsagesCopy := make(map[string]*ResourceUsageCounter, len(m.resourceUsages))
	for k, v := range m.resourceUsages {
		resourceUsagesCopy[k] = v
	}

	m.mutex.Unlock()

	// 在锁外处理数据
	metrics := make(map[string]any)

	// 记录延迟指标
	latencyMetrics := make(map[string]any)
	for operation, timer := range latenciesCopy {
		latencyMetrics[operation] = timer.GetMetrics()
	}
	metrics["latencies"] = latencyMetrics

	// 记录成功指标
	successMetrics := make(map[string]int64)
	for operation, counter := range successesCopy {
		successMetrics[operation] = counter.Get()
	}
	metrics["successes"] = successMetrics

	// 记录错误指标
	errorMetrics := make(map[string]int64)
	for operation, counter := range errorsCopy {
		errorMetrics[operation] = counter.Get()
	}
	metrics["errors"] = errorMetrics

	// 记录 Token 消耗指标
	tokenMetrics := make(map[string]any)
	for operation, tc := range tokenUsagesCopy {
		tokenMetrics[operation] = tc.GetMetrics()
	}
	metrics["token_usage"] = tokenMetrics

	// 记录系统资源使用指标
	resourceMetrics := make(map[string]any)
	for operation, rc := range resourceUsagesCopy {
		resourceMetrics[operation] = rc.GetMetrics()
	}
	metrics["resource_usage"] = resourceMetrics

	return metrics
}

// Reset 重置指标
func (m *DefaultMetrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, timer := range m.latencies {
		timer.Reset()
	}

	for _, counter := range m.successes {
		counter.Reset()
	}

	for _, counter := range m.errors {
		counter.Reset()
	}

	for _, tc := range m.tokenUsages {
		tc.Reset()
	}

	for _, rc := range m.resourceUsages {
		rc.Reset()
	}
}

// Close 关闭指标收集器，释放资源
func (m *DefaultMetrics) Close() error {
	m.Reset()
	return nil
}
