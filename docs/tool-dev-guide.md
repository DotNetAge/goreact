# GoReAct 工具开发指南

本指南面向 GoReAct 框架的工具开发者，覆盖从接口定义、参数校验、安全管线到高级特性的完整开发流程。

## 1. 概述

在 GoReAct 的 T-A-O（Think-Act-Observe）循环中，工具（Tool）是 Agent 与外部世界交互的核心载体。Think 阶段由 LLM 决定调用哪个工具及参数，Act 阶段执行工具，Observe 阶段分析执行结果并决定下一步行动。

工具的执行流程经过多层安全检查和结果处理管线：

```
LLM 输出工具调用 → 参数解析
  → [SecurityPolicy] 遗留策略检查
  → [PreToolUse Hooks] 前置钩子（可拦截/修改）
  → [ToolPermissionChecker] 权限检查（allow/deny/ask）
  → [工具 Execute] 实际执行
  → [PostToolUse Hooks] 后置钩子（可观察/修改结果）
  → [大结果处理] 超阈值结果溢写到磁盘
  → 结果返回给 LLM
```

## 2. 接口定义

### 2.1 FuncTool 接口

所有工具必须实现 `core.FuncTool` 接口（定义在 `core/tool.go`）：

```go
type FuncTool interface {
    Info() *ToolInfo
    Execute(ctx context.Context, params map[string]any) (any, error)
}
```

`Info()` 返回工具的元数据，`Execute()` 是实际执行逻辑。参数以 `map[string]any` 形式传入，由 LLM 生成。

### 2.2 ToolInfo 结构体

```go
type ToolInfo struct {
    Name              string        // 工具名称，全局唯一标识符
    Description       string        // 工具描述，直接发送给 LLM 用于工具选择
    SecurityLevel     SecurityLevel // 安全级别：LevelSafe / LevelSensitive / LevelHighRisk
    IsIdempotent      bool          // 是否幂等（相同输入是否产生相同结果）
    Parameters        []Parameter   // 参数定义列表
    ReturnType        string        // 返回类型描述（信息性字段）
    Examples          []string      // 使用示例（信息性字段）
    MaxResultSizeChars int          // 单工具结果大小阈值，-1 禁用持久化，0 使用全局默认
    IsReadOnly        bool          // 是否只读（无副作用）
}
```

其中 `Description` 是最关键的字段——LLM 完全依赖它来决定何时以及如何使用工具。

### 2.3 Parameter 结构体

```go
type Parameter struct {
    Name        string // 参数名，对应 params map 的 key
    Type        string // 类型："string", "integer", "boolean", "number", "array"
    Required    bool   // 是否必需
    Default     any    // 默认值（可选）
    Description string // 参数描述，帮助 LLM 理解如何填入
    Enum        []any  // 允许的取值范围（可选）
}
```

LLM 基于参数的 `Type`、`Required` 和 `Description` 来构造调用参数。

## 3. 最小可运行示例

以下是一个从零到注册的完整流程，展示如何创建一个时间戳工具。

### 3.1 定义工具

```go
package mytools

import (
    "context"
    "fmt"
    "time"

    "github.com/DotNetAge/goreact/core"
)

// TimestampTool 返回当前时间戳及格式化时间
type TimestampTool struct{}

func NewTimestampTool() *TimestampTool {
    return &TimestampTool{}
}

func (t *TimestampTool) Info() *core.ToolInfo {
    return &core.ToolInfo{
        Name:          "timestamp",
        Description:   "Get the current timestamp. Returns both Unix timestamp and formatted time string.",
        SecurityLevel: core.LevelSafe,
        IsReadOnly:    true,
        IsIdempotent:  false,
        ReturnType:    "string",
    }
}

func (t *TimestampTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    now := time.Now()
    return fmt.Sprintf("Unix: %d | Formatted: %s", now.Unix(), now.Format(time.RFC3339)), nil
}
```

### 3.2 注册到 Reactor

**编译时注册**（推荐）——通过 `WithExtraTools` 选项：

```go
r := reactor.NewReactor(config,
    reactor.WithExtraTools(mytools.NewTimestampTool()),
)
```

**运行时注册**——在 Reactor 创建后动态添加：

```go
_ = r.RegisterTool(mytools.NewTimestampTool())
```

**禁用内置工具**——如果自定义工具要替代某个内置工具：

