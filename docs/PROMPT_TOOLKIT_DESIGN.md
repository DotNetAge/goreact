# Prompt Toolkit 设计方案

## 概述

为 Thinker 开发提供完整的 Prompt 构建和上下文管理工具箱，解决 LLM 调用前的两大核心任务。

## 设计原则

1. **开箱即用**：提供合理的默认配置
2. **渐进增强**：从简单到复杂，按需使用
3. **可组合**：各组件独立，可自由组合
4. **可观测**：提供详细的调试信息
5. **中间件友好**：易于在中间件中使用

---

## 一、Prompt 构建工具

### 1.1 增强的 PromptBuilder

#### 当前问题
- 模板系统简单，只支持基础变量替换
- 工具描述格式化简陋
- 缺少 few-shot 示例管理
- 无法动态选择工具

#### 改进方案

```go
// FluentPromptBuilder 流式 API 的 Prompt 构建器
type FluentPromptBuilder struct {
    systemPrompt    string
    userPrompt      string
    tools           []ToolDesc
    fewShots        []FewShotExample
    variables       map[string]interface{}
    toolFormatter   ToolFormatter
    historyFormatter HistoryFormatter
    maxTokens       int
    tokenCounter    TokenCounter
}

// 使用示例
prompt := builder.New().
    WithSystemPrompt("You are a helpful assistant").
    WithTask(task).
    WithTools(tools).
    WithFewShots(examples).
    WithHistory(history, 10).  // 最多 10 轮
    WithMaxTokens(4096).
    Build()
```

### 1.2 工具描述格式化器

#### 支持多种格式

```go
type ToolFormatter interface {
    Format(tools []ToolDesc) string
}

// 1. 简单文本格式（当前）
type SimpleTextFormatter struct{}
// 输出：
// 1. calculator: Perform arithmetic operations
// 2. http: Make HTTP requests

// 2. JSON Schema 格式（推荐）
type JSONSchemaFormatter struct{}
// 输出：
// {
//   "name": "calculator",
//   "description": "Perform arithmetic operations",
//   "parameters": {
//     "type": "object",
//     "properties": {
//       "operation": {"type": "string", "enum": ["add", "subtract"]},
//       "a": {"type": "number"},
//       "b": {"type": "number"}
//     },
//     "required": ["operation", "a", "b"]
//   }
// }

// 3. Markdown 格式（可读性好）
type MarkdownFormatter struct{}
// 输出：
// ## Available Tools
//
// ### calculator
// Perform arithmetic operations
//
// **Parameters:**
// - operation (string): add, subtract, multiply, divide
// - a (number): First operand
// - b (number): Second operand

// 4. 智能格式化器（根据 LLM 类型自动选择）
type SmartFormatter struct {
    llmType string // "openai", "anthropic", "ollama"
}
```

### 1.3 Few-Shot 示例管理

```go
type FewShotExample struct {
    Task      string
    Thought   string
    Action    string
    Parameters map[string]interface{}
    Result    string
}

type FewShotManager struct {
    examples []FewShotExample
    selector FewShotSelector
}

// 选择策略
type FewShotSelector interface {
    Select(task string, maxExamples int) []FewShotExample
}

// 1. 随机选择
type RandomSelector struct{}

// 2. 相似度选择（基于任务相似度）
type SimilaritySelector struct {
    embedder Embedder
}

// 3. 分类选择（基于任务类型）
type CategorySelector struct {
    categories map[string][]FewShotExample
}

// 使用示例
fewShots := manager.Select(task, 3) // 选择 3 个最相关的示例
builder.WithFewShots(fewShots)
```

### 1.4 动态工具选择

```go
// ToolSelector 根据任务动态选择相关工具
type ToolSelector interface {
    Select(task string, allTools []ToolDesc, maxTools int) []ToolDesc
}

// 1. 全部工具（当前）
type AllToolsSelector struct{}

// 2. 关键词匹配
type KeywordSelector struct {
    keywords map[string][]string // tool_name -> keywords
}

// 3. 语义匹配（使用 embedding）
type SemanticSelector struct {
    embedder Embedder
}

// 4. LLM 辅助选择（两阶段：先选工具，再执行）
type LLMAssistedSelector struct {
    llmClient llm.Client
}

// 使用示例
selector := NewSemanticSelector(embedder)
relevantTools := selector.Select(task, allTools, 5) // 最多 5 个工具
builder.WithTools(relevantTools)
```

---

## 二、上下文管理工具

### 2.1 精确的 Token 计数

#### 当前问题
- 简单的字符数估算（1 token ≈ 4 chars）不准确
- 不同 LLM 的 tokenizer 不同

