# Engine Provider 集成指南

## 概述

Provider 已经完全集成到 Engine 的配置选项中，可以通过 `WithProvider` 或 `WithProviderRegistry` 选项轻松添加外部工具提供者。

## 集成方式

### 方式 1: 使用 WithProvider（推荐）

最简单的方式，直接注册单个 Provider：

```go
// 创建并配置 MCP Provider
mcpProvider := mcp.NewMCPProvider("my-service")
config := map[string]interface{}{
    "server_url": "http://localhost:8080",
    "api_key":    "your-key",
    "timeout":    30,
}
mcpProvider.Initialize(config)

// 创建 Engine 时直接集成
eng := engine.New(
    engine.WithLLMClient(llmClient),
    engine.WithProvider(mcpProvider), // 🎯 一行搞定
    engine.WithMaxIterations(10),
)

// Provider 的工具会自动发现和注册！
```

### 方式 2: 使用 WithProviderRegistry

适合管理多个 Provider：

```go
// 创建 Registry
registry := provider.NewRegistry()

// 注册多个 Provider
registry.Register(mcpProvider1)
registry.Register(mcpProvider2)
registry.Register(openAPIProvider)

// 创建 Engine
eng := engine.New(
    engine.WithLLMClient(llmClient),
    engine.WithProviderRegistry(registry),
    engine.WithMaxIterations(10),
)
```

### 方式 3: 混合使用

结合内置工具和 Provider 工具：

```go
eng := engine.New(
    engine.WithLLMClient(llmClient),
    engine.WithProvider(mcpProvider),    // 外部工具
    engine.WithMaxIterations(10),
)

// 额外注册内置工具
eng.RegisterTools(
    builtin.NewCalculator(),
    builtin.NewHTTP(),
    builtin.NewDateTime(),
)
```

## 工作原理

### 1. Engine 初始化流程

```
Engine.New()
    ↓
应用 Options (WithProvider)
    ↓
providerRegistry.Register(provider)
    ↓
providerRegistry.DiscoverAllTools()
    ↓
自动注册工具到 ToolManager
    ↓
更新 Thinker 的工具描述
    ↓
Engine 就绪
```

### 2. 自动工具发现

Engine 在初始化时会自动：
1. 从所有注册的 Provider 发现工具
2. 将工具注册到 ToolManager
3. 更新 Thinker 的工具列表
4. 无需手动干预

### 3. 工具调用流程

```
Task → Thinker → 选择工具 → Actor → Provider → 外部服务
                                    ↓
                                执行工具
                                    ↓
                                返回结果
```

## 完整示例

### 示例 1: 基础集成

```go
package main

import (
    "github.com/ray/goreact/pkg/engine"
    "github.com/ray/goreact/pkg/tool/provider/mcp"
)

func main() {
    // 1. 配置 MCP Provider
    mcpProvider := mcp.NewMCPProvider("weather-service")
    mcpProvider.Initialize(map[string]interface{}{
        "server_url": "http://localhost:8080",
    })

    // 2. 创建 Engine（自动集成工具）
    eng := engine.New(
        engine.WithProvider(mcpProvider),
    )

    // 3. 执行任务
    result := eng.Execute("What's the weather in Tokyo?", nil)
    fmt.Println(result.Output)
}
```

### 示例 2: 多 Provider 集成

```go
// 创建多个 Provider
mcpProvider := mcp.NewMCPProvider("mcp-service")
mcpProvider.Initialize(mcpConfig)

openAPIProvider := openapi.NewOpenAPIProvider("api-service")
openAPIProvider.Initialize(apiConfig)

// 一次性集成多个 Provider
eng := engine.New(
    engine.WithProvider(mcpProvider),
    engine.WithProvider(openAPIProvider),
    engine.WithMaxIterations(10),
)
```

### 示例 3: 与 Skills 结合

```go
// 同时使用 Skills 和 Providers
eng := engine.New(
    engine.WithLLMClient(llmClient),
    engine.WithSkillManager(skillManager),    // Skills
    engine.WithProvider(mcpProvider),         // External Tools
    engine.WithMaxIterations(10),
)
```

## Engine Options 完整列表

