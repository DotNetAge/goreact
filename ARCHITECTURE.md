# GoReAct: 架构设计文档

## 1. 项目概述

GoReAct 是一个基于 Go 语言开发的高性能、可扩展的 ReAct 引擎框架，专注于提供构建 AI 系统的核心推理和决策能力。

### 1.1 核心价值
- **专注核心**：聚焦于 ReAct 引擎的核心推理和决策能力
- **高性能**：利用 Go 语言的并发特性和内存管理优势
- **可扩展性**：模块化设计，通过扩展点支持外部功能集成
- **易用性**：简洁的 API 设计，极低的学习成本
- **整洁架构**：依赖方向单一，从外向内

### 1.2 设计理念

#### Agent = Name + System Prompt
```go
type Agent struct {
    Name         string  // 智能体名称
    SystemPrompt string  // 系统提示词（定义行为）
}
```
- Agent 是纯数据结构，不持有执行能力
- 通过 Coordinator 注入 Executor 执行任务
- System Prompt 定义 Agent 的角色和行为

#### Skill = 经验数据（可进化）
```go
type Skill struct {
    Name         string            // 技能名称
    Description  string            // 技能描述
    Instructions string            // 指令集（Markdown）
    Scripts      map[string]string // 脚本文件
    References   map[string]string // 参考文档
    Statistics   *SkillStatistics  // 统计数据
}
```
- Skill 是一套做事的方案（基于 Agent Skills 规范）
- 包含分步指令、示例、边界情况处理
- 支持评估、进化、归档（优胜劣汰）

#### 依赖注入 + 单向依赖
```
Coordinator (持有 Executor)
    ↓
  Agent (纯数据)
    ↓
  Executor.Execute(SystemPrompt + Task)
```
- Agent 不依赖 Engine
- Coordinator 注入 Executor
- 依赖方向：外层 → 内层

---

## 2. 当前架构

### 2.1 核心架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        Application Layer                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Coordinator  │  │ SkillManager │  │ AgentManager │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                         Engine Layer                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                    ReAct Engine                       │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │   │
│  │  │ Thinker  │→ │  Actor   │→ │ Observer │           │   │
│  │  └──────────┘  └──────────┘  └──────────┘           │   │
│  │                      ↓                                │   │
│  │              ┌──────────────┐                        │   │
│  │              │LoopController│                        │   │
│  │              └──────────────┘                        │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      Infrastructure Layer                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   LLM    │  │   Tool   │  │  Cache   │  │  Memory  │   │
│  │ Clients  │  │  System  │  │  System  │  │  System  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 包结构

```
pkg/
├── agent/              # Agent 系统
│   ├── agent.go           # Agent 实体（Name + SystemPrompt）
│   ├── coordinator.go     # 协调器（持有 Executor，负责任务分配）
│   ├── decomposer.go      # 任务拆解接口（可集成 AgenticRAG）
│   └── manager.go         # Agent 管理器（注册/查询）
│
├── skill/              # Skill 系统
│   ├── skill.go           # Skill 实体（经验数据）
│   └── manager.go         # SkillManager（加载/评估/进化）
│
├── engine/             # ReAct 引擎核心
│   ├── engine.go          # 主引擎（实现 Executor 接口）
│   └── options.go         # 配置选项
│
├── core/               # 核心模块
│   ├── thinker.go         # 思考模块（LLM 推理）
│   ├── actor.go           # 行动模块（工具执行）
│   ├── observer.go        # 观察模块（结果分析）
│   ├── loop_controller.go # 循环控制
│   └── context.go         # 上下文管理
│
├── tool/               # 工具系统
│   ├── tool.go            # 工具接口
│   ├── manager.go         # 工具管理器
│   └── builtin/           # 内置工具
│       ├── calculator.go  # 计算器
│       ├── datetime.go    # 日期时间
│       ├── http.go        # HTTP 请求
│       ├── bash.go        # Bash 执行
│       ├── filesystem.go  # 文件系统
│       ├── grep.go        # 文本搜索
│       └── ...            # 其他工具
│
├── llm/                # LLM 集成
│   ├── client.go          # LLM 客户端接口
│   ├── mock/              # Mock 客户端（测试用）
│   ├── ollama/            # Ollama 集成
│   ├── openai/            # OpenAI 集成
│   └── anthropic/         # Anthropic 集成
│
├── cache/              # 缓存系统
│   ├── cache.go           # 缓存接口
│   └── memory.go          # 内存缓存实现
│
├── memory/             # 内存管理
│   ├── memory.go          # 内存管理接口
│   └── manager.go         # 默认实现
│
├── prompt/             # 提示管理
│   ├── prompt.go          # 提示管理接口
│   └── manager.go         # 模板管理
│
├── model/              # 模型管理
│   ├── model.go           # 模型接口
│   ├── manager.go         # 模型管理器
│   └── llm_adapter.go     # LLM 适配器
│
├── metrics/            # 指标收集
│   └── metrics.go         # 指标接口和实现
│
├── rag/                # RAG 扩展点
│   └── rag.go             # RAG 接口定义
│
└── types/              # 核心类型定义
    └── types.go           # Result, Thought, Action 等
```

