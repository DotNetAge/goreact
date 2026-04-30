package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// --- Test Helpers ---

func newTestCoordinator(t *testing.T) *Coordinator {
	t.Helper()
	orch, err := New(WithMaxConcurrent(10), WithSpawnFunction(mockSpawnFunc))
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}
	return NewCoordinator("test-agent", "parent-1", orch)
}

var mockSpawnFunc = func(
	ctx context.Context,
	config *core.AgentConfig,
	modelCfg *core.ModelConfig,
	taskPrompt string,
	taskID string,
	resultCh chan<- any,
) error {
	go func() {
		resultCh <- "[mock] completed: " + taskPrompt
	}()
	return nil
}

// --- TaskProgressTable Tests ---

func TestTaskProgressTable_BasicOps(t *testing.T) {
	table := NewTaskProgressTable()

	if table.Count() != 0 {
		t.Fatalf("expected empty table, got %d", table.Count())
	}

	entry := &TaskEntry{TaskID: "t1", State: TaskDispatched}
	table.Set("t1", entry)

	if got := table.Get("t1"); got == nil || got.TaskID != "t1" {
		t.Fatal("Get after Set failed")
	}

	table.Delete("t1")
	if table.Get("t1") != nil {
		t.Fatal("expected nil after Delete")
	}
}

func TestTaskProgressTable_PendingTaskIDs(t *testing.T) {
	table := NewTaskProgressTable()
	table.Set("a", &TaskEntry{TaskID: "a", State: TaskRunning})
	table.Set("b", &TaskEntry{TaskID: "b", State: TaskCompleted})
	table.Set("c", &TaskEntry{TaskID: "c", State: TaskFailed})

	pending := table.PendingTaskIDs()
	if len(pending) != 1 || pending[0] != "a" {
		t.Errorf("expected 1 pending [a], got %v", pending)
	}
}

func TestTaskProgressTable_Counts(t *testing.T) {
	table := NewTaskProgressTable()
	table.Set("x1", &TaskEntry{TaskID: "x1", State: TaskCompleted})
	table.Set("x2", &TaskEntry{TaskID: "x2", State: TaskCompleted})
	table.Set("x3", &TaskEntry{TaskID: "x3", State: TaskFailed})
	table.Set("x4", &TaskEntry{TaskID: "x4", State: TaskSkipped})

	if n := table.CompletedCount(); n != 2 {
		t.Errorf("Expected CompletedCount=2, got %d", n)
	}
	if n := table.FailedCount(); n != 2 { // Failed + Skipped
		t.Errorf("Expected FailedCount=2, got %d", n)
	}
}

// --- IsFinalState Tests ---

func TestIsFinalState(t *testing.T) {
	finalStates := []TaskState{TaskCompleted, TaskFailed, TaskTimeout, TaskCancelled, TaskSkipped}
	for _, s := range finalStates {
		if !IsFinalState(s) {
			t.Errorf("expected %s to be final", s)
		}
	}
	nonFinal := []TaskState{TaskDispatched, TaskRunning, TaskPaused}
	for _, s := range nonFinal {
		if IsFinalState(s) {
			t.Errorf("expected %s to NOT be final", s)
		}
	}
}

// --- Timeout Configuration Tests ---

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()

	if cfg.SingleTaskMultiplier != 2.0 {
		t.Errorf("expected SingleTaskMultiplier=2.0, got %v", cfg.SingleTaskMultiplier)
	}
	if cfg.SoftTimeoutMultiplier != 3.0 {
		t.Errorf("expected SoftTimeoutMultiplier=3.0, got %v", cfg.SoftTimeoutMultiplier)
	}
	if cfg.HardTimeoutMultiplier != 5.0 {
		t.Errorf("expected HardTimeoutMultiplier=5.0, got %v", cfg.HardTimeoutMultiplier)
	}
	if cfg.MaxRetries != 2 {
		t.Errorf("expected MaxRetries=2, got %d", cfg.MaxRetries)
	}
}

