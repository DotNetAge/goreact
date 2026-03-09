package main

import (
	"fmt"
	"context"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/llm/ollama"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct with Ollama Example ===")

	// 创建 Ollama 客户端
	ollamaClient := ollama.NewOllamaClient(
		ollama.WithModel("qwen3:0.6b"),
		ollama.WithTemperature(0.7),
		ollama.WithBaseURL("http://localhost:11434"),
	)

	fmt.Printf("Using Ollama model: %s\n", ollamaClient.GetModel())
	fmt.Printf("Ollama URL: %s\n\n", ollamaClient.GetBaseURL())

	// 创建引擎
	eng := engine.New(
		engine.WithLLMClient(ollamaClient),
		engine.WithMaxIterations(10),
	)

	// 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	// 执行任务
	fmt.Println("Task: Calculate 25 + 17")
	fmt.Println()

	result := eng.Execute(context.Background(), "Calculate 25 + 17", nil)

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
