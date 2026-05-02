package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// LS lists directory contents with metadata (size, type, permissions, modification time).
type LS struct {
	info *core.ToolInfo
}

// NewLsTool creates an Ls tool.
func NewLsTool() core.FuncTool {
	return &LS{
		info: &core.ToolInfo{
			Name:        "Ls",
			Description: "List directory contents with file metadata — size, type, permissions, modification time. Supports recursive tree view and hidden files.",
			Prompt: `List the contents of a directory to browse the filesystem structure. Use this when you need to see what files exist in a directory, check file sizes, or explore the project layout before reading or editing files.

## Operations

### Basic listing — See files in a directory
Call with no parameters to list the current directory. Each entry includes: name, type (file/directory), size in bytes, modification time, and Unix permissions.

### Recursive tree view
Set recursive=true to show the full directory tree two levels deep. Sub-directories expand with their own children listed under them.

### Show hidden files
Set show_hidden=true to include dot-files (.gitignore, .env, .config, etc.). Hidden files are excluded by default.

## When to use this vs other tools
- Use Ls to explore what's in a directory before reading files.
- Use Glob to search for files matching a pattern across the whole project.
- Use Read to read a specific file's content.
- When exploring an unfamiliar codebase, start with Ls on the root directory to understand the project structure.`,
			Tags:         []string{"file", "filesystem", "list", "directory"},
			SecurityLevel: core.LevelSafe,
			Parameters: []core.Parameter{
				{Name: "path", Type: "string", Description: "Directory path to list. Defaults to current directory ('.').", Required: false},
				{Name: "recursive", Type: "boolean", Description: "If true, recursively list sub-directories (2 levels deep). Default: false.", Required: false},
				{Name: "show_hidden", Type: "boolean", Description: "If true, include dot-files and hidden directories. Default: false.", Required: false},
			},
		},
	}
}

func (l *LS) Info() *core.ToolInfo {
	return l.info
}

func (l *LS) Execute(ctx context.Context, params map[string]any) (any, error) {
	// 获取目录路径（默认为当前目录）
	dirPath := "."
	if path, ok := params["path"].(string); ok && path != "" {
		dirPath = path
	}

	// 安全检查
	if err := ValidateFileSafety(dirPath); err != nil {
		return nil, err
	}

	// 检查路径是否存在
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", dirPath)
		}
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// 获取参数
	recursive := false
	if rec, ok := params["recursive"].(bool); ok {
		recursive = rec
	}

	showHidden := false
	if hidden, ok := params["show_hidden"].(bool); ok {
		showHidden = hidden
	}

	// 读取目录内容
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// 构建结果
	var items []map[string]any

	for _, entry := range entries {
		// 跳过隐藏文件（除非指定显示）
		if !showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		finfo, _ := entry.Info()
		item := map[string]any{
			"name": entry.Name(),
			"type": func() string {
				if entry.IsDir() {
					return "directory"
				} else {
					return "file"
				}
			}(),
			"size":    finfo.Size(),
			"modTime": finfo.ModTime().Format("2006-01-02 15:04:05"),
			"mode":    finfo.Mode().String(),
		}

		// 如果是递归模式且是目录，继续读取
		if recursive && entry.IsDir() {
			subDir := filepath.Join(dirPath, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err == nil {
				children := make([]map[string]any, 0)
				for _, subEntry := range subEntries {
					if !showHidden && strings.HasPrefix(subEntry.Name(), ".") {
						continue
					}
					subFinfo, _ := subEntry.Info()
					child := map[string]any{
						"name": subEntry.Name(),
						"type": func() string {
							if subEntry.IsDir() {
								return "directory"
							} else {
								return "file"
							}
						}(),
						"size":    subFinfo.Size(),
						"modTime": subFinfo.ModTime().Format("2006-01-02 15:04:05"),
					}
					children = append(children, child)
				}
				item["children"] = children
			}
		}

		items = append(items, item)
	}

	return map[string]any{
		"success":     true,
		"path":        dirPath,
		"total_items": len(items),
		"items":       items,
		"message":     fmt.Sprintf("Listed %d item(s) in '%s'", len(items), dirPath),
	}, nil
}
