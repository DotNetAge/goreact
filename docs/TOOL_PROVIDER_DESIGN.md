# Tool Provider 架构设计文档

## 概述

Tool Provider 是 GoReAct 框架的扩展点机制，用于集成外部工具系统。通过这个架构，框架可以支持各种外部工具协议，如 MCP、LangChain Tools、OpenAPI 等。

## 设计目标

1. **可扩展性** - 轻松添加新的工具提供者
2. **解耦** - 工具提供者独立于核心引擎
3. **动态性** - 运行时发现和加载工具
4. **标准化** - 统一的工具接口
5. **兼容性** - 与现有内置工具完全兼容

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      GoReAct Engine                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Thinker    │  │    Actor     │  │   Observer   │      │
│  └──────────────┘  └──────┬───────┘  └──────────────┘      │
└────────────────────────────┼──────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │  Tool Manager   │
                    │  - 工具注册      │
                    │  - 工具路由      │
                    │  - 工具执行      │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼────────┐ ┌──▼──────────┐ ┌─▼──────────┐
     │ Provider Registry│ │   Builtin   │ │   Direct   │
     │  - 多Provider管理 │ │    Tools    │ │Registration│
     └────────┬────────┘ └─────────────┘ └────────────┘
              │
    ┌─────────┼─────────┬─────────┬─────────┐
    │         │         │         │         │
┌───▼───┐ ┌──▼───┐ ┌───▼───┐ ┌───▼───┐ ┌───▼───┐
│  MCP  │ │OpenAPI│ │LangCh.│ │Custom │ │ ...   │
│Provider│ │Provider│ │Provider│ │Provider│ │       │
└───┬───┘ └──┬───┘ └───┬───┘ └───┬───┘ └───────┘
    │        │         │         │
    ▼        ▼         ▼         ▼
[External] [REST]  [Python]  [Your Protocol]
[MCP Srv]  [APIs]  [Tools]
```

## 核心组件

### 1. Provider 接口

```go
type Provider interface {
    // Name 返回提供者名称
    Name() string

    // Initialize 初始化提供者
    Initialize(config map[string]interface{}) error

    // DiscoverTools 发现可用工具
    DiscoverTools() ([]tool.Tool, error)

    // GetTool 获取指定工具
    GetTool(name string) (tool.Tool, error)

    // Close 关闭提供者
    Close() error

    // IsHealthy 健康检查
    IsHealthy() bool
}
```

### 2. Provider Registry

Provider Registry 负责管理多个 Provider：

```go
type Registry struct {
    providers map[string]Provider
}

// 功能：
// - 注册/注销 Provider
// - 从所有 Provider 发现工具
// - 健康检查和故障转移
// - 统一的生命周期管理
```

### 3. Tool 接口（保持不变）

```go
type Tool interface {
    Name() string
    Description() string
    Execute(params map[string]interface{}) (interface{}, error)
}
```

## 已实现的 Provider

### 1. MCP Provider

**Model Context Protocol** - 标准化的 AI 工具协议

**特性**：
- HTTP/REST 协议
- 标准化的工具描述格式
- 支持认证（API Key）
- 健康检查
- 超时控制

**配置示例**：
```go
config := map[string]interface{}{
    "server_url": "http://localhost:8080",
    "api_key":    "your-api-key",
    "timeout":    30,
}
```

**MCP Server 端点**：
- `GET /health` - 健康检查
- `GET /tools` - 列出工具
- `POST /execute` - 执行工具

### 2. Builtin Provider（计划中）

包装现有的内置工具（Calculator, HTTP, DateTime, Echo）

### 3. OpenAPI Provider（计划中）

从 OpenAPI/Swagger 规范自动生成工具

### 4. LangChain Provider（计划中）

集成 LangChain 生态系统的工具

## 使用流程

### 1. 创建和配置 Provider

```go
// 创建 MCP Provider
mcpProvider := mcp.NewMCPProvider("my-service")

// 配置
config := map[string]interface{}{
    "server_url": "http://localhost:8080",
    "api_key":    "secret",
}

// 初始化
mcpProvider.Initialize(config)
```

### 2. 注册到 Registry

```go
registry := provider.NewRegistry()
registry.Register(mcpProvider)
```

### 3. 发现工具

```go
// 从单个 Provider 发现
tools, _ := mcpProvider.DiscoverTools()

// 从所有 Provider 发现
allTools, _ := registry.DiscoverAllTools()
```

### 4. 注册到 Engine

```go
eng := engine.New()

// 方式 1: 手动注册
for _, tool := range tools {
    eng.RegisterTool(tool)
}

// 方式 2: 通过 Registry（推荐）
eng.RegisterToolProvider(registry)
```

## 扩展指南

### 创建自定义 Provider

```go
type MyProvider struct {
    name   string
    config map[string]interface{}
    tools  map[string]tool.Tool
}

func (p *MyProvider) Name() string {
    return p.name
}

func (p *MyProvider) Initialize(config map[string]interface{}) error {
    p.config = config
    // 初始化逻辑：连接、认证等
    return nil
}

