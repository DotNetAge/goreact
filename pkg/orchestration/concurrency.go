package orchestration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Concurrency Pool Implementation
// =============================================================================

// DefaultConcurrencyPool implements ConcurrencyPool
type DefaultConcurrencyPool struct {
	config       *ConcurrencyConfig
	semaphore    chan struct{}
	rateLimiter  *SimpleRateLimiter
	wg           sync.WaitGroup
	mu           sync.Mutex
	closed       bool
	stats        PoolStats
}

// SimpleRateLimiter is a simple rate limiter implementation
type SimpleRateLimiter struct {
	interval  time.Duration
	lastTime  time.Time
	tokens    int
	maxTokens int
	mu        sync.Mutex
}

// NewSimpleRateLimiter creates a new simple rate limiter
func NewSimpleRateLimiter(ratePerSecond int, burst int) *SimpleRateLimiter {
	return &SimpleRateLimiter{
		interval:  time.Second / time.Duration(ratePerSecond),
		maxTokens: burst,
		tokens:    burst,
	}
}

// Wait waits for a token to be available
func (r *SimpleRateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(r.lastTime)
	r.lastTime = now
	
	// Add tokens based on elapsed time
	newTokens := int(elapsed / r.interval)
	r.tokens += newTokens
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	
	// Check if we have tokens
	if r.tokens > 0 {
		r.tokens--
		return nil
	}
	
	// Wait for next token
	waitTime := r.interval - (elapsed % r.interval)
	
	select {
	case <-time.After(waitTime):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NewDefaultConcurrencyPool creates a default concurrency pool
func NewDefaultConcurrencyPool(maxWorkers int) *DefaultConcurrencyPool {
	config := DefaultConcurrencyConfig()
	if maxWorkers > 0 {
		config.MaxConcurrent = maxWorkers
	}
	
	return &DefaultConcurrencyPool{
		config:      config,
		semaphore:   make(chan struct{}, config.MaxConcurrent),
		rateLimiter: NewSimpleRateLimiter(config.RateLimitPerAgent, config.TokenBucketSize),
		stats:       PoolStats{},
	}
}

// Submit submits a task for execution
func (p *DefaultConcurrencyPool) Submit(ctx context.Context, task func() error) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return fmt.Errorf("pool is closed")
	}
	p.mu.Unlock()
	
	// Acquire semaphore
	select {
	case p.semaphore <- struct{}{}:
		defer func() { <-p.semaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// Wait for rate limiter
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	
	p.wg.Add(1)
	
	// Update stats atomically under lock
	p.mu.Lock()
	p.stats.ActiveWorkers++
	p.stats.QueuedTasks++
	p.mu.Unlock()
	
	go func() {
		defer p.wg.Done()
		defer func() {
			p.mu.Lock()
			p.stats.ActiveWorkers--
			p.stats.CompletedTasks++
			p.mu.Unlock()
		}()
		
		if err := task(); err != nil {
			p.mu.Lock()
			p.stats.FailedTasks++
			p.mu.Unlock()
		}
	}()
	
	return nil
}

// WaitAll waits for all tasks to complete
func (p *DefaultConcurrencyPool) WaitAll() error {
	p.wg.Wait()
	return nil
}

// Close closes the pool
func (p *DefaultConcurrencyPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
}

// Stats returns pool statistics
func (p *DefaultConcurrencyPool) Stats() PoolStats {
	return p.stats
}

// =============================================================================
// Semaphore Implementation
// =============================================================================

// Semaphore implements a simple semaphore
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore creates a new semaphore
func NewSemaphore(maxConcurrent int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, maxConcurrent),
	}
}

// Acquire acquires the semaphore
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases the semaphore
func (s *Semaphore) Release() {
	<-s.ch
}

// TryAcquire tries to acquire without blocking
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// =============================================================================
// Token Bucket Implementation
// =============================================================================

