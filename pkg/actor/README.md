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

### 3. 安全防线与最后拦截 (Security & The Last Line of Defense)
在 Agent-as-a-Tool 的架构中，Actor 是整个系统中最危险、也是最接近物理机器的组件。由于删除、修改等高危指令可能并非出自顶层 Agent 的直接意图，而是来自于某个被挂载的 `Skill` (Markdown 文本)，因此试图在 Thinker 层面做语义拦截既复杂又不可靠。**Actor 必须作为“原子级”的最后一道防线。**

我们抛弃了复杂且难以模拟真实外部环境（如发邮件、删库）的 Docker/Wasm 沙箱，转而采用一种更务实、类似 OS 权限提升（Sudo）的安全授权模型：

- **工具安全级别定级 (Security Tiers)：**
  系统必须要求所有被注册的 Tool 显式声明其安全级别。
  - `Level 0 (Safe)`: 纯查询类，无副作用（如 `Calculator`, `Read`, `Grep`, `LS`）。Actor 直接放行。
  - `Level 1 (Sensitive)`: 有明确上下文边界的写操作（如在授权的 Workdir 下执行 `Write` 或 `Edit`）。Actor 结合白名单机制放行。
  - `Level 2 (High Risk)`: 一切未知的破坏性操作或强副作用操作（如 `Bash` 任意执行, 包含 DROP/DELETE 的 `DatabaseTool`, `EmailSend` 等）。**必须进行人工干预。**

- **基于 Pipeline Hook 的优雅实现 (Elegant Security via Hooks)：**
  我们在设计上遵循“框架提供机制而非策略”。Actor 本身不应该写死复杂的交互拦截代码，而是得益于 `gochat` 的 Pipeline 机制，通过在 `Actor Step` 前面注册 **Security Hook (安全拦截钩子)** 来实现。
  - 如果应用层在组装 Engine 时没有挂载这个 Security Hook，相当于授予 Agent `root` 权限（适用于全自动的受信任任务或本地实验）。
  - 如果挂载了该 Hook，在执行任意 Tool 之前，Hook 会读取该工具的 `SecurityLevel()`。遇到 Level 2 的工具，Hook 就会进入挂起状态，向宿主触发授权事件。人类有三种选择：
    1. **Reject (拒绝)**：彻底拒绝，Actor 不执行，向大模型返回“操作被人类拒绝”的观察结果。
    2. **Approve Once (单次执行)**：允许执行这一次，下次遇到同样的动作仍需再次询问。
    3. **Approve & Whitelist (授权并加白)**：将此工具+参数签名加入会话的白名单上下文，后续免密执行。

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