```go
r := reactor.NewReactor(config,
    reactor.WithoutTool("echo"),        // 禁用特定内置工具
    reactor.WithExtraTools(mytools.NewTimestampTool()), // 用自定义工具替代
)
```

## 4. 参数校验与错误处理

### 4.1 使用内置校验工具函数

`tools` 包提供了常用校验函数（`tools/utils.go`）：

```go
// 校验必需的字符串参数
func ValidateRequiredString(params map[string]any, key string) (string, error)
// 校验参数是否存在
func ValidateRequired(params map[string]any, key string) error
// 数值类型转换（支持 float64/float32/int/int64/int32）
func ToFloat64(v any) (float64, bool)
// 文件路径安全校验（防路径穿越、限制工作目录范围）
func ValidateFileSafety(path string) error
```

### 4.2 典型的 Execute 方法结构

```go
func (t *MyTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    // 1. 提取并校验必需参数
    path, err := tools.ValidateRequiredString(params, "path")
    if err != nil {
        return nil, err
    }

    // 2. 提取可选参数，提供合理默认值
    limit := 100
    if raw, ok := params["limit"]; ok {
        if v, ok := tools.ToFloat64(raw); ok && v > 0 {
            limit = int(v)
        }
    }

    // 3. 执行核心逻辑
    result, err := doWork(path, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to process %q: %w", path, err)
    }

    // 4. 返回结果（string 或 map[string]any 均可）
    return result, nil
}
```

### 4.3 错误处理原则

- 参数缺失或类型错误：返回明确的 `fmt.Errorf`，描述缺失的参数名
- 业务逻辑错误：使用 `fmt.Errorf` 包装原始错误（`%w`），保留错误链
- 不要吞掉错误，也不要 panic
- 错误信息应该对 LLM 可读，帮助它在下一轮迭代中修正调用

### 4.4 返回值类型

`Execute` 返回 `(any, error)`，`ToolRegistry` 会自动将 `any` 序列化为 JSON 字符串。支持的返回值类型：

- `string`：直接作为结果字符串
- `map[string]any`：序列化为 JSON（推荐用于结构化数据）
- 其他类型：通过 `fmt.Sprintf("%v", result)` 转换

内置工具中常见的模式是返回结构化的 map，例如 Write 工具：

```go
return map[string]any{
    "success":      true,
    "path":         path,
    "bytes_written": bytesWritten,
    "message":      "File written successfully",
}, nil
```

## 5. 安全级别与权限管线

### 5.1 安全级别

```go
const (
    LevelSafe      SecurityLevel = iota // 纯查询，无副作用（如 read、web_search）
    LevelSensitive                      // 有限写入（如 write、file_edit）
    LevelHighRisk                       // 不可逆/高风险（如 bash、delete）
)
```

选择安全级别的原则：

- 工具只读取数据、无任何副作用 → `LevelSafe`
- 工具会创建或修改数据，但可预测、范围有限 → `LevelSensitive`
- 工具执行不可逆操作或存在不可控风险 → `LevelHighRisk`

### 5.2 权限管线执行顺序

`ToolRegistry.ExecuteTool` 按以下顺序执行权限检查（`reactor/action.go`）：

**第一层：SecurityPolicy（遗留）**

```go
// 简单的 allow/deny 函数，已标记为 Deprecated
// 通过 WithSecurityPolicy 设置
type SecurityPolicy func(toolName string, level SecurityLevel) bool
```

**第二层：PreToolUse Hooks**

```go
// 实现 core.Hook 接口，EventType 返回 HookPreToolUse
// 可以：阻止执行、修改输入参数、做出权限决策
type HookResult struct {
    *PermissionResult      // 权限决策（可选）
    UpdatedInput    map[string]any // 修改后的参数（可选）
    PreventContinuation bool        // 是否阻止执行
}
```

PreToolUse hook 的执行结果会影响后续流程：如果 `PreventContinuation` 为 true，工具不会执行；如果返回 `PermissionResult`，会传递给权限检查器。

**第三层：ToolPermissionChecker**

```go
// 完整的权限检查器，支持三种语义
type ToolPermissionChecker interface {
    CheckPermissions(ctx *ToolUseContext) PermissionResult
}

// PermissionResult.Behavior:
//   - PermissionAllow：允许执行
//   - PermissionDeny：拒绝执行
//   - PermissionAsk：暂停等待用户决策（通过 PermissionResponder 机制）
```

