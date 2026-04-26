# goreact 编排工具体系：协作模式与设计原理

**文档版本**: v1.0
**归档日期**: 2026-04-26
**覆盖范围**: `tools/todo.go`, `tools/task_tools.go`, `tools/subagent.go`, `tools/subagent_list.go`, `tools/team_tools.go`, `reactor/accessor_impl.go`, `reactor/reactor.go`

---

## 一、体系总览

goreact 的工具层分为 **三个层次**，每层解决不同粒度的编排问题：

```
┌─────────────────────────────────────────────────────────────────────┐
│                        工具层次架构                                  │
│                                                                     │
│  Layer 1: 基础操作工具 (无状态, 无 accessor)                        │
│  ─────────────────────────────────────────────────────────────────  │
│    bash, read, write, grep, glob, ls, edit, calculator,           │
│    echo, web_search, web_fetch, repl, cron                         │
│                                                                     │
│  Layer 2: 会话内状态管理 (包级变量 / 全局单例)                      │
│  ─────────────────────────────────────────────────────────────────  │
│    todo_write / todo_read / todo_execute   ← 包级变量 todoStore     │
│    memory_save / memory_search             ← 全局 MemoryAccessor   │
│    email (SMTP/IMAP)                       ← 配置驱动               │
│    skill_create / skill_list                ← 文件系统              │
│                                                                     │
│  Layer 3: Reactor 编排工具 (需要 ReactorAccessor 注入)            │
│  ─────────────────────────────────────────────────────────────────  │
│    TaskCreateTool / TaskResultTool / TaskListTool                   │
│    SubAgentTool / SubAgentResultTool / SubAgentListTool            │
│    TeamCreateTool / SendMessageTool / ReceiveMessagesTool          │
│    TeamStatusTool / TeamDeleteTool / WaitTeamTool                 │
└─────────────────────────────────────────────────────────────────────┘
```

Layer 3 的 12 个编排工具共享同一个设计模式：**通过 `SetAccessor(r)` 接收 Reactor 实例的反向引用**，从而能调用 Reactor 的核心设施（TaskManager、MessageBus、事件总线、子循环执行器）。

---

## 二、四套编排体系的定位与区别

### 2.1 对比矩阵

| 维度 | Todo 系统 | Task 系统 | SubAgent 系统 | Team 系统 |
|------|----------|-----------|---------------|-----------|
| **抽象层级** | 意图/计划层 | 执行层 | 独立 Agent 层 | 多 Agent 协作层 |
| **存储方式** | 包级变量 `todoStore` | `TaskManager`（可持久化） | `TaskManager` + `pendingTasks` map | `AgentMessageBus` |
| **生命周期** | 随进程销毁 | 随 Reactor 实例 | 随 Reactor 实例 | 随 Reactor 实例 |
| **是否执行操作** | ❌ 纯计划引擎 | ✅ RunInline 同步执行 T-A-O | ✅ RunSubAgent 异步启动独立 Reactor | ❌ 纯消息路由 |
| **并发模型** | 单线程 | 同步阻塞当前 goroutine | 可异步 goroutine 或同步 | 多 Agent 并行 + channel 通信 |
| **依赖/DAG** | ✅ Dependencies []string | ✅ ParentID 树形层级 | ✅ ParentID + Metadata | ✅ Team.Membership |
| **结果记录** | 无 Output 字段 | ✅ Output + Error 字段 | ✅ Output + Error + resultCh | ✅ Member.Result |
| **Accessor 依赖** | 不需要 | ✅ ReactorAccessor | ✅ ReactorAccessor | ✅ ReactorAccessor |

### 2.2 层次关系

```
Todo (规划)          Task/SubAgent (执行)         Team (协作)
   │                     │                           │
   │ "我打算做 A,B,C"     │ "A 是简单任务 → task_create"  │ "B 和 C 可以并行"
   │ "B 依赖 A"           │ "C 很复杂 → subagent"       │ "让 @coder 做 B"
   │                     │                           │ "让 @reviewer 做 C"
   ▼                     ▼                           ▼
todo_write ──► todo_execute ──► task_create / subagent ──► team_create + subagent(team_name=...)
                      │                           │
                      ▼                           ▼
                 RunInline()                  MessageBus.JoinTeam()
                 (同步子T-A-O)                 SendMessage() / ReceiveMessages()
```

