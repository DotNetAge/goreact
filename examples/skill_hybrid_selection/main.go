package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/skill"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct: Hybrid Skill Selection Demo ===\n")

	// 1. 创建多个不同类型的技能
	skills := []*skill.Skill{
		{
			Name:        "math-wizard",
			Description: "Expert mathematical problem solver with step-by-step reasoning",
			Instructions: `Solve math problems step by step:
1. Break down the expression
2. Use calculator tool for each operation
3. Verify results`,
			Statistics: &skill.SkillStatistics{
				CreatedAt: time.Now(),
			},
		},
		{
			Name:        "data-analyzer",
			Description: "Analyze and process data sets, generate statistics and insights",
			Instructions: `Analyze data:
1. Load data
2. Calculate statistics
3. Generate insights`,
			Statistics: &skill.SkillStatistics{
				CreatedAt: time.Now(),
			},
		},
		{
			Name:        "text-processor",
			Description: "Process and transform text content, including formatting and extraction",
			Instructions: `Process text:
1. Parse input
2. Apply transformations
3. Format output`,
			Statistics: &skill.SkillStatistics{
				CreatedAt: time.Now(),
			},
		},
	}

	// 2. 创建 LLM Client（用于语义匹配）
	semanticResponses := []string{
		// LLM 用于技能选择的响应
		"math-wizard",
		// 任务执行的响应
		`Thought: I need to calculate 18 * 5 + 12. Following the Math Wizard skill, I'll break this down.
Action: calculator
Action Input: {"operation": "multiply", "a": 18, "b": 5}`,
		`Thought: I got 90 from 18 * 5. Now I need to add 12.
Action: calculator
Action Input: {"operation": "add", "a": 90, "b": 12}`,
		`Thought: Perfect! I've completed the calculation.
Final Answer: The result of 18 * 5 + 12 is 102.
Step 1: 18 * 5 = 90
Step 2: 90 + 12 = 102`,
	}
	llmClient := mock.NewMockClient(semanticResponses)

	// 3. 测试三种选择模式
	task := "Calculate 18 * 5 + 12"

	fmt.Println("Task:", task)
	fmt.Println(strings.Repeat("=", 60))

	// 模式 1: 仅关键词匹配
	fmt.Println("\n【Mode 1: Keyword Only】")
	testSelectionMode(task, skills, llmClient, skill.KeywordOnly)

	// 模式 2: 仅语义匹配
	fmt.Println("\n【Mode 2: Semantic Only】")
	testSelectionMode(task, skills, llmClient, skill.SemanticOnly)

	// 模式 3: 混合模式（推荐）
	fmt.Println("\n【Mode 3: Hybrid (Recommended)】")
	testSelectionMode(task, skills, llmClient, skill.Hybrid)

	// 4. 完整执行演示（使用混合模式）
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("【Full Execution with Hybrid Mode】\n")

	// 重新创建 LLM Client（因为 mock 响应已用完）
	executionResponses := []string{
		"math-wizard", // 技能选择
		`Thought: I need to calculate 18 * 5 + 12. Following the Math Wizard skill, I'll break this down.
Action: calculator
Action Input: {"operation": "multiply", "a": 18, "b": 5}`,
		`Thought: I got 90 from 18 * 5. Now I need to add 12.
Action: calculator
Action Input: {"operation": "add", "a": 90, "b": 12}`,
		`Thought: Perfect! I've completed the calculation.
Final Answer: The result of 18 * 5 + 12 is 102.`,
	}
	llmClient2 := mock.NewMockClient(executionResponses)

	// 创建 SkillManager（混合模式）
	skillManager := skill.NewDefaultManager(
		skill.WithLLMClient(llmClient2),
		skill.WithSelectionMode(skill.Hybrid),
		skill.WithTopN(3),
	)

	// 注册所有技能
	for _, s := range skills {
		skillManager.RegisterSkill(s)
	}

	// 创建 Engine
	eng := engine.Reactor(
		engine.WithLLMClient(llmClient2),
		engine.WithSkillManager(skillManager),
		engine.WithMaxIterations(10),
	)

	// 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	// 执行任务
	startTime := time.Now()
	result := eng.Execute(context.Background(), task, nil)
	executionTime := time.Since(startTime)

	// 显示结果
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Output: %s\n", result.Output)
	fmt.Printf("Execution Time: %v\n", executionTime)

	if selectedSkill, ok := result.Metadata["selected_skill"].(string); ok {
		fmt.Printf("Selected Skill: %s ✓\n", selectedSkill)

		// 显示技能统计
		if stats, err := skillManager.GetSkillStatistics(selectedSkill); err == nil {
			fmt.Printf("\nSkill Statistics:\n")
			fmt.Printf("  Usage Count: %d\n", stats.UsageCount)
			fmt.Printf("  Success Rate: %.2f%%\n", stats.SuccessRate*100)
			fmt.Printf("  Avg Execution Time: %v\n", stats.AverageExecutionTime)
		}
	}
}

func testSelectionMode(task string, skills []*skill.Skill, llmClient *mock.MockClient, mode skill.SelectionMode) {
	// 创建 SkillManager
	var manager *skill.DefaultManager
	if mode == skill.SemanticOnly || mode == skill.Hybrid {
		manager = skill.NewDefaultManager(
			skill.WithLLMClient(llmClient),
			skill.WithSelectionMode(mode),
			skill.WithTopN(3),
		)
	} else {
		manager = skill.NewDefaultManager(
			skill.WithSelectionMode(mode),
		)
	}

	// 注册技能
	for _, s := range skills {
		manager.RegisterSkill(s)
	}

	// 测试选择
	startTime := time.Now()
	selected, err := manager.SelectSkill(task)
	selectionTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Selection failed: %v\n", err)
		return
	}

	fmt.Printf("✓ Selected: %s\n", selected.Name)
	fmt.Printf("  Description: %s\n", selected.Description)
	fmt.Printf("  Selection Time: %v\n", selectionTime)

	// 显示选择过程（仅混合模式）
	if mode == skill.Hybrid {
		fmt.Printf("  Process: Keyword filtering → Semantic selection\n")
	}
}