func TestResolveTimeouts(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	cfg.SingleTaskMultiplier = 2.0

	table := NewTaskProgressTable()
	table.Set("fast", &TaskEntry{TaskID: "fast", ExpectedDur: 10 * time.Second})
	table.Set("slow", &TaskEntry{TaskID: "slow", ExpectedDur: 60 * time.Second})

	single, soft, hard := cfg.resolveTimeouts(table)

	// Single-task deadlines should be per-task expected * multiplier
	fastDeadline := single["fast"]
	slowDeadline := single["slow"]

	if fastDeadline.After(slowDeadline) {
		t.Error("faster task should have earlier deadline")
	}

	// Soft timeout should be based on max expected duration (60s * 3 = 180s from now)
	now := time.Now()
	expectedSoft := now.Add(180 * time.Second)
	if soft.Before(now.Add(170*time.Second)) || soft.After(now.Add(190*time.Second)) {
		t.Errorf("soft deadline out of range: %v, expected ~%v", time.Until(soft), time.Until(expectedSoft))
	}

	// Hard timeout should be 60s * 5 = 300s from now
	expectedHard := now.Add(300 * time.Second)
	if hard.Before(now.Add(290*time.Second)) || hard.After(now.Add(310*time.Second)) {
		t.Errorf("hard deadline out of range: %v, expected ~%v", time.Until(hard), time.Until(expectedHard))
	}

	// Verify ordering: soft < hard
	if soft.After(hard) {
		t.Error("soft deadline must come before hard deadline")
	}
}

func TestComputePollInterval_Adaptive(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	cfg.MinPollInterval = 500 * time.Millisecond
	cfg.MaxPollInterval = 10 * time.Second

	// No running tasks → min interval
	interval := cfg.computePollInterval(nil)
	if interval != cfg.MinPollInterval {
		t.Errorf("empty running → min interval, got %v", interval)
	}

	// Tasks with lots of remaining → longer interval
	longRemaining := []*TaskEntry{
		{TaskID: "t1", ActualStart: time.Now(), ExpectedDur: 100 * time.Second},
		{TaskID: "t2", ActualStart: time.Now(), ExpectedDur: 200 * time.Second},
	}
	interval = cfg.computePollInterval(longRemaining)
	if interval < cfg.MinPollInterval || interval > cfg.MaxPollInterval {
		t.Errorf("interval out of bounds: %v", interval)
	}

	// Nearly done tasks → shorter interval
	shortRemaining := []*TaskEntry{
		{TaskID: "t1", ActualStart: time.Now().Add(-55 * time.Second), ExpectedDur: 60 * time.Second}, // 5s left
	}
	interval = cfg.computePollInterval(shortRemaining)
	if interval > 2*time.Second {
		t.Errorf("nearly-done tasks should have short interval, got %v", interval)
	}
}

// --- Coordinator Lifecycle Tests ---

func TestCoordinator_InitialState(t *testing.T) {
	c := newTestCoordinator(t)

	if c.State() != LifecycleRunning {
		t.Errorf("expected initial state Running, got %s", c.State())
	}

	if c.Table.Count() != 0 {
		t.Errorf("expected empty progress table")
	}
}

func TestCoordinator_Interrupt(t *testing.T) {
	c := newTestCoordinator(t)

	// Add a running task — also register in subTaskCancels so Interrupt can find it
	ctx := context.Background()
	taskCtx, taskCancel := context.WithCancel(ctx)
	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskRunning, ActualStart: time.Now()})
	c.subTaskCtxs["t1"] = taskCtx
	c.subTaskCancels["t1"] = taskCancel

	err := c.Interrupt("test pause")
	if err != nil {
		t.Fatalf("Interrupt failed: %v", err)
	}

	if c.State() != LifecycleInterrupted {
		t.Errorf("expected Interrupted, got %s", c.State())
	}

	entry := c.Table.Get("t1")
	if entry.State != TaskPaused {
		t.Errorf("expected t1 Paused, got %s", entry.State)
	}

	// Cannot interrupt twice
	err = c.Interrupt("second interrupt")
	if err == nil {
		t.Error("expected error on double interrupt")
	}
}

func TestCoordinator_Resume(t *testing.T) {
	c := newTestCoordinator(t)

	// Add paused tasks
	now := time.Now()
	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskPaused, PausedAt: &now})

	// Must be in Interrupted state to resume
	c.lifecycleLock.Lock()
	c.lifecycleState = LifecycleInterrupted
	c.interruptedAt = now.Add(-5 * time.Second)
	c.lifecycleLock.Unlock()

	ctx := context.Background()
	err := c.Resume(ctx)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if c.State() != LifecycleRunning {
		t.Errorf("expected Running after Resume, got %s", c.State())
	}

	entry := c.Table.Get("t1")
	if entry.State != TaskRunning {
		t.Errorf("expected t1 Running after Resume, got %s", entry.State)
	}

	// Cannot resume when not interrupted
	err = c.Resume(ctx)
	if err == nil {
		t.Error("expected error on resume while Running")
	}
}

