package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DotNetAge/goreact/core"
)

// Write 文件写入工具
type Write struct {
	info *core.ToolInfo
}

const writeDescription = `Writes a file to the local filesystem.

Usage:
- This tool will overwrite the existing file if there is one at the provided path unless append mode is true.
- If this is an existing file, you should normally use the read tool first to get the current contents.
- ALWAYS prefer editing existing files using file_edit tool in the codebase. NEVER write new files unless explicitly required.
- The path parameter must be an absolute path, not a relative path.`

// NewWriteTool 创建文件写入工具
func NewWriteTool() core.FuncTool {
	return &Write{
		info: &core.ToolInfo{
			Name:          "write",
			Description:   writeDescription,
			SecurityLevel: core.LevelSensitive,
		},
	}
}

func (w *Write) Info() *core.ToolInfo {
	return w.info
}

func (w *Write) Execute(ctx context.Context, params map[string]any) (any, error) {
	path, err := ValidateRequiredString(params, "path")
	if err != nil {
		return nil, err
	}

	content, err := ValidateRequiredString(params, "content")
	if err != nil {
		return nil, err
	}

	// 安全检查
	if err := ValidateFileSafety(path); err != nil {
		return nil, err
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 检查是否是追加模式
	appendMode := false
	if append, ok := params["append"].(bool); ok {
		appendMode = append
	}

	var file *os.File
	if appendMode {
		// 追加模式
		file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for appending: %w", err)
		}
	} else {
		// 覆盖模式
		file, err = os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
	}
	defer file.Close()

	// 写入内容
	bytesWritten, err := file.WriteString(content)
	if err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	// 获取文件信息
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return map[string]any{
		"success": true,
		"path":    path,
		"mode": func() string {
			if appendMode {
				return "append"
			} else {
				return "overwrite"
			}
		}(),
		"bytes_written": bytesWritten,
		"total_size":    info.Size(),
		"message": func() string {
			if appendMode {
				return "Content appended successfully"
			} else {
				return "File written successfully"
			}
		}(),
	}, nil
}
