package core

import (
	"encoding/json"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

// FrozenSession represents a frozen session state for pause-resume
type FrozenSession struct {
	// SessionName is the session name
	SessionName string `json:"session_name" yaml:"session_name"`
	
	// QuestionID is the pending question ID
	QuestionID string `json:"question_id" yaml:"question_id"`
	
	// StateData is the serialized state
	StateData []byte `json:"state_data" yaml:"state_data"`
	
	// CreatedAt is the creation timestamp
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	
	// ExpiresAt is the expiration timestamp
	ExpiresAt time.Time `json:"expires_at" yaml:"expires_at"`
	
	// Status is the frozen status
	Status common.FrozenStatus `json:"status" yaml:"status"`
}

// NewFrozenSession creates a new FrozenSession
func NewFrozenSession(sessionName, questionID string, state *State) (*FrozenSession, error) {
	stateData, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	
	return &FrozenSession{
		SessionName: sessionName,
		QuestionID:  questionID,
		StateData:   stateData,
		CreatedAt:   time.Now(),
		Status:      common.FrozenStatusFrozen,
	}, nil
}

// WithExpiry sets the expiration time
func (f *FrozenSession) WithExpiry(duration time.Duration) *FrozenSession {
	f.ExpiresAt = time.Now().Add(duration)
	return f
}

// IsExpired checks if the session is expired
func (f *FrozenSession) IsExpired() bool {
	if f.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(f.ExpiresAt)
}

// Thaw deserializes the state
func (f *FrozenSession) Thaw() (*State, error) {
	var state State
	if err := json.Unmarshal(f.StateData, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// MarkResumed marks the session as resumed
func (f *FrozenSession) MarkResumed() {
	f.Status = common.FrozenStatusResumed
}

// MarkExpired marks the session as expired
func (f *FrozenSession) MarkExpired() {
	f.Status = common.FrozenStatusExpired
}

// Cancel marks the session as canceled
func (f *FrozenSession) Cancel() {
	f.Status = common.FrozenStatusCanceled
}

// IsFrozen checks if the session is still frozen
func (f *FrozenSession) IsFrozen() bool {
	return f.Status == common.FrozenStatusFrozen && !f.IsExpired()
}