**典型工作流是自顶向下的分解**：

1. **Todo** 定义「做什么」和「先后顺序」（声明式 DAG）
2. **Task/SubAgent** 决定「怎么做」和「谁来执行」（命令式执行）
3. **Team** 解决「多 Agent 如何协作」（通信与协调）

---

## 三、Todo 系统：声明式计划引擎

### 3.1 数据模型

```go
type TodoItem struct {
    ID            string   // 自动生成: "todo_1", "todo_2" ...
    Status        string   // pending | in_progress | completed | cancelled
    Content       string   // 人类可读的任务描述
    Priority      int      // 越小越优先 (0 = 最高优先)
    Dependencies  []string  // 前置任务 ID 列表 (DAG 边)
    ToolCall      string   // 预绑定工具名 (如 "task_create", "write")
    ToolParams    string   // 预绑定参数 JSON
    AssignedAgent string   // 多 Agent 场景下的委派目标
}
```

### 3.2 三工具职责

#### `todo_write` — CRUD 引擎

- **merge=false**: 全量替换 `todoStore`（适合首次初始化完整计划）
- **merge=true**: 按 ID 增量更新（适合完成一个步骤后标记 completed）
- 自动分配 ID（`nextTodoID()` 自增计数器）
- 自动填充 `CreatedAt` / `UpdatedAt` 时间戳

**输入格式**: `todos` 参数为 JSON 字符串（因为 LLM Function Calling 天然输出字符串）

#### `todo_read` — 只读查询

- 可选 `status` 过滤器（通常只查 `"pending"` 或 `"in_progress"`）
- 使用 RLock 读锁（不阻塞并发的 todo_write）
- 同时返回结构化 `items` 数组和文本 `summary` 摘要

#### `todo_execute` — DAG 调度器

核心算法分三步：

**Step 1 — 构建已完成集合**:
```go
completedSet := make(map[string]bool)
for _, item := range todoStore {
    if item.Status == "completed" { completedSet[item.ID] = true }
}
```

**Step 2 — 依赖分析（拓扑分层）**:
```go
for _, item := range todoStore {
    if item.Status != "pending" { continue }
    allDepsMet := true
    for _, dep := range item.Dependencies {
        if !completedSet[dep] { allDepsMet = false; break }
    }
    // 分入 ready 或 blocked 集合
}
```

**Step 3 — 优先级排序（stable bubble sort）**:
- 冒泡排序保证同优先级保持 FIFO 顺序
- 通常列表 < 20 项，O(n²) 完全够用

输出包含：
- `ready_count` / `blocked_count` — 执行进度概览
- `steps[]ExecStep` — 可执行的步骤列表（含预绑定的 tool_call/tool_params）
- `summary` — 人类可读的执行计划文本

### 3.3 设计哲学

- **LLM as Orchestrator**: LLM 不直接执行，而是通过 todo 定义计划、todo_execute 获取下一步、手动选择
- **Explicit State Machine**: Status 四态显式管理，避免隐式推断
- **Declarative over Imperative**: 声明依赖关系，引擎决定顺序
- **Graceful Degradation**: 依赖未满足时放入 blocked 列表等待下一轮，不报错
- **Tool Pre-binding**: `tool_call` + `tool_params` 让 LLM 在规划阶段就决定执行方式

---

## 四、Task 系统：同步内联子任务

### 4.1 核心工具：TaskCreateTool

