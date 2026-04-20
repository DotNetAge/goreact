# GoReAct Memory 开发指南

> Memory 是 ReActor 最重要的扩展点。一个强大的 Memory 实现可以让 ReActor 从"可用"变为"质变"——消除幻觉、语义化工具匹配、上下文永不腐烂、经验自动复用。

## 一、Memory 在 ReActor 中的角色

ReActor 将 Memory 视为 Agent 的"装备"：

| 组件 | 类比 | 职责 |
|---|---|---|
| SystemPrompt | 职能定义 | 告诉 Agent 它是谁、能做什么 |
| Model | 先天能力 | LLM 的推理、理解、生成能力 |
| **Memory** | **后天记忆** | **知识检索、经验复用、上下文管理** |

**关键原则**：Memory 是可选的。没有 Memory，ReActor 仍然完整可用（只是"记忆力差"）。注入 Memory 后，ReActor 的能力由 Memory 的强弱决定。

## 二、接口定义

### 2.1 核心接口：Memory

```go
package core

type Memory interface {
    // Retrieve 语义搜索：根据 query 检索相关记忆记录
    // 返回按相关性排序的记录列表（Score 从高到低）
    Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]MemoryRecord, error)

    // Store 存入一条记忆记录，返回生成的 ID
    Store(ctx context.Context, record MemoryRecord) (string, error)

    // Update 更新已有记录（ID 必须存在，否则返回 ErrMemoryNotFound）
    Update(ctx context.Context, id string, record MemoryRecord) error

    // Delete 删除记录
    Delete(ctx context.Context, id string) error
}
```

### 2.2 可选接口：ReNewer

实现 `ReNewer` 接口的 Memory 可以在上下文压缩时提供**语义化重建**能力，替代传统的 LLM 摘要压缩。

```go
type ReNewer interface {
    // ReNew 根据 sessionID 和当前意图，从临时记忆中语义召回相关上下文
    // 返回精炼后的消息列表，ReActor 用它替换 ConversationHistory
    //
    // 参数：
    //   - sessionID: 当前会话标识，用于隔离不同会话的临时记忆
    //   - intent: 当前用户意图的文本描述（如 "部署Go项目"）
    //   - messages: 当前会话的全部消息历史
    //
    // 返回：
    //   - 精炼后的消息列表（长度应小于输入的 messages）
    ReNew(ctx context.Context, sessionID string, intent string, messages []Message) ([]Message, error)
}
```

**检测方式**：ReActor 通过类型断言检测 Memory 是否实现了 ReNewer：

```go
if renewer, ok := memory.(core.ReNewer); ok {
    // 使用 ReNew 进行语义化上下文重建
    messages, err := renewer.ReNew(ctx, sessionID, intent, allMessages)
}
```

如果 Memory 不实现 ReNewer，ReActor 会 fallback 到传统的 LLM 摘要压缩或截断策略。

### 2.3 数据结构

```go
type MemoryType int
const (
    MemoryTypeSession    MemoryType = iota // 临时记忆：会话上下文
    MemoryTypeUser                         // 短时记忆：用户偏好
    MemoryTypeLongTerm                     // 长期记忆：知识库
    MemoryTypeRefactive                    // 反射性记忆：工具语义索引
    MemoryTypeExperience                   // 经验记忆：成功执行路径
)

type MemoryScope int
const (
    MemoryScopePrivate MemoryScope = iota // 私有：仅当前用户可见
    MemoryScopeTeam                      // 团队：跨 Agent 共享
)

type MemoryRecord struct {
    ID        string       `json:"id"`         // 记录唯一标识
    Type      MemoryType   `json:"type"`       // 记忆类型
    Title     string       `json:"title"`      // 标题/问题描述
    Content   string       `json:"content"`    // 内容/解决办法
    Scope     MemoryScope  `json:"scope"`      // 可见性
    Tags      []string     `json:"tags"`       // 标签（用于辅助检索）
    Score     float64      `json:"score"`      // 相关性分数（Retrieve 时由实现填充）
    Meta      any          `json:"meta"`       // 类型化元数据（如 Type=Experience 时为 *ExperienceData）
    CreatedAt time.Time    `json:"created_at"`
    UpdatedAt time.Time    `json:"updated_at"`
}
```

