package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// LaunchdTool manages macOS launchd agents and daemons via plist files.
// It provides a safe wrapper around launchctl commands for creating,
// loading, unloading, listing, and managing scheduled tasks on macOS.
//
// This tool is NOT registered as a bundled tool by default.
// Users can add it via reactor.WithExtraTools(tools.NewLaunchdTool()).
type LaunchdTool struct {
	info *core.ToolInfo
}

// NewLaunchdTool creates a new LaunchdTool instance.
func NewLaunchdTool() core.FuncTool {
	return &LaunchdTool{
		info: &core.ToolInfo{
			Name:        "launchd",
			Description: `macOS launchd plist manager. Operations: 'list'|'create'|'load'|'unload'|'start'|'stop'|'status'|'remove'. Params: {operation: string, label: string, program: string, arguments: [], interval: number, calendar: string, environment: {}, stdout: string, stderr: string, working_directory: string}. Only available on macOS.`,
			Tags:        []string{"scheduler", "launchd", "system", "macos", "daemon"},
			SecurityLevel: core.LevelHighRisk,
			Parameters: []core.Parameter{
				{Name: "operation", Type: "string", Description: "Operation: list, create, load, unload, start, stop, status, remove", Required: true},
				{Name: "label", Type: "string", Description: "Unique label identifier for the service (e.g. 'com.user.mytask')", Required: false},
				{Name: "program", Type: "string", Description: "Path to the executable program", Required: false},
				{Name: "arguments", Type: "array", Description: "Arguments to pass to the program", Required: false},
				{Name: "interval", Type: "number", Description: "Start interval in seconds (for KeepAlive-style periodic jobs)", Required: false},
				{Name: "calendar", Type: "string", Description: "Calendar interval in launchd format (e.g. 'Hourly', 'Daily', 'Weekly') or a dict with calendar-specific keys", Required: false},
				{Name: "environment", Type: "object", Description: "Environment variables to set for the job", Required: false},
				{Name: "stdout", Type: "string", Description: "Path for stdout redirection (default: /dev/null)", Required: false},
				{Name: "stderr", Type: "string", Description: "Path for stderr redirection (default: /dev/null)", Required: false},
				{Name: "working_directory", Type: "string", Description: "Working directory for the job", Required: false},
			},
		},
	}
}

func (t *LaunchdTool) Info() *core.ToolInfo {
	return t.info
}

const launchDAgentsDir = "/Library/LaunchAgents"
const launchDDaemonsDir = "/Library/LaunchDaemons"
const userLaunchDir = "~/Library/LaunchAgents"

func (t *LaunchdTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("launchd is only available on macOS; current OS is %s", runtime.GOOS)
	}

	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	case "list":
		return t.list(ctx, params)
	case "create":
		return t.create(params)
	case "load":
		return t.loadUnload(ctx, params, "load")
	case "unload":
		return t.loadUnload(ctx, params, "unload")
	case "start":
		return t.startStop(ctx, params, "start")
	case "stop":
		return t.startStop(ctx, params, "stop")
	case "status":
		return t.status(ctx, params)
	case "remove":
		return t.remove(params)
	default:
		return nil, fmt.Errorf("unknown operation: %s (supported: list, create, load, unload, start, stop, status, remove)", operation)
	}
}

func (t *LaunchdTool) list(ctx context.Context, params map[string]any) (map[string]any, error) {
	scope, _ := params["scope"].(string)
	if scope == "" {
		scope = "user"
	}

	dir := t.resolveScope(scope)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"services": []any{}, "count": 0, "directory": dir, "message": "No launch services found"}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", dir, err)
	}

	var services []map[string]any
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".plist") {
			continue
		}
		label := strings.TrimSuffix(entry.Name(), ".plist")
		filePath := filepath.Join(dir, entry.Name())
		services = append(services, map[string]any{
			"label":      label,
			"plist_file": filePath,
			"is_loaded":  t.isLoaded(ctx, label),
		})
	}

	return map[string]any{
		"services":  services,
		"count":     len(services),
		"directory": dir,
		"scope":     scope,
	}, nil
}

