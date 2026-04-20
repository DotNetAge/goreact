# 从 ClueCode 源代码中为 goreact TODO 寻找解决方案

## 研究报告

## 执行摘要

通过对 ClueCode 源代码的深入分析，本文档为 goreact 的 5 个未解决问题提供了来自 ClueCode 实现的具体解决方案参考。ClueCode 采用了多层次策略来管理上下文窗口、并行子任务流式输出、用户交互打断/恢复、以及外部知识注入。每个问题都对应 ClueCode 中一套成熟的实现模式，可以直接借鉴到 goreact 的 Go 实现中。

## 问题一：大量文件查看与上下文爆炸 (TODO Line 14)

### ClueCode 的三层防御策略

ClueCode 通过三个层次来防止文件读取导致的上下文爆炸：

**第一层：文件读取工具的内置限制**

在 `cludecode/tools/FileReadTool/limits.ts` 中，ClueCode 定义了文件读取的硬性上限：

- `maxSizeBytes`：默认 256KB，在读取前检查文件总大小，超限直接抛出错误（pre-read check）
- `maxTokens`：默认 25,000 tokens，对实际输出内容做 token 计数，超限在读取后抛出（post-read check）
- 支持 `offset` 和 `limit` 参数实现分段读取
- 优先级为：环境变量 > GrowthBook 实验配置 > 默认常量

这些限制通过 `ToolUseContext` 的 `fileReadingLimits` 字段注入，每个工具调用都可以获取当前生效的限制值。

**第二层：工具结果的持久化（Persist to Disk）**

在 `cludecode/utils/toolResultStorage.ts` 中，ClueCode 实现了一套工具结果磁盘持久化机制：

- 全局默认上限 `DEFAULT_MAX_RESULT_SIZE_CHARS = 50,000` 字符
- 单条消息内所有工具结果的总上限 `MAX_TOOL_RESULTS_PER_MESSAGE_CHARS = 200,000` 字符
- 当工具结果超过阈值时，结果被保存到磁盘文件（`<persisted-output>` 标签包裹），模型只收到预览文本和文件路径
- 每个 `Tool` 都有 `maxResultSizeChars` 属性，`Read` 工具设为 `Infinity`（避免循环读取问题）
- 通过 `getPersistenceThreshold()` 函数动态决定每个工具的持久化阈值

这套机制防止了 N 个并行工具各自达到上限后累积爆炸的问题。

**第三层：上下文压缩（Compact）**

在 `cludecode/services/compact/compact.ts` 中，ClueCode 实现了自动上下文压缩：

- 通过 `compactMessages()` 函数，将历史消息发送给 LLM 做摘要压缩
- 压缩前后有 `pre_compact` 和 `post_compact` 钩子事件
- 压缩后生成 `SystemCompactBoundaryMessage` 作为边界标记
- 支持 `microCompact`（快速压缩）和完整 `compact` 两种模式
- 自动压缩通过 `autoCompact` 模块在上下文接近上限时自动触发

### goreact 的建议实现

在 goreact 中，建议实现以下三层防御：

1. 为每个文件读取/搜索工具设置 `MaxTokens` 和 `MaxSizeBytes` 限制，并暴露 `Offset` 和 `Limit` 参数
2. 实现 `ToolResultStorage` 接口，对超过字符阈值的工具结果进行磁盘持久化，仅返回预览和路径
3. 在 `ContextWindow` 中实现基于 token 阈值的自动压缩逻辑

---

## 问题二：主任务与子任务的并行流式输出 (TODO Line 15)

### ClueCode 的子任务架构

**任务类型体系**

在 `cludecode/Task.ts` 中，ClueCode 定义了 6 种任务类型：

```go
type TaskType = 
  | "local_bash"      // 本地 Shell 命令
  | "local_agent"     // 本地 Agent（子代理）
  | "remote_agent"    // 远程 Agent
  | "in_process_teammate"  // 进程内队友
  | "local_workflow"  // 本地工作流
  | "monitor_mcp"     // MCP 监控
  | "dream"           // 后台任务
```

每个任务都有独立的状态（pending/running/completed/failed/killed）、输出文件和偏移量追踪。

**进程内队友通信机制**

在 `cludecode/tools/SendMessageTool/SendMessageTool.ts` 中，ClueCode 实现了一套基于邮箱（mailbox）的队友通信协议：

