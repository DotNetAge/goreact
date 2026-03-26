package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/DotNetAge/goreact/pkg/tools"
)

// Replace 文件内容替换工具（支持全局或范围替换）
type Replace struct{}

// NewReplace 创建替换工具
func NewReplace() tools.Tool {
	return &Replace{}
}

// Name 返回工具名称
func (r *Replace) Name() string {
	return "replace"
}

// Description 返回工具描述
func (r *Replace) Description() string {
	return "在文件中查找并替换文本内容。支持全局替换或限定次数。Params: {path: '文件路径', search: '查找文本', replace: '替换文本', limit?: 最大替换次数 (-1 表示全部)}"
}

// SecurityLevel 返回安全级别
func (r *Replace) SecurityLevel() tools.SecurityLevel {
	return tools.LevelSensitive
}

// Execute 执行替换操作
func (r *Replace) Execute(ctx context.Context, params map[string]any) (any, error) {
	path, err := validateRequiredString(params, "path")
	if err != nil {
		return nil, err
	}

	search, err := validateRequiredString(params, "search")
	if err != nil {
		return nil, err
	}

	replace, err := validateRequiredString(params, "replace")
	if err != nil {
		return nil, err
	}

	// 获取可选的替换次数限制
	limit := -1 // 默认全部替换
	if limitVal, ok := params["limit"]; ok {
		if limitFloat, ok := limitVal.(float64); ok {
			limit = int(limitFloat)
		}
	}

	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)

	// 统计匹配数量
	matchCount := strings.Count(originalContent, search)
	if matchCount == 0 {
		return map[string]any{
			"success":      false,
			"path":         path,
			"replacements": 0,
			"message":      fmt.Sprintf("Text '%s' not found in file", truncateString(search, 50)),
		}, nil
	}

	// 执行替换
	var newContent string
	actualReplacements := 0

	if limit == -1 || limit >= matchCount {
		// 全部替换
		newContent = strings.ReplaceAll(originalContent, search, replace)
		actualReplacements = matchCount
	} else {
		// 限定次数替换
		newContent = strings.Replace(originalContent, search, replace, limit)
		actualReplacements = limit
	}

	// 写入修改后的内容
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write modified file: %w", err)
	}

	// 计算大小变化
	sizeDelta := len(newContent) - len(originalContent)

	return map[string]any{
		"success":       true,
		"path":          path,
		"search":        truncateString(search, 100),
		"replace":       truncateString(replace, 100),
		"matches_found": matchCount,
		"replacements":  actualReplacements,
		"original_size": len(originalContent),
		"new_size":      len(newContent),
		"size_delta":    sizeDelta,
		"message": fmt.Sprintf("Successfully replaced %d occurrence(s) of '%s'",
			actualReplacements, truncateString(search, 30)),
	}, nil
}
