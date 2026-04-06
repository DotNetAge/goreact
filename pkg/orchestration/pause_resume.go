package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Pause-Resume Manager
// =============================================================================

// PauseResumeManager manages pause and resume operations
type PauseResumeManager struct {
	stateManager   *StateManager
	snapshotMgr    SnapshotManager
	dependencyGraph *Graph
	mu             sync.RWMutex
}

// NewPauseResumeManager creates a new pause-resume manager
func NewPauseResumeManager(stateManager *StateManager) *PauseResumeManager {
	return &PauseResumeManager{
		stateManager: stateManager,
		snapshotMgr:  NewDefaultSnapshotManager(),
	}
}

// PauseAgent pauses a specific agent
func (m *PauseResumeManager) PauseAgent(ctx context.Context, sessionName string, agentName string, question *PendingQuestion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	state, err := m.stateManager.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	agentState, exists := state.AgentStates[agentName]
	if !exists {
		return fmt.Errorf("agent %s not found in session %s", agentName, sessionName)
	}
	
	// Update agent state to suspended
	agentState.Status = AgentStatusSuspended
	agentState.PendingQuestion = question
	
	// Add pending question to session
	state.PendingQuestions = append(state.PendingQuestions, question)
	
	// Update dependent agents to blocked
	m.blockDependentAgents(state, agentName)
	
	// Save state
	state.ExecutionPhase = PhaseSuspended
	return m.stateManager.SaveState(ctx, state)
}

// PauseAll pauses all agents
func (m *PauseResumeManager) PauseAll(ctx context.Context, sessionName string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	state, err := m.stateManager.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	for _, agentState := range state.AgentStates {
		if agentState.Status == AgentStatusRunning {
			agentState.Status = AgentStatusSuspended
		}
	}
	
	state.ExecutionPhase = PhaseSuspended
	return m.stateManager.SaveState(ctx, state)
}

// Resume resumes execution with user answers
func (m *PauseResumeManager) Resume(ctx context.Context, req *ResumeRequest) (*ResumeResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	state, err := m.stateManager.GetState(ctx, req.SessionName)
	if err != nil {
		return nil, err
	}
	
	result := &ResumeResult{
		SessionName:   req.SessionName,
		ResumedAgents: make([]string, 0),
		StillPending:  make([]string, 0),
		Errors:        make(map[string]error),
	}
	
	// Process answers
	for _, q := range state.PendingQuestions {
		_, hasAnswer := req.AnswerMap.Answers[q.ID]
		
		if !hasAnswer && !req.ResumeAll {
			result.StillPending = append(result.StillPending, q.ID)
			continue
		}
		
		// Apply answer to agent state
		if agentState, exists := state.AgentStates[q.AgentName]; exists {
			// Inject answer as observation
			if agentState.FrozenState != nil {
				// Agent will thaw with the answer
			}
			
			agentState.Status = AgentStatusPending
			agentState.PendingQuestion = nil
			result.ResumedAgents = append(result.ResumedAgents, q.AgentName)
		}
		
		// Remove question
		m.removeQuestion(state, q.ID)
	}
	
	// Unblock dependent agents
	for _, agentName := range result.ResumedAgents {
		m.unblockDependentAgents(state, agentName)
	}
	
	// Update phase if all questions answered
	if len(state.PendingQuestions) == 0 {
		state.ExecutionPhase = PhaseExecuting
	}
	
	if err := m.stateManager.SaveState(ctx, state); err != nil {
		return nil, err
	}
	
	return result, nil
}

// ResumeSelected resumes specific agents
func (m *PauseResumeManager) ResumeSelected(ctx context.Context, sessionName string, agentNames []string, answers map[string]string) (*ResumeResult, error) {
	// Build answer map
	answerMap := &AnswerMap{
		SessionName: sessionName,
		Answers:     answers,
		Timestamp:   time.Now(),
	}
	
	// Get state to map agent names to question IDs
	state, err := m.stateManager.GetState(ctx, sessionName)
	if err != nil {
		return nil, err
	}
	
	// Check dependencies
	for _, agentName := range agentNames {
		if err := m.checkDependencies(state, agentName, agentNames); err != nil {
			return nil, err
		}
	}
	
	return m.Resume(ctx, &ResumeRequest{
		SessionName: sessionName,
		AnswerMap:   answerMap,
		ResumeAll:   false,
	})
}

