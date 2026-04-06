package orchestration

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Snapshot Manager Implementation
// =============================================================================

// DefaultSnapshotManager implements SnapshotManager
type DefaultSnapshotManager struct {
	signer *SnapshotSigner
}

// NewDefaultSnapshotManager creates a default snapshot manager
func NewDefaultSnapshotManager() *DefaultSnapshotManager {
	return &DefaultSnapshotManager{
		signer: NewSnapshotSigner(),
	}
}

// CreateSnapshot creates a snapshot at the specified level
func (m *DefaultSnapshotManager) CreateSnapshot(state *OrchestrationState, level SnapshotLevel) ([]byte, error) {
	var snapshot any
	
	switch level {
	case SnapshotLevelOrchestration:
		snapshot = &OrchestrationSnapshot{
			SessionName:      state.SessionName,
			Plan:             state.Plan,
			AgentStates:      state.AgentStates,
			ExecutionPhase:   state.ExecutionPhase,
			PendingQuestions: state.PendingQuestions,
			CreatedAt:        time.Now(),
		}
	case SnapshotLevelAgent:
		// Create individual agent snapshots
		snapshots := make([]*AgentSnapshot, 0)
		for name, agentState := range state.AgentStates {
			snapshots = append(snapshots, &AgentSnapshot{
				AgentName:   name,
				SessionName: state.SessionName,
				FrozenState: agentState.FrozenState,
				PendingQuestion: agentState.PendingQuestion,
				CreatedAt:   time.Now(),
			})
		}
		snapshot = snapshots
	default:
		snapshot = state
	}
	
	// Serialize
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot: %w", err)
	}
	
	return data, nil
}

// RestoreSnapshot restores state from a snapshot
func (m *DefaultSnapshotManager) RestoreSnapshot(data []byte) (*OrchestrationState, error) {
	var state OrchestrationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}
	return &state, nil
}

// Sign signs a snapshot
func (m *DefaultSnapshotManager) Sign(data []byte) (*SignedSnapshot, error) {
	return m.signer.Sign(data)
}

// Verify verifies a signed snapshot
func (m *DefaultSnapshotManager) Verify(signed *SignedSnapshot) ([]byte, error) {
	return m.signer.Verify(signed)
}

// =============================================================================
// Snapshot Signer
// =============================================================================

// SnapshotSigner handles snapshot signing and verification
type SnapshotSigner struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	keyID      string
}

// NewSnapshotSigner creates a new snapshot signer
func NewSnapshotSigner() *SnapshotSigner {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		// Fallback to deterministic keys for development
		seed := make([]byte, ed25519.SeedSize)
		privateKey = ed25519.NewKeyFromSeed(seed)
		publicKey = privateKey.Public().(ed25519.PublicKey)
	}
	
	return &SnapshotSigner{
		privateKey: privateKey,
		publicKey:  publicKey,
		keyID:      generateKeyID(publicKey),
	}
}

// NewSnapshotSignerWithKeys creates a signer with existing keys
func NewSnapshotSignerWithKeys(privateKey ed25519.PrivateKey) *SnapshotSigner {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return &SnapshotSigner{
		privateKey: privateKey,
		publicKey:  publicKey,
		keyID:      generateKeyID(publicKey),
	}
}

// Sign signs data with Ed25519
func (s *SnapshotSigner) Sign(data []byte) (*SignedSnapshot, error) {
	signature := ed25519.Sign(s.privateKey, data)
	
	return &SignedSnapshot{
		Content:   data,
		Signature: signature,
		Algorithm: "Ed25519",
		KeyID:     s.keyID,
		Timestamp: time.Now().Unix(),
	}, nil
}

// Verify verifies a signed snapshot
func (s *SnapshotSigner) Verify(signed *SignedSnapshot) ([]byte, error) {
	// Check algorithm
	if signed.Algorithm != "Ed25519" {
		return nil, fmt.Errorf("unsupported signature algorithm: %s", signed.Algorithm)
	}
	
	// Verify signature
	if !ed25519.Verify(s.publicKey, signed.Content, signed.Signature) {
		return nil, fmt.Errorf("signature verification failed")
	}
	
	// Check timestamp (allow 24 hours clock skew)
	now := time.Now().Unix()
	if signed.Timestamp > now+86400 || signed.Timestamp < now-86400 {
		return nil, fmt.Errorf("signature timestamp out of acceptable range")
	}
	
	return signed.Content, nil
}

// PublicKey returns the public key
func (s *SnapshotSigner) PublicKey() ed25519.PublicKey {
	return s.publicKey
}

// KeyID returns the key ID
func (s *SnapshotSigner) KeyID() string {
	return s.keyID
}

