# Registry 接口化改造方案

## 目标

将 `IntentRegistry` 和 `ToolRegistry` 抽象为接口（类似已有的 `core.SkillRegistry`），
使客户端可以：
1. 通过 `WithXXXRegistry()` 注入自定义实现
2. 通过嵌入 `DefaultXXXRegistry` 并重写部分方法来增强能力
3. 不注入时自动使用默认实现

## 一、接口定义

### 1.1 IntentRegistryInterface

```go
// intent_interface.go (新文件)

// IntentRegistryInterface defines the contract for intent type management.
// Implementations can override classification behavior, e.g., using LLM-based
// semantic matching instead of the default keyword-based approach.
type IntentRegistryInterface interface {
    // Register adds a new intent definition. Returns error if type already exists.
    Register(def IntentDefinition) error

    // Unregister removes an intent definition by type.
    Unregister(typ string)

    // All returns a copy of all registered intent definitions.
    All() []IntentDefinition

    // FormatPromptSection renders intents into the classification prompt.
    FormatPromptSection() string
}
```

**对应的方法来源** (`intent.go`):
- `Register` — 第 85 行
- `Unregister` — 第 99 行
- `All` — 第 111 行
- `FormatPromptSection` — 第 120 行

### 1.2 ToolRegistryInterface

```go
// tool_interface.go (新文件)

// ToolRegistryInterface defines the contract for tool lifecycle management.
// This interface covers both tool operations and configuration methods,
// enabling full replacement or selective enhancement via embedding.
type ToolRegistryInterface interface {
    // --- Core Operations ---

    // Register adds a tool. Returns error if name already exists.
    Register(tool core.FuncTool) error

    // Get returns a tool by exact name match.
    Get(name string) (core.FuncTool, bool)

    // All returns all registered tools.
    All() []core.FuncTool

    // ToToolInfos extracts metadata from all tools for prompt building.
    ToToolInfos() []core.ToolInfo

    // GetWithSemantic finds a tool by name, falling back to memory-based
    // semantic search if exact match fails.
    GetWithSemantic(ctx context.Context, name string, intent string) (core.FuncTool, bool)

    // ExecuteTool runs a tool with full permission pipeline support.
    ExecuteTool(ctx context.Context, name string, params map[string]any) (string, time.Duration, error)

    // --- Configuration Methods ---

    // SetSecurityPolicy sets the legacy security policy (deprecated).
    SetSecurityPolicy(policy SecurityPolicy)

    // SetPermissionChecker sets the fine-grained permission checker.
    SetPermissionChecker(checker core.ToolPermissionChecker)

    // SetResultStorage configures result persistence for context defense.
    SetResultStorage(storage core.ToolResultStorage)

    // SetResultLimits configures per-result size limits.
    SetResultLimits(limits core.ToolResultLimits)

    // SetMemory injects Memory for reflexive semantic search.
    SetMemory(mem core.Memory)

    // AddHook registers a lifecycle hook (PreToolUse / PostToolUse).
    AddHook(hook core.Hook)

    // SetEventEmitter sets the callback for permission events.
    SetEventEmitter(fn func(core.ReactEvent))

    // ResetMessageCharCounter resets per-cycle char tracking.
    ResetMessageCharCounter()
}
```

**对应的方法来源** (`action.go`):
| 方法 | 行号 | 用途 |
|------|------|------|
| Register | 103 | 注册工具 |
| Get | 115 | 按名查找 |
| All | 123 | 获取全部 |
| ToToolInfos | 181 | 提取元信息给 Prompt |
| GetWithSemantic | 147 | 语义查找（含 Memory fallback） |
| ExecuteTool | 237 | 执行工具（含权限管道） |
| SetSecurityPolicy | 194 | 配置 |
| SetPermissionChecker | 203 | 配置 |
| SetResultStorage | 81 | 配置 |
| SetResultLimits | 88 | 配置 |
| SetMemory | 136 | 配置 |
| AddHook | 211 | 配置 |
| SetEventEmitter | 220 | 配置 |
| ResetMessageCharCounter | 96 | 运行时调用 |

### 1.3 SkillRegistry 已有接口

无需新建。现有的 `core.SkillRegistry` 接口已满足需求：
```go
// core/skill.go:44-58 (已有)
type SkillRegistry interface {
    RegisterSkill(skill *Skill) error
    GetSkill(name string) (*Skill, error)
    ListSkills() []*Skill
    FindApplicableSkills(context any) ([]*Skill, error)
}
```

唯一需要做的是将 `defaultSkillRegistry` 改名为 `DefaultSkillRegistry` 并公开。

---

