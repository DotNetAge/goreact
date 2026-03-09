package main

import (
	"fmt"
	"context"
	"time"

	"github.com/ray/goreact/pkg/cache"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/llm/ollama"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct with Cache Example ===")

	// 创建 Ollama 客户端
	ollamaClient := ollama.NewOllamaClient(
		ollama.WithModel("qwen3:0.6b"),
		ollama.WithTemperature(0.7),
	)

	// 创建缓存
	memCache := cache.NewMemoryCache(
		cache.WithMaxSize(100),
		cache.WithDefaultTTL(5*time.Minute),
	)

	// 创建引擎
	eng := engine.New(
		engine.WithLLMClient(ollamaClient),
		engine.WithCache(memCache),
		engine.WithMaxIterations(10),
	)

	// 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	task := "Calculate 10 + 5"

	// 第一次执行（无缓存）
	fmt.Println("First execution (no cache):")
	fmt.Println("Task:", task)
	start1 := time.Now()
	result1 := eng.Execute(context.Background(), task, nil)
	duration1 := time.Since(start1)

	fmt.Printf("Success: %v\n", result1.Success)
	fmt.Printf("Output: %s\n", result1.Output)
	fmt.Printf("Cached: %v\n", result1.Metadata["cached"])
	fmt.Printf("Duration: %v\n\n", duration1)

	// 第二次执行（应该命中缓存）
	fmt.Println("Second execution (should hit cache):")
	fmt.Println("Task:", task)
	start2 := time.Now()
	result2 := eng.Execute(context.Background(), task, nil)
	duration2 := time.Since(start2)

	fmt.Printf("Success: %v\n", result2.Success)
	fmt.Printf("Output: %s\n", result2.Output)
	fmt.Printf("Cached: %v\n", result2.Metadata["cached"])
	fmt.Printf("Duration: %v\n\n", duration2)

	// 显示性能提升
	if result2.Metadata["cached"] == true {
		speedup := float64(duration1) / float64(duration2)
		fmt.Printf("Cache speedup: %.2fx faster\n", speedup)
	}

	// 显示缓存统计
	fmt.Printf("\nCache size: %d items\n", memCache.Size())
}