这是唯一真正 **执行操作** 的 Task 工具。其 [Execute()](file:///Users/ray/workspaces/ai-ecosystem/goreact/tools/task_tools.go#L56-L106) 流程：

```
1. 解析 description + prompt 参数
2. tm.CreateTask(parentID, description, prompt)  → 注册到 TaskManager
3. tm.UpdateTaskStatus(id, InProgress)
4. emitter(SubtaskSpawned)                    → 通知事件总线
5. accessor.RunInline(ctx, prompt)            ★ 核心调用
6. 根据结果:
   - 成功 → UpdateStatus(Completed) + emitter(SubtaskCompleted)
   - 失败 → UpdateStatus(Failed)    + emitter(SubtaskCompleted{Success:false})
7. 返回 "Task xxx completed. Answer: ..." 给 LLM
```

### 4.2 RunInline：递归 T-A-O 子循环

定义在 [accessor_impl.go:133](file:///Users/ray/workspaces/ai-ecosystem/goreact/reactor/accessor_impl.go#L133-L139):

```go
func (r *Reactor) RunInline(ctx context.Context, prompt string) (answer string, err error) {
    result, runErr := r.Run(ctx, prompt, nil)
    if runErr != nil { return "", runErr }
    return result.Answer, nil
}
```

**关键特性**：
- 在 **当前 goroutine** 内同步执行（阻塞调用方）
- 共享父 Reactor 的 **system prompt、model、tools、event bus**
- 返回值 `answer` 直接注入到当前对话上下文中
- 适用于 **顺序步骤**：先做完 A 再做 B

### 4.3 辅助工具

- **TaskResultTool**: 按 ID 查询已完成任务的结果（含 Status/Description/Output/Error）
- **TaskListTool**: 列出所有任务（支持 parent_id 过滤查看子任务树）

### 4.4 适用场景

TaskCreateTool 适合 **中等复杂度、需要 Agent 思考但不需要并行** 的子任务：
- 重构某个模块（需要分析→设计→实现多轮思考）
- 编写测试用例（需要理解代码结构后生成）
- 生成文档（需要综合多个文件信息）

**不适合**的场景：
- 需要并行探索多个方向 → 用 SubAgent
- 简单的单步操作（grep/read）→ 直接调基础工具即可

---

## 五、SubAgent 系统：异步独立 Agent

### 5.1 核心工具：SubAgentTool

[Execute()](file:///Users/ray/workspaces/ai-ecosystem/goreact/tools/subagent.go#L56-L116) 流程：

```
1. 解析 name(必需), description(必需), prompt(必需)
   + system_prompt(可选), model(可选), team_name(可选)

2. tm.CreateTask(parentID, description, prompt)  → 注册任务
3. tm.UpdateTaskStatus(id, InProgress)

4. 如果指定了 team_name:
   bus.JoinTeam(teamName, name, taskID)      → 加入团队

5. emitter(SubtaskSpawned)

6. 创建 resultCh := make(chan any, 1)
7. accessor.RegisterPendingTask(taskID, resultCh)  → 注册待取结果通道

8. accessor.RunSubAgent(ctx, taskID, sysPrompt, prompt, model, resultCh)  ★ 异步启动

9. 立即返回给 LLM（不等待完成）:
   - 有 team: "SubAgent @xxx (ID: yyy) queued for team zzz..."
   - 无 team:  "SubAgent @xxx (ID: yyy). Use subagent_result with ID yyy to retrieve."
```

### 5.2 RunSubAgent：同步/自适应执行

定义在 [accessor_impl.go:62](file:///Users/ray/workspaces/ai-ecosystem/goreact/reactor/accessor_impl.go#L62-L87)，包含智能决策逻辑：

```go
forceSync := r.config.IsLocal && (model == "" || model == r.config.Model)

if forceSync {
    r.runSubAgentSync(...)   // 本地模型 → 同步（本地模型通常不支持并发）
} else {
    r.runSubAgentAsync(...)  // API 模型 → 异步 goroutine
}
```

**runSubAgentCore** ([accessor_impl.go:103](file:///Usersray/workspaces/ai-ecosystem/goreact/reactor/accessor_impl.go#L103-L128)) 做了什么：

```go
// 1. 创建全新的子 Reactor（继承父 Reactor 的 memory/messageBus/eventBus）
subReactor := NewReactor(subConfig,
    WithMemory(r.memory),
    WithMessageBus(r.messageBus),
    WithEventBus(r.eventBus),
)

// 2. 启动完整的 T-A-O 循环
result, runErr := subReactor.Run(ctx, prompt, nil)

// 3. 更新任务状态 + 通过 channel 发送结果
tm.UpdateTaskStatus(taskID, Completed/Failed, output/error)
resultCh <- answer_or_error

// 4. 清理 pending 注册
r.RemovePendingTask(taskID)
```

**子 Reactor 与父 Reactor 的关系**：

| 属性 | 父 Reactor | 子 Reactor |
|------|-----------|-----------|
| SystemPrompt | 原始 prompt | 可覆盖 (`system_prompt` 参数) |
| Model | 原始 model | 可覆盖 (`model` 参数) |
| Memory | 共享同一实例 | 继承引用 |
| MessageBus | 共享同一实例 | 继承引用（可加入不同 team） |
| EventBus | 共享同一实例 | 继承引用 |
| TaskManager | 共享同一实例 | 继承引用（任务注册在同一 TM 中） |
| Config | 原始配置 | 继承+覆盖 |

### 5.3 结果获取：SubAgentResultTool

[Execute()](file:///Users/ray/workspaces/ai-ecosystem/goreact/tools/subagent.go#144-L193) 实现了带超时的异步结果等待：

```go
ch, exists := accessor.GetPendingTask(taskID)
if !exists {
    // 任务不在 pending 中 → 可能已完成或从未创建
    // 回退到 TaskManager.GetTask() 查询最终状态
    task, _ := accessor.TaskManager().GetTask(taskID)
    return 格式化后的状态信息
}

select {
case result := <-ch:
    // 收到结果！移除 pending 记录
    accessor.RemovePendingTask(taskID)
    return 格式化结果
case <-time.After(time.Duration(waitSeconds) * time.Second):
    // 超时 → 返回当前进度，不报错
    return "SubAgent Task xxx is still running (status: in_progress)..."
}
```

### 5.4 SubAgentListTool

列出所有 SubAgent 类型的任务（通过检查 `Metadata["subagent_name"] != nil` 过滤），展示每个 SubAgent 的 name/model/team/status/output。

### 5.5 Task vs SubAgent 选择指南

```
场景判断流程:

任务是否需要 LLM 多轮思考？
  ├── 否 → 直接用基础工具 (bash, read, write...)
  │
  └── 是 → 需要与其他任务并行吗？
        ├── 否 → TaskCreateTool (RunInline, 同步, 共享 context)
        │         适用于: 顺序重构、逐步实现、单线深度分析
        │
        └── 是 → SubAgentTool (RunSubAgent, 异步/自适应, 独立 context)
                  适用于: 并行研究 (@researcher + @reviewer)
                          大规模代码生成 (@coder-A + @coder-B)
                          需要不同 model 或 system_prompt 的子任务
```

---

## 六、Team 系统：多 Agent 协作基础设施

### 6.1 核心概念

Team 系统建立在 **AgentMessageBus** 之上，提供：

- **Team** (团队): 容器，包含名称、描述、成员列表
- **Member** (成员): 每个 Agent 在团队中的身份，含 Name/TaskID/Status/Result
- **Mailbox** (邮箱): 每个 Agent 一个 buffered channel (`chan *AgentMessage`, cap=64)
- **Message** (消息): 含 From/To/Type/Content/Summary/Timestamp

### 6.2 消息类型

| Type | 方向 | 用途 |
|------|------|------|
| `message` | 点对点 | Agent A 发送给 Agent B |
| `broadcast` | 一对多 | Agent 发送给所有团队成员 |
| `shutdown_request` | 控制 | 主 Agent 要求某 Agent 终止 |
| `shutdown_response` | 控制 | Agent 确认已终止 |

### 6.3 工具清单与协作流

#### TeamCreateTool — 创建团队

```go
bus := accessor.MessageBus()
team, _ := bus.CreateTeam(name, description)
// → 返回 team.ID, 后续所有操作都用 team_ID
```

#### SubAgentTool + Team — 加入团队

```go
// SubAgentTool.Execute() 中:
if teamName != "" && bus != nil {
    bus.JoinTeam(teamName, name, taskID)  // 自动加入 + 创建 mailbox
}
// SubAgent 启动后自动成为团队成员
```

#### SendMessageTool — 发送消息

```go
bus.SendMessage(teamID, fromAgent, toAgent, msgType, content, summary)
// direct: 投递到 toAgent 的 mailbox
// broadcast: 投递到除 from 外的所有成员 mailbox
```

#### ReceiveMessagesTool — 接收消息

```go
messages := bus.ReceiveMessages(agentName)
// 非阻塞读取 mailbox 中所有可用消息
// 支持可选 wait_seconds 阻塞等待
```

#### TeamStatusTool — 查看团队状态

```go
team, _ := bus.GetTeam(teamID)
// 返回成员列表及其 status/running/completed/failed/result
```

#### WaitTeamTool — 等待全部完成

```go
for time.Now().Before(deadline) {
    team := bus.GetTeam(teamID)
    allDone := 所有 member.Status != "running"
    if allDone { return buildTeamResult(team) }  // 汇总所有成员结果
    time.Sleep(2 * time.Second)
}
// 超时返回部分结果 + WARNING
```

#### TeamDeleteTool — 清理

```go
bus.DeleteTeam(teamID)
// 关闭所有成员 mailbox + 移除团队记录
```

### 6.4 典型多 Agent 协作流程

```
主 Agent (main):
│
│  1. team_create(name="refactor-squad", description="代码重构团队")
│     → team_id: "team_1"
│
│  2. subagent(name="@architect", description="设计重构方案",
│              system_prompt="你是架构师...", team_name="team_1")
│     → SubAgent @architect 启动, 自动加入 team_1
│
│  3. subagent(name="@implementer", description="实现重构",
│              system_prompt="你是实现者...", team_name="team_1")
│     → SubAgent @implementer 启动, 自动加入 team_1
│
│  4. subagent(name="@tester", description="编写测试",
│              system_prompt="你是QA...", team_name="team_1")
│     → SubAgent @tester 启动, 自动加入 team_1
│
│  5. send_message(type="broadcast", content="开始重构 auth 模块...")
│     → @architect, @implementer, @tester 都收到
│
│  ... 各 Agent 独立运行, 通过 send/receive 协作 ...
│
│  6. wait_team(team_id="team_1", timeout_seconds=300)
│     → 阻塞直到所有成员 running → completed/failed
│     → 返回汇总结果
│
│  7. team_delete(team_id="team_1")  // 清理
```

---

## 七、ReactorAccessor 注入机制

### 7.1 为什么需要 Accessor？

工具层 (`tools/`) 不能直接依赖 reactor 层 (`reactor/)），否则会形成循环依赖。解决方案：

```
tools/          reactor/
  ├─ task_tools.go    ├─ reactor.go
  ├─ subagent.go     ├─ accessor_impl.go  ◄── 实现 ReactorAccessor 接口
  ├─ team_tools.go   └─ ...
  └─ reactor_accessor.go              ◀── 定义接口（tools 包拥有接口定义权）
```

**接口定义在 tools 包，实现在 reactor 包** — 依赖方向：tools → core（接口）← reactor（实现）

### 7.2 接口定义

[reactor_accessor.go](file:///Users/ray/workspaces/ai-ecosystem/goreact/tools/reactor_accessor.go#L13-L51):

```go
type ReactorAccessor interface {
    TaskManager() core.TaskManager
    MessageBus() *core.AgentMessageBus
    EventEmitter() func(core.ReactEvent)
    RegisterPendingTask(taskID string, resultCh chan any)
    GetPendingTask(taskID string) (<-chan any, bool)
    RemovePendingTask(taskID string)
    RunInline(ctx context.Context, prompt string) (answer string, err error)
    RunSubAgent(ctx context.Context, taskID string, systemPrompt, prompt string,
               model string, resultCh chan<- any)
    Scheduler() *core.CronScheduler
    Config() ReactorConfig
}
```

### 7.3 注入时机

[NewReactor()](file:///Users/ray/workspaces/ai-ecosystem/goreact/reactor/reactor.go#L181) → [registerOrchestrationTools()](file:///Users/ray/workspaces/ai-ecosystem/goreact/reactor/reactor.go#L268) → [accessor_impl.go:158-216](file:///Users/ray/workspaces/ai-ecosystem/goreact/reactor/accessor_impl.go#L158-L216):

```go
func NewReactor(config ReactorConfig, opts ...ReactorOption) *Reactor {
    r := &Reactor{config: config, ...}
    // ... 初始化各组件 ...
    r.registerOrchestrationTools()  // ← 在构造函数末尾注入
    return r
}

func (r *Reactor) registerOrchestrationTools() {
    // 每个编排工具: NewXxx() → SetAccessor(r) → RegisterTool()
    // 共注入 12 个工具，全部获得同一个 Reactor 实例的引用
}
```

编译期断言 ([accessor_impl.go:11](file:///Users/ray/workspaces/ai-ecosystem/goreact/reactor/accessor_impl.go#L11)) 确保 Reactor 完整实现接口：
```go
var _ tools.ReactorAccessor = (*Reactor)(nil)
```

### 7.4 两种注入模式对比

| 模式 | 使用方 | 注入方式 | 特点 |
|------|--------|---------|------|
| **实例方法 SetAccessor** | Task/SubAgent/Team (12 个工具) | 构造时 `tool.SetAccessor(r)` | 每个工具有独立 accessor，支持多 Reactor 实例隔离 |
| **全局变量 SetMemory** | MemorySave/MemorySearch (2 个工具) | `SetMemory(m)` 全局设置 | 全局共享，不绑定具体 Reactor |

选择依据：Task/SubAgent/Team 的操作 **必须发生在特定 Reactor 上下文内**（RunInline/RunSubAgent/MessageBus 都属于某个 Reactor），而 Memory 是跨会话的全局长期记忆。

---

## 八、端到端协作示例

### 示例 1: 顺序执行（Todo + TaskCreate）

```
用户: "重构 auth 模块，然后加单元测试"

=== T-A-O Iteration 1 (Think) ===
LLM: 这是一个复杂的多步骤任务，我先用 todo_write 规划

=== Act ===
todo_write(todos=[
  {"content":"分析现有 auth 代码结构",   priority:1},
  {"content":"重构 Auth 中间件",          priority:2, deps:["todo_1"], tool_call:"task_create",
   tool_params:"{\"description\":\"重构Auth中间件\",\"prompt\":\"请分析...\"}"},
  {"content":"编写单元测试",              priority:3, deps:["todo_2"], tool_call:"task_create",
   tool_params:"{\"description\":\"编写auth单元测试\",\"prompt\":\"请基于...\"}"}
], merge=false)
→ success, count=3

=== T-A-O Iteration 2 (Act) ===
todo_execute()
→ ready_count=1, steps=[{todo_1: 分析代码}]
→ LLM 选择执行 t1: 用 grep/read 分析代码...

=== Act (完成后) ===
todo_write(todos=[{"id":"todo_1","status":"completed"}], merge=true)

=== T-A-O Iteration 3 ===
todo_execute()
→ ready_count=1, steps=[{todo_2: 重构Auth}]  (t1 已完成, t2 解除阻塞)
→ LLM 调用 task_create(description="重构Auth中间件", prompt="详细指令...")
  → TaskCreateTool.Execute()
    → RunInline() 启动子 T-A-O 循环
    → 子循环完成: "Task task_4 completed. Answer: 重构完成，主要改动..."
  → 返回给主 LLM

=== 后续迭代 ===
重复: todo_execute → task_create → todo_write(completed) → 下一步...
直到所有 todo 完成
```

### 示例 2: 并行探索（Todo + SubAgent + Team）

```
用户: "全面审查这个项目的代码质量"

=== Think ===
LLM: 这需要从多个角度并行审查，我用团队模式

=== Act ===
team_create(name="code-review", description="代码质量审查团队")

subagent(name="@security-expert", description:"安全审查",
         system_prompt:"你是一个安全专家...", team_name="code-review")
subagent(name="@perf-engineer",   description:"性能审查",
         system_prompt:"你是一个性能优化专家...", team_name="code-review")
subagent(name="@architect",      description:"架构审查",
         system_prompt:"你是一个架构师...", team_name="code-review")

→ 3 个 SubAgent 异步启动（API 模型 → goroutine 并行）

=== Act (主 Agent 做自己的事) ===
send_message(type="broadcast", content="请重点关注认证和授权模块")

... 主 Agent 继续其他工作 ...

=== Act (收集结果) ===
wait_team(team_id="team_xxx", timeout_seconds=300)
→ 阻塞等待 3 个 Agent 全部完成
→ 返回:
  Team "code-review" — All members finished.
  === @security-expert (status: completed) ===
    发现 3 个 SQL 注入风险点，2 个 XSS 漏洞...
  === @perf-engineer (status: completed) ===
    数据库查询有 N+1 问题，建议添加索引...
  === @architect (status: failed) ===
    (no result)

=== 最终 ===
team_delete(team_id="team_xxx")
→ 基于 3 个 Agent 的审查结果生成综合报告
```

---

## 九、设计原则总结

### 9.1 分层解耦

- **工具不知道 Reactor 的存在** — 只知道 `ReactorAccessor` 接口
- **Reactor 不知道工具的具体实现** — 只通过接口调用
- **接口定义权归属消费者方** (tools 包) — 符合 DIP (Dependency Inversion Principle)

### 9.2 渐进复杂度

```
简单任务  →  基础工具 (bash, read, write)
              ↓ 够复杂?
顺序多步  →  Todo + TaskCreate (计划 + 同步执行)
              ↓ 需要并行?
并行探索  →  SubAgent (独立 Agent, 异步/自适应)
              ↓ 需要协作?
多角色    →  SubAgent + Team (消息通信 + 团队协调)
```

Agent 不必一开始就用最复杂的模式。从简单开始，按需升级。

### 9.3 弹性容错

- **Todo blocked 不报错** — 放入 blocked 列表等下一轮
- **SubAgentResult 超时不崩溃** — 返回当前进度，LLM 可稍后重试
- **WaitTeam 超时有警告** — 返回已完成的部分结果
- **TaskCreate 失败有错误传播** — error 信息写入 Task.Error + 事件发射
- **SendMessage mailbox full 不阻塞** — non-blocking send，满则跳过

### 9.4 可观测性

所有编排操作都通过 EventEmitter 发送事件：

| 事件 | 触发时机 | 携带数据 |
|------|---------|---------|
| `SubtaskSpawned` | TaskCreate / SubAgent 开始执行 | TaskID, Description |
| `SubtaskCompleted` | TaskCreate / SubAgent 结束 | TaskID, Success, Answer/Error |

外部监听器（如 UI、日志系统）可以订阅这些事件来构建实时进度面板。

### 9.5 测试友好性

所有编排工具通过 `SetAccessor()` 接收依赖，测试时可注入 mock：

```go
mockAcc := &mockReactorAccessor{
    taskManager: core.NewInMemoryTaskManager(),
    messageBus:  core.NewAgentMessageBus(),
    pending:     make(map[string]chan any),
    runInlineFn: func(ctx, prompt) (string, error) {
        return "mock answer", nil
    },
}
tool.SetAccessor(mockAcc)
// tool.Execute() 不再需要真实 Reactor
```

这正是我们在测试中使用的模式（见 `subagent_task_team_test.go`）。
