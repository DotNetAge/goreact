# Middleware 使用指南

## 什么是 Middleware？

Middleware（中间件）是一种在 Think 执行前后插入自定义逻辑的机制，借鉴了 Web 开发中的成熟模式。

```
Request → Middleware 1 → Middleware 2 → Thinker → Middleware 2 → Middleware 1 → Response
          (Pre-Think)                              (Post-Think)
```

---

## 为什么需要 Middleware？

### 用户痛点

1. **增强输入**：如何在 Think 之前获取相关知识（RAG）、用户画像、历史意图？
2. **优化输出**：如何评估结果置信度、验证输出格式？
3. **可靠性**：如何处理 LLM 调用失败、速率限制？
4. **可观测性**：如何记录日志、监控性能？

### Middleware 解决方案

```go
// Pre-Think：增强输入
thinker.Use(middlewares.RAGMiddleware(retriever, 3))         // 检索相关知识
thinker.Use(middlewares.UserProfileMiddleware(profileStore)) // 加载用户画像
thinker.Use(middlewares.IntentCacheMiddleware(cache, 0.9))   // 查询意图缓存

// Post-Think：优化输出
thinker.Use(middlewares.ConfidenceMiddleware(0.7, nil))      // 评估置信度

// 可靠性
thinker.Use(middlewares.RetryMiddleware(3, time.Second))     // 自动重试
thinker.Use(middlewares.RateLimitMiddleware(limiter))        // 速率限制

// 可观测性
thinker.Use(middlewares.LoggingMiddleware(logger))           // 日志记录
```

---

## 内置 Middleware

### 1. LoggingMiddleware - 日志记录

记录每次 Think 的输入、输出、耗时。

```go
import "github.com/ray/goreact/pkg/core/thinker/middlewares"

// 使用默认 logger
thinker.Use(middlewares.LoggingMiddleware(nil))

// 自定义 logger
type MyLogger struct{}

func (l *MyLogger) Info(msg string, fields map[string]interface{}) {
	// 你的日志逻辑
}

func (l *MyLogger) Error(msg string, err error, fields map[string]interface{}) {
	// 你的错误日志逻辑
}

thinker.Use(middlewares.LoggingMiddleware(&MyLogger{}))
```

**输出示例**：
```
[INFO] Think started {"task": "Calculate 1+1", "timestamp": "2024-01-01T10:00:00Z"}
[INFO] Think completed {"duration_ms": 150, "success": true, "action": "calculator"}
```

---

### 2. RetryMiddleware - 自动重试

LLM 调用失败时自动重试。

```go
// 最多重试 3 次，每次间隔 1 秒
thinker.Use(middlewares.RetryMiddleware(3, time.Second))
```

**工作原理**：
```
Attempt 1: Failed → Wait 1s
Attempt 2: Failed → Wait 1s
Attempt 3: Success → Return
```

**元数据**：
```go
thought.Metadata["retry_attempts"] = 2 // 重试了 2 次
```

---

### 3. IntentCacheMiddleware - 意图缓存

缓存相似任务的结果，避免重复调用 LLM。

```go
cache := middlewares.NewMemoryIntentCache()
thinker.Use(middlewares.IntentCacheMiddleware(cache, 0.85)) // 相似度阈值 0.85
```

**工作原理**：
```
Task: "Calculate 10+5"
  → 检查缓存
  → 未命中，调用 LLM
  → 缓存结果

Task: "Calculate 10+5" (相同任务)
  → 检查缓存
  → 命中！直接返回
```

**元数据**：
```go
thought.Metadata["cached"] = true
thought.Metadata["cache_hit"] = true
thought.Metadata["similarity"] = 1.0
```

**自定义缓存**：
```go
type MyCache struct {
	// 你的缓存实现（Redis、Memcached 等）
}

func (c *MyCache) Get(key string) (*middlewares.CachedIntent, bool) { ... }
func (c *MyCache) Set(key string, intent *middlewares.CachedIntent) { ... }

thinker.Use(middlewares.IntentCacheMiddleware(&MyCache{}, 0.85))
```

