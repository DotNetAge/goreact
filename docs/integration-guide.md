# GoReAct 集成指南

本指南面向需要将 GoReAct 嵌入到自有应用中的开发者，涵盖事件系统、流式输出、Token 追踪、会话管理、中断-恢复交互模式以及团队通信等所有集成要点。

---

## 1. 概述

GoReAct 采用分层架构，集成开发者可以根据需求选择不同的接入层次。

最底层是 `Reactor`，它实现了 `T-A-O (Think-Act-Observe)` 循环引擎，提供完整的意图分类、工具执行、上下文防御能力。`Reactor` 通过 `Run` 方法执行单次推理，返回 `RunResult`。

中间层是 `EventBus`，它与 `Reactor` 解耦，以发布-订阅模式将推理过程中的所有状态变化（思维流、工具调用、子任务派发、执行摘要等）实时推送出去，使外部客户端能够同步渲染推理过程。

最上层是 `Agent`，它在 `Reactor` 之上封装了 `ContextWindow` 会话管理和多轮对话能力。开发者调用 `Ask` 即可完成一次完整的交互，无需手动管理对话历史。

此外，`AgentMessageBus` 提供了多 Agent 团队协作的消息通信机制，支持创建团队、异步派发 SubAgent、Channel 式消息收发。

选择接入层次的建议：

- 仅需单次推理：直接使用 `Reactor.Run`
- 需要实时渲染推理过程：使用 `Reactor` + `EventBus`
- 需要多轮对话：使用 `Agent`
- 需要多 Agent 协作：使用 `AgentMessageBus` + SubAgent 工具

---

## 2. 事件系统详解

### 2.1 EventBus 接口

GoReAct 的 `EventBus` 定义在 `reactor` 包中：

```go
type EventBus interface {
    Emit(event core.ReactEvent)
    Subscribe() (ch <-chan core.ReactEvent, cancel func())
    SubscribeFiltered(filter func(core.ReactEvent) bool) (ch <-chan core.ReactEvent, cancel func())
}
```

内置实现为 `InProcessEventBus`，基于 Go channel 的 fan-out 模型，支持多 goroutine 并发发布与订阅。每个订阅者的 channel 缓冲区为 256，当消费者处理不过来时会丢弃溢出事件，避免阻塞 T-A-O 主循环。

创建 EventBus：

```go
bus := reactor.NewEventBus()
```

在创建 `Reactor` 时注入：

```go
r := reactor.NewReactor(config, reactor.WithEventBus(bus))
```

如果不显式注入，`NewReactor` 会自动创建一个 `InProcessEventBus` 实例。集成时通常建议自行创建，以便在外部订阅事件。

关闭 EventBus：

```go
bus.Close() // 关闭所有订阅者的 channel
```

### 2.2 事件类型全览

以下所有常量定义在 `core` 包中：

| 事件类型 | 常量名 | 说明 | Data 类型 |
|---------|--------|------|-----------|
| 思维流片段 | `ThinkingDelta` | Think 阶段的流式输出 | `string` |
| 思维完成 | `ThinkingDone` | Think 阶段结束 | `Thought` |
| 工具开始 | `ActionStart` | 工具即将执行 | `ActionStartData` |
| 工具进度 | `ActionProgress` | 长时间运行工具的进度报告 | `string` |
| 工具结果 | `ActionResult` | 工具执行完成 | `ActionResultData` |
| 观察完成 | `ObservationDone` | Observe 阶段结束 | `Observation` |
| 子任务创建 | `SubtaskSpawned` | SubAgent 任务已创建 | `SubtaskInfo` |
| 子任务完成 | `SubtaskCompleted` | SubAgent 任务已完成 | `SubtaskResult` |
| 最终答案 | `FinalAnswer` | Reactor 产出最终答案 | `string` |
| 需要澄清 | `ClarifyNeeded` | ask_user 工具触发 | `string` |
| 请求授权 | `PermissionRequest` | 工具需要用户授权 | `PermissionRequestData` |
| 授权被拒 | `PermissionDenied` | 授权被拒绝 | `string` |
| 执行摘要 | `ExecutionSummary` | Run 结束时的统计 | `ExecutionSummaryData` |
| 错误 | `Error` | Reactor 级别错误 | `string` |
| 循环结束 | `CycleEnd` | 单次 T-A-O 循环结束 | `CycleInfo` |
| 经验保存 | `ExperienceSaved` | 成功经验已保存到记忆 | `ExperienceSavedData` |

