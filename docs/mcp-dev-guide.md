# MCP 开发指南

## 1. 概述

MCP（Model Context Protocol）是 GoReAct 框架中用于接入外部工具服务器的标准化协议层。通过 MCP，GoReAct 可以发现和调用由独立进程或远程服务提供的工具，并将这些外部工具无缝转换为框架内部的 `FuncTool`，与内置工具享有同等的注册和调度能力。

GoReAct 的 MCP 实现采用接口驱动的设计：框架本身只定义核心抽象（`MCPClient` 接口、`MCPToolRegistry`、`MCPToolAdapter`），不绑定任何具体的传输实现。开发者可以根据实际需求，实现 stdio（子进程通信）或 SSE（HTTP 长连接）等传输方式。

## 2. MCP 协议简介

MCP（Model Context Protocol）是一种标准化的工具通信协议，定义了 AI 模型与外部工具服务之间的交互方式。其核心目标是解决工具接入的碎片化问题——不同工具服务的通信方式各异，MCP 提供了一套统一的发现、调用和错误处理机制。

在 GoReAct 的架构中，MCP 扮演以下角色：

- **工具扩展通道**：允许 Agent 在不修改框架代码的前提下接入任意数量的外部工具。
- **关注点分离**：MCP 服务器负责工具的具体实现逻辑，GoReAct 只负责调度和结果整合。
- **跨语言兼容**：MCP 服务器可以用 Python、Node.js、Rust 等任何语言编写，只要遵循 MCP 协议即可。

MCP 协议基于 JSON-RPC 2.0，核心操作包括三个阶段：

1. **初始化（initialize）**：客户端与服务端握手，协商能力集。
2. **工具发现（tools/list）**：客户端查询服务端提供的所有工具及其参数定义。
3. **工具调用（tools/call）**：客户端按名称调用指定工具，传递结构化参数并获取结果。

## 3. 核心接口详解

### 3.1 MCPClient

`MCPClient` 是 GoReAct 定义的 MCP 通信接口，位于 `core` 包中。任何 MCP 传输实现都必须满足此接口：

```go
type MCPClient interface {
    Connect(ctx context.Context) error
    Disconnect() error
    ListTools(ctx context.Context) ([]MCPToolInfo, error)
    CallTool(ctx context.Context, toolName string, params map[string]any) (any, error)
    IsConnected() bool
}
```

各方法职责：

| 方法 | 职责 |
|------|------|
| `Connect` | 建立 MCP 连接，完成协议握手（initialize 请求/响应） |
| `Disconnect` | 关闭连接，释放子进程、网络连接等资源 |
| `ListTools` | 查询 MCP 服务器提供的全部工具，返回 `MCPToolInfo` 列表 |
| `CallTool` | 按工具名称调用指定工具，传入参数，返回执行结果 |
| `IsConnected` | 查询当前连接状态，用于在调用前判断是否需要重连 |

### 3.2 MCPToolRegistry

`MCPToolRegistry` 管理多个 MCP 服务器连接，负责批量发现工具：

```go
type MCPToolRegistry struct {
    clients map[string]MCPClient
    mu      sync.Mutex
}
```

关键方法：

- `NewMCPToolRegistry()`：创建空的注册表实例。
- `RegisterClient(name, client)`：将一个 `MCPClient` 注册到注册表，`name` 为服务器标识。
- `DiscoverTools(ctx)`：遍历所有已注册的客户端，自动连接未连接的服务器，调用 `ListTools` 收集工具，并通过 `MCPToolAdapter` 包装为 `FuncTool` 列表返回。此方法内部持有互斥锁，并发安全。
- `DisconnectAll()`：关闭所有已连接的 MCP 服务器连接。

### 3.3 MCPToolAdapter

`MCPToolAdapter` 实现了 `FuncTool` 接口，将 MCP 工具桥接为 GoReAct 的标准工具：

```go
type MCPToolAdapter struct {
    info        *ToolInfo
    client      MCPClient
    mcpToolName string
}
```

