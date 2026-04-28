package tools

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// ---------------------------------------------------------------------------
// scriptResult — internal execution result
// ---------------------------------------------------------------------------

type scriptResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Duration string `json:"duration"`
}

// ---------------------------------------------------------------------------
// scriptExecutor — internal interface for script execution strategies
// ---------------------------------------------------------------------------

type scriptExecutor interface {
	Execute(ctx context.Context, skillRoot, scriptPath string, args []string) (*scriptResult, error)
}

// ---------------------------------------------------------------------------
// defaultScriptExecutor — Python venv + Shell execution
// ---------------------------------------------------------------------------

type defaultScriptExecutor struct {
	mu           sync.Mutex
	venvManagers map[string]*venvManager
}

func newDefaultScriptExecutor() *defaultScriptExecutor {
	return &defaultScriptExecutor{
		venvManagers: make(map[string]*venvManager),
	}
}

func (e *defaultScriptExecutor) Execute(ctx context.Context, skillRoot, scriptPath string, args []string) (*scriptResult, error) {
	skillRoot = filepath.Clean(skillRoot)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found: %s", scriptPath)
	}

	ext := strings.ToLower(filepath.Ext(scriptPath))
	switch ext {
	case ".py":
		return e.executePython(ctx, skillRoot, scriptPath, args)
	case ".sh", ".bash", ".zsh":
		return e.executeShell(ctx, skillRoot, scriptPath, args)
	default:
		return e.executeGeneric(ctx, skillRoot, scriptPath, args)
	}
}

func (e *defaultScriptExecutor) executePython(ctx context.Context, skillRoot, scriptPath string, args []string) (*scriptResult, error) {
	key := venvKey(skillRoot)

	e.mu.Lock()
	vm, ok := e.venvManagers[key]
	if !ok {
		vm = newVenvManager(skillRoot)
		e.venvManagers[key] = vm
	}
	e.mu.Unlock()

	if err := vm.ensureVenv(ctx); err != nil {
		return nil, fmt.Errorf("failed to setup python environment: %w", err)
	}

	pythonBin := filepath.Join(vm.venvPath, "bin", "python")
	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		pythonBin = filepath.Join(vm.venvPath, "Scripts", "python.exe")
	}

	absScript, _ := filepath.Abs(scriptPath)
	fullArgs := append([]string{absScript}, args...)
	cmd := exec.CommandContext(ctx, pythonBin, fullArgs...)
	cmd.Dir = skillRoot

	start := time.Now()
	stdout, err := cmd.Output()
	duration := time.Since(start).String()

	if exitErr, ok := err.(*exec.ExitError); ok {
		return &scriptResult{
			ExitCode: exitErr.ExitCode(),
			Stdout:    string(stdout),
			Stderr:    string(exitErr.Stderr),
			Duration:  duration,
		}, nil
	} else if err != nil {
		return nil, err
	}

	return &scriptResult{
		ExitCode: 0,
		Stdout:   string(stdout),
		Duration: duration,
	}, nil
}

func (e *defaultScriptExecutor) executeShell(ctx context.Context, skillRoot, scriptPath string, args []string) (*scriptResult, error) {
	cmd := exec.CommandContext(ctx, "/bin/bash", scriptPath)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = skillRoot

	start := time.Now()
	stdout, err := cmd.Output()
	duration := time.Since(start).String()

	if exitErr, ok := err.(*exec.ExitError); ok {
		return &scriptResult{
			ExitCode: exitErr.ExitCode(),
			Stdout:    string(stdout),
			Stderr:    string(exitErr.Stderr),
			Duration:  duration,
		}, nil
	} else if err != nil {
		return nil, err
	}

	return &scriptResult{ExitCode: 0, Stdout: string(stdout), Duration: duration}, nil
}

func (e *defaultScriptExecutor) executeGeneric(ctx context.Context, skillRoot, scriptPath string, args []string) (*scriptResult, error) {
	cmd := exec.CommandContext(ctx, scriptPath, args...)
	cmd.Dir = skillRoot

	start := time.Now()
	stdout, err := cmd.Output()
	duration := time.Since(start).String()

	if exitErr, ok := err.(*exec.ExitError); ok {
		return &scriptResult{
			ExitCode: exitErr.ExitCode(),
			Stdout:    string(stdout),
			Stderr:    string(exitErr.Stderr),
			Duration:  duration,
		}, nil
	} else if err != nil {
		return nil, err
	}

	return &scriptResult{ExitCode: 0, Stdout: string(stdout), Duration: duration}, nil
}

// ---------------------------------------------------------------------------
// venvManager — per-skill Python virtual environment
// ---------------------------------------------------------------------------

type venvManager struct {
	skillRoot string
	venvPath  string
	reqHash   string
	once      sync.Once
}

func newVenvManager(skillRoot string) *venvManager {
	return &venvManager{
		skillRoot: skillRoot,
		venvPath:  filepath.Join(skillRoot, ".venv"),
	}
}

func (m *venvManager) ensureVenv(ctx context.Context) error {
	var initErr error
	m.once.Do(func() {
		initErr = m.initOnce(ctx)
	})
	return initErr
}

func (m *venvManager) initOnce(ctx context.Context) error {
	if _, err := os.Stat(m.venvPath); os.IsNotExist(err) {
		if err := m.createVenv(); err != nil {
			return fmt.Errorf("create venv: %w", err)
		}
	}

	reqFile := filepath.Join(m.skillRoot, "scripts", "requirements.txt")
	if _, err := os.Stat(reqFile); os.IsNotExist(err) {
		return nil
	}

	currentHash, _ := hashFile(reqFile)
	if currentHash == m.reqHash && dirExists(m.venvPath) {
		return nil
	}

	if err := m.installRequirements(ctx, reqFile); err != nil {
		return fmt.Errorf("install requirements: %w", err)
	}
	m.reqHash = currentHash
	return nil
}

