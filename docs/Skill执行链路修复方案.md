# 三阶段渐进式披露 T-A-O 重构 — 完整归档

> **重构周期**: 2025-04-25
> **核心目标**: 修复 Skill 执行链路断裂 + 实现 ContextWindow 滑动窗口机制
> **涉及维度**: 架构设计 / Prompt 工程 / Token 管理 / 模板系统 / 接口演进

---

## 一、问题全景（修复前状态）

### 1.1 五大断裂点

| # | 问题 | 严重度 | 根因 |
|---|------|--------|------|
| **S1** | Skill L2 Instructions 被截断 | 🔴 致命 | `BuildSkillsSystemPrompt()` 只发 L1 摘要，完整 SOP 从未到达 LLM |
| **S2** | 无 Skill 选择机制 | 🔴 致命 | Think 输出只有 `action_target`(Tool)，无 `skill_name` 字段 |
| **S3** | Tool 未按 Skill 过滤 | 🟡 中等 | 全部工具通过 `builder.Tools()` 发给 LLM，无视 `allowed-tools` |
| **S4** | Act 缺少 Skill 上下文 | 🔴 致命 | Act 直接执行 Tool，无 SOP 引导 |
| **C1** | Skills 写入 User Prompt | 🟠 中等 | 应在 System Prompt 层（高优先级、不占 user context） |
| **C2** | Tools 写入 Prompt 文本 | 🟠 中等 | 应通过 LLM 原生 function calling (`builder.Tools()`) |
| **C3** | Tool Input Tokens 未计入 CW | 🟡 中等 | Tool JSON Schema 的 token 开销被忽略 |
| **C4** | UserPrompt Input Tokens 未计入 CW | 🟡 中等 | `persistMessage()` 只存消息不计算 |
| **W1** | 滑动触发位置错误 | 🔴 致命 | 在 LLM 返回后检查滑动（太晚），应在调用前预检 |
| **W2** | T-A-O 中间步骤未持久化 | 🟡 中等 | `runTAOLoop` 的 stepSummary 用 `AddMessage` 而非 `persistMessage` |

### 1.2 修复前数据流（完全断裂）

```
Think():
  FindApplicableSkills(intent) → []*Skill {Name, Desc, Instructions✅, AllowedTools}
       ↓
  BuildSkillsSystemPrompt(skills) → 只有 L1 ❌ (S1)
       ↓
  ToolInfosToLLMTools(allTools) → 全部工具未过滤 🟡 (S3)
       ↓
  callLLMStream(..., llmTools, skillsSection)
    Layer 1b: SystemMessage(L1 only)
    Layer 2:  UserMessage(含 tools 文本 ❌ C2, 含 skills ❌ C1)
    Layer 4:  Tools(全部未过滤 🟡 S3)

  LLM 返回: { decision:"act", action_target:"bash" }
       ↓ (无 skill 选择 S2)
Act():
  ExecuteTool("bash", params)
  ⚠️ 无 Skill SOP 引导 (S4)

  checkSlide() ← 在 Run() 返回后才触发 (W1 太晚!)
```

---

## 二、架构总览（修复后）

### 2.1 分层架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Agent (门面层)                             │
│   纯代理：持有 Reactor + Config/Model + 提供 Ask() 入口     │
│   不再直接操作 ContextWindow                                │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                    Reactor (引擎层)                           │
│                                                             │
│  ┌──────────┐  ┌─────────────┐  ┌─────────────────────┐    │
│  │ContextWin │  │SessionStore │  │Memory/RAG           │    │
│  │(短期记忆) │  │(WAL 持久化) │  │(长期知识/语义检索)   │    │
│  └────┬─────┘  └──────┬──────┘  └──────────┬──────────┘    │
│       │               │                     │                │
│  ┌────▼───────────────▼─────────────────────▼──────────┐   │
│  │              Think() — 两阶段推理引擎                   │   │
│  │  Phase1: 选 Skill → ActivateSkill → Phase2: 选 Tool   │   │
│  └────────────────────────┬────────────────────────────┘   │
│                           │                                 │
│  ┌────────────────────────▼──────────────────────────┐   │
│  │              Act() → Observe()                      │   │
│  │  在 Skill SOP 引导下执行工具                         │   │
│  └───────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────┘