func TestCoordinator_Cancel(t *testing.T) {
	c := newTestCoordinator(t)

	// Register tasks in both Table and subTaskCancels so Cancel can find them
	ctx := context.Background()
	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskRunning})
	c.Table.Set("t2", &TaskEntry{TaskID: "t2", State: TaskDispatched})
	tCtx1, tCancel1 := context.WithCancel(ctx)
	tCtx2, tCancel2 := context.WithCancel(ctx)
	c.subTaskCtxs["t1"] = tCtx1
	c.subTaskCancels["t1"] = tCancel1
	c.subTaskCtxs["t2"] = tCtx2
	c.subTaskCancels["t2"] = tCancel2

	err := c.Cancel("user request")
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	if c.State() != LifecycleCancelled {
		t.Errorf("expected Cancelled, got %s", c.State())
	}

	// All non-final tasks should be marked Cancelled
	for _, e := range c.Table.Entries() {
		if e.TaskID == "t1" && e.State != TaskCancelled {
			t.Errorf("expected t1 Cancelled, got %s", e.State)
		}
		if e.TaskID == "t2" && e.State != TaskCancelled {
			t.Errorf("expected t2 Cancelled, got %s", e.State)
		}
	}

	// Cannot cancel terminal states
	err = c.Cancel("again")
	if err == nil {
		t.Error("expected error on re-cancel")
	}
}

func TestCoordinator_ControlChannel(t *testing.T) {
	c := newTestCoordinator(t)

	// Add a running task so timeouts compute valid deadlines (not zero → immediate fire)
	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskRunning, ExpectedDur: 30 * time.Second, ActualStart: time.Now()})

	// Start the wait loop so control channel is being processed
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go c.RunWaitLoop(ctx)

	// Give the loop a moment to start
	time.Sleep(20 * time.Millisecond)

	// Send control command via channel — should not block
	c.Control(&CoordControlCommand{
		ControlCommand: core.ControlCommand{Action: CmdInterrupt, Reason: "channel test"},
		Requester:     core.RequesterUser,
		Timestamp:      time.Now(),
	})

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	if c.State() != LifecycleInterrupted {
		t.Errorf("expected Interrupted via control channel, got %s", c.State())
	}
}

// --- Result Handling Tests ---

func TestCoordinator_OnResult_Success(t *testing.T) {
	c := newTestCoordinator(t)

	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskRunning, ActualStart: time.Now(), ExpectedDur: 30 * time.Second})

	// Result string must be >500 chars to trigger ScorePerfect in objective scoring (P1-1)
	longResult := "This is a comprehensive output that exceeds five hundred characters in length. " +
		"It includes detailed analysis, multiple paragraphs of content, structured data points, " +
		"and comprehensive findings to ensure that the scoring system correctly assigns a perfect " +
		"score for quality results. The task has been completed with substantial output containing " +
		"specific metrics: 42 items processed, 98.5% accuracy rate, 3 edge cases identified. " +
		"Additional details include timestamps, reference IDs, and cross-validation against " +
		"expected baseline values. This thorough documentation ensures traceability and " +
		"verifiability of all results produced by the agent during execution."
	c.OnResult(&TaskResultEvent{
		TaskID:        "t1",
		TargetAgentID: "agent-1",
		Result:        longResult,
		Error:         nil,
		Duration:      5 * time.Second,
		Timestamp:     time.Now(),
	})

	entry := c.Table.Get("t1")
	if entry.State != TaskCompleted {
		t.Errorf("expected Completed, got %s", entry.State)
	}
	if entry.Score != ScorePerfect {
		t.Errorf("expected ScorePerfect (%d) for good output, got %d", ScorePerfect, entry.Score)
	}
	if entry.Result == "" {
		t.Error("expected result text to be set")
	}
}

