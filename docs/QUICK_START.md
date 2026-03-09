# GoReAct 快速开始

## 5 分钟上手

### 1. 安装

```bash
go get github.com/ray/goreact
```

### 2. 最简单的例子

```go
package main

import (
	"fmt"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/llm/mock"
	"github.com/ray/goreact/pkg/tool/builtin"
	"github.com/ray/goreact/pkg/core"
)

func main() {
	// 1. 创建 LLM 客户端（这里用 mock，实际使用 Ollama/OpenAI）
	llmClient := mock.NewMockClient([]string{
		"Thought: I need to calculate\nAction: calculator\nParameters: {\"operation\": \"add\", \"a\": 10, \"b\": 5}\nReasoning: Use calculator",
		"Thought: Got result\nFinal Answer: 15",
	})

	// 2. 创建 Engine
	eng := engine.New(
		engine.WithLLMClient(llmClient),
	)

	// 3. 注册工具
	eng.RegisterTool(builtin.NewCalculator())

	// 4. 执行任务
	result := eng.Execute("Calculate 10 + 5", core.NewContext())

	// 5. 查看结果
	fmt.Printf("结果: %s\n", result.Output)
	fmt.Printf("成功: %v\n", result.Success)
}
```

### 3. 使用真实 LLM（Ollama）

```go
import (
	"github.com/ray/goreact/pkg/llm/ollama"
)

func main() {
	// 使用 Ollama
	llmClient := ollama.NewOllamaClient(
		ollama.WithModel("qwen2.5:0.5b"),
		ollama.WithBaseURL("http://localhost:11434"),
	)

	eng := engine.New(
		engine.WithLLMClient(llmClient),
	)

	// ... 其他代码相同
}
```

---

## 核心概念

### ReAct 循环

```
Think (思考) → Act (行动) → Observe (观察) → Loop Control (循环控制)
     ↑                                                    ↓
     └────────────────────────────────────────────────────┘
```

### Engine 是什么？

Engine 是 ReAct 循环的驱动器，负责：
- 驱动 Think → Act → Observe 循环
- 管理 Tool（工具）
- 处理重试和降级
- 缓存和指标收集

### Thinker 是什么？

Thinker 负责"思考"环节：
- 接收任务和上下文
- 调用 LLM 进行推理
- 解析 LLM 响应
- 返回 Thought（包含 Action 或 FinalAnswer）

---

## 进阶使用

### 使用 Middleware 增强 Thinker

```go
import (
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/core/thinker/middlewares"
	"github.com/ray/goreact/pkg/core/thinker/presets"
)

func main() {
	// 创建基础 Thinker
	baseThinker := presets.NewReActThinker(llmClient, toolDescriptions)

	// 创建支持中间件的 Thinker
	enhancedThinker := thinker.NewMiddlewareThinker(baseThinker)

	// 添加中间件
	enhancedThinker.Use(middlewares.LoggingMiddleware(nil))           // 日志
	enhancedThinker.Use(middlewares.RetryMiddleware(3, time.Second))  // 重试
	enhancedThinker.Use(middlewares.IntentCacheMiddleware(cache, 0.9)) // 缓存

	// 使用增强的 Thinker
	eng := engine.New(engine.WithThinker(enhancedThinker))
}
```

### 自定义 Thinker

```go
type MyThinker struct {
	llmClient llm.Client
}

func (t *MyThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
	// 你的自定义逻辑
	response, err := t.llmClient.Generate("Your custom prompt: " + task)
	// ... 解析响应
	return thought, nil
}

// 使用
eng := engine.New(engine.WithThinker(&MyThinker{llmClient: llmClient}))
```

---

## 内置工具

GoReAct 提供了丰富的内置工具：

```go
import "github.com/ray/goreact/pkg/tool/builtin"

// 数学计算
eng.RegisterTool(builtin.NewCalculator())

// HTTP 请求
eng.RegisterTool(builtin.NewHTTP())

// 日期时间
eng.RegisterTool(builtin.NewDateTime())

// 文件系统
eng.RegisterTool(builtin.NewFilesystem())

// Bash 命令
eng.RegisterTool(builtin.NewBash())

// 文本搜索
eng.RegisterTool(builtin.NewGrep())
```

---

## 下一步

- [Thinker 完整指南](./THINKER_GUIDE.md) - 深入了解 Thinker 环节
- [Middleware 指南](./MIDDLEWARE_GUIDE.md) - 学习如何使用和编写中间件
- [示例代码](../examples/) - 查看更多实际例子
- [架构文档](../ARCHITECTURE.md) - 理解框架设计

---

## 常见问题

### Q: 如何切换不同的 LLM？

只需要更换 LLM 客户端：

```go
// Ollama
llmClient := ollama.NewOllamaClient(...)

// OpenAI
llmClient := openai.NewOpenAIClient(apiKey, "gpt-4")

// Anthropic
llmClient := anthropic.NewAnthropicClient(apiKey, "claude-3-opus")
```

### Q: 如何添加自定义工具？

实现 `tool.Tool` 接口：

```go
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "My custom tool" }
func (t *MyTool) Execute(params map[string]interface{}) (interface{}, error) {
	// 你的逻辑
	return result, nil
}

eng.RegisterTool(&MyTool{})
```

### Q: 如何调试？

使用日志中间件：

```go
enhancedThinker.Use(middlewares.LoggingMiddleware(nil))
```

或者查看执行轨迹：

```go
result := eng.Execute(task, ctx)
for _, step := range result.Trace {
	fmt.Printf("[%s] %s\n", step.Type, step.Content)
}
```
