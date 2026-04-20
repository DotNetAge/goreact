# goreact Memory/RAG 接口设计：抑制幻觉与动态上下文

## 执行摘要

goreact 当前在 `core/memory.go` 中定义了一个极简的 `Memory` 接口（`Search` + `Save`），但该接口存在三个核心问题：(1) 接口过于简单，无法支撑 RAG 检索和结构化记忆管理；(2) Memory 与 Reactor 完全解耦，`Agent` 持有 `*core.Memory` 但 Reactor 从未使用它；(3) 没有实现"动态上下文"——即按相关性而非时间顺序从记忆中检索上下文。本报告基于对 ClueCode 知识管理架构的深度分析、goreact 现有代码库的全面审计，以及 gorag（同仓库 RAG 引擎）的能力评估，提出一套分层 Memory 架构设计方案，并通过直接代码实现验证其可行性。

## 背景

goreact 的 TODO.md 第 18-19 行明确列出了两个待解决目标：

1. **TODO Line 18**：在 T-A-O 循环中集成 RAG 以抑制大模型幻觉，设计一个外部可实现的 Memory 接口
2. **TODO Line 19**：通过 Memory(RAG) 实现动态上下文模式，按相关性检索而非时间顺序获取全部上下文

当前实现现状：
- `core/memory.go` 定义了 `Memory` 接口，仅包含 `Search(query string, memType MemoryType) ([]string, error)` 和 `Save(value string, memType MemoryType) error`
- `MemoryType` 有三种：`ShortTerms`、`LongTerms`、`Refactive`
- `Agent` 结构体持有 `*core.Memory`，但在 `Ask()`/`AskWithContext()` 方法中从未调用
- Reactor 不知道 Memory 的存在，Think 阶段构建 prompt 时没有注入任何记忆内容
- 没有提供任何 Memory 接口的实现（无 InMemory 版本，无 RAG 版本）

## 研究发现

### 一、当前 Memory 接口的缺陷分析

**接口粒度不足**：现有 `Search` 返回 `[]string`，丢失了所有结构化信息（来源、类型、相关性分数、时间戳）。当 Think 阶段需要知道"这条记忆是用户偏好还是项目规范"时，无法区分。

**缺少 Update/Delete 操作**：记忆需要生命周期管理，过时的信息需要更新或删除，当前接口只支持追加式写入。

**无上下文感知**：`Search` 接收裸字符串 query，没有 session scope 的概念。无法区分"这次对话的短期记忆"和"跨会话的长期记忆"。

**无与 gorag 的桥接能力**：同仓库的 gorag 项目提供了完整的 RAG pipeline（文档分块、向量化、混合检索），但 Memory 接口的设计使得 gorag 无法自然地作为底层实现。

### 二、ClueCode 知识管理架构的启示

ClueCode 通过四层互补机制管理外部知识：

**第一层：结构化 Memory（memdir/）**
- 四种记忆类型：user（用户偏好）、feedback（用户反馈/指导）、project（项目上下文）、reference（外部资源指针）
- 每种记忆定义了 `when_to_save`（何时保存）、`how_to_use`（如何使用）、`body_structure`（正文结构）
- 核心原则：Memory 存储的是"无法从当前项目状态推导出的上下文"

**第二层：Session Memory（会话记忆）**
- 使用 forked subagent 在后台自动从对话中提取关键信息
- 生成结构化的 Session Memory 文件，记录本次对话的关键决策和发现
- 通过 `postSamplingHook` 注册，在每个 LLM 响应后检查是否需要更新

**第三层：MCP（Model Context Protocol）**
- MCP 工具通过 `mcp__server__tool` 命名空间暴露给 LLM
- 支持 stdio、SSE、HTTP、WebSocket 等传输方式

**第四层：Skill 系统**
- 每个 Skill 本质上是一段可加载的结构化系统提示词
- 通过关键词匹配自动激活

### 三、gorag 的 RAG 能力评估

gorag 项目位于同仓库 `/gorag/` 目录，提供：
- `chunker/`：文本分块（语义分块、固定大小分块等）
- `embedder/`：向量化（ONNX 模型，支持本地推理）
- `store/`：向量存储（基于 bleve）
- `query/`：查询处理
- `result/`：结果排序
- `hybrid.go`：混合检索（BM25 + 向量相似度）

gorag 的 `svc.go` 提供了 `RAGService`，可以作为 Memory 接口的 RAG 实现。

## 设计方案

### 核心设计原则

1. **渐进增强**：Memory 接口必须支持从最简单的 InMemory 实现到完整的 RAG 实现，调用者无需修改代码
2. **ContextWindow 集成**：Memory 检索结果是 ContextWindow 的补充而非替代，两者协同工作
3. **Reactor 感知**：Reactor 在 Think 阶段自动查询 Memory，将相关记忆注入 system prompt
4. **类型安全**：每条记忆携带类型信息，Think prompt 可据此决定如何使用

### 重构后的 Memory 接口