其中 `Meta` 字段是 ReActor 写入经验记忆时自动填充的类型化指针。对于 `MemoryTypeExperience` 类型的记录，`Meta` 是一个 `*core.ExperienceData`，Memory 实现可以直接通过类型断言获取，而无需解析 Content 中的 JSON。

### 2.4 Retrieve 选项

```go
// 按类型过滤
memory.Retrieve(ctx, query, core.WithMemoryTypes(core.MemoryTypeLongTerm, core.MemoryTypeUser))

// 按作用域过滤
memory.Retrieve(ctx, query, core.WithMemoryScope(core.MemoryScopeTeam))

// 限制返回数量
memory.Retrieve(ctx, query, core.WithMemoryLimit(10))

// 设置最低相关性分数
memory.Retrieve(ctx, query, core.WithMinScore(0.5))

// 组合使用
memory.Retrieve(ctx, query,
    core.WithMemoryTypes(core.MemoryTypeLongTerm),
    core.WithMemoryLimit(3),
    core.WithMinScore(0.3),
)
```

## 三、五种记忆类型详解

### 3.1 临时记忆（MemoryTypeSession）

**用途**：存储会话上下文，支持跨会话恢复。

**ReActor 如何使用**：
- 当 Memory 实现了 `ReNewer` 接口时，上下文压缩（Compact）会调用 `ReNew` 方法
- ReNew 接收 sessionID，Memory 实现按 sessionID 隔离数据

**实现建议**：
- 每个 sessionID 对应一组有序的消息记录
- ReNew 时：以当前意图为查询条件，从全部临时记忆中语义召回相关上下文 + 保留近期 N 轮对话
- 可以将重要的临时记忆内容迁移到长期记忆中

**数据格式示例**：
```go
MemoryRecord{
    Type:    core.MemoryTypeSession,
    Title:   "",  // 临时记忆通常不需要标题
    Content: "用户问：如何连接MySQL？\n助手答：使用 mysql.Client{} 并配置 DSN...",
    Scope:   core.MemoryScopePrivate,
}
```

### 3.2 短时记忆（MemoryTypeUser）

**用途**：记录用户偏好、操作习惯。Agent 可以主动写入，也可以由 LLM 通过 `memory_save` 工具写入。

**ReActor 如何使用**：
- Think 阶段自动检索：`Retrieve(input, WithMemoryTypes(MemoryTypeUser))`
- 检索到的用户偏好注入到 Think prompt 中

**实现建议**：
- 单条记录代表一个偏好项（如 "用户偏好中文回复"、"项目使用 Go 1.21"）
- 支持快速更新（用户偏好可能随时变化）
- 可以从对话中自动提取：同一会话中用户多次提到的内容自动标记为偏好

**数据格式示例**：
```go
MemoryRecord{
    Type:    core.MemoryTypeUser,
    Title:   "用户偏好中文回复",
    Content: "用户在所有对话中倾向于使用简体中文回复，代码注释也使用中文。",
    Scope:   core.MemoryScopePrivate,
    Tags:    []string{"preference", "language", "zh-CN"},
}
```

### 3.3 长期记忆（MemoryTypeLongTerm）

**用途**：通用知识库。包括项目架构决策、领域知识、代码规范、文档等一切"应该被记住的事实"。

**ReActor 如何使用**：
- Think 阶段自动检索：`Retrieve(input, WithMemoryTypes(MemoryTypeLongTerm))`
- LLM 可通过 `memory_save` 工具主动存入重要发现
- 团队级别的知识使用 `MemoryScopeTeam`，跨 Agent 共享