LLM 调用四层模型:
  Layer 1a: SystemMessage(Agent 身份)          — 固定开销
  Layer 1b: SystemMessage(Skills P1 列表)      — ~100/skill (追加式 ✅)
  Layer 2:  UserMessage(Phase 指令 + 用户输入)  — 动态内容
  Layer 3:  Messages(History, token-trimmed)   — 滑动窗口管理
  Layer 4:  Tools(原生 function calling)       — 过滤后的子集
```

### 2.2 三阶段渐进披露（P1/P2/P3）

| 阶段 | 名称 | 内容 | Token量 | 触发时机 | 注入位置 |
|------|------|------|----------|----------|----------|
| **P1** | 轻量模式 | Name + Description + AllowedTools (L1 Metadata) | ~100/skill | 始终（有 Skills 时） | **System Prompt Layer 1b** |
| **P2** | 重量模式 | SKILL.md 正文完整 SOP (L2 Instructions) + 过滤后 Tool 子集 + ResourceBasePath | <5000/skill | Phase 1 选中 Skill 后激活 | **UserMessage `<active_skill_instructions>`** |
| **P3** | 全量模式 | scripts/, references/, assets/ 等资源文件按需访问 | 0 预加载 | Action 阶段 LLM 通过 ResourceBasePath 构造路径 | **不注入 prompt，LLM 按需 read/bash** |

---

## 三、T-A-O 循环完整数据流（修复后）

### 3.1 单次迭代全链路

```
Run(input):
  │
  ├─① persistMessage("user", input)
  │     ├─ CW.AddMessage("user", input)
  │     ├─ SS.Append(sessionID, msg)
  │     └─ CW.AddTokens(estimate(input))        ← C4 ✅
  │
  ├─② SS.CurrentContext(sessionID, budget) → history (token-aware retrieval)
  │
  └─③ runTAOLoop(reactCtx):
        │
        ├──▶ Think(reactCtx): ════════════════════════════
        │     │
        │     ├─ 准备:
        │     │   toolInfos = ToToolInfos()
        │     │   llmTools = ToolInfosToLLMTools(toolInfos)
        │     │   skills = FindApplicableSkills(intent)
        │     │   skillsSection = BuildCapabilitiesList(skills)
        │     │   memoryRecords = Memory.Retrieve(...)
        │     │
        │     ├─ Token 记入固定开销:
        │     │   CW.AddTokens(skillsSection)             ← P1 ✅
        │     │   CW.AddTokens(EstimateTokensForTools())  ← C3 ✅
        │     │
        │     ├─╌ Phase 1: Skill Selection (轻量 LLM 调用)╨
        │     │  │ selectInstructions = BuildSkillSelectPrompt()
        │     │  │ CW.AddTokens(selectInstructions)
        │     │  │ preSlideCheck()                          ← W1 ✅ (调用前!)
        │     │  │ callLLMStream(..., nil, capabilitiesSection)
        │     │  │   → Layer 1b: SystemMessage(capabilities) ← C1 ✅
        │     │  │ → { selected_skill: "bug-hunter" }
        │     │  │
        │     │  └─ ActivateSkill("bug-hunter"):
        │     │       ├─ GetSkill("bug-hunter") → full Skill obj
        │     │       ├─ filterToolsByAllowed(allTools, allowed)
        │     │       └─ return ActivatedSkillContext{
        │     │            Instructions:  L2 SOP text,
        │     │            FilteredTools: [grep,glob,bash,...],
        │     │            ResourceBasePath: chosen.RootDir  ← P3 ✅
        │     │          }
        │     │
        │     ├─ Token 记入 P2/P3:
        │     │   CW.AddTokens(actCtx.Instructions)         ← P2 ✅
        │     │   CW.AddTokens(EstimateTokensForTools(filtered))
        │     │
        │     ├─╌ Phase 2: Skill-Guided Planning (重量)╨
        │     │  │ instructions = BuildThinkPrompt(+actCtx)
        │     │  │ CW.AddTokens(instructions)
        │     │  │ preSlideCheck()
        │     │  │ callLLMStream(..., filteredTools, skillsSection)
        │     │  │   → Layer 2: <active_skill_instructions>
        │     │  │     === SKILL: bug-hunter ===
        │     │  │     (L2 SOP 完整文本)                    ← P2 ✅
        │     │  │     Available tools: grep, glob, bash
        │     │  │     ResourceBasePath: /.../bug-hunter/    ← P3 ✅
        │     │  │   → Layer 4: Tools([grep, glob, bash])   ← S3 ✅
        │     │  │ → { decision:"act", action_target:"grep",
        │     │  │     action_params:{pattern:"ERROR"},
        │     │  │     reasoning:"按照 Step1 先 grep 定位..." }
        │     │  │
        │     └─ ctx.LastThought = thought (含 SelectedSkill)
        │
        ├──▶ Act(reactCtx): ═══════════════════════════════
        │     case DecisionAct:
        │       action.Target = "grep"
        │       action.Params = {"pattern": "ERROR"}
        │       r.toolRegistry.ExecuteTool("grep", params)
        │       → 在 bug-hunter Step1 引导下执行 ✅          ← S4 ✅
        │
        ├──▶ Observe(reactCtx): ═══════════════════════════
        │     Package result as Observation
        │
        └─④ stepSummary = "Thought: ... \nAction: grep(...) \nObservation: ..."
              reactCtx.AddMessage("assistant", stepSummary)   → 内存历史
              r.persistMessage("assistant", stepSummary)      ← W2 ✅ 持久化+计Token
  │
  ├─⑤ persistMessage("assistant", answer)
  │     ├─ CW.AddMessage("assistant", answer)
  │     ├─ SS.Append(sessionID, msg)
  │     └─ CW.AddTokens(estimate(answer))
  │
  └─⑥ checkSlide() → 最终滑动检查
