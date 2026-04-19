package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// FileEditTool implements a tool for editing files with staleness checks.
type FileEditTool struct{}

func NewFileEditTool() core.FuncTool {
	return &FileEditTool{}
}

const editDescription = `Performs exact string replacements in files.

Usage:
- You must use your Read tool at least once in the conversation before editing. This tool will error if you attempt an edit without reading the file. 
- When editing text from Read tool output, ensure you preserve the exact indentation (tabs/spaces) as it appears AFTER the line number prefix. Everything after that is the actual file content to match. Never include any part of the line number prefix in the old_string or new_string.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- The edit will FAIL if old_string is not unique in the file. Either provide a larger string with more surrounding context to make it unique or use replace_all to change every instance.
- Use replace_all for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.`

func (t *FileEditTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "file_edit",
		Description: editDescription,
		Parameters: []core.Parameter{
			{
				Name:        "path",
				Type:        "string",
				Description: "Path to the file to edit.",
				Required:    true,
			},
			{
				Name:        "old_string",
				Type:        "string",
				Description: "The exact string to replace.",
				Required:    true,
			},
			{
				Name:        "new_string",
				Type:        "string",
				Description: "The new string to insert.",
				Required:    true,
			},
			{
				Name:        "replace_all",
				Type:        "boolean",
				Description: "Whether to replace all occurrences.",
				Required:    false,
			},
			{
				Name:        "last_read_time",
				Type:        "string",
				Description: "Optional: The timestamp of when the file was last read (to prevent stale writes).",
				Required:    false,
			},
		},
	}
}

func (t *FileEditTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	filePath, err := ValidateRequiredString(params, "path")
	if err != nil {
		return nil, err
	}

	// 安全检查
	if err := ValidateFileSafety(filePath); err != nil {
		return nil, err
	}

	oldStr, err := ValidateRequiredString(params, "old_string")
	if err != nil {
		return nil, err
	}

	newStr, err := ValidateRequiredString(params, "new_string")
	if err != nil {
		return nil, err
	}

	replaceAll, _ := params["replace_all"].(bool)
	lastReadTimeStr, _ := params["last_read_time"].(string)

	// Staleness check
	if lastReadTimeStr != "" {
		info, err := os.Stat(filePath)
		if err == nil {
			lastReadTime, parseErr := time.Parse(time.RFC3339, lastReadTimeStr)
			if parseErr == nil && info.ModTime().After(lastReadTime) {
				return nil, fmt.Errorf("file has been modified since it was last read. please read it again before editing")
			}
		}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileContent := string(content)
	if !strings.Contains(fileContent, oldStr) && oldStr != "" {
		return nil, fmt.Errorf("old_string not found in file")
	}

	var updatedContent string
	if replaceAll {
		updatedContent = strings.ReplaceAll(fileContent, oldStr, newStr)
	} else {
		updatedContent = strings.Replace(fileContent, oldStr, newStr, 1)
	}

	err = os.WriteFile(filePath, []byte(updatedContent), 0644)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("File %s updated successfully.", filePath), nil
}
