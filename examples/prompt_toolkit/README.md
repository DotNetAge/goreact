# Prompt Toolkit Example

演示 GoReAct 的 Prompt 构建和上下文管理工具箱。

## 功能展示

### 1. 工具格式化器
- **SimpleTextFormatter** - 简单文本格式
- **JSONSchemaFormatter** - JSON Schema 格式（推荐）
- **MarkdownFormatter** - Markdown 格式（可读性好）
- **CompactFormatter** - 紧凑格式（节省 tokens）

### 2. Token 计数器
- **SimpleEstimator** - 简单估算（1 token ≈ 4 chars）
- **UniversalEstimator** - 通用估算（支持中英文混合）
- **CachedTokenCounter** - 带缓存的计数器

### 3. FluentPromptBuilder
流式 API 构建 Prompt：
```go
prompt := builder.New().
    WithTask(task).
    WithTools(tools).
    WithHistory(history).
    WithFewShots(examples).
    WithToolFormatter(jsonFormatter).
    WithTokenCounter(counter).
    Build()
```

### 4. 上下文压缩策略
- **TruncateStrategy** - 截断最早的消息
- **SlidingWindowStrategy** - 滑动窗口（保留最近 N 轮）
- **PriorityStrategy** - 优先级压缩（保留重要消息）
- **HybridStrategy** - 混合策略

### 5. 调试工具
- **PromptDebugger** - 记录 Prompt 构建信息
- **TokenTracker** - 追踪 token 使用情况

## 运行

```bash
go run main.go
```

## 示例输出

```
=== Prompt Toolkit 示例 ===

--- 工具格式化器对比 ---

1. Simple Text Format:
1. calculator: Perform arithmetic operations
2. http: Make HTTP requests

2. JSON Schema Format:
[
  {
    "name": "calculator",
    "description": "Perform arithmetic operations",
    "parameters": {
      "type": "object",
      "properties": {
        "operation": {
          "type": "string",
          "description": "The operation to perform",
          "enum": ["add", "subtract", "multiply", "divide"]
        },
        ...
      }
    }
  }
]

3. Markdown Format:
## Available Tools

### calculator
Perform arithmetic operations

**Parameters:**
- `operation` (string) (required): The operation to perform - Options: [add subtract multiply divide]
- `a` (number) (required): First operand
- `b` (number) (required): Second operand

--- Token 计数器对比 ---

Simple Estimator: 28 tokens
Universal Estimator (mixed): 45 tokens
Universal Estimator (en): 32 tokens
Universal Estimator (zh): 52 tokens

--- FluentPromptBuilder 示例 ---

[INFO] Prompt Built system_tokens=150 user_tokens=80 total_tokens=230 tools_count=2 history_turns=3 few_shots_count=1
[DEBUG] Prompt Content system=You are a helpful AI assistant... user=Task: Calculate 100 + 200...
[INFO] Prompt Build Time duration_ms=2

--- 上下文压缩示例 ---

原始历史: 11 轮, 180 tokens
截断策略: 8 轮, 135 tokens (移除了 27.3%)
滑动窗口策略: 5 轮, 82 tokens (保留最近 5 轮)
优先级策略: 7 轮, 145 tokens (保留重要消息)

优先级策略保留的消息:
  1. [system]: You are a helpful assistant
  2. [user]: I need to calculate something
  3. [user]: Calculate 10 + 5
  4. [assistant]: The result is 15
  5. [user]: Now calculate 20 * 3
  6. [assistant]: The result is 60

--- Token 使用报告 ---

Token Usage Report:
  System Prompt: 150 (65.2%)
  User Prompt: 80 (34.8%)
  History: 0 (0.0%)
  Tools: 0 (0.0%)
  Few-Shots: 0 (0.0%)
  Total: 230
```

## 核心概念

### Token 预算管理

推荐的 token 分配比例：
- System Prompt: 20-30%
- Tools: 20-30%
- History: 30-40%
- Few-Shots: 10-20%
- 预留输出: 10-15%

### 压缩策略选择

| 对话长度 | 推荐策略 |
|---------|---------|
| < 10 轮 | 不压缩 |
| 10-50 轮 | 滑动窗口 |
| > 50 轮 | 优先级 + 滑动窗口 |

### 工具格式选择

| LLM 类型 | 推荐格式 |
|---------|---------|
| OpenAI GPT | JSON Schema |
| Anthropic Claude | Markdown |
| 本地模型 (Ollama) | Simple Text |

## 集成到 Thinker

```go
type EnhancedThinker struct {
    llmClient     llm.Client
    promptBuilder *builder.FluentPromptBuilder
    debugger      *debug.PromptDebugger
}

func (t *EnhancedThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
    // 构建 Prompt
    prompt := t.promptBuilder.
        WithTask(task).
        WithTools(t.tools).
        WithHistory(t.history).
        Build()

    // 调试输出
    t.debugger.LogPrompt(prompt, metadata)

    // 调用 LLM
    response, err := t.llmClient.Generate(prompt.String())
    ...
}
```

## 相关文档

- [PROMPT_TOOLKIT_DESIGN.md](../../docs/PROMPT_TOOLKIT_DESIGN.md) - 完整设计文档
- [THINKER_GUIDE.md](../../docs/THINKER_GUIDE.md) - Thinker 开发指南
- [MIDDLEWARE_GUIDE.md](../../docs/MIDDLEWARE_GUIDE.md) - 中间件系统指南
