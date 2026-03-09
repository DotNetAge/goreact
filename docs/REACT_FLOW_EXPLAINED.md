# GoReAct 执行流程详解

## 概述

GoReAct 框架实现了标准的 ReAct (Reasoning + Acting) 循环模式。本文档详细说明工具如何装配给 LLM、如何获取调用列表、如何执行工具以及如何反馈结果。

## 完整执行流程

```
┌─────────────────────────────────────────────────────────────┐
│                    Engine.Execute(task)                      │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
         ┌────────────────────────┐
         │  1. 初始化 & 准备      │
         │  - 创建 Context        │
         │  - 选择 Skill (可选)   │
         │  - 注入 Skill 指令     │
         └────────┬───────────────┘
                  │
                  ▼
         ┌────────────────────────┐
         │  2. 开始 ReAct 循环    │
         └────────┬───────────────┘
                  │
                  ▼
    ┌─────────────────────────────────────┐
    │         迭代 N (循环开始)            │
    └─────────────┬───────────────────────┘
                  │
                  ▼
    ┌─────────────────────────────────────┐
    │  Step 1: Think (思考)                │
    │  ┌───────────────────────────────┐  │
    │  │ Thinker.Think(task, context)  │  │
    │  └───────────┬───────────────────┘  │
    │              │                       │
    │              ▼                       │
    │  ┌───────────────────────────────┐  │
    │  │ 构建 Prompt:                  │  │
    │  │ - Task 描述                   │  │
    │  │ - 工具列表 (toolDesc) ✅      │  │
    │  │ - 历史记录 (上次结果)         │  │
    │  │ - Skill 指令 (如果有)         │  │
    │  └───────────┬───────────────────┘  │
    │              │                       │
    │              ▼                       │
    │  ┌───────────────────────────────┐  │
    │  │ LLM.Generate(prompt) ✅       │  │
    │  └───────────┬───────────────────┘  │
    │              │                       │
    │              ▼                       │
    │  ┌───────────────────────────────┐  │
    │  │ 解析 LLM 响应:                │  │
    │  │ - Thought (推理)              │  │
    │  │ - Action (工具名称) ✅        │  │
    │  │ - Parameters (参数) ✅        │  │
    │  │ - Final Answer (最终答案)     │  │
    │  └───────────┬───────────────────┘  │
    │              │                       │
    │              ▼                       │
    │         返回 Thought                 │
    └─────────────┬───────────────────────┘
                  │
                  ▼
         ┌────────────────────┐
         │  检查是否完成？     │
         └────┬───────────┬───┘
              │           │
         Yes  │           │ No
              │           │
              ▼           ▼
    ┌─────────────┐  ┌──────────────────────────┐
    │  返回结果    │  │  Step 2: Act (行动)       │
    │  (结束)     │  │  ┌────────────────────┐  │
    └─────────────┘  │  │ Actor.Act(action)  │  │
                     │  └────────┬───────────┘  │
                     │           │               │
                     │           ▼               │
                     │  ┌────────────────────┐  │
                     │  │ 查找工具 ✅        │  │
                     │  │ ToolManager.Get()  │  │
                     │  └────────┬───────────┘  │
                     │           │               │
                     │           ▼               │
                     │  ┌────────────────────┐  │
                     │  │ 执行工具 ✅        │  │
                     │  │ Tool.Execute()     │  │
                     │  └────────┬───────────┘  │
                     │           │               │
                     │           ▼               │
                     │      返回 ExecutionResult │
                     └──────────┬────────────────┘
                                │
                                ▼
                     ┌──────────────────────────┐
                     │  Step 3: Observe (观察)   │
                     │  ┌────────────────────┐  │
                     │  │ Observer.Observe() │  │
                     │  └────────┬───────────┘  │
                     │           │               │
                     │           ▼               │
                     │  ┌────────────────────┐  │
                     │  │ 分析执行结果        │  │
                     │  │ 生成反馈信息        │  │
                     │  └────────┬───────────┘  │
                     │           │               │
                     │           ▼               │
                     │      返回 Feedback        │
                     └──────────┬────────────────┘
                                │
                                ▼
                     ┌──────────────────────────┐
                     │  Step 4: 更新 Context ✅  │
                     │  - last_action           │
                     │  - last_result           │
                     │  - last_feedback         │
                     └──────────┬────────────────┘
                                │
                                ▼
                     ┌──────────────────────────┐
                     │  循环控制检查             │
                     │  - 是否达到最大迭代？     │
                     │  - 是否应该继续？         │
                     └──────────┬────────────────┘
                                │
                                ▼
                          回到迭代开始 ↑
```

## 详细步骤说明

### 1. 工具装配给 LLM ✅

**位置**: `pkg/core/thinker/simple_thinker.go:68-94`

