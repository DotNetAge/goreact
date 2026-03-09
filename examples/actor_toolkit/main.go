package main

import (
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/actor/debug"
	"github.com/ray/goreact/pkg/actor/resultfmt"
	"github.com/ray/goreact/pkg/actor/schema"
	"github.com/ray/goreact/pkg/actor/wrapper"
)

// ============================================================
// 示例 1：Schema-based Tool（告别手写参数验证）
// ============================================================

func NewCalculator() *schema.Tool {
	return schema.NewTool(
		"calculator",
		"Perform arithmetic operations",
		schema.Define(
			schema.Param("operation", schema.String, "The operation to perform").
				Enum("add", "subtract", "multiply", "divide").Required(),
			schema.Param("a", schema.Number, "First operand").Required(),
			schema.Param("b", schema.Number, "Second operand").Required(),
		),
		func(p schema.ValidatedParams) (any, error) {
			op := p.GetString("operation")
			a := p.GetFloat64("a")
			b := p.GetFloat64("b")

			switch op {
			case "add":
				return a + b, nil
			case "subtract":
				return a - b, nil
			case "multiply":
				return a * b, nil
			case "divide":
				if b == 0 {
					return nil, schema.NewUserError("division by zero")
				}
				return a / b, nil
			}
			return nil, schema.NewUserError("unknown operation: %s", op)
		},
	)
}

func NewWeatherTool() *schema.Tool {
	return schema.NewTool(
		"weather",
		"Get weather forecast for a city",
		schema.Define(
			schema.Param("city", schema.String, "City name").Required(),
			schema.Param("unit", schema.String, "Temperature unit").
				Enum("celsius", "fahrenheit").Default("celsius"),
			schema.Param("days", schema.Int, "Forecast days (1-7)").
				Range(1, 7).Default(1),
		),
		func(p schema.ValidatedParams) (any, error) {
			city := p.GetString("city")
			unit := p.GetString("unit")
			days := p.GetInt("days")
			return fmt.Sprintf("Weather for %s: 25°%s, forecast %d days", city, unit[:1], days), nil
		},
	)
}

