package builtin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

// Grep 文本查找工具
type Grep struct{}

// NewGrep 创建文本查找工具
func NewGrep() *Grep {
	return &Grep{}
}

// Name 返回工具名称
func (g *Grep) Name() string {
	return "grep"
}

// Description 返回工具描述
func (g *Grep) Description() string {
	return "文本查找工具，在文件中查找特定的文本模式"
}

// Execute 执行文本查找操作
func (g *Grep) Execute(params map[string]interface{}) (interface{}, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'pattern' parameter")
	}

	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// 检查路径类型
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var results []map[string]interface{}

	if info.IsDir() {
		// 遍历目录
		err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				// 读取文件内容
				content, err := ioutil.ReadFile(filePath)
				if err != nil {
					return err
				}

				// 查找匹配
				if re.Match(content) {
					results = append(results, map[string]interface{}{
						"file": filePath,
						"matched": true,
					})
				}
			}

			return nil
		})
	} else {
		// 读取单个文件
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// 查找匹配
		if re.Match(content) {
			results = append(results, map[string]interface{}{
				"file": path,
				"matched": true,
			})
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return map[string]interface{}{
		"pattern": pattern,
		"path":    path,
		"results": results,
	}, nil
}
