# Prompt 工具箱使用指南

## 概述

这份指南不是 API 文档，而是**实战手册**。我们会通过真实场景告诉你：
- 遇到什么问题时，用什么工具
- 如何组合使用这些工具
- 常见陷阱和解决方案

## 核心理念

> 这些工具不是框架强制的，而是**可选的工具箱**。你可以：
> - 完全不用（使用默认实现）
> - 只用其中几个
> - 全部使用
> - 实现自己的版本

---

## 场景 1：工具太多，Prompt 太长

### 问题描述

你的系统有 50+ 个工具，如果全部放进 Prompt：
- Token 消耗巨大（可能占用 2000+ tokens）
- LLM 容易混淆，选错工具
- 成本高昂

### 解决方案：动态工具选择

**不要这样做：**
```go
// ❌ 把所有工具都塞进 Prompt
prompt := builder.New().
    WithTools(allTools).  // 50 个工具！
    Build()
```

**应该这样做：**
```go
// ✅ 根据任务动态选择相关工具
func selectRelevantTools(task string, allTools []ToolDesc) []ToolDesc {
    // 方案 1：关键词匹配（简单快速）
    keywords := extractKeywords(task)
    relevant := []ToolDesc{}
    for _, tool := range allTools {
        if matchesKeywords(tool, keywords) {
            relevant = append(relevant, tool)
        }
    }

    // 限制数量（最多 5-10 个）
    if len(relevant) > 10 {
        relevant = relevant[:10]
    }

    return relevant
}

// 使用
relevantTools := selectRelevantTools(task, allTools)
prompt := builder.New().
    WithTools(relevantTools).  // 只有 5-10 个相关工具
    Build()
```

**效果对比：**
- Token 消耗：2000+ → 200-400
- 成本降低：80%
- 准确率提升：LLM 更容易选对工具

---

## 场景 2：对话越来越长，超出上下文窗口

### 问题描述

用户和 AI 对话了 50 轮，历史记录占用 3000+ tokens，导致：
- 超出模型上下文限制
- 或者挤占了工具描述的空间

### 解决方案：智能压缩

**不要这样做：**
```go
// ❌ 简单截断，可能丢失重要信息
if len(history) > 10 {
    history = history[len(history)-10:]
}
```

**应该这样做：**
```go
// ✅ 使用优先级压缩策略
import "github.com/ray/goreact/pkg/prompt/compression"
import "github.com/ray/goreact/pkg/prompt/counter"

// 1. 创建 token 计数器
counter := counter.NewUniversalEstimator("mixed")

// 2. 创建优先级策略
strategy := compression.NewPriorityStrategy(map[string]int{
    "system":    100,  // 系统消息最重要
    "user":      80,   // 用户消息次之
    "assistant": 60,   // AI 回复可以适当丢弃
})

// 3. 设置保留规则
strategy.KeepRecent = 5           // 无论如何保留最近 5 轮
strategy.KeepSystemFirst = true   // 保留第一条系统消息

// 4. 压缩
maxTokens := 1000  // 历史记录最多 1000 tokens
compressed, _ := strategy.Compress(history, maxTokens, counter)

// 5. 使用压缩后的历史
prompt := builder.New().
    WithHistory(compressed).
    Build()
```

**效果对比：**
- 原始：50 轮，3000 tokens
- 压缩后：15 轮，950 tokens
- 保留了：系统消息 + 最近 5 轮 + 重要的用户消息

---

## 场景 3：不同 LLM 需要不同的工具格式

### 问题描述

- OpenAI GPT 喜欢 JSON Schema
- Anthropic Claude 喜欢 Markdown
- 本地小模型喜欢简单文本

### 解决方案：可切换的格式化器

```go
import "github.com/ray/goreact/pkg/prompt/formatter"

// 根据 LLM 类型选择格式化器
func getToolFormatter(llmType string) formatter.ToolFormatter {
    switch llmType {
    case "openai":
        return formatter.NewJSONSchemaFormatter(true)
    case "anthropic":
        return formatter.NewMarkdownFormatter()
    case "ollama":
        return formatter.NewSimpleTextFormatter()
    default:
        return formatter.NewCompactFormatter()  // 节省 tokens
    }
}

// 使用
toolFormatter := getToolFormatter("openai")
prompt := builder.New().
    WithTools(tools).
    WithToolFormatter(toolFormatter).
    Build()
```

**实际效果：**

**OpenAI (JSON Schema):**
```json
{
  "name": "calculator",
  "parameters": {
    "type": "object",
    "properties": {
      "operation": {"type": "string", "enum": ["add", "subtract"]}
    }
  }
}
```

**Anthropic (Markdown):**
```markdown
### calculator
Perform arithmetic operations

**Parameters:**
- `operation` (string): add, subtract, multiply, divide
```

**Ollama (Simple):**
```
1. calculator: Perform arithmetic operations
```

---

## 场景 4：Token 计数不准确，导致超限