// blockDependentAgents blocks agents that depend on the paused agent
func (m *PauseResumeManager) blockDependentAgents(state *OrchestrationState, pausedAgent string) {
	if state.Plan == nil || state.Plan.DependencyGraph == nil {
		return
	}
	
	// Find sub-task for this agent
	var pausedSubTask string
	for name, agentState := range state.AgentStates {
		if name == pausedAgent {
			pausedSubTask = agentState.SubTaskName
			break
		}
	}
	
	if pausedSubTask == "" {
		return
	}
	
	// Find dependent sub-tasks
	for _, edge := range state.Plan.DependencyGraph.Edges {
		if edge.From == pausedSubTask {
			// Find agent for this sub-task
			for _, agentState := range state.AgentStates {
				if agentState.SubTaskName == edge.To && agentState.Status == AgentStatusPending {
					agentState.Status = AgentStatusBlocked
				}
			}
		}
	}
}

// unblockDependentAgents unblocks agents when their dependency is resolved
func (m *PauseResumeManager) unblockDependentAgents(state *OrchestrationState, resumedAgent string) {
	if state.Plan == nil || state.Plan.DependencyGraph == nil {
		return
	}
	
	var resumedSubTask string
	for name, agentState := range state.AgentStates {
		if name == resumedAgent {
			resumedSubTask = agentState.SubTaskName
			break
		}
	}
	
	if resumedSubTask == "" {
		return
	}
	
	// Find dependent sub-tasks and check if all dependencies are met
	for _, edge := range state.Plan.DependencyGraph.Edges {
		if edge.From == resumedSubTask {
			// Check if this sub-task is now unblocked
			allDepsMet := true
			for _, e := range state.Plan.DependencyGraph.Edges {
				if e.To == edge.To {
					depAgent := m.findAgentForSubTask(state, e.From)
					if depAgent != nil && depAgent.Status != AgentStatusCompleted {
						allDepsMet = false
						break
					}
				}
			}
			
			if allDepsMet {
				agentState := m.findAgentForSubTask(state, edge.To)
				if agentState != nil && agentState.Status == AgentStatusBlocked {
					agentState.Status = AgentStatusPending
				}
			}
		}
	}
}

// checkDependencies checks if dependencies allow resume
func (m *PauseResumeManager) checkDependencies(state *OrchestrationState, agentName string, selectedAgents []string) error {
	agentState, exists := state.AgentStates[agentName]
	if !exists {
		return fmt.Errorf("agent %s not found", agentName)
	}
	
	// Check if agent has dependencies
	if agentState.SubTaskName == "" || state.Plan == nil {
		return nil
	}
	
	for _, edge := range state.Plan.DependencyGraph.Edges {
		if edge.To == agentState.SubTaskName {
			depAgent := m.findAgentForSubTask(state, edge.From)
			if depAgent != nil {
				// Check if dependency agent is completed or being resumed
				if depAgent.Status == AgentStatusSuspended {
					isSelected := false
					for _, name := range selectedAgents {
						if name == depAgent.AgentName {
							isSelected = true
							break
						}
					}
					if !isSelected {
						return fmt.Errorf("agent %s depends on suspended agent %s which is not being resumed", agentName, depAgent.AgentName)
					}
				}
			}
		}
	}
	
	return nil
}

// findAgentForSubTask finds the agent assigned to a sub-task
func (m *PauseResumeManager) findAgentForSubTask(state *OrchestrationState, subTaskName string) *AgentState {
	for _, agentState := range state.AgentStates {
		if agentState.SubTaskName == subTaskName {
			return agentState
		}
	}
	return nil
}

// removeQuestion removes a question from the pending list
func (m *PauseResumeManager) removeQuestion(state *OrchestrationState, questionID string) {
	for i, q := range state.PendingQuestions {
		if q.ID == questionID {
			state.PendingQuestions = append(state.PendingQuestions[:i], state.PendingQuestions[i+1:]...)
			return
		}
	}
}

// =============================================================================
// Synchronization Barrier
// =============================================================================

// SyncBarrier implements a synchronization barrier
type SyncBarrier struct {
	count     int
	threshold int
	cond      *sync.Cond
}

// NewSyncBarrier creates a new sync barrier
func NewSyncBarrier(threshold int) *SyncBarrier {
	return &SyncBarrier{
		count:     0,
		threshold: threshold,
		cond:      sync.NewCond(&sync.Mutex{}),
	}
}

// Wait waits at the barrier
func (b *SyncBarrier) Wait() {
	b.cond.L.Lock()
	b.count++
	
	if b.count >= b.threshold {
		b.count = 0
		b.cond.Broadcast()
	} else {
		b.cond.Wait()
	}
	b.cond.L.Unlock()
}

// Reset resets the barrier
func (b *SyncBarrier) Reset() {
	b.cond.L.Lock()
	b.count = 0
	b.cond.Broadcast()
	b.cond.L.Unlock()
}

