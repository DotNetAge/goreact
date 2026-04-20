package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// ValidateFileSafety 验证文件安全性，采用路径锚定模式。
// 使用 filepath.Clean 规范化路径，解析符号链接确保真实路径在允许范围内。
func ValidateFileSafety(path string) error {
	// Step 1: 规范化路径，消除 ../ 等相对路径成分
	cleaned := filepath.Clean(path)

	// Step 2: 获取绝对路径
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Step 3: 解析符号链接，获取真实路径
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// 如果文件不存在（比如即将创建的文件），使用绝对路径作为回退
		// 仅对已存在的路径解析符号链接
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to resolve symlinks: %w", err)
		}
		realPath = absPath
	}

	// Step 4: 获取工作目录的真实路径（同样解析符号链接）
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	realCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		realCwd = cwd
	}

	// Step 5: 确保真实路径在当前工作目录内（白名单目录锚定）
	if !strings.HasPrefix(realPath, realCwd+string(filepath.Separator)) && realPath != realCwd {
		return fmt.Errorf("access denied: path %q resolves to %q which is outside the workspace %q", path, realPath, realCwd)
	}

	// Step 6: 检查是否是敏感系统文件
	baseName := filepath.Base(realPath)
	restrictedFiles := []string{".env", "id_rsa", "id_ed25519", "passwd", "shadow", "sudoers"}
	for _, restricted := range restrictedFiles {
		if strings.Contains(baseName, restricted) {
			return fmt.Errorf("access to %s is restricted for security reasons", baseName)
		}
	}

	return nil
}

// TruncateString 截断字符串，超过 maxLen 时用 "..." 省略（按 rune 计数，安全处理多字节字符）
func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
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