**实现建议**：
- 这是 Memory 实现最能发挥价值的地方——一个基于 RAG 的语义搜索实现可以极大提升 Agent 的知识获取能力
- 可以接入向量数据库（如 gorag、Chroma、Pinecone、Milvus）
- 支持文档级别的索引：将项目文件、文档、Wiki 等批量导入
- Score 反映语义相似度（如余弦相似度）

**数据格式示例**：
```go
MemoryRecord{
    Type:    core.MemoryTypeLongTerm,
    Title:   "项目使用 Clean Architecture 分层",
    Content: "项目遵循 Clean Architecture 原则：domain 层不依赖外部包，handler 层处理 HTTP...",
    Scope:   core.MemoryScopeTeam,
    Tags:    []string{"architecture", "clean-architecture", "project"},
}
```

### 3.4 反射性记忆（MemoryTypeRefactive）

**用途**：工具、技能、代理的语义索引。让 ReActor 通过意图语义找到最匹配的工具，而不是依赖精确名称匹配。

**ReActor 如何使用**：
- 当 LLM 调用一个不存在的工具名时，`ToolRegistry.GetWithSemantic()` 会 fallback 到语义搜索
- 搜索流程：`精确匹配 map[string]` → 失败 → `Memory.Retrieve(intent, WithMemoryTypes(MemoryTypeRefactive))` → 得到工具名列表 → 再精确查找

**实现建议**：
- MemoryRecord 的 `Title` 字段存储**工具/技能的名称**（必须与注册表中的名称一致）
- `Content` 字段存储工具的语义描述（用于语义匹配）
- 索引数据由客户端在启动时写入：遍历注册表（`ToolRegistry.All()`、`SkillRegistry.ListSkills()`），将每个工具的 Name 和 Description 写入 Memory

**数据格式示例**：
```go
MemoryRecord{
    Type:    core.MemoryTypeRefactive,
    Title:   "web_search",     // 必须与 ToolRegistry 中的名称一致
    Content: "在互联网上搜索信息，返回相关网页内容。适用于需要获取实时信息、查找文档、了解最新动态等场景。",
    Scope:   core.MemoryScopePrivate,
    Tags:    []string{"search", "internet", "web", "information"},
}
```

**写入反射性索引的示例代码**：

```go
func IndexToolsToMemory(memory core.Memory, registry *reactor.ToolRegistry) {
    for _, tool := range registry.All() {
        info := tool.Info()
        memory.Store(context.Background(), core.MemoryRecord{
            Type:    core.MemoryTypeRefactive,
            Title:   info.Name,
            Content: info.Description,
            Scope:   core.MemoryScopePrivate,
            Tags:    extractTags(info.Description),
        })
    }
}
```

### 3.5 经验记忆（MemoryTypeExperience）

**用途**：存储成功解决问题的分析结果。下次遇到类似问题时，直接参考历史分析，避免重复消耗 Token。

**ReActor 如何使用**：
- 任务成功完成后，ReActor **自动**调用 `Memory.Store()` 写入经验（`reactor/experience.go`）
- Think 阶段自动检索：`Retrieve(input, WithMemoryTypes(MemoryTypeExperience))`
- 检索到的经验作为"参考分析"注入 prompt

**经验的结构**：经验由两部分组成——"问题"和"解决办法"。

| 部分 | MemoryRecord 字段 | 作用 | 说明 |
|---|---|---|---|
| 问题 | Title + Tags | 语义索引 | 用当前问题匹配历史问题 |
| 解决办法 | Content + Meta | 参考内容 | LLM 分析推理 + 工具 + SubAgent + 步骤 |

**ReActor 写入的完整结构**（`core.ExperienceData`）：

ReActor 会将一个 `*core.ExperienceData` 同时写入 `Content`（JSON 序列化）和 `Meta`（类型化指针）。你的 Memory 实现可以通过类型断言直接获取结构体，无需反序列化：

