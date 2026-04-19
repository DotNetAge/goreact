package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// Glob 文件名模式匹配工具
type Glob struct {
	info *core.ToolInfo
}

// NewGlob 创建 Glob 工具
func NewGlob() core.FuncTool {
	return &Glob{
		info: &core.ToolInfo{
			Name:          "glob",
			Description:   "文件名模式匹配。支持 glob 语法 (*, ?, []). Params: {pattern: '文件模式', path?: '搜索路径'}",
			SecurityLevel: core.LevelSafe,
		},
	}
}

func (g *Glob) Info() *core.ToolInfo {
	return g.info
}

func (g *Glob) Execute(ctx context.Context, params map[string]any) (any, error) {
	pattern, err := ValidateRequiredString(params, "pattern")
	if err != nil {
		return nil, err
	}

	// 获取搜索路径（默认为当前目录）
	searchPath := "."
	if path, ok := params["path"].(string); ok && path != "" {
		searchPath = path
	}

	// 检查路径是否存在
	info, err := os.Stat(searchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("search path does not exist: %s", searchPath)
		}
		return nil, fmt.Errorf("failed to stat search path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("search path must be a directory: %s", searchPath)
	}

	// 收集匹配的文件
	matchedFiles := make([]string, 0)

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续处理其他文件
		}

		// 跳过隐藏文件和目录（以.开头）
		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, ".") {
			return nil
		}

		// 使用 filepath.Match 进行模式匹配
		matched, err := filepath.Match(pattern, baseName)
		if err != nil {
			return fmt.Errorf("invalid pattern '%s': %w", pattern, err)
		}

		if matched {
			// 转换为相对路径
			relPath, err := filepath.Rel(searchPath, path)
			if err != nil {
				relPath = path
			}
			matchedFiles = append(matchedFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return map[string]any{
		"success":       true,
		"pattern":       pattern,
		"search_path":   searchPath,
		"matches_found": len(matchedFiles),
		"files":         matchedFiles,
		"message":       fmt.Sprintf("Found %d file(s) matching '%s'", len(matchedFiles), pattern),
	}, nil
}
