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
	info *core.ToolInfo
}

const readDescription = `Reads a file from the local filesystem. You can access any file directly by using this tool.
Assume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.

Usage:
- The path parameter must be an absolute path, not a relative path.
- By default, it reads up to 2000 lines starting from the beginning of the file.
- You can optionally specify a line offset and limit (especially handy for long files), but it's recommended to read the whole file by not providing these parameters.
- When you already know which part of the file you need, only read that part. This can be important for larger files.
- Results are returned using cat -n format, with line numbers starting at 1.
- This tool can only read files, not directories. To read a directory, use an ls command via the bash tool.`

// NewReadTool 创建文件读取工具
func NewReadTool() core.FuncTool {
	return &Read{
		info: &core.ToolInfo{
			Name:          "read",
			Description:   readDescription,
			SecurityLevel: core.LevelSafe,
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

	// 检查文件大小（限制最大 10MB）
	const maxFileSize = 10 * 1024 * 1024
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large (%.2f MB), maximum is 10MB", float64(info.Size())/1024/1024)
	}

	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 获取行范围参数
	startLine := 0 // 0 表示从头开始
	if start, ok := params["start_line"].(float64); ok {
		startLine = int(start)
	}

	endLine := -1 // -1 表示读到结尾
	if end, ok := params["end_line"].(float64); ok {
		endLine = int(end)
	}

	// 读取文件内容
	var content strings.Builder
	scanner := bufio.NewScanner(file)
	lineNum := 0
	linesRead := 0

	for scanner.Scan() {
		lineNum++

		// 如果还没到起始行，跳过
		if startLine > 0 && lineNum < startLine {
			continue
		}

		// 如果超过结束行，停止
		if endLine > 0 && lineNum > endLine {
			break
		}

		// 添加行号和内容
		content.WriteString(fmt.Sprintf("%d\t%s\n", lineNum, scanner.Text()))
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 构建响应
	result := map[string]any{
		"success":     true,
		"path":        path,
		"size_bytes":  info.Size(),
		"lines_read":  linesRead,
		"total_lines": lineNum,
		"content":     content.String(),
	}

	// 如果指定了行范围，添加到结果中
	if startLine > 0 || endLine > 0 {
		result["start_line"] = max(1, startLine)
		if endLine > 0 {
			result["end_line"] = endLine
		}
	}

	return result, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
