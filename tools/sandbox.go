package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type SandboxProfile string

const (
	ProfileSandbox    SandboxProfile = "sandbox"
	ProfileWorkspace  SandboxProfile = "workspace"
	ProfileUnconfined SandboxProfile = "unconfined"
)

type SandboxConfig struct {
	Enabled      bool
	Profile      SandboxProfile
	AllowedPaths []string
	AllowNetwork bool
	TempDir      string
	CustomPolicy string
	mu           sync.RWMutex
}

func (c *SandboxConfig) GetEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Enabled
}

func (c *SandboxConfig) GetProfile() SandboxProfile {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Profile
}

func (c *SandboxConfig) GetAllowedPaths() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	paths := make([]string, len(c.AllowedPaths))
	copy(paths, c.AllowedPaths)
	return paths
}

func (c *SandboxConfig) GetAllowNetwork() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AllowNetwork
}

func (c *SandboxConfig) GetCustomPolicy() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CustomPolicy
}

func (c *SandboxConfig) Update(fn func(cfg *SandboxConfig)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(c)
}

func NewSandboxConfig(enabled bool, profile SandboxProfile, allowedPaths []string, allowNetwork bool, tempDir string) *SandboxConfig {
	return &SandboxConfig{
		Enabled:      enabled,
		Profile:      profile,
		AllowedPaths: allowedPaths,
		AllowNetwork: allowNetwork,
		TempDir:      tempDir,
	}
}

func DefaultSandboxConfig() *SandboxConfig {
	cwd, _ := os.Getwd()
	return &SandboxConfig{
		Enabled:      true,
		Profile:      ProfileWorkspace,
		AllowedPaths: []string{cwd},
		AllowNetwork: true,
		TempDir:      filepath.Join(os.TempDir(), "goreact-sandbox"),
	}
}

func UnrestrictedSandboxConfig() *SandboxConfig {
	return &SandboxConfig{
		Enabled: false,
		Profile: ProfileUnconfined,
	}
}

func RestrictedSandboxConfig(allowedPaths ...string) *SandboxConfig {
	cwd, _ := os.Getwd()
	if len(allowedPaths) == 0 {
		allowedPaths = []string{cwd}
	}
	return &SandboxConfig{
		Enabled:      true,
		Profile:      ProfileSandbox,
		AllowedPaths: allowedPaths,
		AllowNetwork: false,
		TempDir:      filepath.Join(os.TempDir(), "goreact-sandbox"),
	}
}

type SandboxApplier func(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd

var globalSandboxApplier SandboxApplier

// sandboxExecUnavailable is set to true on macOS if sandbox-exec is broken.
// On non-darwin platforms it defaults to false.
var sandboxExecUnavailable bool

func SetGlobalSandboxApplier(applier SandboxApplier) {
	globalSandboxApplier = applier
}

func GetGlobalSandboxApplier() SandboxApplier {
	return globalSandboxApplier
}

func ApplySandbox(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	if config == nil || !config.Enabled {
		return cmd
	}

	if globalSandboxApplier != nil {
		return globalSandboxApplier(cmd, config)
	}

	return cmd
}

func ensureTempDir(tempDir string) error {
	if tempDir == "" {
		return nil
	}
	return os.MkdirAll(tempDir, 0755)
}

var sensitiveEnvVars = map[string]bool{
	"AWS_SECRET_ACCESS_KEY": true,
	"AWS_ACCESS_KEY_ID":     true,
	"AWS_SESSION_TOKEN":     true,
	"GCP_SERVICE_ACCOUNT":   true,
	"GOOGLE_CREDENTIALS":    true,
	"AZURE_CLIENT_SECRET":   true,
	"GITHUB_TOKEN":          true,
	"PRIVATE_KEY":           true,
	"SSH_AUTH_SOCK":         true,
	"GPG_AGENT_INFO":        true,
}

func filterSensitiveEnv(env []string) []string {
	var filtered []string
	for _, e := range env {
		eqIdx := strings.Index(e, "=")
		if eqIdx < 0 {
			filtered = append(filtered, e)
			continue
		}
		key := e[:eqIdx]
		if sensitiveEnvVars[key] {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}
