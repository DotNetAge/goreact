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
	fmt.Println("=== GoReAct: Load Skill from SKILL.md File ===\n")

	// 1. 创建 Skill Manager
	skillManager := skill.NewDefaultManager()

	// 2. 从文件加载 Skill
	fmt.Println("Loading skill from file...")
	skillPath := "./skills/math-wizard"
	loadedSkill, err := skillManager.LoadSkill(skillPath)
	if err != nil {
		fmt.Printf("❌ Failed to load skill: %v\n", err)
		return
	}

	fmt.Printf("✅ Successfully loaded skill: %s\n", loadedSkill.Name)
	fmt.Printf("   Description: %s\n", loadedSkill.Description)
	fmt.Printf("   License: %s\n", loadedSkill.License)
	fmt.Printf("   Compatibility: %s\n", loadedSkill.Compatibility)
	fmt.Printf("   Allowed Tools: %v\n", loadedSkill.AllowedTools)
	fmt.Printf("   Metadata: %v\n", loadedSkill.Metadata)
	fmt.Printf("   Scripts: %d files\n", len(loadedSkill.Scripts))
	fmt.Printf("   References: %d files\n", len(loadedSkill.References))
	fmt.Printf("   Instructions length: %d characters\n\n", len(loadedSkill.Instructions))

	// 3. 显示加载的内容
	fmt.Println("=== Loaded Content ===")
	if len(loadedSkill.Scripts) > 0 {
		fmt.Println("\n📜 Scripts:")
		for name := range loadedSkill.Scripts {
			fmt.Printf("   - %s\n", name)
		}
	}

	if len(loadedSkill.References) > 0 {
		fmt.Println("\n📚 References:")
		for name := range loadedSkill.References {
			fmt.Printf("   - %s\n", name)
		}
	}

	fmt.Println("\n📝 Instructions Preview:")
	preview := loadedSkill.Instructions
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	fmt.Printf("   %s\n\n", preview)

	// 4. 注册 Skill
	if err := skillManager.RegisterSkill(loadedSkill); err != nil {
		fmt.Printf("❌ Failed to register skill: %v\n", err)
		return
	}
	fmt.Println("✅ Skill registered successfully\n")

	// 5. 创建 Mock LLM Client
	mockResponses := []string{
		`Thought: I need to calculate 25 * 4 + 10. Following the Math Wizard skill, I'll break this down step by step.
Action: calculator
Action Input: {"operation": "multiply", "a": 25, "b": 4}`,
		`Thought: I got 100 from 25 * 4. Now I need to add 10 to complete the calculation.
Action: calculator
Action Input: {"operation": "add", "a": 100, "b": 10}`,
		`Thought: Perfect! I've completed the calculation following the Math Wizard methodology.
Final Answer: The result of 25 * 4 + 10 is 110.

Step-by-step breakdown:
1. First, multiply: 25 * 4 = 100
2. Then, add: 100 + 10 = 110
3. Final Answer: 110

This follows the order of operations (PEMDAS), where multiplication is performed before addition.`,
	}
	llmClient := mock.NewMockClient(mockResponses)

	// 6. 创建 Engine
	eng := engine.Reactor(
		engine.WithLLMClient(llmClient),
		engine.WithSkillManager(skillManager),
		engine.WithMaxIterations(10),
	)

	// 7. 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	// 8. 执行任务
	fmt.Println("=== Executing Task with Loaded Skill ===")
	fmt.Println("Task: Calculate 25 * 4 + 10")
	fmt.Println("---\n")

	startTime := time.Now()
	result := eng.Execute(context.Background(), "Calculate 25 * 4 + 10", nil)
	executionTime := time.Since(startTime)

	// 9. 显示结果
	fmt.Printf("\n=== Execution Result ===\n")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Output: %s\n", result.Output)
	fmt.Printf("Execution Time: %v\n", executionTime)
	fmt.Printf("Total Steps: %d\n", len(result.Trace))

	// 10. 显示关键执行步骤
	fmt.Printf("\n=== Key Execution Steps ===\n")
	for _, step := range result.Trace {
		if step.Type == "think" || step.Type == "finish" {
			fmt.Printf("[%s] %s\n", step.Type, step.Content)
		}
	}

	// 11. 显示 Skill 统计
	fmt.Printf("\n=== Skill Statistics ===\n")

	// 先检查是否有选中的 skill
	if selectedSkillName, ok := result.Metadata["selected_skill"].(string); ok {
		fmt.Printf("Selected Skill: %s\n", selectedSkillName)
	} else {
		fmt.Println("⚠️  No skill was selected for this task")
	}

	stats, err := skillManager.GetSkillStatistics("math-wizard")
	if err != nil {
		fmt.Printf("Failed to get statistics: %v\n", err)
	} else {
		fmt.Printf("Skill: %s\n", loadedSkill.Name)
		fmt.Printf("Usage Count: %d\n", stats.UsageCount)
		fmt.Printf("Success Count: %d\n", stats.SuccessCount)
		fmt.Printf("Failure Count: %d\n", stats.FailureCount)
		if stats.UsageCount > 0 {
			fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate*100)
			fmt.Printf("Average Execution Time: %v\n", stats.AverageExecutionTime)
			fmt.Printf("Overall Score: %.2f\n", stats.OverallScore)
		} else {
			fmt.Println("⚠️  No usage statistics recorded yet")
		}
	}

	// 12. 测试 Skill 列表功能
	fmt.Printf("\n=== Available Skills ===\n")
	skillList := skillManager.ListSkills()
	for i, skillMeta := range skillList {
		fmt.Printf("%d. %s - %s\n", i+1, skillMeta.Name, skillMeta.Description)
	}
}
