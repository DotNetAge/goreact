package main

import (
	"context"
	"fmt"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct Simple Example ===")

	// 创建 Mock LLM 客户端
	mockResponses := []string{
		"Thought: I should echo the message.\nAction: echo\nParameters: {\"message\": \"Hello, GoReAct!\"}\nReasoning: Using echo tool to respond",
		"Thought: Task completed.\nFinal Answer: Successfully echoed the message: Echo: Hello, GoReAct!",
	}
	mockLLM := mock.NewMockClient(mockResponses)

	// 创建引擎
	eng := engine.Reactor(
		engine.WithMaxIterations(5),
		engine.WithLLMClient(mockLLM),
	)

	// 注册工具
	eng.RegisterTool(builtin.NewEcho())

	// 执行任务
	result := eng.Execute(context.Background(), "Echo the message: Hello, GoReAct!", nil)

	// 打印结果
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Output: %s\n\n", result.Output)

	// 打印执行轨迹
	fmt.Println("Execution Trace:")
	for _, step := range result.Trace {
		fmt.Printf("[Step %d - %s] %s\n", step.Step, step.Type, step.Content)
	}

	fmt.Printf("\nExecution time: %v\n", result.EndTime.Sub(result.StartTime))
}
