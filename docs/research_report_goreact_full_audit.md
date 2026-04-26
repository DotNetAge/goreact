# goreact 全项目系统性代码质量审计报告

**审计日期**: 2026-04-26  
**项目路径**: `/Users/ray/workspaces/ai-ecosystem/goreact`  
**审计范围**: core/, reactor/, tools/, 根目录 全部 .go 源文件（~60+ 文件）  
**综合评分**: 7/10（良好）

---

## 一、项目架构总览

```
goreact/
├── agent.go              # 门面层 (Facade) — 唯一对外入口
├── errors.go             # 错误定义（别名层）
├── model_registry.go     # 模型注册
├── agent_registry.go     # Agent 注册
│
├── core/                 # 核心抽象层 — 类型、接口、基础设施
│   ├── session.go        # SessionStore 接口 + SlideEvent
│   ├── ctx_win.go        # ContextWindow + Sliding Window
│   ├── memory.go         # Memory 接口 + Experience 类型
│   ├── tool.go           # ToolRegistryInterface + FuncTool
│   ├── skill.go          # Skill 类型 + SkillRegistry 接口
│   ├── compact.go        # ContextCompactor + MicroCompact + TrimJSONResult (70%死代码)
│   ├── tiktoken.go       # Token 估算
│   └── session_store_memory.go  # MemorySessionStore 实现
│
├── reactor/               # 引擎层 — T-A-O 循环实现
│   ├── reactor.go         # Reactor 主结构体 (~1400行, SRP违反)
│   ├── prompts.go        # 提示词构建 + ActivateSkill
│   ├── context.go        # ReactContext
│   ├── thought/action/observation/step  # T-A-O 数据类型
│   ├── snapshot.go       # 快照机制
│   ├── intent_*.go / skill_registry.go / tool_registry.go
│   ├── accessor_impl.go  # 子代理/团队访问器桥接
│   ├── experience.go     # 经验存储
│   ├── eventbus.go       # 事件总线
│   └── reactor_options.go / skills_bundled.go
│
└── tools/                # 工具层 — 具体工具实现
    ├── bash/read/write/edit/grep/glob/ls/replace  # 文件操作
    ├── ask_user/ask_permission                   # 交互工具
    ├── subagent/task_tools/team_tools            # 编排工具
    ├── cron/memory/skill                         # 管理工具
    └── reactor_accessor.go                       # ReactorAccessor 接口(15方法, ISP违反)
```

**依赖方向**: `tools → reactor → core` （单向，无循环依赖 ✅）

---

## 二、问题清单总表

| 类别 | 🔴 高 | 🟡 中 | 🟢 低 | 合计 |
|------|-------|-------|-------|------|
| 死代码 | 6 | 4 | 2 | **12** |
| 重复代码 | 0 | 3 | 2 | **5** |
| 逻辑错误 | 3 | 4 | 2 | **9** |
| SOLID 违反 | 1 | 4 | 2 | **7** |
| 性能问题 | 0 | 2 | 3 | **5** |
| 规范问题 | 2 | 2 | 1 | **5** |
| **合计** | **12** | **19** | **12** | **43** |

---

## 三、P1 — 正确性 Bug（立即修复 ✅ 已完成）

### P1-1: LE-1 SlideEvent.Slided JSON tag 拼写错误

