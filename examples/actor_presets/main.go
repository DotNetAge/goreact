package main

import (
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/actor/debug"
	"github.com/ray/goreact/pkg/actor/schema"
	"github.com/ray/goreact/pkg/core"
	actorpresets "github.com/ray/goreact/pkg/core/actor/presets"
	"github.com/ray/goreact/pkg/tools"
	"github.com/ray/goreact/pkg/types"
)

func newCalculator() *schema.Tool {
	return schema.NewTool(
		"calculator",
		"Perform arithmetic operations",
		schema.Define(
			schema.Param("operation", schema.String, "The operation").
				Enum("add", "subtract", "multiply", "divide").Required(),
			schema.Param("a", schema.Number, "First operand").Required(),
			schema.Param("b", schema.Number, "Second operand").Required(),
		),
		func(p schema.ValidatedParams) (any, error) {
			a := p.GetFloat64("a")
			b := p.GetFloat64("b")
			switch p.GetString("operation") {
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
			return nil, schema.NewUserError("unknown operation")
		},
	)
}

func newSlowTool() tools.Tool {
	return schema.NewTool(
		"slow_api",
		"A slow API call",
		schema.Define(
			schema.Param("url", schema.String, "URL").Required(),
		),
		func(p schema.ValidatedParams) (any, error) {
			time.Sleep(50 * time.Millisecond)
			return "response from " + p.GetString("url"), nil
		},
	)
}

func main() {
	fmt.Println("=== Actor Presets 示例 ===")

	// 准备工具
	tm := tools.NewManager()
	tm.RegisterTools(newCalculator(), newSlowTool())

	ctx := core.NewContext()
	action := &types.Action{
		ToolName:   "calculator",
		Parameters: map[string]any{"operation": "add", "a": 100, "b": 200},
	}

	// ============================================================
	// 1. ResilientActor - 弹性模式（超时 + 重试）
	// ============================================================
	fmt.Println("\n--- 1. ResilientActor ---")

	resilient := actorpresets.NewResilientActor(tm,
		actorpresets.WithTimeout(5*time.Second),
		actorpresets.WithRetry(3, 500*time.Millisecond),
	)

	result, err := resilient.Act(action, ctx)
	fmt.Printf("结果: %v, 成功: %v, err: %v\n", result.Output, result.Success, err)

	// ============================================================
	// 2. DebugActor - 调试模式（完整追踪）
	// ============================================================
	fmt.Println("\n--- 2. DebugActor ---")

	tracer := debug.NewExecutionTracer(true)
	profiler := debug.NewPerformanceProfiler()

	debugActor := actorpresets.NewDebugActor(tm, tracer, profiler)

	// 执行多次
	debugActor.Act(action, ctx)
	debugActor.Act(&types.Action{
		ToolName:   "calculator",
		Parameters: map[string]any{"operation": "multiply", "a": 6, "b": 7},
	}, ctx)
	debugActor.Act(&types.Action{
		ToolName:   "calculator",
		Parameters: map[string]any{"operation": "divide", "a": 10, "b": 0},
	}, ctx)

	fmt.Println("\n追踪报告:")
	fmt.Println(tracer.Report())
	fmt.Println("性能报告:")
	fmt.Println(profiler.Report())

	// ============================================================
	// 3. SafeActor - 安全模式（工具白名单）
	// ============================================================
	fmt.Println("--- 3. SafeActor ---")

	safe := actorpresets.NewSafeActor(tm,
		actorpresets.WithAllowedTools("calculator"),
	)

	// 允许的工具
	result, err = safe.Act(action, ctx)
	fmt.Printf("calculator: 成功=%v, err=%v\n", result.Success, err)

	// 不允许的工具
	result, err = safe.Act(&types.Action{
		ToolName:   "slow_api",
		Parameters: map[string]any{"url": "https://evil.com"},
	}, ctx)
	fmt.Printf("slow_api: 成功=%v, err=%v\n", result.Success, err)

	// ============================================================
	// 4. ProductionActor - 生产模式（全部最佳实践）
	// ============================================================
	fmt.Println("\n--- 4. ProductionActor ---")

	production := actorpresets.NewProductionActor(tm)

	result, err = production.Act(action, ctx)
	fmt.Printf("结果: %v, 成功: %v, err: %v\n", result.Output, result.Success, err)

	// 测试结果格式化（大输出自动截断）
	result, err = production.Act(&types.Action{
		ToolName:   "slow_api",
		Parameters: map[string]any{"url": "https://api.example.com"},
	}, ctx)
	fmt.Printf("结果: %v, 成功: %v, err: %v\n", result.Output, result.Success, err)

	fmt.Println("\n=== 示例完成 ===")
}