---

### 4. RAGMiddleware - RAG 增强

在 Think 之前检索相关知识，增强输入。

```go
type MyRAGRetriever struct {
	// 你的 RAG 实现
}

func (r *MyRAGRetriever) Retrieve(query string, topK int) ([]middlewares.Document, error) {
	// 检索逻辑
	return docs, nil
}

retriever := &MyRAGRetriever{}
thinker.Use(middlewares.RAGMiddleware(retriever, 3)) // 检索 top 3 文档
```

**增强效果**：
```
原始任务: "What is the capital of France?"

增强后:
What is the capital of France?

Relevant context from knowledge base:

[Document 1] (relevance: 0.95)
France is a country in Europe. Its capital is Paris.

[Document 2] (relevance: 0.87)
Paris is the largest city in France...
```

**元数据**：
```go
ctx.Get("rag_documents")  // 检索到的文档
ctx.Get("rag_enhanced")   // true
ctx.Get("rag_doc_count")  // 3
```

---

### 5. UserProfileMiddleware - 用户画像

加载用户画像，个性化响应。

```go
type MyProfileStore struct{}

func (s *MyProfileStore) Get(userID string) (*middlewares.UserProfile, error) {
	return &middlewares.UserProfile{
		UserID: userID,
		Name:   "Alice",
		Preferences: map[string]interface{}{
			"language": "zh-CN",
			"tone":     "friendly",
		},
	}, nil
}

thinker.Use(middlewares.UserProfileMiddleware(&MyProfileStore{}))
```

**使用**：
```go
ctx := core.NewContext()
ctx.Set("user_id", "user123") // 设置用户 ID

thought, _ := thinker.Think(task, ctx)
```

**增强效果**：
```
原始任务: "Help me"

增强后:
Help me

User context: User: Alice, Preferences: map[language:zh-CN tone:friendly]
```

---

### 6. ConfidenceMiddleware - 置信度评估

评估 Thought 的置信度，低置信度时标记需要澄清。

```go
// 最低置信度 0.7
thinker.Use(middlewares.ConfidenceMiddleware(0.7, nil))

// 自定义评估器
type MyEvaluator struct{}

func (e *MyEvaluator) Evaluate(thought *types.Thought, ctx *core.Context) float64 {
	// 你的评估逻辑
	return 0.85
}

thinker.Use(middlewares.ConfidenceMiddleware(0.7, &MyEvaluator{}))
```

**评估因素**（默认）：
- 是否有明确的 Action 或 FinalAnswer
- Reasoning 的长度
- Parameters 是否完整
- 是否有 RAG 增强

**元数据**：
```go
thought.Metadata["confidence"] = 0.85

// 如果低于阈值
thought.Metadata["needs_clarification"] = true
thought.Metadata["clarification_reason"] = "Low confidence in intent recognition"
```

---

### 7. RateLimitMiddleware - 速率限制

限制 Think 调用频率，避免超出 LLM API 限制。

```go
// 令牌桶：容量 10，每秒补充 2 个
limiter := middlewares.NewTokenBucketLimiter(10, 2)
thinker.Use(middlewares.RateLimitMiddleware(limiter))
```

**工作原理**：
```
Request 1-10: 通过（消耗 10 个令牌）
Request 11:   拒绝（令牌不足）
等待 1 秒:    补充 2 个令牌
Request 12-13: 通过
```

**错误**：
```go
_, err := thinker.Think(task, ctx)
// err: "rate limit exceeded, please try again later"
```

---

## 编写自定义 Middleware

### 基本结构

```go
func MyMiddleware() thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 1. Pre-Think：处理输入
			task = strings.ToUpper(task) // 示例：转大写

			// 2. 调用下一个处理器
			thought, err := next(task, ctx)

			// 3. Post-Think：处理输出
			if thought != nil {
				thought.Metadata["custom_flag"] = true
			}

			return thought, err
		}
	}
}

// 使用
thinker.Use(MyMiddleware())
```

### 高级示例：短路

