package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Orchestration State Manager
// =============================================================================

// StateManager manages orchestration state
type StateManager struct {
	storage StateStorage
	mu      sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager(storage StateStorage) *StateManager {
	return &StateManager{
		storage: storage,
	}
}

// CreateState creates a new orchestration state
func (m *StateManager) CreateState(sessionName string, plan *OrchestrationPlan) *OrchestrationState {
	return &OrchestrationState{
		SessionName:       sessionName,
		Plan:              plan,
		AgentStates:       make(map[string]*AgentState),
		ExecutionPhase:    PhaseIdle,
		PendingQuestions:  make([]*PendingQuestion, 0),
		CompletedSubTasks: make([]string, 0),
		FailedSubTasks:    make([]string, 0),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// GetState retrieves the orchestration state
func (m *StateManager) GetState(ctx context.Context, sessionName string) (*OrchestrationState, error) {
	if m.storage != nil {
		return m.storage.Get(ctx, sessionName)
	}
	return nil, fmt.Errorf("no storage configured")
}

// SaveState saves the orchestration state
func (m *StateManager) SaveState(ctx context.Context, state *OrchestrationState) error {
	state.UpdatedAt = time.Now()
	if m.storage != nil {
		return m.storage.Store(ctx, state)
	}
	return nil
}

// UpdatePhase updates the execution phase
func (m *StateManager) UpdatePhase(ctx context.Context, sessionName string, phase ExecutionPhase) error {
	state, err := m.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	state.ExecutionPhase = phase
	return m.SaveState(ctx, state)
}

// UpdateAgentState updates a specific agent's state
func (m *StateManager) UpdateAgentState(ctx context.Context, sessionName string, agentName string, agentState *AgentState) error {
	if m.storage != nil {
		return m.storage.UpdateAgentState(ctx, sessionName, agentName, agentState)
	}
	
	state, err := m.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	state.AgentStates[agentName] = agentState
	return m.SaveState(ctx, state)
}

// AddCompletedSubTask marks a sub-task as completed
func (m *StateManager) AddCompletedSubTask(ctx context.Context, sessionName string, subTaskName string) error {
	state, err := m.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	state.CompletedSubTasks = append(state.CompletedSubTasks, subTaskName)
	return m.SaveState(ctx, state)
}

// AddFailedSubTask marks a sub-task as failed
func (m *StateManager) AddFailedSubTask(ctx context.Context, sessionName string, subTaskName string) error {
	state, err := m.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	state.FailedSubTasks = append(state.FailedSubTasks, subTaskName)
	return m.SaveState(ctx, state)
}

// AddPendingQuestion adds a pending question
func (m *StateManager) AddPendingQuestion(ctx context.Context, sessionName string, question *PendingQuestion) error {
	state, err := m.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	state.PendingQuestions = append(state.PendingQuestions, question)
	state.ExecutionPhase = PhaseSuspended
	return m.SaveState(ctx, state)
}

// RemovePendingQuestion removes a pending question
func (m *StateManager) RemovePendingQuestion(ctx context.Context, sessionName string, questionID string) error {
	state, err := m.GetState(ctx, sessionName)
	if err != nil {
		return err
	}
	
	for i, q := range state.PendingQuestions {
		if q.ID == questionID {
			state.PendingQuestions = append(state.PendingQuestions[:i], state.PendingQuestions[i+1:]...)
			break
		}
	}
	
	// If no more pending questions, transition to executing
	if len(state.PendingQuestions) == 0 {
		state.ExecutionPhase = PhaseExecuting
	}
	
	return m.SaveState(ctx, state)
}

// =============================================================================
// In-Memory State Storage
// =============================================================================

// MemoryStateStorage implements StateStorage in memory
type MemoryStateStorage struct {
	states map[string]*OrchestrationState
	mu     sync.RWMutex
}

// NewMemoryStateStorage creates an in-memory state storage
func NewMemoryStateStorage() *MemoryStateStorage {
	return &MemoryStateStorage{
		states: make(map[string]*OrchestrationState),
	}
}

// Store stores the orchestration state
func (s *MemoryStateStorage) Store(ctx context.Context, state *OrchestrationState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state.SessionName] = state
	return nil
}

// Get retrieves the orchestration state
func (s *MemoryStateStorage) Get(ctx context.Context, sessionName string) (*OrchestrationState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, exists := s.states[sessionName]
	if !exists {
		return nil, fmt.Errorf("state not found for session: %s", sessionName)
	}
	return state, nil
}

// UpdateAgentState updates a specific agent's state
func (s *MemoryStateStorage) UpdateAgentState(ctx context.Context, sessionName string, agentName string, agentState *AgentState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	state, exists := s.states[sessionName]
	if !exists {
		return fmt.Errorf("state not found for session: %s", sessionName)
	}
	
	state.AgentStates[agentName] = agentState
	state.UpdatedAt = time.Now()
	return nil
}

// Delete deletes the orchestration state
func (s *MemoryStateStorage) Delete(ctx context.Context, sessionName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, sessionName)
	return nil
}

// =============================================================================
// State Machine
// =============================================================================

// StateMachine manages state transitions
type StateMachine struct {
	currentPhase ExecutionPhase
	transitions  map[ExecutionPhase][]ExecutionPhase
	mu           sync.RWMutex
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		currentPhase: PhaseIdle,
		transitions: map[ExecutionPhase][]ExecutionPhase{
			PhaseIdle:        {PhasePlanning},
			PhasePlanning:    {PhaseSelecting, PhaseFailed},
			PhaseSelecting:   {PhaseExecuting, PhaseFailed},
			PhaseExecuting:   {PhaseAggregating, PhaseSuspended, PhaseFailed},
			PhaseSuspended:   {PhaseExecuting, PhaseFailed},
			PhaseAggregating: {PhaseCompleted, PhaseFailed},
			PhaseRetrying:    {PhasePlanning, PhaseFailed},
			PhaseCompleted:   {},
			PhaseFailed:      {PhaseRetrying}, // Allow retry
		},
	}
}