- **位置**: [core/session.go:12](../core/session.go#L12)
- **问题**: `SlideEvent.Slided` 的 JSON tag 写为 `` `json:"slied"` ``，少一个 'd'
- **影响**: 所有 JSON 序列化场景下 `Slided` 字段数据静默丢失。RAG/Memory 消费者无法获取被滑出的消息。
- **修复**: `` `json:"slied"` `` → `` `json:"slided"` ``

### P1-2: LE-2 MemorySessionStore.CurrentContext() 读-写竞态

- **位置**: [core/session_store_memory.go:43-63](../core/session_store_memory.go#L43-L63)
- **问题**: 先 RLock→RUnlock，然后在未持锁状态下遍历 msgs 切片。如果此时 Append 触发底层数组扩容，遍历的是过期数据。
- **修复**: 将整个函数体放在 RLock/RUnlock 保护范围内。

### P1-3: ST-1 MemoryTypeRefactive 拼写错误

- **位置**: [core/memory.go:24](../core/memory.go#L24)
- **问题**: `MemoryTypeRefactive` 应为 `MemoryTypeReflexive`
- **修复**: 重命名常量并同步更新所有引用者

---

## 四、P2 — 死代码清理（计划中）

### DC-2~DC-6: compact.go 大规模废弃代码

| 编号 | 代码 | 行数 | 说明 |
|------|------|------|------|
| DC-2 | `ContextCompactor` 接口 + `Compact()` 方法 | ~10 行 | 零调用方，已被 Slide 替代 |
| DC-3 | `ReNewer` 接口 + `ReNew()` 方法 | ~15 行 | 零调用方，从未集成到 Reactor |
| DC-4 | `CompactorConfig` 结构体 + `DefaultCompactorConfig()` | ~15 行 | 配合 DC-2 使用 |
| DC-5 | `SummaryMessage()` 函数 | ~15 行 | Compaction 边界消息生成器，无使用者 |
| DC-6 | `Prune()` 方法 | ~30 行 | 被 Slide() 替代，功能重叠且不通知 RAG |

**建议**: 将存活的 `MicroCompact`、`TrimJSONResult`、`TokenEstimator`、`DefaultTokenEstimator` 迁移到 `ctx_win.go` 或新文件 `utils.go`，然后删除 `compact.go`。净减约 180 行。

### DC-7~DC-8: 其他零调用方

| 编号 | 位置 | 代码 |
|------|------|------|
| DC-7 | ctx_win.go:150 | `TruncateResultSize()` |
| DC-8 | compact.go:279 | `IsJSONString()` |

---

## 五、P3 — 架构优化（计划中）

### SRP-1: 拆分 reactor.go (1400+ 行)

建议拆分为：
- `reactor_core.go`: struct 定义 + NewReactor + Run/runTAOLoop (~400行)
- `llm_calls.go`: callLLMStream/callLLMWithHistory/buildLLMBuilder (~200行)
- `think_act_observe.go`: Think/Act/Observe (~250行)
- `termination.go`: CheckTermination + 所有 helper 函数 (~200行)
- `session_manager.go`: ensureContextWindow/persistMessage/checkSlide (~100行)

### ISP-1: 拆分 ReActor 接口

当前 7 个方法全部暴露给外部消费者，但通常只需要 `Run`:
```go
// Public API
type ReActor interface {
    Run(ctx context.Context, input string, options ...RunOption) (*RunResult, error)
    RunFromSnapshot(ctx context.Context, snapshot *RunSnapshot) (*RunResult, error)
}

// Testing/internal use only
type ReActorInternal interface {
    ReActor
    Think(ctx *ReactContext) (*Thought, error)
    Act(ctx *ReactContext) (*Action, error)
    Observe(ctx *ReactContext, action *Action) *Observation
    CheckTermination(ctx *ReactContext) bool
}
```

### ISP-2: 拆分 ReactorAccessor 接口 (15+ 方法)

建议拆分为：
- `SubAgentAccessor`: RunSubAgent, GetPendingTask, RemovePendingTask
- `TeamAccessor`: TeamCreate, TeamJoin, TeamLeave, ListTeams, ...
- `SkillAccessor`: ListSkills, GetSkillInstructions, ...

### DIP-1/2: 注入 ReactorFactory

当前 `accessor_impl.go` 和 `agent.go` 内部直接 `NewReactor()` 创建子实例，不可 mock。建议通过 Option 注入工厂函数。

---

## 六、P4 — 性能与防御优化（计划中）

| # | 问题 | 优化方案 |
|---|------|----------|
| PF-2 | Slide() 每次 O(n²) token 计算 | 缓存 totalTokens 为字段，增量更新 |
| LE-5 | CurrentContext O(n²) prepend | 改用 append+reverse 模式 |
| LE-6 | Bash 工具命令注入风险 | 添加危险模式检测黑名单 |
| Dup-1 | calculateTotalTokens 重复定义 | 合并为单一方法 |
| Dup-3 | tool_result_storage 错误 fallback 重复 | 提取 makeFallbackResult() |
| Dup-4 | bash stdout/stderr 截断重复 | 提取 truncateOutput() |

---

## 七、修复进度跟踪

| Phase | 内容 | 状态 | 完成日期 |
|-------|------|------|----------|
| P0 | reactor.go 局部审计修复 (13项) | ✅ 完成 | 2026-04-26 |
| P1 | 全项目正确性 Bug 修复 (LE-1/LE-2/ST-1) | ✅ 完成 | 2026-04-26 |
| P2 | 死代码清理 (compact.go 等 ~246行) | ✅ 完成 | 2026-04-26 |
| P3 | 架构优化 (拆分4文件+接口, reactor.go 1435→582) | ✅ 完成 | 2026-04-26 |
| **P4** | **性能/防御优化 (Slide O(n²)→O(n), Bash危险检测, 3处重复合并)** | **✅ 完成** | **2026-04-26** |