```go
// 方式 1：通过 Meta 直接获取（推荐）
if exp, ok := record.Meta.(*core.ExperienceData); ok {
    // exp.Analysis — LLM 的推理过程（最值钱的部分）
    // exp.Tools    — 调用过的工具列表
    // exp.SubAgents — spawn 过的子任务
    // exp.Steps    — 每轮 T-A-O 的紧凑摘要
    // exp.Answer   — 最终答案
    // exp.TokenCost — Token 消耗量
}

// 方式 2：从 Content 反序列化（兼容性更好）
var exp core.ExperienceData
json.Unmarshal([]byte(record.Content), &exp)
```

`ExperienceData` 的关键字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| Problem | string | 原始用户输入 |
| Analysis | string | 所有 Think 阶段的 Reasoning 合并（最值钱，可直接参考） |
| Tools | []string | 去重的工具调用列表（如 `["bash", "read_file"]`） |
| SubAgents | []ExperienceSubAgent | spawn 过的子任务（task_create / subagent），含名称、prompt、成功与否 |
| Steps | []ExperienceStep | 每轮 T-A-O 的紧凑摘要 |
| Answer | string | 最终答案 |
| TokenCost | int | Token 消耗量（可用于排序优先级） |

**实现建议**：
- Title 和 Tags 的语义质量决定了召回效果——ReActor 会自动从 Intent + 用户输入提取 Tags
- SubAgents 信息对于生成 Skill 非常关键——它记录了任务的编排模式
- Analysis 是最值钱的部分——它是 LLM 在 Think 阶段消耗大量 Token 产出的推理过程

## 四、ReActor 的 Memory 调用时机

了解 ReActor 何时调用 Memory，有助于你优化实现：

```
ReActor.Run(userInput)
│
├─ Think 阶段
│   └─ Memory.Retrieve(input, Types=[LongTerm, User, Experience], Limit=3)
│     → 检索结果注入 Think prompt 的 <relevant_memory> 区域
│
├─ Act 阶段
│   └─ ToolRegistry.GetWithSemantic(toolName, intent)
│     → 精确匹配失败时 Memory.Retrieve(intent, Types=[Refactive])
│
├─ Observe 阶段（工具执行后）
│   └─ LLM 可能主动调用 memory_save / memory_search 工具
│
└─ Compact 阶段（上下文超限时）
    └─ Memory.(ReNewer).ReNew(sessionID, intent, messages)
      → 成功时替换 ConversationHistory
      → 失败时 fallback 到传统 LLM 摘要压缩
```

## 五、实现一个 Memory

### 5.1 最小实现（仅 CRUD，无语义搜索）

```go
package mymemory

import (
    "context"
    "database/sql"
    "github.com/DotNetAge/goreact/core"
)

// SQLiteMemory 基于 SQLite 的简单实现
type SQLiteMemory struct {
    db *sql.DB
}

func NewSQLiteMemory(dsn string) (*SQLiteMemory, error) {
    db, err := sql.Open("sqlite3", dsn)
    if err != nil {
        return nil, err
    }
    // 建表
    db.Exec(`CREATE TABLE IF NOT EXISTS memories (
        id TEXT PRIMARY KEY,
        type INTEGER NOT NULL,
        title TEXT DEFAULT '',
        content TEXT NOT NULL,
        scope INTEGER DEFAULT 0,
        tags TEXT DEFAULT '[]',
        score REAL DEFAULT 0,
        created_at DATETIME,
        updated_at DATETIME
    )`)
    return &SQLiteMemory{db: db}, nil
}

func (m *SQLiteMemory) Retrieve(ctx context.Context, query string, opts ...core.RetrieveOption) ([]core.MemoryRecord, error) {
    cfg := core.DefaultRetrieveConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    // 简单的 LIKE 搜索（非语义化，但足够起步）
    rows, err := m.db.QueryContext(ctx,
        "SELECT id, type, title, content, scope, tags, created_at, updated_at FROM memories WHERE content LIKE ? ORDER BY updated_at DESC LIMIT ?",
        "%"+query+"%", cfg.Limit,
    )
    // ... 遍历 rows 填充 []core.MemoryRecord ...
}

func (m *SQLiteMemory) Store(ctx context.Context, record core.MemoryRecord) (string, error) {
    // INSERT INTO memories ...
}

func (m *SQLiteMemory) Update(ctx context.Context, id string, record core.MemoryRecord) error {
    // UPDATE memories SET ... WHERE id = ?
}

func (m *SQLiteMemory) Delete(ctx context.Context, id string) error {
    // DELETE FROM memories WHERE id = ?
}
```