### 问题描述

使用简单的 `len(text) / 4` 估算，结果：
- 中文文本严重低估
- 实际调用 LLM 时超出限制
- 浪费 API 调用

### 解决方案：精确的 Token 计数

```go
import "github.com/ray/goreact/pkg/prompt/counter"

// 不要这样做
// ❌ tokens := len(text) / 4

// 应该这样做
// ✅ 根据内容类型选择计数器
func getTokenCounter(text string) counter.TokenCounter {
    // 检测语言
    if containsChinese(text) {
        if containsEnglish(text) {
            return counter.NewUniversalEstimator("mixed")
        }
        return counter.NewUniversalEstimator("zh")
    }
    return counter.NewUniversalEstimator("en")
}

// 使用
tokenCounter := getTokenCounter(prompt)
tokens := tokenCounter.Count(prompt)

// 如果频繁计数，使用缓存版本
cachedCounter := counter.NewCachedTokenCounter(tokenCounter, 1000)
```

**准确度对比：**

| 文本 | 简单估算 | 精确估算 | 实际 |
|------|---------|---------|------|
| "Hello World" | 3 | 2 | 2 |
| "你好世界" | 3 | 8 | 8 |
| "Calculate 100+200 计算结果" | 8 | 18 | 17 |

---

## 场景 5：调试 Prompt 构建过程

### 问题描述

Prompt 构建是黑盒，不知道：
- 各部分占用多少 tokens
- 是否超出限制
- 哪里可以优化

### 解决方案：使用调试器

```go
import "github.com/ray/goreact/pkg/prompt/debug"

// 1. 创建调试器
logger := debug.NewSimpleLogger(true)  // true = 启用 debug 模式
debugger := debug.NewPromptDebugger(true, logger)

// 2. 构建 Prompt
prompt := builder.New().
    WithTask(task).
    WithTools(tools).
    WithHistory(history).
    Build()

// 3. 记录调试信息
debugger.LogPrompt(prompt, map[string]interface{}{
    "tools_count":     len(tools),
    "history_turns":   len(history),
    "token_counter":   tokenCounter,
})

// 4. 查看 Token 使用报告
tracker := debugger.GetTracker()
fmt.Println(tracker.Report())
```

**输出示例：**
```
[INFO] Prompt Built system_tokens=450 user_tokens=280 total_tokens=730 tools_count=5 history_turns=8
[DEBUG] Prompt Content system=You are a helpful... user=Task: Calculate...

Token Usage Report:
  System Prompt: 450 (61.6%)
  User Prompt: 280 (38.4%)
  History: 120 (16.4%)
  Tools: 200 (27.4%)
  Total: 730
```

**优化建议：**
- 如果 Tools 占比过高 → 减少工具数量或使用紧凑格式
- 如果 History 占比过高 → 使用压缩策略
- 如果 System 占比过高 → 简化系统提示词

---

## 场景 6：需要 Few-Shot 示例，但不知道选哪些

### 问题描述

你有 100 个 Few-Shot 示例，但：
- 全部放进去太占 tokens
- 随机选择效果不好
- 不知道哪些示例最相关

### 解决方案：智能示例选择

```go
// 方案 1：按类别选择（简单）
func selectExamplesByCategory(task string, allExamples []FewShotExample) []FewShotExample {
    category := detectCategory(task)  // "math", "text", "api" 等

    var relevant []FewShotExample
    for _, ex := range allExamples {
        if ex.Category == category {
            relevant = append(relevant, ex)
        }
    }

    // 最多 3 个
    if len(relevant) > 3 {
        relevant = relevant[:3]
    }

    return relevant
}

// 方案 2：按相似度选择（高级）
func selectExamplesBySimilarity(task string, allExamples []FewShotExample) []FewShotExample {
    // 计算任务和每个示例的相似度
    type scored struct {
        example FewShotExample
        score   float64
    }

    var scores []scored
    for _, ex := range allExamples {
        similarity := calculateSimilarity(task, ex.Task)
        scores = append(scores, scored{ex, similarity})
    }

    // 按相似度排序
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].score > scores[j].score
    })

    // 返回 Top-3
    var result []FewShotExample
    for i := 0; i < 3 && i < len(scores); i++ {
        result = append(result, scores[i].example)
    }

    return result
}

// 使用
examples := selectExamplesByCategory(task, allExamples)
prompt := builder.New().
    WithTask(task).
    WithFewShots(examples).
    Build()
```

---

## 场景 7：在中间件中使用这些工具

### 问题描述

你想在中间件中自动优化 Prompt，但不知道如何集成。

### 解决方案：创建 Prompt 优化中间件

