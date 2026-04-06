# 开发指南

当您将 GoReAct 集成到 Web 后台、IM 机器人或企业内部系统中时，您需要与框架进行深度的实时交互。本指南涵盖了开发者最关注的 7 个核心场景：流式输出、任务挂起与恢复、记忆管理以及全景状态监控。

## 获取实时的流式思考过程

在 ReAct 循环中，Agent 的思考（Thought）和工具执行状态对用户体验至关重要。GoReAct 提供了事件流（Event Stream）和探针（Hook）两种方式来捕获这些过程。

**方式 A：通过 AskStream 获取实时事件（推荐用于 Web UI/SSE）**

```go
// 使用 AskStream 替代 Ask
stream, err := agent.AskStream(ctx, "帮我分析这段代码并搜索相关漏洞", "main.go")
if err != nil {
    return err
}

for event := range stream {
    switch e := event.(type) {
    case *goreact.ThoughtEvent:
        fmt.Printf("[思考] %s (置信度: %.2f)\n", e.Reasoning, e.Confidence)
    case *goreact.ToolCallEvent:
        fmt.Printf("[执行] 正在调用工具: %s, 参数: %v\n", e.ToolName, e.Params)
    case *goreact.ToolResultEvent:
        fmt.Printf("[观察] 工具返回结果耗时: %v\n", e.Duration)
    }
}
```

## 获取流式文本输出

当模型最终决定回复用户（Chat/Answer 意图）时，为了降低 TTFB（首字点时间），您可以通过解析 `OutputEvent` 来获取流式的文本块（Chunks）。

```go
stream, _ := agent.AskStream(ctx, "介绍一下你自己")

for event := range stream {
    if e, ok := event.(*goreact.OutputEvent); ok {
        // 将流式文本块实时推送到前端（如通过 WebSocket / Server-Sent Events）
        fmt.Print(e.TextChunk)
    }
    
    if e, ok := event.(*goreact.FinalResultEvent); ok {
        fmt.Println("\n[输出完成] 最终结果:", e.Answer)
    }
}
```

## 获取等待用户处理的任务

当 Agent 遇到 `LevelHighRisk`（高危工具）或由于参数缺失而暂停时，会话会被冻结并转为 Pending 状态。

```go
// 检索整个系统中所有正在等待用户输入/授权的会话
pendingTasks, err := engine.ListPendingTasks(ctx)
if err != nil {
    panic(err)
}

for _, task := range pendingTasks {
    fmt.Printf("会话 ID: %s\n", task.SessionName)
    fmt.Printf("挂起原因: %s\n", task.Question.Type) // 如 Authorization, Clarification
    fmt.Printf("询问内容: %s\n", task.Question.Question)
    fmt.Printf("可选回答: %v\n", task.Question.Options)
}
```

## 将等待处理的任务重新执行

用户在前端点击“授权”或补充了信息后，后端需要将答案回传给引擎，唤醒被冻结的会话。

```go
sessionName := "session-uuid-1234"
userAnswer := "yes" // 或者用户补充的具体参数

// 唤醒并继续执行，Resume 同样支持返回流式结果 (ResumeStream)
result, err := agent.Resume(ctx, sessionName, userAnswer)
if err != nil {
    panic(err)
}

if result.Status == goreact.StatusCompleted {
    fmt.Println("任务已恢复并执行完成:", result.Answer)
}
```

## 添加/编辑/删除短期记忆

短期记忆（Short-term Memory）用于存储当前会话中的重要偏好和事实，PromptBuilder 会将其动态注入到后续对话的上下文中。虽然系统会自动提取，但开发者也可以通过 API 手动干预。

```go
memory := engine.GetMemory()
sessionName := "session-uuid-1234"

// 1. 添加短期记忆
item, err := memory.AddShortTermMemoryItem(ctx, sessionName, &goreact.MemoryItem{
    Type:          goreact.MemoryPreference,
    Content:       "用户偏好使用 Markdown 格式输出",
    Importance:    0.9,
    EmphasisLevel: goreact.EmphasisImportant,
})

// 2. 编辑短期记忆
item.Content = "用户偏好使用 JSON 格式输出"
err = memory.UpdateShortTermMemoryItem(ctx, item)

// 3. 删除短期记忆
err = memory.RemoveShortTermMemoryItem(ctx, sessionName, item.ID)

// 4. 获取当前会话的所有短期记忆
items, _ := memory.GetShortTermMemoryItems(ctx, sessionName)
for _, i := range items {
    fmt.Println("-", i.Content)
}
```

## 获取与操作会话

Session 是对话和上下文的最小隔离单元。您可以检索会话历史、清空会话或获取会话内的特定变量。

```go
memory := engine.GetMemory()

// 1. 获取会话历史记录（用于渲染聊天界面）
history, err := memory.GetSessionHistory(ctx, "session-uuid-1234")
for _, msg := range history.Messages {
    fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
}

// 2. 获取系统中所有的活跃会话
activeSessions, _ := memory.ListSessions(ctx, &goreact.SessionFilter{
    Status: goreact.SessionActive,
    Limit:  50,
})

// 3. 彻底删除会话及其相关上下文（不会删除已固化的长期记忆）
err = memory.DeleteSession(ctx, "session-uuid-1234")
```

## 任务状态监控与完成简报

对于后台运行的长时间任务（例如深入的代码审查或多 Agent 并行搜集），开发者需要了解任务的全景状态，并在完成后提取结构化的简报（Summary）。

```go
// 1. 获取任务实时状态
state, err := engine.GetTaskState(ctx, "session-uuid-1234")
fmt.Printf("当前阶段: %s\n", state.ExecutionPhase) // 如 Planning, Executing, Aggregating
fmt.Printf("当前计划进度: %d/%d\n", state.CurrentPlanStep, len(state.Plan.Steps))

// 2. 任务完成后，获取轨迹(Trajectory)与简报(Summary)
trajectory, err := memory.GetTrajectory(ctx, "session-uuid-1234")
if err == nil && trajectory != nil {
    fmt.Println("=== 任务完成简报 ===")
    // Summary 是 LLM 对整个执行轨迹的提炼总结
    fmt.Println(trajectory.Summary)
    
    fmt.Println("=== 关键决策点 ===")
    for _, decision := range trajectory.ExtractKeyDecisions() {
        fmt.Printf("步骤 %d: 决定 %s, 原因: %s\n", 
            decision.Step, decision.Action, decision.Reasoning)
    }
}

// 3. 多 Agent 编排时的全局状态
orchestratorStatus := engine.GetOrchestrationStatus(ctx, "task-uuid-5678")
fmt.Printf("总子任务: %d, 已完成: %d, 失败: %d\n", 
    orchestratorStatus.TotalSubTasks, 
    orchestratorStatus.CompletedCount,
    orchestratorStatus.FailedCount,
)
```

通过以上 API，您可以轻松地将 GoReAct 的核心能力以非阻塞、响应式的方式集成到您的前端交互层或后端控制台中。