# Thinker 环节完整指南

## 什么是 Thinker？

Thinker 是 ReAct 循环中的"思考"环节，负责：
1. 接收任务和上下文
2. 调用 LLM 进行推理
3. 解析 LLM 响应
4. 返回 Thought（包含推理过程、要执行的动作或最终答案）

```
Task + Context → Thinker → Thought (Reasoning + Action/FinalAnswer)
```

---

## 三种使用方式

### 1. 使用预设 Thinker（最简单）

适合快速开始，开箱即用。

```go
import (
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/core/thinker/presets"
)

// SimpleThinker - 最简单的实现
simpleThinker := thinker.NewSimpleThinker(llmClient, toolDescriptions)

// ReActThinker - 标准 ReAct 模式（推荐）
reactThinker := presets.NewReActThinker(llmClient, toolDescriptions)

// 使用
engine := engine.New(engine.WithThinker(reactThinker))
```

**预设 Thinker 对比**：

| Thinker | 特点 | 适用场景 |
|---------|------|----------|
| SimpleThinker | 最简单，无依赖 | 快速测试、学习 |
| ReActThinker | 标准 ReAct，组件化 | 生产环境推荐 |
| ConversationalThinker | 对话式（待实现） | 聊天机器人 |
| PlanningThinker | 规划式（待实现） | 复杂任务拆解 |

---

### 2. 使用组件组装自定义 Thinker

适合有特定需求，需要灵活组合。

```go
import (
	"github.com/ray/goreact/pkg/core/thinker/components"
)

// 1. 选择 Prompt 模板
promptBuilder := components.NewPromptBuilder(components.ReActTemplate)

// 或使用自定义模板
customTemplate := &thinker.PromptTemplate{
	System: "You are a helpful assistant...",
	User:   "Task: {{.task}}\n\nTools: {{.tools}}",
}
promptBuilder := components.NewPromptBuilder(customTemplate)

// 2. 配置上下文管理器
contextManager := components.NewContextManager(8192) // 8k tokens

// 3. 选择响应解析器
parser := components.NewReActParser() // ReAct 格式
// 或 parser := components.NewJSONParser() // JSON 格式

// 4. 组装自定义 Thinker
type MyThinker struct {
	llmClient      llm.Client
	promptBuilder  *components.DefaultPromptBuilder
	contextManager *components.DefaultContextManager
	parser         components.ResponseParser
}

func (t *MyThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
	// 1. 构建 prompt
	prompt := t.promptBuilder.Build(task, ctx, tools)

	// 2. 调用 LLM
	response, err := t.llmClient.Generate(prompt.String())
	if err != nil {
		return nil, err
	}

	// 3. 解析响应
	thought, err := t.parser.Parse(response)
	if err != nil {
		return nil, err
	}

	// 4. 更新上下文
	t.contextManager.AddTurn(&thinker.Turn{
		Role:    "assistant",
		Content: response,
	})

	return thought, nil
}
```

---

### 3. 完全自定义 Thinker

适合有特殊需求，需要完全控制。

```go
type MyCustomThinker struct {
	// 你的字段
}

// 实现 Thinker 接口
func (t *MyCustomThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
	// 完全自定义的逻辑
	// 1. 你可以不调用 LLM（使用规则引擎）
	// 2. 你可以调用多个 LLM 并投票
	// 3. 你可以使用本地模型
	// 4. 任何你想要的逻辑

	return &types.Thought{
		Reasoning:    "...",
		Action:       &types.Action{...},
		ShouldFinish: false,
	}, nil
}

engine := engine.New(engine.WithThinker(&MyCustomThinker{}))
```

---

## 组件详解

### PromptBuilder - 提示词构建器

**作用**：将任务、工具描述、历史对话组装成 LLM prompt。

**内置模板**：

```go
// 1. ReActTemplate - 标准 ReAct 模式
components.ReActTemplate

// 2. ConversationalTemplate - 对话式
components.ConversationalTemplate

// 3. PlanningTemplate - 规划式
components.PlanningTemplate

// 4. MinimalTemplate - 最简模板
components.MinimalTemplate
```

**自定义模板**：

```go
myTemplate := &thinker.PromptTemplate{
	System: `You are {{.role}}.
Available tools: {{.tools}}`,

	User: `Task: {{.task}}
{{if .history}}History: {{.history}}{{end}}
Respond with your reasoning and action.`,
}

builder := components.NewPromptBuilder(myTemplate)

// 添加自定义变量
builder.AddVariable("role", "a helpful assistant")
```

**模板变量**：
- `{{.task}}` - 当前任务
- `{{.tools}}` - 工具描述
- `{{.history}}` - 历史对话
- 自定义变量通过 `AddVariable()` 添加