## 二、文件变更清单

### 新增文件 (3 个)

| 文件 | 内容 |
|------|------|
| `reactor/intent_interface.go` | `IntentRegistryInterface` 接口定义 |
| `reactor/tool_interface.go` | `ToolRegistryInterface` 接口定义 |

### 修改文件 (5 个)

| 文件 | 变更内容 |
|------|---------|
| `reactor/reactor.go` | Reactor struct 字段改为接口类型 + 添加 WithXXXRegistry Option + 延迟初始化逻辑 |
| `reactor/reactorSetup` | 添加 3 个可选 registry 字段 |
| `reactor/thought.go` | `BuildIntentPrompt` 参数从 `*IntentRegistry` 改为 `IntentRegistryInterface` |
| `reactor/action.go` | 无需改动（ToolRegistry 变为接口后自动兼容） |
| `reactor/skill_registry.go` | `defaultSkillRegistry` → `DefaultSkillRegistry`（公开） |

---

## 三、详细代码变更

### 3.1 reactorSetup 添加 registry 注入字段

```go
type reactorSetup struct {
    // ... existing fields ...

    // === Registry Injection (optional, nil = use default) ===
    intentRegistry IntentRegistryInterface   // NEW
    toolRegistry   ToolRegistryInterface     // NEW
    skillRegistry  core.SkillRegistry        // NEW
}
```

### 3.2 新增 3 个 WithOption 函数

```go
// WithIntentRegistry sets a custom IntentRegistry implementation.
// Use this to provide LLM-based intent classification, custom intent types, etc.
// If not set, DefaultIntentRegistry with built-in definitions is used.
func WithIntentRegistry(reg IntentRegistryInterface) ReactorOption {
    return func(s *reactorSetup) {
        s.intentRegistry = reg
    }
}

// WithToolRegistry sets a custom ToolRegistry implementation.
// Use this to add dynamic tool discovery, MCP integration, semantic filtering, etc.
// If not set, DefaultToolRegistry is used.
func WithToolRegistry(reg ToolRegistryInterface) ReactorOption {
    return func(s *reactorSetup) {
        s.toolRegistry = reg
    }
}

// WithSkillRegistry sets a custom SkillRegistry implementation.
// Use this to provide embedding-based semantic skill matching, etc.
// If not set, DefaultSkillRegistry is used.
func WithSkillRegistry(reg core.SkillRegistry) ReactorOption {
    return func(s *reactorSetup) {
        s.skillRegistry = reg
    }
}
```

### 3.3 Reactor struct 字段改为接口

```go
type Reactor struct {
    config         ReactorConfig
    intentRegistry IntentRegistryInterface      // CHANGED: was *IntentRegistry
    toolRegistry   ToolRegistryInterface        // CHANGED: was *ToolRegistry
    skillRegistry  core.SkillRegistry           // UNCHANGED: already interface
    taskManager    core.TaskManager
    // ... rest unchanged ...
}
```

### 3.4 NewReactor 延迟初始化逻辑

```go
r := &Reactor{
    config:          config,
    taskManager:     core.NewInMemoryTaskManager(),
    compactorConfig: core.DefaultCompactorConfig(),
    tokenEstimator:  core.NewDefaultTokenEstimator(3.0),
    memory:          setup.memory,
    mockLLM:         setup.mockLLM,
}

// === Registry Initialization (use injected or create defaults) ===
if setup.intentRegistry != nil {
    r.intentRegistry = setup.intentRegistry
} else {
    r.intentRegistry = NewDefaultIntentRegistry()  // CHANGED: renamed
}

if setup.toolRegistry != nil {
    r.toolRegistry = setup.toolRegistry
} else {
    r.toolRegistry = NewDefaultToolRegistry()    // CHANGED: renamed
}

if setup.skillRegistry != nil {
    r.skillRegistry = setup.skillRegistry
} else {
    r.skillRegistry = NewDefaultSkillRegistry()  // CHANGED: renamed & public
}
```

### 3.5 重命名公开默认实现

**intent.go:**
```go
// BEFORE:
type IntentRegistry struct { ... }       // concrete, public
func NewIntentRegistry() *IntentRegistry { ... }

// AFTER:
type DefaultIntentRegistry struct { ... } // concrete, public (renamed)
func NewDefaultIntentRegistry() *DefaultIntentRegistry { ... }

// Type alias for backward compatibility
type IntentRegistry = DefaultIntentRegistry
var _ IntentRegistryInterface = (*DefaultIntentRegistry)(nil) // compile-time check
```

