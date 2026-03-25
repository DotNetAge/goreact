// Package builtin 提供一组轻量、实用而强大的内置工具集
//
// 设计理念（参考 Claude Code）：
// - 职责单一：每个工具只做一件事，做到极致
// - 组合强大：通过工具组合实现复杂功能
// - 覆盖全面：文件操作、搜索、执行三大核心场景
//
// 工具分类：
// Tier 1 - 文件操作：Read, Write, Edit
// Tier 2 - 搜索浏览：Glob, Grep, LS
// Tier 3 - 执行扩展：Bash, Calculator, DateTime, Cron
//
// 已移除：
// - Email → 移至独立插件包 (goreact-plugins/email)
// - Docker, Git → 低频使用，移至独立插件包
// - HTTP, Curl → 使用 bash curl 替代
// - Filesystem → 拆分为 Read/Write/Edit
// - Echo, Port → 无价值，直接删除
package builtin

import (
	"fmt"
	"path/filepath"
)

// validateRequired 验证必需参数
func validateRequired(params map[string]any, key string) error {
	if _, ok := params[key]; !ok {
		return fmt.Errorf("missing required parameter: %s", key)
	}
	return nil
}

// validateRequiredString 验证必需的字符串参数
func validateRequiredString(params map[string]any, key string) (string, error) {
	if err := validateRequired(params, key); err != nil {
		return "", err
	}

	str, ok := params[key].(string)
	if !ok {
		return "", fmt.Errorf("invalid type for parameter '%s': expected string", key)
	}
	return str, nil
}

// validateFileSafety 验证文件安全性
func validateFileSafety(path string) error {
	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查是否是特殊文件
	baseName := filepath.Base(cleanPath)
	specialFiles := []string{"passwd", "shadow", "sudoers"}
	for _, special := range specialFiles {
		if baseName == special {
			return fmt.Errorf("access to %s is restricted for security reasons", special)
		}
	}

	return nil
}
