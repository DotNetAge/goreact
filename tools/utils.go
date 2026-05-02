package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateRequired checks that a required parameter exists.
func ValidateRequired(params map[string]any, key string) error {
	if _, ok := params[key]; !ok {
		return fmt.Errorf("missing required parameter: %s", key)
	}
	return nil
}

// ValidateRequiredString validates that a required string parameter exists and is of string type.
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

// ValidateFileSafety verifies file access safety using path anchoring.
// It normalizes the path via filepath.Clean, resolves symlinks, and ensures
// the real path stays within the allowed workspace boundary.
func ValidateFileSafety(path string) error {
	// Step 1: Clean the path to eliminate relative components like ../
	cleaned := filepath.Clean(path)

	// Step 2: Resolve to absolute path
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Step 3: Resolve symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the file does not exist (e.g. a file about to be created), fall back to the absolute path.
		// Only evaluate symlinks for paths that already exist.
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to resolve symlinks: %w", err)
		}
		realPath = absPath
	}

	// Step 4: Resolve the working directory's real path (also resolving symlinks)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	realCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		realCwd = cwd
	}

	// Step 5: Ensure the real path is anchored within the current working directory
	if !strings.HasPrefix(realPath, realCwd+string(filepath.Separator)) && realPath != realCwd {
		return fmt.Errorf("access denied: path %q resolves to %q which is outside the workspace %q", path, realPath, realCwd)
	}

	// Step 6: Check for sensitive system files
	baseName := filepath.Base(realPath)
	restrictedFiles := []string{".env", "id_rsa", "id_ed25519", "passwd", "shadow", "sudoers"}
	for _, restricted := range restrictedFiles {
		if strings.Contains(baseName, restricted) {
			return fmt.Errorf("access to %s is restricted for security reasons", baseName)
		}
	}

	return nil
}

// TruncateString truncates a string to maxLen runes, appending "..." if truncated.
// It counts by runes to safely handle multi-byte characters.
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

// ToFloat64 converts a numeric value of any common type to float64.
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
