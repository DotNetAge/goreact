package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/metrics"
	"github.com/ray/goreact/pkg/tool/builtin"
)

// MockLLMWithTokens 模拟带 Token 统计的 LLM Client
type MockLLMWithTokens struct {
	responses      []string
	callCount      int
	lastTokenUsage *gochatcore.Usage
}

func NewMockLLMWithTokens(responses []string) *MockLLMWithTokens {
	return &MockLLMWithTokens{
		responses: responses,
		callCount: 0,
	}
}

// Chat 实现 gochatcore.Client 接口
func (m *MockLLMWithTokens) Chat(ctx context.Context, messages []gochatcore.Message, opts ...gochatcore.Option) (*gochatcore.Response, error) {
	if m.callCount < len(m.responses) {
		response := m.responses[m.callCount]
		m.callCount++

		// 模拟 Token 消耗（根据 prompt 和 response 长度估算）
		promptTokens := len(messages[0].TextContent()) / 4 // 粗略估算：4 字符 ≈ 1 token
		completionTokens := len(response) / 4
		m.lastTokenUsage = &gochatcore.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}

		return &gochatcore.Response{
			Content: response,
			Usage:   m.lastTokenUsage,
		}, nil
	}

	return &gochatcore.Response{
		Content: "Final Answer: Task completed.",
	}, nil
}

// ChatStream 实现 gochatcore.Client 接口
func (m *MockLLMWithTokens) ChatStream(ctx context.Context, messages []gochatcore.Message, opts ...gochatcore.Option) (*gochatcore.Stream, error) {
	return nil, fmt.Errorf("ChatStream not implemented in mock")
}

// LastTokenUsage 返回最近一次的 Token 使用情况
func (m *MockLLMWithTokens) LastTokenUsage() *gochatcore.Usage {
	return m.lastTokenUsage
}

func main() {
	fmt.Println("=== GoReAct: Token Metrics 演示 ===\n")

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

	// 3. 创建支持 Token 统计的 Mock LLM
	fmt.Println("Step 3: 创建 LLM Client（支持 Token 统计）")
	mockResponses := []string{
		`Thought: I need to calculate 100 + 200.
Action: calculator
Parameters: {"operation": "add", "a": 100, "b": 200}`,
		`Thought: The result is 300.
Final Answer: The sum of 100 and 200 is 300.`,
	}
	mockLLM := NewMockLLMWithTokens(mockResponses)
	fmt.Println("✓ Mock LLM 创建完成（实现了 TokenReporter 接口）\n")

	// 4. 创建 Engine
	fmt.Println("Step 4: 创建 Engine")
	eng := engine.Reactor(
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
	task := "Calculate 100 + 200"
	fmt.Printf("Task: %s\n\n", task)

	startTime := time.Now()
	result := eng.Execute(context.Background(), task, nil)
	executionTime := time.Since(startTime)

	fmt.Printf("\n执行结果:\n")
	fmt.Printf("  Success: %v\n", result.Success)
	fmt.Printf("  Output: %s\n", result.Output)
	fmt.Printf("  Execution Time: %v\n", executionTime)

	// 6. 显示 Metrics（包括 Token 消耗）
	fmt.Println("\n\n=== Metrics 统计（包含 Token 消耗）===")
	allMetrics := metricsCollector.GetMetrics()

	metricsJSON, _ := json.MarshalIndent(allMetrics, "", "  ")
	fmt.Println(string(metricsJSON))

	// 7. 解析并显示 Token 指标
	fmt.Println("\n=== Token 消耗详情 ===")
	if tokenUsage, ok := allMetrics["token_usage"].(map[string]any); ok {
		if len(tokenUsage) == 0 {
			fmt.Println("  无 Token 消耗记录")
		} else {
			for operation, metrics := range tokenUsage {
				if m, ok := metrics.(map[string]any); ok {
					fmt.Printf("\n%s:\n", operation)
					fmt.Printf("  调用次数: %v\n", m["call_count"])
					fmt.Printf("  总输入 Token: %v\n", m["prompt_tokens"])
					fmt.Printf("  总输出 Token: %v\n", m["completion_tokens"])
					fmt.Printf("  总 Token 数: %v\n", m["total_tokens"])
					fmt.Printf("  平均输入 Token: %v\n", m["avg_prompt_tokens"])
					fmt.Printf("  平均输出 Token: %v\n", m["avg_completion_tokens"])
					fmt.Printf("  平均总 Token: %v\n", m["avg_total_tokens"])

					// 估算成本（以 GPT-4 为例：$0.03/1K prompt tokens, $0.06/1K completion tokens）
					if totalTokens, ok := m["total_tokens"].(int64); ok {
						promptTokens, _ := m["prompt_tokens"].(int64)
						completionTokens, _ := m["completion_tokens"].(int64)

						estimatedCost := (float64(promptTokens)/1000)*0.03 + (float64(completionTokens)/1000)*0.06
						fmt.Printf("  估算成本 (GPT-4): $%.6f\n", estimatedCost)
						fmt.Printf("  (实际成本取决于使用的模型)\n")

						_ = totalTokens // 避免未使用变量警告
					}
				}
			}
		}
	}

	// 8. 演示 Ollama Client 的 Token 统计
	fmt.Println("\n\n=== Ollama Client Token 统计支持 ===")
	fmt.Println("Ollama Client 已实现 TokenReporter 接口：")
	fmt.Println("  - 自动从 API 响应中提取 prompt_eval_count 和 eval_count")
	fmt.Println("  - 通过 LastTokenUsage() 方法获取最近一次调用的 Token 使用量")
	fmt.Println("  - Engine 自动检测并记录到 Metrics")

	// 创建 Ollama Client 示例（不实际调用）
	fmt.Println("\nOllama Client 已实现 TokenReporter 接口：")
	fmt.Println("  - 自动从 API 响应中提取 prompt_eval_count 和 eval_count")
	fmt.Println("  - 通过 Response.Usage 获取 Token 使用量")
	fmt.Println("  - Engine 自动检测并记录到 Metrics")

	fmt.Println("\n✅ Token Metrics 演示完成！")
	fmt.Println("\n总结：")
	fmt.Println("1. Token 消耗是 LLM 应用的核心成本指标")
	fmt.Println("2. 通过 TokenReporter 接口，LLM Client 可以报告 Token 使用量")
	fmt.Println("3. Engine 自动检测并记录到 Metrics 系统")
	fmt.Println("4. 支持统计：总量、平均值、调用次数")
	fmt.Println("5. 可用于成本估算、性能优化、预算管理")
}
