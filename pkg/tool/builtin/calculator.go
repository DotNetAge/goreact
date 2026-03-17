package builtin

import (
	"fmt"

	"github.com/ray/goreact/pkg/tools"
)

// Calculator 计算器工具
type Calculator struct{}

// NewCalculator 创建计算器工具
func NewCalculator() tools.Tool {
	return &Calculator{}
}

// Name 返回工具名称
func (c *Calculator) Name() string {
	return "calculator"
}

// Description 返回工具描述
func (c *Calculator) Description() string {
	return "Performs basic arithmetic operations. Params: {operation: 'add'|'subtract'|'multiply'|'divide', a: number, b: number}"
}

// Execute 执行计算
func (c *Calculator) Execute(params map[string]interface{}) (interface{}, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	a, ok := toFloat64(params["a"])
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'a' parameter")
	}

	b, ok := toFloat64(params["b"])
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'b' parameter")
	}

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	return result, nil
}

// toFloat64 将 interface{} 转换为 float64
func toFloat64(v interface{}) (float64, bool) {
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
