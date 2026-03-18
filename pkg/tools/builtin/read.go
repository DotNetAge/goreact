package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ray/goreact/pkg/tools"
)

// Read 文件读取工具
type Read struct{}

// NewRead 创建文件读取工具
func NewRead() tools.Tool {
	return &Read{}
}

// Name 返回工具名称
func (r *Read) Name() string {
	return "read"
}

// Description 返回工具描述
func (r *Read) Description() string {
	return "读取文件内容。支持指定行范围、自动检测编码。Params: {path: '文件路径', start_line?: 起始行，end_line?: 结束行}"
}

// Execute 执行文件读取
func (r *Read) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, err := validateRequiredString(params, "path")
	if err != nil {
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
		if linesRead == 0 {
			content.WriteString(fmt.Sprintf("%d\t%s\n", lineNum, scanner.Text()))
		} else {
			content.WriteString(fmt.Sprintf("%d\t%s\n", lineNum-startLine+1, scanner.Text()))
		}
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 构建响应
	result := map[string]interface{}{
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
