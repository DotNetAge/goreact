package tools

import (
	"fmt"
	"path/filepath"
)

// ValidateRequired 验证必需参数
func ValidateRequired(params map[string]any, key string) error {
	if _, ok := params[key]; !ok {
		return fmt.Errorf("missing required parameter: %s", key)
	}
	return nil
}

// ValidateRequiredString 验证必需的字符串参数
func ValidateRequiredString(params map[string]any, key string) (string, error) {
	if err := ValidateRequired(params, key); err != nil {
		return "", err
	}

	str, ok := params[key].(string)
	if !ok {
		return "", fmt.Errorf("invalid type for parameter '%s': expected string", key)
	}
	return str, nil
}

// ValidateFileSafety 验证文件安全性
func ValidateFileSafety(path string) error {
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

// TruncateString 截断字符串，超过 maxLen 时用 "..." 省略
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ToFloat64 将 any 转换为 float64
func ToFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}