---

### ContextManager - 上下文管理器

**作用**：管理多轮对话历史，支持压缩和 token 估算。

```go
manager := components.NewContextManager(8192) // 最大 8k tokens

// 添加对话轮次
manager.AddTurn(&thinker.Turn{
	Role:    "user",
	Content: "Calculate 10 + 5",
})

manager.AddTurn(&thinker.Turn{
	Role:    "assistant",
	Content: "The result is 15",
})

// 获取历史（最近 N 轮）
history := manager.GetHistory(5)

// 压缩上下文
manager.Compress(thinker.StrategyTruncate)       // 截断最早的
manager.Compress(thinker.StrategySlidingWindow)  // 滑动窗口

// 估算 token 数
tokens := manager.EstimateTokens()
```

**压缩策略**：

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| Truncate | 移除最早的 25% | 简单场景 |
| SlidingWindow | 保留最近 N 轮 | 对话场景 |
| Summarize | 摘要（需要 LLM） | 长对话场景 |

---

### ResponseParser - 响应解析器

**作用**：将 LLM 的文本响应解析为结构化的 Thought。

```go
// ReAct 格式解析器
parser := components.NewReActParser()

response := `
Thought: I need to calculate
Action: calculator
Parameters: {"operation": "add", "a": 10, "b": 5}
Reasoning: Use calculator to add numbers
`

thought, err := parser.Parse(response)
// thought.Action.ToolName == "calculator"
// thought.Action.Parameters == {"operation": "add", "a": 10, "b": 5}
```

**支持的格式**：

1. **ReAct 格式**（推荐）：
```
Thought: [推理过程]
Action: [工具名]
Parameters: {"key": "value"}
Reasoning: [为什么用这个工具]
```

2. **Final Answer 格式**：
```
Thought: [推理过程]
Final Answer: [最终答案]
```

3. **JSON 格式**（待实现）：
```json
{
  "thought": "...",
  "action": "calculator",
  "parameters": {...}
}
```

---

## Middleware 增强

Thinker 可以通过 Middleware 增强功能。详见 [Middleware 指南](./MIDDLEWARE_GUIDE.md)。

```go
// 创建基础 Thinker
baseThinker := presets.NewReActThinker(llmClient, tools)

// 包装为 MiddlewareThinker
enhancedThinker := thinker.NewMiddlewareThinker(baseThinker)

// 添加中间件
enhancedThinker.Use(middlewares.LoggingMiddleware(nil))
enhancedThinker.Use(middlewares.RAGMiddleware(retriever, 3))
enhancedThinker.Use(middlewares.IntentCacheMiddleware(cache, 0.9))
enhancedThinker.Use(middlewares.ConfidenceMiddleware(0.7, nil))

// 使用增强后的 Thinker
engine := engine.New(engine.WithThinker(enhancedThinker))
```

---

## 最佳实践

### 1. 选择合适的 Thinker

- **快速原型**：使用 `SimpleThinker`
- **生产环境**：使用 `ReActThinker` + Middleware
- **特殊需求**：组装自定义 Thinker

### 2. Prompt 模板设计

**好的 Prompt**：
- ✅ 清晰的角色定义
- ✅ 明确的输出格式
- ✅ 提供示例
- ✅ 工具描述详细

**避免**：
- ❌ 过于冗长
- ❌ 模糊的指令
- ❌ 缺少格式说明

### 3. 上下文管理

```go
// 设置合理的 token 限制
contextManager := components.NewContextManager(4096)

// 定期压缩
if contextManager.EstimateTokens() > 3000 {
	contextManager.Compress(thinker.StrategySlidingWindow)
}
```

### 4. 错误处理

```go
thought, err := thinker.Think(task, ctx)
if err != nil {
	// 1. 记录错误
	log.Printf("Think failed: %v", err)

	// 2. 降级处理
	return fallbackThought, nil
}

// 3. 检查置信度
if confidence, ok := thought.Metadata["confidence"]; ok {
	if confidence.(float64) < 0.7 {
		// 需要澄清
	}
}
```

---

## 示例代码

完整示例请查看：
- [examples/basic/](../examples/basic/) - 基础使用
- [examples/custom_thinker/](../examples/custom_thinker/) - 自定义 Thinker
- [examples/thinker_middleware/](../examples/thinker_middleware/) - Middleware 使用

---

## 下一步

- [Middleware 指南](./MIDDLEWARE_GUIDE.md) - 学习如何增强 Thinker
- [架构文档](../ARCHITECTURE.md) - 理解整体设计
- [API 文档](./API.md) - 查看完整 API