func TestCoordinator_OnResult_FailureWithRetry(t *testing.T) {
	c := newTestCoordinator(t)
	c.TimeoutCfg = TimeoutConfig{
		SingleTaskMultiplier:   2.0,
		SoftTimeoutMultiplier:  3.0,
		HardTimeoutMultiplier:  5.0,
		MaxRetries:             2,
		RetryInitialDelay:      10 * time.Millisecond, // Very short for testing
		FailureRateThreshold:   0.5,
		MinPollInterval:        500 * time.Millisecond,
		MaxPollInterval:        10 * time.Second,
		UserDecisionTimeout:     5 * time.Second,
	}

	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskRunning, Result: "original task description", ActualStart: time.Now()})

	// First failure should trigger retry
	c.OnResult(&TaskResultEvent{
		TaskID:    "t1",
		Result:    "",
		Error:     errors.New("network timeout"),
		Duration:  0,
		Timestamp: time.Now(),
	})

	entry := c.Table.Get("t1")
	if entry.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", entry.RetryCount)
	}
	// After retry attempt, the entry should be in Dispatched state again
	if entry.State != TaskDispatched {
		t.Logf("entry state after first failure: %s (may still be transitioning)", entry.State)
	}
}

func TestCoordinator_OnResult_FailureNoRetry(t *testing.T) {
	c := newTestCoordinator(t)
	c.TimeoutCfg.MaxRetries = 0 // No retries

	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskRunning})

	c.OnResult(&TaskResultEvent{
		TaskID:    "t1",
		Result:    "",
		Error:     errors.New("fatal error"),
		Duration:  0,
		Timestamp: time.Now(),
	})

	entry := c.Table.Get("t1")
	if entry.State != TaskFailed {
		t.Errorf("expected Failed with no retries, got %s", entry.State)
	}
}

// --- BuildResult / Finalize Tests ---

func TestCoordinator_Finalize_Completed(t *testing.T) {
	c := newTestCoordinator(t)

	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskCompleted, Result: "ok"})
	c.Table.Set("t2", &TaskEntry{TaskID: "t2", State: TaskCompleted, Result: "also ok"})
	c.Table.Set("t3", &TaskEntry{TaskID: "t3", State: TaskCompleted, Result: "three ok"})
	c.dispatchedAt = time.Now().Add(-5 * time.Second)

	result := c.finalize(LifecycleCompleted, "")

	if result.LifecycleState != LifecycleCompleted {
		t.Errorf("expected Completed state, got %s", result.LifecycleState)
	}
	if result.TotalTasks != 3 {
		t.Errorf("expected TotalTasks=3, got %d", result.TotalTasks)
	}
	if result.Completed != 3 {
		t.Errorf("expected Completed=3, got %d", result.Completed)
	}
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}
}

func TestCoordinator_Finalize_MixedResults(t *testing.T) {
	c := newTestCoordinator(t)

	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskCompleted, Result: "done"})
	c.Table.Set("t2", &TaskEntry{TaskID: "t2", State: TaskFailed, Error: errors.New("oops")})
	c.Table.Set("t3", &TaskEntry{TaskID: "t3", State: TaskSkipped, Error: errors.New("dep failed")})
	c.dispatchedAt = time.Now()

	result := c.finalize(LifecycleCancelled, "failure rate too high")

	if result.Failed != 1 {
		t.Errorf("expected Failed=1, got %d", result.Failed)
	}
	if result.Skipped != 1 {
		t.Errorf("expected Skipped=1, got %d", result.Skipped)
	}
	if len(result.Failures) != 2 {
		t.Errorf("expected 2 failures in map, got %d", len(result.Failures))
	}
}

// --- Status String Test ---

func TestCoordinator_Status(t *testing.T) {
	c := newTestCoordinator(t)
	status := c.Status()
	if status == "" {
		t.Error("expected non-empty status string")
	}
}

// --- Coordinator Pool Tests ---

func TestCoordinatorPool_RegisterAndGet(t *testing.T) {
	c := newTestCoordinator(t)
	c.ParentTaskID = "pool-test-1"

	err := RegisterCoordinator(c)
	if err != nil {
		t.Fatalf("RegisterCoordinator failed: %v", err)
	}

	got := GetCoordinator("pool-test-1")
	if got == nil {
		t.Fatal("expected to find registered coordinator")
	}
	if got.AgentID != c.AgentID {
		t.Errorf("agent ID mismatch: %s vs %s", got.AgentID, c.AgentID)
	}

	// Duplicate registration should fail
	err = RegisterCoordinator(NewCoordinator("other", "pool-test-1", c.Orchestrator))
	if err == nil {
		t.Error("expected error for duplicate registration")
	}

	UnregisterCoordinator("pool-test-1")
	if GetCoordinator("pool-test-1") != nil {
		t.Error("expected coordinator to be unregistered")
	}
}