// TokenBucket implements a token bucket for rate limiting
type TokenBucket struct {
	tokens     int64
	maxTokens  int64
	refillRate int64 // tokens per second
	mu         sync.Mutex
	stopCh     chan struct{}
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(maxTokens, refillRate int) *TokenBucket {
	tb := &TokenBucket{
		tokens:     int64(maxTokens),
		maxTokens:  int64(maxTokens),
		refillRate: int64(refillRate),
		stopCh:     make(chan struct{}),
	}
	
	go tb.refill()
	return tb
}

// refill refills tokens periodically
func (tb *TokenBucket) refill() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			tb.mu.Lock()
			tb.tokens += tb.refillRate
			if tb.tokens > tb.maxTokens {
				tb.tokens = tb.maxTokens
			}
			tb.mu.Unlock()
		case <-tb.stopCh:
			return
		}
	}
}

// Take takes tokens from the bucket
func (tb *TokenBucket) Take(count int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	if tb.tokens >= int64(count) {
		tb.tokens -= int64(count)
		return true
	}
	return false
}

// Wait waits until tokens are available
func (tb *TokenBucket) Wait(ctx context.Context, count int) error {
	for {
		if tb.Take(count) {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Stop stops the token bucket
func (tb *TokenBucket) Stop() {
	close(tb.stopCh)
}

// =============================================================================
// Resource Isolation
// =============================================================================

// ResourceIsolation provides resource isolation for agents
type ResourceIsolation struct {
	agentPools    map[string]*AgentPool
	taskPools     map[string]*TaskPool
	priorityPools map[int]*PriorityPool
	mu            sync.RWMutex
}

// AgentPool represents a pool for a specific agent
type AgentPool struct {
	AgentName string
	Semaphore *Semaphore
	Config    *ConcurrencyConfig
}

// TaskPool represents a pool for a task type
type TaskPool struct {
	TaskType  string
	Semaphore *Semaphore
	Config    *ConcurrencyConfig
}

// PriorityPool represents a pool for a priority level
type PriorityPool struct {
	Priority  int
	Semaphore *Semaphore
	Config    *ConcurrencyConfig
}

// NewResourceIsolation creates a new resource isolation
func NewResourceIsolation() *ResourceIsolation {
	return &ResourceIsolation{
		agentPools:    make(map[string]*AgentPool),
		taskPools:     make(map[string]*TaskPool),
		priorityPools: make(map[int]*PriorityPool),
	}
}

// GetAgentPool gets or creates a pool for an agent
func (r *ResourceIsolation) GetAgentPool(agentName string, config *ConcurrencyConfig) *AgentPool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if pool, exists := r.agentPools[agentName]; exists {
		return pool
	}
	
	pool := &AgentPool{
		AgentName: agentName,
		Semaphore: NewSemaphore(config.MaxConcurrent),
		Config:    config,
	}
	r.agentPools[agentName] = pool
	return pool
}

// GetTaskPool gets or creates a pool for a task type
func (r *ResourceIsolation) GetTaskPool(taskType string, config *ConcurrencyConfig) *TaskPool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if pool, exists := r.taskPools[taskType]; exists {
		return pool
	}
	
	pool := &TaskPool{
		TaskType:  taskType,
		Semaphore: NewSemaphore(config.MaxConcurrent),
		Config:    config,
	}
	r.taskPools[taskType] = pool
	return pool
}

// GetPriorityPool gets or creates a pool for a priority
func (r *ResourceIsolation) GetPriorityPool(priority int, config *ConcurrencyConfig) *PriorityPool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if pool, exists := r.priorityPools[priority]; exists {
		return pool
	}
	
	pool := &PriorityPool{
		Priority:  priority,
		Semaphore: NewSemaphore(config.MaxConcurrent),
		Config:    config,
	}
	r.priorityPools[priority] = pool
	return pool
}

