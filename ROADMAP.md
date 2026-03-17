# GoReAct 路线图

本文档包含未来规划和暂未实现但有价值的功能设计。

---

## 已完成功能

### ✅ Phase 1: MVP（已完成）
- [x] 核心 ReAct 循环（Thinker → Actor → Observer → LoopController）
- [x] 工具系统（Tool 接口 + ToolManager）
- [x] Mock LLM 客户端（用于测试）
- [x] 基础上下文管理

### ✅ Phase 2: 真实 LLM 集成（已完成）
- [x] Ollama 集成
- [x] OpenAI 集成
- [x] Anthropic 集成
- [x] 智能缓存系统（内存缓存 + TTL）
- [x] 增强的内置工具（HTTP, DateTime, Calculator, Echo, Bash, Filesystem, Grep 等）

### ✅ Phase 2.1: 可靠性（已完成）
- [x] 错误处理和重试机制
- [x] 优雅降级（LLM 不可用时的简化模式）
- [x] 增强的错误消息
- [x] 缓存错误恢复

### ✅ Phase 3: 高级功能（已完成）
- [x] Agent 系统（极简化设计：Agent = Name + SystemPrompt）
- [x] Skill 系统（基于 Agent Skills 规范）
- [x] Skill 评估和进化机制
- [x] 任务拆解接口（TaskDecomposer）
- [x] RAG 扩展点
- [x] 指标收集系统
- [x] 多 LLM 提供商支持

### ✅ Phase 4: 架构清理（已完成）
- [x] 移除冗余模块（task, token, config）
- [x] 清理过时接口（types.Agent, types.AgentManager）
- [x] 依赖注入模式（Agent 不依赖 Engine）
- [x] 单向依赖（整洁架构）

---

## 进行中

### 🔄 Phase 5: 文档和示例（进行中）
- [x] 更新 ARCHITECTURE.md 反映当前架构
- [ ] 创建完整的 Agent + Skill 示例
- [ ] 创建 TaskDecomposer 集成示例
- [ ] 添加 API 文档
- [ ] 添加最佳实践指南

---

## 未来规划

### 📋 Phase 6: Skill 系统完善

#### 6.1 Skill 加载器
**目标**：从文件系统加载 SKILL.md

**设计**：
```go
type SkillLoader interface {
    LoadFromDirectory(path string) (*Skill, error)
    LoadFromFile(path string) (*Skill, error)
    ValidateSkill(skill *Skill) error
}
```

**功能**：
- 解析 SKILL.md 的 YAML frontmatter
- 加载 Markdown 指令
- 加载 scripts/、references/、assets/ 目录
- 验证 Skill 格式（基于 Agent Skills 规范）

#### 6.2 Skill 进化调度器
**目标**：定期执行技能评估和进化

**设计**：
```go
type SkillEvolutionScheduler struct {
    manager  *SkillManager
    interval time.Duration
}

func (s *SkillEvolutionScheduler) Start() {
    ticker := time.NewTicker(s.interval)
    go func() {
        for range ticker.C {
            s.manager.EvolveSkills()
        }
    }()
}
```

**功能**：
- 定期评估所有 Skill
- 自动归档劣质 Skill
- 推广优秀 Skill
- 生成进化报告

#### 6.3 Skill 推荐系统
**目标**：根据任务智能推荐 Skill

**设计**：
```go
type SkillRecommender interface {
    Recommend(task string, topK int) ([]*Skill, error)
    ExplainRecommendation(skill *Skill, task string) string
}
```

**功能**：
- 基于任务关键词匹配
- 基于历史执行成功率
- 基于 Skill 综合评分
- 提供推荐理由

---

### 📋 Phase 7: Agent 协作增强

#### 7.1 Agent 通信协议
**目标**：规范化 Agent 之间的消息传递

**设计**：
```go
type Message struct {
    ID        string                 // 消息 ID
    From      string                 // 发送者 Agent 名称
    To        string                 // 接收者 Agent 名称
    Type      string                 // 消息类型（request, response, broadcast）
    Content   string                 // 消息内容
    Metadata  map[string]interface{} // 元数据
    Timestamp time.Time              // 时间戳
}

type MessageBus interface {
    Send(msg *Message) error
    Subscribe(agentName string, handler MessageHandler) error
    Broadcast(msg *Message) error
}
```

**功能**：
- 点对点消息传递
- 广播消息
- 消息订阅机制
- 消息历史记录

#### 7.2 Agent 状态管理
**目标**：跟踪 Agent 的运行状态

**设计**：
```go
type AgentState string

const (
    StateIdle    AgentState = "idle"
    StateBusy    AgentState = "busy"
    StateError   AgentState = "error"
    StateOffline AgentState = "offline"
)

type AgentStatus struct {
    Name          string
    State         AgentState
    CurrentTask   string
    TasksCompleted int
    LastActive    time.Time
}
```