func (p *MyProvider) DiscoverTools() ([]tool.Tool, error) {
    // 发现工具逻辑
    tools := []tool.Tool{}

    // 从你的系统获取工具列表
    // 创建 Tool 包装器

    return tools, nil
}

func (p *MyProvider) GetTool(name string) (tool.Tool, error) {
    tool, exists := p.tools[name]
    if !exists {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    return tool, nil
}

func (p *MyProvider) Close() error {
    // 清理资源
    return nil
}

func (p *MyProvider) IsHealthy() bool {
    // 健康检查逻辑
    return true
}
```

### Tool 包装器模式

如果外部工具的接口与 GoReAct 的 Tool 接口不同，使用适配器模式：

```go
type ExternalToolAdapter struct {
    name        string
    description string
    externalTool ExternalTool // 外部工具
    provider    *MyProvider
}

func (t *ExternalToolAdapter) Name() string {
    return t.name
}

func (t *ExternalToolAdapter) Description() string {
    return t.description
}

func (t *ExternalToolAdapter) Execute(params map[string]interface{}) (interface{}, error) {
    // 转换参数格式
    externalParams := convertParams(params)

    // 调用外部工具
    result, err := t.externalTool.Call(externalParams)
    if err != nil {
        return nil, err
    }

    // 转换返回值格式
    return convertResult(result), nil
}
```

## 最佳实践

### 1. 错误处理

```go
func (p *MyProvider) DiscoverTools() ([]tool.Tool, error) {
    // 使用超时
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 优雅处理错误
    tools, err := p.fetchTools(ctx)
    if err != nil {
        // 记录日志
        log.Printf("Failed to discover tools: %v", err)
        // 返回空列表而不是失败
        return []tool.Tool{}, nil
    }

    return tools, nil
}
```

### 2. 健康检查

```go
func (p *MyProvider) IsHealthy() bool {
    // 定期 ping
    if time.Since(p.lastHealthCheck) > 1*time.Minute {
        p.healthy = p.ping()
        p.lastHealthCheck = time.Now()
    }
    return p.healthy
}
```

### 3. 缓存

```go
type MyProvider struct {
    toolsCache      []tool.Tool
    cacheExpiry     time.Time
    cacheDuration   time.Duration
}

func (p *MyProvider) DiscoverTools() ([]tool.Tool, error) {
    // 检查缓存
    if time.Now().Before(p.cacheExpiry) && len(p.toolsCache) > 0 {
        return p.toolsCache, nil
    }

    // 重新发现
    tools, err := p.fetchTools()
    if err != nil {
        return nil, err
    }

    // 更新缓存
    p.toolsCache = tools
    p.cacheExpiry = time.Now().Add(p.cacheDuration)

    return tools, nil
}
```

### 4. 认证管理

```go
type MyProvider struct {
    apiKey      string
    tokenCache  string
    tokenExpiry time.Time
}

func (p *MyProvider) getAuthToken() (string, error) {
    // 检查 token 缓存
    if time.Now().Before(p.tokenExpiry) && p.tokenCache != "" {
        return p.tokenCache, nil
    }

    // 刷新 token
    token, expiry, err := p.refreshToken()
    if err != nil {
        return "", err
    }

    p.tokenCache = token
    p.tokenExpiry = expiry

    return token, nil
}
```

## 性能考虑

1. **延迟加载** - 只在需要时创建工具实例
2. **连接池** - 复用 HTTP 连接
3. **并发控制** - 限制并发请求数
4. **超时设置** - 避免长时间阻塞
5. **缓存策略** - 缓存工具列表和结果

## 安全考虑

1. **认证** - 安全存储和传输 API 密钥
2. **授权** - 验证工具访问权限
3. **输入验证** - 验证工具参数
4. **输出过滤** - 过滤敏感信息
5. **审计日志** - 记录工具调用

## 未来扩展

### 计划中的 Provider

1. **OpenAPI Provider** - 从 OpenAPI 规范生成工具
2. **LangChain Provider** - 集成 LangChain 工具
3. **Function Calling Provider** - OpenAI/Anthropic 函数调用
4. **gRPC Provider** - gRPC 服务集成
5. **WebSocket Provider** - 实时工具通信

### 计划中的功能

1. **工具版本管理** - 支持工具版本控制
2. **工具依赖** - 工具间依赖关系
3. **工具组合** - 组合多个工具
4. **工具市场** - 共享和发现工具
5. **工具监控** - 性能和使用统计

## 总结

Tool Provider 架构为 GoReAct 框架提供了强大的扩展能力：

✅ **可扩展** - 支持任意外部工具协议
✅ **解耦** - Provider 独立开发和维护
✅ **动态** - 运行时发现和加载工具
✅ **标准化** - 统一的接口和规范
✅ **兼容** - 与现有系统无缝集成

这个架构使得 GoReAct 能够轻松集成各种外部工具生态系统，大大扩展了框架的能力边界。