```go
// 核心组件
engine.WithThinker(thinker)
engine.WithActor(actor)
engine.WithObserver(observer)
engine.WithLoopController(controller)

// LLM 和缓存
engine.WithLLMClient(client)
engine.WithCache(cache)

// 工具系统
engine.WithProvider(provider)              // 🆕 单个 Provider
engine.WithProviderRegistry(registry)      // 🆕 Provider Registry

// Skills 系统
engine.WithSkillManager(skillManager)

// 配置
engine.WithMaxIterations(max)
engine.WithMaxRetries(max)
engine.WithRetryInterval(interval)
engine.WithMetrics(metrics)
```

## 优势

### 1. 简化配置

**之前**：
```go
// 手动创建 Registry
registry := provider.NewRegistry()
registry.Register(mcpProvider)

// 手动发现工具
tools, _ := registry.DiscoverAllTools()

// 手动注册工具
for _, tool := range tools {
    eng.RegisterTool(tool)
}
```

**现在**：
```go
// 一行搞定
eng := engine.New(
    engine.WithProvider(mcpProvider),
)
```

### 2. 自动化

- ✅ 自动发现工具
- ✅ 自动注册工具
- ✅ 自动更新工具描述
- ✅ 无需手动干预

### 3. 灵活性

- ✅ 支持单个 Provider
- ✅ 支持多个 Provider
- ✅ 支持动态添加 Provider
- ✅ 与现有工具系统兼容

### 4. 一致性

- ✅ 统一的配置接口
- ✅ 与其他 Options 风格一致
- ✅ 易于理解和使用

## 测试结果

```
=== Engine with MCP Provider Integration ===

✓ MCP Provider initialized
✓ Engine created with MCP tools automatically registered

Task: What's the weather in San Francisco?

=== Execution Result ===
Success: true
Output: The weather in San Francisco is 22°C and Sunny with 65% humidity.
Execution Time: 2.103291ms

✅ Integration test passed!
```

## 最佳实践

### 1. Provider 初始化

```go
// 先初始化 Provider
mcpProvider := mcp.NewMCPProvider("service")
if err := mcpProvider.Initialize(config); err != nil {
    log.Fatal(err)
}

// 再创建 Engine
eng := engine.New(
    engine.WithProvider(mcpProvider),
)
```

### 2. 错误处理

```go
mcpProvider := mcp.NewMCPProvider("service")
if err := mcpProvider.Initialize(config); err != nil {
    // 处理初始化错误
    log.Printf("Provider init failed: %v", err)
    // 可以选择不添加这个 Provider
    return
}

// 检查健康状态
if !mcpProvider.IsHealthy() {
    log.Println("Provider is not healthy")
}
```

### 3. 资源清理

```go
// Engine 使用完毕后清理 Provider
defer func() {
    if eng.providerRegistry != nil {
        eng.providerRegistry.Close()
    }
}()
```

### 4. 多环境配置

```go
func createEngine(env string) *engine.Engine {
    var mcpURL string
    switch env {
    case "dev":
        mcpURL = "http://localhost:8080"
    case "prod":
        mcpURL = "https://api.production.com"
    }

    mcpProvider := mcp.NewMCPProvider("service")
    mcpProvider.Initialize(map[string]interface{}{
        "server_url": mcpURL,
    })

    return engine.New(
        engine.WithProvider(mcpProvider),
    )
}
```

## 扩展示例

### 添加自定义 Provider

```go
// 1. 实现 Provider 接口
type MyProvider struct {
    // ...
}

func (p *MyProvider) Name() string { return "my-provider" }
func (p *MyProvider) Initialize(config map[string]interface{}) error { /* ... */ }
func (p *MyProvider) DiscoverTools() ([]tool.Tool, error) { /* ... */ }
// ... 其他方法

// 2. 使用自定义 Provider
myProvider := &MyProvider{}
myProvider.Initialize(config)

eng := engine.New(
    engine.WithProvider(myProvider),
)
```

## 总结

Provider 集成到 Engine Options 后：

✅ **更简单** - 一行代码完成集成
✅ **更自动** - 自动发现和注册工具
✅ **更灵活** - 支持多种集成方式
✅ **更一致** - 统一的配置风格
✅ **更强大** - 轻松扩展外部工具生态

现在 GoReAct 框架具备了完整的外部工具集成能力，可以轻松对接各种工具服务！🎉
