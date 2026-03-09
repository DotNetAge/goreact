# MCP Provider Example

演示如何使用 MCP (Model Context Protocol) Provider 集成外部工具服务器。

## 什么是 MCP？

Model Context Protocol (MCP) 是一个开放标准，用于连接 AI 应用与外部工具和数据源。

## 示例架构

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│  GoReAct     │─────▶│ MCP Provider │─────▶│  MCP Server  │
│  Engine      │      │              │      │  (External)  │
└──────────────┘      └──────────────┘      └──────────────┘
                             │
                             ▼
                      ┌──────────────┐
                      │ Discovered   │
                      │   Tools      │
                      └──────────────┘
```

## 运行示例

### 1. 启动 Mock MCP Server

```bash
cd examples/mcp_provider
go run mock_server.go
```

这会启动一个模拟的 MCP Server，监听在 `http://localhost:8080`

### 2. 运行客户端

```bash
go run main.go
```

## Mock MCP Server 提供的工具

1. **weather** - 获取天气信息
2. **translate** - 翻译文本
3. **search** - 搜索信息

## 示例输出

```
=== MCP Provider Integration Demo ===

Initializing MCP Provider...
✓ MCP Provider initialized successfully

Discovering tools from MCP Server...
✓ Discovered 3 tools:
  1. weather - Get current weather information
  2. translate - Translate text between languages
  3. search - Search for information

Registering tools with Engine...
✓ All tools registered

Executing task with MCP tools...
Task: What's the weather in Tokyo?

=== Execution Result ===
Success: true
Output: The weather in Tokyo is 22°C and Sunny
```

## 配置选项

```go
config := map[string]interface{}{
    "server_url": "http://localhost:8080",  // MCP Server URL
    "api_key":    "your-api-key",           // API Key (可选)
    "timeout":    30,                        // 超时时间（秒）
}
```

## 与真实 MCP Server 集成

要连接到真实的 MCP Server，只需修改配置：

```go
config := map[string]interface{}{
    "server_url": "https://your-mcp-server.com",
    "api_key":    os.Getenv("MCP_API_KEY"),
    "timeout":    60,
}
```

## MCP Server 实现参考

如果你想实现自己的 MCP Server，参考 `mock_server.go` 中的实现。

关键端点：
- `GET /health` - 健康检查
- `GET /tools` - 列出工具
- `POST /execute` - 执行工具
