package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// SystemDTool manages Linux systemd timers, services, and units.
// It provides a safe wrapper around systemctl commands for creating,
// enabling, disabling, starting, stopping, and listing timers.
//
// This tool is NOT registered as a bundled tool by default.
// Users can add it via reactor.WithExtraTools(tools.NewSystemDTool()).
type SystemDTool struct {
	info *core.ToolInfo
}

// NewSystemDTool creates a new SystemDTool instance.
func NewSystemDTool() core.FuncTool {
	return &SystemDTool{
		info: &core.ToolInfo{
			Name:        "systemd",
			Description: `Linux systemd timer/service manager. Operations: 'list_timers'|'list_services'|'create_timer'|'create_service'|'enable'|'disable'|'start'|'stop'|'status'|'remove'|'journal'. Params: {operation: string, name: string, description: string, command: string, expression: 'cron_expr', user: bool, unit_type: 'timer'|'service', lines: number}. Only available on Linux systems with systemd.`,
			Tags:        []string{"scheduler", "systemd", "system", "linux", "timer", "service"},
			SecurityLevel: core.LevelHighRisk,
			Parameters: []core.Parameter{
				{Name: "operation", Type: "string", Description: "Operation: list_timers, list_services, create_timer, create_service, enable, disable, start, stop, status, remove, journal", Required: true},
				{Name: "name", Type: "string", Description: "Unit name without extension (e.g. 'my-backup')", Required: false},
				{Name: "description", Type: "string", Description: "Unit description", Required: false},
				{Name: "command", Type: "string", Description: "ExecStart command for service units", Required: false},
				{Name: "expression", Type: "string", Description: "Cron-like expression for timer OnCalendar (e.g. '*-*-* 09:00:00' or 'Mon..Fri *-*-* 09:00:00')", Required: false},
				{Name: "user", Type: "boolean", Description: "Use --user flag for user-level units (default: true)", Required: false},
				{Name: "unit_type", Type: "string", Description: "Unit type filter: 'timer' or 'service'", Required: false},
				{Name: "lines", Type: "number", Description: "Number of journal lines to show (default: 50)", Required: false},
			},
		},
	}
}

func (t *SystemDTool) Info() *core.ToolInfo {
	return t.info
}

func (t *SystemDTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("systemd is only available on Linux; current OS is %s", runtime.GOOS)
	}

	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	case "list_timers":
		return t.listTimers(ctx, params)
	case "list_services":
		return t.listServices(ctx, params)
	case "create_timer":
		return t.createTimer(ctx, params)
	case "create_service":
		return t.createService(ctx, params)
	case "enable":
		return t.enableDisable(ctx, params, "enable")
	case "disable":
		return t.enableDisable(ctx, params, "disable")
	case "start":
		return t.startStop(ctx, params, "start")
	case "stop":
		return t.startStop(ctx, params, "stop")
	case "status":
		return t.unitStatus(ctx, params)
	case "remove":
		return t.removeUnit(params)
	case "journal":
		return t.showJournal(ctx, params)
	default:
		return nil, fmt.Errorf("unknown operation: %s (supported: list_timers, list_services, create_timer, create_service, enable, disable, start, stop, status, remove, journal)", operation)
	}
}

func (t *SystemDTool) listTimers(ctx context.Context, params map[string]any) (map[string]any, error) {
	user := t.getUserFlag(params)
	args := []string{user, "list-timers", "--all", "--no-pager"}
	out, err := runCommand(ctx, "systemctl", args...)
	if err != nil {
		return nil, fmt.Errorf("systemctl list-timers failed: %w\n%s", err, out)
	}

	timers := parseSystemctlListOutput(out)
	return map[string]any{
		"timers": timers,
		"count":  len(timers),
		"scope":  t.scopeName(user),
	}, nil
}