#### 改进方案

```go
type TokenCounter interface {
    Count(text string) int
    CountMessages(messages []Message) int
}

// 1. 简单估算器（当前）
type SimpleEstimator struct{}

// 2. TikToken（OpenAI）
type TikTokenCounter struct {
    encoding string // "cl100k_base", "p50k_base"
}

// 3. SentencePiece（Llama, Qwen）
type SentencePieceCounter struct {
    modelPath string
}

// 4. 通用估算器（基于正则和统计）
type UniversalEstimator struct {
    language string // "en", "zh", "mixed"
}

// 使用示例
counter := NewTikTokenCounter("cl100k_base")
tokens := counter.Count(prompt)
if tokens > maxTokens {
    // 需要压缩
}
```

### 2.2 智能上下文压缩

#### 当前问题
- 只有简单的截断和滑动窗口
- 没有考虑消息重要性
- 没有语义压缩

#### 改进方案

```go
type CompressionStrategy interface {
    Compress(turns []Turn, maxTokens int, counter TokenCounter) ([]Turn, error)
}

// 1. 截断策略（当前）
type TruncateStrategy struct{}

// 2. 滑动窗口（当前）
type SlidingWindowStrategy struct{}

// 3. 优先级压缩（保留重要消息）
type PriorityStrategy struct {
    priorities map[string]int // role -> priority
}
// 优先级：system > user > assistant
// 保留最近的用户消息和系统消息

// 4. 语义压缩（合并相似消息）
type SemanticStrategy struct {
    embedder Embedder
    threshold float64 // 相似度阈值
}

// 5. LLM 摘要压缩
type SummarizeStrategy struct {
    llmClient llm.Client
    template  string
}
// 将多轮对话摘要为一段文字

// 6. 混合策略（组合多种策略）
type HybridStrategy struct {
    strategies []CompressionStrategy
}

// 使用示例
strategy := NewPriorityStrategy(map[string]int{
    "system": 100,
    "user": 80,
    "assistant": 60,
})
compressed, err := strategy.Compress(turns, maxTokens, counter)
```

### 2.3 上下文窗口管理器

```go
// ContextWindow 上下文窗口管理器
type ContextWindow struct {
    turns         []Turn
    maxTokens     int
    counter       TokenCounter
    strategy      CompressionStrategy
    reservedTokens int // 为输出预留的 tokens
}

// 智能添加
func (w *ContextWindow) Add(turn Turn) error {
    w.turns = append(w.turns, turn)

    // 检查是否超限
    currentTokens := w.counter.CountMessages(w.turns)
    if currentTokens > w.maxTokens - w.reservedTokens {
        // 自动压缩
        compressed, err := w.strategy.Compress(w.turns, w.maxTokens - w.reservedTokens, w.counter)
        if err != nil {
            return err
        }
        w.turns = compressed
    }

    return nil
}

// 获取可用空间
func (w *ContextWindow) AvailableTokens() int {
    used := w.counter.CountMessages(w.turns)
    return w.maxTokens - used - w.reservedTokens
}

// 使用示例
window := NewContextWindow(4096, counter, strategy)
window.SetReservedTokens(512) // 为输出预留 512 tokens
window.Add(Turn{Role: "user", Content: "Hello"})
```

---

## 三、历史记录格式化

### 3.1 HistoryFormatter

```go
type HistoryFormatter interface {
    Format(turns []Turn, maxTurns int) string
}

// 1. 简单格式（当前）
type SimpleFormatter struct{}
// [user]: Hello
// [assistant]: Hi there!

// 2. 对话格式
type ConversationalFormatter struct{}
// User: Hello
// Assistant: Hi there!

// 3. XML 格式（Anthropic 推荐）
type XMLFormatter struct{}
// <conversation>
//   <turn role="user">Hello</turn>
//   <turn role="assistant">Hi there!</turn>
// </conversation>

// 4. JSON 格式
type JSONFormatter struct{}
// [
//   {"role": "user", "content": "Hello"},
//   {"role": "assistant", "content": "Hi there!"}
// ]

// 5. Markdown 格式
type MarkdownFormatter struct{}
// **User:** Hello
//
// **Assistant:** Hi there!
```

---

## 四、可观测性工具

### 4.1 Prompt 调试器

