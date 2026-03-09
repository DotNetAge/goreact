package main

import (
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/prompt/builder"
	"github.com/ray/goreact/pkg/prompt/compression"
	"github.com/ray/goreact/pkg/prompt/counter"
	"github.com/ray/goreact/pkg/prompt/debug"
	"github.com/ray/goreact/pkg/prompt/formatter"
)

func main() {
	fmt.Println("=== Prompt Toolkit 示例 ===\n")

	// 1. 创建工具描述
	tools := []formatter.ToolDesc{
		{
			Name:        "calculator",
			Description: "Perform arithmetic operations",
			Parameters: &formatter.ParameterSchema{
				Type: "object",
				Properties: map[string]*formatter.Property{
					"operation": {
						Type:        "string",
						Description: "The operation to perform",
						Enum:        []interface{}{"add", "subtract", "multiply", "divide"},
					},
					"a": {
						Type:        "number",
						Description: "First operand",
					},
					"b": {
						Type:        "number",
						Description: "Second operand",
					},
				},
				Required: []string{"operation", "a", "b"},
			},
		},
		{
			Name:        "http",
			Description: "Make HTTP requests",
			Parameters: &formatter.ParameterSchema{
				Type: "object",
				Properties: map[string]*formatter.Property{
					"method": {
						Type:        "string",
						Description: "HTTP method",
						Enum:        []interface{}{"GET", "POST", "PUT", "DELETE"},
					},
					"url": {
						Type:        "string",
						Description: "Target URL",
					},
				},
				Required: []string{"method", "url"},
			},
		},
	}

	// 2. 演示不同的工具格式化器
	fmt.Println("--- 工具格式化器对比 ---\n")

	fmt.Println("1. Simple Text Format:")
	simpleFormatter := formatter.NewSimpleTextFormatter()
	fmt.Println(simpleFormatter.Format(tools))

	fmt.Println("\n2. JSON Schema Format:")
	jsonFormatter := formatter.NewJSONSchemaFormatter(true)
	fmt.Println(jsonFormatter.Format(tools))

	fmt.Println("\n3. Markdown Format:")
	mdFormatter := formatter.NewMarkdownFormatter()
	fmt.Println(mdFormatter.Format(tools))

	fmt.Println("\n4. Compact Format (节省 tokens):")
	compactFormatter := formatter.NewCompactFormatter()
	fmt.Println(compactFormatter.Format(tools))

	// 3. 演示 Token 计数器
	fmt.Println("\n--- Token 计数器对比 ---\n")

	testText := "Calculate 100 + 200 using the calculator tool. 计算器工具可以执行加减乘除运算。"

	simpleCounter := counter.NewSimpleEstimator()
	fmt.Printf("Simple Estimator: %d tokens\n", simpleCounter.Count(testText))

	universalCounter := counter.NewUniversalEstimator("mixed")
	fmt.Printf("Universal Estimator (mixed): %d tokens\n", universalCounter.Count(testText))

	enCounter := counter.NewUniversalEstimator("en")
	fmt.Printf("Universal Estimator (en): %d tokens\n", enCounter.Count(testText))

	zhCounter := counter.NewUniversalEstimator("zh")
	fmt.Printf("Universal Estimator (zh): %d tokens\n", zhCounter.Count(testText))

	// 4. 演示 FluentPromptBuilder
	fmt.Println("\n--- FluentPromptBuilder 示例 ---\n")

	// 创建历史记录
	history := []builder.Turn{
		{Role: "user", Content: "Hello, I need help with calculations"},
		{Role: "assistant", Content: "I can help you with that using the calculator tool"},
		{Role: "user", Content: "Great!"},
	}

	// 创建 Few-Shot 示例
	fewShots := []builder.FewShotExample{
		{
			Task:    "Calculate 10 + 5",
			Thought: "I need to use the calculator tool for addition",
			Action:  "calculator",
			Parameters: map[string]interface{}{
				"operation": "add",
				"a":         10,
				"b":         5,
			},
			Result: "15",
		},
	}

	// 创建调试器
	logger := debug.NewSimpleLogger(true)
	debugger := debug.NewPromptDebugger(true, logger)

	// 使用 FluentPromptBuilder
	start := time.Now()
	promptBuilder := builder.New().
		WithTask("Calculate 100 + 200").
		WithTools(tools).
		WithHistory(history).
		WithFewShots(fewShots).
		WithToolFormatter(jsonFormatter).
		WithHistoryFormatter(builder.NewMarkdownHistoryFormatter()).
		WithTokenCounter(universalCounter).
		WithMaxTokens(4096)

	p := promptBuilder.Build()
	buildTime := time.Since(start)

	// 记录调试信息
	debugger.LogPrompt(p, map[string]interface{}{
		"tools_count":     len(tools),
		"history_turns":   len(history),
		"few_shots_count": len(fewShots),
		"token_counter":   universalCounter,
	})
	debugger.LogBuildTime(buildTime)

	fmt.Println("\n生成的 Prompt:")
	fmt.Println("--- System ---")
	fmt.Println(p.System[:200] + "...")
	fmt.Println("\n--- User ---")
	fmt.Println(p.User[:200] + "...")

	// 5. 演示上下文压缩
	fmt.Println("\n--- 上下文压缩示例 ---\n")

	// 创建长历史记录
	longHistory := []compression.Turn{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "Can you help me?"},
		{Role: "assistant", Content: "Of course!"},
		{Role: "user", Content: "I need to calculate something"},
		{Role: "assistant", Content: "I can help with calculations"},
		{Role: "user", Content: "Calculate 10 + 5"},
		{Role: "assistant", Content: "The result is 15"},
		{Role: "user", Content: "Now calculate 20 * 3"},
		{Role: "assistant", Content: "The result is 60"},
	}

	// 计算原始 token 数
	originalTokens := 0
	for _, turn := range longHistory {
		originalTokens += universalCounter.Count(turn.Content)
	}
	fmt.Printf("原始历史: %d 轮, %d tokens\n", len(longHistory), originalTokens)

	// 1. 截断策略
	truncateStrategy := compression.NewTruncateStrategy()
	truncated, _ := truncateStrategy.Compress(longHistory, 100, universalCounter)
	truncatedTokens := 0
	for _, turn := range truncated {
		truncatedTokens += universalCounter.Count(turn.Content)
	}
	fmt.Printf("截断策略: %d 轮, %d tokens (移除了 %.1f%%)\n",
		len(truncated), truncatedTokens,
		float64(len(longHistory)-len(truncated))/float64(len(longHistory))*100)

	// 2. 滑动窗口策略
	windowStrategy := compression.NewSlidingWindowStrategy(5)
	windowed, _ := windowStrategy.Compress(longHistory, 100, universalCounter)
	windowedTokens := 0
	for _, turn := range windowed {
		windowedTokens += universalCounter.Count(turn.Content)
	}
	fmt.Printf("滑动窗口策略: %d 轮, %d tokens (保留最近 5 轮)\n",
		len(windowed), windowedTokens)

	// 3. 优先级策略
	priorityStrategy := compression.NewPriorityStrategy(map[string]int{
		"system":    100,
		"user":      80,
		"assistant": 60,
	})
	prioritized, _ := priorityStrategy.Compress(longHistory, 150, universalCounter)
	prioritizedTokens := 0
	for _, turn := range prioritized {
		prioritizedTokens += universalCounter.Count(turn.Content)
	}
	fmt.Printf("优先级策略: %d 轮, %d tokens (保留重要消息)\n",
		len(prioritized), prioritizedTokens)

	fmt.Println("\n优先级策略保留的消息:")
	for i, turn := range prioritized {
		fmt.Printf("  %d. [%s]: %s\n", i+1, turn.Role, turn.Content)
	}

	// 6. 演示混合策略
	fmt.Println("\n--- 混合策略示例 ---\n")

	hybridStrategy := compression.NewHybridStrategy(
		priorityStrategy,
		windowStrategy,
	)
	hybrid, _ := hybridStrategy.Compress(longHistory, 100, universalCounter)
	hybridTokens := 0
	for _, turn := range hybrid {
		hybridTokens += universalCounter.Count(turn.Content)
	}
	fmt.Printf("混合策略 (优先级 + 滑动窗口): %d 轮, %d tokens\n",
		len(hybrid), hybridTokens)

	// 7. Token 使用报告
	fmt.Println("\n--- Token 使用报告 ---\n")
	tracker := debugger.GetTracker()
	fmt.Println(tracker.Report())

	fmt.Println("\n=== 示例完成 ===")
}