```go
func (t *SimpleThinker) buildPrompt(task string, context *core.Context) string {
    var sb strings.Builder

    sb.WriteString("You are a helpful AI assistant that uses tools to solve tasks.\n\n")

    // ✅ 工具描述被包含在 prompt 中
    sb.WriteString(t.toolDesc)  // 这里包含所有可用工具的描述
    sb.WriteString("\n\n")

    sb.WriteString("Task: " + task + "\n\n")

    // 添加历史记录（上次执行的结果）
    if history, ok := context.Get("history"); ok {
        sb.WriteString("Previous steps:\n")
        sb.WriteString(historyStr)
    }

    // 告诉 LLM 如何响应
    sb.WriteString("Please respond in the following format:\n")
    sb.WriteString("Thought: [your reasoning]\n")
    sb.WriteString("Action: [tool_name]\n")
    sb.WriteString("Parameters: {\"key\": \"value\"}\n")

    return sb.String()
}
```

**工具描述来源**:
- `Engine.New()` 时从 `ToolManager.GetToolDescriptions()` 获取
- 包含所有已注册工具的名称、描述和参数说明
- 每次注册新工具时会更新

### 2. 获取 LLM 的工具调用列表 ✅

**位置**: `pkg/core/thinker/simple_thinker.go:97-145`

```go
func (t *SimpleThinker) parseResponse(response string) *types.Thought {
    thought := &types.Thought{}

    // ✅ 解析 LLM 返回的推理
    thoughtRegex := regexp.MustCompile(`(?i)Thought:\s*(.+?)(?:\n|$)`)
    if matches := thoughtRegex.FindStringSubmatch(response); len(matches) > 1 {
        thought.Reasoning = strings.TrimSpace(matches[1])
    }

    // 检查是否是最终答案
    finalAnswerRegex := regexp.MustCompile(`(?i)Final Answer:\s*(.+?)(?:\n|$)`)
    if matches := finalAnswerRegex.FindStringSubmatch(response); len(matches) > 1 {
        thought.ShouldFinish = true
        thought.FinalAnswer = strings.TrimSpace(matches[1])
        return thought
    }

    // ✅ 解析工具调用
    actionRegex := regexp.MustCompile(`(?i)Action:\s*(.+?)(?:\n|$)`)
    if matches := actionRegex.FindStringSubmatch(response); len(matches) > 1 {
        actionName := strings.TrimSpace(matches[1])

        action := &types.Action{
            ToolName: actionName,  // ✅ 工具名称
        }

        // ✅ 解析参数
        paramsRegex := regexp.MustCompile(`(?i)Parameters:\s*(\{.+?\})`)
        if paramMatches := paramsRegex.FindStringSubmatch(response); len(paramMatches) > 1 {
            action.Parameters = t.parseSimpleJSON(paramMatches[1])  // ✅ 工具参数
        }

        thought.Action = action
    }

    return thought
}
```

**LLM 响应格式示例**:
```
Thought: I need to calculate 15 * 23 + 7
Action: calculator
Parameters: {"operation": "multiply", "a": 15, "b": 23}
Reasoning: First step is multiplication
```

### 3. 执行工具 ✅

**位置**: `pkg/engine/engine.go:292-340`

```go
// 2. 行动（如果有动作）
if thought.Action != nil {
    // ✅ 执行工具
    execResult, err = e.actor.Act(thought.Action, ctx)

    // 记录执行结果
    state.LastResult = execResult

    // 添加轨迹
    result.Trace = append(result.Trace, types.TraceStep{
        Type:    "result",
        Content: fmt.Sprintf("Result: %v (Success: %v)",
                            execResult.Output, execResult.Success),
    })
}
```

**Actor 实现** (`pkg/core/actor.go`):
```go
func (a *DefaultActor) Act(action *types.Action, ctx *Context) (*types.ExecutionResult, error) {
    // ✅ 从 ToolManager 获取工具
    tool, err := a.toolManager.GetTool(action.ToolName)
    if err != nil {
        return nil, err
    }

    // ✅ 执行工具
    output, err := tool.Execute(action.Parameters)

    return &types.ExecutionResult{
        Output:  output,
        Success: err == nil,
        Error:   err,
    }, nil
}
```

### 4. 结果反馈给 LLM ✅

**位置**: `pkg/engine/engine.go:342-377` + `pkg/core/thinker/simple_thinker.go:29-51`

```go
// 3. 观察
feedback, observeErr = e.observer.Observe(execResult, ctx)

// ✅ 更新 Context（下次迭代时会用到）
ctx.Set("last_action", thought.Action)
ctx.Set("last_result", execResult)
ctx.Set("last_feedback", feedback)
```