```go
type PromptDebugger struct {
    enabled bool
    logger  Logger
}

func (d *PromptDebugger) LogPrompt(prompt *Prompt, metadata map[string]interface{}) {
    if !d.enabled {
        return
    }

    d.logger.Info("Prompt Built",
        "system_tokens", d.countTokens(prompt.System),
        "user_tokens", d.countTokens(prompt.User),
        "total_tokens", d.countTokens(prompt.String()),
        "tools_count", metadata["tools_count"],
        "history_turns", metadata["history_turns"],
        "few_shots_count", metadata["few_shots_count"],
    )

    if d.logger.IsDebug() {
        d.logger.Debug("Prompt Content",
            "system", prompt.System,
            "user", prompt.User,
        )
    }
}

// 使用示例
debugger := NewPromptDebugger(true, logger)
builder.WithDebugger(debugger)
prompt := builder.Build()
// 输出：
// [INFO] Prompt Built: system_tokens=150, user_tokens=80, total_tokens=230, tools_count=5, history_turns=3
```

### 4.2 Token 使用追踪

```go
type TokenTracker struct {
    systemTokens   int
    userTokens     int
    historyTokens  int
    toolsTokens    int
    fewShotsTokens int
    totalTokens    int
}

func (t *TokenTracker) Report() string {
    return fmt.Sprintf(`Token Usage:
  System Prompt: %d (%.1f%%)
  User Prompt: %d (%.1f%%)
  History: %d (%.1f%%)
  Tools: %d (%.1f%%)
  Few-Shots: %d (%.1f%%)
  Total: %d`,
        t.systemTokens, t.percentage(t.systemTokens),
        t.userTokens, t.percentage(t.userTokens),
        t.historyTokens, t.percentage(t.historyTokens),
        t.toolsTokens, t.percentage(t.toolsTokens),
        t.fewShotsTokens, t.percentage(t.fewShotsTokens),
        t.totalTokens,
    )
}
```

---

## 五、集成示例

### 5.1 在 Thinker 中使用

```go
type EnhancedThinker struct {
    llmClient      llm.Client
    promptBuilder  *FluentPromptBuilder
    contextWindow  *ContextWindow
    toolSelector   ToolSelector
    fewShotManager *FewShotManager
    debugger       *PromptDebugger
}

func (t *EnhancedThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
    // 1. 选择相关工具
    relevantTools := t.toolSelector.Select(task, t.allTools, 5)

    // 2. 选择 few-shot 示例
    fewShots := t.fewShotManager.Select(task, 3)

    // 3. 获取历史
    history := t.contextWindow.GetTurns()

    // 4. 构建 prompt
    prompt := t.promptBuilder.
        WithTask(task).
        WithTools(relevantTools).
        WithFewShots(fewShots).
        WithHistory(history).
        Build()

    // 5. 调试输出
    t.debugger.LogPrompt(prompt, map[string]interface{}{
        "tools_count": len(relevantTools),
        "few_shots_count": len(fewShots),
        "history_turns": len(history),
    })

    // 6. 调用 LLM
    response, err := t.llmClient.Generate(prompt.String())
    if err != nil {
        return nil, err
    }

    // 7. 更新上下文窗口
    t.contextWindow.Add(Turn{Role: "assistant", Content: response})

    // 8. 解析响应
    return t.parser.Parse(response)
}
```

### 5.2 在中间件中使用

```go
// PromptEnhancementMiddleware 增强 Prompt 的中间件
func PromptEnhancementMiddleware(
    fewShotManager *FewShotManager,
    toolSelector ToolSelector,
) ThinkMiddleware {
    return func(next ThinkHandler) ThinkHandler {
        return func(task string, ctx *core.Context) (*types.Thought, error) {
            // 注入 few-shot 示例
            fewShots := fewShotManager.Select(task, 3)
            ctx.Set("few_shots", fewShots)

            // 注入相关工具
            if allTools, ok := ctx.Get("all_tools").([]ToolDesc); ok {
                relevantTools := toolSelector.Select(task, allTools, 5)
                ctx.Set("tools", relevantTools)
            }

            return next(task, ctx)
        }
    }
}

// ContextCompressionMiddleware 自动压缩上下文的中间件
func ContextCompressionMiddleware(
    maxTokens int,
    counter TokenCounter,
    strategy CompressionStrategy,
) ThinkMiddleware {
    return func(next ThinkHandler) ThinkHandler {
        return func(task string, ctx *core.Context) (*types.Thought, error) {
            // 获取历史
            if history, ok := ctx.Get("history").([]Turn); ok {
                // 检查 token 数
                tokens := counter.CountMessages(history)
                if tokens > maxTokens {
                    // 压缩
                    compressed, err := strategy.Compress(history, maxTokens, counter)
                    if err != nil {
                        return nil, err
                    }
                    ctx.Set("history", compressed)
                    ctx.Set("compressed", true)
                    ctx.Set("original_tokens", tokens)
                    ctx.Set("compressed_tokens", counter.CountMessages(compressed))
                }
            }

            return next(task, ctx)
        }
    }
}
```

