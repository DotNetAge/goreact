package main

import (
	"fmt"
	"log"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/core/thinker/middlewares"
	"github.com/ray/goreact/pkg/core/thinker/presets"
	"github.com/ray/goreact/pkg/mock"
)

func main() {
	// 创建Mock LLM客户端
	mockResponses := []string{
		"Thought: I need to calculate 123456789 * 987654321.\nAction: calculator\nParameters: {\"expression\": \"123456789 * 987654321\"}\nReasoning: Using calculator tool to perform the multiplication",
		"Thought: The calculation result is 121932631112635269.\nFinal Answer: 121932631112635269",
		"反思: 我应该使用计算器工具来执行这个复杂的乘法运算，这样可以确保结果的准确性。",
	}
	mockLLM := mock.NewMockClient(mockResponses)

	// 创建ReAct思考器
	reactThinker := presets.NewReActThinker(mockLLM, "calculator: 执行数学计算")

	// 添加Reflexion中间件
	rethinker := thinker.NewMiddlewareThinker(reactThinker)
	rethinker.Use(
		middlewares.ReflexionMiddleware(mockLLM, 2), // 最多反思2次
		middlewares.RetryMiddleware(3, 1),             // 最多重试3次
	)

	// 创建上下文
	ctx := core.NewContext()

	// 定义任务
	task := "计算 123456789 * 987654321"

	// 执行思考
	fmt.Printf("执行任务: %s\n", task)
	thought, err := rethinker.Think(task, ctx)
	if err != nil {
		log.Fatalf("思考失败: %v", err)
	}

	// 打印结果
	fmt.Printf("思考结果: %+v\n", thought)
	if thought.Metadata != nil {
		if reflectionCount, ok := thought.Metadata["reflection_count"]; ok {
			fmt.Printf("反思次数: %v\n", reflectionCount)
		}
		if retryAttempts, ok := thought.Metadata["retry_attempts"]; ok {
			fmt.Printf("重试次数: %v\n", retryAttempts)
		}
	}
}
