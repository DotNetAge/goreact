package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DiskToolResultStorage implements ToolResultStorage by writing large results
// to the filesystem under a session-scoped directory.
type DiskToolResultStorage struct {
	mu              sync.Mutex
	baseDir         string
	maxResultChars  int
	previewChars    int
}

// StorageOption configures DiskToolResultStorage behavior.
type StorageOption func(*DiskToolResultStorage)

// WithStorageDir sets the base directory for persisted results.
func WithStorageDir(dir string) StorageOption {
	return func(s *DiskToolResultStorage) {
		s.baseDir = dir
	}
}

// WithMaxResultChars overrides the per-result size threshold (in characters).
func WithMaxResultChars(n int) StorageOption {
	return func(s *DiskToolResultStorage) {
		s.maxResultChars = n
	}
}

// WithPreviewChars sets how many characters to keep in the inline preview.
func WithPreviewChars(n int) StorageOption {
	return func(s *DiskToolResultStorage) {
		s.previewChars = n
	}
}

// NewDiskToolResultStorage creates a new storage backed by the local filesystem.
func NewDiskToolResultStorage(opts ...StorageOption) *DiskToolResultStorage {
	s := &DiskToolResultStorage{
		baseDir:        filepath.Join(os.TempDir(), "goreact", "tool-results"),
		maxResultChars: DefaultToolResultLimits().MaxResultSizeChars,
		previewChars:   2000,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Persist saves a tool result to disk if it exceeds the size threshold.
// Returns a PersistedToolResult with preview + path, or nil if inline is fine.
func (s *DiskToolResultStorage) Persist(toolName string, result string) *PersistedToolResult {
	charCount := len([]rune(result))
	if charCount <= s.maxResultChars {
		return nil // small enough to keep inline
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create session-scoped directory
	sessionDir := filepath.Join(s.baseDir, fmt.Sprintf("session_%d", os.Getpid()))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		// Fallback: return truncated inline if disk write fails
		return &PersistedToolResult{
			ToolName: toolName,
			FullSize: charCount,
			Preview:  truncatePreview(result, s.previewChars),
			FilePath: "",
		}
	}

	// Write full result to a unique file
	filename := fmt.Sprintf("%s_%d.txt", sanitizeFileName(toolName), os.Getpid())
	filePath := filepath.Join(sessionDir, filename)
	if err := os.WriteFile(filePath, []byte(result), 0644); err != nil {
		return &PersistedToolResult{
			ToolName: toolName,
			FullSize: charCount,
			Preview:  truncatePreview(result, s.previewChars),
			FilePath: "",
		}
	}

	return &PersistedToolResult{
		ToolName: toolName,
		FullSize: charCount,
		Preview:  truncatePreview(result, s.previewChars),
		FilePath: filePath,
	}
}

// Read retrieves the full content of a previously persisted result.
func (s *DiskToolResultStorage) Read(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read persisted result: %w", err)
	}
	return string(data), nil
}

// Cleanup removes the session's persisted result directory.
func (s *DiskToolResultStorage) Cleanup(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.baseDir, fmt.Sprintf("session_%s", sessionID))
	return os.RemoveAll(dir)
}

// PersistedResultTag wraps the persisted result info into a tagged string
// that the LLM can recognize, following ClueCode's <persisted-output> pattern.
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
