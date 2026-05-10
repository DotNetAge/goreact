//go:build linux

package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	globalSandboxApplier = linuxSandboxApplier
}

func linuxSandboxApplier(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	if config.Profile == ProfileUnconfined {
		return cmd
	}

	if isFirejailAvailable() {
		return applyFirejailSandbox(cmd, config)
	}

	if isBubblewrapAvailable() {
		return applyBubblewrapSandbox(cmd, config)
	}

	if isAppArmorAvailable() {
		return applyAppArmorSandbox(cmd, config)
	}

	if isSELinuxAvailable() {
		return applySELinuxSandbox(cmd, config)
	}

	return cmd
}

func isFirejailAvailable() bool {
	_, err := exec.LookPath("firejail")
	return err == nil
}

func isBubblewrapAvailable() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}

func isAppArmorAvailable() bool {
	_, err := os.Stat("/sys/module/apparmor")
	return err == nil
}

func isSELinuxAvailable() bool {
	_, err := os.Stat("/sys/fs/selinux")
	return err == nil
}

func applyFirejailSandbox(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	args := []string{"--quiet", "--nonewprivs"}

	switch config.Profile {
	case ProfileSandbox:
		args = append(args, "--net=none", "--seccomp", "--disable-mnt")
	case ProfileWorkspace:
		if !config.AllowNetwork {
			args = append(args, "--net=none")
		}
	}

	args = append(args, "--caps.drop=all")

	for _, path := range config.AllowedPaths {
		args = append(args, fmt.Sprintf("--whitelist=%s", filepath.Clean(path)))
	}

	if config.TempDir != "" {
		args = append(args, fmt.Sprintf("--whitelist=%s", filepath.Clean(config.TempDir)))
	}

	args = append(args, "--private-tmp")
	args = append(args, "--")
	args = append(args, cmd.Args...)

	newCmd := exec.Command("firejail", args...)
	newCmd.Stdout = cmd.Stdout
	newCmd.Stderr = cmd.Stderr
	newCmd.Stdin = cmd.Stdin
	newCmd.Dir = cmd.Dir
	newCmd.Env = filterSensitiveEnv(cmd.Env)
	newCmd.SysProcAttr = cmd.SysProcAttr

	return newCmd
}

func applyBubblewrapSandbox(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	args := []string{
		"--unshare-user",
		"--unshare-ipc",
		"--unshare-pid",
		"--unshare-uts",
		"--unshare-cgroup",
		"--die-with-parent",
		"--dev-bind", "/dev", "/dev",
		"--proc", "/proc",
	}

	if config.AllowNetwork {
		args = append(args, "--share-net")
	}

	args = append(args, "--ro-bind", "/usr", "/usr")
	args = append(args, "--ro-bind", "/lib", "/lib")
	args = append(args, "--ro-bind", "/lib64", "/lib64")

	for _, path := range config.AllowedPaths {
		cleanPath := filepath.Clean(path)
		args = append(args, "--bind", cleanPath, cleanPath)
	}

	if config.TempDir != "" {
		cleanTempDir := filepath.Clean(config.TempDir)
		args = append(args, "--bind", cleanTempDir, cleanTempDir)
	} else {
		args = append(args, "--tmpfs", "/tmp")
	}

	switch config.Profile {
	case ProfileSandbox:
		args = append(args, "--ro-bind", "/usr/bin", "/usr/bin")
		args = append(args, "--hide-pid")
	case ProfileWorkspace:
		args = append(args, "--ro-bind", "/usr/bin", "/usr/bin")
	}

	args = append(args, "--dir", "/run")
	args = append(args, "--symlink", "usr/lib", "/lib")
	args = append(args, "--")
	args = append(args, cmd.Args...)

	newCmd := exec.Command("bwrap", args...)
	newCmd.Stdout = cmd.Stdout
	newCmd.Stderr = cmd.Stderr
	newCmd.Stdin = cmd.Stdin
	newCmd.Dir = cmd.Dir
	newCmd.Env = filterSensitiveEnv(cmd.Env)
	newCmd.SysProcAttr = cmd.SysProcAttr

	return newCmd
}

func applyAppArmorSandbox(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	profile := generateAppArmorProfile(config)

	tempFile, err := writeProfileToTempLinux(profile)
	if err != nil {
		return cmd
	}

	aaExec := fmt.Sprintf("aa-exec -p %s", tempFile)
	args := []string{"-c", aaExec + " " + strings.Join(cmd.Args, " ")}

	newCmd := exec.Command("/bin/sh", args...)
	newCmd.Stdout = cmd.Stdout
	newCmd.Stderr = cmd.Stderr
	newCmd.Stdin = cmd.Stdin
	newCmd.Dir = cmd.Dir
	newCmd.Env = filterSensitiveEnv(cmd.Env)
	newCmd.SysProcAttr = cmd.SysProcAttr

	return newCmd
}

func applySELinuxSandbox(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	cmd.Env = filterSensitiveEnv(cmd.Env)

	if config.CustomPolicy != "" {
		return cmd
	}

	args := []string{}

	switch config.Profile {
	case ProfileSandbox:
		args = append(args, "runcon", "sandbox_t")
	case ProfileWorkspace:
		args = append(args, "runcon", "user_t")
	default:
		return cmd
	}

	args = append(args, cmd.Args...)

	newCmd := exec.Command(args[0], args[1:]...)
	newCmd.Stdout = cmd.Stdout
	newCmd.Stderr = cmd.Stderr
	newCmd.Stdin = cmd.Stdin
	newCmd.Dir = cmd.Dir
	newCmd.Env = cmd.Env
	newCmd.SysProcAttr = cmd.SysProcAttr

	return newCmd
}

func generateAppArmorProfile(config *SandboxConfig) string {
	if config.CustomPolicy != "" {
		return config.CustomPolicy
	}

	var sb strings.Builder

	sb.WriteString("#include <tunables/global>\n\n")
	sb.WriteString("profile goreact-sandbox flags=(attach_disconnected,mediate_deleted) {\n")
	sb.WriteString("  #include <abstractions/base>\n\n")

	switch config.Profile {
	case ProfileSandbox:
		if config.AllowNetwork {
			sb.WriteString("  network,\n")
		} else {
			sb.WriteString("  deny network,\n")
		}
	case ProfileWorkspace:
		if config.AllowNetwork {
			sb.WriteString("  network,\n")
		}
	}

	for _, path := range config.AllowedPaths {
		sb.WriteString(fmt.Sprintf("  %s/** rwmk,\n", filepath.Clean(path)))
	}

	if config.TempDir != "" {
		sb.WriteString(fmt.Sprintf("  %s/** rwmk,\n", filepath.Clean(config.TempDir)))
	}

	sb.WriteString("}\n")

	return sb.String()
}

func writeProfileToTempLinux(profile string) (string, error) {
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "goreact-sandbox-*.conf")
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
