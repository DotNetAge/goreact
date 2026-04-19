package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/DotNetAge/goreact/core"
)

// Read 文件读取工具
type Read struct {
	info   *core.ToolInfo
	limits core.FileReadingLimits
}

const readDescription = `Reads a file from the local filesystem. You can access any file directly by using this tool.

Usage:
- The path parameter must be an absolute path, not a relative path.
- By default, it reads up to 500 lines starting from the beginning of the file.
- You can optionally specify offset and limit parameters (especially handy for long files), but it's recommended to read the whole file by not providing these parameters.
- When you already know which part of the file you need, only read that part. This can be important for larger files.
- Results are returned using cat -n format, with line numbers starting at 1.
- This tool can only read files, not directories. To read a directory, use an ls command via the bash tool.
- CRITICAL: filePath must be returned before other fields.

IMPORTANT: Large files may be truncated for context budget. Use offset and limit to read specific sections.`

// NewReadTool 创建文件读取工具
func NewReadTool() core.FuncTool {
	return NewReadToolWithLimits(core.DefaultFileReadingLimits())
}

// NewReadToolWithLimits creates a read tool with custom limits.
func NewReadToolWithLimits(limits core.FileReadingLimits) core.FuncTool {
	return &Read{
		limits: limits,
		info: &core.ToolInfo{
			Name:          "read",
			Description:   readDescription,
			SecurityLevel: core.LevelSafe,
			IsReadOnly:    true,
			// Read tool gets special treatment: MaxResultSizeChars = -1 means
			// never persist to disk (the tool already supports offset/limit pagination).
			// Instead, we enforce size limits at read time.
			MaxResultSizeChars: -1,
			Parameters: []core.Parameter{
				{
					Name:        "path",
					Type:        "string",
					Required:    true,
					Description: "The absolute path to the file to read.",
				},
				{
					Name:        "offset",
					Type:        "integer",
					Required:    false,
					Description: "The line number to start reading from (1-based).",
				},
				{
					Name:        "limit",
					Type:        "integer",
					Required:    false,
					Description: "The maximum number of lines to read. Defaults to 500.",
				},
			},
		},
	}
}

func (r *Read) Info() *core.ToolInfo {
	return r.info
}

func (r *Read) Execute(ctx context.Context, params map[string]any) (any, error) {
	path, err := ValidateRequiredString(params, "path")
	if err != nil {
		return nil, err
	}

	// 安全检查
	if err := ValidateFileSafety(path); err != nil {
		return nil, err
	}

	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", path)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// 如果是目录，返回错误
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// 第一层防御：Pre-read check — 文件大小限制 (256KB)
	if r.limits.MaxSizeBytes > 0 && info.Size() > r.limits.MaxSizeBytes {
		return map[string]any{
			"success": false,
			"path":    path,
			"size_bytes": info.Size(),
			"error": fmt.Sprintf(
				"file too large (%.2f KB), maximum allowed is %d KB. "+
					"Use offset and limit parameters to read specific sections, "+
					"or use grep/glob to locate the relevant parts first.",
				float64(info.Size())/1024, r.limits.MaxSizeBytes/1024),
		}, nil
	}

	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 获取分页参数
	startLine := 1
	if offset, ok := ToFloat64(params["offset"]); ok && offset > 0 {
		startLine = int(offset)
	}

	maxLines := r.limits.DefaultLines
	if limit, ok := ToFloat64(params["limit"]); ok && limit > 0 {
		maxLines = int(limit)
	}
	endLine := startLine + maxLines - 1

	// 读取文件内容
	var content strings.Builder
	scanner := bufio.NewScanner(file)
	// Increase scanner buffer for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	linesRead := 0
	totalLines := 0

	// First pass: count total lines
	file.Seek(0, 0)
	for scanner.Scan() {
		totalLines++
	}
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lineNum++

		if lineNum < startLine {
			continue
		}
		if lineNum > endLine {
			break
		}

		content.WriteString(fmt.Sprintf("%d\t%s\n", lineNum, scanner.Text()))
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 第二层防御：Post-read check — Token 估算
	outputChars := content.Len()
	estimatedTokens := outputChars / 3 // rough estimate
	if r.limits.MaxTokens > 0 && estimatedTokens > r.limits.MaxTokens {
		// Truncate to fit within token budget
		targetChars := r.limits.MaxTokens * 3
		runes := []rune(content.String())
		if len(runes) > targetChars {
			content.Reset()
			content.WriteString(string(runes[:targetChars]))
			content.WriteString("\n... [truncated: output exceeds token budget] ...")
		}
	}

	// 构建响应
	result := map[string]any{
		"success":     true,
		"path":        path,
		"size_bytes":  info.Size(),
		"lines_read":  linesRead,
		"total_lines": totalLines,
		"content":     content.String(),
		"start_line":  startLine,
	}

	if linesRead >= maxLines && lineNum < totalLines {
		result["has_more"] = true
		result["next_offset"] = endLine + 1
	}

	return result, nil
}
