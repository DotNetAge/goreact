package builtin

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ray/goreact/pkg/tools"
)

// Grep 文本内容搜索工具
type Grep struct{}

// NewGrep 创建 Grep 工具
func NewGrep() tools.Tool {
	return &Grep{}
}

// Name 返回工具名称
func (gr *Grep) Name() string {
	return "grep"
}

// Description 返回工具描述
func (gr *Grep) Description() string {
	return "文本内容搜索。支持正则表达式、上下文显示。Params: {pattern: '搜索模式', path?: '搜索路径', include?: '*.go'}"
}

// Execute 执行文本搜索
func (gr *Grep) Execute(params map[string]interface{}) (interface{}, error) {
	pattern, err := validateRequiredString(params, "pattern")
	if err != nil {
		return nil, err
	}

	// 获取搜索路径（默认为当前目录）
	searchPath := "."
	if path, ok := params["path"].(string); ok && path != "" {
		searchPath = path
	}

	// 获取文件过滤模式（可选）
	includePattern := ""
	if include, ok := params["include"].(string); ok {
		includePattern = include
	}

	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	// 搜索结果
	matches := make([]map[string]interface{}, 0)
	filesSearched := 0
	totalMatches := 0

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		// 跳过目录和隐藏文件
		if info.IsDir() {
			baseName := filepath.Base(path)
			if strings.HasPrefix(baseName, ".") || baseName == "node_modules" || baseName == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查文件扩展名过滤
		if includePattern != "" {
			matched, _ := filepath.Match(includePattern, info.Name())
			if !matched {
				return nil
			}
		}

		// 跳过二进制文件和大文件（>5MB）
		const maxFileSize = 5 * 1024 * 1024
		if info.Size() > maxFileSize {
			return nil
		}

		filesSearched++

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		// 逐行搜索
		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// 查找匹配
			loc := re.FindStringIndex(line)
			if loc != nil {
				// 记录匹配
				match := map[string]interface{}{
					"file":      path,
					"line":      lineNum,
					"content":   strings.TrimSpace(line),
					"match":     line[loc[0]:loc[1]],
					"start_col": loc[0],
					"end_col":   loc[1],
				}
				matches = append(matches, match)
				totalMatches++
			}
		}

		if err := scanner.Err(); err != nil {
			// 忽略读取错误，继续处理其他文件
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return map[string]interface{}{
		"success":        true,
		"pattern":        pattern,
		"search_path":    searchPath,
		"files_searched": filesSearched,
		"matches_found":  totalMatches,
		"matches":        matches,
		"message":        fmt.Sprintf("Found %d match(es) in %d file(s)", totalMatches, filesSearched),
	}, nil
}