```go
func CacheMiddleware(cache Cache) thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 检查缓存
			if cached, ok := cache.Get(task); ok {
				// 短路：直接返回，不调用 next
				return cached, nil
			}

			// 缓存未命中，调用 next
			thought, err := next(task, ctx)

			// 缓存结果
			if err == nil {
				cache.Set(task, thought)
			}

			return thought, err
		}
	}
}
```

### 高级示例：错误处理

```go
func ErrorHandlingMiddleware() thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			thought, err := next(task, ctx)

			if err != nil {
				// 记录错误
				log.Printf("Think error: %v", err)

				// 返回降级响应
				return &types.Thought{
					Reasoning:    "Error occurred, using fallback",
					ShouldFinish: true,
					FinalAnswer:  "Sorry, I encountered an error. Please try again.",
				}, nil // 吞掉错误
			}

			return thought, nil
		}
	}
}
```

---

## Middleware 组合

### 顺序很重要

```go
// 正确顺序
thinker.Use(middlewares.LoggingMiddleware(nil))      // 1. 最外层：记录所有
thinker.Use(middlewares.RateLimitMiddleware(limiter)) // 2. 速率限制
thinker.Use(middlewares.RetryMiddleware(3, time.Second)) // 3. 重试
thinker.Use(middlewares.IntentCacheMiddleware(cache, 0.9)) // 4. 缓存
thinker.Use(middlewares.RAGMiddleware(retriever, 3))  // 5. RAG 增强
thinker.Use(middlewares.ConfidenceMiddleware(0.7, nil)) // 6. 最内层：评估结果
```

**执行流程**（洋葱模型）：
```
Request
  → Logging (start)
    → RateLimit (check)
      → Retry (attempt 1)
        → Cache (miss)
          → RAG (enhance)
            → Thinker (think)
          → Confidence (evaluate)
        → Cache (set)
      → Retry (success)
    → RateLimit (pass)
  → Logging (end)
Response
```

### 条件组合

```go
// 开发环境
if isDev {
	thinker.Use(middlewares.LoggingMiddleware(nil))
}

// 生产环境
if isProd {
	thinker.Use(middlewares.RateLimitMiddleware(limiter))
	thinker.Use(middlewares.RetryMiddleware(3, time.Second))
	thinker.Use(middlewares.IntentCacheMiddleware(cache, 0.9))
}

// 高级功能
if enableRAG {
	thinker.Use(middlewares.RAGMiddleware(retriever, 3))
}
```

---

## 最佳实践

### 1. 性能优化

```go
// ✅ 好：缓存在外层
thinker.Use(middlewares.IntentCacheMiddleware(cache, 0.9))
thinker.Use(middlewares.RAGMiddleware(retriever, 3))

// ❌ 差：缓存在内层（RAG 总是执行）
thinker.Use(middlewares.RAGMiddleware(retriever, 3))
thinker.Use(middlewares.IntentCacheMiddleware(cache, 0.9))
```

### 2. 错误处理

```go
// ✅ 好：记录错误但不中断
func SafeMiddleware() thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 尝试增强
			if err := enhance(task, ctx); err != nil {
				log.Printf("Enhancement failed: %v", err)
				// 继续执行，不中断
			}

			return next(task, ctx)
		}
	}
}
```

### 3. 元数据传递

```go
// 使用 Context 传递数据
func Middleware1() thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			ctx.Set("middleware1_data", "some value")
			return next(task, ctx)
		}
	}
}

func Middleware2() thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 读取 Middleware1 的数据
			if data, ok := ctx.Get("middleware1_data"); ok {
				// 使用 data
			}
			return next(task, ctx)
		}
	}
}
```

---

## 示例代码

完整示例请查看：
- [examples/thinker_middleware/](../examples/thinker_middleware/) - 完整 Middleware 示例
- [examples/rag_enhanced/](../examples/rag_enhanced/) - RAG 增强示例

---

## 下一步

- [Thinker 指南](./THINKER_GUIDE.md) - 深入了解 Thinker
- [快速开始](./QUICK_START.md) - 5 分钟上手
- [架构文档](../ARCHITECTURE.md) - 理解整体设计
