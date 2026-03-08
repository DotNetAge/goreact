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
	
	// GetMetrics 获取所有指标
	GetMetrics() map[string]interface{}
	
	// Reset 重置指标
	Reset()
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
	count    int64
	sum      time.Duration
	min      time.Duration
	max      time.Duration
	mutex    sync.Mutex
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
func (t *Timer) GetMetrics() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	metrics := make(map[string]interface{})
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

// DefaultMetrics 默认指标收集实现
type DefaultMetrics struct {
	latencies map[string]*Timer
	successes map[string]*Counter
	errors    map[string]*Counter
	mutex     sync.Mutex
}

// NewDefaultMetrics 创建新的默认指标收集器
func NewDefaultMetrics() *DefaultMetrics {
	return &DefaultMetrics{
		latencies: make(map[string]*Timer),
		successes: make(map[string]*Counter),
		errors:    make(map[string]*Counter),
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

// GetMetrics 获取所有指标
func (m *DefaultMetrics) GetMetrics() map[string]interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	metrics := make(map[string]interface{})
	
	// 记录延迟指标
	latencyMetrics := make(map[string]interface{})
	for operation, timer := range m.latencies {
		latencyMetrics[operation] = timer.GetMetrics()
	}
	metrics["latencies"] = latencyMetrics
	
	// 记录成功指标
	successMetrics := make(map[string]int64)
	for operation, counter := range m.successes {
		successMetrics[operation] = counter.Get()
	}
	metrics["successes"] = successMetrics
	
	// 记录错误指标
	errorMetrics := make(map[string]int64)
	for operation, counter := range m.errors {
		errorMetrics[operation] = counter.Get()
	}
	metrics["errors"] = errorMetrics
	
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
}
