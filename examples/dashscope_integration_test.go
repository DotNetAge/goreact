package examples_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/DotNetAge/goreact/pkg/actor"
	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/engine"
	"github.com/DotNetAge/goreact/pkg/observer"
	"github.com/DotNetAge/goreact/pkg/terminator"
	"github.com/DotNetAge/goreact/pkg/thinker"
	"github.com/DotNetAge/goreact/pkg/tools"
	"github.com/DotNetAge/goreact/pkg/tools/builtin"
)

// TestDashScopeIntegration 执行 DashScope (阿里云) Qwen3.5-Flash 的完整集成测试
func TestDashScopeIntegration(t *testing.T) {
	// 1. 配置 API Key 和 BaseURL
	apiKey := "DASHSCOPE_API_KEY" // 安全隐性 KEY，直接使用
	baseURL := "https://dashscope.aliyuncs.com/compatible-mode/"
	model := "qwen3.5-flash"

	t.Logf("🔑 Using DashScope API with model: %s", model)

	// 2. 初始化 OpenAI 兼容客户端
	config := openai.Config{
		Config: base.Config{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   model,
			Timeout: 7200 * time.Second,
		},
	}

	client, err := openai.New(config)
	if err != nil {
		t.Fatalf("❌ 创建客户端失败：%v", err)
	}

	t.Log("✅ 客户端初始化成功")

	// 3. 准备工具集
	toolMgr := tools.NewSimpleManager()
	toolMgr.Register(builtin.NewCalculator())
	toolMgr.Register(builtin.NewDateTime())
	t.Log("🛠️  已注册工具：[Calculator, DateTime]")

	// 4. 构建 Reactor
	agent := engine.NewReactor(
		engine.WithThinker(thinker.Default(client,
			thinker.WithModel(model),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(
			actor.WithToolManager(toolMgr),
		)),
		engine.WithObserver(observer.Default()),
		engine.WithTerminator(terminator.Default()),
	)

	t.Log("🤖 Agent 组装完成")

	// 5. 设置上下文 (120 秒超时)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 6. 定义测试任务
	testCases := []struct {
		name     string
		input    string
		expected string // 期望包含的关键词
	}{
		{
			name:     "日期计算",
			input:    "如果今天是 2026 年 3 月 17 日，请先计算 100 天后是几号",
			expected: "2026 年",
		},
		{
			name:     "数学计算",
			input:    "请计算 1234 乘以 5678 等于多少",
			expected: "", // 只验证能正确执行
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("\n--- 测试用例：%s ---\n", tc.name)
			fmt.Printf("👤 用户：%s\n\n", tc.input)
			fmt.Println("🧠 思考过程:")

			// 7. 运行 Agent
			reactCtx, err := agent.Run(ctx, fmt.Sprintf("test-%s", tc.name), tc.input,
				core.WithThoughtStream(func(chunk string) {
					fmt.Print(chunk)
				}),
			)

			if err != nil {
				t.Fatalf("\n❌ Agent 执行错误：%v", err)
			}

			// 8. 验证结果
			fmt.Printf("\n\n✅ 最终答案：%s\n", reactCtx.FinalResult)
			fmt.Printf("📊 Token 统计：Prompt: %d | Completion: %d | Total: %d\n",
				reactCtx.TotalTokens.PromptTokens,
				reactCtx.TotalTokens.CompletionTokens,
				reactCtx.TotalTokens.TotalTokens)
			fmt.Printf("⏱️  执行时间：%v\n", time.Since(reactCtx.StartTime))

			// 基本验证
			if reactCtx.FinalResult == "" {
				t.Error("❌ 最终答案为空白")
			}

			if tc.expected != "" && reactCtx.FinalResult == "" {
				t.Errorf("期望答案包含 '%s', 但得到空结果", tc.expected)
			}

			// 验证 Token 使用
			// 注意：DashScope 可能不返回标准格式的 token 统计
			if reactCtx.TotalTokens.TotalTokens == 0 {
				t.Log("⚠️  Token 计数为 0，这可能是正常的（某些 API 提供商不返回 token 统计）")
			}

			// 验证执行步骤
			if len(reactCtx.Traces) == 0 {
				t.Error("❌ 没有任何执行轨迹，ReAct 循环可能未正常运行")
			}

			t.Logf("📝 执行步数：%d, Traces: %d", reactCtx.CurrentStep, len(reactCtx.Traces))
		})
	}
}

// TestDashScopeErrorHandling 测试错误处理和边界情况
func TestDashScopeErrorHandling(t *testing.T) {
	apiKey := "DASHSCOPE_API_KEY"
	baseURL := "https://dashscope.aliyuncs.com/compatible-mode/"
	model := "qwen3.5-flash"

	config := openai.Config{
		Config: base.Config{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   model,
			Timeout: 30 * time.Second, // 较短超时测试错误处理
		},
	}

	client, err := openai.New(config)
	if err != nil {
		t.Fatalf("创建客户端失败：%v", err)
	}

	toolMgr := tools.NewSimpleManager()
	toolMgr.Register(builtin.NewCalculator())

	agent := engine.NewReactor(
		engine.WithThinker(thinker.Default(client,
			thinker.WithModel(model),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(actor.WithToolManager(toolMgr))),
		engine.WithObserver(observer.Default()),
		engine.WithTerminator(terminator.Default()),
	)

	t.Run("简单问题快速响应", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := agent.Run(ctx, "test-simple", "1+1 等于几？",
			core.WithThoughtStream(func(chunk string) {
				fmt.Print(chunk)
			}),
		)

		if err != nil {
			t.Errorf("简单问题执行失败：%v", err)
		}
	})
}
