//go:build windows

package tools

import (
	"os/exec"
	"syscall"
)

func init() {
	globalSandboxApplier = windowsSandboxApplier
}

func windowsSandboxApplier(cmd *exec.Cmd, config *SandboxConfig) *exec.Cmd {
	if config.Profile == ProfileUnconfined {
		return cmd
	}

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	switch config.Profile {
	case ProfileSandbox:
		applyWindowsStrictIsolation(cmd, config)
	case ProfileWorkspace:
		applyWindowsWorkspaceIsolation(cmd, config)
	}

	cmd.Env = filterSensitiveEnv(cmd.Env)

	return cmd
}

func applyWindowsStrictIsolation(cmd *exec.Cmd, config *SandboxConfig) {
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP

	if config.TempDir != "" {
		cmd.Env = append(cmd.Env, "TEMP="+config.TempDir)
		cmd.Env = append(cmd.Env, "TMP="+config.TempDir)
		cmd.Env = append(cmd.Env, "LOCALAPPDATA="+config.TempDir)
	}
}

func applyWindowsWorkspaceIsolation(cmd *exec.Cmd, config *SandboxConfig) {
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP

	if config.TempDir != "" {
		cmd.Env = append(cmd.Env, "TEMP="+config.TempDir)
		cmd.Env = append(cmd.Env, "TMP="+config.TempDir)
	}
}
