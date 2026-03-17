package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/agent"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/log"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/model"
	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== GoReAct: Agent + Model + LLM 完整集成示例 ===\n")

	// 1. 配置 Models（纯配置，不持有运行时对象）
	fmt.Println("Step 1: 配置 Models")
	modelManager := model.NewManager()

	// 配置 Ollama 模型（本地）
	ollamaModel, err := model.NewModel("qwen3-local", "ollama", "qwen3:8b")
	if err != nil {
		fmt.Printf("Failed to create Ollama model: %v\n", err)
		return
	}
	ollamaModel.WithBaseURL("http://localhost:11434")
	if _, err := ollamaModel.WithTemperature(0.7); err != nil {
		fmt.Printf("Failed to set temperature: %v\n", err)
		return
	}
	if _, err := ollamaModel.WithTimeout(30); err != nil {
		fmt.Printf("Failed to set timeout: %v\n", err)
		return
	}

	// 配置 OpenAI 模型
	gpt4Model, err := model.NewModel("gpt4-turbo", "openai", "gpt-4-turbo")
	if err != nil {
		fmt.Printf("Failed to create OpenAI model: %v\n", err)
		return
	}
	gpt4Model.WithAPIKey("sk-xxx") // 实际使用时需要真实 API Key
	if _, err := gpt4Model.WithTemperature(0.8); err != nil {
		fmt.Printf("Failed to set temperature: %v\n", err)
		return
	}
	if _, err := gpt4Model.WithMaxTokens(4096); err != nil {
		fmt.Printf("Failed to set max tokens: %v\n", err)
		return
	}

	// 配置 Claude 模型
	claudeModel, err := model.NewModel("claude-opus", "anthropic", "claude-3-opus-20240229")
	if err != nil {
		fmt.Printf("Failed to create Anthropic model: %v\n", err)
		return
	}
	claudeModel.WithAPIKey("sk-ant-xxx") // 实际使用时需要真实 API Key
	if _, err := claudeModel.WithTemperature(0.7); err != nil {
		fmt.Printf("Failed to set temperature: %v\n", err)
		return
	}

	modelManager.RegisterModel(ollamaModel)
	modelManager.RegisterModel(gpt4Model)
	modelManager.RegisterModel(claudeModel)

	fmt.Printf("✓ 注册了 %d 个模型配置\n", len(modelManager.ListModels()))
	for _, m := range modelManager.ListModels() {
		fmt.Printf("  - %s (%s: %s)\n", m.Name, m.Provider, m.ModelID)
	}
	fmt.Println()

	// 2. 配置 Agents（纯配置，System Prompt + Model Name）
	fmt.Println("Step 2: 配置 Agents")
	agentManager := agent.NewManager()

	// 数学专家 Agent（使用本地 Ollama 模型）
	mathAgent := agent.NewAgent(
		"math-expert",
		"Expert in mathematical calculations, arithmetic, algebra, and problem solving with numbers",
		`You are a mathematical expert. You excel at:
- Breaking down complex calculations into steps
- Using the calculator tool accurately
- Explaining mathematical reasoning clearly
- Verifying results for correctness

Always show your work step by step.`,
		"qwen3-local", // 使用本地 Ollama 模型
	)

	// 代码审查 Agent（使用 GPT-4）
	codeReviewAgent := agent.NewAgent(
		"code-reviewer",
		"Expert in code review and software quality assurance",
		`You are a senior code reviewer. You focus on:
- Code quality and best practices
- Security vulnerabilities
- Performance optimization
- Maintainability and readability

Provide constructive feedback with specific examples.`,
		"gpt4-turbo", // 使用 GPT-4
	)

	// 通用助手 Agent（使用 Claude）
	generalAgent := agent.NewAgent(
		"general-assistant",
		"General purpose AI assistant for various tasks",
		`You are a helpful AI assistant. You can:
- Answer questions on various topics
- Help with problem-solving
- Provide explanations and guidance
- Use tools when necessary

Be friendly, clear, and concise.`,
		"claude-opus", // 使用 Claude
	)

	agentManager.Register(mathAgent)
	agentManager.Register(codeReviewAgent)
	agentManager.Register(generalAgent)

	fmt.Printf("✓ 注册了 %d 个 Agent 配置\n", len(agentManager.List()))
	for _, a := range agentManager.List() {
		fmt.Printf("  - %s (Model: %s)\n", a.Name, a.ModelName)
	}
	fmt.Println()

	// 3. 创建 Logger（开发环境）
	fmt.Println("Step 3: 创建 Logger")
	logger, err := log.NewDevelopmentZapLogger()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}
	fmt.Println("✓ Logger 创建完成（使用 Zap 开发模式）\n")

	// 4. 创建 Mock LLM Client（用于演示）
	fmt.Println("Step 4: 创建 Engine")
	mockResponses := []string{
		`Thought: I need to calculate 25 * 8 + 15. Following my mathematical expertise, I'll break this down.
Action: calculator
Parameters: {"operation": "multiply", "a": 25, "b": 8}`,
		`Thought: I got 200 from 25 * 8. Now I need to add 15.
Action: calculator
Parameters: {"operation": "add", "a": 200, "b": 15}`,
		`Thought: Perfect! I've completed the calculation step by step.
Final Answer: The result of 25 * 8 + 15 is 215.

Calculation steps:
1. First: 25 × 8 = 200
2. Then: 200 + 15 = 215
3. Final answer: 215`,
	}
	mockLLM := mock.NewMockClient(mockResponses)

	// 5. 创建 Engine（集成 Agent + Model + Logger）
	// 注意：由于我们使用 Mock LLM，实际不会调用真实的 Model
	// 但架构上已经完整：Agent → Model → LLM Client
	eng := engine.Reactor(
		engine.WithLLMClient(mockLLM),         // 使用 Mock LLM（实际场景会被 Model 动态创建的 Client 替换）
		engine.WithAgentManager(agentManager), // Agent 管理器
		engine.WithModelManager(modelManager), // Model 管理器（暂时不会真正调用，因为 Mock LLM 优先）
		engine.WithLogger(logger),             // 日志记录器
		engine.WithMaxIterations(10),
	)

	// 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	fmt.Println("✓ Engine 创建完成，集成了 Agent、Model 和 Logger\n")

	// 6. 执行任务（自动选择 Agent 和 Model）
	fmt.Println("Step 5: 执行任务")
	fmt.Println("─────────────────────────────────────────")
	task := "Calculate 25 * 8 + 15"
	fmt.Printf("Task: %s\n\n", task)

	startTime := time.Now()
	result := eng.Execute(context.Background(), task, nil)
	executionTime := time.Since(startTime)

	// 6. 显示结果
	fmt.Println("=== 执行结果 ===")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Output: %s\n", result.Output)
	fmt.Printf("Execution Time: %v\n", executionTime)
	fmt.Printf("Total Steps: %d\n\n", len(result.Trace))

	// 7. 显示选择的 Agent 和 Model
	if selectedAgent, ok := result.Metadata["selected_agent"].(string); ok {
		fmt.Printf("✓ Selected Agent: %s\n", selectedAgent)
	}
	if selectedModel, ok := result.Metadata["selected_model"].(string); ok {
		fmt.Printf("✓ Selected Model: %s\n", selectedModel)
	}

	// 8. 显示执行轨迹
	fmt.Println("\n=== 执行轨迹 ===")
	for _, step := range result.Trace {
		if step.Type == "think" || step.Type == "act" || step.Type == "finish" {
			fmt.Printf("[%s] %s\n", step.Type, step.Content)
		}
	}

	fmt.Println("✅ 完整集成演示完成！")
	fmt.Println("\n架构说明：")
	fmt.Println("1. Agent = System Prompt + Model Name（纯配置）")
	fmt.Println("2. Model = Provider + API Config（纯配置）")
	fmt.Println("3. Logger = 统一日志接口 + Zap 实现（依赖注入）")
	fmt.Println("4. Engine 根据任务自动选择 Agent")
	fmt.Println("5. 根据 Agent 的 ModelName 从 ModelManager 获取配置")
	fmt.Println("6. 动态创建 llm.Client 并执行任务")
	fmt.Println("7. 整个流程完全自动化，带有完整的日志记录")
}