### 2.3 ReactEvent 结构

每个事件都携带以下元数据：

```go
type ReactEvent struct {
    SessionID string         `json:"session_id"`              // 会话标识
    TaskID    string         `json:"task_id"`                 // "main" 或 "task_1", "task_2" 等
    ParentID  string         `json:"parent_id,omitempty"`     // 父任务 ID，main 为空
    Type      ReactEventType `json:"type"`                    // 事件类型
    Data      any            `json:"data,omitempty"`          // 事件负载
    Timestamp int64          `json:"timestamp"`               // 毫秒时间戳
}
```

其中 `TaskID` 是事件路由的关键字段。主 Reactor 的事件 `TaskID` 为 `"main"`，SubAgent 的事件为 `"task_1"`、`"task_2"` 等。集成时可通过 `TaskID` 将事件路由到正确的 UI 面板。

### 2.4 事件数据负载

**ActionStartData** -- 工具调用开始时发出：

```go
type ActionStartData struct {
    ToolName string         `json:"tool_name"`
    Params   map[string]any `json:"params,omitempty"`
}
```

**ActionResultData** -- 工具执行完成后发出：

```go
type ActionResultData struct {
    ToolName string        `json:"tool_name"`
    Result   string        `json:"result,omitempty"`
    Error    string        `json:"error,omitempty"`
    Duration time.Duration `json:"duration_ms"`
    Success  bool          `json:"success"`
}
```

**SubtaskInfo** -- SubAgent 任务创建时发出：

```go
type SubtaskInfo struct {
    TaskID      string `json:"task_id"`
    Description string `json:"description"`
    Timeout     string `json:"timeout,omitempty"`
}
```

**SubtaskResult** -- SubAgent 任务完成时发出：

```go
type SubtaskResult struct {
    TaskID  string `json:"task_id"`
    Success bool   `json:"success"`
    Answer  string `json:"answer,omitempty"`
    Error   string `json:"error,omitempty"`
}
```

**ExecutionSummaryData** -- Run 结束时发出，包含完整统计：

```go
type ExecutionSummaryData struct {
    TotalIterations   int            `json:"total_iterations"`
    ToolCalls         int            `json:"tool_calls"`
    ToolsUsed         []string       `json:"tools_used,omitempty"`
    TotalDuration     time.Duration  `json:"total_duration_ms"`
    TokensUsed        int            `json:"tokens_used"`
    TerminationReason string         `json:"termination_reason,omitempty"`
}
```

**PermissionRequestData** -- 高风险工具需要授权时发出：

```go
type PermissionRequestData struct {
    ToolName     string         `json:"tool_name"`
    Params       map[string]any `json:"params,omitempty"`
    Reason       string         `json:"reason,omitempty"`
    SecurityLevel SecurityLevel  `json:"security_level"`
}
```

**CycleInfo** -- 每个 T-A-O 循环结束时发出：

```go
type CycleInfo struct {
    Iteration         int           `json:"iteration"`
    TerminationReason string        `json:"termination_reason,omitempty"`
    Duration          time.Duration `json:"duration_ms"`
}
```

**ExperienceSavedData** -- 任务成功后经验被保存时发出：

```go
type ExperienceSavedData struct {
    Problem    string   `json:"problem"`
    Iterations int      `json:"iterations"`
    ToolsUsed  []string `json:"tools_used"`
}
```

**SecurityLevel** -- 安全等级枚举：

```go
const (
    LevelSafe       SecurityLevel = iota // 只读操作，自动放行
    LevelSensitive                       // 有限写入，需要确认
    LevelHighRisk                        // 高风险操作，必须授权
)
```

### 2.5 订阅模式

**全量订阅**：接收所有事件。

```go
ch, cancel := bus.Subscribe()
defer cancel()

for event := range ch {
    // 处理事件
}
```

**过滤订阅**：只接收特定类型或特定任务的事件。

```go
// 只接收工具调用相关事件
ch, cancel := bus.SubscribeFiltered(func(e core.ReactEvent) bool {
    return e.Type == core.ActionStart || e.Type == core.ActionResult
})

// 只接收特定 SubAgent 的事件
ch, cancel := bus.SubscribeFiltered(func(e core.ReactEvent) bool {
    return e.TaskID == "task_1"
})
```