GoReAct 默认使用 `AskPermission` 作为权限检查器（`tools/ask_permission.go`）。当安全级别为 `LevelHighRisk` 时，它会返回 `PermissionAsk`，触发用户确认流程。

**第四层：PostToolUse Hooks**

在工具执行完成后触发，可以观察或修改执行结果：

```go
type PostToolUseContext struct {
    *ToolUseContext
    Result   string // 工具执行结果
    Err      error  // 执行错误
    Duration int64  // 执行时长（毫秒）
}
```

### 5.3 Hook 注册示例

```go
// 自定义 PreToolUse hook：阻止对特定路径的文件写入
type PathGuardHook struct{}

func (h *PathGuardHook) EventType() core.HookEventType {
    return core.HookPreToolUse
}

func (h *PathGuardHook) Execute(ctx any) core.HookResult {
    useCtx := ctx.(*core.ToolUseContext)
    if useCtx.ToolName == "write" || useCtx.ToolName == "file_edit" {
        if path, ok := useCtx.Params["path"].(string); ok {
            if strings.Contains(path, "/etc/") {
                return core.HookResult{
                    PreventContinuation: true,
                    Message:            "writing to /etc/ is not allowed",
                }
            }
        }
    }
    return core.HookResult{}
}

// 注册到 ToolRegistry
r.ToolRegistry().AddHook(&PathGuardHook{})
```

## 6. 大结果处理策略

GoReAct 内置了两层结果大小防护机制，防止工具输出撑爆上下文窗口。

### 6.1 机制概述

**单工具结果限制**（`MaxResultSizeChars`）：

- 超过阈值的结果自动溢写到磁盘（`ToolResultStorage`）
- 只在上下文中保留预览文本和文件路径
- LLM 被告知可使用 read 工具读取完整内容

**单消息总量限制**（`MaxToolResultsPerMessageChars`）：

- 一个 T-A-O 循环内所有工具结果的字符总量上限
- 超出时强制截断当前结果

默认值（`core.DefaultToolResultLimits()`）：

| 配置项 | 默认值 |
|--------|--------|
| MaxResultSizeChars | 50,000 字符 |
| MaxToolResultsPerMessageChars | 200,000 字符 |

### 6.2 配置大结果处理

```go
// 创建磁盘持久化存储
storage := core.NewDiskToolResultStorage(
    core.WithStorageDir("/tmp/myapp/tool-results"),
    core.WithPreviewChars(2000), // 预览文本长度
    core.WithSessionID("session-001"),
)

// 设置限制
limits := core.ToolResultLimits{
    MaxResultSizeChars:           30000,
    MaxToolResultsPerMessageChars: 150000,
}

r := reactor.NewReactor(config,
    reactor.WithResultStorage(storage),
    reactor.WithResultLimits(limits),
)
```

### 6.3 按工具覆盖

通过 `ToolInfo.MaxResultSizeChars` 针对特定工具设置不同的阈值：

```go
// 禁用持久化（适合自带分页的工具，如 read）
MaxResultSizeChars: -1

// 使用全局默认值
MaxResultSizeChars: 0

// 自定义阈值
MaxResultSizeChars: 100000
```

内置的 Read 工具使用 `MaxResultSizeChars: -1` 禁用持久化，因为它通过 `offset`/`limit` 参数自行控制输出大小。

### 6.4 持久化结果的格式

当结果被持久化后，LLM 看到的上下文文本格式为：

```
[Result from <tool_name>: <N> chars total, persisted to disk]
Preview:
<前 N 个字符的预览>

Full result saved at: <文件路径>
To read the full content, use the read tool with path: <文件路径>
```

## 7. 访问 Reactor 资源

部分工具需要访问 Reactor 的内部资源（任务管理器、消息总线、事件总线等），例如 task_create、subagent、team_create 等编排工具。

### 7.1 ReactorAccessor 接口

```go
// tools/reactor_accessor.go
type ReactorAccessor interface {
    TaskManager() core.TaskManager
    MessageBus() *core.AgentMessageBus
    EventEmitter() func(core.ReactEvent)
    RegisterPendingTask(taskID string, resultCh chan any)
    GetPendingTask(taskID string) (<-chan any, bool)
    RemovePendingTask(taskID string)
    RunInline(ctx context.Context, prompt string) (answer string, err error)
    RunSubAgent(ctx context.Context, taskID string, systemPrompt, prompt string, model string, resultCh chan<- any)
    Config() ReactorConfig
}
```

### 7.2 实现模式

