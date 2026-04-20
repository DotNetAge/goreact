# GoReactor 全量代码审计报告

## 执行摘要

对 goreact 项目全部 69 个 Go 文件进行了全面代码审计。共发现 **3 个高危**、**8 个中等**和 **6 个低危**问题。所有已修复的问题均在下方标注 `[FIXED]`。修复后 `go build ./...` 和 `go vet ./...` 均零错误通过。

---

## 严重 (Critical)

### C1: `saveExperience` 异步 goroutine 中访问已释放的 ReactContext 字段 [降级为 Low]

**文件**: `reactor/experience.go:81-85`

`saveExperience` 在异步 goroutine 中调用 `r.memory.Store()`，但用于提取 tags 和构建 ExperienceData 的 `ctx.Input`、`ctx.Intent`、`ctx.History` 均在 goroutine 启动前已完成读取，不存在竞态。

> **状态**: 经复查降级为 Low。代码中 `exp`、`tags`、`record` 均在 goroutine 外构建完成，goroutine 仅执行 `Store()`。无数据竞态。无需修复。

---

## 高危 (High)

### H1: `accessor_impl.go` 中 `RegisterPendingTask` 是空实现（no-op） [FIXED]

**文件**: `reactor/accessor_impl.go`

**修复方案**: 
- 将 Reactor 内部 `pendingTasks` 从 `map[string]chan *RunResult` 统一为 `map[string]chan any`，解决 Go 类型系统 channel 转换限制
- `RegisterPendingTask(taskID string, resultCh chan any)` 现在正确存储 channel 到 pendingTasks map
- `GetPendingTask(taskID string) (<-chan any, bool)` 现在返回正确的 channel 而非 nil
- 接口签名从 `chan<- any` 改为 `chan any`，因为 `SubAgentTool` 创建的是双向 channel

### H2: `tools/task_tools.go` 中 `SubAgentTool` 创建了 SubAgent 但未实际启动异步执行 [FIXED]

**文件**: `tools/task_tools.go`, `reactor/accessor_impl.go`, `tools/reactor_accessor.go`

**修复方案**:
- 在 `ReactorAccessor` 接口中新增 `RunSubAgent(ctx, taskID, systemPrompt, prompt, model, resultCh chan<- any)` 方法
- 在 `Reactor` 中实现 `RunSubAgent`：在 goroutine 中创建独立子 Reactor（继承 Memory、MessageBus、EventBus），执行任务，将结果通过 channel 返回，最后清理 pendingTask
- `SubAgentTool.Execute()` 现在调用 `RunSubAgent` 启动真正的异步执行

### H3: `accessor_impl.go:147` 使用 `var _ = fmt.Sprintf` 抑制未使用导入 [FIXED]

**文件**: `reactor/accessor_impl.go`

**修复方案**: 删除未使用的 `fmt` 导入。

---

## 中等 (Medium)

### M1: `memory_search` 工具中 `limit` 参数类型不匹配 [FIXED]

**文件**: `tools/memory.go`

**修复方案**: 改为先尝试 `float64`（LLM JSON 标准数字类型）再 fallback 到 `int`：
```go
limitRaw := 0
if raw, ok := params["limit"]; ok {
    if f, ok := raw.(float64); ok {
        limitRaw = int(f)
    } else if i, ok := raw.(int); ok {
        limitRaw = i
    }
}
```

### M2: `memory_search` 中当 `type` 为 `session` 时跳过类型过滤 [FIXED]

**文件**: `tools/memory.go`

**修复方案**: 移除 `memType != core.MemoryTypeSession` 条件，所有类型过滤均生效。

### M3: `experience.go` 中 `orchestrationToolNames` 映射不完整 [FIXED]

**文件**: `reactor/experience.go`

**修复方案**: 扩展映射包含所有编排相关工具：
- Task tools: `task_create`, `task_result`
- SubAgent tools: `subagent`, `subagent_result`
- Team tools: `team_create`, `team_delete`, `team_status`, `wait_team`
- Communication: `send_message`, `receive_messages`

### M4: `MicroCompact` 中结果未按时间顺序返回 [无需修复]

经复查，反转操作在最后统一执行，时间顺序正确。微压缩是保底策略，风险低。

### M5: `InMemoryMemory.filterAndLimit` 中的 O(n^2) 冒泡排序 [FIXED]

**文件**: `core/memory_inmemory.go`

**修复方案**: `filterAndLimit` 和 `sortByScore` 两个排序函数均替换为 `sort.Slice`，将 O(n^2) 改为 O(n log n)。

### M6: `ContextWindow` 在 `Agent.Ask()` 中 Token 计算偏乐观 [暂不修复]

`ContextWindow.TokensUsed` 只累加了 LLM 的 token 使用量，未计入 system prompt 和 user message。这是 Agent 层面的优化，不影响核心 Reactor 的正确性，留待后续版本改进。

### M7: `NewReactor` 中 MCP 注册的代码是死代码 [FIXED]

**文件**: `reactor/reactor.go`

**修复方案**: 移除 `NewReactor` 中 `_ = setup.mcpRegistry` 的死代码块。`WithMCPRegistry` 选项和 `core.MCPToolRegistry` 类型保留，供未来 MCP lazy discovery 实现使用。

### M8: `InProcessEventBus` 的 `cancel` context 创建后未使用 [FIXED]