注意：`SubscribeFiltered` 的 filter 函数在 `Emit` 的读锁内执行，应保持轻量。如果 filter 逻辑复杂，建议使用全量订阅后在消费端过滤。

---

## 3. 流式输出实现

GoReAct 的 Think 阶段使用流式 LLM 调用，每收到一个文本片段就通过 EventBus 发出 `ThinkingDelta` 事件，使客户端能够实时展示推理过程。

以下是一个完整的流式输出示例：

```go
package main

import (
    "context"
    "fmt"

    "github.com/DotNetAge/goreact/core"
    "github.com/DotNetAge/goreact/reactor"
)

func main() {
    bus := reactor.NewEventBus()
    ch, cancel := bus.Subscribe()
    defer cancel()

    // 启动事件消费 goroutine
    go func() {
        for event := range ch {
            switch event.Type {
            case core.ThinkingDelta:
                // 实时输出思维片段，实现打字机效果
                fmt.Print(event.Data)

            case core.ThinkingDone:
                fmt.Println("\n--- 思维完成 ---")

            case core.ActionStart:
                data := event.Data.(core.ActionStartData)
                fmt.Printf("\n[调用工具: %s]\n", data.ToolName)
                // 如需展示参数，可格式化 data.Params

            case core.ActionProgress:
                fmt.Print(event.Data) // 进度文本

            case core.ActionResult:
                data := event.Data.(core.ActionResultData)
                if data.Success {
                    fmt.Printf("[完成: %s, 耗时: %v]\n", data.ToolName, data.Duration)
                } else {
                    fmt.Printf("[失败: %s, 错误: %s]\n", data.ToolName, data.Error)
                }

            case core.ObservationDone:
                // 可在此处展示观察结果摘要

            case core.SubtaskSpawned:
                data := event.Data.(core.SubtaskInfo)
                fmt.Printf("\n[子任务创建: %s - %s]\n", data.TaskID, data.Description)

            case core.SubtaskCompleted:
                data := event.Data.(core.SubtaskResult)
                if data.Success {
                    fmt.Printf("[子任务完成: %s]\n", data.TaskID)
                } else {
                    fmt.Printf("[子任务失败: %s - %s]\n", data.TaskID, data.Error)
                }

            case core.CycleEnd:
                data := event.Data.(core.CycleInfo)
                fmt.Printf("\n[循环 #%d 结束, 耗时: %v]\n", data.Iteration, data.Duration)

            case core.FinalAnswer:
                fmt.Printf("\n最终答案: %s\n", event.Data)

            case core.ExecutionSummary:
                data := event.Data.(core.ExecutionSummaryData)
                fmt.Printf("\n=== 执行摘要 ===\n")
                fmt.Printf("迭代次数: %d\n", data.TotalIterations)
                fmt.Printf("Token 消耗: %d\n", data.TokensUsed)
                fmt.Printf("工具调用: %d 次\n", data.ToolCalls)
                fmt.Printf("使用工具: %v\n", data.ToolsUsed)
                fmt.Printf("总耗时: %v\n", data.TotalDuration)
                fmt.Printf("终止原因: %s\n", data.TerminationReason)

            case core.ExperienceSaved:
                data := event.Data.(core.ExperienceSavedData)
                fmt.Printf("\n[经验已保存: %s, 迭代: %d, 工具: %v]\n",
                    data.Problem, data.Iterations, data.ToolsUsed)

            case core.Error:
                fmt.Printf("\n[错误: %s]\n", event.Data)

            case core.ClarifyNeeded:
                fmt.Printf("\n[需要澄清: %s]\n", event.Data)

            case core.PermissionRequest:
                data := event.Data.(core.PermissionRequestData)
                fmt.Printf("\n[请求授权: %s, 安全等级: %d, 原因: %s]\n",
                    data.ToolName, data.SecurityLevel, data.Reason)
            }
        }
    }()

    r := reactor.NewReactor(
        reactor.ReactorConfig{
            APIKey: "your-api-key",
            Model:  "qwen3.5-flash",
        },
        reactor.WithEventBus(bus),
    )

    result, err := r.Run(context.Background(), "帮我写一个 Go 的 HTTP server", nil)
    if err != nil {
        panic(err)
    }
    _ = result
}
```

关键要点：事件消费必须在独立 goroutine 中进行，因为 `r.Run` 是阻塞调用。`cancel()` 应在 `Run` 返回后调用，确保事件 channel 被正确关闭。

