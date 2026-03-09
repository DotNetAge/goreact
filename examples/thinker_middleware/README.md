# Thinker Middleware Example

演示强大的 Thinker 中间件系统 - 借鉴 Web 框架的中间件模式来增强 Think 阶段。

## 概述

Thinker Middleware 允许你在 Think 执行前后插入自定义逻辑，实现：
- 日志记录
- 自动重试
- 智能缓存
- RAG 知识增强
- 用户画像
- 置信度评估
- 速率限制

## 运行

```bash
go run main.go
```

## 核心概念

### 洋葱模型（Onion Model）

中间件按添加顺序形成层层包裹的结构：

```
Request → Logging → RateLimit → Retry → Cache → RAG → Thinker → Response
          ↓                                                        ↑
          └────────────────────────────────────────────────────────┘
```

### 中间件链

```go
// 1. 创建基础 Thinker
baseThinker := presets.NewReActThinker(llmClient, toolsDesc)

// 2. 创建支持中间件的 Thinker
mt := thinker.NewMiddlewareThinker(baseThinker)

// 3. 添加中间件（按需组合）
mt.Use(
    middlewares.LoggingMiddleware(nil),
    middlewares.RetryMiddleware(3, time.Second),
    middlewares.IntentCacheMiddleware(cache, 0.85),
)
```

## 内置中间件

### 1. LoggingMiddleware - 日志记录

记录每次 Think 的输入、输出和执行时间。

```go
mt.Use(middlewares.LoggingMiddleware(logger))
```

**输出示例**：
```
[Logging] Task: Calculate 100 + 200
[Logging] Duration: 125ms
[Logging] Success: true
```

### 2. RetryMiddleware - 自动重试

LLM 调用失败时自动重试。

```go
mt.Use(middlewares.RetryMiddleware(
    3,              // 最多重试 3 次
    1*time.Second,  // 每次间隔 1 秒
))
```

**适用场景**：
- 网络不稳定
- LLM 服务偶发性错误
- 提高系统可靠性

### 3. IntentCacheMiddleware - 意图缓存

基于语义相似度的智能缓存，不需要完全匹配。

```go
cache := middlewares.NewMemoryIntentCache()
mt.Use(middlewares.IntentCacheMiddleware(
    cache,
    0.85,  // 相似度阈值（0.0-1.0）
))
```

**示例**：
- "Calculate 10+5" 和 "计算 10 加 5" 可能命中同一缓存
- 相似度 >= 0.85 时返回缓存结果

### 4. RAGMiddleware - 知识增强

在 Think 前检索相关知识，增强 LLM 的上下文。

```go
mt.Use(middlewares.RAGMiddleware(
    ragRetriever,  // 实现 RAGRetriever 接口
    2,             // 检索 Top-2 文档
))
```

**工作流程**：
1. 根据任务检索相关文档
2. 将文档内容注入 Context
3. Thinker 可以利用这些知识进行推理

### 5. UserProfileMiddleware - 用户画像

加载用户偏好和历史记录，实现个性化。

```go
mt.Use(middlewares.UserProfileMiddleware(userStore))
```

**Context 注入**：
```go
ctx.Set("user_id", "user123")
// 中间件自动加载用户画像到 ctx.Get("user_profile")
```

### 6. ConfidenceMiddleware - 置信度评估

评估 Thought 的置信度，标记低置信度结果。

```go
mt.Use(middlewares.ConfidenceMiddleware(
    0.7,  // 置信度阈值
    nil,  // 可选的评估器
))
```

**输出**：
- `thought.Metadata["confidence"]` - 置信度分数
- `thought.Metadata["low_confidence"]` - 是否低置信度

### 7. RateLimitMiddleware - 速率限制

基于令牌桶算法的速率限制。

```go
limiter := middlewares.NewTokenBucketLimiter(
    10,  // 容量：10 个令牌
    10,  // 速率：每秒补充 10 个
)
mt.Use(middlewares.RateLimitMiddleware(limiter))
```

**适用场景**：
- 控制 LLM API 调用频率
- 防止突发流量
- 成本控制

## 自定义中间件

创建自定义中间件非常简单：

```go
func MyCustomMiddleware() thinker.ThinkMiddleware {
    return func(next thinker.ThinkHandler) thinker.ThinkHandler {
        return func(task string, ctx *core.Context) (*types.Thought, error) {
            // 前置处理
            fmt.Println("Before Think:", task)

            // 调用下一个处理器
            thought, err := next(task, ctx)

            // 后置处理
            if thought != nil {
                thought.Metadata["custom_field"] = "value"
            }

            return thought, err
        }
    }
}

// 使用
mt.Use(MyCustomMiddleware())
```

## 中间件顺序

中间件的添加顺序很重要！推荐顺序：

```go
mt.Use(
    middlewares.LoggingMiddleware(nil),        // 1. 最外层：记录所有
    middlewares.RateLimitMiddleware(limiter),  // 2. 速率控制
    middlewares.RetryMiddleware(3, time.Second), // 3. 重试机制
    middlewares.IntentCacheMiddleware(cache, 0.85), // 4. 缓存（避免重复操作）
    middlewares.RAGMiddleware(rag, 2),         // 5. 知识增强
    middlewares.UserProfileMiddleware(store),  // 6. 用户上下文
    middlewares.ConfidenceMiddleware(0.7, nil), // 7. 最内层：结果评估
)
```

**原则**：
- 缓存放在外层（避免不必要的内层操作）
- 日志放在最外层（记录完整流程）
- 评估放在最内层（评估最终结果）

## 示例输出

```
=== Thinker Middleware 示例 ===

添加中间件:
  - LoggingMiddleware: 记录每次 Think 的输入输出和耗时
  - RateLimitMiddleware: 限制每秒最多 10 次请求
  - RAGMiddleware: 检索相关知识增强输入
  - UserProfileMiddleware: 加载用户偏好
  - IntentCacheMiddleware: 缓存相似意图
  - RetryMiddleware: LLM 失败时自动重试
  - ConfidenceMiddleware: 评估结果置信度

=== 执行任务 ===

[RAG] Retrieved 2 documents for task
[UserProfile] Loaded profile for user: user123
[Logging] Task: Calculate 100 + 200
[Logging] Duration: 125ms
[Confidence] Score: 0.95

结果: 300
成功: true

=== 第二次执行相同任务（应该命中缓存）===

[IntentCache] Cache hit! Similarity: 1.00
结果: 300
```

## 最佳实践

1. **按需组合**：只添加需要的中间件，避免不必要的开销
2. **注意顺序**：缓存应该在外层，避免重复执行内层逻辑
3. **错误处理**：中间件应该优雅处理错误，不要破坏链条
4. **Context 传递**：使用 Context 在中间件间传递数据
5. **性能监控**：使用 LoggingMiddleware 监控各环节耗时

## 相关文档

- [MIDDLEWARE_GUIDE.md](../../docs/MIDDLEWARE_GUIDE.md) - 完整的中间件指南
- [THINKER_GUIDE.md](../../docs/THINKER_GUIDE.md) - Thinker 组件详解
- [QUICK_START.md](../../docs/QUICK_START.md) - 快速开始

## 下一步

- 实现自己的 RAGRetriever 集成真实的向量数据库
- 实现自己的 UserProfileStore 集成用户系统
- 创建业务特定的自定义中间件