func main() {
	fmt.Println("=== Actor Toolkit 示例 ===\n")

	// ============================================================
	// 1. Schema-based Tool 基本使用
	// ============================================================
	fmt.Println("--- 1. Schema-based Tool ---\n")

	calc := NewCalculator()

	// 正常调用
	result, err := calc.Execute(map[string]any{
		"operation": "add",
		"a":         100,
		"b":         200,
	})
	fmt.Printf("100 + 200 = %v (err: %v)\n", result, err)

	// LLM 返回字符串数字（自动转换）
	result, err = calc.Execute(map[string]any{
		"operation": "multiply",
		"a":         "6", // 字符串！自动转换
		"b":         "7", // 字符串！自动转换
	})
	fmt.Printf("6 * 7 = %v (err: %v)\n", result, err)

	// 缺少必需参数（自动验证）
	_, err = calc.Execute(map[string]any{
		"operation": "add",
		"a":         100,
		// 缺少 "b"
	})
	fmt.Printf("缺少参数: %v\n", err)

	// 无效的 enum 值（自动验证）
	_, err = calc.Execute(map[string]any{
		"operation": "power", // 不在 enum 中
		"a":         2,
		"b":         3,
	})
	fmt.Printf("无效操作: %v\n", err)

	// 类型错误（自动验证）
	_, err = calc.Execute(map[string]any{
		"operation": "add",
		"a":         "hello", // 无法转换为数字
		"b":         200,
	})
	fmt.Printf("类型错误: %v\n", err)

	// ============================================================
	// 2. 默认值和可选参数
	// ============================================================
	fmt.Println("\n--- 2. 默认值和可选参数 ---\n")

	weather := NewWeatherTool()

	// 使用默认值
	result, _ = weather.Execute(map[string]any{
		"city": "Beijing",
		// unit 默认 "celsius"，days 默认 1
	})
	fmt.Printf("默认值: %v\n", result)

	// 覆盖默认值
	result, _ = weather.Execute(map[string]any{
		"city": "New York",
		"unit": "fahrenheit",
		"days": 3,
	})
	fmt.Printf("自定义: %v\n", result)

	// 范围检查
	_, err = weather.Execute(map[string]any{
		"city": "Tokyo",
		"days": 30, // 超出范围 1-7
	})
	fmt.Printf("范围错误: %v\n", err)

	// ============================================================
	// 3. Schema 自动生成工具描述
	// ============================================================
	fmt.Println("\n--- 3. 自动生成工具描述 ---\n")

	fmt.Printf("Name: %s\n", calc.Name())
	fmt.Printf("Description: %s\n", calc.Description())
	fmt.Printf("Schema JSON:\n%s\n", calc.SchemaJSON())

	// ============================================================
	// 4. Wrapper：超时 + 重试
	// ============================================================
	fmt.Println("\n--- 4. Execution Wrappers ---\n")

	// 模拟一个可能超时的工具
	slowTool := schema.NewTool(
		"slow_api",
		"A slow API call",
		schema.Define(
			schema.Param("url", schema.String, "API URL").Required(),
		),
		func(p schema.ValidatedParams) (any, error) {
			time.Sleep(100 * time.Millisecond) // 模拟慢请求
			return "response from " + p.GetString("url"), nil
		},
	)

	// 添加超时包装
	withTimeout := wrapper.WithTimeout(50 * time.Millisecond).Wrap(slowTool)
	_, err = withTimeout.Execute(map[string]any{"url": "https://api.example.com"})
	fmt.Printf("超时测试: %v\n", err)

	// 添加重试包装
	callCount := 0
	flaky := schema.NewTool(
		"flaky_api",
		"An unreliable API",
		schema.Define(
			schema.Param("url", schema.String, "API URL").Required(),
		),
		func(p schema.ValidatedParams) (any, error) {
			callCount++
			if callCount < 3 {
				return nil, fmt.Errorf("connection refused")
			}
			return "success on attempt " + fmt.Sprint(callCount), nil
		},
	)

	withRetry := wrapper.WithRetry(3, 10*time.Millisecond).Wrap(flaky)
	result, err = withRetry.Execute(map[string]any{"url": "https://api.example.com"})
	fmt.Printf("重试测试: %v (err: %v, 调用次数: %d)\n", result, err, callCount)

	// 组合包装
	combined := wrapper.Wrap(slowTool,
		wrapper.WithTimeout(200*time.Millisecond),
		wrapper.WithRetry(2, 10*time.Millisecond),
	)
	result, err = combined.Execute(map[string]any{"url": "https://api.example.com"})
	fmt.Printf("组合包装: %v (err: %v)\n", result, err)

	// ============================================================
	// 5. Result Formatter
	// ============================================================
	fmt.Println("\n--- 5. Result Formatter ---\n")

	formatter := resultfmt.New(
		resultfmt.WithMaxLength(50),
	)

	// 短结果：原样返回
	short := formatter.Format("Hello, World!")
	fmt.Printf("短结果: %s\n", short)

	// 长结果：自动截断
	longText := "This is a very long response that contains a lot of information and should be truncated to save tokens and improve LLM comprehension."
	truncated := formatter.Format(longText)
	fmt.Printf("长结果: %s\n", truncated)

	// 错误格式化
	errFmt := resultfmt.NewErrorFormatter()
	friendly := errFmt.Format(
		fmt.Errorf("dial tcp 93.184.216.34:443: connect: connection refused"),
		"http",
		map[string]any{"url": "https://api.example.com"},
	)
	fmt.Printf("友好错误:\n%s\n", friendly)

	fmt.Println("\n=== 示例完成 ===")

	// ============================================================
	// 6. Execution Tracer（调试工具）
	// ============================================================
	fmt.Println("\n--- 6. Execution Tracer ---\n")

	tracer := debug.NewExecutionTracer(true)

	// 包装工具
	tracedCalc := debug.WithTracing(tracer).Wrap(calc)

	// 执行并自动追踪
	result, err = tracedCalc.Execute(map[string]any{
		"operation": "multiply",
		"a":         12,
		"b":         34,
	})
	fmt.Printf("结果: %v (err: %v)\n", result, err)

	// 查看追踪信息
	fmt.Println("\n追踪报告:")
	fmt.Println(tracer.Report())

	// 性能分析
	fmt.Println("\n性能分析:")
	profiler := debug.NewPerformanceProfiler()
	profiledCalc := debug.WithProfiling(profiler).Wrap(calc)

	// 执行多次
	for i := 0; i < 5; i++ {
		profiledCalc.Execute(map[string]any{
			"operation": "add",
			"a":         i,
			"b":         i * 2,
		})
	}

	fmt.Println(profiler.Report())
}
