# Tools 工具库与 Tool RAG 系统设计

本目录 (`pkg/tools`) 定义了 GoReAct 架构中的基础行动单元——**工具 (Tool)**，并提供了内置的核心工具集。

## 1. Tool 在系统中的定位

在 GoReAct 的设计哲学中：
- **Tool 是一种单一的、原子的执行方法（Action）。** 它类似于操作系统的 CLI 命令（如 `ls`, `grep`）或编程语言中的一个具体函数。
- **Tool 强调“不求多，只求实用”。** 我们不主张将框架设计成一个臃肿的万能胶水库。Tool 是用来打地基的，它负责最原子的操作（读写文件、执行 Bash、计算）。

更高级的业务逻辑应该由 **Skill** 去编排，或者通过把其他 **Agent** 包装成 Tool（Agent-as-a-Tool）来实现。

---

## 2. 核心架构设计

### 2.1 Tool 接口
任何能力，只要实现了以下极其简洁的接口，就可以成为引擎可调用的工具：
```go
type SecurityLevel int

const (
    LevelSafe      SecurityLevel = 0 // 纯查询、无副作用（如 Calculator, Grep）
    LevelSensitive SecurityLevel = 1 // 敏感或有边界的写操作（如 Write, Edit）
    LevelHighRisk  SecurityLevel = 2 // 高危、不可预测的破坏性操作（如 Bash, 删库）
)

type Tool interface {
	Name() string
	Description() string // 极其重要：写给大模型看的语义说明
    SecurityLevel() SecurityLevel // 向 Actor 声明自己的危险等级
	Execute(ctx context.Context, input map[string]any) (any, error)
}
```

### 2.2 ToolManager 接口与 RAG Tool Manager
`Manager` 是 `Thinker` 发现工具和 `Actor` 执行工具的注册中心。

**当前的默认实现 (`SimpleManager`)：** 仅仅是一个简单的 `map[string]Tool`。这种方式在工具数量较少时运行良好，但在面对海量工具时会遭遇严重的瓶颈（由于上下文窗口限制，无法将成百上千个工具的 Prompt 一次性塞给 LLM）。

**未来的落地形态 (RAG Tool Manager)：**
在生产级的 GoReAct 落地中，应用层应当实现基于向量检索的 `Manager` 接口。
当 `Thinker` 需要执行任务时，它调用 `ListAvailableTools(ctx, intent)`，底层的 RAG 系统会将用户的意图 Embedding，从海量工具库中召回最匹配的 Top-K 个工具的 Schema 注入到当前轮次的提示词中。这解决了长上下文瓶颈问题。

---

## 3. 关于 MCP (Model Context Protocol) 的战略态度

在 `pkg/tools/provider/mcp` 中，我们提供了对 MCP 的兼容接入能力。但作为框架的战略方向，**我们不推荐将 MCP 作为主力扩展模式。**

**我们的技术观点：**
MCP 试图建立一套连接 AI 与外部工具的庞大协议，但在实战中，它表现出了明显的缺点：
1. **过度设计与臃肿**：它像当年的 SOAP 一样，把原本只需一个简单 RESTful 调用的事情，包装成了需要数百兆内存（如引入整个 Node.js 运行时）和繁琐握手的怪物。
2. **极差的分布式扩展性**：复杂的有状态连接机制使得在多机部署或高并发 Agent 调度时容易产生雪崩。

**替代方案：Skill-Driven API 与硬核二进制注册**
在 GoReAct 生态中，我们坚信：
1. **轻量级网络请求**：要让大模型访问外部系统，给 Agent 写一个简单的 `Skill`（一段 Markdown），教大模型如何使用内置的 `Bash` 或原生的 RESTful API，这比部署 MCP Server 要高效得多。
2. **硬核能力扩展**：如果确实需要扩展极高性能或复杂的本地计算能力，GoReAct 推荐的第二种扩展方向是**直接向 ToolManager 注册“任意”可执行程序（CLI Binaries）**。这比搞一套沉重的 MCP 协议更有价值、更具工程美感，也完全符合操作系统级的第一性原理。

---

## 4. 内置工具集 (Built-in Tools)

`pkg/tools/builtin` 提供了一套经过严苛修剪的、核心实用工具集。这套工具是对标业界顶尖实践（如 Claude Code）打造的。

### 4.1 设计原则
- **职责单一**：比如废弃大而全的 `filesystem`，拆分为纯粹的 `Read`, `Write`, `Edit`。
- **降维合并**：废弃了 `HTTP`, `Curl`, `Port` 等工具，统一交由强大的 `Bash` 工具通过原生 shell 完成，彻底消除工具冗余。

### 4.2 核心工具清单 (Tier 1 & Tier 2)

| 工具分类 | 工具名称 | 核心能力 |
|---------|---------|---------|
| **文件操作** | **Read** | 读取文件，支持指定行范围与大小限制。 |
| | **Write** | 写入或追加文件，自动处理目录创建。 |
| | **Edit** | 极其强大的多位置精确编辑（Diff 修改），专为代码重构设计。 |
| **搜索与浏览** | **Glob** | 高效的文件名匹配与枚举。 |
| | **Grep** | 正则文本搜索，带文件类型与行列定位。 |
| | **LS** | 目录树状列表展示。 |
| **基础扩展** | **Bash** | 原生 Shell 命令执行引擎。 |
| | **Calculator** | 基础数学计算。 |
| | **DateTime** | 时间获取与格式化。 |

### 4.3 为什么没有 Git、Docker 或 HTTP 工具？（设计约束）

细心的开发者可能会发现，内置工具集中没有提供针对 `Git`、`Docker` 或特定网络请求（如 `HTTP` / `Curl`）的独立工具。
这正是 GoReAct **"Skill 优先"** 理念的体现：
- 像发 HTTP 请求、拉取 Git 代码这种事情，完全不应该用 Go 语言去硬编码一个僵硬的 `tool`。
- 最佳实践是：使用内置的 `Bash` 工具作为通用底座，然后编写一个 `GitSkill` 或 `RestAPISkill` (Markdown 形式)，教导大模型如何使用 Bash 组合命令来完成这些高级操作。
这不仅极大地缩减了框架维护成本，还赋予了 Agent 处理未知情况的强大适应力。

*获取更详细的使用文档，请参阅内置的 [CHEATSHEET.md](./builtin/CHEATSHEET.md) 和 [USAGE_GUIDE.md](./builtin/USAGE_GUIDE.md)*
