package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/tool/builtin"
	"github.com/ray/goreact/pkg/tool/provider"
	"github.com/ray/goreact/pkg/tool/provider/mcp"
)

func main() {
	fmt.Println("=== GoReAct: MCP Provider Integration Demo ===\n")

	// 1. 创建 Provider Registry
	registry := provider.NewRegistry()

	// 2. 创建并初始化 MCP Provider
	fmt.Println("Initializing MCP Provider...")
	mcpProvider := mcp.NewMCPProvider("weather-service")

	config := map[string]interface{}{
		"server_url": "http://localhost:55987",
		"api_key":    "",
		"timeout":    30,
	}

	if err := mcpProvider.Initialize(config); err != nil {
		fmt.Printf("❌ Failed to initialize MCP Provider: %v\n", err)
		fmt.Println("\n💡 Tip: Make sure the Mock MCP Server is running:")
		fmt.Println("   go run mock_server.go")
		return
	}
	fmt.Println("✓ MCP Provider initialized successfully\n")

	// 3. 注册 Provider
	if err := registry.Register(mcpProvider); err != nil {
		fmt.Printf("❌ Failed to register provider: %v\n", err)
		return
	}

	// 4. 从 MCP Server 发现工具
	fmt.Println("Discovering tools from MCP Server...")
	tools, err := mcpProvider.DiscoverTools()
	if err != nil {
		fmt.Printf("❌ Failed to discover tools: %v\n", err)
		return
	}

	fmt.Printf("✓ Discovered %d tools:\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("  %d. %s - %s\n", i+1, tool.Name(), tool.Description())
	}
	fmt.Println()

	// 5. 创建 Mock LLM Client
	mockResponses := []string{
		`Thought: I need to get the weather information for Tokyo. I'll use the weather tool.
Action: weather
Action Input: {"location": "Tokyo", "unit": "celsius"}`,
		`Thought: I have the weather information. Let me provide the answer.
Final Answer: The weather in Tokyo is 22°C and Sunny with 65% humidity.`,
	}
	llmClient := mock.NewMockClient(mockResponses)

	// 6. 创建 Engine 并注册工具
	fmt.Println("Creating Engine and registering tools...")
	eng := engine.Reactor(
		engine.WithLLMClient(llmClient),
		engine.WithMaxIterations(10),
	)

	// 注册内置工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	// 注册从 MCP 发现的工具
	for _, tool := range tools {
		eng.RegisterTool(tool)
	}
	fmt.Println("✓ All tools registered\n")

	// 7. 执行任务
	fmt.Println("Executing task with MCP tools...")
	task := "What's the weather in Tokyo?"
	fmt.Printf("Task: %s\n\n", task)

	startTime := time.Now()
	result := eng.Execute(context.Background(), task, nil)
	executionTime := time.Since(startTime)

	// 8. 显示结果
	fmt.Println("=== Execution Result ===")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Output: %s\n", result.Output)
	fmt.Printf("Execution Time: %v\n", executionTime)
	fmt.Printf("Total Steps: %d\n\n", len(result.Trace))

	// 9. 显示执行轨迹
	fmt.Println("=== Execution Trace ===")
	for _, step := range result.Trace {
		if step.Type == "think" || step.Type == "act" || step.Type == "finish" {
			fmt.Printf("[%s] %s\n", step.Type, step.Content)
		}
	}

	// 10. 测试其他 MCP 工具
	fmt.Println("\n=== Testing Other MCP Tools ===\n")

	// 测试翻译工具
	fmt.Println("Testing translate tool...")
	translateTool, err := mcpProvider.GetTool("translate")
	if err == nil {
		result, err := translateTool.Execute(map[string]interface{}{
			"text": "Hello World",
			"to":   "zh",
		})
		if err == nil {
			fmt.Printf("✓ Translation result: %v\n", result)
		}
	}

	// 测试搜索工具
	fmt.Println("\nTesting search tool...")
	searchTool, err := mcpProvider.GetTool("search")
	if err == nil {
		result, err := searchTool.Execute(map[string]interface{}{
			"query": "GoReAct framework",
			"limit": 3,
		})
		if err == nil {
			fmt.Printf("✓ Search result: %v\n", result)
		}
	}

	// 11. 显示 Provider 状态
	fmt.Println("\n=== Provider Status ===")
	fmt.Printf("Provider Name: %s\n", mcpProvider.Name())
	fmt.Printf("Health Status: %v\n", mcpProvider.IsHealthy())
	fmt.Printf("Registered Providers: %v\n", registry.List())

	// 12. 清理
	fmt.Println("\nClosing connections...")
	if err := registry.Close(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}
	fmt.Println("✓ Done")
}