- `SendMessageTool` 支持三种消息类型：`message`（私信）、`broadcast`（广播）、`shutdown_request`/`shutdown_response`（关闭请求）
- 通过 `writeToMailbox()` 将消息写入磁盘文件作为持久化邮箱
- 每个队友有独立的 `AbortController`，支持异步独立执行
- `queuePendingMessage()` 用于向正在运行的队友排队消息
- `resumeAgentBackground()` 用于从停止状态恢复队友

**流式输出的关键设计**

在 `cludecode/services/tools/toolExecution.ts` 中，ClueCode 的工具执行核心使用了 `AsyncGenerator` 模式：

```typescript
async function* runToolUse(...): AsyncGenerator<MessageUpdateLazy, void>
```

每个工具执行都是异步生成器，可以逐步产出进度消息（`ProgressMessage`）和最终结果（`UserMessage`），实现了真正的流式输出。并行工具调用时，主线程通过 `Promise.all` 同时消费多个生成器。

**UI 渲染层**

ClueCode 使用 Ink（React CLI 框架）渲染终端 UI。在 `cludecode/ink/` 目录中，组件系统支持：
- `renderToolUseProgressMessage()` 显示工具执行进度
- `renderGroupedToolUse()` 将多个并行工具调用渲染为一个组
- `SpinnerMode` 控制 spinner 状态（thinking、executing 等）

### goreact 的建议实现

1. 定义 `Task` 接口，支持 `local_agent` 和 `in_process_teammate` 等类型
2. 实现基于 `AsyncGenerator` 或 Go channel 的流式输出管道，主任务和子任务各自向独立 channel 发送进度
3. 客户端通过 WebSocket/SSE 订阅多个流，使用独立的 UI 区域渲染每个任务的输出
4. 使用 `AbortController`（Go 中的 `context.Context`）管理每个子任务的生命周期


---

## 问题三：Memory/RAG 接口设计抑制幻觉 (TODO Line 18)

### ClueCode 的知识管理架构

ClueCode 没有传统意义上的 "RAG 接口"，而是通过三种互补机制来实现外部知识注入：

**机制一：结构化 Memory 系统**

在 `cludecode/memdir/memoryTypes.ts` 中，ClueCode 实现了一套精心设计的记忆分类体系：

- 四种记忆类型：`user`（用户偏好）、`feedback`（用户反馈/指导）、`project`（项目上下文）、`reference`（外部资源指针）
- 每种记忆有明确的 `when_to_save`（何时保存）、`how_to_use`（如何使用）、`body_structure`（正文结构）
- 记忆以 Markdown 文件存储在 `memdir/` 目录中，使用 frontmatter 格式（`---` + YAML header）
- 支持 private（个人）和 team（团队）两种作用域
- 显式排除可从代码/git/CLAUDE.md 推导的信息，避免冗余

这个设计的核心理念是：Memory 存储的是"无法从当前项目状态推导出的上下文"。

**机制二：Session Memory（会话记忆）**

在 `cludecode/services/SessionMemory/sessionMemory.ts` 中，ClueCode 实现了自动化的会话记忆：

- 使用 forked subagent（后台子代理）定期从对话中提取关键信息
- 不阻塞主对话流，在后台运行
- 生成结构化的 Session Memory 文件，记录本次对话的关键决策和发现
- 通过 `postSamplingHook` 注册，在每个 LLM 响应后检查是否需要更新

**机制三：MCP（Model Context Protocol）集成**

ClueCode 通过 MCP 协议接入外部知识源：

- MCP 工具通过 `mcp__server__tool` 命名空间暴露
- 支持 stdio、SSE、HTTP、WebSocket 等多种传输方式
- `MCPTool` 类作为通用 MCP 工具的包装器
- MCP 服务器可以提供工具、资源和提示模板

**机制四：Skill 系统**

在 `cludecode/skills/` 目录中，ClueCode 的 Skill 是可加载的知识包：

- 每个 Skill 本质上是一段结构化的系统提示词
- Skill 通过 `use_skill` 工具激活，将专业知识注入当前上下文
- 支持 bundled（内置）和 user-defined（用户自定义）两种类型

### goreact 的建议实现

goreact 的 `Memory` 接口设计可以借鉴 ClueCode 的分层架构：