### 5.2 RAG 实现（语义搜索）

```go
package ragmemory

import (
    "context"
    "github.com/DotNetAge/goreact/core"
    // 假设使用 gorag 作为 RAG 引擎
)

// RAGMemory 基于 gorag 的语义搜索实现
type RAGMemory struct {
    // gorag 的索引和查询接口
    indexer  gorag.Indexer
    searcher gorag.Searcher
    // 元数据存储（SQLite/PostgreSQL）
    metaDB *sql.DB
}

func (m *RAGMemory) Retrieve(ctx context.Context, query string, opts ...core.RetrieveOption) ([]core.MemoryRecord, error) {
    cfg := core.DefaultRetrieveConfig()
    for _, opt := range opts {
        opt(&cfg)
    }

    // 1. 使用 gorag 进行向量搜索
    results, err := m.searcher.Search(ctx, query, gorag.WithLimit(cfg.Limit), gorag.WithMinScore(cfg.MinScore))
    if err != nil {
        return nil, err
    }

    // 2. 过滤类型和作用域
    var records []core.MemoryRecord
    for _, r := range results {
        record := m.loadRecord(ctx, r.ID) // 从 metaDB 加载完整记录
        if record == nil {
            continue
        }
        // 应用类型过滤
        if len(cfg.Types) > 0 && !matchType(record.Type, cfg.Types) {
            continue
        }
        // 应用作用域过滤
        if cfg.Scope != 0 && record.Scope != cfg.Scope {
            continue
        }
        record.Score = r.Score
        records = append(records, *record)
    }
    return records, nil
}

func (m *RAGMemory) Store(ctx context.Context, record core.MemoryRecord) (string, error) {
    // 1. 生成向量嵌入并索引到 gorag
    vector, err := m.indexer.Embed(ctx, record.Title+" "+record.Content)
    // 2. 存储元数据到 metaDB
    // 3. 返回 ID
}

// 同时实现 ReNewer 接口
func (m *RAGMemory) ReNew(ctx context.Context, sessionID string, intent string, messages []core.Message) ([]core.Message, error) {
    // 1. 将 intent 作为查询条件，从 MemoryTypeSession 中语义召回
    relevant, _ := m.Retrieve(ctx, intent,
        core.WithMemoryTypes(core.MemoryTypeSession),
        core.WithMemoryLimit(10),
    )
    // 2. 保留最近 5 轮对话
    recentCount := 10 // 5 轮 = 10 条消息
    if len(messages) < recentCount {
        recentCount = len(messages)
    }
    recent := messages[len(messages)-recentCount:]

    // 3. 组合：语义召回的上下文 + 最近对话
    var rebuilt []core.Message
    for _, rec := range relevant {
        rebuilt = append(rebuilt, core.Message{
            Role:      "system",
            Content:   "[Memory Recalled] " + rec.Content,
            Timestamp: rec.CreatedAt.Unix(),
        })
    }
    rebuilt = append(rebuilt, recent...)
    return rebuilt, nil
}
```

### 5.3 完整的 ReNew 实现

ReNew 是 Memory 实现最能体现"语义化能力"的地方。一个好的 ReNew 实现应该做到：

1. **不丢关键信息**：通过语义搜索确保与当前意图最相关的历史上下文被保留
2. **控制长度**：返回的消息总长度应显著小于输入
3. **保持连贯**：最近几轮对话必须完整保留，避免破坏对话流畅性