**下次迭代时，Thinker 会读取这些信息**:
```go
func (t *SimpleThinker) Think(task string, context *core.Context) (*types.Thought, error) {
    // ✅ 从 Context 中获取历史信息
    var history string

    if lastAction, ok := context.Get("last_action"); ok {
        history += fmt.Sprintf("Action: %s with params %v\n",
                              action.ToolName, action.Parameters)
    }

    if lastResult, ok := context.Get("last_result"); ok {
        history += fmt.Sprintf("Result: %v (Success: %v)\n",
                              result.Output, result.Success)
    }

    if lastFeedback, ok := context.Get("last_feedback"); ok {
        history += fmt.Sprintf("Feedback: %s\n", feedback.Message)
    }

    // ✅ 历史信息会被添加到下次的 prompt 中
    prompt := t.buildPrompt(task, context)  // 包含历史记录

    // 再次调用 LLM
    response, err := t.llmClient.Generate(prompt)
    // ...
}
```

## 完整示例追踪

### 迭代 1

**1. Prompt 发送给 LLM**:
```
You are a helpful AI assistant that uses tools to solve tasks.

Available tools:
- calculator: Perform mathematical calculations
  Parameters: operation, a, b

Task: Calculate 15 * 23 + 7

Please respond in the following format:
Thought: [your reasoning]
Action: [tool_name]
Parameters: {"key": "value"}
```

**2. LLM 响应**:
```
Thought: I need to multiply 15 by 23 first
Action: calculator
Parameters: {"operation": "multiply", "a": 15, "b": 23}
```

**3. 解析并执行**:
- 提取: `Action = "calculator"`, `Parameters = {operation: multiply, a: 15, b: 23}`
- 执行: `calculator.Execute({operation: multiply, a: 15, b: 23})`
- 结果: `345`

**4. 更新 Context**:
```
last_action: calculator with {operation: multiply, a: 15, b: 23}
last_result: 345 (Success: true)
last_feedback: Calculation successful
```

### 迭代 2

**1. Prompt 发送给 LLM** (包含历史):
```
You are a helpful AI assistant that uses tools to solve tasks.

Available tools:
- calculator: Perform mathematical calculations

Task: Calculate 15 * 23 + 7

Previous steps:
Action: calculator with params {operation: multiply, a: 15, b: 23}
Result: 345 (Success: true)
Feedback: Calculation successful

Please respond in the following format:
Thought: [your reasoning]
Action: [tool_name]
Parameters: {"key": "value"}
```

**2. LLM 响应**:
```
Thought: Now I need to add 7 to 345
Action: calculator
Parameters: {"operation": "add", "a": 345, "b": 7}
```

**3. 执行并更新** (重复流程)

### 迭代 3

**1. Prompt** (包含所有历史)

**2. LLM 响应**:
```
Thought: I have completed the calculation
Final Answer: The result of 15 * 23 + 7 is 352
```

**3. 检测到 Final Answer，结束循环**

## 流程确认 ✅

你的理解完全正确！当前实现确实遵循以下流程：

1. ✅ **装配工具给 LLM**:
   - 工具描述在 `buildPrompt()` 中包含
   - 通过 `t.toolDesc` 传递给 LLM

2. ✅ **获取工具调用列表**:
   - LLM 返回格式化的响应
   - `parseResponse()` 解析出 Action 和 Parameters

3. ✅ **执行工具**:
   - `Actor.Act()` 调用 `Tool.Execute()`
   - 返回 `ExecutionResult`

4. ✅ **反馈结果给 LLM**:
   - 结果存入 `Context`
   - 下次迭代时包含在 prompt 的历史记录中
   - LLM 基于历史结果继续推理

## 关键特性

### 1. 上下文传递
- 使用 `Context` 对象在迭代间传递信息
- 每次迭代都能看到之前的执行结果

### 2. 工具发现
- 工具通过 `ToolManager` 统一管理
- 支持内置工具和 Provider 工具
- 自动生成工具描述

### 3. 错误处理
- 每个步骤都有重试机制
- 失败时提供详细的错误信息
- 支持优雅降级

### 4. 可观测性
- 完整的执行轨迹 (`Trace`)
- 每个步骤都有时间戳
- 便于调试和分析

## 总结

GoReAct 框架完整实现了标准的 ReAct 循环：

```
Think (带工具列表) → Parse (提取工具调用) → Act (执行工具) →
Observe (生成反馈) → Update Context (保存结果) → Think (带历史) → ...
```

这个流程确保了：
- ✅ LLM 知道有哪些工具可用
- ✅ LLM 可以选择合适的工具
- ✅ 工具被正确执行
- ✅ 执行结果反馈给 LLM
- ✅ LLM 基于结果继续推理
- ✅ 循环直到任务完成

整个流程是标准的、完整的 ReAct 实现！🎉