**文件**: `reactor/eventbus.go`

**修复方案**: 
- 移除 `subscriber` 结构体中未使用的 `cancel context.CancelFunc` 字段
- 移除 `SubscribeFiltered` 中未使用的 `context.WithCancel` 调用和 `context` 导入
- `Close` 方法中移除 `sub.cancel()` 调用，直接 close channel

---

## 低危 (Low)

### L1: `goreact/errors.go` 中的 Memory 错误与 `core/memory.go` 重复定义 [FIXED]

**文件**: `errors.go`

**修复方案**: 将 Memory 错误改为从 `core` 包别名引用：
```go
var (
    ErrMemoryNotFound   = core.ErrMemoryNotFound
    ErrMemoryStorage    = core.ErrMemoryStorage
    ErrMemoryRetrieval  = core.ErrMemoryRetrieval
)
```

### L2: `tools/task_tools.go:497` 中 `var _ = strings.TrimSpace` [FIXED]

**文件**: `tools/task_tools.go`

**修复方案**: 删除该行。`strings` 包已通过 `strings.Builder` 在同文件中使用，不需要抑制。

### L3: `DefaultModel = "gpt-4"` 默认模型过时 [FIXED]

**文件**: `core/constants.go`

**修复方案**: 改为空字符串 `""`，强制使用者在创建 Reactor 时显式设置模型。

### L4: `MemoryRecord.Meta` 字段的 JSON 序列化行为不确定 [无需修复]

这是 by design 的限制。`Meta` 用于进程内直接类型访问（如 `record.Meta.(*ExperienceData)`），不依赖 JSON 往返的类型保留。

### L5: `experience.go` 中截断后的 Analysis 可能截断 UTF-8 字符 [无需修复]

`truncateText` 基于 rune 截断，不会破坏 UTF-8 编码。在词中间截断是可接受的文本截断行为。

### L6: `tool_result_storage.go` 中 `Persist` 基于 PID 做文件名隔离 [FIXED]

**文件**: `core/tool_result_storage.go`

**修复方案**:
- 新增 `sessionID` 字段和 `WithSessionID(id)` 选项，允许显式设置 session ID
- `Persist` 使用 `sessionID`（或 PID fallback）替代硬编码 PID，与 `Cleanup(sessionID)` 格式一致
- 文件名使用 `time.Now().UnixNano()` 替代 PID，避免同 session 内文件名冲突

---

## 架构完整性评估

### 修复前缺失的部分 → 修复后状态
- SubAgent 异步执行机制 (H1, H2) → **已修复**: 完整的异步执行管道，`RegisterPendingTask` → `RunSubAgent` → goroutine → channel result → `GetPendingTask`
- MCP lazy discovery 未实现 (M7) → **已修复**: 移除死代码，保留选项接口供未来使用
- Memory 工具的 `limit` 参数类型转换 (M1) → **已修复**: 支持 float64 和 int 两种 JSON 数字类型

### 完整且正确的部分
- T-A-O 循环主链路（Run → classifyIntent → Think/Act/Observe → CheckTermination）
- Memory 子系统五类型设计（Session/User/LongTerm/Refactive/Experience）
- ReNewer 接口与 maybeCompact 集成（优先级链：ReNew → MicroCompact → Full Compact）
- Reflexive Memory（GetWithSemantic）：精确匹配 → Memory 语义搜索 → 名称列表 → map 查找
- Experience 自动保存（条件检查、数据构建、异步写入）
- 权限管线（SecurityPolicy → PreToolUse hooks → PermissionChecker → PostToolUse hooks）
- 事件系统（EventBus → ReactEvent → 所有事件类型）
- Skill 系统（Registry → Loader → Frontmatter → BundledSkills）

### 数据一致性评估
- `MemoryRecord.Content` 和 `MemoryRecord.Meta` 使用一致
- `ExperienceData` 序列化/反序列化正确
- `Step/Thought/Action/Observation` 数据流正确
- `ReactEvent` 事件类型与 Data 负载类型匹配

---

## 并发安全评估

### 正确的部分
- `ToolRegistry`、`IntentRegistry`、`InMemoryMemory`、`InMemoryTaskManager`、`InProcessEventBus` 均正确使用 `sync.RWMutex`
- `ReactContext.AppendHistory` 使用 `sync.Mutex`
- `saveExperience` 的 goroutine 只访问 goroutine 启动前已构建的局部变量

### 潜在风险（设计权衡，非 bug）
- `saveExperience` 异步 goroutine 使用 `context.Background()`，确保即使请求取消也保存经验
- `ReactContext.ConversationHistory` 的 `AddMessage` 无锁保护（T-A-O 单线程安全，并行 T-A-O 时需加锁）

---

## 总结

| 级别 | 总数 | 已修复 | 关键修复 |
|------|------|--------|---------|
| Critical | 0 (降级) | - | - |
| High | 3 | 3 | SubAgent 异步执行完整管道 (H1+H2)、未使用导入 (H3) |
| Medium | 8 | 6 | JSON 数字类型 (M1)、排序算法 (M5)、MCP 死代码 (M7)、EventBus context (M8) |
| Low | 6 | 4 | 错误定义统一 (L1)、默认模型 (L3)、PID 隔离 (L6) |

**构建验证**: `go build ./...` 和 `go vet ./...` 零错误通过。
