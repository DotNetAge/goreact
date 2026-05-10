package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type SessionSandboxManager struct {
	mu            sync.RWMutex
	sessions      map[string]*SandboxConfig
	defaultConfig *SandboxConfig
}

func NewSessionSandboxManager(defaultConfig ...*SandboxConfig) *SessionSandboxManager {
	cfg := DefaultSandboxConfig()
	if len(defaultConfig) > 0 {
		cfg = defaultConfig[0]
	}

	cfg.TempDir = filepath.Join(cfg.TempDir, generateSessionTempDir())

	return &SessionSandboxManager{
		sessions:      make(map[string]*SandboxConfig),
		defaultConfig: cfg,
	}
}

func (m *SessionSandboxManager) GetConfig(sessionID string) *SandboxConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if cfg, ok := m.sessions[sessionID]; ok {
		tempDir := cfg.TempDir
		if tempDir == "" {
			tempDir = m.defaultConfig.TempDir
		}
		return &SandboxConfig{
			Enabled:      cfg.Enabled,
			Profile:      cfg.Profile,
			AllowedPaths: cfg.AllowedPaths,
			AllowNetwork: cfg.AllowNetwork,
			CustomPolicy: cfg.CustomPolicy,
			TempDir:      tempDir,
		}
	}

	if m.defaultConfig.TempDir != "" {
		return &SandboxConfig{
			Enabled:      m.defaultConfig.Enabled,
			Profile:      m.defaultConfig.Profile,
			AllowedPaths: m.defaultConfig.AllowedPaths,
			AllowNetwork: m.defaultConfig.AllowNetwork,
			CustomPolicy: m.defaultConfig.CustomPolicy,
			TempDir:      filepath.Join(m.defaultConfig.TempDir, sessionID),
		}
	}
	return &SandboxConfig{
		Enabled:      m.defaultConfig.Enabled,
		Profile:      m.defaultConfig.Profile,
		AllowedPaths: m.defaultConfig.AllowedPaths,
		AllowNetwork: m.defaultConfig.AllowNetwork,
		CustomPolicy: m.defaultConfig.CustomPolicy,
		TempDir:      m.defaultConfig.TempDir,
	}
}

func (m *SessionSandboxManager) SetConfig(sessionID string, config *SandboxConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tempDir := config.TempDir
	if tempDir == "" && m.defaultConfig.TempDir != "" {
		tempDir = filepath.Join(m.defaultConfig.TempDir, sessionID)
	}

	cfgCopy := &SandboxConfig{
		Enabled:      config.Enabled,
		Profile:      config.Profile,
		AllowedPaths: config.AllowedPaths,
		AllowNetwork: config.AllowNetwork,
		CustomPolicy: config.CustomPolicy,
		TempDir:      tempDir,
	}
	m.sessions[sessionID] = cfgCopy

	if cfgCopy.TempDir != "" {
		os.MkdirAll(cfgCopy.TempDir, 0755)
	}
}

func (m *SessionSandboxManager) UpdateConfig(sessionID string, fn func(cfg *SandboxConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, ok := m.sessions[sessionID]
	if !ok {
		newCfg := &SandboxConfig{
			Enabled:      m.defaultConfig.Enabled,
			Profile:      m.defaultConfig.Profile,
			AllowedPaths: m.defaultConfig.AllowedPaths,
			AllowNetwork: m.defaultConfig.AllowNetwork,
			CustomPolicy: m.defaultConfig.CustomPolicy,
			TempDir:      filepath.Join(m.defaultConfig.TempDir, sessionID),
		}
		cfg = newCfg
		m.sessions[sessionID] = cfg
	}

	fn(cfg)

	if cfg.TempDir != "" {
		os.MkdirAll(cfg.TempDir, 0755)
	}
}

func (m *SessionSandboxManager) RemoveSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, ok := m.sessions[sessionID]
	if ok && cfg.TempDir != "" {
		os.RemoveAll(cfg.TempDir)
	}
	delete(m.sessions, sessionID)
}

func (m *SessionSandboxManager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cfg := range m.sessions {
		if cfg.TempDir != "" {
			os.RemoveAll(cfg.TempDir)
		}
	}
	m.sessions = make(map[string]*SandboxConfig)
}

func (m *SessionSandboxManager) ListSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

func (m *SessionSandboxManager) ApplyToCommand(cmd *exec.Cmd, sessionID string) *exec.Cmd {
	config := m.GetConfig(sessionID)
	return ApplySandbox(cmd, config)
}

func (m *SessionSandboxManager) CleanupSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, ok := m.sessions[sessionID]
	if ok && cfg.TempDir != "" {
		os.RemoveAll(cfg.TempDir)
	}
	delete(m.sessions, sessionID)
}

type SessionContextKey struct{}

func ExtractSessionID(ctx context.Context) string {
	if sessionID, ok := ctx.Value(SessionContextKey{}).(string); ok {
		return sessionID
	}
	return ""
}

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionContextKey{}, sessionID)
}

func GenerateSessionTempDir(sessionID string) string {
	baseTemp := os.TempDir()
	return filepath.Join(baseTemp, "goreact-sandbox", sessionID)
}

func generateSessionTempDir() string {
	return "default"
}

func CleanupSandboxTemp() {
	baseTemp := filepath.Join(os.TempDir(), "goreact-sandbox")
	if _, err := os.Stat(baseTemp); err == nil {
		entries, err := os.ReadDir(baseTemp)
		if err != nil {
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				fullPath := filepath.Join(baseTemp, entry.Name())
				if isStaleSession(fullPath) {
					os.RemoveAll(fullPath)
				}
			}
		}
	}
}

func isStaleSession(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.ModTime().Add(24 * 60 * 60 * time.Second).Before(time.Now())
}