```go
func (m *RAGMemory) ReNew(ctx context.Context, sessionID string, intent string, messages []core.Message) ([]core.Message, error) {
    const preserveRecentTurns = 5 // 至少保留 5 轮（10 条消息）

    // 1. 从全部临时记忆中语义召回与意图相关的内容
    relevantContext, err := m.Retrieve(ctx, intent,
        core.WithMemoryTypes(core.MemoryTypeSession),
        core.WithMemoryLimit(8),
        core.WithMinScore(0.3),
    )
    if err != nil {
        return nil, err
    }

    // 2. 保留最近 N 轮完整对话
    preserveCount := preserveRecentTurns * 2
    if len(messages) < preserveCount {
        preserveCount = len(messages)
    }
    recentMessages := messages[len(messages)-preserveCount:]

    // 3. 从语义召回结果中提取不在最近对话中的内容（去重）
    recentContent := extractContentSet(recentMessages)
    var additional []core.Message
    for _, rec := range relevantContext {
        if !recentContent.Contains(rec.Content) {
            additional = append(additional, core.Message{
                Role:      "system",
                Content:   fmt.Sprintf("[Recalled context] %s", rec.Content),
                Timestamp: rec.CreatedAt.Unix(),
            })
        }
    }

    // 4. 组合结果
    result := make([]core.Message, 0, len(additional)+len(recentMessages))
    result = append(result, additional...)
    result = append(result, recentMessages...)

    return result, nil
}
```

## 六、注入 Memory 到 ReActor

```go
package main

import (
    "github.com/DotNetAge/goreact"
    "github.com/DotNetAge/goreact/reactor"
    "github.com/DotNetAge/goreact/core"
)

func main() {
    config := reactor.DefaultReactorConfig()
    config.APIKey = "your-api-key"
    config.Model = "gpt-4o"

    // 创建你的 Memory 实现
    memory := NewRAGMemory("path/to/index")

    // 通过 WithMemory 注入
    r := reactor.NewReactor(config,
        reactor.WithMemory(memory),
    )

    agent := goreact.NewAgent(
        &core.AgentConfig{Name: "my-agent"},
        &core.ModelConfig{Model: "gpt-4o"},
        memory,  // Agent 层也持有 Memory
        r,
    )

    // 使用...
    result, err := agent.Chat("如何部署这个项目？")
}
```

## 七、反射性索引的写入

ReActor 不会自动将工具信息写入 Memory——这是客户端的职责。你需要在 Agent 启动后手动完成索引：

```go
// 在 Agent 启动后，将工具信息写入 Memory 的反射性索引
func BuildRefactiveIndex(memory core.Memory, r *reactor.Reactor) {
    ctx := context.Background()

    // 索引工具
    for _, tool := range r.ToolRegistry().All() {
        info := tool.Info()
        memory.Store(ctx, core.MemoryRecord{
            Type:    core.MemoryTypeRefactive,
            Title:   info.Name,       // 必须与注册表名称一致
            Content: info.Description, // 语义描述，用于匹配
            Tags:    extractKeywords(info.Description),
        })
    }

    // 索引技能
    if sr, ok := r.SkillRegistry().(*reactor.SkillRegistry); ok {
        for _, skill := range sr.ListSkills() {
            memory.Store(ctx, core.MemoryRecord{
                Type:    core.MemoryTypeRefactive,
                Title:   skill.Name,
                Content: skill.Description,
                Tags:    extractKeywords(skill.Description),
            })
        }
    }
}
```

## 八、进阶：自成长系统

Memory 实现可以在存储经验记忆时，自动将经验生成为 Skill 文件，实现 Agent 的自成长：

