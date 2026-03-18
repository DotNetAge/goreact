package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/tools"
)

// DateTime 日期时间工具
type DateTime struct{}

// NewDateTime 创建日期时间工具
func NewDateTime() tools.Tool {
	return &DateTime{}
}

// Name 返回工具名称
func (d *DateTime) Name() string {
	return "datetime"
}

// Description 返回工具描述
func (d *DateTime) Description() string {
	return "Date and time operations. Params: {operation: 'now'|'format'|'parse', format: 'layout', value: 'time_string'}"
}

// Execute 执行日期时间操作
// SecurityLevel returns the tool's security risk level
func (t *DateTime) SecurityLevel() tools.SecurityLevel {
    return tools.LevelSafe // Default, needs manual update for risky tools
}

func (d *DateTime) Execute(ctx context.Context, params map[string]any) (any, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	case "now":
		// 返回当前时间
		format := "2006-01-02 15:04:05"
		if f, ok := params["format"].(string); ok {
			format = f
		}
		return time.Now().Format(format), nil

	case "format":
		// 格式化时间
		value, ok := params["value"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'value' parameter")
		}
		format, ok := params["format"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'format' parameter")
		}

		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse time: %w", err)
		}
		return t.Format(format), nil

	case "parse":
		// 解析时间字符串
		value, ok := params["value"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'value' parameter")
		}

		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse time: %w", err)
		}
		return t.Unix(), nil

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}
