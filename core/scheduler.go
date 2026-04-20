package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ScheduledTask represents a cron-based scheduled task.
type ScheduledTask struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Expression  string    `json:"expression"`  // cron expression
	Prompt      string    `json:"prompt"`      // the prompt to send to the agent when triggered
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	NextRunAt   time.Time `json:"next_run_at,omitempty"`
	LastRunAt   time.Time `json:"last_run_at,omitempty"`
	RunCount    int       `json:"run_count"`
}

// TaskCallback is the function called when a scheduled task fires.
// It receives the ScheduledTask and should trigger agent execution with the task's prompt.
type TaskCallback func(ctx context.Context, task ScheduledTask)

// CronScheduler manages scheduled tasks using cron expressions.
// When a task's next execution time arrives, the scheduler invokes the registered callback
// to trigger agent execution.
//
// Usage:
//
//	scheduler := core.NewCronScheduler()
//	scheduler.Start(ctx) // starts the background ticker
//	scheduler.Schedule("my-task", "0 9 * * 1-5", "Daily standup summary")
//	// When 9:00 AM arrives on a weekday, the callback fires with the task.
//	scheduler.Stop()
type CronScheduler struct {
	mu      sync.RWMutex
	tasks   map[string]*ScheduledTask
	counter int

	callback TaskCallback // called when a task fires

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCronScheduler creates a new CronScheduler.
// Set the callback with SetCallback before calling Start.
func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		tasks: make(map[string]*ScheduledTask),
	}
}

// SetCallback sets the function invoked when a scheduled task fires.
// This is typically set to trigger agent.Ask() with the task's prompt.
func (s *CronScheduler) SetCallback(cb TaskCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callback = cb
}

// Schedule adds a new scheduled task. Returns the task ID.
func (s *CronScheduler) Schedule(name, expression, prompt string) (string, error) {
	fields, err := parseCronExpression(expression)
	if err != nil {
		return "", fmt.Errorf("invalid cron expression: %w", err)
	}

	nextRun := s.findNextRun(time.Now(), fields)
	if nextRun.IsZero() {
		return "", fmt.Errorf("could not find next run time for expression: %s", expression)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	id := strconv.Itoa(s.counter)

	task := &ScheduledTask{
		ID:         id,
		Name:       name,
		Expression: expression,
		Prompt:     prompt,
		Enabled:    true,
		CreatedAt:  time.Now(),
		NextRunAt:  nextRun,
	}

	s.tasks[id] = task
	return id, nil
}

// Unschedule removes a scheduled task by ID.
// Returns false if the task was not found.
func (s *CronScheduler) Unschedule(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	return true
}

// Enable enables a scheduled task.
func (s *CronScheduler) Enable(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	t.Enabled = true
	return nil
}

// Disable disables a scheduled task without removing it.
func (s *CronScheduler) Disable(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	t.Enabled = false
	return nil
}

// List returns all scheduled tasks.
func (s *CronScheduler) List() []ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ScheduledTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, *t)
	}
	return result
}

// Get returns a scheduled task by ID, or nil if not found.
func (s *CronScheduler) Get(id string) *ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if t, ok := s.tasks[id]; ok {
		cp := *t
		return &cp
	}
	return nil
}

// Start begins the background scheduling loop.
// It checks every 30 seconds for tasks whose NextRunAt has arrived.
// Must be paired with Stop() for cleanup.
func (s *CronScheduler) Start(parentCtx context.Context) {
	s.mu.Lock()
	if s.ctx != nil {
		s.mu.Unlock()
		return // already started
	}
	s.ctx, s.cancel = context.WithCancel(parentCtx)
	s.mu.Unlock()

	s.wg.Add(1)
	go s.runLoop()
}

// Stop stops the scheduling loop and waits for it to finish.
func (s *CronScheduler) Stop() {
	s.mu.Lock()
	if s.cancel == nil {
		s.mu.Unlock()
		return
	}
	s.cancel()
	s.ctx = nil
	s.cancel = nil
	s.mu.Unlock()

	s.wg.Wait()
}

// IsRunning returns whether the scheduler loop is active.
func (s *CronScheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx != nil
}

// TaskCount returns the number of registered tasks.
func (s *CronScheduler) TaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tasks)
}

// SetNextRunAt overrides a task's NextRunAt to a specific time.
// This is useful in tests to force immediate firing without waiting for the ticker.
func (s *CronScheduler) SetNextRunAt(id string, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	task.NextRunAt = t
	return nil
}

// CheckAndFire exposes checkAndFire for direct invocation in tests.
// It checks for due tasks and fires their callbacks immediately.
func (s *CronScheduler) CheckAndFire() {
	s.checkAndFire()
}

// runLoop checks for due tasks every 30 seconds.
func (s *CronScheduler) runLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndFire()
		}
	}
}

