# Tool 开发指南 — 工具即编排

> **核心理念：Description 就是 Prompt。** 当开发者理解 `Description` 和 `Prompt` 实际上是大模型的指令时，就会意识到工具不仅仅是功能函数，更是编排智能体行为的编程语言。

---

## 目录

- [1. 范式转变：工具即编排](#1-范式转变工具即编排)
- [2. 工具的本质 — 双重接口](#2-工具的本质双重接口)
- [3. 最小工具](#3-最小工具)
- [4. Description 工程 — 教 LLM 使用你的工具](#4-description-工程教-llm-使用你的工具)
- [5. Prompt 工程 — 教 LLM 何时使用工具](#5-prompt-工程教-llm-何时使用工具)
- [6. 参数设计 — 引导 LLM 的输入方式](#6-参数设计引导-llm-的输入方式)
- [7. 返回值设计 — 让 LLM 理解结果](#7-返回值设计让-llm-理解结果)
- [8. 高级模式](#8-高级模式)
- [9. 工具编排模式](#9-工具编排模式)
- [10. 最佳实践清单](#10-最佳实践清单)

---

## 1. 范式转变：工具即编排

### 传统思维（错误）

```
工具 = 函数
调用 = LLM 传入参数 → 执行代码 → 返回结果
```

这是把工具当作普通的 API 调用。LLM 只会机械地传参，不知道为什么要用、什么时候用、用完之后该做什么。

### 编排思维（正确）

```
工具 = 行为指令
Description = 告诉 LLM 这个工具能做什么
Prompt = 教 LLM 在什么场景下使用它
参数 = 限制 LLM 的输入方式
返回值 = 引导 LLM 的下一步决策
```

**工具不只是代码 — 它是你编程 LLM 决策的接口。**

### 类比

| 传统编程 | Agent 工具编排 |
|----------|---------------|
| 函数签名 | `ToolInfo{Name, Description, Parameters}` |
| 函数实现 | `Execute(ctx, params)` |
| **文档注释** | **`Description` — LLM 看到的"函数文档"** |
| **使用场景说明** | **`Prompt` — LLM 的"使用手册"** |
| 返回值 | LLM 下一步行动的输入 |

**关键洞察：** 当你写好一个工具的 `Description` 和 `Prompt` 时，你实际上是在 **编写 LLM 的行为脚本**。

---

## 2. 工具的本质 — 双重接口

每个工具都有两个接口：

### 接口 1：LLM 看到的接口

LLM 只看到 `ToolInfo` 的 JSON 描述：

```json
{
  "name": "Read",
  "description": "Read a file from the filesystem.",
  "parameters": [
    {"name": "path", "type": "string", "description": "The file path to read."}
  ]
}
```

**这就是 LLM 唯一能理解的"函数签名"。** 如果 `description` 写得不好，LLM 就不会正确使用这个工具。

### 接口 2：代码实现的接口

```go
func (t *MyTool) Execute(ctx context.Context, params map[string]any) (any, error)
```

这是 Go 代码的函数签名，LLM **完全看不到**。

### 结论

```
Description + Prompt + Parameters
         ↓
    教 LLM 如何用工具  ← 这才是编排
         ↓
    Execute() 只是执行逻辑  ← 这只是实现
```

---

## 3. 最小工具

```go
package tools

import (
    "context"
    "fmt"

    "github.com/DotNetAge/goreact/core"
)

// GreetTool 是最简单的工具示例
type GreetTool struct{}

func NewGreetTool() *GreetTool {
    return &GreetTool{}
}

// Info 返回工具的元数据 — 这是 LLM 看到的"函数签名"
func (t *GreetTool) Info() *core.ToolInfo {
    return &core.ToolInfo{
        Name:        "Greet",              // 工具名：动词、简洁
        Description: "Generate a greeting message for a person.",  // ← LLM 看到的能力描述
        Parameters: []core.Parameter{
            {
                Name:        "name",
                Type:        "string",
                Description: "The person's name to greet.",
                Required:    true,
            },
        },
    }
}

// Execute 是工具的实际逻辑 — LLM 看不到这个
func (t *GreetTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    name, _ := params["name"].(string)
    if name == "" {
        return nil, fmt.Errorf("name is required")
    }
    return fmt.Sprintf("Hello, %s!", name), nil
}
```

这就是全部。一个工具只需要：
1. `Info()` — 定义 LLM 看到的接口
2. `Execute()` — 实现逻辑

---

## 4. Description 工程 — 教 LLM 使用你的工具

### 核心原则：Description 是 LLM 的唯一信息来源

LLM 无法阅读你的代码注释、无法运行单元测试、无法查看 README。**它唯一能看到的就是 `Description` 字段。**

### 好的 Description 的特征

| 特征 | 说明 | 示例 |
|------|------|------|
| **动词开头** | 描述工具做什么动作 | `"Search files by pattern"` |
| **具体明确** | 不说"处理文件"，说"搜索文件名" | 好：`"Search for files matching a glob pattern"`<br>差：`"Work with files"` |
| **包含用途** | 一句话说明为什么用它 | `"Find all Go source files in a directory tree"` |
| **避免技术术语** | 用 LLM 能理解的自然语言 | 好：`"Read file contents"`<br>差：`"Perform OS-level file descriptor read with buffering"` |

### 对比示例

```go
// ❌ 差的 Description
Description: "A tool for file operations."
// LLM 不知道能做什么操作、什么时候用

// ❌ 不够好
Description: "Read and write files."
// 还是太模糊，LLM 不知道该用 Read 还是 Write

// ✅ 好的 Description
Description: "Read a file from the filesystem. Returns the file contents as text."
// LLM 明确知道：读取文件 → 返回文本

// ✅ 更好的 Description（包含场景）
Description: "Read a file from the filesystem. Use this when you need to examine file contents, check code, or read configuration. Returns the file contents as text."
// LLM 知道三种使用场景
```

### Description 模板

```
[动词] [对象] [补充说明]. [使用场景 1-2 个]. 返回 [结果类型].
```

---

## 5. Prompt 工程 — 教 LLM 何时使用工具

### Prompt 是什么

`Prompt` 是 `ToolInfo` 中比 `Description` 更详细的字段。它不会被发送到 LLM API，而是注入到系统提示中，**教 LLM 在什么情况下应该选择这个工具**。

### Prompt 是编排的核心

**Description 告诉 LLM "这个工具能做什么"，Prompt 教 LLM "什么时候用、怎么用、用完该做什么"。**

### 好的 Prompt 结构

```
[一句话总结功能]

使用场景：
- 场景 1
- 场景 2
- 场景 3

用法说明：
- 参数说明
- 返回值说明
- 注意事项

不要：
- 不该用的场景 1
- 不该用的场景 2
```

### 实际案例：Bash 工具的 Prompt

```go
Prompt: `Execute a shell command using Bash.

Use this for:
- Running system commands (ls, grep, git, etc.)
- Executing scripts (Python, Node.js, etc.)
- Piping commands together
- Running commands with environment variables

Parameters:
- cmd (required): The command to execute.
- timeout_ms (optional): Timeout in milliseconds. Default: 30000.

Returns:
- stdout and stderr output (truncated at 30KB if too large)

Important:
- Commands run in a sandbox with restricted permissions.
- Long-running commands will be killed after timeout.
- Output is truncated to 30KB to save context.

Don't:
- Don't use for file read/write (use Read/Write tools instead).
- Don't run interactive commands (they will hang).
- Don't assume the working directory is the project root.`,
```

**这个 Prompt 教会了 LLM：**
- 5 个使用场景
- 2 个参数及默认值
- 返回值的格式和限制
- 3 个安全注意事项
- 3 个不该做的事

### 编排效果

当 LLM 收到用户请求 `"运行 Python 脚本分析数据"` 时，它会根据 Prompt 中的信息做出决策：

1. ✅ 这是 "执行脚本" 的场景 → 选择 Bash
2. ✅ 有 `cmd` 参数 → 构造 `python analyze.py`
3. ✅ 知道输出可能被截断 → 做好处理准备

### Prompt 与 Description 的配合

| 字段 | 长度 | 作用 | LLM 何时看到 |
|------|------|------|-------------|
| **Description** | 一句话 | 告诉 LLM **能做什么** | 工具列表（Think 阶段） |
| **Prompt** | 多段落 | 教 LLM **何时/如何用** | 系统提示（注入上下文） |

**关键：** Description 用于工具选择，Prompt 用于工具使用。两者配合才能实现完整的编排。

---

## 6. 参数设计 — 引导 LLM 的输入方式

### 参数的本质

参数是你**约束 LLM 行为的方式**。好的参数设计让 LLM 知道：
- 必须提供什么
- 可选提供什么
- 每个参数的类型和含义

### 参数设计原则

| 原则 | 说明 | 示例 |
|------|------|------|
| **Required = true** | 必填参数强制 LLM 提供 | `{"name": "path", "required": true}` |
| **类型明确** | string, integer, boolean, array | `{"type": "integer", "description": "Timeout in ms"}` |
| **描述具体** | 不只是"路径"，而是"文件的绝对路径" | `{"description": "The absolute file path to read"}` |
| **数量适度** | 3-5 个参数最佳，超过会增加 LLM 困惑 | 考虑合并相关参数 |
| **避免重叠** | 不要有含义相似的参数 | 用 `path` 而不是 `file_path` + `filepath` |

### 参数类型参考

```go
// 字符串参数 — 最常用
{Name: "path", Type: "string", Description: "File path.", Required: true}

// 整数参数 — 用于数值限制
{Name: "max_lines", Type: "integer", Description: "Maximum lines to read.", Required: false}

// 布尔参数 — 用于开关
{Name: "recursive", Type: "boolean", Description: "Search recursively.", Required: false}

// 数组参数 — 用于列表
{Name: "paths", Type: "array", Description: "List of file paths."}
```

### 参数编排技巧

**技巧 1：用参数引导行为**

```go
// 不写 "append" 参数 → LLM 只能覆盖写入
// 写 "append" 参数 → LLM 知道可以追加
{Name: "append", Type: "boolean", Description: "Append to existing file instead of overwriting.", Required: false}
```

**技巧 2：用默认值减少 LLM 负担**

```go
// 不需要 LLM 传 timeout → 使用默认值
// 需要时 LLM 可以覆盖
{Name: "timeout_ms", Type: "integer", Description: "Timeout in milliseconds. Default: 30000.", Required: false}
```

---

## 7. 返回值设计 — 让 LLM 理解结果

### 返回值的本质

返回值是 LLM **下一步决策的输入**。好的返回值格式让 LLM 能：
- 理解工具执行的结果
- 判断是否成功
- 决定下一步行动

### 返回值格式

```go
// ✅ 推荐：返回结构化的 map
return map[string]any{
    "status":  "success",
    "message": "File written successfully",
    "bytes":   len(content),
}

// ❌ 避免：只返回纯文本
return "done"

// ✅ 返回错误信息（供 LLM 理解失败原因）
return map[string]any{
    "status": "error",
    "error":  "Permission denied: /etc/passwd",
}
```

### 返回值的编排作用

**返回值影响 LLM 的下一步行为。** 如果你返回的信息不完整，LLM 可能会：
- 不知道该工具是否成功
- 重复调用同一个工具
- 做出错误的后续决策

### 返回值模板

```go
// 成功场景
return map[string]any{
    "status": "success",
    // 业务数据...
    "count": 42,
    "items": results,
}

// 失败场景
return nil, fmt.Errorf("具体错误信息：原因")
```

### 截断与通知

对于可能产生大量输出的工具（Bash, Read 等），**必须截断并通知 LLM**：

```go
if len(output) > 30000 {
    output = output[:30000]
    output += "\n\n[Output truncated — use more specific parameters to reduce output size]"
}
```

这个通知教会 LLM **"输出太大了，下次用更精确的参数"**。

---

## 8. 高级模式

### 8.1 异步工具 (IsAsync)

异步工具让 LLM 知道工具会立即返回，需要后续收集结果：

```go
func (t *TaskCreateTool) Info() *core.ToolInfo {
    return &core.ToolInfo{
        Name:        "TaskCreate",
        Description: "Create a background task that runs an agent asynchronously.",
        IsAsync:     true,  // ← 标记为异步
        // ...
    }
}
```

**编排意义：** LLM 会理解这个工具是 "发射后不管"，需要用 `TaskGet` 或 `CollectResults` 来获取结果。

### 8.2 工具上下文 (ToolContext)

工具可以通过 `ToolContext` 访问框架提供的能力：

```go
func (t *MyTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    toolCtx := core.GetToolContext(ctx)
    if toolCtx == nil {
        return nil, fmt.Errorf("ToolContext not available")
    }

    // 访问 KVStore 实现工具间数据共享
    if toolCtx.KVStore != nil {
        data, _ := toolCtx.KVStore.Get(ctx, toolCtx.SessionID, "config")
    }

    // 发送事件通知
    if toolCtx.EmitEvent != nil {
        toolCtx.EmitEvent(core.ReactEvent{
            Type: core.ToolCall,
            Data: map[string]any{"tool": "my_tool"},
        })
    }

    return "result", nil
}
```

**编排意义：** KVStore 和 FileStore 让多个工具可以在同一会话内共享数据，实现**工具间的隐式编排**。

### 8.3 Hook 机制

Hook 让你在工具执行前后注入自定义逻辑：

```go
// 安全拦截器
type SecurityHook struct{}

func (h *SecurityHook) EventType() core.HookEventType {
    return core.HookPreToolUse
}

func (h *SecurityHook) Execute(ctx *core.HookContext) core.HookResult {
    if ctx.ToolUseContext.ToolName == "Bash" {
        cmd := ctx.ToolUseContext.Params["cmd"].(string)
        if strings.Contains(cmd, "rm -rf") {
            return core.HookResult{
                PreventContinuation: true,
                Message:             "Dangerous command blocked: rm -rf",
            }
        }
    }
    return core.HookResult{}
}

// 注册 Hook
agent := goreact.NewAgent(
    goreact.WithModel(model),
    goreact.WithPreHook(&SecurityHook{}),
)
```

**编排意义：** Hook 让你可以在**不修改工具代码**的情况下改变工具的行为。

---

## 9. 工具编排模式

### 9.1 工具链模式

一个工具的输出是下一个工具的输入。LLM 自动串联：

```
Grep (搜索内容) → Read (读取匹配文件) → Write (修改内容)
```

**实现方式：** 返回清晰的输出，让 LLM 能理解并传给下一个工具。

### 9.2 并行模式

多个独立任务同时执行：

```
TaskCreate → agent-1 (分析数据)
TaskCreate → agent-2 (生成报告)
TaskCreate → agent-3 (发送邮件)
```

**实现方式：** `IsAsync: true` 的工具让 LLM 知道可以同时创建多个。

### 9.3 协调模式

主 Agent 协调多个子 Agent 协作：

```
TeamCreate (创建团队)
  → leader 分配任务
    → member-1 执行
    → member-2 执行
  → leader 汇总结果
```

**实现方式：** TeamCreate 工具 + Task 工具组合。

### 9.4 条件模式

根据工具返回值决定下一步：

```
Read (读取配置文件)
  → 如果配置存在 → Bash (执行脚本)
  → 如果配置不存在 → Write (创建配置)
```

**实现方式：** 返回结构化结果，让 LLM 能做条件判断。

---

## 10. 最佳实践清单

### 设计阶段

- [ ] **Description 用动词开头**，描述工具做什么
- [ ] **Prompt 包含使用场景**，至少 2-3 个
- [ ] **Prompt 包含注意事项**，告诉 LLM 不该做什么
- [ ] **参数数量控制在 3-5 个**
- [ ] **每个参数有明确的 Description**
- [ ] **返回值结构化**（map[string]any）

### 实现阶段

- [ ] **参数验证** — 检查必填参数，返回明确错误
- [ ] **类型安全** — 使用类型断言，不要假设参数类型
- [ ] **错误信息清晰** — 返回 LLM 能理解的错误
- [ ] **输出截断** — 大输出必须截断并通知 LLM
- [ ] **并发安全** — goroutine 中使用局部变量拷贝

### 编排思考

- [ ] **这个工具的 Prompt 是否教会 LLM 何时使用？**
- [ ] **返回值是否足够引导 LLM 的下一步？**
- [ ] **是否能与其他工具组合形成工作流？**
- [ ] **是否有不必要的工具？**（两个功能相似的工具会增加 LLM 决策负担）

### 测试阶段

- [ ] **测试工具在正常参数下的行为**
- [ ] **测试缺少必填参数时的错误**
- [ ] **测试错误参数类型时的处理**
- [ ] **测试工具输出的可读性**

---

## 附录：工具设计检查表

当你设计一个新工具时，问自己这些问题：

| 问题 | 说明 | 检查项 |
|------|------|--------|
| **LLM 能理解这个工具吗？** | Description 是否清晰？ | 读一遍 Description，想象你是 LLM |
| **LLM 知道什么时候用吗？** | Prompt 是否有场景？ | Prompt 中至少 2 个使用场景 |
| **LLM 知道怎么用吗？** | 参数描述是否具体？ | 每个参数有类型、描述、是否必填 |
| **LLM 理解结果吗？** | 返回值是否结构化？ | 返回 map 而非纯文本 |
| **失败时 LLM 能恢复吗？** | 错误信息是否有指导性？ | 错误信息包含原因和建议 |
| **这个工具是必须的吗？** | 是否与现有工具重叠？ | 检查现有工具列表 |
| **工具能与其他工具配合吗？** | 输出格式是否通用？ | 考虑工具链场景 |

**记住：** 工具开发的本质不是写 Go 代码，而是**编程 LLM 的决策流程**。你写的每一行 Description、每一个参数描述、每一个返回值，都在塑造 LLM 与你工具的交互方式。
