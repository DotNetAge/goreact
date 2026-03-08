package main

import (
	"fmt"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/llm/mock"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct Calculator Example ===")

	// 创建 Mock LLM 客户端（会根据 prompt 智能生成响应）
	mockLLM := mock.NewMockClient([]string{})

	// 创建引擎
	eng := engine.New(
		engine.WithMaxIterations(10),
		engine.WithLLMClient(mockLLM),
	)

	// 注册计算器工具
	eng.RegisterTool(builtin.NewCalculator())

	// 执行任务：计算 15 * 23 + 7
	fmt.Println("Task: Calculate 15 * 23 + 7")
	fmt.Println()

	result := eng.Execute("Calculate 15 * 23 + 7", nil)

	// 打印结果
	fmt.Printf("\nSuccess: %v\n", result.Success)
	if result.Error != nil {
		fmt.Printf("Error: %v\n", result.Error)
	}
	fmt.Printf("Output: %s\n\n", result.Output)

	// 打印执行轨迹
	fmt.Println("Execution Trace:")
	fmt.Println("================")
	for _, step := range result.Trace {
		fmt.Printf("[Step %d - %s]\n", step.Step, step.Type)
		fmt.Printf("  %s\n", step.Content)
		fmt.Println()
	}

	fmt.Printf("Total execution time: %v\n", result.EndTime.Sub(result.StartTime))
}