**功能**：
- 实时状态监控
- 负载均衡（选择空闲 Agent）
- 故障检测和恢复
- 性能统计

#### 7.3 协作模式
**目标**：支持多种 Agent 协作模式

**模式**：
1. **层次化协作**：主 Agent 分配任务给子 Agent
2. **对等协作**：多个 Agent 平等协作
3. **竞争模式**：多个 Agent 竞争执行任务，选择最优结果
4. **流水线模式**：任务按顺序经过多个 Agent

**设计**：
```go
type CollaborationMode interface {
    Execute(agents []*Agent, task string) (*Result, error)
}

type HierarchicalMode struct{}  // 层次化
type PeerToPeerMode struct{}    // 对等
type CompetitiveMode struct{}   // 竞争
type PipelineMode struct{}      // 流水线
```

---

### 📋 Phase 8: TaskDecomposer 集成

#### 8.1 AgenticRAG 集成
**目标**：使用 RAG 系统辅助任务拆解

**设计**：
```go
type AgenticRAGDecomposer struct {
    ragClient    RAGClient
    llmClient    llm.Client
    maxSubTasks  int
}

func (d *AgenticRAGDecomposer) Decompose(task string, ctx interface{}) ([]SubTask, error) {
    // 1. 使用 RAG 检索相关知识
    docs := d.ragClient.Retrieve(task, 5)

    // 2. 构建增强的 prompt
    prompt := buildPromptWithDocs(task, docs)

    // 3. 使用 LLM 拆解任务
    messages := []llm.Message{llm.NewUserMessage(prompt)}
    response, err := d.llmClient.Chat(ctx.Context(), messages)
    if err != nil {
        return nil, err
    }

    // 4. 解析子任务
    subTasks := parseSubTasks(response.Content)

    return subTasks, nil
}
```

**功能**：
- 基于知识库的任务拆解
- 考虑历史任务经验
- 自动识别任务依赖
- 优化子任务粒度

#### 8.2 基于 LLM 的任务拆解
**目标**：使用 LLM 智能拆解复杂任务

**设计**：
```go
type LLMTaskDecomposer struct {
    llmClient llm.Client
    template  string
}
```

**Prompt 模板**：
```
You are a task decomposition expert. Break down the following complex task into smaller, manageable sub-tasks.

Complex Task: {{task}}

Requirements:
1. Each sub-task should be specific and actionable
2. Identify dependencies between sub-tasks
3. Estimate the complexity of each sub-task
4. Suggest the best order of execution

Output format (JSON):
{
  "subtasks": [
    {
      "id": "task-1",
      "description": "...",
      "dependencies": [],
      "complexity": "low|medium|high"
    }
  ]
}
```

---

### 📋 Phase 9: 分布式支持

#### 9.1 分布式缓存
**目标**：支持 Redis 等分布式缓存

**设计**：
```go
type RedisCache struct {
    client *redis.Client
    ttl    time.Duration
}

func (c *RedisCache) Get(key string) (interface{}, bool) {
    val, err := c.client.Get(context.Background(), key).Result()
    if err != nil {
        return nil, false
    }
    return val, true
}
```

**功能**：
- 跨实例缓存共享
- 缓存失效通知
- 缓存预热
- 缓存统计

#### 9.2 分布式 Agent 调度
**目标**：支持跨机器的 Agent 部署

**设计**：
```go
type DistributedCoordinator struct {
    localAgents  []*Agent
    remoteAgents map[string]*RemoteAgent
    registry     ServiceRegistry
}

type RemoteAgent struct {
    Name     string
    Endpoint string
    Client   *http.Client
}
```

**功能**：
- 服务注册与发现
- 远程 Agent 调用
- 负载均衡
- 故障转移

#### 9.3 任务队列
**目标**：支持异步任务处理

**设计**：
```go
type TaskQueue interface {
    Enqueue(task *Task) error
    Dequeue() (*Task, error)
    GetStatus(taskID string) (*TaskStatus, error)
}

type RedisTaskQueue struct {
    client *redis.Client
}
```

**功能**：
- 异步任务提交
- 任务优先级队列
- 任务状态跟踪
- 失败重试

---

### 📋 Phase 10: 高级功能

#### 10.1 Workflow 系统
**目标**：支持复杂的工作流编排

**设计**：
```go
type Workflow struct {
    ID    string
    Name  string
    Steps []WorkflowStep
}

type WorkflowStep struct {
    ID           string
    Type         string // agent, tool, condition, loop
    Config       map[string]interface{}
    Dependencies []string
}

type WorkflowEngine interface {
    Execute(workflow *Workflow) (*Result, error)
    Pause(workflowID string) error
    Resume(workflowID string) error
}
```

