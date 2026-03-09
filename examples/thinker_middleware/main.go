package main

import (
	"fmt"
	"context"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/core/thinker/middlewares"
	"github.com/ray/goreact/pkg/core/thinker/presets"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/llm/mock"
	"github.com/ray/goreact/pkg/tool/builtin"
	"github.com/ray/goreact/pkg/types"
)

// MockRAGRetriever 模拟 RAG 检索器
type MockRAGRetriever struct{}

func (r *MockRAGRetriever) Retrieve(query string, topK int) ([]middlewares.Document, error) {
	// 模拟返回相关文档
	return []middlewares.Document{
		{
			ID:      "doc1",
			Content: "The calculator tool can perform basic arithmetic operations like addition, subtraction, multiplication, and division.",
			Score:   0.95,
		},
		{
			ID:      "doc2",
			Content: "For mathematical calculations, use the calculator tool with operation parameter.",
			Score:   0.87,
		},
	}, nil
}

// MockUserProfileStore 模拟用户画像存储
type MockUserProfileStore struct{}

func (s *MockUserProfileStore) Get(userID string) (*middlewares.UserProfile, error) {
	return &middlewares.UserProfile{
		UserID: userID,
		Name:   "Alice",
		Preferences: map[string]interface{}{
			"language": "English",
			"timezone": "UTC",
		},
		History: []string{
			"Previous calculation: 10 + 5 = 15",
		},
	}, nil
}

func main() {
	fmt.Println("=== Thinker Middleware 示例 ===\n")

	// 1. 创建基础 Thinker
	llmClient := mock.NewMockClient([]string{
		"Thought: I need to calculate 100 + 200\nAction: calculator\nParameters: {\"operation\": \"add\", \"a\": 100, \"b\": 200}\nReasoning: Using calculator for addition",
	})
	baseThinker := presets.NewReActThinker(llmClient, "calculator: perform arithmetic operations")

	// 2. 创建支持中间件的 Thinker
	mt := thinker.NewMiddlewareThinker(baseThinker)

	// 3. 添加中间件（按需组合）
	fmt.Println("添加中间件:")

	// 日志中间件
	fmt.Println("  - LoggingMiddleware: 记录每次 Think 的输入输出和耗时")
	mt.Use(middlewares.LoggingMiddleware(nil))

	// 速率限制中间件
	fmt.Println("  - RateLimitMiddleware: 限制每秒最多 10 次请求")
	limiter := middlewares.NewTokenBucketLimiter(10, 10)
	mt.Use(middlewares.RateLimitMiddleware(limiter))

	// RAG 增强中间件
	fmt.Println("  - RAGMiddleware: 检索相关知识增强输入")
	ragRetriever := &MockRAGRetriever{}
	mt.Use(middlewares.RAGMiddleware(ragRetriever, 2))

	// 用户画像中间件
	fmt.Println("  - UserProfileMiddleware: 加载用户偏好")
	userStore := &MockUserProfileStore{}
	mt.Use(middlewares.UserProfileMiddleware(userStore))

	// 意图缓存中间件
	fmt.Println("  - IntentCacheMiddleware: 缓存相似意图")
	intentCache := middlewares.NewMemoryIntentCache()
	mt.Use(middlewares.IntentCacheMiddleware(intentCache, 0.85))

	// 重试中间件
	fmt.Println("  - RetryMiddleware: LLM 失败时自动重试")
	mt.Use(middlewares.RetryMiddleware(3, 1*time.Second))

	// 置信度评估中间件
	fmt.Println("  - ConfidenceMiddleware: 评估结果置信度")
	mt.Use(middlewares.ConfidenceMiddleware(0.7, nil))

	fmt.Println()

	// 4. 创建 Engine 并使用增强的 Thinker
	eng := engine.New(
		engine.WithThinker(mt),
	)

	// 注册工具
	eng.RegisterTool(builtin.NewCalculator())

	// 5. 执行任务
	fmt.Println("=== 执行任务 ===\n")

	ctx := core.NewContext()
	ctx.Set("user_id", "user123") // 设置用户 ID

	result := eng.Execute(context.Background(), "Calculate 100 + 200", ctx)

	fmt.Printf("\n结果: %s\n", result.Output)
	fmt.Printf("成功: %v\n", result.Success)

	// 6. 第二次执行相同任务（测试缓存）
	fmt.Println("\n=== 第二次执行相同任务（应该命中缓存）===\n")
	result2 := eng.Execute(context.Background(), "Calculate 100 + 200", ctx)
	fmt.Printf("结果: %s\n", result2.Output)

	// 7. 展示如何自定义中间件
	fmt.Println("\n=== 自定义中间件示例 ===\n")

	customThinker := thinker.NewMiddlewareThinker(baseThinker)

	// 自定义中间件：添加时间戳
	customThinker.Use(func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			fmt.Printf("[CustomMiddleware] Task received at: %s\n", time.Now().Format(time.RFC3339))

			thought, err := next(task, ctx)

			if thought != nil && thought.Metadata == nil {
				thought.Metadata = make(map[string]interface{})
			}
			if thought != nil {
				thought.Metadata["custom_timestamp"] = time.Now().Unix()
			}

			return thought, err
		}
	})

	eng2 := engine.New(engine.WithThinker(customThinker))
	eng2.RegisterTool(builtin.NewCalculator())

	result3 := eng2.Execute(context.Background(), "Calculate 50 + 50", core.NewContext())
	fmt.Printf("结果: %s\n", result3.Output)

	fmt.Println("\n=== 示例完成 ===")
}