---

## 4. Token 消耗追踪

GoReAct 提供两种 Token 追踪机制，分别适用于不同场景。

### 4.1 RunResult（同步结果）

`Run` 方法返回的 `RunResult` 包含基本的 Token 信息：

```go
type RunResult struct {
    Answer            string        `json:"answer"`
    Intent            *Intent       `json:"intent,omitempty"`
    Steps             []Step        `json:"steps,omitempty"`
    TotalIterations   int           `json:"total_iterations"`
    TerminationReason string        `json:"termination_reason,omitempty"`
    Confidence        float64       `json:"confidence"`
    TokensUsed        int           `json:"tokens_used"`
    TotalDuration     time.Duration `json:"total_duration_ms"`
}
```

直接从返回值获取：

```go
result, err := r.Run(ctx, "问题", nil)
if err != nil { /* ... */ }

fmt.Printf("Token 消耗: %d\n", result.TokensUsed)
fmt.Printf("T-A-O 循环: %d 次\n", result.TotalIterations)
fmt.Printf("总耗时: %v\n", result.TotalDuration)
```

`TokensUsed` 包含了意图分类和所有 T-A-O 循环中 LLM 调用的 Token 总量。

### 4.2 ExecutionSummary 事件（异步统计）

`ExecutionSummary` 事件在 `Run` 即将返回前发出，提供更详细的统计信息：

```go
type ExecutionSummaryData struct {
    TotalIterations   int           `json:"total_iterations"`   // T-A-O 循环总次数
    ToolCalls         int           `json:"tool_calls"`         // 工具调用总次数（含重复）
    ToolsUsed         []string      `json:"tools_used"`         // 去重后的工具名称列表
    TotalDuration     time.Duration `json:"total_duration_ms"`  // Run 从开始到结束的总耗时
    TokensUsed        int           `json:"tokens_used"`        // Token 总消耗
    TerminationReason string        `json:"termination_reason"` // 终止原因
}
```

与 `RunResult.TokensUsed` 的区别：`ExecutionSummary` 额外提供了 `ToolCalls`（工具调用次数）和 `ToolsUsed`（去重工具列表），适合用于日志记录和监控。

### 4.3 在多轮对话中追踪

当使用 `Agent` 封装时，`ContextWindow` 也会累计 Token 消耗：

```go
agent := goreact.NewAgentWithSession(config, model, memory, r, "session-1", 8192)
answer, _ := agent.Ask("问题")

// 查看 ContextWindow 中的 Token 使用情况
cw := agent.ContextWindow()
fmt.Printf("已使用: %d, 剩余: %d, 上限: %d\n",
    cw.TokensUsed, cw.TokensRemaining(), cw.MaxTokens)
```

---

## 5. ContextWindow 使用

`ContextWindow` 是 `core` 包中提供的多轮对话上下文管理器，`Agent` 内部使用它来管理会话历史和 Token 预算。

### 5.1 基本使用

```go
// 创建上下文窗口，设置 Token 上限为 8192
cw := core.NewContextWindow("session-1", 8192)

// 添加消息
cw.AddMessage("user", "你好")
cw.AddMessage("assistant", "你好，有什么可以帮助你的？")

// 获取最近 N 条消息（0 = 全部）
recent := cw.RecentMessages(10)

// 获取全部消息
all := cw.GetMessages()
```

### 5.2 Token 预算管理

```go
cw := core.NewContextWindow("session-1", 8192)

// 累计 Token 消耗
cw.AddTokens(1500)

// 查看剩余 Token
remaining := cw.TokensRemaining() // 返回 int64

// 检查是否超预算
if cw.TokensRemaining() <= 0 {
    // 触发裁剪
}
```

### 5.3 Prune（裁剪）

当 Token 预算耗尽时，`Prune` 会从最早的消息开始移除，直到总 Token 估算值在预算范围内。它始终保留最后 2 条消息（1 条 user + 1 条 assistant）。

```go
// 使用默认估算函数（约 3 字符/token）
cw.Prune(nil)

// 使用自定义 Token 估算函数
cw.Prune(func(content string) int {
    // 例如使用 tiktoken 或其他分词器
    return estimateTokens(content)
})
```

### 5.4 在 Agent 中的自动化管理

`Agent.Ask` 方法内部自动管理 `ContextWindow`：

