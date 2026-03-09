package main

import (
	"encoding/json"
	"fmt"
	"context"
	"time"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/llm"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/tool/builtin"
)

// MockLLMWithResources 模拟带资源统计的 LLM Client
type MockLLMWithResources struct {
	responses      []string
	callCount      int
	lastTokenUsage *llm.TokenUsage
}

func NewMockLLMWithResources(responses []string) *MockLLMWithResources {
	return &MockLLMWithResources{
		responses: responses,
		callCount: 0,
	}
}

func (m *MockLLMWithResources) Generate(ctx context.Context, prompt string) (string, error) {
	if m.callCount < len(m.responses) {
		response := m.responses[m.callCount]
		m.callCount++

		// 模拟 LLM 计算（消耗一些资源）
		// 在实际场景中，这里会是真实的 LLM 推理
		time.Sleep(10 * time.Millisecond) // 模拟计算时间

		// 模拟 Token 消耗
		promptTokens := len(prompt) / 4
		completionTokens := len(response) / 4
		m.lastTokenUsage = &llm.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}

		return response, nil
	}

	return "Final Answer: Task completed.", nil
}

func (m *MockLLMWithResources) LastTokenUsage() *llm.TokenUsage {
	return m.lastTokenUsage
}

func main() {
	fmt.Println("=== GoReAct: 系统资源监控演示 ===\n")

	// 1. 创建 Metrics 收集器
	fmt.Println("Step 1: 创建 Metrics 收集器")
	metricsCollector := metrics.NewDefaultMetrics()
	fmt.Println("✓ Metrics 收集器创建完成\n")

	// 2. 创建 Logger
	fmt.Println("Step 2: 创建 Logger")
	logger, err := log.NewDevelopmentZapLogger()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}
	fmt.Println("✓ Logger 创建完成\n")

	// 3. 创建 Mock LLM
	fmt.Println("Step 3: 创建 LLM Client")
	mockResponses := []string{
		`Thought: I need to calculate 500 * 600.
Action: calculator
Parameters: {"operation": "multiply", "a": 500, "b": 600}`,
		`Thought: The result is 300000.
Final Answer: The result of 500 * 600 is 300000.`,
	}
	mockLLM := NewMockLLMWithResources(mockResponses)
	fmt.Println("✓ Mock LLM 创建完成\n")

	// 4. 创建 Engine
	fmt.Println("Step 4: 创建 Engine")
	eng := engine.New(
		engine.WithLLMClient(mockLLM),
		engine.WithMetrics(metricsCollector),
		engine.WithLogger(logger),
		engine.WithMaxIterations(10),
	)

	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)
	fmt.Println("✓ Engine 创建完成\n")

	// 5. 执行任务
	fmt.Println("Step 5: 执行任务")
	fmt.Println("─────────────────────────────────────────")
	task := "Calculate 500 * 600"
	fmt.Printf("Task: %s\n\n", task)

	startTime := time.Now()
	result := eng.Execute(context.Background(), task, nil)
	executionTime := time.Since(startTime)

	fmt.Printf("\n执行结果:\n")
	fmt.Printf("  Success: %v\n", result.Success)
	fmt.Printf("  Output: %s\n", result.Output)
	fmt.Printf("  Execution Time: %v\n", executionTime)

	// 6. 显示完整 Metrics
	fmt.Println("\n\n=== 完整 Metrics 统计 ===")
	allMetrics := metricsCollector.GetMetrics()

	metricsJSON, _ := json.MarshalIndent(allMetrics, "", "  ")
	fmt.Println(string(metricsJSON))

	// 7. 解析并显示系统资源指标
	fmt.Println("\n\n=== 系统资源使用详情 ===")
	if resourceUsage, ok := allMetrics["resource_usage"].(map[string]any); ok {
		if len(resourceUsage) == 0 {
			fmt.Println("  无资源使用记录")
		} else {
			for operation, metrics := range resourceUsage {
				if m, ok := metrics.(map[string]any); ok {
					fmt.Printf("\n%s:\n", operation)
					fmt.Printf("  调用次数: %v\n", m["call_count"])
					fmt.Printf("  平均 CPU 使用率: %.2f%%\n", m["avg_cpu_percent"])
					fmt.Printf("  平均内存使用: %.2f MB\n", m["avg_memory_mb"])
					fmt.Printf("  最大 CPU 使用率: %.2f%%\n", m["max_cpu_percent"])
					fmt.Printf("  最大内存使用: %.2f MB\n", m["max_memory_mb"])

					if gpuPercent, ok := m["avg_gpu_percent"].(float64); ok && gpuPercent > 0 {
						fmt.Printf("  平均 GPU 使用率: %.2f%%\n", gpuPercent)
						fmt.Printf("  平均 GPU 内存: %.2f MB\n", m["avg_gpu_memory_mb"])
						fmt.Printf("  最大 GPU 使用率: %.2f%%\n", m["max_gpu_percent"])
						fmt.Printf("  最大 GPU 内存: %.2f MB\n", m["max_gpu_memory_mb"])
					}
				}
			}
		}
	}

	// 8. Token 消耗详情
	fmt.Println("\n\n=== Token 消耗详情 ===")
	if tokenUsage, ok := allMetrics["token_usage"].(map[string]any); ok {
		if len(tokenUsage) > 0 {
			for operation, metrics := range tokenUsage {
				if m, ok := metrics.(map[string]any); ok {
					fmt.Printf("\n%s:\n", operation)
					fmt.Printf("  调用次数: %v\n", m["call_count"])
					fmt.Printf("  总 Token 数: %v\n", m["total_tokens"])
					fmt.Printf("  平均 Token 数: %v\n", m["avg_total_tokens"])
				}
			}
		}
	}

	// 9. 资源监控器演示
	fmt.Println("\n\n=== 资源监控器独立使用演示 ===")
	fmt.Println("ResourceMonitor 可以独立使用来监控任意代码块的资源消耗：\n")

	monitor := metrics.NewResourceMonitor()
	before := monitor.Snapshot()

	fmt.Println("执行一些计算...")
	// 模拟一些计算
	sum := 0
	for i := 0; i < 1000000; i++ {
		sum += i
	}

	after := monitor.Snapshot()
	delta := after.Delta(before)

	fmt.Printf("\n资源使用快照:\n")
	fmt.Printf("  执行时间: %v\n", delta.Duration)
	fmt.Printf("  内存分配: %.2f MB\n", after.MemoryAllocMB)
	fmt.Printf("  内存变化: %.2f MB\n", delta.MemoryAllocMB)
	fmt.Printf("  堆内存: %.2f MB\n", after.HeapAllocMB)
	fmt.Printf("  Goroutine 数: %d\n", after.NumGoroutines)
	fmt.Printf("  GC 次数: %d\n", delta.GCCount)
	fmt.Printf("  最近 GC 暂停: %.2f ms\n", after.GCPauseMs)

	fmt.Println("\n✅ 系统资源监控演示完成！")
	fmt.Println("\n总结：")
	fmt.Println("1. 系统资源指标对 AI 应用至关重要")
	fmt.Println("2. 监控 CPU、内存、GPU 使用情况")
	fmt.Println("3. 用于性能优化、成本控制、容量规划")
	fmt.Println("4. 支持实时监控和历史统计")
	fmt.Println("5. 可以独立使用 ResourceMonitor 监控任意代码块")
	fmt.Println("\n应用场景：")
	fmt.Println("  - 本地 LLM 部署：监控 GPU 内存，避免 OOM")
	fmt.Println("  - 云服务成本：根据资源使用优化实例类型")
	fmt.Println("  - 并发处理：评估单机能处理多少并发请求")
	fmt.Println("  - 模型选择：比较不同模型的资源效率")
}
