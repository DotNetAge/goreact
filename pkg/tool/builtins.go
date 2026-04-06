package tool

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DotNetAge/goreact/pkg/common"
)

// BashTool executes shell commands
type BashTool struct {
	*BaseTool
	workDir string
}

// NewBashTool creates a new BashTool
func NewBashTool(workDir string) *BashTool {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	
	return &BashTool{
		BaseTool: NewBaseTool(
			"bash",
			"Execute bash commands in a persistent shell session",
			common.LevelSensitive,
			false,
		).WithParameter(Parameter{
			Name:        "command",
			Type:        "string",
			Required:    true,
			Description: "The bash command to execute",
		}).WithParameter(Parameter{
			Name:        "timeout",
			Type:        "integer",
			Required:    false,
			Default:     30000,
			Description: "Timeout in milliseconds",
		}),
		workDir: workDir,
	}
}

// Run executes the bash command
func (t *BashTool) Run(ctx context.Context, params map[string]any) (any, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command parameter must be a string")
	}
	
	// Build the command
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = t.workDir
	
	// Capture output
	output, err := cmd.CombinedOutput()
	
	result := map[string]any{
		"command":  command,
		"output":   string(output),
		"exit_code": 0,
	}
	
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		}
		return result, err
	}
	
	return result, nil
}

// ReadTool reads file contents
type ReadTool struct {
	*BaseTool
}

// NewReadTool creates a new ReadTool
func NewReadTool() *ReadTool {
	return &ReadTool{
		BaseTool: NewBaseTool(
			"read",
			"Read the contents of a file",
			common.LevelSafe,
			true,
		).WithParameter(Parameter{
			Name:        "file_path",
			Type:        "string",
			Required:    true,
			Description: "The absolute path to the file to read",
		}).WithParameter(Parameter{
			Name:        "limit",
			Type:        "integer",
			Required:    false,
			Default:     2000,
			Description: "Maximum number of lines to read",
		}).WithParameter(Parameter{
			Name:        "offset",
			Type:        "integer",
			Required:    false,
			Default:     0,
			Description: "Number of lines to skip from start",
		}),
	}
}

// Run reads the file
func (t *ReadTool) Run(ctx context.Context, params map[string]any) (any, error) {
	filePath, ok := params["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter must be a string")
	}
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Get offset and limit
	offset := 0
	if o, ok := params["offset"].(int); ok {
		offset = o
	}
	limit := 2000
	if l, ok := params["limit"].(int); ok {
		limit = l
	}
	
	// Read lines
	lines := []string{}
	scanner := newLineScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		if lineNum >= offset {
			lines = append(lines, scanner.Text())
			if len(lines) >= limit {
				break
			}
		}
		lineNum++
	}
	
	// Get file info
	info, _ := file.Stat()
	
	return map[string]any{
		"path":      filePath,
		"content":   strings.Join(lines, "\n"),
		"lines":     len(lines),
		"total":     lineNum,
		"size":      info.Size(),
	}, nil
}

// WriteTool writes content to a file
type WriteTool struct {
	*BaseTool
}

// NewWriteTool creates a new WriteTool
func NewWriteTool() *WriteTool {
	return &WriteTool{
		BaseTool: NewBaseTool(
			"write",
			"Write content to a file",
			common.LevelSensitive,
			false,
		).WithParameter(Parameter{
			Name:        "file_path",
			Type:        "string",
			Required:    true,
			Description: "The absolute path to the file to write",
		}).WithParameter(Parameter{
			Name:        "content",
			Type:        "string",
			Required:    true,
			Description: "The content to write to the file",
		}).WithParameter(Parameter{
			Name:        "mode",
			Type:        "string",
			Required:    false,
			Default:     "write",
			Description: "Write mode: 'write' to overwrite, 'append' to append",
			Enum:        []any{"write", "append"},
		}),
	}
}