```

### 3.2 多轮迭代中的滑动窗口行为

```
Iteration 1:  用户输入 "帮我找这个 bug"
  → CW: [user:帮我找这个bug] (tokens: 50)
  → Phase1: 选择 bug-hunter
  → Phase2: 决定 grep ERROR
  → Act: grep("ERROR") → 找到 10 处
  → persist: [assistant: Thought+Action+Observation] (tokens: 200)
  → CW total: ~250 tokens (远低于 65% 阈值)

Iteration 2-5: 继续调试...
  → 每轮增加 ~200-500 tokens (Think + Act + Observe)
  → CW 逐步增长...

Iteration N:  tokens 达到 MaxTokens * 65%
  → preSlideCheck() 触发!
  → Slide(): 移除 oldest 消息直到降至 45%
  → EmitSlideEvent → RAG/Memory 消费滑出消息
  → SessionStore 保留完整历史 (WAL)

Iteration N+1:  继续...
  → SS.CurrentContext() 返回最近的 fit-in-budget 消息
  → 新消息正常追加到 SessionStore
  → 无限上下文得以维持
```

---

## 四、接口与类型变更

### 4.1 新增/修改的类型

```go
// Thought — 新增字段
type Thought struct {
    // ... existing fields ...
    SelectedSkill string `json:"selected_skill,omitempty"` // Phase1 选择结果
}

// ActivatedSkillContext — P2/P3 激活上下文 (新增)
type ActivatedSkillContext {
    Skill            *core.Skill       // 完整 Skill 对象
    Instructions     string            // P2: L2 SOP 文本
    FilteredTools    []gochatcore.Tool // P2: 过滤后的 Tool 子集 (native FC)
    FilteredInfos    []core.ToolInfo   // P2: 同上 (goreact 格式)
    ResourceBasePath string            // P3: skill.RootDir
}

// SlideConfig — 滑动窗口参数 (新增)
type SlideConfig struct {
    SlideTriggerRatio   float64 // 默认 0.65
    TargetRatio         float64 // 默认 0.45
    MinPreserveMessages int     // 默认 4
    MaxSlideBatch       int     // 默认 0 (不限)
}

// SlidedMessages — 滑动操作返回值 (新增)
type SlidedMessages struct {
    Messages   []Message
    TokenCount int64
}

