package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// Calculator 计算器工具
type Calculator struct{}

// NewCalculatorTool 创建计算器工具
func NewCalculatorTool() core.FuncTool {
	return &Calculator{}
}

func (c *Calculator) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:          "calculator",
		Description:   "Performs basic arithmetic operations. Params: {operation: 'add'|'subtract'|'multiply'|'divide', a: number, b: number}",
		SecurityLevel: core.LevelSafe,
	}
}

func (c *Calculator) Execute(ctx context.Context, params map[string]any) (any, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	a, ok := ToFloat64(params["a"])
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'a' parameter")
	}

	b, ok := ToFloat64(params["b"])
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