// =============================================================================
// Deadlock Detection
// =============================================================================

// DeadlockDetector detects deadlocks in agent dependencies
type DeadlockDetector struct {
	dependencyGraph *Graph
}

// NewDeadlockDetector creates a new deadlock detector
func NewDeadlockDetector(graph *Graph) *DeadlockDetector {
	return &DeadlockDetector{dependencyGraph: graph}
}

// Detect detects deadlocks
func (d *DeadlockDetector) Detect(state *OrchestrationState) []string {
	// Build wait-for graph based on agent states
	waitFor := make(map[string][]string)
	
	for _, agentState := range state.AgentStates {
		if agentState.Status == AgentStatusBlocked || agentState.Status == AgentStatusSuspended {
			// Find what this agent is waiting for
			for _, edge := range d.dependencyGraph.Edges {
				if edge.To == agentState.SubTaskName {
					depAgent := d.findAgentForSubTask(state, edge.From)
					if depAgent != nil && depAgent.Status != AgentStatusCompleted {
						waitFor[agentState.AgentName] = append(waitFor[agentState.AgentName], depAgent.AgentName)
					}
				}
			}
		}
	}
	
	// Detect cycle in wait-for graph
	return d.detectCycle(waitFor)
}

// detectCycle detects cycle in wait-for graph
func (d *DeadlockDetector) detectCycle(graph map[string][]string) []string {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cycle []string
	
	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		inStack[node] = true
		
		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					cycle = append(cycle, node)
					return true
				}
			} else if inStack[neighbor] {
				cycle = append(cycle, node, neighbor)
				return true
			}
		}
		
		inStack[node] = false
		return false
	}
	
	for node := range graph {
		if !visited[node] {
			if dfs(node) {
				return cycle
			}
		}
	}
	
	return nil
}

// findAgentForSubTask finds agent for sub-task
func (d *DeadlockDetector) findAgentForSubTask(state *OrchestrationState, subTaskName string) *AgentState {
	for _, agentState := range state.AgentStates {
		if agentState.SubTaskName == subTaskName {
			return agentState
		}
	}
	return nil
}

// =============================================================================
// Timeout Handler
// =============================================================================

// TimeoutHandler handles timeouts for paused agents
type TimeoutHandler struct {
	defaultTimeout time.Duration
}

// NewTimeoutHandler creates a new timeout handler
func NewTimeoutHandler(defaultTimeout time.Duration) *TimeoutHandler {
	return &TimeoutHandler{defaultTimeout: defaultTimeout}
}

// CheckTimeouts checks for timed-out agents
func (h *TimeoutHandler) CheckTimeouts(state *OrchestrationState) []string {
	timedOut := []string{}
	
	for _, agentState := range state.AgentStates {
		if agentState.Status == AgentStatusSuspended {
			// Check if question has timeout
			if agentState.PendingQuestion != nil {
				// Simplified: would check actual timeout
			}
		}
	}
	
	return timedOut
}

// =============================================================================
// State Lock Manager
// =============================================================================

// StateLockManager manages locks for session state
type StateLockManager struct {
	mu           sync.RWMutex
	sessionLocks map[string]*SessionLock
	lockTimeout  time.Duration
}

// SessionLock represents a lock for a session
type SessionLock struct {
	SessionName string
	WriteLock   *sync.Mutex
	ReadCount   int32
	LastAccess  time.Time
	PendingOps  int32
}

// NewStateLockManager creates a new state lock manager
func NewStateLockManager(lockTimeout time.Duration) *StateLockManager {
	return &StateLockManager{
		sessionLocks: make(map[string]*SessionLock),
		lockTimeout:  lockTimeout,
	}
}

// AcquireWriteLock acquires a write lock for a session
func (lm *StateLockManager) AcquireWriteLock(ctx context.Context, sessionName string) (*SessionLock, error) {
	lm.mu.Lock()
	sessionLock, exists := lm.sessionLocks[sessionName]
	if !exists {
		sessionLock = &SessionLock{
			SessionName: sessionName,
			WriteLock:   &sync.Mutex{},
		}
		lm.sessionLocks[sessionName] = sessionLock
	}
	lm.mu.Unlock()
	
	done := make(chan struct{})
	go func() {
		sessionLock.WriteLock.Lock()
		close(done)
	}()
	
	select {
	case <-done:
		sessionLock.LastAccess = time.Now()
		return sessionLock, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReleaseWriteLock releases a write lock
func (lm *StateLockManager) ReleaseWriteLock(lock *SessionLock) {
	lock.WriteLock.Unlock()
}