**功能**：
- 可视化工作流设计
- 条件分支
- 循环执行
- 工作流暂停/恢复

#### 10.2 人机协作
**目标**：支持人类参与决策

**设计**：
```go
type HumanInTheLoop interface {
    RequestApproval(task string, options []string) (string, error)
    RequestInput(prompt string) (string, error)
    Notify(message string) error
}
```

**功能**：
- 关键决策人工审批
- 人工输入补充
- 异常情况人工介入
- 审计日志

#### 10.3 多模态支持
**目标**：支持图像、音频等多模态输入

**设计**：
```go
type MultimodalInput struct {
    Text   string
    Images []Image
    Audio  []Audio
}

type MultimodalTool interface {
    Tool
    SupportedModalities() []string
}
```

**功能**：
- 图像识别工具
- 语音转文字工具
- 文字转语音工具
- 多模态 LLM 集成

---

### 📋 Phase 11: 企业级功能

#### 11.1 权限管理
**目标**：支持细粒度的权限控制

**设计**：
```go
type Permission struct {
    Resource string // agent, tool, skill
    Action   string // read, write, execute
}

type Role struct {
    Name        string
    Permissions []Permission
}

type AuthManager interface {
    CheckPermission(user string, resource string, action string) bool
    AssignRole(user string, role string) error
}
```

#### 11.2 审计日志
**目标**：记录所有操作用于审计

**设计**：
```go
type AuditLog struct {
    ID        string
    User      string
    Action    string
    Resource  string
    Timestamp time.Time
    Result    string
    Details   map[string]interface{}
}

type AuditLogger interface {
    Log(log *AuditLog) error
    Query(filter AuditFilter) ([]*AuditLog, error)
}
```

#### 11.3 配置管理系统
**目标**：集中管理配置

**设计**：
```go
type ConfigManager interface {
    Get(key string) (interface{}, error)
    Set(key string, value interface{}) error
    Watch(key string, callback func(value interface{})) error
    LoadFromFile(path string) error
    LoadFromEnv() error
}

type DynamicConfig struct {
    manager ConfigManager
    cache   map[string]interface{}
}
```

**功能**：
- 配置热更新
- 配置版本管理
- 配置验证
- 环境隔离

---

## 性能优化计划

### 🚀 优化目标

| 指标 | 当前 | 目标 | 优化方案 |
|------|------|------|----------|
| 简单任务响应时间 | ~100ms | < 50ms | 并发优化 + 缓存预热 |
| 复杂任务响应时间 | ~5s | < 2s | 智能路径选择 + 并行执行 |
| 并发处理能力 | ~1,000 QPS | > 10,000 QPS | Go 协程池 + 连接池 |
| 内存占用 | ~500MB | < 300MB | 内存池 + 对象复用 |
| 缓存命中率 | ~60% | > 90% | 智能缓存策略 + 预测性缓存 |

### 优化方案

#### 1. 并发优化
- 使用 Go 协程池管理并发
- 工具并行执行
- 批量任务处理

#### 2. 缓存优化
- 多层缓存（L1: 内存, L2: Redis）
- 预测性缓存（基于历史模式）
- 缓存预热

#### 3. 内存优化
- 对象池复用
- 上下文压缩
- 及时释放资源

#### 4. 网络优化
- HTTP/2 支持
- 连接池
- 请求合并

---

## 技术债务

### 需要重构的部分

1. **Model 系统**
   - 当前与 LLM Client 重叠
   - 建议：合并到 LLM 包或明确职责

2. **Memory 系统**
   - 当前与 Context 重叠
   - 建议：明确 Memory 用于长期存储，Context 用于短期上下文

3. **Prompt 系统**
   - 当前功能简单
   - 建议：增强模板功能或简化为工具函数

4. **Metrics 系统**
   - 当前使用较少
   - 建议：集成 Prometheus 标准指标

---

## 社区贡献指南

### 欢迎贡献的领域

1. **新的 Tool 实现**
   - 数据库工具（MySQL, PostgreSQL, MongoDB）
   - API 工具（GraphQL, gRPC）
   - 云服务工具（AWS, GCP, Azure）

2. **新的 LLM 集成**
   - Gemini
   - Mistral
   - 本地模型（llama.cpp）

3. **示例和教程**
   - 实际应用场景示例
   - 最佳实践文档
   - 视频教程

4. **测试和文档**
   - 单元测试
   - 集成测试
   - API 文档

---

## 参考资料

- [ReAct Paper](https://arxiv.org/abs/2210.03629)
- [Agent Skills 规范](https://agentskills.io)
- [整洁架构](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