```go
// MemoryRecord represents a single piece of stored knowledge.
type MemoryRecord struct {
    ID        string
    Type      MemoryType   // user, project, reference, session
    Title     string       // brief title for context injection
    Content   string       // the actual knowledge content
    Scope     MemoryScope  // private (user-level) or team (shared)
    Tags      []string     // optional tags for keyword matching
    Score     float64      // relevance score from Search (0.0 = not scored)
    CreatedAt time.Time
    UpdatedAt time.Time
}

type MemoryType int
const (
    MemoryTypeUser       MemoryType = iota // user preferences/conventions
    MemoryTypeProject                       // project-specific context
    MemoryTypeReference                     // external resource pointers
    MemoryTypeSession                       // auto-extracted session memory
)

type MemoryScope int
const (
    MemoryScopePrivate MemoryScope = iota // user-level
    MemoryScopeTeam                        // team-level (shared)
)

// Memory is the core interface for knowledge retrieval and storage.
// Implementations range from simple in-memory to full RAG.
type Memory interface {
    // Retrieve searches memory for records relevant to the query.
    Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]MemoryRecord, error)

    // Store persists a new memory record.
    Store(ctx context.Context, record MemoryRecord) (string, error)

    // Update modifies an existing memory record by ID.
    Update(ctx context.Context, id string, record MemoryRecord) error

    // Delete removes a memory record by ID.
    Delete(ctx context.Context, id string) error
}

// RetrieveOption configures Retrieve behavior.
type RetrieveOption func(*RetrieveConfig)

type RetrieveConfig struct {
    Types  []MemoryType  // filter by types (empty = all)
    Scope  MemoryScope   // filter by scope (0 = all)
    Limit  int           // max results (0 = default 5)
    MinScore float64     // minimum relevance score (0 = no filter)
}
```

### Memory 与 ContextWindow 的协作模式

关键设计：Memory 不是替代 ContextWindow，而是为其提供"语义检索"能力。

```
用户输入 → Intent分类 → Think阶段
                          ↓
                    Memory.Retrieve(input)
                          ↓
               相关记忆注入 system prompt
                          ↓
                 ContextWindow（时间顺序的历史消息）
                          ↓
                    LLM 推理 → Act → Observe
```

在 Think 阶段，Reactor 会：
1. 从 `Memory.Retrieve(ctx.Input)` 获取与当前输入相关的记忆
2. 将相关记忆格式化后注入 Think prompt 的 `<relevant_memory>` section
3. LLM 同时看到相关记忆（语义相关）和 ConversationHistory（时间顺序）
4. 如果在 T-A-O 循环中产生了值得保存的决策，可以通过 `memory_save` 工具保存

### 提供两种内置实现

**InMemoryMemory**：用于快速开发和测试，基于 map + 关键词匹配

**gorag Memory 适配器**：将 gorag 的 RAGService 包装为 Memory 接口实现

## 实现概述

以下代码变更已在 goreact 项目中直接实现：

1. **`core/memory.go`** — 重写，定义完整的 Memory 接口、MemoryRecord、RetrieveOption
2. **`core/memory_inmemory.go`** — 新增 InMemoryMemory 实现（关键词匹配 + map 存储）
3. **`reactor/reactor.go`** — 在 Reactor 结构体中集成 Memory，新增 `WithMemory` 选项
4. **`reactor/reactor.go`** — 修改 Think 阶段，自动从 Memory 检索相关记忆注入 prompt
5. **`reactor/prompts/think_prompt.tmpl`** — 添加 `<relevant_memory>` section
6. **`reactor/prompts.go`** — thinkPromptData 新增 MemorySection 字段
7. **`reactor/thought.go`** — BuildThinkPrompt 接受 memoryRecords 参数
8. **`tools/memory.go`** — 新增 `memory_save` 和 `memory_search` 工具，LLM 可主动管理记忆
9. **`agent.go`** — 更新 Agent 使用新的 Memory 接口

### Think 阶段的记忆注入流程

```
Think(ctx):
  1. 正常构建 intent + tools + skills
  2. 如果 reactor.memory != nil:
     records = memory.Retrieve(ctx.Input, WithTypes(MemoryTypeProject, MemoryTypeUser))
     memorySection = FormatMemoryRecords(records)
  3. BuildThinkPrompt(..., memoryRecords)
  4. LLM 看到:
     <relevant_memory>
     ## Project: goreact 使用 Go Template 渲染 prompt
     ## User: 用户偏好中文回复
     </relevant_memory>
  5. LLM 基于记忆+上下文做出更准确的决策，减少幻觉
```

### memory_save 工具的设计

LLM 在 T-A-O 循环中可以通过 `memory_save` 工具主动保存重要发现：

```go
// memory_save: LLM saves important findings/conventions to long-term memory
tools.NewMemorySaveTool(memory)
// Input: { "title": "goreact prompt 架构", "content": "...", "type": "project" }
```

这对应 ClueCode 的 Session Memory 机制——从对话中自动提取关键信息持久化。

## 局限性与后续工作

1. **gorag 适配器未实现**：本次仅实现了 InMemoryMemory。gorag 的 RAGService 适配需要 gorag 暴露更清晰的接口，建议后续单独实现。
2. **自动记忆提取未实现**：ClueCode 使用后台 subagent 自动从对话中提取记忆，goreact 目前依赖 LLM 主动调用 `memory_save` 工具。后续可在 Observe 阶段后添加自动提取 hook。
3. **记忆冲突解决**：当多条记忆存在矛盾时（如用户偏好前后不一致），需要定义更新策略。当前 Update 按 ID 覆盖，后续可添加时间戳比较和 merge 策略。
4. **性能基准测试**：InMemoryMemory 在大量记忆（>10,000条）时的检索性能未测试，关键词匹配可能成为瓶颈。

## 参考资料

1. [ClueCode memdir/memoryTypes.ts](https://github.com/anthropics/claude-code/blob/main/cludecode/memdir/memoryTypes.ts)
2. [ClueCode SessionMemory](https://github.com/anthropics/claude-code/blob/main/cludecode/services/SessionMemory/sessionMemory.ts)
3. goreact 项目源码 — `/goreact/core/memory.go`, `/goreact/reactor/reactor.go`, `/goreact/agent.go`
4. gorag 项目源码 — `/gorag/hybrid.go`, `/gorag/svc.go`