// SlideEvent — 滑动事件通知 (新增)
type SlideEvent struct {
    SessionID string
    Slided    []Message
    Remaining int
    Timestamp int64
}

// SlideHandler — 滑动事件回调 (新增)
type SlideHandler func(ctx context.Context, event SlideEvent)
```

### 4.2 SessionStore 接口增强

```go
type SessionStore interface {
    Append(ctx, sessionID, message) error
    Get(ctx, sessionID) ([]Message, error)
    CurrentContext(ctx, sessionID, maxTokens int64) ([]Message, error) // 🆕 Token预算感知
    Delete(ctx, timestamp, sessionID) error
    Clear(ctx, sessionID) error                               // 🆕 会话重置
    SetSlideHandler(handler SlideHandler)                       // 🆕 滑动事件回调
    Close() error                                             // 🆕 资源清理
}
```

### 4.3 Reactor 新增方法

```go
// 滑动窗口
func (r *Reactor) ensureContextWindow(sessionID string) *core.ContextWindow
func (r *Reactor) persistMessage(ctx, role, content string)
func (r *Reactor) checkSlide(ctx context.Context)

// Skill 激活
func (r *Reactor) ActivateSkill(skillName string, allToolInfos []core.ToolInfo) (*ActivatedSkillContext, error)
func (r *Reactor) SetContextWindow(cw *core.ContextWindow)

// 访问器
func (r *Reactor) SessionStore() core.SessionStore
func (r *Reactor) ContextWindow() *core.ContextWindow
```

### 4.4 prompts.go 新增函数

```go
// Skill 相关
func ToolInfosToLLMTools(infos []core.ToolInfo) []gochatcore.Tool
func filterToolsByAllowed(infos []core.ToolInfo, allowedTools string) []core.ToolInfo
func EstimateTokensForTools(tools []gochatcore.Tool, estimateFn func(string) int) int64
func BuildCapabilitiesList(skills []*core.Skill) string           // P1 L1 摘要列表
func BuildSkillsSystemPrompt(skills []*core.Skill) string         // System Prompt 格式
func BuildSkillSelectPrompt(input string, intent *Intent, skills []*core.Skill) string  // Phase1 prompt
func BuildThinkPrompt(input string, intent *Intent, memoryRecords []core.MemoryRecord, actCtx *ActivatedSkillContext) string  // Phase2 prompt (+P2/P3)
```

---

## 五、模板系统

### 5.1 模板文件清单

| 文件 | 用途 | 渲染时机 |
|------|------|----------|
| `default_system_prompt.tmpl` | Agent 身份定义 | 每次 LLM 调用 (Layer 1a) |
| `skill_select_prompt.tmpl` | **🆕** Phase 1 Skill 选择 | Think Phase 1 (仅当有 Skills 时) |
| `think_prompt.tmpl` | Phase 2 Tool 规划 (含 P2/P3 注入) | Think Phase 2 |
| `intent_prompt.tmpl` | Intent 分类 | 每次输入处理 |
| `summary_prompt.tmpl` | 最终答案总结 | 任务完成时 |

### 5.2 Phase 1 模板 (skill_select_prompt.tmpl)

关键特征：
- **不含** `<available_capabilities>` 区块（已移至 System Prompt Layer 1b）
- 规则引用 `"provided in the SYSTEM PROMPT above"`
- 输出格式只要求 `selected_skill` + `reasoning` + `confidence`
- 无需 Tool 信息（纯 Skill 选择）

### 5.3 Phase 2 模板 (think_prompt.tmpl)

关键特征：
- 条件渲染 `{{if .HasActiveSkill}}` → `<active_skill_instructions>`
- 包含完整 L2 Instructions (P2)
- 包含过滤后 Tool 列表 (P2)
- 条件包含 `ResourceBasePath` (P3)：当非空时提示 LLM 使用该前缀构造文件路径
- Rule #11 动态生成：当有 ActiveSkill 时强调遵循其指令

---

## 六、Token 计算完整覆盖

### 6.1 所有 Token 记入点

```
Run() 入口:
  ├─ persistMessage("user", input)     → AddTokens(estimate(input))          ✅ C4
  │