---

## 六、实现优先级

### Phase 1: 基础增强（立即实现）
1. ✅ FluentPromptBuilder - 流式 API
2. ✅ JSONSchemaFormatter - 标准工具格式
3. ✅ TikTokenCounter - 精确 token 计数
4. ✅ PriorityStrategy - 优先级压缩
5. ✅ PromptDebugger - 调试工具

### Phase 2: 智能功能（短期）
1. ⏳ FewShotManager - 示例管理
2. ⏳ KeywordSelector - 关键词工具选择
3. ⏳ ContextWindow - 窗口管理
4. ⏳ TokenTracker - token 追踪

### Phase 3: 高级功能（中期）
1. 🔮 SemanticSelector - 语义工具选择
2. 🔮 SummarizeStrategy - LLM 摘要压缩
3. 🔮 LLMAssistedSelector - LLM 辅助选择

---

## 七、配置示例

### 7.1 简单配置（开箱即用）

```go
// 使用默认配置
thinker := thinker.NewEnhancedThinker(llmClient, tools)
```

### 7.2 自定义配置

```go
// 完全自定义
thinker := thinker.NewEnhancedThinker(llmClient, tools,
    thinker.WithPromptBuilder(
        builder.New().
            WithToolFormatter(NewJSONSchemaFormatter()).
            WithMaxTokens(4096),
    ),
    thinker.WithContextWindow(
        NewContextWindow(4096,
            NewTikTokenCounter("cl100k_base"),
            NewPriorityStrategy(priorities),
        ),
    ),
    thinker.WithToolSelector(NewKeywordSelector(keywords)),
    thinker.WithFewShotManager(NewFewShotManager(examples)),
    thinker.WithDebugger(NewPromptDebugger(true, logger)),
)
```

### 7.3 中间件配置

```go
mt := thinker.NewMiddlewareThinker(baseThinker)
mt.Use(
    PromptEnhancementMiddleware(fewShotManager, toolSelector),
    ContextCompressionMiddleware(4096, counter, strategy),
    LoggingMiddleware(logger),
)
```

---

## 八、最佳实践

1. **Token 预算分配**
   - System Prompt: 20-30%
   - Tools: 20-30%
   - History: 30-40%
   - Few-Shots: 10-20%
   - 预留输出: 10-15%

2. **工具选择策略**
   - 简单任务：全部工具（< 10 个）
   - 复杂任务：关键词选择（10-50 个工具）
   - 大规模：语义选择（> 50 个工具）

3. **压缩策略选择**
   - 短对话（< 10 轮）：不压缩
   - 中等对话（10-50 轮）：滑动窗口
   - 长对话（> 50 轮）：优先级 + 摘要

4. **Few-Shot 使用**
   - 新任务类型：3-5 个示例
   - 熟悉任务：1-2 个示例
   - 简单任务：不使用

---

## 九、性能考虑

1. **Token 计数缓存**
   ```go
   type CachedTokenCounter struct {
       counter TokenCounter
       cache   *lru.Cache
   }
   ```

2. **工具选择缓存**
   ```go
   type CachedToolSelector struct {
       selector ToolSelector
       cache    map[string][]ToolDesc
       ttl      time.Duration
   }
   ```

3. **延迟加载**
   ```go
   // 只在需要时加载 tokenizer
   counter := NewLazyTokenCounter(func() TokenCounter {
       return NewTikTokenCounter("cl100k_base")
   })
   ```

---

## 十、测试策略

1. **单元测试**
   - 每个组件独立测试
   - Mock LLM 客户端

2. **集成测试**
   - 完整的 Thinker 流程
   - 真实 LLM 调用

3. **性能测试**
   - Token 计数准确性
   - 压缩效果
   - 工具选择准确性

4. **基准测试**
   ```go
   BenchmarkPromptBuilder
   BenchmarkTokenCounter
   BenchmarkCompression
   BenchmarkToolSelection
   ```

---

## 总结

这套工具箱提供了：
1. **完整性**：覆盖 Prompt 构建和上下文管理的所有场景
2. **灵活性**：从简单到复杂，按需使用
3. **可扩展性**：接口设计，易于扩展
4. **可观测性**：详细的调试和追踪
5. **中间件友好**：易于在中间件中使用

用户可以：
- 快速开始：使用默认配置
- 渐进增强：逐步添加高级功能
- 完全自定义：实现自己的策略
