package builtin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileSystem 文件系统工具
type FileSystem struct{}

// NewFileSystem 创建文件系统工具
func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

// Name 返回工具名称
func (f *FileSystem) Name() string {
	return "filesystem"
}

// Description 返回工具描述
func (f *FileSystem) Description() string {
	return "文件系统操作工具，支持读写文件、列出目录等操作"
}

// Execute 执行文件系统操作
func (f *FileSystem) Execute(params map[string]interface{}) (interface{}, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	case "read":
		return f.readFile(params)
	case "write":
		return f.writeFile(params)
	case "list":
		return f.listDir(params)
	case "mkdir":
		return f.mkdir(params)
	case "rm":
		return f.rm(params)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// readFile 读取文件内容
func (f *FileSystem) readFile(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// writeFile 写入文件内容
func (f *FileSystem) writeFile(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'content' parameter")
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File written successfully: %s", path), nil
}

// listDir 列出目录内容
func (f *FileSystem) listDir(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var result []map[string]interface{}
	for _, entry := range entries {
		result = append(result, map[string]interface{}{
			"name": entry.Name(),
			"type": func() string {
				if entry.IsDir() {
					return "directory"
				}
				return "file"
			}(),
			"size":    entry.Size(),
			"modTime": entry.ModTime(),
		})
	}

	return result, nil
}

// mkdir 创建目录
func (f *FileSystem) mkdir(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return fmt.Sprintf("Directory created successfully: %s", path), nil
}

// rm 删除文件或目录
func (f *FileSystem) rm(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	if err := os.RemoveAll(path); err != nil {
		return nil, fmt.Errorf("failed to remove: %w", err)
	}

	return fmt.Sprintf("Removed successfully: %s", path), nil
}