// Run writes to the file
func (t *WriteTool) Run(ctx context.Context, params map[string]any) (any, error) {
	filePath, ok := params["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter must be a string")
	}
	
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter must be a string")
	}
	
	mode := "write"
	if m, ok := params["mode"].(string); ok {
		mode = m
	}
	
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write file
	flag := os.O_CREATE | os.O_WRONLY
	if mode == "append" {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	
	file, err := os.OpenFile(filePath, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	if _, err := file.WriteString(content); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	
	return map[string]any{
		"path":    filePath,
		"mode":    mode,
		"written": len(content),
	}, nil
}

// GlobTool finds files matching patterns
type GlobTool struct {
	*BaseTool
}

// NewGlobTool creates a new GlobTool
func NewGlobTool() *GlobTool {
	return &GlobTool{
		BaseTool: NewBaseTool(
			"glob",
			"Find files matching a pattern",
			common.LevelSafe,
			true,
		).WithParameter(Parameter{
			Name:        "pattern",
			Type:        "string",
			Required:    true,
			Description: "The glob pattern to match files",
		}).WithParameter(Parameter{
			Name:        "path",
			Type:        "string",
			Required:    false,
			Default:     ".",
			Description: "The base directory to search from",
		}),
	}
}

// Run finds matching files
func (t *GlobTool) Run(ctx context.Context, params map[string]any) (any, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern parameter must be a string")
	}
	
	path := "."
	if p, ok := params["path"].(string); ok {
		path = p
	}
	
	// Glob files
	matches, err := filepath.Glob(filepath.Join(path, pattern))
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}
	
	return map[string]any{
		"pattern": pattern,
		"path":    path,
		"files":   matches,
		"count":   len(matches),
	}, nil
}

// ListTool lists directory contents
type ListTool struct {
	*BaseTool
}

// NewListTool creates a new ListTool
func NewListTool() *ListTool {
	return &ListTool{
		BaseTool: NewBaseTool(
			"list",
			"List the contents of a directory",
			common.LevelSafe,
			true,
		).WithParameter(Parameter{
			Name:        "path",
			Type:        "string",
			Required:    false,
			Default:     ".",
			Description: "The directory path to list",
		}),
	}
}

// Run lists the directory
func (t *ListTool) Run(ctx context.Context, params map[string]any) (any, error) {
	path := "."
	if p, ok := params["path"].(string); ok {
		path = p
	}
	
	// Read directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	// Build result
	files := []map[string]any{}
	dirs := []map[string]any{}
	
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		item := map[string]any{
			"name":  entry.Name(),
			"size":  info.Size(),
			"mode":  info.Mode().String(),
			"isDir": entry.IsDir(),
		}
		
		if entry.IsDir() {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}
	
	return map[string]any{
		"path":  path,
		"files": files,
		"dirs":  dirs,
	}, nil
}

// DeleteTool deletes files or directories
type DeleteTool struct {
	*BaseTool
}

// NewDeleteTool creates a new DeleteTool
func NewDeleteTool() *DeleteTool {
	return &DeleteTool{
		BaseTool: NewBaseTool(
			"delete",
			"Delete a file or directory",
			common.LevelHighRisk,
			false,
		).WithParameter(Parameter{
			Name:        "path",
			Type:        "string",
			Required:    true,
			Description: "The path to delete",
		}).WithParameter(Parameter{
			Name:        "recursive",
			Type:        "boolean",
			Required:    false,
			Default:     false,
			Description: "Whether to delete recursively",
		}),
	}
}

// Run deletes the file or directory
func (t *DeleteTool) Run(ctx context.Context, params map[string]any) (any, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}
	
	recursive := false
	if r, ok := params["recursive"].(bool); ok {
		recursive = r
	}
	
	// Check if path exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("path not found: %s", path)
	}
	
	// Delete
	if info.IsDir() {
		if recursive {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
	} else {
		err = os.Remove(path)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}
	
	return map[string]any{
		"path":      path,
		"deleted":   true,
		"wasDir":    info.IsDir(),
	}, nil
}

// RegisterBuiltins registers built-in tools
func RegisterBuiltins() error {
	// File system tools
	if err := Register(NewBashTool("")); err != nil {
		return err
	}
	if err := Register(NewReadTool()); err != nil {
		return err
	}
	if err := Register(NewWriteTool()); err != nil {
		return err
	}
	if err := Register(NewGlobTool()); err != nil {
		return err
	}
	if err := Register(NewListTool()); err != nil {
		return err
	}
	if err := Register(NewDeleteTool()); err != nil {
		return err
	}
	
	return nil
}

// Line scanner helper
type lineScanner struct {
	reader io.Reader
	buffer []byte
	pos    int
}

func newLineScanner(reader io.Reader) *lineScanner {
	return &lineScanner{
		reader: reader,
		buffer: make([]byte, 4096),
	}
}

func (s *lineScanner) Scan() bool {
	return false // Simplified - use bufio.Scanner in real implementation
}

func (s *lineScanner) Text() string {
	return ""
}