func (t *LaunchdTool) create(params map[string]any) (map[string]any, error) {
	label, ok := params["label"].(string)
	if !ok || label == "" {
		return nil, fmt.Errorf("missing or invalid 'label' parameter")
	}
	program, ok := params["program"].(string)
	if !ok || program == "" {
		return nil, fmt.Errorf("missing or invalid 'program' parameter")
	}

	scope, _ := params["scope"].(string)
	if scope == "" {
		scope = "user"
	}
	dir := t.resolveScope(scope)

	argsRaw, _ := params["arguments"].([]any)
	var args []string
	for _, a := range argsRaw {
		if s, ok := a.(string); ok {
			args = append(args, s)
		}
	}

	envRaw, _ := params["environment"].(map[string]any)
	envMap := make(map[string]string)
	for k, v := range envRaw {
		if s, ok := v.(string); ok {
			envMap[k] = s
		}
	}

	stdoutPath, _ := params["stdout"].(string)
	stderrPath, _ := params["stderr"].(string)
	workingDir, _ := params["working_directory"].(string)
	intervalFloat, _ := params["interval"].(float64)
	calendarStr, _ := params["calendar"].(string)

	plistContent := t.buildPlist(label, program, args, envMap, stdoutPath, stderrPath, workingDir, int(intervalFloat), calendarStr)

	plistName := label + ".plist"
	plistPath := filepath.Join(dir, plistName)

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write plist file: %w", err)
	}

	return map[string]any{
		"created":    true,
		"label":      label,
		"plist_path": plistPath,
		"scope":      scope,
		"message":    fmt.Sprintf("Plist created at %s. Use 'load' operation to register it with launchd.", plistPath),
	}, nil
}

func (t *LaunchdTool) loadUnload(ctx context.Context, params map[string]any, action string) (map[string]any, error) {
	label, ok := params["label"].(string)
	if !ok || label == "" {
		return nil, fmt.Errorf("missing or invalid 'label' parameter")
	}

	guionFlag := "gui/$(id -u)"
	out, err := runCommand(ctx, "launchctl", guionFlag, action, t.plistPathForLabel(label))
	if err != nil {
		return nil, fmt.Errorf("launchctl %s failed for '%s': %w\n%s", action, label, err, out)
	}

	return map[string]any{
		"success":  true,
		"label":    label,
		"action":   action,
		"output":   strings.TrimSpace(out),
		"message":  fmt.Sprintf("Service '%s' %sed successfully", label, action),
	}, nil
}

func (t *LaunchdTool) startStop(ctx context.Context, params map[string]any, action string) (map[string]any, error) {
	label, ok := params["label"].(string)
	if !ok || label == "" {
		return nil, fmt.Errorf("missing or invalid 'label' parameter")
	}

	guionFlag := "gui/$(id -u)"
	out, err := runCommand(ctx, "launchctl", guionFlag, action, label)
	if err != nil {
		return nil, fmt.Errorf("launchctl %s failed for '%s': %w\n%s", action, label, err, out)
	}

	return map[string]any{
		"success": true,
		"label":   label,
		"action":  action,
		"output":  strings.TrimSpace(out),
	}, nil
}

func (t *LaunchdTool) status(ctx context.Context, params map[string]any) (map[string]any, error) {
	label, ok := params["label"].(string)
	if !ok || label == "" {
		return nil, fmt.Errorf("missing or invalid 'label' parameter")
	}

	guionFlag := "gui/$(id -u)"
	out, err := runCommand(ctx, "launchctl", guionFlag, "print", label)
	if err != nil {
		if strings.Contains(out, "Could not find service") || strings.Contains(err.Error(), "not found") {
			return map[string]any{"label": label, "loaded": false, "running": false, "pid": 0, "message": "Service not loaded"}, nil
		}
		return nil, fmt.Errorf("launchctl print failed: %w\n%s", err, out)
	}

	pid := t.extractPID(out)
	state := t.extractState(out)

	return map[string]any{
		"label":   label,
		"loaded":  true,
		"running": state == "running",
		"state":   state,
		"pid":     pid,
		"detail":  strings.TrimSpace(out),
	}, nil
}

func (t *LaunchdTool) remove(params map[string]any) (map[string]any, error) {
	label, ok := params["label"].(string)
	if !ok || label == "" {
		return nil, fmt.Errorf("missing or invalid 'label' parameter")
	}

	plistPath := t.plistPathForLabel(label)
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plist file not found: %s", plistPath)
	}

	if err := os.Remove(plistPath); err != nil {
		return nil, fmt.Errorf("failed to remove plist: %w", err)
	}

	return map[string]any{
		"removed":   true,
		"label":     label,
		"file":      plistPath,
		"message":   fmt.Sprintf("Plist file removed: %s", plistPath),
	}, nil
}