```go
agent := goreact.NewAgentWithSession(config, model, memory, r, "session-1", 8192)

// 每次调用 Ask，Agent 自动：
// 1. 将用户输入追加到 ContextWindow
// 2. 将历史消息传递给 Reactor
// 3. 将回答追加到 ContextWindow
// 4. 累加 Token 消耗
// 5. 如果超预算，自动调用 Prune
answer1, _ := agent.Ask("第一个问题")
answer2, _ := agent.Ask("第二个问题")  // 自动携带之前的上下文
```

### 5.5 会话重置

```go
// 清空当前会话的所有消息和 Token 计数
cw.Reset()

// 或者创建一个全新会话
agent.NewSession("session-2", 16384)
```

### 5.6 线程安全

`ContextWindow` 的所有方法都是 goroutine 安全的，内部使用 `sync.RWMutex` 保护。

---

## 6. 中断-恢复模式

GoReAct 提供两个内置的中断-恢复交互工具：`ask_user`（多轮澄清）和 `ask_permission`（工具授权）。两者都基于 channel 阻塞机制实现，允许外部代码在 T-A-O 循环暂停时介入，并通过 `Respond` 恢复执行。

### 6.1 ask_user（多轮澄清）

当 LLM 在 Think 阶段判断需要用户补充信息时，它会调用 `ask_user` 工具。该工具会：

1. 发出 `ClarifyNeeded` 事件（携带问题文本）
2. 阻塞当前 T-A-O 循环，等待用户回复
3. 用户通过 `Respond(answer)` 提供答案后，工具返回答案作为结果
4. T-A-O 循环继续执行，LLM 在下一轮 Think 中看到用户回答

**基本用法：**

```go
r := reactor.NewReactor(config, reactor.WithEventBus(bus))

// 在另一个 goroutine 中监听事件并处理中断
go func() {
    for event := range eventCh {
        if event.Type == core.ClarifyNeeded {
            question := event.Data.(string)
            fmt.Printf("\nAgent 提问: %s\n", question)

            // 获取用户输入
            answer := getUserInput(question)

            // 恢复执行
            r.AskUser().Respond(answer)
        }
    }
}()

result, _ := r.Run(ctx, "帮我订一张机票", nil)
```

**轮询方式（适用于非事件驱动场景）：**

```go
// 启动 Run 在 goroutine 中执行
var result *reactor.RunResult
var runErr error
done := make(chan struct{})
go func() {
    result, runErr = r.Run(ctx, "帮我订一张机票", nil)
    close(done)
}()

// 轮询检查是否需要用户输入
for {
    if r.AskUser().IsWaiting() {
        // 注意：AskUser 本身没有直接获取问题的方法，
        // 问题通过 ClarifyNeeded 事件传递。
        // 如果使用轮询模式，需要配合 EventBus 或其他机制获取问题内容。
        answer := getUserInput("请回答 Agent 的问题")
        r.AskUser().Respond(answer)
    }
    select {
    case <-done:
        goto finished
    default:
        time.Sleep(100 * time.Millisecond)
    }
}
finished:
```

**错误处理：**

```go
// 如果用户取消或出现错误
r.AskUser().RespondError(fmt.Errorf("用户取消了请求"))

// 等待 ask_user 完成（带超时）
err := r.AskUser().WaitWithTimeout(30 * time.Second)
if err != nil {
    fmt.Println("等待超时:", err)
}
```

### 6.2 ask_permission（工具授权）

当 LLM 决定调用一个安全等级为 `LevelSensitive` 或 `LevelHighRisk` 的工具时，权限检查系统会暂停工具执行，发出 `PermissionRequest` 事件，等待用户批准或拒绝。

**权限判断逻辑：**

- `LevelSafe` + `IsReadOnly`：自动放行，不触发授权流程
- `LevelSensitive`：需要用户确认
- `LevelHighRisk`：必须用户明确授权

**基本用法：**