工作原理：
- `NewMCPToolAdapter` 将 MCP 工具的 `ToolInfo` 复制一份，并强制将 `IsReadOnly` 设为 `false`（因为 MCP 工具可能产生副作用）。
- `Info()` 返回工具元数据，供 LLM 在工具选择阶段使用。
- `Execute()` 在调用前检查连接状态，未连接则返回错误，已连接则委托给 `MCPClient.CallTool`。

## 4. MCP 服务器配置

`MCPServerConfig` 定义了 MCP 服务器的连接参数，支持 JSON 和 YAML 格式序列化：

```go
type MCPServerConfig struct {
    Name      string            `json:"name" yaml:"name"`
    Transport string            `json:"transport" yaml:"transport"`
    Command   string            `json:"command,omitempty" yaml:"command,omitempty"`
    Args      []string          `json:"args,omitempty" yaml:"args,omitempty"`
    URL       string            `json:"url,omitempty" yaml:"url,omitempty"`
    Env       map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}
```

字段说明：

| 字段 | 类型 | 说明 |
|------|------|------|
| `Name` | string | 服务器标识，用于注册和日志输出 |
| `Transport` | string | 传输类型，取值 `"stdio"` 或 `"sse"` |
| `Command` | string | stdio 模式下要启动的可执行文件路径 |
| `Args` | []string | 传递给可执行文件的命令行参数 |
| `URL` | string | SSE 模式下的服务器端点地址 |
| `Env` | map[string]string | 传递给子进程的额外环境变量 |

配置示例（stdio 模式，JSON）：

```json
{
    "name": "my-tools",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@my-org/mcp-server"],
    "env": {
        "API_KEY": "sk-xxx"
    }
}
```

配置示例（SSE 模式，YAML）：

```yaml
name: remote-tools
transport: sse
url: https://tools.example.com/mcp
```

## 5. 使用流程

以下是将 MCP 工具接入 GoReAct 的完整步骤：

### 步骤 1：实现 MCPClient 接口

根据目标 MCP 服务器的传输方式，实现 `core.MCPClient` 接口（参见第 6 节）。

### 步骤 2：创建 MCPToolRegistry 并注册客户端

```go
registry := core.NewMCPToolRegistry()
registry.RegisterClient("my-server", myClient)
```

### 步骤 3：创建 Reactor 并传入注册表

```go
r := reactor.NewReactor(config,
    reactor.WithMCPRegistry(registry),
)
```

### 步骤 4：发现工具并注册到 Reactor

```go
tools, err := registry.DiscoverTools(ctx)
if err != nil {
    log.Fatalf("MCP tool discovery failed: %v", err)
}
for _, tool := range tools {
    if err := r.RegisterTool(tool); err != nil {
        log.Printf("failed to register MCP tool %s: %v", tool.Info().Name, err)
    }
}
```

### 步骤 5：关闭时清理连接

```go
defer registry.DisconnectAll()
```

**重要说明**：当前 `WithMCPRegistry` 选项保留了接口但尚未实现自动 lazy discovery。工具需要手动调用 `DiscoverTools` 后注册到 Reactor。这是有意为之的设计，让调用者完全控制 MCP 工具的发现时机和错误处理策略。

## 6. 实现 MCPClient 接口指南

### 6.1 stdio 传输

stdio 传输通过启动子进程，在 stdin/stdout 上进行 JSON-RPC 通信。适用于本地 MCP 服务器（如 Python、Node.js 编写的工具服务）。

基本结构：

```go
type StdioMCPClient struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    stderr io.ReadCloser
    mu     sync.Mutex
    conn   bool
    nextID atomic.Int64
}

func NewStdioMCPClient(cfg core.MCPServerConfig) *StdioMCPClient {
    cmd := exec.Command(cfg.Command, cfg.Args...)
    cmd.Env = os.Environ()
    for k, v := range cfg.Env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }
    return &StdioMCPClient{cmd: cmd}
}
```

Connect 实现：

