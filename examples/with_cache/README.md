# Caching Example

演示智能缓存系统如何大幅提升性能。

## 功能

- 内存缓存（Memory Cache）
- TTL（Time-To-Live）支持
- 自动缓存键生成（基于任务内容）
- 性能对比展示

## 运行

```bash
go run main.go
```

## 代码说明

这个示例展示了：

1. **创建缓存**：
```go
memCache := cache.NewMemoryCache(
    cache.WithMaxSize(100),              // 最多缓存 100 条
    cache.WithDefaultTTL(1 * time.Hour), // 默认 1 小时过期
)
```

2. **配置 Engine**：
```go
eng := engine.New(
    engine.WithLLMClient(llmClient),
    engine.WithCache(memCache),  // 启用缓存
)
```

3. **性能对比**：
   - 第一次执行：调用 LLM，耗时 7-14 秒
   - 第二次执行：命中缓存，耗时 ~7 微秒
   - **性能提升：~900,000 倍**

## 缓存机制

- **缓存键**：基于任务内容的 SHA256 哈希
- **缓存内容**：完整的执行结果（包括输出、轨迹等）
- **过期策略**：TTL 到期自动清除

## 适用场景

- 重复性任务（如定时查询）
- 高频相同请求
- 开发调试（避免重复调用 LLM）
- 降低 LLM API 成本

## 配置选项

```go
cache.NewMemoryCache(
    cache.WithMaxSize(100),           // 最大缓存条目数
    cache.WithDefaultTTL(time.Hour),  // 默认过期时间
)
```

## 注意事项

- 缓存是基于任务字符串的精确匹配
- 不同的任务描述会产生不同的缓存键
- 缓存仅在内存中，重启后清空

## 下一步

- 查看 [thinker_middleware](../thinker_middleware/) 示例了解更高级的缓存策略（意图缓存）