func (t *SystemDTool) listServices(ctx context.Context, params map[string]any) (map[string]any, error) {
	user := t.getUserFlag(params)
	unitType, _ := params["unit_type"].(string)
	args := []string{user, "list-units", "--type=service", "--all", "--no-pager"}
	if unitType != "" {
		args = append(args, fmt.Sprintf("*.%s*", unitType))
	}
	out, err := runCommand(ctx, "systemctl", args...)
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units failed: %w\n%s", err, out)
	}

	services := parseSystemctlListOutput(out)
	return map[string]any{
		"services": services,
		"count":    len(services),
		"scope":    t.scopeName(user),
	}, nil
}

func (t *SystemDTool) createTimer(ctx context.Context, params map[string]any) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("missing or invalid 'command' parameter")
	}
	expression, ok := params["expression"].(string)
	if !ok || expression == "" {
		return nil, fmt.Errorf("missing or invalid 'expression' parameter (OnCalendar format)")
	}

	description, _ := params["description"].(string)
	if description == "" {
		description = fmt.Sprintf("Timer: %s", name)
	}

	user := t.getUserFlag(params)
	unitDir := t.getUnitDir(user)

	serviceContent := fmt.Sprintf(`[Unit]
Description=%s

[Service]
Type=oneshot
ExecStart=%s

[Install]
WantedBy=default.target
`, description, command)

	timerContent := fmt.Sprintf(`[Unit]
Description=%s (timer)

[Timer]
OnCalendar=%s
Persistent=true

[Install]
WantedBy=timers.target
`, description, expression)

	serviceFile := fmt.Sprintf("%s/%s.service", unitDir, name)
	timerFile := fmt.Sprintf("%s/%s.timer", unitDir, name)

	if err := writeFile(serviceFile, serviceContent); err != nil {
		return nil, fmt.Errorf("failed to write service file: %w", err)
	}
	if err := writeFile(timerFile, timerContent); err != nil {
		return nil, fmt.Errorf("failed to write timer file: %w", err)
	}

	daemonReload(ctx, user)

	return map[string]any{
		"created":       true,
		"name":          name,
		"service_file":  serviceFile,
		"timer_file":    timerFile,
		"on_calendar":   expression,
		"next_step":     fmt.Sprintf("Run 'enable' operation with name='%s' to activate the timer", name),
		"message":       "Timer unit created. Use 'enable' to start it.",
	}, nil
}