func (m *venvManager) createVenv() error {
	pythonCmd := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		pythonCmd = "python"
	}

	cmd := exec.Command(pythonCmd, "-m", "venv", m.venvPath)
	cmd.Dir = m.skillRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("venv creation failed (%s): %w", out, err)
	}
	return nil
}

func (m *venvManager) installRequirements(ctx context.Context, reqFile string) error {
	pipBin := filepath.Join(m.venvPath, "bin", "pip")
	if _, err := os.Stat(pipBin); os.IsNotExist(err) {
		pipBin = filepath.Join(m.venvPath, "Scripts", "pip.exe")
	}

	cmd := exec.CommandContext(ctx, pipBin, "install", "-r", reqFile)
	cmd.Dir = m.skillRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pip install failed (%s): %w", out, err)
	}
	return nil
}

func venvKey(dir string) string {
	h := sha256.Sum256([]byte(dir))
	return fmt.Sprintf("%x", h)[:16]
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ---------------------------------------------------------------------------
// RunScript — the Tool
// ---------------------------------------------------------------------------

type RunScript struct {
	info           *core.ToolInfo
	scriptExecutor scriptExecutor
}

func NewRunScriptTool() core.FuncTool {
	return &RunScript{
		info: &core.ToolInfo{
			Name:        "run_script",
			Description: `Execute a script file from the current skill's scripts/ directory. Supports Python (.py), Shell (.sh), Node (.js), Ruby (.rb), and other interpreters. For Python scripts, automatically manages virtual environments and dependencies from requirements.txt.

Usage examples (pass the command as-is from Skill instructions):
- "python scripts/analyze.py --input data.json"
- "./scripts/build.sh --target release"
- "scripts/fetch_data arg1 arg2"

The tool auto-detects the language from the command string and routes to the appropriate executor.`,
			Tags:          []string{"script", "execute", "python", "shell", "skill"},
			SecurityLevel: core.LevelSensitive,
			Parameters: []core.Parameter{
				{
					Name:        "command",
					Type:        "string",
					Description: "The script invocation command exactly as specified in Skill instructions. Include interpreter name if needed (e.g., 'python scripts/foo.py').",
					Required:    true,
				},
				{
					Name:        "working_dir",
					Type:        "string",
					Description: "Working directory for script execution. Defaults to the current directory. Usually the {base_dir} of the active skill.",
					Required:    false,
				},
				{
					Name:        "args",
					Type:        "array",
					Description: "Additional arguments to pass to the script.",
					Required:    false,
				},
			},
		},
		scriptExecutor: newDefaultScriptExecutor(),
	}
}

func (t *RunScript) Info() *core.ToolInfo {
	return t.info
}

func (t *RunScript) Execute(ctx context.Context, params map[string]any) (any, error) {
	command, ok := params["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("missing required parameter: command")
	}

	workingDir, _ := params["working_dir"].(string)
	if workingDir == "" {
		workingDir = "."
	}
	workingDir = filepath.Clean(workingDir)

	language, scriptPath := parseCommand(command, workingDir)
	if scriptPath == "" {
		return nil, fmt.Errorf("could not extract script path from command: %q", command)
	}

	var args []string
	if rawArgs, ok := params["args"].([]any); ok {
		for _, a := range rawArgs {
			if s, ok := a.(string); ok {
				args = append(args, s)
			}
		}
	} else if rawArgs, ok := params["args"].([]string); ok {
		args = rawArgs
	}

	result, err := t.scriptExecutor.Execute(ctx, workingDir, scriptPath, args)
	if err != nil {
		return nil, err
	}
	return formatScriptResult(language, scriptPath, result), nil
}

// ---------------------------------------------------------------------------
// Command parsing
// ---------------------------------------------------------------------------

func parseCommand(command, baseDir string) (language, scriptPath string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", ""
	}

	interpreters := map[string]bool{
		"python": true, "python3": true, "pypy": true,
		"node": true, "nodejs": true,
		"ruby": true,
		"perl": true,
		"php": true,
		"bash": true, "sh": true, "zsh": true,
	}

	if len(parts) > 1 && interpreters[parts[0]] {
		language = parts[0]
		candidate := parts[1]
		if filepath.IsAbs(candidate) {
			scriptPath = candidate
		} else {
			scriptPath = filepath.Join(baseDir, candidate)
		}
		return
	}

	candidate := parts[0]
	if filepath.IsAbs(candidate) {
		scriptPath = candidate
	} else {
		scriptPath = filepath.Join(baseDir, candidate)
	}

	switch strings.ToLower(filepath.Ext(scriptPath)) {
	case ".py":
		language = "python"
	case ".sh", ".bash", ".zsh":
		language = "sh"
	case ".js":
		language = "node"
	case ".rb":
		language = "ruby"
	}

	return
}

// ---------------------------------------------------------------------------
// Result formatting
// ---------------------------------------------------------------------------

func formatScriptResult(language, scriptPath string, result *scriptResult) map[string]any {
	output := result.Stdout
	if output == "" {
		output = result.Stderr
	}
	if output == "" {
		output = "(no output)"
	}

	return map[string]any{
		"status":       "completed",
		"language":     language,
		"script":       filepath.Base(scriptPath),
		"exit_code":    result.ExitCode,
		"output":       truncateScriptOutput(output, 2000),
		"duration":     result.Duration,
		"truncated":    len(result.Stdout) > 2000 || len(result.Stderr) > 2000,
	}
}

func truncateScriptOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... [truncated by run_script tool]"
}