// generateKeyID generates a key ID from public key
func generateKeyID(publicKey ed25519.PublicKey) string {
	hash := sha256.Sum256(publicKey)
	return hex.EncodeToString(hash[:8])
}

// =============================================================================
// Checksum Calculator
// =============================================================================

// ChecksumCalculator calculates checksums for data integrity
type ChecksumCalculator struct{}

// NewChecksumCalculator creates a new checksum calculator
func NewChecksumCalculator() *ChecksumCalculator {
	return &ChecksumCalculator{}
}

// Calculate calculates a SHA256 checksum
func (c *ChecksumCalculator) Calculate(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Verify verifies data against a checksum
func (c *ChecksumCalculator) Verify(data []byte, checksum string) bool {
	return c.Calculate(data) == checksum
}

// =============================================================================
// Snapshot Store
// =============================================================================

// SnapshotStore stores snapshots
type SnapshotStore struct {
	snapshots map[string][]byte
	checksums map[string]string
	mu        sync.RWMutex
	calc      *ChecksumCalculator
}

// NewSnapshotStore creates a new snapshot store
func NewSnapshotStore() *SnapshotStore {
	return &SnapshotStore{
		snapshots: make(map[string][]byte),
		checksums: make(map[string]string),
		calc:      NewChecksumCalculator(),
	}
}

// Store stores a snapshot
func (s *SnapshotStore) Store(sessionName string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.snapshots[sessionName] = data
	s.checksums[sessionName] = s.calc.Calculate(data)
	return nil
}

// Get retrieves a snapshot
func (s *SnapshotStore) Get(sessionName string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, exists := s.snapshots[sessionName]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", sessionName)
	}
	
	// Verify checksum
	checksum := s.checksums[sessionName]
	if !s.calc.Verify(data, checksum) {
		return nil, fmt.Errorf("snapshot checksum verification failed")
	}
	
	return data, nil
}

// Delete deletes a snapshot
func (s *SnapshotStore) Delete(sessionName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.snapshots, sessionName)
	delete(s.checksums, sessionName)
}

// List lists all snapshot names
func (s *SnapshotStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	names := make([]string, 0, len(s.snapshots))
	for name := range s.snapshots {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// Snapshot Compression (Placeholder)
// =============================================================================

// SnapshotCompressor compresses snapshots
type SnapshotCompressor struct {
	level int
}

// NewSnapshotCompressor creates a new snapshot compressor
func NewSnapshotCompressor(level int) *SnapshotCompressor {
	return &SnapshotCompressor{level: level}
}

// Compress compresses data (placeholder - would use compression library)
func (c *SnapshotCompressor) Compress(data []byte) ([]byte, error) {
	// Placeholder - would implement actual compression
	return data, nil
}

// Decompress decompresses data (placeholder)
func (c *SnapshotCompressor) Decompress(data []byte) ([]byte, error) {
	// Placeholder
	return data, nil
}

// =============================================================================
// Key Rotation
// =============================================================================

// KeyRotationManager manages key rotation for snapshot signing
type KeyRotationManager struct {
	currentSigner  *SnapshotSigner
	previousSigner *SnapshotSigner
	rotationPeriod time.Duration
	lastRotation   time.Time
	mu             sync.RWMutex
}

// NewKeyRotationManager creates a new key rotation manager
func NewKeyRotationManager(rotationPeriod time.Duration) *KeyRotationManager {
	return &KeyRotationManager{
		currentSigner:  NewSnapshotSigner(),
		rotationPeriod: rotationPeriod,
		lastRotation:   time.Now(),
	}
}

// GetCurrentSigner gets the current signer
func (m *KeyRotationManager) GetCurrentSigner() *SnapshotSigner {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSigner
}

// Rotate rotates the signing key
func (m *KeyRotationManager) Rotate() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if rotation is needed
	if time.Since(m.lastRotation) < m.rotationPeriod {
		return nil
	}
	
	// Rotate keys
	m.previousSigner = m.currentSigner
	m.currentSigner = NewSnapshotSigner()
	m.lastRotation = time.Now()
	
	return nil
}

// VerifyWithFallback verifies with current or previous key
func (m *KeyRotationManager) VerifyWithFallback(signed *SignedSnapshot) ([]byte, error) {
	// Try current key first
	data, err := m.currentSigner.Verify(signed)
	if err == nil {
		return data, nil
	}
	
	// Try previous key if available
	m.mu.RLock()
	previous := m.previousSigner
	m.mu.RUnlock()
	
	if previous != nil {
		return previous.Verify(signed)
	}
	
	return nil, fmt.Errorf("signature verification failed with all keys")
}