// checkAndFire finds tasks whose NextRunAt has arrived, fires them, and advances NextRunAt.
func (s *CronScheduler) checkAndFire() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cb := s.callback

	for _, task := range s.tasks {
		if !task.Enabled || task.NextRunAt.IsZero() {
			continue
		}
		if now.Before(task.NextRunAt) {
			continue
		}

		// Task is due — fire the callback in a goroutine to avoid blocking the loop
		taskCopy := *task
		if cb != nil {
			// Capture context reference before launching goroutine
			// to avoid race with Stop() setting s.ctx = nil
			taskCtx := s.ctx
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				cb(taskCtx, taskCopy)
			}()
		}

		// Update task state
		task.LastRunAt = now
		task.RunCount++

		// Advance to next run time
		fields, err := parseCronExpression(task.Expression)
		if err == nil {
			task.NextRunAt = s.findNextRun(now.Add(time.Minute), fields)
		}
	}
}

// findNextRun finds the next time matching the cron fields, starting from 'after'.
func (s *CronScheduler) findNextRun(after time.Time, fields [][]int) time.Time {
	current := after
	maxIterations := 366 * 24 * 60 // one year of minutes

	for i := 0; i < maxIterations; i++ {
		if matchesCron(current, fields) {
			return current
		}
		current = current.Add(time.Minute)
	}

	return time.Time{} // zero value = not found
}

// matchesCron checks if a time matches the parsed cron fields.
func matchesCron(t time.Time, fields [][]int) bool {
	return containsInt(fields[0], t.Minute()) &&
		containsInt(fields[1], t.Hour()) &&
		containsInt(fields[2], t.Day()) &&
		containsInt(fields[3], int(t.Month())) &&
		containsInt(fields[4], int(t.Weekday()))
}

func containsInt(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// --- Cron expression parsing (mirrors tools/cron.go for core package independence) ---

// parseCronExpression parses a 5-field cron expression (min hour dom month dow).
func parseCronExpression(expr string) ([][]int, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	ranges := [][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}
	result := make([][]int, 5)

	for i, field := range fields {
		var err error
		result[i], err = parseCronField(field, ranges[i][0], ranges[i][1])
		if err != nil {
			names := []string{"minute", "hour", "day", "month", "weekday"}
			return nil, fmt.Errorf("invalid %s field: %w", names[i], err)
		}
	}

	return result, nil
}

// parseCronField parses a single cron field value (supports *, */n, ranges, lists).
func parseCronField(field string, min, max int) ([]int, error) {
	var values []int

	parts := strings.Split(field, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "*" {
			for i := min; i <= max; i++ {
				values = append(values, i)
			}
			continue
		}

		if strings.Contains(part, "/") {
			stepParts := strings.Split(part, "/")
			if len(stepParts) != 2 {
				return nil, fmt.Errorf("invalid step: %s", part)
			}
			step, err := strconv.Atoi(stepParts[1])
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step value: %s", stepParts[1])
			}

			start, end := min, max
			if stepParts[0] != "*" {
				if strings.Contains(stepParts[0], "-") {
					rp := strings.Split(stepParts[0], "-")
					if len(rp) != 2 {
						return nil, fmt.Errorf("invalid range: %s", stepParts[0])
					}
					start, err = strconv.Atoi(rp[0])
					if err != nil {
						return nil, fmt.Errorf("invalid value: %s", rp[0])
					}
					end, err = strconv.Atoi(rp[1])
					if err != nil {
						return nil, fmt.Errorf("invalid value: %s", rp[1])
					}
				} else {
					start, err = strconv.Atoi(stepParts[0])
					if err != nil {
						return nil, fmt.Errorf("invalid value: %s", stepParts[0])
					}
				}
			}

			if start < min || start > max || end < min || end > max {
				return nil, fmt.Errorf("values out of range [%d-%d]", min, max)
			}

			for i := start; i <= end; i += step {
				values = append(values, i)
			}
			continue
		}

		if strings.Contains(part, "-") {
			rp := strings.Split(part, "-")
			if len(rp) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			start, err := strconv.Atoi(rp[0])
			if err != nil {
				return nil, fmt.Errorf("invalid value: %s", rp[0])
			}
			end, err := strconv.Atoi(rp[1])
			if err != nil {
				return nil, fmt.Errorf("invalid value: %s", rp[1])
			}
			if start < min || start > max || end < min || end > max || start > end {
				return nil, fmt.Errorf("range error in: %s", part)
			}
			for i := start; i <= end; i++ {
				values = append(values, i)
			}
			continue
		}

		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", part)
		}
		if val < min || val > max {
			return nil, fmt.Errorf("value %d out of range [%d-%d]", val, min, max)
		}
		values = append(values, val)
	}

	return values, nil
}