```go
// PromptOptimizationMiddleware 自动优化 Prompt 的中间件
func PromptOptimizationMiddleware(
    allTools []ToolDesc,
    maxHistoryTokens int,
) thinker.ThinkMiddleware {
    counter := counter.NewUniversalEstimator("mixed")
    compressor := compression.NewPriorityStrategy(nil)

    return func(next thinker.ThinkHandler) thinker.ThinkHandler {
        return func(task string, ctx *core.Context) (*types.Thought, error) {
            // 1. 动态选择工具
            relevantTools := selectRelevantTools(task, allTools)
            ctx.Set("tools", relevantTools)

            // 2. 压缩历史
            if history, ok := ctx.Get("history").([]Turn); ok {
                compressed, _ := compressor.Compress(history, maxHistoryTokens, counter)
                ctx.Set("history", compressed)
            }

            // 3. 记录优化信息
            ctx.Set("tools_count", len(relevantTools))
            ctx.Set("optimization_applied", true)

            return next(task, ctx)
        }
    }
}

// 使用
mt := thinker.NewMiddlewareThinker(baseThinker)
mt.Use(
    PromptOptimizationMiddleware(allTools, 1000),
    LoggingMiddleware(logger),
)
```

---

## 最佳实践总结

### 1. Token 预算管理

```go
// 为不同部分设置 token 预算
const (
    SystemPromptBudget = 500   // 20%
    ToolsBudget        = 800   // 30%
    HistoryBudget      = 1000  // 40%
    FewShotsBudget     = 250   // 10%
    TotalBudget        = 2550  // 留 10% 给输出
)

// 构建时检查预算
func buildWithBudget(task string, tools []ToolDesc, history []Turn) *Prompt {
    counter := counter.NewUniversalEstimator("mixed")

    // 1. 压缩历史到预算内
    compressor := compression.NewPriorityStrategy(nil)
    history, _ = compressor.Compress(history, HistoryBudget, counter)

    // 2. 选择工具到预算内
    tools = selectToolsWithBudget(tools, ToolsBudget, counter)

    // 3. 构建
    return builder.New().
        WithTask(task).
        WithTools(tools).
        WithHistory(history).
        Build()
}
```

### 2. 渐进式优化

```go
// 第一步：使用默认实现
prompt := builder.New().WithTask(task).Build()

// 第二步：添加工具格式化
prompt := builder.New().
    WithTask(task).
    WithToolFormatter(formatter.NewJSONSchemaFormatter(true)).
    Build()

// 第三步：添加 token 计数
prompt := builder.New().
    WithTask(task).
    WithToolFormatter(formatter.NewJSONSchemaFormatter(true)).
    WithTokenCounter(counter.NewUniversalEstimator("mixed")).
    Build()

// 第四步：添加调试
debugger := debug.NewPromptDebugger(true, logger)
prompt := builder.New().
    WithTask(task).
    WithToolFormatter(formatter.NewJSONSchemaFormatter(true)).
    WithTokenCounter(counter.NewUniversalEstimator("mixed")).
    Build()
debugger.LogPrompt(prompt, metadata)
```

### 3. 性能优化

```go
// 使用缓存避免重复计算
var (
    cachedCounter = counter.NewCachedTokenCounter(
        counter.NewUniversalEstimator("mixed"),
        1000,
    )

    cachedFormatter = &CachedFormatter{
        formatter: formatter.NewJSONSchemaFormatter(true),
        cache:     make(map[string]string),
    }
)

// 定期清理缓存
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cachedCounter.Clear()
    }
}()
```

---

## 常见陷阱

### ❌ 陷阱 1：过度优化

```go
// 不要为了节省 10 个 tokens 而牺牲可读性
// ❌ 使用极度压缩的格式
formatter := formatter.NewCompactFormatter()

// ✅ 在可读性和 token 之间平衡
formatter := formatter.NewJSONSchemaFormatter(false)  // 不缩进
```

### ❌ 陷阱 2：忽略语言特性

```go
// ❌ 对中文文本使用英文计数器
counter := counter.NewUniversalEstimator("en")
tokens := counter.Count("这是一段中文文本")  // 严重低估

// ✅ 自动检测或明确指定
counter := counter.NewUniversalEstimator("mixed")
```

### ❌ 陷阱 3：压缩过度

```go
// ❌ 压缩到只剩 1 轮对话
strategy := compression.NewSlidingWindowStrategy(1)

// ✅ 保留足够的上下文
strategy := compression.NewPriorityStrategy(nil)
strategy.KeepRecent = 5  // 至少保留 5 轮
```

---

## 何时不需要这些工具

1. **工具数量 < 10**：直接全部放进 Prompt
2. **对话轮次 < 10**：不需要压缩
3. **只用英文**：简单估算就够了
4. **原型阶段**：先跑通流程，再优化

---

## 总结

这些工具的价值在于：
- **解决真实问题**：token 超限、成本过高、准确率低
- **渐进式采用**：从简单到复杂，按需使用
- **可组合**：工具之间可以自由组合
- **可扩展**：可以实现自己的版本

记住：**工具是手段，不是目的。先让系统跑起来，再根据实际问题选择合适的工具。**