```go
func (c *StdioMCPClient) Connect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    stdin, err := c.cmd.StdinPipe()
    if err != nil {
        return fmt.Errorf("create stdin pipe: %w", err)
    }
    stdout, err := c.cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("create stdout pipe: %w", err)
    }
    c.stdin = stdin
    c.stdout = stdout

    if err := c.cmd.Start(); err != nil {
        return fmt.Errorf("start process: %w", err)
    }

    // 发送 initialize 请求
    initResp, err := c.sendRequest(ctx, "initialize", map[string]any{
        "protocolVersion": "2024-11-05",
        "capabilities":    map[string]any{},
        "clientInfo": map[string]any{
            "name":    "goreact",
            "version": "1.0.0",
        },
    })
    if err != nil {
        c.cmd.Process.Kill()
        return fmt.Errorf("initialize handshake: %w", err)
    }

    // 发送 initialized 通知
    c.sendNotification("notifications/initialized", map[string]any{})

    c.conn = true
    return nil
}
```

ListTools 实现：

```go
func (c *StdioMCPClient) ListTools(ctx context.Context) ([]core.MCPToolInfo, error) {
    resp, err := c.sendRequest(ctx, "tools/list", map[string]any{})
    if err != nil {
        return nil, err
    }

    rawTools, ok := resp["tools"].([]any)
    if !ok {
        return nil, fmt.Errorf("invalid tools/list response")
    }

    var result []core.MCPToolInfo
    for _, raw := range rawTools {
        toolMap := raw.(map[string]any)
        info := convertMCPToolToToolInfo(toolMap) // 参见第 7 节
        result = append(result, core.MCPToolInfo{
            ServerName: c.serverName,
            ToolInfo:   info,
        })
    }
    return result, nil
}
```

CallTool 实现：

```go
func (c *StdioMCPClient) CallTool(ctx context.Context, toolName string, params map[string]any) (any, error) {
    resp, err := c.sendRequest(ctx, "tools/call", map[string]any{
        "name":      toolName,
        "arguments": params,
    })
    if err != nil {
        return nil, err
    }
    return resp, nil
}
```

### 6.2 SSE 传输

SSE（Server-Sent Events）传输通过 HTTP 建立长连接，适用于远程 MCP 服务器。

基本结构：

```go
type SSEMCPClient struct {
    baseURL    string
    httpClient *http.Client
    sessionID  string
    mu         sync.Mutex
    conn       bool
}
```

Connect 实现：

```go
func (c *SSEMCPClient) Connect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 建立 SSE 连接获取 session ID
    req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/sse", nil)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("SSE connect: %w", err)
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")
            // 解析 event，提取 session endpoint
            // 通常为 {"endpoint": "/messages?session_id=xxx"}
        }
    }

    // 通过 session endpoint 发送 initialize 请求
    initResp, err := c.postMessage(ctx, "initialize", map[string]any{
        "protocolVersion": "2024-11-05",
        "capabilities":    map[string]any{},
        "clientInfo": map[string]any{
            "name":    "goreact",
            "version": "1.0.0",
        },
    })

    c.conn = true
    return nil
}
```

ListTools 和 CallTool 实现：

```go
func (c *SSEMCPClient) ListTools(ctx context.Context) ([]core.MCPToolInfo, error) {
    resp, err := c.postMessage(ctx, "tools/list", map[string]any{})
    if err != nil {
        return nil, err
    }
    // 解析方式与 stdio 相同
    return parseToolList(resp), nil
}

func (c *SSEMCPClient) CallTool(ctx context.Context, toolName string, params map[string]any) (any, error) {
    resp, err := c.postMessage(ctx, "tools/call", map[string]any{
        "name":      toolName,
        "arguments": params,
    })
    if err != nil {
        return nil, err
    }
    return resp, nil
}
```

## 7. MCP Schema 到 ToolInfo 的转换规则

MCP 协议使用 JSON Schema 定义工具参数，GoReAct 使用自己的 `ToolInfo` 和 `Parameter` 结构体。实现 `MCPClient` 时需要完成这一转换。以下是转换映射规则：

### ToolInfo 字段映射

