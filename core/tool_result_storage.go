package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func defaultPersistDir() string {
	return filepath.Join(os.TempDir(), "goreact", "tool-results")
}

func PersistToDisk(toolName, result string, dir string, maxChars int, previewChars int) *PersistedToolResult {
	charCount := len([]rune(result))
	if charCount <= 0 || charCount <= maxChars {
		return nil
	}

	sessionDir := filepath.Join(dir, fmt.Sprintf("session_%d", os.Getpid()))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return &PersistedToolResult{
			ToolName: toolName,
			FullSize: charCount,
			Preview:  truncatePreview("", previewChars),
			FilePath: "",
		}
	}

	filename := fmt.Sprintf("%s_%d.txt", sanitizeFileName(toolName), time.Now().UnixNano())
	filePath := filepath.Join(sessionDir, filename)
	if err := os.WriteFile(filePath, []byte(result), 0644); err != nil {
		return &PersistedToolResult{
			ToolName: toolName,
			FullSize: charCount,
			Preview:  truncatePreview("", previewChars),
			FilePath: "",
		}
	}

	return &PersistedToolResult{
		ToolName: toolName,
		FullSize: charCount,
		Preview: truncatePreview(result, previewChars),
		FilePath: filePath,
	}
}

func PersistedResultTag(p *PersistedToolResult) string {
	if p == nil {
		return ""
	}
	if p.FilePath == "" {
		return fmt.Sprintf(
			"[Result from %s: %d chars, truncated for context budget]\n%s\n[End of truncated result]",
			p.ToolName, p.FullSize, p.Preview,
		)
	}
	return fmt.Sprintf(
		"[Result from %s: %d chars total, persisted to disk]\nPreview:\n%s\n\nFull result saved at: %s\nTo read the full content, use the read tool with path: %s",
		p.ToolName, p.FullSize, p.Preview, p.FilePath, p.FilePath,
	)
}

func truncatePreview(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars]) + "\n... [content truncated, see full file] ..."
}

func sanitizeFileName(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			result = append(result, byte(c))
		} else {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "unnamed"
	}
	return string(result)
}
