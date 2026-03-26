# Tool Provider System

通用的工具提供者（Provider）架构，用于集成外部工具系统。

## 概述

Provider 系统允许 GoReAct 框架集成各种外部工具源：

- **MCP (Model Context Protocol)**: 标准化的 AI 工具协议
- **LangChain Tools**: LangChain 生态系统的工具
- **OpenAI Functions**: OpenAI 函数调用
- **Custom Providers**: 自定义工具提供者

## 架构

```
┌─────────────────────────────────────────┐
│           GoReAct Engine                │
└─────────────────┬───────────────────────┘
                  │
         ┌────────▼────────┐
         │ Provider Registry│
         └────────┬────────┘
                  │
      ┌───────────┼───────────┐
      │           │           │
┌─────▼─────┐ ┌──▼──────┐ ┌──▼──────┐
│    MCP    │ │LangChain│ │ Custom  │
│ Provider  │ │Provider │ │Provider │
└───────────┘ └─────────┘ └─────────┘
      │           │           │
      │           │           │
   [Tools]     [Tools]     [Tools]
```

## Provider 接口

```go
type Provider interface {
    Name() string
    Initialize(config map[string]interface{}) error
    DiscoverTools() ([]tool.Tool, error)
    GetTool(name string) (tool.Tool, error)
    Close() error
    IsHealthy() bool
}
```

## 使用示例

### 1. 注册 MCP Provider

```go
import (
    "github.com/DotNetAge/goreact/pkg/tool/provider"
    "github.com/DotNetAge/goreact/pkg/tool/provider/mcp"
)

// 创建 Provider Registry
registry := provider.NewRegistry()

// 创建 MCP Provider
mcpProvider := mcp.NewMCPProvider("my-mcp-server")

// 初始化配置
config := map[string]interface{}{
    "server_url": "http://localhost:8080",
    "api_key":    "your-api-key",
    "timeout":    30,
}

if err := mcpProvider.Initialize(config); err != nil {
    log.Fatal(err)
}

// 注册到 Registry
registry.Register(mcpProvider)
```

### 2. 发现和使用工具

```go
// 从所有 Provider 发现工具
tools, err := registry.DiscoverAllTools()
if err != nil {
    log.Fatal(err)
}

// 注册到 Engine
for _, tool := range tools {
    engine.RegisterTool(tool)
}
```

### 3. 与 Engine 集成

```go
// 创建 Engine 时集成 Provider Registry
eng := engine.New(
    engine.WithProviderRegistry(registry),
    engine.WithMaxIterations(10),
)

// Engine 会自动从 Registry 发现和注册工具
```

## MCP Provider 详解

### MCP 协议

Model Context Protocol (MCP) 是一个标准化的协议，用于连接 AI 模型与外部工具和数据源。

### MCP Server 要求

MCP Server 需要实现以下端点：

1. **GET /health** - 健康检查
2. **GET /tools** - 列出可用工具
3. **POST /execute** - 执行工具

### 工具响应格式

```json
{
  "tools": [
    {
      "name": "weather",
      "description": "Get weather information",
      "schema": {
        "type": "object",
        "properties": {
          "location": {"type": "string"},
          "unit": {"type": "string"}
        }
      }
    }
  ]
}
```

### 执行请求格式

```json
{
  "tool": "weather",
  "params": {
    "location": "San Francisco",
    "unit": "celsius"
  }
}
```

### 执行响应格式

```json
{
  "success": true,
  "result": {
    "temperature": 18,
    "condition": "Sunny"
  }
}
```

## 创建自定义 Provider

```go
type MyCustomProvider struct {
    name string
    // ... 其他字段
}

func (p *MyCustomProvider) Name() string {
    return p.name
}

func (p *MyCustomProvider) Initialize(config map[string]interface{}) error {
    // 初始化逻辑
    return nil
}

func (p *MyCustomProvider) DiscoverTools() ([]tool.Tool, error) {
    // 发现工具逻辑
    return tools, nil
}

// ... 实现其他接口方法
```

## 最佳实践

1. **健康检查**: 定期检查 Provider 健康状态
2. **错误处理**: 优雅处理 Provider 不可用的情况
3. **缓存**: 缓存已发现的工具，避免重复请求
4. **超时**: 设置合理的超时时间
5. **认证**: 安全地管理 API 密钥和凭证

## 扩展性

Provider 系统设计为高度可扩展：

- ✅ 支持多个 Provider 同时运行
- ✅ 动态发现和注册工具
- ✅ 自动健康检查和故障转移
- ✅ 统一的工具接口
- ✅ 易于添加新的 Provider 类型
