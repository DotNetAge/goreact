package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/skill"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct Skill Execution Example ===\n")

	// 1. 创建 Skill Manager
	skillManager := skill.NewDefaultManager()

	// 2. 创建一个示例 Skill
	calculationSkill := skill.NewSkill(
		"advanced-calculation",
		"Advanced mathematical calculation with step-by-step reasoning",
	)
	calculationSkill.Instructions = `When performing calculations:
1. Break down complex expressions into simple steps
2. Use the calculator tool for each step
3. Verify intermediate results
4. Provide clear explanations for each step
5. Double-check the final answer

Example:
- For "15 * 23 + 7", first calculate 15 * 23, then add 7
- Always show your work`

	// 注册 Skill
	if err := skillManager.RegisterSkill(calculationSkill); err != nil {
		fmt.Printf("Failed to register skill: %v\n", err)
		return
	}

	// 3. 创建 Mock LLM Client（模拟响应）
	mockResponses := []string{
		// 第一次思考：分解任务
		`Thought: I need to calculate 15 * 23 + 7. Following the skill instructions, I'll break this down into steps.
Action: calculator
Action Input: {"operation": "multiply", "a": 15, "b": 23}`,
		// 第二次思考：继续计算
		`Thought: I got 345 from 15 * 23. Now I need to add 7.
Action: calculator
Action Input: {"operation": "add", "a": 345, "b": 7}`,
		// 第三次思考：完成
		`Thought: I have completed the calculation step by step as instructed.
Final Answer: The result of 15 * 23 + 7 is 352.
Step 1: 15 * 23 = 345
Step 2: 345 + 7 = 352`,
	}
	llmClient := mock.NewMockClient(mockResponses)

	// 4. 创建 Engine 并注入 SkillManager
	eng := engine.Reactor(
		engine.WithLLMClient(llmClient),
		engine.WithSkillManager(skillManager),
		engine.WithMaxIterations(10),
	)

	// 5. 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	// 6. 执行任务
	fmt.Println("Task: Calculate 15 * 23 + 7")
	fmt.Println("---")

	startTime := time.Now()
	result := eng.Execute(context.Background(), "Calculate 15 * 23 + 7", nil)
	executionTime := time.Since(startTime)

	// 7. 显示结果
	fmt.Printf("\n=== Execution Result ===\n")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Output: %s\n", result.Output)
	fmt.Printf("Execution Time: %v\n", executionTime)
	fmt.Printf("Iterations: %d\n", len(result.Trace))

	// 8. 显示执行轨迹
	fmt.Printf("\n=== Execution Trace ===\n")
	for _, step := range result.Trace {
		fmt.Printf("[Step %d - %s] %s\n", step.Step, step.Type, step.Content)
	}

	// 9. 显示 Skill 统计
	fmt.Printf("\n=== Skill Statistics ===\n")
	stats, err := skillManager.GetSkillStatistics("advanced-calculation")
	if err != nil {
		fmt.Printf("Failed to get statistics: %v\n", err)
	} else {
		fmt.Printf("Skill: %s\n", calculationSkill.Name)
		fmt.Printf("Usage Count: %d\n", stats.UsageCount)
		fmt.Printf("Success Count: %d\n", stats.SuccessCount)
		fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate*100)
		fmt.Printf("Average Execution Time: %v\n", stats.AverageExecutionTime)
		fmt.Printf("Overall Score: %.2f\n", stats.OverallScore)
	}

	// 10. 显示 Skill 排名
	fmt.Printf("\n=== Skill Ranking ===\n")
	rankings := skillManager.GetSkillRanking()
	for _, ranking := range rankings {
		fmt.Printf("Rank %d: %s (Score: %.2f)\n", ranking.Rank, ranking.SkillName, ranking.OverallScore)
	}
}