| ToolInfo 字段 | MCP Schema 来源 | 说明 |
|---------------|-----------------|------|
| `Name` | `tool.name` | 直接映射 |
| `Description` | `tool.description` | 直接映射 |
| `SecurityLevel` | 默认 `LevelSensitive` | MCP 未定义安全级别，统一设为 Sensitive |
| `IsIdempotent` | `tool.annotations?.idempotentHint` | 从注解推断，默认 false |
| `IsReadOnly` | `tool.annotations?.readOnlyHint` | 从注解推断，但 `MCPToolAdapter` 会强制设为 false |
| `Parameters` | `tool.inputSchema.properties` | 遍历 properties 转换 |
| `ReturnType` | 默认 `"any"` | MCP 未定义返回类型 |
| `Examples` | `tool.annotations?.examples` 或空 | 从注解提取 |

### Parameter 字段映射

| Parameter 字段 | MCP Schema 来源 | 说明 |
|----------------|-----------------|------|
| `Name` | property key | JSON Schema 中 properties 的键名 |
| `Type` | `property.type` | 将 JSON Schema 类型（string/number/boolean/array/object）映射为 GoReAct 类型标识 |
| `Required` | `tool.inputSchema.required` 列表 | 检查该参数名是否在 required 数组中 |
| `Default` | `property.default` | 直接映射 |
| `Description` | `property.description` | 直接映射 |
| `Enum` | `property.enum` | 直接映射 |

### 转换函数参考实现

```go
func convertMCPToolToToolInfo(toolMap map[string]any) core.ToolInfo {
    info := core.ToolInfo{
        Name:          toolMap["name"].(string),
        Description:   toString(toolMap["description"]),
        SecurityLevel: core.LevelSensitive,
        ReturnType:    "any",
    }

    // 处理注解
    if ann, ok := toolMap["annotations"].(map[string]any); ok {
        info.IsIdempotent = toBool(ann["idempotentHint"])
        info.IsReadOnly = toBool(ann["readOnlyHint"])
    }

    // 转换参数
    if schema, ok := toolMap["inputSchema"].(map[string]any); ok {
        info.Parameters = convertSchemaToParameters(schema)
    }

    return info
}

func convertSchemaToParameters(schema map[string]any) []core.Parameter {
    props, _ := schema["properties"].(map[string]any)
    required, _ := schema["required"].([]any)
    requiredSet := make(map[string]bool)
    for _, r := range required {
        requiredSet[r.(string)] = true
    }

    var params []core.Parameter
    for name, prop := range props {
        pm, _ := prop.(map[string]any)
        params = append(params, core.Parameter{
            Name:        name,
            Type:        toString(pm["type"]),
            Required:    requiredSet[name],
            Default:     pm["default"],
            Description: toString(pm["description"]),
            Enum:        toSlice(pm["enum"]),
        })
    }
    return params
}
```

## 8. 错误处理与重连策略

### 8.1 连接阶段错误

`DiscoverTools` 在遍历客户端时，遇到连接失败或工具列表获取失败会立即返回错误，中止后续服务器的发现。如果需要更精细的控制，建议先逐个调用 `client.Connect()`，确认成功后再调用 `DiscoverTools`：

```go
// 先逐个连接
for name, client := range registry.clients {
    if err := client.Connect(ctx); err != nil {
        log.Printf("warning: MCP server %q failed to connect: %v", name, err)
        continue
    }
}

// 再发现工具（已连接的客户端会被跳过，避免重复连接）
tools, err := registry.DiscoverTools(ctx)
```

### 8.2 工具调用阶段错误

`MCPToolAdapter.Execute` 在调用前检查 `IsConnected()`，如果 MCP 服务器断开连接，会返回 `"MCP server is not connected"` 错误。建议在 Reactor 的安全策略中处理这类错误，或者在自定义的 MCPClient 实现中加入自动重连逻辑。

### 8.3 重连策略

在 `MCPClient` 实现中加入重连时需要注意：