需要 Reactor 资源的工具应：

1. 定义 `accessor` 字段
2. 提供 `SetAccessor` 注入方法
3. 在 `Execute` 中检查 accessor 是否已注入
4. 通过 accessor 访问 Reactor 资源

```go
type MyOrchestrationTool struct {
    accessor ReactorAccessor
}

func (t *MyOrchestrationTool) SetAccessor(a ReactorAccessor) {
    t.accessor = a
}

func (t *MyOrchestrationTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    if t.accessor == nil {
        return nil, fmt.Errorf("reactor accessor not configured")
    }

    // 通过 accessor 访问资源
    tm := t.accessor.TaskManager()
    task, err := tm.CreateTask("", "description", "prompt")
    if err != nil {
        return nil, fmt.Errorf("failed to create task: %w", err)
    }

    // 发射事件
    if emitter := t.accessor.EventEmitter(); emitter != nil {
        emitter(core.NewReactEvent("", "main", "", core.SubtaskSpawned,
            core.SubtaskInfo{TaskID: task.ID}))
    }

    return task.ID, nil
}
```

### 7.3 Reactor 自动注入

Reactor 在 `registerOrchestrationTools()` 中会自动为编排工具注入 accessor（`reactor/accessor_impl.go`）。Reactor 自身实现了 `ReactorAccessor` 接口：

```go
var _ tools.ReactorAccessor = (*Reactor)(nil)
```

如果自定义工具需要 Reactor 资源，可以在 Reactor 初始化后手动注入：

```go
myTool := mytools.NewMyOrchestrationTool()
myTool.SetAccessor(r) // r 是 *reactor.Reactor 实例
_ = r.RegisterTool(myTool)
```

## 8. 语义工具搜索（反射记忆）

### 8.1 工作原理

`ToolRegistry.GetWithSemantic()` 实现了两阶段工具查找：

1. 精确匹配：按工具名查找（`tools["name"]`）
2. 语义回退：如果 Memory 已配置且提供了 intent，通过 `Memory.Retrieve` 进行语义搜索，从返回的记录中提取工具名，再从注册表中查找

这使得工具可以通过意图描述被找到，而非精确名称匹配。

### 8.2 前提条件

要启用语义搜索，需要：

1. 配置 Memory 实现（通过 `WithMemory` 选项）
2. Memory 中存在工具相关的 `MemoryTypeRefactive` 类型记录
3. 记录的 `Title` 字段与注册的工具名称一致

Reactor 初始化时会自动将 Memory 注入到 ToolRegistry（`reactor.go` 第 488 行）：

```go
if r.memory != nil {
    r.toolRegistry.SetMemory(r.memory)
}
```

### 8.3 搜索流程

```go
func (r *ToolRegistry) GetWithSemantic(ctx context.Context, name string, intent string) (core.FuncTool, bool) {
    // Phase 1: 精确匹配
    if tool, ok := r.Get(name); ok {
        return tool, true
    }

    // Phase 2: 语义搜索
    records, err := mem.Retrieve(ctx, intent,
        core.WithMemoryTypes(core.MemoryTypeRefactive),
        core.WithMemoryLimit(5),
    )

    // Phase 3: 从搜索结果中提取工具名并精确查找
    for _, rec := range records {
        if tool, ok := r.Get(rec.Title); ok {
            return tool, true
        }
    }
    return nil, false
}
```

## 9. 注册方式

### 9.1 编译时注册

通过 `ReactorOption` 在创建 Reactor 时注册：

```go
// 添加自定义工具（与内置工具共存）
r := reactor.NewReactor(config,
    reactor.WithExtraTools(tool1, tool2, tool3),
)

// 禁用特定内置工具
r := reactor.NewReactor(config,
    reactor.WithoutTool("bash"),       // 禁用 bash
    reactor.WithoutTool("web_search"), // 禁用 web_search
)

// 禁用全部内置工具（仅保留编排工具如 task_create、subagent 等）
r := reactor.NewReactor(config,
    reactor.WithoutBundledTools(),
)
```

`WithoutBundledTools()` 不会影响编排工具（task、subagent、team、skill），它们在 `registerOrchestrationTools()` 中独立注册。

### 9.2 运行时注册

在 Reactor 创建后动态注册：

```go
_ = r.RegisterTool(myTool)
```

返回 error，但内置代码中通常忽略（`_ = r.RegisterTool(...)`），因为重复注册不会导致 panic。