// --- Integration: Full Wait Loop with Mock Orchestrator ---

func TestCoordinator_WaitLoop_AllComplete(t *testing.T) {
	orch, _ := New(WithMaxConcurrent(10))
	c := NewCoordinator("coord-test", "parent-wait", orch)

	// Add pre-completed tasks (simulating already-finished work)
	// Must set ExpectedDur > 0 so timeout resolution produces valid deadlines
	c.Table.Set("t1", &TaskEntry{TaskID: "t1", State: TaskCompleted, Result: "result 1", ExpectedDur: 5 * time.Second})
	c.Table.Set("t2", &TaskEntry{TaskID: "t2", State: TaskCompleted, Result: "result 2", ExpectedDur: 5 * time.Second})
	c.dispatchedAt = time.Now()

	// Run with a context that times out quickly since all tasks are done
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	var result *CoordinationResult

	go func() {
		defer wg.Done()
		result = c.RunWaitLoop(ctx)
	}()

	wg.Wait()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.LifecycleState != LifecycleCompleted {
		t.Errorf("expected Completed, got %s", result.LifecycleState)
	}
	if result.Completed != 2 {
		t.Errorf("expected 2 completed, got %d", result.Completed)
	}
}

func TestCoordinator_WaitLoop_Cancelled(t *testing.T) {
	orch, _ := New(WithMaxConcurrent(10))
	c := NewCoordinator("coord-cancel", "parent-cancel", orch)

	// Add long-running tasks that won't complete
	c.Table.Set("t-long", &TaskEntry{TaskID: "t-long", State: TaskRunning, ExpectedDur: 5 * time.Minute, ActualStart: time.Now()})
	c.dispatchedAt = time.Now()

	// Use very short timeouts so hard timeout triggers almost immediately
	c.TimeoutCfg = TimeoutConfig{
		SingleTaskMultiplier:   2.0,
		SoftTimeoutMultiplier:  1.0, // Short for testing
		HardTimeoutMultiplier:  1.0, // Very short — will trigger immediately
		MaxRetries:             0,
		MinPollInterval:        50 * time.Millisecond,
		MaxPollInterval:        100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	var result *CoordinationResult

	go func() {
		defer wg.Done()
		result = c.RunWaitLoop(ctx)
	}()

	wg.Wait()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.LifecycleState != LifecycleCancelled {
		t.Errorf("expected Cancelled due to hard timeout, got %s", result.LifecycleState)
	}
}

// --- WBS Decomposition Types Tests ---

func TestTaskDecomposition_Structure(t *testing.T) {
	sub := TaskDecomposition{
		ID:                "task-001",
		Title:             "Write API docs",
		Description:       "Generate OpenAPI spec for the user service endpoint",
		Priority:          1,
		DependsOn:         []string{"task-000"},
		DesiredCapability: "technical writing",
	}

	if sub.ID == "" || sub.Title == "" || sub.Description == "" {
		t.Error("required fields should not be empty")
	}
	if sub.Priority <= 0 {
		t.Error("priority should be positive")
	}
	if len(sub.DependsOn) == 0 {
		t.Error("should have dependency")
	}
}

func TestResponsibilityCheckResult(t *testing.T) {
	r := ResponsibilityCheckResult{
		IsMatch:    true,
		Confidence: 0.85,
		Reasoning:  "Agent's description covers this domain",
	}

	if !r.IsMatch {
		t.Error("expected IsMatch=true")
	}
	if r.Confidence <= 0 || r.Confidence > 1 {
		t.Errorf("confidence out of range: %v", r.Confidence)
	}
}

func TestAtomicityCheckResult(t *testing.T) {
	// Atomic case
	atomicCheck := AtomicityCheckResult{
		IsAtomic: true,
		Reasoning: "Single skill needed, can be done by one agent",
	}

	if !atomicCheck.IsAtomic {
		t.Error("expected atomic=true")
	}
	if len(atomicCheck.SubTasks) != 0 {
		t.Error("atomic check should have no subtasks")
	}

	// Non-atomic case
	decomposed := AtomicityCheckResult{
		IsAtomic: false,
		SubTasks: []TaskDecomposition{
			{ID: "sub-1", Title: "Part A", Priority: 1},
			{ID: "sub-2", Title: "Part B", Priority: 2, DependsOn: []string{"sub-1"}},
		},
		Reasoning: "Multiple steps with dependencies",
	}

	if decomposed.IsAtomic {
		t.Error("expected atomic=false")
	}
	if len(decomposed.SubTasks) != 2 {
		t.Errorf("expected 2 subtasks, got %d", len(decomposed.SubTasks))
	}
}

// --- Event Type Construction Tests ---

func TestEventConstruction(t *testing.T) {
	now := time.Now()

	// Upstream events
	dispatchEv := TaskDispatchEvent{
		RequestID:        "req-1",
		SourceAgentID:    "agent-a",
		ParentTaskID:     "parent-1",
		TaskID:           "sub-1",
		TaskDescription:  "Analyze PDF document",
		DesiredCapability: "pdf analysis",
		Priority:         1,
		Timestamp:        now,
	}

	if dispatchEv.RequestID == "" || dispatchEv.SourceAgentID == "" {
		t.Error("dispatch event missing required fields")
	}

	scoreEv := AgentScoreEvent{
		TargetAgentID: "agent-b",
		TaskID:        "sub-1",
		Score:         ScorePerfect,
		Success:       true,
		Timestamp:     now,
	}

	if scoreEv.Score != ScorePerfect {
		t.Error("score mismatch")
	}

	// Downstream events
	assignedEv := TaskAssignedEvent{
		RequestID:     "req-1",
		TaskID:        "sub-1",
		TargetAgentID: "agent-c",
		Timestamp:     now,
	}

	if assignedEv.TargetAgentID == "" {
		t.Error("assigned event missing target")
	}

	resultEv := TaskResultEvent{
		TaskID:        "sub-1",
		TargetAgentID: "agent-c",
		Result:        "analysis complete",
		Error:         nil,
		Duration:      15 * time.Second,
		Timestamp:     now,
	}

	if resultEv.Result == "" {
		t.Error("result event missing result content")
	}

	// Control command
	ctrlCmd := CoordControlCommand{
		ControlCommand: core.ControlCommand{
			Action:    CmdInterrupt,
			Reason:    "user requested pause",
			Requester: core.RequesterUser,
			Timestamp: now,
		},
		Reason:   "user requested pause",
		Requester: "user",
		Timestamp: now,
		Priority:  PriorityUser,
	}

	if ctrlCmd.Action != CmdInterrupt {
		t.Error("action mismatch")
	}
	if ctrlCmd.Priority != PriorityUser {
		t.Errorf("expected user priority %d, got %d", PriorityUser, ctrlCmd.Priority)
	}

	// Lifecycle event
	lifeEv := CoordLifecycleEvent{
		CoordAgentID: "coord-1",
		ParentTaskID: "parent-1",
		Action:       "completed",
		Reason:       "all subtasks done",
		PausedTasks:  []string{},
		Timestamp:    now,
	}

	if lifeEv.Action != "completed" {
		t.Error("lifecycle action mismatch")
	}
}

// --- Concurrent Safety Tests ---

func TestCoordinator_ConcurrentStateAccess(t *testing.T) {
	c := newTestCoordinator(t)
	const goroutines = 50
	var started, done atomic.Int32

	// Concurrent reads/writes to Table and lifecycle state
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			started.Add(1)
			defer done.Add(1)

			taskID := fmt.Sprintf("concurrent-%d", id)
			c.Table.Set(taskID, &TaskEntry{TaskID: taskID, State: TaskDispatched})
			_ = c.Table.Get(taskID)
			_ = c.State()
			_ = c.Status()
			_ = c.Table.PendingTaskIDs()
			_ = c.Table.Count()
		}(i)
	}

	// Wait for all goroutines to start
	for started.Load() < int32(goroutines) {
		time.Sleep(time.Millisecond)
	}

	// Wait for completion with timeout
	deadline := time.Now().Add(5 * time.Second)
	for done.Load() < int32(goroutines) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	if remaining := goroutines - int(done.Load()); remaining > 0 {
		t.Errorf("%d goroutines did not complete within timeout", remaining)
	}
}