func (t *SystemDTool) createService(ctx context.Context, params map[string]any) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("missing or invalid 'command' parameter")
	}

	description, _ := params["description"].(string)
	if description == "" {
		description = fmt.Sprintf("Service: %s", name)
	}

	user := t.getUserFlag(params)
	unitDir := t.getUnitDir(user)

	content := fmt.Sprintf(`[Unit]
Description=%s

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, description, command)

	filePath := fmt.Sprintf("%s/%s.service", unitDir, name)
	if err := writeFile(filePath, content); err != nil {
		return nil, fmt.Errorf("failed to write service file: %w", err)
	}

	daemonReload(ctx, user)

	return map[string]any{
		"created":    true,
		"name":       name,
		"file_path":  filePath,
		"message":    "Service unit created. Use 'enable' then 'start' to activate it.",
	}, nil
}

func (t *SystemDTool) enableDisable(ctx context.Context, params map[string]any, action string) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	user := t.getUserFlag(params)
	out, err := runCommand(ctx, "systemctl", user, action, name)
	if err != nil {
		return nil, fmt.Errorf("systemctl %s failed for '%s': %w\n%s", action, name, err, out)
	}

	return map[string]any{
		"success": true,
		"name":    name,
		"action":  action,
		"output":  strings.TrimSpace(out),
		"message": fmt.Sprintf("Unit '%s' %sd successfully", name, action),
	}, nil
}

func (t *SystemDTool) startStop(ctx context.Context, params map[string]any, action string) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	user := t.getUserFlag(params)
	out, err := runCommand(ctx, "systemctl", user, action, name)
	if err != nil {
		return nil, fmt.Errorf("systemctl %s failed for '%s': %w\n%s", action, name, err, out)
	}

	return map[string]any{
		"success": true,
		"name":    name,
		"action":  action,
		"output":  strings.TrimSpace(out),
	}, nil
}

func (t *SystemDTool) unitStatus(ctx context.Context, params map[string]any) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	user := t.getUserFlag(params)
	out, err := runCommand(ctx, "systemctl", user, "status", name, "--no-pager")
	if err != nil {
		if strings.Contains(out, "could not be found") || strings.Contains(out, "Unit .* could not be found") {
			return map[string]any{"name": name, "loaded": false, "active": false, "message": "Unit not found"}, nil
		}
		return nil, fmt.Errorf("systemctl status failed: %w\n%s", err, out)
	}

	activeState := extractStatusField(out, "Active:")
	subState := extractStatusField(out, "Main PID:")
	enabled := strings.Contains(out, "enabled")

	return map[string]any{
		"name":         name,
		"loaded":       true,
		"active":       activeState == "active",
		"active_state": activeState,
		"sub_state":    subState,
		"enabled":      enabled,
		"detail":       strings.TrimSpace(out),
	}, nil
}

func (t *SystemDTool) removeUnit(params map[string]any) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	user := t.getUserFlag(params)
	unitDir := t.getUnitDir(user)

	var removedFiles []string
	for _, ext := range []string{".service", ".timer"} {
		path := fmt.Sprintf("%s/%s%s", unitDir, name, ext)
		if err := removeFile(path); err == nil {
			removedFiles = append(removedFiles, path)
		}
	}

	if len(removedFiles) == 0 {
		return nil, fmt.Errorf("no unit files found for '%s' in %s", name, unitDir)
	}

	daemonReload(context.Background(), user)

	return map[string]any{
		"removed":   true,
		"name":      name,
		"files":     removedFiles,
		"message":   fmt.Sprintf("Removed %d unit file(s) for '%s'", len(removedFiles), name),
	}, nil
}

func (t *SystemDTool) showJournal(ctx context.Context, params map[string]any) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	user := t.getUserFlag(params)
	lines := 50
	if l, ok := params["lines"].(float64); ok {
		lines = int(l)
	}

	out, err := runCommand(ctx, "journalctl", user, "-u", name, "-n", strconv.Itoa(lines), "--no-pager")
	if err != nil {
		return nil, fmt.Errorf("journalctl failed for '%s': %w\n%s", name, err, out)
	}

	journalLines := strings.Split(strings.TrimSpace(out), "\n")

	return map[string]any{
		"name":   name,
		"lines":  journalLines,
		"count":  len(journalLines),
		"output": out,
	}, nil
}

func (t *SystemDTool) getUserFlag(params map[string]any) string {
	userVal, ok := params["user"].(bool)
	if !ok {
		userVal = true
	}
	if userVal {
		return "--user"
	}
	return "--system"
}

func (t *SystemDTool) getUnitDir(userFlag string) string {
	if userFlag == "--user" {
		home := homeDir()
		if home != "" {
			return home + "/.config/systemd/user"
		}
		return "/etc/systemd/user"
	}
	return "/etc/systemd/system"
}

func (t *SystemDTool) scopeName(userFlag string) string {
	if userFlag == "--user" {
		return "user"
	}
	return "system"
}

func parseSystemctlListOutput(output string) []map[string]any {
	var items []map[string]any
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i == 0 || strings.HasPrefix(line, "UNIT") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			item := map[string]any{
				"unit":   parts[0],
				"load":   parts[1],
				"active": parts[2],
				"sub":    parts[3],
			}
			if len(parts) > 4 {
				item["description"] = strings.Join(parts[4:], " ")
			}
			items = append(items, item)
		}
	}
	return items
}

func extractStatusField(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return "unknown"
}

func daemonReload(ctx context.Context, userFlag string) {
	runCommand(ctx, "systemctl", userFlag, "daemon-reload")
}

func homeDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func removeFile(path string) error {
	return os.Remove(path)
}