---

## 3. 核心模块详解

### 3.1 Agent 系统

#### Agent 实体
```go
type Agent struct {
    Name         string  // 智能体名称
    SystemPrompt string  // 系统提示词
}
```

**设计原则**：
- Agent 极简化，只包含名称和系统提示词
- 不持有任何执行能力（Engine/Executor）
- 不持有 Skills（Skills 由外部管理）

#### Coordinator（协调器）
```go
type Coordinator struct {
    agents         []*Agent
    executor       Executor       // 执行器（注入）
    taskDecomposer TaskDecomposer // 任务拆解器（注入）
}
```

**职责**：
1. 注册和管理 Agent
2. 根据任务选择合适的 Agent
3. 使用 TaskDecomposer 拆解复杂任务
4. 调用 Executor 执行任务

**执行流程**：
```
Task → TaskDecomposer.Decompose() → SubTasks
     → Coordinator.selectAgent() → Agent
     → Executor.Execute(Agent.SystemPrompt + Task) → Result
```

#### TaskDecomposer（任务拆解器）
```go
type TaskDecomposer interface {
    Decompose(task string, context interface{}) ([]SubTask, error)
}

type SubTask struct {
    ID           string   // 子任务 ID
    Description  string   // 子任务描述
    Dependencies []string // 依赖的子任务 ID
}
```

**设计目的**：
- 提供外部接入点，可集成 AgenticRAG 等系统
- 默认实现不拆解，直接返回原任务
- 支持子任务依赖关系

---

### 3.2 Skill 系统

#### Skill 实体
```go
type Skill struct {
    // Frontmatter（来自 SKILL.md）
    Name         string            // 技能名称
    Description  string            // 技能描述
    License      string            // 许可证
    Metadata     map[string]string // 元数据

    // Body 内容
    Instructions string            // 技能指令（Markdown）

    // 可选目录
    Scripts      map[string]string // scripts/ 目录
    References   map[string]string // references/ 目录
    Assets       map[string][]byte // assets/ 目录

    // 运行时统计
    Statistics   *SkillStatistics
}
```

**基于 Agent Skills 规范**：
- SKILL.md 包含 YAML frontmatter + Markdown 指令
- 支持 scripts/、references/、assets/ 目录
- 渐进式加载：先加载 name/description，需要时加载完整指令

#### SkillManager
```go
type Manager interface {
    LoadSkill(path string) (*Skill, error)
    RegisterSkill(skill *Skill) error
    GetSkill(name string) (*Skill, error)
    SelectSkill(task string) (*Skill, error)
    RecordExecution(name string, success bool, ...) error
    EvolveSkills() error  // 技能进化（优胜劣汰）
}
```

**技能进化机制**：
```
评分公式：
Score = 成功率×0.4 + 效率×0.25 + 质量×0.25 + 频率×0.1

进化规则：
- Score ≥ 0.8：优秀技能，保留并推广
- 0.6 ≤ Score < 0.8：良好技能，保留观察
- 0.4 ≤ Score < 0.6：一般技能，触发优化
- Score < 0.4：劣质技能，进入淘汰流程
```