Think() 内部:
  ├─ skillsSection (P1 L1)             → AddTokens                          ✅
  ├─ toolDefs (全部 JSON Schema)        → AddTokens(EstimateForTools)         ✅ C3
  │
  ├─ [Phase 1 仅当有 Skills]:
  │   ├─ selectInstructions             → AddTokens                          ✅
  │   ├─ capabilitiesSection           → AddTokens                          ✅
  │   └─ preSlideCheck()               → Slide if >= 65%                    ✅ W1
  │
  ├─ [ActivateSkill 后]:
  │   ├─ actCtx.Instructions (L2 SOP)   → AddTokens                          ✅ P2
  │   └─ filteredToolDefs              → AddTokens(EstimateForTools)         ✅ P2
  │
  ├─ thinkPrompt (Phase2 指令)         → AddTokens                          ✅
  ├─ preSlideCheck()                   → Slide if >= 65%                    ✅ W1
  │
  └─ LLM response tokens               → totalTokens +=                     ✅
  │
runTAOLoop 结束:
  └─ persistMessage("assistant", stepSummary) → AddTokens(estimate(step)) ✅ W2
  │
Run() 出口:
  ├─ persistMessage("assistant", answer)→ AddTokens(estimate(answer))       ✅
  ├─ AddTokens(LLM response usage)                                         ✅
  └─ checkSlide()                                                        ✅