// Execute executes a task with resource isolation
func (r *ResourceIsolation) Execute(ctx context.Context, poolType string, key string, task func() error, config *ConcurrencyConfig) error {
	var sem *Semaphore
	
	switch poolType {
	case "agent":
		sem = r.GetAgentPool(key, config).Semaphore
	case "task":
		sem = r.GetTaskPool(key, config).Semaphore
	case "priority":
		// key should be priority number string
		sem = r.GetPriorityPool(atoi(key), config).Semaphore
	default:
		sem = NewSemaphore(config.MaxConcurrent)
	}
	
	if err := sem.Acquire(ctx); err != nil {
		return err
	}
	defer sem.Release()
	
	return task()
}

// atoi converts string to int
func atoi(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

// =============================================================================
// Priority Queue
// =============================================================================

// PriorityQueue implements a priority queue for tasks
type PriorityQueue struct {
	items []*PriorityItem
	mu    sync.Mutex
}

// PriorityItem represents an item in the priority queue
type PriorityItem struct {
	Value    any
	Priority int
	Index    int
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items: make([]*PriorityItem, 0),
	}
}

// Push pushes an item to the queue
func (pq *PriorityQueue) Push(value any, priority int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	item := &PriorityItem{
		Value:    value,
		Priority: priority,
		Index:    len(pq.items),
	}
	pq.items = append(pq.items, item)
	pq.heapifyUp(len(pq.items) - 1)
}

// Pop pops the highest priority item
func (pq *PriorityQueue) Pop() *PriorityItem {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	if len(pq.items) == 0 {
		return nil
	}
	
	item := pq.items[0]
	last := len(pq.items) - 1
	pq.items[0] = pq.items[last]
	pq.items = pq.items[:last]
	
	if len(pq.items) > 0 {
		pq.heapifyDown(0)
	}
	
	return item
}

// Len returns the length of the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

// heapifyUp moves item up the heap
func (pq *PriorityQueue) heapifyUp(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if pq.items[parent].Priority >= pq.items[index].Priority {
			break
		}
		pq.items[parent], pq.items[index] = pq.items[index], pq.items[parent]
		index = parent
	}
}

// heapifyDown moves item down the heap
func (pq *PriorityQueue) heapifyDown(index int) {
	for {
		left := 2*index + 1
		right := 2*index + 2
		largest := index
		
		if left < len(pq.items) && pq.items[left].Priority > pq.items[largest].Priority {
			largest = left
		}
		if right < len(pq.items) && pq.items[right].Priority > pq.items[largest].Priority {
			largest = right
		}
		
		if largest == index {
			break
		}
		
		pq.items[index], pq.items[largest] = pq.items[largest], pq.items[index]
		index = largest
	}
}

// =============================================================================
// Concurrent Execution Context
// =============================================================================

// ConcurrentExecutionContext provides context for concurrent execution
type ConcurrentExecutionContext struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	errCh      chan error
	doneCh     chan struct{}
	errorCount int32
}

// NewConcurrentExecutionContext creates a new concurrent execution context
func NewConcurrentExecutionContext(parent context.Context) *ConcurrentExecutionContext {
	ctx, cancel := context.WithCancel(parent)
	return &ConcurrentExecutionContext{
		ctx:    ctx,
		cancel: cancel,
		errCh:  make(chan error, 1),
		doneCh: make(chan struct{}),
	}
}

// Context returns the context
func (c *ConcurrentExecutionContext) Context() context.Context {
	return c.ctx
}

// AddTask adds a task to be executed
func (c *ConcurrentExecutionContext) AddTask(task func(ctx context.Context) error) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := task(c.ctx); err != nil {
			atomic.AddInt32(&c.errorCount, 1)
			select {
			case c.errCh <- err:
			default:
			}
		}
	}()
}

// Wait waits for all tasks to complete
func (c *ConcurrentExecutionContext) Wait() error {
	go func() {
		c.wg.Wait()
		close(c.doneCh)
	}()
	
	select {
	case err := <-c.errCh:
		c.cancel()
		return err
	case <-c.doneCh:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

// Cancel cancels the context
func (c *ConcurrentExecutionContext) Cancel() {
	c.cancel()
}
