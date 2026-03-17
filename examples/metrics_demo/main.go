package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct: Metrics 使用示例 ===\n")

	// 1. 创建自定义 Metrics 收集器（也可以使用默认的）
	fmt.Println("Step 1: 创建 Metrics 收集器")
	metricsCollector := metrics.NewDefaultMetrics()
	fmt.Println("✓ Metrics 收集器创建完成\n")

	// 2. 创建 Mock LLM Client
	fmt.Println("Step 2: 创建 Engine")
	mockResponses := []string{
		`Thought: I need to calculate 10 + 20.
Action: calculator
Parameters: {"operation": "add", "a": 10, "b": 20}`,
		`Thought: The result is 30.
Final Answer: The sum of 10 and 20 is 30.`,
	}
	mockLLM := mock.NewMockClient(mockResponses)

	// 3. 创建 Engine（注入 Metrics）
	eng := engine.Reactor(
		engine.WithLLMClient(mockLLM),
		engine.WithMetrics(metricsCollector), // 注入 Metrics 收集器
		engine.WithMaxIterations(10),
	)

	// 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	fmt.Println("✓ Engine 创建完成，已注入 Metrics 收集器\n")

	// 4. 执行多个任务
	fmt.Println("Step 3: 执行任务")
	fmt.Println("─────────────────────────────────────────")

	tasks := []string{
		"Calculate 10 + 20",
		"Calculate 5 * 6",
		"Echo hello world",
	}

	for i, task := range tasks {
		fmt.Printf("\nTask %d: %s\n", i+1, task)
		startTime := time.Now()
		result := eng.Execute(context.Background(), task, nil)
		executionTime := time.Since(startTime)

		fmt.Printf("  Success: %v\n", result.Success)
		fmt.Printf("  Output: %s\n", result.Output)
		fmt.Printf("  Execution Time: %v\n", executionTime)
	}

	// 5. 获取并显示 Metrics
	fmt.Println("\n\n=== Metrics 统计 ===")
	allMetrics := metricsCollector.GetMetrics()

	// 格式化输出
	metricsJSON, _ := json.MarshalIndent(allMetrics, "", "  ")
	fmt.Println(string(metricsJSON))

	// 6. 解析并显示关键指标
	fmt.Println("\n=== 关键指标摘要 ===")

	// 延迟指标
	if latencies, ok := allMetrics["latencies"].(map[string]interface{}); ok {
		fmt.Println("\n延迟统计:")
		for operation, metrics := range latencies {
			if m, ok := metrics.(map[string]interface{}); ok {
				fmt.Printf("  %s:\n", operation)
				fmt.Printf("    调用次数: %v\n", m["count"])
				fmt.Printf("    平均延迟: %v\n", m["avg"])
				fmt.Printf("    最小延迟: %v\n", m["min"])
				fmt.Printf("    最大延迟: %v\n", m["max"])
			}
		}
	}

	// 成功指标
	if successes, ok := allMetrics["successes"].(map[string]int64); ok {
		fmt.Println("\n成功统计:")
		for operation, count := range successes {
			fmt.Printf("  %s: %d 次\n", operation, count)
		}
	}

	// 错误指标
	if errors, ok := allMetrics["errors"].(map[string]int64); ok {
		fmt.Println("\n错误统计:")
		if len(errors) == 0 {
			fmt.Println("  无错误")
		} else {
			for operation, count := range errors {
				fmt.Printf("  %s: %d 次\n", operation, count)
			}
		}
	}

	fmt.Println("\n✅ Metrics 演示完成！")
	fmt.Println("\n说明：")
	fmt.Println("1. Metrics 接口已经定义在 pkg/metrics/metrics.go")
	fmt.Println("2. Engine 内部已经在关键节点记录指标：")
	fmt.Println("   - RecordLatency: 记录每次 Execute 的延迟")
	fmt.Println("   - RecordSuccess: 记录成功的操作")
	fmt.Println("   - RecordError: 记录失败的操作")
	fmt.Println("3. 可以通过 WithMetrics() 注入自定义的 Metrics 实现")
	fmt.Println("4. 使用 GetMetrics() 获取所有收集的指标数据")
}
