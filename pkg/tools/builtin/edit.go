package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ray/goreact/pkg/tools"
)

// Edit 文件编辑工具（支持多位置精确编辑）
type Edit struct{}

// NewEdit 创建文件编辑工具
func NewEdit() tools.Tool {
	return &Edit{}
}

// Name 返回工具名称
func (e *Edit) Name() string {
	return "edit"
}

// Description 返回工具描述
func (e *Edit) Description() string {
	return "精确编辑文件内容。支持多位置、diff 式修改。Params: {path: '文件路径', edits: [{old_text: '原文本', new_text: '新文本'}, ...]}"
}

// Execute 执行文件编辑
// SecurityLevel returns the tool's security risk level
func (t *Edit) SecurityLevel() tools.SecurityLevel {
    return tools.LevelSensitive // Default, needs manual update for risky tools
}

func (e *Edit) Execute(ctx context.Context, params map[string]any) (any, error) {
	path, err := validateRequiredString(params, "path")
	if err != nil {
		return nil, err
	}

	// 读取原始内容
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	currentContent := originalContent

	// 获取编辑列表
	editsRaw, ok := params["edits"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: edits")
	}

	edits, ok := editsRaw.([]any)
	if !ok || len(edits) == 0 {
		return nil, fmt.Errorf("edits must be a non-empty array")
	}

	// 应用所有编辑
	editedRegions := make([]map[string]any, 0)
	for i, editRaw := range edits {
		edit, ok := editRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("edit[%d] must be an object with old_text and new_text", i)
		}

		oldText, ok1 := edit["old_text"].(string)
		newText, ok2 := edit["new_text"].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("edit[%d] must have 'old_text' and 'new_text' string fields", i)
		}

		// 查找并替换
		if !strings.Contains(currentContent, oldText) {
			return nil, fmt.Errorf("edit[%d]: text not found in file:\n%s", i, truncateString(oldText, 200))
		}

		// 执行替换
		currentContent = strings.Replace(currentContent, oldText, newText, 1)

		// 记录编辑区域
		editedRegions = append(editedRegions, map[string]any{
			"index":      i,
			"old_length": len(oldText),
			"new_length": len(newText),
			"delta":      len(newText) - len(oldText),
		})
	}

	// 写入修改后的内容
	if err := os.WriteFile(path, []byte(currentContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write edited file: %w", err)
	}

	// 计算统计信息
	totalOldLength := 0
	totalNewLength := 0
	for _, region := range editedRegions {
		totalOldLength += int(region["old_length"].(int))
		totalNewLength += int(region["new_length"].(int))
	}

	return map[string]any{
		"success":        true,
		"path":           path,
		"edits_applied":  len(editedRegions),
		"original_size":  len(originalContent),
		"new_size":       len(currentContent),
		"size_delta":     len(currentContent) - len(originalContent),
		"edited_regions": editedRegions,
		"message":        fmt.Sprintf("Successfully applied %d edit(s)", len(editedRegions)),
	}, nil
}

// truncateString 截断字符串（用于错误消息）
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
