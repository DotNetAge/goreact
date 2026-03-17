# Actor (执行者)

`Actor` 位于 GoReAct 架构（Thinker - Actor - Observer - Terminator）的第二环。如果说 Thinker 是 Agent 的“大脑”，那么 Actor 就是它的“双手”。

它专门负责接收 Thinker 产生的结构化指令（`Action` 和 `ActionInput`），并与外部环境（API、数据库、文件系统、代码沙箱等）进行实质性的交互，最终产生原始的执行结果供 `Observer` 观察。

---

## 核心职责 (Core Responsibilities)

一个工业级的 Actor 模块不仅需要“能调用工具”，还需要具备极高的健壮性和安全性。具体包含以下四大核心能力：

### 1. 工具路由与参数绑定 (Tool Routing & Binding)
Thinker 输出的往往是文本或 JSON，Actor 需要将其转化为对实际 Go 语言函数或底层协议的调用：
- **工具发现与注册表匹配：** 根据解析出的工具名称（`ToolName`），在系统的工具注册表（Tool Registry）中快速路由到具体的工具实现。
- **参数校验与反序列化：** 将大模型生成的 `ActionInput`（通常是 JSON 字符串）反序列化为 Go 的强类型结构体，并验证必填字段和数据类型是否符合工具的 Schema 规范。

### 2. 并发与编排执行 (Concurrency & Orchestration)
现代大模型（如 GPT-4, Claude 3）原生支持并行函数调用（Parallel Function Calling）：
- **并行执行：** 当 Thinker 决定同时调用多个独立工具（例如：同时查询北京、上海、广州的天气）时，Actor 负责利用 Go 的 goroutine 发起高并发请求，并使用 `errgroup` 或 `WaitGroup` 聚合执行结果。
- **依赖执行：** 如果存在局部依赖的动作组合，Actor 需要按照依赖图顺序执行（不过通常复杂的依赖会交给 Thinker 去做规划拆解，Actor 尽量保持纯粹）。

### 3. 安全沙箱与人工介入 (Security & Human-in-the-Loop)
Actor 是整个系统中最危险的组件，它能真实地改变世界状态（如写库、发邮件、执行命令）：
- **人工审查拦截 (Human-in-the-Loop, HITL)：** 针对高危操作（如 `DeleteDatabase` 或 `SendEmail`），Actor 提供中断机制，暂停管线执行，向终端用户发送确认请求（Approve/Reject），待人工授权后再继续执行。
- **代码沙箱隔离 (Sandboxing)：** 如果工具是执行 Python 代码或 Shell 脚本，Actor 负责通过 Docker、WASM 或 gVisor 建立安全隔离的执行环境，防止恶意代码破坏宿主系统。
- **权限边界与审计：** 严格控制 Agent 的 API Token 和权限域，记录所有越界尝试。

### 4. 局部容错与鲁棒性 (Fault Tolerance & Local Retry)
并非所有的错误都需要退回给 Thinker 去“反思”，有些错误在 Actor 层面就可以被消化：
- **局部重试机制：** 对于偶发的网络超时、HTTP 502 等非致命错误，Actor 内部自动执行退避重试（Exponential Backoff），避免无谓地消耗大模型 Token 去处理网络波动。
- **超时控制与熔断：** 通过 `context.Context` 对工具执行时间实施强力阻断，防止某个工具死锁导致整个 Agent 管线挂起。

---

## 架构集成位置 (Integration in ReAct Loop)

在一次典型的 GoReAct 执行循环中，Actor 的位置如下：

1. **[Thinker]** 经过推理，输出即将执行的操作：`Action: SearchEngine`, `ActionInput: {"query": "Go 1.25 release notes"}`。
2. **[Actor]** 从全局上下文中接收到上述指令。
3. **[Actor]** 拦截与权限校验（触发中间件，例如请求用户授权）。
4. **[Actor]** 执行工具逻辑（发起真实的 HTTP 请求），捕获成功的结果数据或发生的 panic/error。
5. **[Observer]** 接收 Actor 产生的“原始输出”或“系统错误”，将其格式化为大模型易于理解的文字反馈。

## 设计指引 (Design Guidelines)

- **接口隔离：** `Actor` 必须是接口，允许用户实现 `LocalActor`、`RemoteActor` (通过 gRPC 调用的分布式节点) 甚至 `MockActor` (用于单测)。
- **Actor Hooks：** 由于Actor也是可以通过Pipeline模式将每个处理制作成为一个独立的步骤，gochat 的 Step 是可以支持自定义的Hook。允许在工具执行前后插入逻辑（如：鉴权拦截器、限流器、Metrics 打点、链路追踪 Trace 注入）。
- **返回值标准化：** Actor 的输出不应该仅仅是一个 `string`，而应该是一个结构化的 `ExecuteResult` 对象，包含：执行时长、原始字节流、执行状态码、安全告警等元数据，方便 `Observer` 提炼。