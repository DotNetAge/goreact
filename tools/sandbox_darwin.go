//go:build darwin

package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	globalSandboxApplier = darwinSandboxApplier
	testCmd := exec.Command("sandbox-exec", "-p", "(version 1)(allow default)", "true")
	if err := testCmd.Run(); err != nil {
		sandboxExecUnavailable = true
	}
}

func darwinSandboxApplier(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	if sandboxExecUnavailable || !config.Enabled || config.Profile == ProfileUnconfined {
		return cmd
	}
	profile := generateSeatbeltProfile(config)

	tempFile, err := writeProfileToTemp(profile)
	if err != nil {
		return cmd
	}

	originalCmd := cmd.Path
	originalArgs := cmd.Args[1:]

	newArgs := append([]string{"-f", tempFile, originalCmd}, originalArgs...)

	newCmd := exec.Command("sandbox-exec", newArgs...)
	newCmd.Stdout = cmd.Stdout
	newCmd.Stderr = cmd.Stderr
	newCmd.Stdin = cmd.Stdin
	newCmd.Dir = cmd.Dir
	newCmd.Env = cmd.Env
	newCmd.SysProcAttr = cmd.SysProcAttr

	return newCmd
}

func applyPlatformSandbox(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	return darwinSandboxApplier(cmd, config)
}

func generateSeatbeltProfile(config *SandboxConfig) string {
	if config.CustomPolicy != "" {
		return config.CustomPolicy
	}

	var sb strings.Builder

	sb.WriteString("(version 1)\n")

	switch config.Profile {
	case ProfileSandbox:
		sb.WriteString(generateStrictProfile(config))
	case ProfileWorkspace:
		sb.WriteString(generateWorkspaceProfile(config))
	case ProfileUnconfined:
		sb.WriteString(generateUnconfinedProfile())
	default:
		sb.WriteString(generateWorkspaceProfile(config))
	}

	return sb.String()
}

func generateStrictProfile(config *SandboxConfig) string {
	var sb strings.Builder

	sb.WriteString("(deny default)\n")

	sb.WriteString("(allow file-read*\n")
	sb.WriteString("  (subpath \"/usr/lib\")\n")
	sb.WriteString("  (subpath \"/System/Library\")\n")
	sb.WriteString("  (subpath \"/Library\")\n")
	sb.WriteString("  (subpath \"/private/var/db/timezone\")\n")
	sb.WriteString(")\n")

	sb.WriteString("(allow file-read*\n")
	sb.WriteString("  (subpath \"/bin\")\n")
	sb.WriteString("  (subpath \"/usr/bin\")\n")
	sb.WriteString("  (subpath \"/usr/local/bin\")\n")
	sb.WriteString("  (subpath \"/opt/homebrew/bin\")\n")
	sb.WriteString("  (subpath \"/usr/sbin\")\n")
	sb.WriteString(")\n")

	for _, path := range config.AllowedPaths {
		cleanPath := filepath.Clean(path)
		sb.WriteString(fmt.Sprintf("(allow file-read*\n"))
		sb.WriteString(fmt.Sprintf("  (subpath %q))\n", cleanPath))
	}

	if config.TempDir != "" {
		sb.WriteString(fmt.Sprintf("(allow file-read* file-write*\n"))
		sb.WriteString(fmt.Sprintf("  (subpath %q))\n", config.TempDir))
	}

	if config.AllowNetwork {
		sb.WriteString("(allow network*)\n")
	} else {
		sb.WriteString("(deny network*)\n")
	}

	sb.WriteString("(allow process-exec\n")
	sb.WriteString("  (subpath \"/bin\")\n")
	sb.WriteString("  (subpath \"/usr/bin\")\n")
	sb.WriteString("  (subpath \"/usr/local/bin\")\n")
	sb.WriteString(")\n")

	sb.WriteString("(allow sysctl-read)\n")

	sb.WriteString("(deny file-write*\n")
	sb.WriteString("  (subpath \"/\")\n")
	sb.WriteString("  (except (subpath \"/private/tmp\"))\n")
	sb.WriteString(")\n")

	return sb.String()
}

func generateWorkspaceProfile(config *SandboxConfig) string {
	var sb strings.Builder

	sb.WriteString("(deny default)\n")

	sb.WriteString("(allow file-read*\n")
	sb.WriteString("  (subpath \"/usr/lib\")\n")
	sb.WriteString("  (subpath \"/System/Library\")\n")
	sb.WriteString("  (subpath \"/Library\")\n")
	sb.WriteString("  (subpath \"/private\")\n")
	sb.WriteString(")\n")

	sb.WriteString("(allow file-read*\n")
	sb.WriteString("  (subpath \"/bin\")\n")
	sb.WriteString("  (subpath \"/usr/bin\")\n")
	sb.WriteString("  (subpath \"/usr/local/bin\")\n")
	sb.WriteString("  (subpath \"/opt/homebrew/bin\")\n")
	sb.WriteString(")\n")

	for _, path := range config.AllowedPaths {
		cleanPath := filepath.Clean(path)
		sb.WriteString(fmt.Sprintf("(allow file-read* file-write*\n"))
		sb.WriteString(fmt.Sprintf("  (subpath %q))\n", cleanPath))
	}

	if config.TempDir != "" {
		sb.WriteString(fmt.Sprintf("(allow file-read* file-write*\n"))
		sb.WriteString(fmt.Sprintf("  (subpath %q))\n", config.TempDir))
	}

	sb.WriteString(fmt.Sprintf("(allow file-read* file-write*\n"))
	sb.WriteString(fmt.Sprintf("  (subpath %q))\n", os.TempDir()))

	if config.AllowNetwork {
		sb.WriteString("(allow network*)\n")
	} else {
		sb.WriteString("(deny network*)\n")
	}

	sb.WriteString("(allow process-exec\n")
	sb.WriteString("  (subpath \"/bin\")\n")
	sb.WriteString("  (subpath \"/usr/bin\")\n")
	sb.WriteString("  (subpath \"/usr/local/bin\")\n")
	sb.WriteString("  (subpath \"/opt/homebrew/bin\")\n")
	sb.WriteString(")\n")

	sb.WriteString("(allow sysctl*)\n")

	sb.WriteString("(deny file-write*\n")
	sb.WriteString("  (subpath \"/tmp\")\n")
	sb.WriteString("  (subpath \"/var/tmp\")\n")
	sb.WriteString(")\n")

	return sb.String()
}

func generateUnconfinedProfile() string {
	return "(version 1)\n(allow default)\n"
}

func writeProfileToTemp(profile string) (string, error) {
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "goreact-sandbox-*.sb")
	if err != nil {
		return "", fmt.Errorf("failed to create temp profile: %w", err)
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(profile)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write profile: %w", err)
	}

	return tmpFile.Name(), nil
}
