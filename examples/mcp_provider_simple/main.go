package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/tool/builtin"
	"github.com/ray/goreact/pkg/tool/provider/mcp"
)

func main() {
	fmt.Println("=== GoReAct: Engine with MCP Provider Integration ===\n")

	// 1. 创建并配置 MCP Provider
	mcpProvider := mcp.NewMCPProvider("weather-service")
	config := map[string]interface{}{
		"server_url": "http://localhost:55987",
		"api_key":    "",
		"timeout":    30,
	}

	if err := mcpProvider.Initialize(config); err != nil {
		fmt.Printf("❌ Failed to initialize MCP Provider: %v\n", err)
		fmt.Println("\n💡 Tip: Make sure the Mock MCP Server is running:")
		fmt.Println("   cd server && go run main.go")
		return
	}
	fmt.Println("✓ MCP Provider initialized\n")

	// 2. 创建 Mock LLM Client
	mockResponses := []string{
		`Thought: I need to get the weather for San Francisco. I'll use the weather tool.
Action: weather
Action Input: {"location": "San Francisco", "unit": "celsius"}`,
		`Thought: Great! I have the weather information. Let me provide the answer.
Final Answer: The weather in San Francisco is 22°C and Sunny with 65% humidity.`,
	}
	llmClient := mock.NewMockClient(mockResponses)

	// 3. 创建 Engine 并直接集成 MCP Provider
	fmt.Println("Creating Engine with integrated MCP Provider...")
	eng := engine.Reactor(
		engine.WithLLMClient(llmClient),
		engine.WithProvider(mcpProvider), // 🎯 直接集成 Provider
		engine.WithMaxIterations(10),
	)

	// 4. 注册内置工具（可选）
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)
	fmt.Println("✓ Engine created with MCP tools automatically registered\n")

	// 5. 执行任务
	task := "What's the weather in San Francisco?"
	fmt.Printf("Task: %s\n\n", task)

	startTime := time.Now()
	result := eng.Execute(context.Background(), task, nil)
	executionTime := time.Since(startTime)

	// 6. 显示结果
	fmt.Println("=== Execution Result ===")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Output: %s\n", result.Output)
	fmt.Printf("Execution Time: %v\n", executionTime)
	fmt.Printf("Total Steps: %d\n\n", len(result.Trace))

	// 7. 显示执行轨迹
	fmt.Println("=== Key Steps ===")
	for _, step := range result.Trace {
		if step.Type == "think" || step.Type == "act" || step.Type == "finish" {
			fmt.Printf("[%s] %s\n", step.Type, step.Content)
		}
	}

	fmt.Println("\n✅ Demo completed!")
}
