package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// BashTool implements a tool for executing shell commands.
type BashTool struct{}

// NewBashTool 创建 Bash 工具
func NewBashTool() core.FuncTool {
	return &BashTool{}
}

const bashDescription = `Executes a given bash command and returns its output.

The working directory persists between commands, but shell state does not. The shell environment is initialized from the user's profile (bash or zsh).

IMPORTANT: Avoid using this tool to run cat, head, tail, sed, awk, or echo commands, unless explicitly instructed or after you have verified that a dedicated tool cannot accomplish your task. Instead, use the appropriate dedicated tool as this will provide a much better experience for the user:
- File search: Use glob (NOT find or ls)
- Content search: Use grep (NOT grep or rg)
- Read files: Use read (NOT cat/head/tail)
- Edit files: Use file_edit (NOT sed/awk)
- Write files: Use write (NOT echo >/cat <<EOF)
- Communication: Output text directly (NOT echo/printf)

While the bash tool can do similar things, it’s better to use the built-in tools as they provide a better user experience and make it easier to review tool calls and give permission.

# Instructions
- If your command will create new directories or files, first use this tool to run ls to verify the parent directory exists and is the correct location.
- Always quote file paths that contain spaces with double quotes in your command.
- Try to maintain your current working directory throughout the session by using absolute paths and avoiding usage of cd. You may use cd if the User explicitly requests it.
- When issuing multiple commands:
  - If the commands are independent and can run in parallel, make multiple tool calls in a single message.
  - If the commands depend on each other and must run sequentially, use a single call with '&&' to chain them together.
  - Use ';' only when you need to run commands sequentially but don't care if earlier commands fail.
  - DO NOT use newlines to separate commands (newlines are ok in quoted strings).
- For git commands:
  - Prefer to create a new commit rather than amending an existing commit.
  - Before running destructive operations (e.g., git reset --hard, git push --force), consider safer alternatives.
  - Never skip hooks (--no-verify) or bypass signing unless the user explicitly asked for it.`

func (t *BashTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:          "bash",
		Description:   bashDescription,
		SecurityLevel: core.LevelHighRisk,
		Parameters: []core.Parameter{
			{
				Name:        "command",
				Type:        "string",
				Description: "The command to execute.",
				Required:    true,
			},
			{
				Name:        "timeout",
				Type:        "number",
				Description: "Optional timeout in milliseconds. Default is 30000ms.",
				Required:    false,
			},
		},
	}
}

func (t *BashTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing command parameter")
	}

	if blocked := detectDangerousCommand(command); blocked != "" {
		return map[string]any{
			"stdout":      "",
			"stderr":      fmt.Sprintf("BLOCKED: %s", blocked),
			"exit_code":   126,
			"interrupted": false,
			"success":     false,
			"error":       blocked,
		}, nil
	}

	timeoutMs := 30000
	if val, ok := params["timeout"].(float64); ok {
		timeoutMs = int(val)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "sh", "-c", command)

	// Use strings.Builder to capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	result := map[string]any{
		"stdout":      stdoutStr,
		"stderr":      stderrStr,
		"exit_code":   0,
		"interrupted": false,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitError.ExitCode()
		} else if timeoutCtx.Err() == context.DeadlineExceeded {
			result["interrupted"] = true
			stderrStr += "\nCommand timed out."
			result["stderr"] = stderrStr
		} else {
			return nil, err
		}
	}

	// Truncate output if too large (ClueCode style)
	const maxOutputSize = 30000
	result["stdout"] = truncateOutput(stdoutStr, maxOutputSize)
	result["stderr"] = truncateOutput(stderrStr, maxOutputSize)

	// Map exit_code == 0 to success for tests
	result["success"] = result["exit_code"] == 0
	if !result["success"].(bool) {
		result["error"] = fmt.Sprintf("Command failed with exit code %v", result["exit_code"])
	}

	return result, nil
}

// truncateOutput truncates a string to maxRunes characters, appending a truncation notice if needed.
func truncateOutput(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "\n... [output truncated due to size] ..."
}

// dangerousPatterns defines commands that are too destructive to allow even with permission.
// This is defense-in-depth on top of the AskPermission tool.
var dangerousPatterns = []struct {
	pattern string
	reason  string
}{
	{`rm\s+-rf\s+/\s*`, "destructive: rm -rf / would erase the entire filesystem"},
	{`rm\s+-rf\s+/\*`, "destructive: rm -rf /* would erase the entire filesystem"},
	{`rm\s+-rf\s+/[a-z]*\s*$`, "destructive: recursive root-level removal is blocked"},
	{`>\s*/dev/sd[a-z]\b`, "dangerous: writing to raw disk device"},
	{`dd\s+if=.*of=/dev/sd`, "dangerous: raw disk overwrite via dd"},
	{`mkfs\.`, "dangerous: disk formatting command"},
	{`:.*\|.*:&\s*;:\s*}`, "dangerous: fork bomb detected"},
	{`(curl|wget)\s+.*\|\s*(sh|bash)`, "dangerous: remote code execution pipe (curl|sh)"},
	{`(curl|wget)\s+.*\s*>\s*/(bin|usr/bin)/`, "dangerous: remote binary download to system path"},
	{`chmod\s+-R\s+777\s+/`, "dangerous: world-writable root filesystem"},
	{`chown\s+-R.*\s+/`, "dangerous: recursive root ownership change"},
	{`shutdown\s+(now|-h|-r)`, "dangerous: system shutdown command"},
	{`reboot\b`, "dangerous: system reboot command"},
}

// detectDangerousCommand checks a shell command against known dangerous patterns.
// Returns an empty string if safe, or a block reason if matched.
func detectDangerousCommand(command string) string {
	lower := strings.ToLower(strings.TrimSpace(command))
	for _, dp := range dangerousPatterns {
		if matchPattern(lower, dp.pattern) {
			return dp.reason
		}
	}
	return ""
}

// matchPattern performs a simple substring check for the given pattern.
// Uses case-insensitive matching for ASCII patterns.
func matchPattern(s, pattern string) bool {
	lowerS := strings.ToLower(s)
	lowerP := strings.ToLower(pattern)
	if len(lowerP) > len(lowerS) {
		return false
	}
	for i := 0; i <= len(lowerS)-len(lowerP); i++ {
		if lowerS[i:i+len(lowerP)] == lowerP {
			return true
		}
	}
	return false
}