```go
type Memory interface {
    // 检索相关记忆（对应 RAG 查询）
    Retrieve(ctx context.Context, query string, opts ...MemoryOption) ([]MemoryRecord, error)
    
    // 保存新记忆
    Store(ctx context.Context, record MemoryRecord) error
    
    // 更新已有记忆
    Update(ctx context.Context, id string, record MemoryRecord) error
    
    // 删除记忆
    Delete(ctx context.Context, id string) error
}

type MemoryRecord struct {
    ID          string
    Type        MemoryType  // user, feedback, project, reference
    Title       string
    Content     string
    Scope       MemoryScope // private, team
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

实现方可以选择：
- 简单数据库实现：直接 CRUD
- RAG 实现：Retrieve 时做向量相似度搜索
- 混合实现：结构化字段走数据库，Content 走向量检索

---

## 问题四：Think 阶段的澄清交互 (TODO Line 17)

### ClueCode 的打断-恢复机制

ClueCode 处理"需要用户澄清"的场景主要通过 AskUserQuestion 工具实现：

**AskUserQuestion 工具**

在 `cludecode/tools/AskUserQuestionTool/AskUserQuestionTool.tsx` 中：

- 这是一个特殊的只读工具，设置 `requiresUserInteraction() = true` 和 `checkPermissions() = { behavior: 'ask' }`
- 当 LLM 调用此工具时，流程被挂起等待用户交互
- 支持结构化多选题（1-4 个问题，每个 2-4 个选项）和自由文本输入
- 用户回答后，答案被序列化为 `tool_result` 返回给 LLM，对话自然继续
- 关键设计：用户答案被注入为 tool_result 的 content：`User has answered your questions: "Q1"="A1", "Q2"="A2". You can now continue with the user's answers in mind.`

**权限检查的异步挂起机制**

在 `cludecode/services/tools/toolExecution.ts` 的 `checkPermissionsAndCallTool()` 函数中：

```typescript
const resolved = await resolveHookPermissionDecision(
    hookPermissionResult, tool, processedInput, toolUseContext,
    canUseTool, assistantMessage, toolUseID,
)
```

权限解析是一个 `await` 操作，当 `behavior === 'ask'` 时，函数会挂起直到用户响应。在此期间，消息列表保持不变，上下文完全保留。

**上下文连续性保证**

- 整个打断-恢复过程发生在同一个 API 调用的 tool_use 阶段
- 用户响应被添加为 `tool_result` block，是 API 对话链的自然延续
- 不需要特殊的状态保存/恢复机制，因为消息列表一直存在内存中
- `AbortController` 确保用户可以在等待期间取消操作

### goreact 的建议实现

goreact 的 Think 阶段澄清流程可以设计为：

1. 在 Think 阶段，当 Agent 判断需要澄清时，调用一个特殊的 "AskUser" 动作
2. 这个动作的执行函数向客户端发送一个 "需要用户输入" 的事件，然后通过 channel/condition 挂起
3. 客户端显示问题，收集用户输入后发送回服务端
4. 服务端将用户输入作为 AskUser 动作的 tool_result 注入消息列表
5. Agent 恢复执行，从当前消息列表自然继续

关键：整个过程不需要序列化/反序列化整个 Agent 状态，只需要将用户回答作为 tool_result 追加到消息列表即可。

---

## 问题五：Action 阶段的高风险工具授权 (TODO Line 18)

### ClueCode 的工具授权体系

ClueCode 实现了一套非常完善的多层工具授权机制：

**Hooks 系统**

在 `cludecode/utils/hooks/hookEvents.ts` 和 `cludecode/services/tools/toolHooks.ts` 中：

- 钩子事件包括：`SessionStart`、`PreToolUse`、`PostToolUse`、`PreCompact`、`PostCompact`、`Stop`、`PermissionRequest` 等
- `PreToolUse` 钩子可以：阻止工具执行（`preventContinuation`）、修改输入（`hookUpdatedInput`）、直接做出权限决定（`hookPermissionResult`）
- 钩子可以返回多种结果类型：`message`、`hookPermissionResult`、`hookUpdatedInput`、`preventContinuation`、`stopReason`、`stop`

**权限决策流水线**

在 `cludecode/services/tools/toolExecution.ts` 的 `checkPermissionsAndCallTool()` 中，权限检查是一个多层决策流水线：