**skill_registry.go:**
```go
// BEFORE:
type defaultSkillRegistry struct { ... }  // private!
func NewSkillRegistry() core.SkillRegistry { return &defaultSkillRegistry{...} }

// AFTER:
type DefaultSkillRegistry struct { ... }  // public! (renamed)
func NewDefaultSkillRegistry() core.SkillRegistry { return &DefaultSkillRegistry{...} }

var _ core.SkillRegistry = (*DefaultSkillRegistry)(nil) // compile-time check
```

**action.go:**
```go
// BEFORE:
type ToolRegistry struct { ... }        // concrete, public
func NewToolRegistry() *ToolRegistry { ... }

// AFTER:
type DefaultToolRegistry struct { ... }  // concrete, public (renamed)
func NewDefaultToolRegistry() *DefaultToolRegistry { ... }

// Type alias for backward compatibility
type ToolRegistry = DefaultToolRegistry
var _ ToolRegistryInterface = (*DefaultToolRegistry)(nil) // compile-time check
```

### 3.6 BuildIntentPrompt 参数调整

```go
// thought.go:155
// BEFORE:
func BuildIntentPrompt(input string, context string, registry *IntentRegistry) string {

// AFTER:
func BuildIntentPrompt(input string, context string, registry IntentRegistryInterface) string {
    if registry == nil {
        registry = NewDefaultIntentRegistry()
    }
    // ... rest unchanged
}
```

---

## 四、向后兼容性保证

### 类型别名策略

通过 Go 的类型别名（type alias）保证零破坏性迁移：

```go
// 用户旧代码：
reg := reactor.NewIntentRegistry()           // 编译错误！→ 改为 NewDefaultIntentRegistry()

// 或者使用类型别名：
type IntentRegistry = DefaultIntentRegistry  // 别名存在，但构造函数名变了
```

**注意**: 构造函数名必须变化（`NewIntentRegistry` → `NewDefaultIntentRegistry`）。
这是一个小的 breaking change，但可以通过保留旧函数名作为 deprecated wrapper 来缓解：

```go
// Deprecated: Use NewDefaultIntentRegistry instead.
func NewIntentRegistry() *DefaultIntentRegistry {
    return NewDefaultIntentRegistry()
}

// Deprecated: Use NewDefaultToolRegistry instead.
func NewToolRegistry() *DefaultToolRegistry {
    return NewDefaultToolRegistry()
}
```

### 辅助访问器方法不变

这些公开方法保持原样，返回接口类型：
```go
func (r *Reactor) ToolRegistry() ToolRegistryInterface   // 返回类型变了
func (r *Reactor) IntentRegistry() IntentRegistryInterface // 返回类型变了
func (r *Reactor) SkillRegistry() core.SkillRegistry      // 不变
```

---

## 五、使用示例

### 示例 1：语义匹配的 SkillRegistry

```go
import "github.com/DotNetAge/goreact/core"
import "github.com/DotNetAge/goreact/reactor"

// SemanticSkillRegistry 使用 embedding 增强 DefaultSkillRegistry
type SemanticSkillRegistry struct {
    *reactor.DefaultSkillRegistry  // 嵌入默认实现
    embedder  *embedding.Client    // 外部模型用于语义匹配
}

func (s *SemanticSkillRegistry) FindApplicableSkills(context any) ([]*core.Skill, error) {
    // 先尝试父类关键词匹配（兜底）
    baseResults, _ := s.DefaultSkillRegistry.FindApplicableSkills(context)
    
    // 如果关键词匹配有结果，直接返回
    if len(baseResults) > 0 {
        return baseResults, nil
    }
    
    // 关键词无结果 → 使用语义匹配增强
    intent := context.(*reactor.Intent)
    query := intent.Type + " " + intent.Topic + " " + intent.Summary
    
    skills := s.DefaultSkillRegistry.ListSkills()
    var bestMatch *core.Skill
    bestScore := 0.0
    
    for _, sk := range skills {
        score, _ := s.embedder Similarity(query, sk.Description+" "+sk.Name)
        if score > bestScore && score > 0.7 {
            bestScore = score
            bestMatch = sk
        }
    }
    
    if bestMatch != nil {
        return []*core.Skill{bestMatch}, nil
    }
    return nil, nil
}

// 使用
r := reactor.NewReactor(config,
    reactor.WithSkillRegistry(&SemanticSkillRegistry{
        DefaultSkillRegistry: reactor.NewDefaultSkillRegistry().(*reactor.DefaultSkillRegistry),
        embedder: myEmbeddingClient,
    }),
)
```

### 示例 2：动态发现的 ToolRegistry