// CurrentPhase returns the current phase
func (sm *StateMachine) CurrentPhase() ExecutionPhase {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentPhase
}

// Transition transitions to a new phase
func (sm *StateMachine) Transition(newPhase ExecutionPhase) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	allowed, exists := sm.transitions[sm.currentPhase]
	if !exists {
		return fmt.Errorf("invalid current phase: %s", sm.currentPhase)
	}
	
	for _, phase := range allowed {
		if phase == newPhase {
			sm.currentPhase = newPhase
			return nil
		}
	}
	
	return fmt.Errorf("invalid transition from %s to %s", sm.currentPhase, newPhase)
}

// CanTransition checks if transition is valid
func (sm *StateMachine) CanTransition(newPhase ExecutionPhase) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	allowed, exists := sm.transitions[sm.currentPhase]
	if !exists {
		return false
	}
	
	for _, phase := range allowed {
		if phase == newPhase {
			return true
		}
	}
	return false
}

// Reset resets the state machine
func (sm *StateMachine) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentPhase = PhaseIdle
}

// =============================================================================
// Agent State Builder
// =============================================================================

// AgentStateBuilder builds agent states
type AgentStateBuilder struct {
	state *AgentState
}

// NewAgentStateBuilder creates a new agent state builder
func NewAgentStateBuilder() *AgentStateBuilder {
	return &AgentStateBuilder{
		state: &AgentState{
			StartTime: time.Now(),
		},
	}
}

// WithAgentName sets the agent name
func (b *AgentStateBuilder) WithAgentName(name string) *AgentStateBuilder {
	b.state.AgentName = name
	return b
}

// WithSubTaskName sets the sub-task name
func (b *AgentStateBuilder) WithSubTaskName(name string) *AgentStateBuilder {
	b.state.SubTaskName = name
	return b
}

// WithStatus sets the status
func (b *AgentStateBuilder) WithStatus(status AgentStatus) *AgentStateBuilder {
	b.state.Status = status
	return b
}

// WithFrozenState sets the frozen state
func (b *AgentStateBuilder) WithFrozenState(data []byte) *AgentStateBuilder {
	b.state.FrozenState = data
	return b
}

// WithPendingQuestion sets the pending question
func (b *AgentStateBuilder) WithPendingQuestion(q *PendingQuestion) *AgentStateBuilder {
	b.state.PendingQuestion = q
	return b
}

// WithResult sets the result
func (b *AgentStateBuilder) WithResult(result *SubResult) *AgentStateBuilder {
	b.state.Result = result
	return b
}

// Build builds the agent state
func (b *AgentStateBuilder) Build() *AgentState {
	if b.state.EndTime.IsZero() && (b.state.Status == AgentStatusCompleted || b.state.Status == AgentStatusFailed) {
		b.state.EndTime = time.Now()
	}
	return b.state
}

// =============================================================================
// State Serialization
// =============================================================================

// SerializeState serializes orchestration state to JSON
func SerializeState(state *OrchestrationState) ([]byte, error) {
	return json.Marshal(state)
}

// DeserializeState deserializes orchestration state from JSON
func DeserializeState(data []byte) (*OrchestrationState, error) {
	var state OrchestrationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// =============================================================================
// State Cloning
// =============================================================================

// CloneState creates a deep copy of orchestration state
func CloneState(state *OrchestrationState) (*OrchestrationState, error) {
	data, err := SerializeState(state)
	if err != nil {
		return nil, err
	}
	return DeserializeState(data)
}

// =============================================================================
// State Utilities
// =============================================================================

// IsTerminal checks if the phase is terminal
func IsTerminal(phase ExecutionPhase) bool {
	return phase == PhaseCompleted || phase == PhaseFailed
}

// IsRunning checks if the phase is running
func IsRunning(phase ExecutionPhase) bool {
	return phase == PhasePlanning || phase == PhaseSelecting || 
		phase == PhaseExecuting || phase == PhaseAggregating || phase == PhaseRetrying
}

// IsSuspended checks if the phase is suspended
func IsSuspended(phase ExecutionPhase) bool {
	return phase == PhaseSuspended
}

// GetAgentStatusSummary returns a summary of agent statuses
func GetAgentStatusSummary(state *OrchestrationState) map[AgentStatus]int {
	summary := make(map[AgentStatus]int)
	for _, agentState := range state.AgentStates {
		summary[agentState.Status]++
	}
	return summary
}

// GetProgress returns the progress percentage
func GetProgress(state *OrchestrationState) float64 {
	if state.Plan == nil || len(state.Plan.SubTasks) == 0 {
		return 0
	}
	
	total := len(state.Plan.SubTasks)
	completed := len(state.CompletedSubTasks)
	
	return float64(completed) / float64(total) * 100
}
