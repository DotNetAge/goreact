package tool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WhitelistEntry represents an entry in the whitelist
type WhitelistEntry struct {
	ToolName     string    `json:"tool_name" yaml:"tool_name"`
	AuthorizedAt time.Time `json:"authorized_at" yaml:"authorized_at"`
	AuthorizedBy string    `json:"authorized_by,omitempty" yaml:"authorized_by,omitempty"`
	Permanent    bool      `json:"permanent" yaml:"permanent"`
	SessionID    string    `json:"session_id,omitempty" yaml:"session_id,omitempty"`
}

// WhitelistManager defines the interface for managing tool authorization whitelist
type WhitelistManager interface {
	// List returns all whitelist entries
	List() []WhitelistEntry
	// Revoke removes a tool from the whitelist
	Revoke(toolName string) error
	// RevokeAll clears all whitelist entries
	RevokeAll() error
	// IsAllowed checks if a tool is authorized
	IsAllowed(toolName string) bool
	// Add authorizes a tool
	Add(toolName string, permanent bool) error
}

// Whitelist manages tool authorization
type Whitelist struct {
	mu       sync.RWMutex
	entries  map[string]*WhitelistEntry
	filePath string
}

// NewWhitelist creates a new Whitelist
func NewWhitelist() *Whitelist {
	return &Whitelist{
		entries: make(map[string]*WhitelistEntry),
	}
}

// WithFile sets the file path for persistence
func (w *Whitelist) WithFile(path string) *Whitelist {
	w.filePath = path
	w.Load()
	return w
}

// Add adds a tool to the whitelist
func (w *Whitelist) Add(name string, permanent bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.entries[name] = &WhitelistEntry{
		ToolName:     name,
		AuthorizedAt: time.Now(),
		Permanent:    permanent,
	}
	
	return w.save()
}

// AddTemporary adds a tool for a single session
func (w *Whitelist) AddTemporary(name, sessionID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.entries[name] = &WhitelistEntry{
		ToolName:     name,
		AuthorizedAt: time.Now(),
		Permanent:    false,
		SessionID:    sessionID,
	}
	
	return w.save()
}

// Remove removes a tool from the whitelist
func (w *Whitelist) Remove(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	delete(w.entries, name)
	return w.save()
}

// IsAllowed checks if a tool is authorized
func (w *Whitelist) IsAllowed(name string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	entry, exists := w.entries[name]
	if !exists {
		return false
	}
	
	// Permanent authorization is always valid
	if entry.Permanent {
		return true
	}
	
	// Temporary authorization is valid for the session
	// (Session validity is checked elsewhere)
	return true
}

// Get returns the whitelist entry for a tool
func (w *Whitelist) Get(name string) (*WhitelistEntry, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	entry, exists := w.entries[name]
	return entry, exists
}

// List lists all whitelist entries
func (w *Whitelist) List() []*WhitelistEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	entries := make([]*WhitelistEntry, 0, len(w.entries))
	for _, entry := range w.entries {
		entries = append(entries, entry)
	}
	return entries
}

// ListPermanent lists permanent authorizations
func (w *Whitelist) ListPermanent() []*WhitelistEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	entries := []*WhitelistEntry{}
	for _, entry := range w.entries {
		if entry.Permanent {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Clear clears all whitelist entries
func (w *Whitelist) Clear() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.entries = make(map[string]*WhitelistEntry)
	return w.save()
}

// ClearTemporary clears temporary authorizations
func (w *Whitelist) ClearTemporary(sessionID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	for name, entry := range w.entries {
		if !entry.Permanent && (sessionID == "" || entry.SessionID == sessionID) {
			delete(w.entries, name)
		}
	}
	return w.save()
}

// Revoke removes a tool from the whitelist (implements WhitelistManager)
func (w *Whitelist) Revoke(toolName string) error {
	return w.Remove(toolName)
}

// RevokeAll clears all whitelist entries (implements WhitelistManager)
func (w *Whitelist) RevokeAll() error {
	return w.Clear()
}

// Load loads the whitelist from file
func (w *Whitelist) Load() error {
	if w.filePath == "" {
		return nil
	}
	
	data, err := os.ReadFile(w.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	var entries []*WhitelistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.entries = make(map[string]*WhitelistEntry)
	for _, entry := range entries {
		w.entries[entry.ToolName] = entry
	}
	
	return nil
}

// save saves the whitelist to file
func (w *Whitelist) save() error {
	if w.filePath == "" {
		return nil
	}
	
	entries := make([]*WhitelistEntry, 0, len(w.entries))
	for _, entry := range w.entries {
		entries = append(entries, entry)
	}
	
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	dir := filepath.Dir(w.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(w.filePath, data, 0644)
}