### 9.3 注册冲突处理

如果注册了同名工具，`Register` 会返回错误。内置工具的注册顺序在 `NewReactor` 中确定，自定义工具通过 `WithExtraTools` 注册时在内置工具之后，因此可以覆盖同名内置工具。

## 10. 最佳实践

### 10.1 Description 编写

`Description` 是 LLM 选择和使用工具的唯一依据，应当：

- 第一句话概括工具的核心功能
- 明确说明参数的语义和格式要求
- 列出关键行为和边界条件
- 说明与相关工具的区别和协作关系
- 避免过于简短或模糊

参考 Bash 工具的 Description 写法（`tools/bash.go`）：

```go
const bashDescription = `Executes a given bash command and returns its output.

The working directory persists between commands, but shell state does not.

IMPORTANT: Avoid using this tool to run cat, head, tail, sed, awk, or echo
commands, unless explicitly instructed. Instead, use the appropriate dedicated
tool as this will provide a much better experience for the user:
- File search: Use glob (NOT find or ls)
- Content search: Use grep (NOT grep or rg)
- Read files: Use read (NOT cat/head/tail)
...

# Instructions
- If your command will create new directories or files, first use this tool
  to run ls to verify the parent directory exists and is the correct location.
...`
```

### 10.2 幂等性

如果工具的执行结果不受调用次数影响（即相同输入始终产生相同结果），应设置 `IsIdempotent: true`。这有助于 LLM 判断是否可以安全重试。

- 幂等工具：read、grep、glob、web_search、calculator
- 非幂等工具：bash（可能有副作用）、write（覆盖文件）、file_edit

### 10.3 超时处理

工具应当尊重传入的 `context.Context`，支持取消和超时：

```go
func (t *MyTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    // 方式一：使用 context 的 Done channel
    select {
    case result := <-doWorkAsync():
        return result, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    // 方式二：使用 context.WithTimeout
    timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    // 使用 timeoutCtx 进行操作...
}
```

Bash 工具使用 `exec.CommandContext` 来实现超时控制（`tools/bash.go` 第 82 行）。

### 10.4 结果格式化

- 对于简单结果，直接返回字符串
- 对于结构化数据，返回 `map[string]any`，ToolRegistry 会自动 JSON 序列化
- 控制输出大小，避免不必要的冗余信息
- 对于列表数据，限制每项的显示长度（参考 SubAgentListTool 的截断处理）
- 字符串截断使用 rune 感知的方式，安全处理多字节字符：`[]rune(s)[:n]`

### 10.5 文件操作工具的安全实践

所有涉及文件路径的工具都应调用 `tools.ValidateFileSafety(path)` 进行安全校验。该函数执行以下检查：

1. 路径规范化（`filepath.Clean`）
2. 解析为绝对路径
3. 解析符号链接，获取真实路径
4. 确保路径在工作目录范围内（防路径穿越）
5. 检查是否访问受限系统文件（`.env`、`id_rsa` 等）

### 10.6 工具命名

- 使用小写蛇形命名：`web_search`、`file_edit`、`task_create`
- 名称应简明且具有描述性
- 避免与内置工具冲突（内置工具列表见 `reactor/reactor.go` 第 424-447 行）

### 10.7 不需要 Reactor 资源的工具

如果工具不需要访问 Reactor 内部资源（任务管理、消息总线等），只需实现 `FuncTool` 接口即可，无需 `SetAccessor`。大多数自定义工具属于这一类。

### 10.8 内置工具参考

GoReact 内置了丰富的工具，可作为开发的参考：

| 工具名 | 文件 | 特点 |
|--------|------|------|
| echo | `tools/echo.go` | 最简单的工具，纯查询，无参数校验 |
| write | `tools/write.go` | LevelSensitive，使用 ValidateRequiredString 和 ValidateFileSafety |
| read | `tools/read.go` | LevelSafe，MaxResultSizeChars=-1，内置分页和大小限制 |
| bash | `tools/bash.go` | LevelHighRisk，使用 context 超时，结构化返回 map |
| web_search | `tools/web_search.go` | 适配器模式，缓存，结果过滤 |
| task_create | `tools/task_tools.go` | 需要 ReactorAccessor，创建并执行子任务 |
| subagent | `tools/task_tools.go` | 需要 ReactorAccessor，异步独立 Agent |
| file_edit | `tools/edit.go` | LevelSensitive，文件新鲜度检查 |