---

### 3.3 Engine（ReAct 引擎）

#### Engine 结构
```go
type Engine struct {
    // 核心模块
    thinker        core.Thinker
    actor          core.Actor
    observer       core.Observer
    loopController core.LoopController

    // 管理器
    toolManager    *tool.Manager
    agentManager   *agent.Manager
    skillManager   *skill.DefaultManager
    modelManager   model.ModelManager
    memoryManager  memory.MemoryManager
    promptManager  prompt.PromptManager

    // 基础设施
    llmClient      llm.Client
    cache          cache.Cache
    metrics        metrics.Metrics
}
```

#### ReAct 循环
```
1. Thinker.Think(task, context) → Thought
   ↓
2. 检查 Thought.ShouldFinish
   ↓ (如果有 Action)
3. Actor.Act(action, context) → ExecutionResult
   ↓
4. Observer.Observe(result, context) → Feedback
   ↓
5. LoopController.Control(state) → Continue/Stop
   ↓
6. 回到步骤 1（如果继续）
```

#### 核心模块职责

**Thinker（思考模块）**：
- 使用 LLM 分析任务
- 生成推理过程和行动计划
- 决定是否需要执行工具
- 决定是否完成任务

**Actor（行动模块）**：
- 执行 Thought 中的 Action
- 调用 ToolManager 执行工具
- 返回执行结果

**Observer（观察模块）**：
- 分析执行结果
- 生成反馈信息
- 判断是否需要继续

**LoopController（循环控制）**：
- 控制循环次数
- 防止无限循环
- 提供停止条件

---

### 3.4 Tool 系统

#### Tool 接口
```go
type Tool interface {
    Name() string
    Description() string
    Execute(params map[string]interface{}) (interface{}, error)
}
```

#### 内置工具
- **calculator**：数学计算
- **datetime**：日期时间操作
- **http**：HTTP 请求
- **echo**：回显（测试用）
- **bash**：执行 Bash 命令
- **filesystem**：文件系统操作
- **grep**：文本搜索
- **curl**：HTTP 客户端
- **port**：端口检查

---

### 3.5 LLM 集成

#### LLM Client 接口
```go
type Client interface {
    Generate(prompt string) (string, error)
}
```

#### 支持的 LLM 提供商
- **Mock**：测试用 Mock 客户端
- **Ollama**：本地 LLM（qwen, llama 等）
- **OpenAI**：GPT-3.5, GPT-4 等
- **Anthropic**：Claude 系列

---

## 4. 数据流

### 4.1 简单任务执行流程
```
User → Engine.Execute(task)
     → Thinker.Think() → Thought
     → Actor.Act() → ExecutionResult
     → Observer.Observe() → Feedback
     → LoopController.Control() → Continue/Stop
     → Result
```

### 4.2 多 Agent 协作流程
```
User → Coordinator.ExecuteTask(task)
     → TaskDecomposer.Decompose() → SubTasks
     → For each SubTask:
         → Coordinator.selectAgent() → Agent
         → Executor.Execute(Agent.SystemPrompt + SubTask)
     → Merge Results → Final Result
```

### 4.3 Skill 使用流程
```
Task → SkillManager.SelectSkill() → Skill
     → Load Skill.Instructions
     → Inject into Prompt
     → Engine.Execute(Prompt with Instructions)
     → SkillManager.RecordExecution()
     → Update Statistics
```

---

## 5. 扩展点

### 5.1 TaskDecomposer
**用途**：集成外部任务拆解系统（如 AgenticRAG）

**接口**：
```go
type TaskDecomposer interface {
    Decompose(task string, context interface{}) ([]SubTask, error)
}
```

**示例**：
```go
// 集成 AgenticRAG
type AgenticRAGDecomposer struct {
    ragClient RAGClient
}

func (d *AgenticRAGDecomposer) Decompose(task string, ctx interface{}) ([]SubTask, error) {
    // 使用 RAG 检索相关知识
    docs := d.ragClient.Retrieve(task)

    // 基于知识拆解任务
    subTasks := analyzeAndDecompose(task, docs)

    return subTasks, nil
}
```