func (t *LaunchdTool) resolveScope(scope string) string {
	switch scope {
	case "system_agents":
		return launchDAgentsDir
	case "system_daemons":
		return launchDDaemonsDir
	default:
		expanded, _ := os.UserHomeDir()
		return filepath.Join(expanded, "Library/LaunchAgents")
	}
}

func (t *LaunchdTool) plistPathForLabel(label string) string {
	expanded, _ := os.UserHomeDir()
	return filepath.Join(expanded, "Library/LaunchAgents", label+".plist")
}

func (t *LaunchdTool) isLoaded(ctx context.Context, label string) bool {
	guionFlag := "gui/$(id -u)"
	_, err := runCommand(ctx, "launchctl", guionFlag, "print", label)
	return err == nil
}

func (t *LaunchdTool) extractPID(output string) int {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "pid = ") {
			var pid int
			fmt.Sscanf(line, "pid = %d", &pid)
			return pid
		}
	}
	return 0
}

func (t *LaunchdTool) extractState(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "state = ") {
			var state string
			fmt.Sscanf(line, "state = %s", &state)
			return strings.Trim(state, `"`)
		}
	}
	return "unknown"
}

func (t *LaunchdTool) buildPlist(label, program string, args []string, env map[string]string, stdout, stderr, workingDir string, interval int, calendar string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	b.WriteString(fmt.Sprintf("\t<key>Label</key>\n\t<string>%s</string>\n", label))
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	b.WriteString(fmt.Sprintf("\t\t<string>%s</string>\n", program))
	for _, arg := range args {
		b.WriteString(fmt.Sprintf("\t\t<string>%s</string>\n", arg))
	}
	b.WriteString("\t</array>\n")

	if interval > 0 {
		b.WriteString(fmt.Sprintf("\t<key>StartInterval</key>\n\t<integer>%d</integer>\n", interval))
	}
	if calendar != "" {
		b.WriteString(t.buildCalendarInterval(calendar))
	}
	if len(env) > 0 {
		b.WriteString("\t<key>EnvironmentVariables</key>\n\t<dict>\n")
		for k, v := range env {
			b.WriteString(fmt.Sprintf("\t\t<key>%s</key>\n\t\t<string>%s</string>\n", k, v))
		}
		b.WriteString("\t</dict>\n")
	}
	if stdout != "" {
		b.WriteString(fmt.Sprintf("\t<key>StandardOutPath</key>\n\t<string>%s</string>\n", stdout))
	} else {
		b.WriteString("\t<key>StandardOutPath</key>\n\t<string>/dev/null</string>\n")
	}
	if stderr != "" {
		b.WriteString(fmt.Sprintf("\t<key>StandardErrorPath</key>\n\t<string>%s</string>\n", stderr))
	} else {
		b.WriteString("\t<key>StandardErrorPath</key>\n\t<string>/dev/null</string>\n")
	}
	if workingDir != "" {
		b.WriteString(fmt.Sprintf("\t<key>WorkingDirectory</key>\n\t<string>%s</string>\n", workingDir))
	}

	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func (t *LaunchdTool) buildCalendarInterval(calendar string) string {
	switch strings.ToLower(calendar) {
	case "hourly":
		return "\t<key>StartCalendarInterval</key>\n\t<dict>\n\t\t<key>Minute</key>\n\t\t<integer>0</integer>\n\t</dict>\n"
	case "daily":
		return "\t<key>StartCalendarInterval</key>\n\t<dict>\n\t\t<key>Hour</key>\n\t\t<integer>9</integer>\n\t\t<key>Minute</key>\n\t\t<integer>0</integer>\n\t</dict>\n"
	case "weekly":
		return "\t<key>StartCalendarInterval</key>\n\t<dict>\n\t\t<key>Weekday</key>\n\t\t<integer>1</integer>\n\t\t<key>Hour</key>\n\t\t<integer>9</integer>\n\t\t<key>Minute</key>\n\t\t<integer>0</integer>\n\t</dict>\n"
	default:
		return ""
	}
}