```go
r := reactor.NewReactor(config, reactor.WithEventBus(bus))

// 监听授权请求
go func() {
    for event := range eventCh {
        if event.Type == core.PermissionRequest {
            data := event.Data.(core.PermissionRequestData)

            fmt.Printf("\n工具授权请求:\n")
            fmt.Printf("  工具: %s\n", data.ToolName)
            fmt.Printf("  参数: %v\n", data.Params)
            fmt.Printf("  安全等级: %d\n", data.SecurityLevel)
            fmt.Printf("  原因: %s\n", data.Reason)

            // 获取用户决定
            approved := getUserApproval(data.ToolName, data.Params)

            if approved {
                r.AskPermission().Respond(core.PermissionResult{
                    Behavior: core.PermissionAllow,
                })
            } else {
                r.AskPermission().Respond(core.PermissionResult{
                    Behavior: core.PermissionDeny,
                    Message:  "用户拒绝了此操作",
                })
            }
        }
    }
}()

result, _ := r.Run(ctx, "删除 /tmp 目录下的所有文件", nil)
```

**高级用法 -- 修改工具参数：**

```go
// 用户可以批准操作但同时修改参数
r.AskPermission().Respond(core.PermissionResult{
    Behavior: core.PermissionAllow,
    Message:  "已批准，但修改了参数",
    UpdatedInput: map[string]any{
        "path": "/tmp/my-project", // 限制操作范围
    },
})
```

**错误处理：**

```go
// 授权超时
r.AskPermission().RespondError(fmt.Errorf("授权超时"))

// 等待授权完成
err := r.AskPermission().WaitWithTimeout(30 * time.Second)
```

### 6.3 同时处理 ask_user 和 ask_permission

在实际应用中，两者可能在同一次 `Run` 中交替出现。建议在事件循环中统一处理：

```go
go func() {
    for event := range eventCh {
        switch event.Type {
        case core.ClarifyNeeded:
            question := event.Data.(string)
            answer := promptUser(question)
            r.AskUser().Respond(answer)

        case core.PermissionRequest:
            data := event.Data.(core.PermissionRequestData)
            handlePermissionRequest(r, data)
        }
    }
}()
```

---

## 7. Agent 封装使用

`Agent` 是 GoReAct 提供的高层封装，将 `Reactor`、`ContextWindow`、`Memory` 整合为一个简洁的多轮对话接口。

### 7.1 创建 Agent

```go
import (
    "github.com/DotNetAge/goreact"
    "github.com/DotNetAge/goreact/core"
    "github.com/DotNetAge/goreact/reactor"
)

// 准备配置
agentConfig := &core.AgentConfig{
    Name:        "my-agent",
    Domain:      "general",
    Description: "通用助手",
}
modelConfig := &core.ModelConfig{
    Model:       "qwen3.5-flash",
    Temperature: 0.7,
}
memory := core.NewInMemoryMemory()
r := reactor.NewReactor(reactor.ReactorConfig{
    APIKey: "your-api-key",
    Model:  "qwen3.5-flash",
}, reactor.WithMemory(memory))

// 创建带会话的 Agent
agent := goreact.NewAgentWithSession(
    agentConfig, modelConfig, memory, r,
    "session-abc", 8192, // sessionID, maxTokens
)
```

### 7.2 多轮对话

```go
answer1, err := agent.Ask("我叫小明")
if err != nil { /* ... */ }
fmt.Println(answer1)

answer2, err := agent.Ask("我叫什么名字？")
fmt.Println(answer2) // ContextWindow 携带历史，Agent 能回答"小明"
```

### 7.3 带取消的调用

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

answer, err := agent.AskWithContext(ctx, "复杂问题")
```

### 7.4 会话管理

```go
// 获取当前会话信息
sessionID := agent.SessionID()
msgCount := agent.ContextWindow().MessageCount()

// 切换到新会话（旧会话丢弃）
agent.NewSession("session-xyz", 16384)

// 查看剩余 Token
remaining := agent.ContextWindow().TokensRemaining()
```

### 7.5 不使用 ContextWindow

如果只需要单次推理而不需要多轮对话管理，可以使用不带会话的 `NewAgent`：

```go
agent := goreact.NewAgent(agentConfig, modelConfig, memory, r)
answer, _ := agent.Ask("单次问题")
// 不会自动管理历史
```

---

## 8. 团队通信

GoReAct 通过 `AgentMessageBus` 实现多 Agent 团队协作。`MessageBus` 使用基于 channel 的异步消息传递模型，每个 Agent 拥有独立的邮箱（mailbox）。

### 8.1 创建和管理团队

```go
// 创建 MessageBus（通常通过 Reactor 选项共享）
bus := core.NewAgentMessageBus()
r := reactor.NewReactor(config, reactor.WithMessageBus(bus))

// 或者使用 Reactor 自带的 MessageBus
bus = r.MessageBus()

