package builtin

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ray/goreact/pkg/tools"
)

// LS 列出目录内容工具
type LS struct{}

// NewLS 创建 LS 工具
func NewLS() tools.Tool {
	return &LS{}
}

// Name 返回工具名称
func (l *LS) Name() string {
	return "ls"
}

// Description 返回工具描述
func (l *LS) Description() string {
	return "列出目录内容。支持树形结构、过滤、详细信息。Params: {path?: '目录路径', recursive?: false, show_hidden?: false}"
}

// Execute 执行目录列表
func (l *LS) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 获取目录路径（默认为当前目录）
	dirPath := "."
	if path, ok := params["path"].(string); ok && path != "" {
		dirPath = path
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
	entries, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// 构建结果
	var items []map[string]interface{}

	for _, entry := range entries {
		// 跳过隐藏文件（除非指定显示）
		if !showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		item := map[string]interface{}{
			"name": entry.Name(),
			"type": func() string {
				if entry.IsDir() {
					return "directory"
				} else {
					return "file"
				}
			}(),
			"size":    entry.Size(),
			"modTime": entry.ModTime().Format("2006-01-02 15:04:05"),
			"mode":    entry.Mode().String(),
		}

		// 如果是递归模式且是目录，继续读取
		if recursive && entry.IsDir() {
			subDir := filepath.Join(dirPath, entry.Name())
			subEntries, err := ioutil.ReadDir(subDir)
			if err == nil {
				children := make([]map[string]interface{}, 0)
				for _, subEntry := range subEntries {
					if !showHidden && strings.HasPrefix(subEntry.Name(), ".") {
						continue
					}
					child := map[string]interface{}{
						"name": subEntry.Name(),
						"type": func() string {
							if subEntry.IsDir() {
								return "directory"
							} else {
								return "file"
							}
						}(),
						"size":    subEntry.Size(),
						"modTime": subEntry.ModTime().Format("2006-01-02 15:04:05"),
					}
					children = append(children, child)
				}
				item["children"] = children
			}
		}

		items = append(items, item)
	}

	return map[string]interface{}{
		"success":     true,
		"path":        dirPath,
		"total_items": len(items),
		"items":       items,
		"message":     fmt.Sprintf("Listed %d item(s) in '%s'", len(items), dirPath),
	}, nil
}