```go
func (m *RAGMemory) Store(ctx context.Context, record core.MemoryRecord) (string, error) {
    id, err := m.doStore(ctx, record)
    if err != nil {
        return "", err
    }

    // 当存储经验记忆时，自动生成 Skill
    if record.Type == core.MemoryTypeExperience {
        go m.generateSkillFromExperience(record)
    }

    return id, nil
}

func (m *RAGMemory) generateSkillFromExperience(record core.MemoryRecord) {
    // 通过 Meta 直接获取结构化数据（无需反序列化 Content）
    exp, ok := record.Meta.(*core.ExperienceData)
    if !ok {
        return
    }

    // 构建工具列表和编排模式
    var toolList string
    if len(exp.Tools) > 0 {
        toolList = "allowed-tools: " + strings.Join(exp.Tools, " ")
    }
    if len(exp.SubAgents) > 0 {
        toolList += " subagent subagent-result"
    }

    // 将经验转化为 SKILL.md
    skillContent := fmt.Sprintf(`---
name: %s
description: Auto-generated from experience. %s
%s
---

## 触发条件

当用户遇到以下类型的问题时，使用此经验：

%s

## 分析参考

> 以下是 LLM 在解决类似问题时产生的分析推理，可直接参考：

%s

## 解决步骤

%s

## 编排模式（如有子任务）

%s
`,
        slugify(record.Title),
        record.Title,
        toolList,
        strings.Join(record.Tags, ", "),
        exp.Analysis,
        formatSteps(exp.Steps),
        formatSubAgents(exp.SubAgents),
    )

    // 写入用户的 Skill 目录
    skillDir := filepath.Join("skills", slugify(record.Title))
    os.MkdirAll(skillDir, 0755)
    os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644)
}

func formatSteps(steps []core.ExperienceStep) string {
    var sb strings.Builder
    for i, s := range steps {
        fmt.Fprintf(&sb, "%d. ", i+1)
        if s.Action != "" {
            fmt.Fprintf(&sb, "[Action: %s] ", s.Action)
        }
        if s.Thought != "" {
            sb.WriteString(s.Thought)
        }
        if s.HasError {
            sb.WriteString(" (had error, retried)")
        }
        sb.WriteString("\n")
    }
    return sb.String()
}

func formatSubAgents(agents []core.ExperienceSubAgent) string {
    if len(agents) == 0 {
        return "(无子任务编排)"
    }
    var sb strings.Builder
    for _, a := range agents {
        status := "成功"
        if !a.Success {
            status = "失败"
        }
        fmt.Fprintf(&sb, "- [%s] %s (via %s)\n", status, a.Name, a.Tool)
    }
    return sb.String()
}
```

这样，ReActor 每解决一个新问题，就多一个新 Skill。Agent 越使用越强大。

## 九、实现清单

根据你的需求选择实现等级：

| 等级 | 实现内容 | 效果 |
|---|---|---|
| L0 | 仅 `Memory` 接口（CRUD） | Agent 可以主动存储和检索知识 |
| L1 | + `ReNewer` 接口 | 上下文语义化重建，解决"上下文腐烂" |
| L2 | + 反射性索引写入 | 语义化工具匹配，不依赖精确名称 |
| L3 | + 经验记忆自动存储 | Token 消耗递减，"越用越省" |
| L4 | + 经验→Skill 自动生成 | 自成长系统，Agent 越用越强 |

## 十、注意事项

1. **线程安全**：Memory 的所有方法都可能被并发调用（Think 阶段、工具执行等），实现必须保证线程安全
2. **非阻塞**：ReNew 在 T-A-O 循环的热路径上调用，实现应控制延迟。如果语义搜索耗时较长，考虑异步预加载或缓存
3. **容错**：Memory 调用失败不应导致 ReActor 崩溃。ReActor 在所有 Memory 调用处都有 fallback 逻辑
4. **Score 归一化**：不同实现的 Score 含义可能不同（余弦相似度 0-1 vs BM25 分数）。ReActor 只用 Score 做排序和 `MinScore` 过滤，不依赖具体数值范围
5. **ID 格式**：ID 由 Memory 实现生成。可以使用 UUID、自增序列、或内容哈希，但必须保证全局唯一