// 创建团队
team, err := bus.CreateTeam("research-team", "负责调研的团队")

// 查看团队
team, err = bus.GetTeam(team.ID)
fmt.Printf("团队: %s, 成员: %d\n", team.Name, len(team.Members))

// 列出所有团队
teams := bus.ListTeams()

// 删除团队（同时关闭所有成员邮箱）
err = bus.DeleteTeam(team.ID)
```

### 8.2 加入和离开团队

```go
// Agent 加入团队
err := bus.JoinTeam(team.ID, "agent-alice", "task_1")

// Agent 离开团队（邮箱保留，可继续读取未读消息）
err = bus.LeaveTeam(team.ID, "agent-alice")
```

### 8.3 发送消息

```go
// 点对点消息
msg, err := bus.SendMessage(
    team.ID,       // 团队 ID
    "agent-alice", // 发送者
    "agent-bob",   // 接收者
    core.MessageDirect, // 消息类型
    "请分析这份报告",   // 内容
    "分析报告请求",     // 摘要（用于上下文效率）
)

// 广播消息（发送给除自己外的所有成员）
msg, err := bus.SendMessage(
    team.ID,
    "agent-alice",
    "", // 空接收者 = 广播
    core.MessageBroadcast,
    "团队任务已完成",
    "任务完成通知",
)

// 关机请求
msg, err := bus.SendMessage(
    team.ID, "agent-alice", "agent-bob",
    core.MessageShutdownRequest,
    "工作完成，请关闭",
    "关机请求",
)
```

### 8.4 接收消息

```go
// 非阻塞读取所有待处理消息
messages := bus.ReceiveMessages("agent-bob")
for _, msg := range messages {
    fmt.Printf("来自 %s: %s\n", msg.From, msg.Content)
}

// 阻塞等待新消息（通过 channel）
mailbox, err := bus.WaitMailbox("agent-bob")
msg := <-mailbox // 阻塞直到收到消息
```

### 8.5 更新成员状态

```go
// 更新成员执行状态
err = bus.UpdateMemberStatus(team.ID, "agent-bob", "completed", "分析结果: ...")
```

### 8.6 在 SubAgent 中使用团队通信

当 LLM 使用内置的 `subagent`、`send_message`、`receive_messages` 等工具时，Reactor 内部自动通过共享的 `MessageBus` 完成团队通信。集成开发者只需确保同一个 `MessageBus` 实例被注入到主 Reactor 中，SubAgent 会自动继承：

```go
// 创建共享的 MessageBus
bus := core.NewAgentMessageBus()

// 主 Reactor 使用该 bus
mainReactor := reactor.NewReactor(config, reactor.WithMessageBus(bus))

// Run 时如果 LLM 决定创建团队和 SubAgent，
// SubAgent 会自动使用同一个 bus 进行通信
result, _ := mainReactor.Run(ctx, "组织一个团队调研 AI 技术趋势", nil)
```

---

## 9. 完整集成示例

以下示例展示了一个完整的集成场景：带流式输出、中断-恢复交互、Token 预算管理的多轮对话 Agent。

```go
package main

import (
    "context"
    "fmt"
    "os"
    "bufio"
    "time"

    "github.com/DotNetAge/goreact"
    "github.com/DotNetAge/goreact/core"
    "github.com/DotNetAge/goreact/reactor"
)

func main() {
    // === 1. 创建事件总线 ===
    bus := reactor.NewEventBus()

    // === 2. 创建 Reactor ===
    memory := core.NewInMemoryMemory()
    r := reactor.NewReactor(
        reactor.ReactorConfig{
            APIKey: os.Getenv("LLM_API_KEY"),
            Model:  "qwen3.5-flash",
        },
        reactor.WithMemory(memory),
        reactor.WithEventBus(bus),
    )

    // === 3. 创建带会话的 Agent ===
    agent := goreact.NewAgentWithSession(
        &core.AgentConfig{
            Name:        "assistant",
            Domain:      "general",
            Description: "通用智能助手",
        },
        &core.ModelConfig{
            Model:       "qwen3.5-flash",
            Temperature: 0.7,
        },
        memory, r,
        "cli-session", 32768,
    )

    // === 4. 启动事件消费 ===
    eventCh, cancel := bus.Subscribe()
    defer cancel()

    go consumeEvents(eventCh, r)

    // === 5. 交互式对话循环 ===
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Print("\n> ")
        input, _ := reader.ReadString('\n')
        if len(input) <= 1 {
            continue
        }

        ctx, cancelCtx := context.WithTimeout(context.Background(), 120*time.Second)

        answer, err := agent.AskWithContext(ctx, input[:len(input)-1])
        cancelCtx()

        if err != nil {
            fmt.Printf("错误: %v\n", err)
        } else {
            fmt.Printf("\n%s\n", answer)
        }

        // 显示当前 Token 使用情况
        cw := agent.ContextWindow()
        fmt.Printf("\n--- Token: %d / %d (剩余: %d) ---\n",
            cw.TokensUsed, cw.MaxTokens, cw.TokensRemaining())

        // 切换会话命令
        if input[:len(input)-1] == "/new" {
            agent.NewSession(fmt.Sprintf("session-%d", time.Now().Unix()), 32768)
            fmt.Println("已创建新会话")
        }
    }
}