- **指数退避**：首次重试间隔 1 秒，每次翻倍，上限 30 秒。
- **最大重试次数**：建议设为 3 次，避免无限重试阻塞 Reactor 的执行循环。
- **上下文传递**：始终使用 `context.Context` 控制超时和取消，不要忽略 context 的 Done 信号。
- **状态一致性**：重连成功后应重新调用 `ListTools` 刷新工具列表，因为工具可能发生变化。

```go
func (c *StdioMCPClient) CallToolWithRetry(ctx context.Context, name string, params map[string]any) (any, error) {
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        if !c.IsConnected() {
            if err := c.Connect(ctx); err != nil {
                lastErr = err
                time.Sleep(time.Duration(math.Min(math.Pow(2, float64(attempt)), 30)) * time.Second)
                continue
            }
        }
        result, err := c.CallTool(ctx, name, params)
        if err == nil {
            return result, nil
        }
        lastErr = err
    }
    return nil, fmt.Errorf("after 3 retries: %w", lastErr)
}
```

### 8.4 DisconnectAll 错误处理

`DisconnectAll` 会尝试关闭所有已连接的客户端，收集所有错误并在最后返回合并错误。这意味着即使部分服务器断开失败，其他服务器仍然会被尝试关闭。

## 9. 最佳实践

### 9.1 安全性

- **环境变量隔离**：通过 `MCPServerConfig.Env` 传递敏感信息（API Key 等），不要将密钥硬编码在代码中。
- **安全级别标注**：转换工具时根据工具功能合理设置 `SecurityLevel`。纯查询类工具设为 `LevelSafe`，修改数据类的设为 `LevelSensitive`，删除或执行代码类的设为 `LevelHighRisk`。
- **工具白名单**：如果不需要 MCP 服务器的全部工具，可以在注册前过滤 `DiscoverTools` 返回的工具列表，只注册可信的工具。
- **进程沙箱**：对于 stdio 传输的 MCP 服务器，考虑在容器或受限用户下运行子进程，限制其文件系统和网络访问。

### 9.2 超时控制

- **连接超时**：`Connect` 应支持 context 超时，建议默认 10 秒。
- **调用超时**：`CallTool` 通过 `context.Context` 控制超时，Reactor 会在工具调用时传入带超时的 context。建议 MCPClient 实现也内置合理的默认超时（如 30 秒）。
- **发现超时**：`DiscoverTools` 应该使用带超时的 context，避免某个慢速服务器阻塞整个工具发现流程。

### 9.3 资源清理

- **defer DisconnectAll**：在应用启动时注册 `defer registry.DisconnectAll()`，确保进程退出前清理所有子进程和网络连接。
- **信号处理**：捕获 `SIGINT` 和 `SIGTERM` 信号，在信号处理函数中调用 `DisconnectAll`，确保子进程不会成为孤儿进程。
- **stdout/stderr 消费**：对于 stdio 传输，MCP 服务器的 stderr 应被持续读取或丢弃，否则缓冲区填满后子进程会阻塞。

### 9.4 日志与可观测性

- **连接事件日志**：在 `Connect`/`Disconnect` 中记录连接状态变更，包含服务器名称和耗时。
- **工具调用日志**：在 `CallTool` 中记录调用工具名称、参数摘要和执行耗时，便于排查问题。
- **JSON-RPC 调试**：在开发阶段可以添加 JSON-RPC 请求/响应的详细日志，生产环境中应通过日志级别控制关闭。

### 9.5 并发安全

- `MCPToolRegistry` 的所有公开方法都通过 `sync.Mutex` 保证并发安全，可以在多个 goroutine 中安全使用。
- 自定义的 `MCPClient` 实现也需要保证并发安全，因为 Reactor 可能同时调用多个 MCP 工具。使用 `sync.Mutex` 保护共享状态（连接状态、JSON-RPC 请求 ID 等）。

### 9.6 配置管理

- 将 `MCPServerConfig` 放在配置文件中管理，不要硬编码在源代码中。
- 支持通过环境变量覆盖配置中的敏感字段（如 URL、API Key），便于在不同环境（开发/测试/生产）间切换。
- 提供配置验证函数，在启动时检查 `Transport`、`Command`/`URL` 等必填字段的合法性。