```

### 6.2 滑动参数

```go
DefaultSlideConfig = SlideConfig{
    SlideTriggerRatio:   0.65,  // 65% 触发 (Lost in the Middle 临界点前)
    TargetRatio:         0.45,  // 滑动后目标 (为 T-A-O 预留 55%)
    MinPreserveMessages: 4,     // 最少保留 2 轮对话
}
```

Token 预算分配 (MaxTokens = 100%):

```
┌─────────────────────────────────────────────────┐
│ System Prompt (Layer 1a+1b)   ~5-15%            │
│ Tool Definitions (Layer 4)     ~10-20%          │
│ User Input + Instructions      ~3-5%            │
├─────────────────────────────────────────────────┤
│ Conversation History (CW)      ~25-45%          │
│   ← 滑动目标: 45%, 触发点: 65%                 │
├─────────────────────────────────────────────────┤
│ Reserved for LLM Output        ~25-30%          │
│   (Think + Act + Observation × N iterations)    │
├─────────────────────────────────────────────────┤
│ Safety Buffer                  ~5-10%           │
└─────────────────────────────────────────────────┘
```

---

## 七、变更文件清单

### 7.1 新建文件 (6 个)

| 文件 | 行数 | 说明 |
|------|------|------|
| `core/session_store_memory.go` | ~120 | MemorySessionStore 纯内存实现 |
| `core/slide_test.go` | ~420 | 滑动窗口专项测试 (22 个用例) |
| `reactor/prompts/skill_select_prompt.tmpl` | ~40 | Phase 1 Skill Selection 模板 |
| `docs/无限上下文实现方案.md` | ~250 | 滑动窗口架构设计文档 |
| `docs/Skill执行链路修复方案.md` | ~300 | 本归档文档 |

### 7.2 修改文件 (7 个)

| 文件 | 变更量级 | 关键改动 |
|------|----------|----------|
| `core/session.go` | **重写** | 接口增强: context.Context, Clear/Close/SetSlideHandler, SlideEvent 类型 |
| `core/ctx_win.go` | **扩展** | +SlideConfig, SlidedMessages, Slide()/SlideTriggered/UsageRatio() |
| `core/tiktoken.go` | **修复** | getGlobalEncoder 加 5s 超时保护 |
| `agent.go` | **简化** | 移除 contextWindow 字段; +WithSessionStore Option; Ask() 变纯代理 |
| `reactor/reactor.go` | **大幅重构** | Think() 两阶段; buildLLMBuilder 四层; persistMessage+checkSlide; +sessionStore/contextWindow/slideConfig 字段 |
| `reactor/prompts.go` | **大幅扩展** | +11 个新函数/类型; 模板数据结构更新; BuildThinkPrompt 签名变更 |
| `reactor/prompts/think_prompt.tmpl` | **重写** | 移除 tools/skills 区块; 增加 active_skill_instructions 条件渲染 + ResourceBasePath |
| `reactor/thought.go` | **修改** | Thought +SelectedSkill; 移除旧 BuildThinkPrompt |
| `reactor/reactor_options.go` | **扩展** | +WithSessionStore Option + sessionStore 字段 |

---

## 八、验证矩阵

### 8.1 编译与静态分析

```
go build ./...     → ✓ 通过
go vet ./...        → ✓ 零警告
```

### 8.2 测试覆盖

| 测试套件 | 用例数 | 状态 | 说明 |
|----------|--------|------|------|
| ContextWindow 滑动 | 10 | ✅ PASS | UsageRatio/SlideTriggered/Slide/MaxBatch/TokenCountAccuracy/边界条件 |
| MemorySessionStore | 8 | ✅ PASS | Append/Get/CurrentContext(Clear/Delete/Handler/Close/并发50协程) |
| 工具函数 | 2 | ✅ PASS | NoopSlideHandler/EmitSlideEvent(nil) |
| 集成测试 | 2 | ✅ PASS | 完整生命周期(Persist→Slide→Event→WAL验证)/多轮累积 |
| core 全量 (排除 tiktimeout) | ~60 | ✅ PASS | ok (5.7-5.8s) |
| reactor 单元 | ~12 | ✅ PASS | ok (2.6-2.7s) |

### 8.3 向后兼容性

| 场景 | 行为 |
|------|------|
| 无 SessionStore 设置 | Fallback 到 MemorySessionStore ✅ |
| 无 Skills 注册 | Think 跳过 Phase 1，直接进入 Phase 2（全部工具）✅ |
| Phase 1 未选 Skill (`selected_skill == ""`) | Phase 2 使用全部工具，无 Skill Instructions ✅ |
| Skill 无 AllowedTools 限制 | `filterToolsByAllowed("")` 返回全部工具 ✅ |
| Skill 无 RootDir (bundled) | ResourceBasePath 为空，模板中 P3 区块不渲染 ✅ |
| tiktoken 网络不可达 | 5s 超时 fallback 到启发式估算 ✅ |

---

## 九、设计决策记录 (ADR)

### ADR-001: System Prompt 追加而非替换

**决策**: `builder.SystemMessage()` 为追加 API。Agent 身份 (Layer 1a) 和 Skill 列表 (Layer 1b) 作为两条独立 system message 共存。

**理由**: 保持原 SystemPrompt 完整性；Skills 作为能力声明自然属于 system layer。

### ADR-002: Phase 1 Skill 列表放入 System Prompt

**决策**: P1 的 `BuildCapabilitiesList()` 结果通过 `skillsSection` 参数传入 `callLLMStream` → 注入 Layer 1b。

**理由**: System Prompt 拥有最高优先级且不计入 user message 的 attention budget；选择指令模板只需引用 "system prompt above" 即可。

### ADR-003: 滑动在 Think 内部 LLM 调用前触发

**决策**: `preSlideCheck()` 放在每个 LLM 调用之前（Phase 1 前 + Phase 2 前），而非仅在 Run() 返回后。

**理由**: 滑动的目的是为即将到来的 LLM 调用预留空间；事后滑动无法挽回已经超预算的请求失败。

### ADR-004: P3 资源不预加载，仅暴露 RootDir

**决策**: 不在 Think 阶段读取 scripts/references/assets 文件内容，而是将 `skill.RootDir` 作为 `ResourceBasePath` 透传给 LLM。

**理由**: 脚本可能很大或需要运行时参数；LLM 在 Action 阶段通过 `bash("python {root}/scripts/x.py")` 或 `read("{root}/refs/y.md")` 按需访问更灵活。

### ADR-005: T-A-O 步骤持久化通过 persistMessage

**决策**: `runTAOLoop` 中的 stepSummary 同时写入 `AddMessage` (内存历史) 和 `persistMessage` (SS + CW + Tokens)。

**理由**: 中间步骤是完整对话的一部分，应参与滑动窗口的 Token 计算和持久化存储。
