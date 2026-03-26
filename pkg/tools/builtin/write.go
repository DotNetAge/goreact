package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DotNetAge/goreact/pkg/tools"
)

// Write 文件写入工具
type Write struct{}

// NewWrite 创建文件写入工具
func NewWrite() tools.Tool {
	return &Write{}
}

// Name 返回工具名称
func (w *Write) Name() string {
	return "write"
}

// Description 返回工具描述
func (w *Write) Description() string {
	return "写入文件内容。自动创建目录、权限控制。Params: {path: '文件路径', content: '文件内容', append?: false}"
}

// Execute 执行文件写入
// SecurityLevel returns the tool's security risk level
func (t *Write) SecurityLevel() tools.SecurityLevel {
	return tools.LevelSensitive // Default, needs manual update for risky tools
}

func (w *Write) Execute(ctx context.Context, params map[string]any) (any, error) {
	path, err := validateRequiredString(params, "path")
	if err != nil {
		return nil, err
	}

	content, err := validateRequiredString(params, "content")
	if err != nil {
		return nil, err
	}

	// 安全检查
	if err := validateFileSafety(path); err != nil {
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