```go
type MCPToolRegistry struct {
    *reactor.DefaultToolRegistry
    mcpClient *mcp.Client
}

func (m *MCPToolRegistry) ToToolInfos() []core.ToolInfo {
    // 合并本地工具 + MCP 远程工具
    local := m.DefaultToolRegistry.ToToolInfos()
    remote := m.mcpClient.ListTools()
    return append(local, remote...)
}

func (m *MCPToolRegistry) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, time.Duration, error) {
    // 先检查是否为远程工具
    if m.mcpClient.HasTool(name) {
        return m.mcpClient.CallTool(ctx, name, params)
    }
    // 否则委托给默认实现
    return m.DefaultToolRegistry.ExecuteTool(ctx, name, params)
}
```

### 示例 3：LLM 增强的 IntentRegistry

```go
type LLMIntentRegistry struct {
    *reactor.DefaultIntentRegistry
    llmClient gochat.ClientBuilder
}

func (l *LLMIntentRegistry) FormatPromptSection() string {
    // 可以在这里加入自定义的 intent 类型描述
    // 例如从数据库加载领域特定的 intent 定义
    base := l.DefaultIntentRegistry.FormatPromptSection()
    
    // 追加自定义意图类型
    customIntents := l.loadCustomIntentsFromDB()
    for _, ci := range customIntents {
        base += fmt.Sprintf("%d. **%s** - %s\n", len(customIntents)+1, ci.Type, ci.Description)
    }
    return base
}
```

---

## 六、影响范围评估

### 必须同步更新的内部调用点

| 文件 | 行号 | 调用 | 影响 |
|------|------|------|------|
| `reactor.go:806` | Think | `r.toolRegistry.ToToolInfos()` | 无需改（接口包含此方法） |
| `reactor.go:809` | Think | `r.skillRegistry.FindApplicableSkills(ctx.Intent)` | 无需改 |
| `reactor.go:884` | Act | `r.toolRegistry.ExecuteTool(...)` | 无需改 |
| `reactor.go:1080` | runTAOLoop | `r.toolRegistry.ResetMessageCharCounter()` | 无需改 |
| `reactor.go:516-521` | Init | `r.toolRegistry.SetXxx(...)` | 无需改（接口包含） |
| `reactor.go:556` | Init | `r.toolRegistry.Get("cron")` | 无需改 |
| `reactor.go:629` | RegisterTool | `r.toolRegistry.Register(tool)` | 无需改 |
| `thought.go:155` | BuildIntentPrompt | 参数 `*IntentRegistry` → `IntentRegistryInterface` | 需要改 |
| `reactor.go:774` | classifyIntent | `BuildIntentPrompt(..., r.intentRegistry)` | 无需改（自动适配） |

### 外部用户影响

| 变更 | 影响范围 | 兼容方式 |
|------|---------|---------|
| `*IntentRegistry` → `IntentRegistryInterface` | 直接引用 `*IntentRegistry` 的外部代码 | 类型别名 |
| `*ToolRegistry` → `ToolRegistryInterface` | 直接引用 `*ToolRegistry` 的外部代码 | 类型别名 |
| `NewIntentRegistry()` | 构造函数名 | Deprecated wrapper |
| `NewToolRegistry()` | 构造函数名 | Deprecated wrapper |
| `NewSkillRegistry()` | 返回值不变（已是接口） | 无影响 |
| `defaultSkillRegistry` | 包私有，外部不可见 | 仅改名 |

---

## 七、实施步骤建议

### Step 1: 创建接口定义文件
- `reactor/intent_interface.go` — `IntentRegistryInterface`
- `reactor/tool_interface.go` — `ToolRegistryInterface`

### Step 2: 重命名默认实现
- `IntentRegistry` → `DefaultIntentRegistry` (+ 类型别名 + Deprecated 构造函数)
- `ToolRegistry` → `DefaultToolRegistry` (+ 类型别名 + Deprecated 构造函数)
- `defaultSkillRegistry` → `DefaultSkillRegistry`

### Step 3: 修改 Reactor 结构
- 字段类型改为接口
- reactorSetup 添加 3 个可选字段
- 添加 3 个 WithXXXRegistry Option
- NewReactor 中延迟初始化

### Step 4: 更新 BuildIntentPrompt
- 参数类型从 `*IntentRegistry` 改为 `IntentRegistryInterface`

### Step 5: 更新辅助访问器
- `ToolRegistry()` 返回类型改为 `ToolRegistryInterface`
- `IntentRegistry()` 返回类型改为 `IntentRegistryInterface`

### Step 6: 编译验证 + 测试
- 确保所有现有测试通过
- 编写新的接口实现示例测试