### 5.2 RAG 集成
**用途**：集成检索增强生成系统

**接口**：
```go
type Retriever interface {
    Retrieve(query string, topK int) ([]Document, error)
}
```

### 5.3 自定义 Tool
**用途**：扩展工具能力

**示例**：
```go
type MyTool struct{}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "My custom tool description"
}

func (t *MyTool) Execute(params map[string]interface{}) (interface{}, error) {
    // 实现工具逻辑
    return result, nil
}

// 注册
engine.RegisterTool(&MyTool{})
```

---

## 6. 性能优化

### 6.1 缓存系统
- **内存缓存**：基于 LRU 的内存缓存
- **缓存键**：基于任务和工具描述生成 SHA256 哈希
- **TTL 支持**：可配置缓存过期时间
- **性能提升**：缓存命中可提升 ~900,000x

### 6.2 并发处理
- **Go 协程**：利用 Go 的并发特性
- **并行工具调用**：支持多个工具并行执行
- **异步 Agent 通信**：Agent 之间异步通信

### 6.3 错误处理与重试
- **自动重试**：LLM 和工具执行失败自动重试
- **优雅降级**：LLM 不可用时使用简化模式
- **错误恢复**：利用缓存处理 LLM 故障

---

## 7. 依赖关系

### 7.1 依赖方向
```
Application Layer (Coordinator, SkillManager)
    ↓
Engine Layer (Engine, Core Modules)
    ↓
Infrastructure Layer (LLM, Tool, Cache)
```

**原则**：
- 依赖方向单一：从外向内
- 内层不依赖外层
- 通过接口解耦

### 7.2 关键依赖
```
Coordinator → Executor (interface)
Engine → LLM Client (interface)
Engine → Tool Manager
Engine → Cache (interface)
Thinker → Prompt Manager
Thinker → Memory Manager
```

---

## 8. 配置选项

### 8.1 Engine 配置
```go
engine.New(
    engine.WithLLMClient(llmClient),      // 设置 LLM 客户端
    engine.WithCache(cache),              // 启用缓存
    engine.WithMaxIterations(10),         // 最大迭代次数
    engine.WithThinker(customThinker),    // 自定义 Thinker
    engine.WithActor(customActor),        // 自定义 Actor
    engine.WithObserver(customObserver),  // 自定义 Observer
)
```

### 8.2 LLM 客户端配置
```go
// Ollama
ollama.NewOllamaClient(
    ollama.WithModel("qwen3:0.6b"),
    ollama.WithTemperature(0.7),
    ollama.WithBaseURL("http://localhost:11434"),
)

// OpenAI
openai.NewOpenAIClient(apiKey, "gpt-4")

// Anthropic
anthropic.NewAnthropicClient(apiKey, "claude-3-opus")
```

### 8.3 缓存配置
```go
cache.NewMemoryCache(
    cache.WithMaxSize(100),
    cache.WithDefaultTTL(1 * time.Hour),
)
```

---

## 9. 测试策略

### 9.1 单元测试
- 每个模块独立测试
- 使用 Mock LLM 客户端
- 覆盖核心逻辑

### 9.2 集成测试
- 测试模块间协作
- 使用真实 LLM（Ollama）
- 验证完整流程

### 9.3 示例程序
- `examples/simple/`：基础示例
- `examples/calculator/`：计算器示例
- `examples/ollama/`：Ollama 集成示例
- `examples/with_cache/`：缓存示例

---

## 10. 未来规划

详见 [ROADMAP.md](./ROADMAP.md)

---

## 附录

### A. 术语表
- **ReAct**：Reasoning + Acting，推理与行动结合的 AI 模式
- **Agent**：智能体，由 Name + System Prompt 组成
- **Skill**：技能，一套做事的方案（经验数据）
- **Tool**：工具，可执行的操作单元
- **Executor**：执行器，负责执行任务的接口
- **Coordinator**：协调器，负责 Agent 选择和任务分配

### B. 参考资料
- [ReAct Paper](https://arxiv.org/abs/2210.03629)
- [Agent Skills 规范](https://agentskills.io)
- [Ollama](https://ollama.com/)