1. `validateInput()` — 输入类型校验
2. `runPreToolUseHooks()` — 执行所有 PreToolUse 钩子（并行执行）
3. `resolveHookPermissionDecision()` — 解析钩子的权限决策结果
4. `canUseTool()` — 最终权限判定（可能是交互式的，等待用户）
5. 如果 `behavior !== 'allow'`，返回错误结果，工具不执行
6. 如果允许，使用 `permissionDecision.updatedInput`（可能被用户修改）继续执行

**工具属性标记**

在 `cludecode/Tool.ts` 的 `Tool` 类型定义中：

- `isDestructive?(input)` — 标记工具是否执行不可逆操作（删除、覆写、发送）
- `isReadOnly(input)` — 标记是否只读
- `isConcurrencySafe(input)` — 标记是否可以并行执行
- `checkPermissions(input, context)` — 每个工具可以有自己的权限检查逻辑
- `interruptBehavior?()` — 定义用户提交新消息时的行为：`'cancel'`（停止）或 `'block'`（等待）
- `requiresUserInteraction?()` — 标记是否需要用户交互

**权限模式**

ClueCode 支持多种权限模式：
- `default` — 每次需要交互确认
- `auto` — 自动模式，使用安全分类器自动判定
- `bypassPermissions` — 绕过所有权限检查
- `plan` — 计划模式，只生成计划不执行

### goreact 的建议实现

在 goreact 的 T-A-O 框架中，Action 阶段的高风险工具授权流程：

```
[工具调用请求]
    ↓
[validateInput] → 失败 → 返回错误
    ↓ 成功
[PreToolUse Hooks] → 可阻止/修改/放行
    ↓
[checkPermissions] 
    ├─ allow → 继续执行
    ├─ deny → 返回拒绝消息  
    └─ ask → 挂起等待用户
              ↓
         [发送授权请求给客户端]
              ↓
         [用户响应: accept/reject/modify]
              ↓
         [注入 tool_result 继续流程]
```

Go 实现的关键接口：

```go
type PermissionResult struct {
    Behavior   string      // "allow", "deny", "ask"
    Message    string      // 拒绝时的原因
    UpdatedInput any       // 用户可能修改的输入
}

type ToolPermissionChecker interface {
    CheckPermissions(input ToolInput, ctx ToolUseContext) PermissionResult
}

type Hook interface {
    OnPreToolUse(ctx HookContext) HookResult
    OnPostToolUse(ctx HookContext) HookResult
}

type HookResult struct {
    PermissionResult *PermissionResult  // 钩子可以做出权限决定
    UpdatedInput     any               // 钩子可以修改输入
    PreventContinuation bool           // 钩子可以阻止执行
}
```

---

## 结论

ClueCode 的源代码为 goreact 的 5 个 TODO 问题提供了成熟且经过生产验证的解决方案：

1. 上下文爆炸：三层防御 — 工具内置限制 + 结果持久化 + 自动压缩
2. 并行流式输出：AsyncGenerator 模式 + 独立任务类型 + 邮箱通信协议
3. Memory/RAG：分层知识架构 — 结构化 Memory + 会话记忆 + MCP + Skill
4. Think 阶段澄清：AskUserQuestion 工具 + 异步权限挂起 + 自然 tool_result 恢复
5. Action 阶段授权：Hooks 系统 + 多层权限流水线 + 工具属性标记

## References

1. `cludecode/tools/FileReadTool/limits.ts` — 文件读取限制定义
2. `cludecode/constants/toolLimits.ts` — 全局工具结果大小限制
3. `cludecode/utils/toolResultStorage.ts` — 工具结果磁盘持久化
4. `cludecode/services/compact/compact.ts` — 上下文压缩实现
5. `cludecode/Task.ts` — 任务类型体系
6. `cludecode/tools/SendMessageTool/SendMessageTool.ts` — 队友通信工具
7. `cludecode/services/tools/toolExecution.ts` — 工具执行核心流水线
8. `cludecode/memdir/memoryTypes.ts` — Memory 类型定义
9. `cludecode/services/SessionMemory/sessionMemory.ts` — 会话记忆系统
10. `cludecode/tools/AskUserQuestionTool/AskUserQuestionTool.tsx` — 用户澄清工具
11. `cludecode/utils/hooks/hookEvents.ts` — Hooks 事件系统
12. `cludecode/services/tools/toolHooks.ts` — Hooks 执行逻辑
13. `cludecode/Tool.ts` — 工具类型定义和接口