func consumeEvents(ch <-chan core.ReactEvent, r *reactor.Reactor) {
    for event := range ch {
        switch event.Type {
        case core.ThinkingDelta:
            fmt.Print(event.Data)

        case core.ActionStart:
            data := event.Data.(core.ActionStartData)
            fmt.Printf("\n  >> 调用: %s\n", data.ToolName)

        case core.ActionResult:
            data := event.Data.(core.ActionResultData)
            if data.Success {
                fmt.Printf("  >> 完成: %s (%v)\n", data.ToolName, data.Duration)
            } else {
                fmt.Printf("  >> 失败: %s - %s\n", data.ToolName, data.Error)
            }

        case core.ClarifyNeeded:
            // ask_user 触发 -- 在终端直接获取输入
            question := event.Data.(string)
            fmt.Printf("\n  [需要确认] %s\n> ", question)
            reader := bufio.NewReader(os.Stdin)
            answer, _ := reader.ReadString('\n')
            if len(answer) > 1 {
                r.AskUser().Respond(answer[:len(answer)-1])
            }

        case core.PermissionRequest:
            // ask_permission 触发
            data := event.Data.(core.PermissionRequestData)
            fmt.Printf("\n  [授权请求] 工具: %s, 等级: %d\n", data.ToolName, data.SecurityLevel)
            fmt.Printf("  参数: %v\n", data.Params)
            fmt.Printf("  允许执行? (y/n): ")
            reader := bufio.NewReader(os.Stdin)
            answer, _ := reader.ReadString('\n')
            if len(answer) > 0 && answer[0] == 'y' {
                r.AskPermission().Respond(core.PermissionResult{
                    Behavior: core.PermissionAllow,
                })
            } else {
                r.AskPermission().Respond(core.PermissionResult{
                    Behavior: core.PermissionDeny,
                    Message:  "用户拒绝",
                })
            }

        case core.SubtaskSpawned:
            data := event.Data.(core.SubtaskInfo)
            fmt.Printf("\n  [子任务] %s: %s\n", data.TaskID, data.Description)

        case core.SubtaskCompleted:
            data := event.Data.(core.SubtaskResult)
            status := "完成"
            if !data.Success {
                status = "失败: " + data.Error
            }
            fmt.Printf("  [子任务] %s: %s\n", data.TaskID, status)

        case core.ExecutionSummary:
            data := event.Data.(core.ExecutionSummaryData)
            fmt.Printf("\n  [摘要] 迭代: %d, Token: %d, 工具: %v, 耗时: %v\n",
                data.TotalIterations, data.TokensUsed, data.ToolsUsed, data.TotalDuration)

        case core.ExperienceSaved:
            data := event.Data.(core.ExperienceSavedData)
            fmt.Printf("  [经验] 已保存: %q (迭代: %d)\n", data.Problem, data.Iterations)

        case core.Error:
            fmt.Printf("\n  [错误] %s\n", event.Data)

        case core.CycleEnd:
            data := event.Data.(core.CycleInfo)
            fmt.Printf("  [循环 #%d] %v\n", data.Iteration, data.Duration)
        }
    }
}
```

这个示例涵盖了 GoReAct 集成的所有核心特性：事件订阅与流式输出、中断-恢复交互（ask_user + ask_permission）、多轮对话与 Token 预算管理、子任务监控以及经验保存追踪。开发者可以基于此模式，将 GoReAct 集成到 Web 服务、CLI 工具、聊天应用等不同形态的产品中。
