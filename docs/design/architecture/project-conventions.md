# 项目结构与开发约定

本文档定义了 GoReAct 项目的全局项目结构和开发约定，适用于所有模块的开发工作。

## 0. 规范依据与强制要求

本约定为所有代码编写、架构设计、文档生成及版本控制等工作均需严格遵循以下规范：

### 0.1 核心规范来源

| 规范来源                    | 说明                                                | 链接                                                 |
| --------------------------- | --------------------------------------------------- | ---------------------------------------------------- |
| **Effective Go**            | Go 语言官方编写指南，涵盖代码风格、命名、控制结构等 | https://go.dev/doc/effective_go                      |
| **Go Code Review Comments** | Go 官方代码评审规范，涵盖常见问题和最佳实践         | https://github.com/golang/go/wiki/CodeReviewComments |
| **Go 命名规范**             | 官方包、变量、函数命名约定                          | https://go.dev/blog/package-names                    |
| **Go 内存模型**             | 并发编程的内存可见性保证                            | https://go.dev/ref/mem                               |
| **Go 模块指南**             | 依赖管理和版本控制最佳实践                          | https://go.dev/blog/using-go-modules                 |

### 0.2 强制要求

所有开发人员必须确保项目中的每一行代码、每个模块设计及文档内容均符合上述标准，以保证代码质量、可维护性及项目的长期健康发展。具体要求如下：

**代码风格一致性**：
- 使用 `gofmt` 和 `goimports` 格式化所有代码
- 遵循 Go 官方缩进、括号、注释风格
- 行长度不超过 120 字符（非强制，但建议）

**错误处理机制**：
- 禁止忽略错误，所有错误必须显式处理
- 使用 `errors.Is()` 和 `errors.As()` 进行错误判断
- 错误信息不应大写开头，不应以标点结尾
- 使用 `fmt.Errorf("context: %w", err)` 包装错误保留调用栈

**包管理策略**：
- 使用 Go Modules 进行依赖管理
- 遵循语义化版本 (SemVer) 规范
- 内部实现使用 `internal` 包隔离
- 避免循环依赖

**并发编程模式**：
- 所有导出类型必须线程安全
- 使用 `context.Context` 进行取消和超时控制
- 使用 `sync` 包的同步原语，避免 `sync.Mutex` 值拷贝
- goroutine 泄漏检测

**性能优化标准**：
- 避免不必要的内存分配
- 使用 `sync.Pool` 复用对象
- 使用 `pprof` 进行性能分析
- 基准测试覆盖关键路径

**测试覆盖率要求**：
- 单元测试覆盖率不低于 80%
- 使用表驱动测试 (table-driven tests)
- 使用 `t.Helper()` 标记测试辅助函数
- 使用 `t.Parallel()` 并行执行独立测试

### 0.3 工具配置

项目应配置以下工具以确保规范执行：

```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - ineffassign
    - typecheck
    - gosimple
    - goconst
    - gocyclo
    - dupl
    - misspell

linters-settings:
  gocyclo:
    min-complexity: 15
  goconst:
    min-len: 3
    min-occurrences: 3
  errcheck:
    check-type-assertions: true
    check-blank: true

run:
  timeout: 5m
  skip-dirs:
    - vendor
    - test/fixtures
```

```yaml
# .github/workflows/lint.yml
name: Lint
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
```

## 1. 项目目录结构

```
goreact/
├── cmd/                          # 应用程序入口
│   ├── server/                   # API 服务入口
│   │   └── main.go
│   └── cli/                      # CLI 工具入口
│       └── main.go
│
├── pkg/                          # 可导入的公共包
│   ├── agent/                    # Agent 模块
│   ├── reactor/                  # Reactor 模块
│   ├── orchestration/           # 编排模块
│   ├── memory/                  # 记忆模块
│   ├── skill/                   # 技能模块
│   ├── tool/                    # 工具模块
│   ├── prompt/                  # Prompt 构建模块
│   ├── observer/                # 可观测性模块
│   └── common/                  # 公共工具
│
├── internal/                    # 内部实现（不对外暴露）
│   ├── dag/                     # DAG 算法
│   ├── topo_sort/               # 拓扑排序
│   └── retry/                   # 重试逻辑
│
├── configs/                     # 配置文件
│   ├── default.yaml            # 默认配置
│   └── dev.yaml                # 开发配置
│
├── test/                        # 测试工具和数据
│   ├── fixtures/               # 测试数据
│   └── mocks/                  # Mock 对象
│
├── docs/                        # 文档
│   ├── concepts/               # 概念文档
│   ├── design/                 # 设计文档
│   └── guide/                  # 使用指南
│
├── go.mod
├── go.sum
└── Makefile
```

## 2. 模块目录结构约定

每个核心模块应遵循以下结构：

```
module/
├── module.go                    # 模块入口，导出主要接口
├── module_test.go              # 入口测试
├── interface.go                # 接口定义（可选）
├── {component}.go              # 组件实现
├── {component}_test.go         # 组件测试
├── types/                      # 类型定义
│   ├── types.go
│   ├── request.go
│   ├── response.go
│   └── errors.go
├── strategy/                   # 策略模式实现
│   ├── interface.go
│   └── strategy_*.go
├── storage/                   # 存储层
│   ├── interface.go
│   ├── memory.go
│   └── redis.go
└── internal/                  # 内部实现
    ├── algo.go
    └── utils.go
```

## 3. 文件命名规范

| 文件类型 | 命名规范                          | 示例                                |
| -------- | --------------------------------- | ----------------------------------- |
| 模块入口 | `module.go`                       | `orchestration/module.go`           |
| 接口定义 | `interface.go` 或 `{name}.go`     | `interface.go`, `reactor.go`        |
| 实现文件 | 小写名词                          | `planner.go`, `selector.go`         |
| 实现变体 | `{名称}_{变体}.go`                | `planner_llm.go`, `planner_rule.go` |
| 测试文件 | 原文件名 + `_test.go`             | `orchestrator_test.go`              |
| 类型定义 | `types.go` 或 `{domain}_types.go` | `types.go`, `plan_types.go`         |
| 内部实现 | 小写，可带下划线                  | `internal/dag.go`                   |

## 4. 接口定义规范

### 4.1 接口命名约定

```go
// 核心功能接口：使用 -er 后缀
type Reader interface { ... }
type Executor interface { ... }

// 策略接口：使用 Strategy 后缀
type DecompositionStrategy interface { ... }

// 工厂接口：使用 Factory 后缀
type AgentFactory interface { ... }

// 构建器接口：使用 Builder 后缀
type PromptBuilder interface { ... }
```

### 4.2 接口方法签名约定

```go
// 标准方法签名
type Service interface {
    // 1. 第一个参数必须是 context.Context
    // 2. 返回值必须是 (结果, error) 或仅 error
    // 3. 错误不使用 panic，必须显式返回
    
    Execute(ctx context.Context, req *Request) (*Response, error)
    Stream(ctx context.Context, req *Request) (<-chan Event, error)
}
```

## 5. 项目特定错误类型

> 注：通用错误处理规范见 [0.2 强制要求](#02-强制要求)，本节仅定义项目特定错误类型。

```go
// Error 项目统一错误结构，包含详细上下文
type Error struct {
    Code    string    // 错误码
    Message string    // 错误消息
    Module  string    // 模块名称
    Cause   error     // 原始错误
}

func (e *Error) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Module, e.Message, e.Cause)
}

func (e *Error) Unwrap() error {
    return e.Cause
}

// 错误码定义
const (
    // 通用错误码 (1xxx)
    CodeNotFound      = "E1001"  // 资源未找到
    CodeInvalidInput  = "E1002"  // 输入参数无效
    CodeTimeout       = "E1003"  // 操作超时
    CodeInternal      = "E1004"  // 内部错误
    CodeUnauthorized  = "E1005"  // 未授权访问
    CodeConflict      = "E1006"  // 资源冲突
    
    // Agent 模块错误码 (2xxx)
    CodeAgentNotFound    = "E2001"  // Agent 未找到
    CodeAgentFailed      = "E2002"  // Agent 执行失败
    CodeAgentPaused      = "E2003"  // Agent 已暂停
    CodeAgentStopped     = "E2004"  // Agent 已停止
    CodeAgentTimeout     = "E2005"  // Agent 执行超时
    
    // Reactor 模块错误码 (3xxx)
    CodePlanFailed       = "E3001"  // 规划失败
    CodeThinkFailed      = "E3002"  // 思考失败
    CodeActFailed        = "E3003"  // 行动失败
    CodeObserveFailed    = "E3004"  // 观察失败
    CodeReflectFailed    = "E3005"  // 反思失败
    CodeMaxStepsExceeded = "E3006"  // 超过最大步数
    CodeLoopDetected     = "E3007"  // 检测到循环
    CodeNoAction         = "E3008"  // 无可用行动
    
    // Memory 模块错误码 (4xxx)
    CodeMemoryLoadFailed = "E4001"  // Memory 加载失败
    CodeNodeNotFound     = "E4002"  // 节点未找到
    CodeNodeCreateFailed = "E4003"  // 节点创建失败
    CodeNodeUpdateFailed = "E4004"  // 节点更新失败
    CodeQueryFailed      = "E4005"  // 查询失败
    CodeIndexFailed      = "E4006"  // 索引失败
    
    // Tool 模块错误码 (5xxx)
    CodeToolNotFound     = "E5001"  // 工具未找到
    CodeToolNotWhitelisted = "E5002"  // 工具未在白名单
    CodeToolExecutionFailed = "E5003"  // 工具执行失败
    CodeToolTimeout      = "E5004"  // 工具执行超时
    CodeToolDenied       = "E5005"  // 工具调用被拒绝
    
    // Skill 模块错误码 (6xxx)
    CodeSkillNotFound    = "E6001"  // 技能未找到
    CodeSkillCompileFailed = "E6002"  // 技能编译失败
    CodeSkillExecuteFailed = "E6003"  // 技能执行失败
    
    // Orchestration 模块错误码 (7xxx)
    CodeOrchestrateFailed = "E7001"  // 编排失败
    CodeDecomposeFailed  = "E7002"  // 任务分解失败
    CodeSelectFailed     = "E7003"  // Agent 选择失败
    CodeCoordinateFailed = "E7004"  // 执行协调失败
    CodeAggregateFailed  = "E7005"  // 结果聚合失败
    CodeDependencyCycle  = "E7006"  // 依赖循环
    
    // LLM 模块错误码 (8xxx)
    CodeLLMRequestFailed = "E8001"  // LLM 请求失败
    CodeLLMResponseInvalid = "E8002"  // LLM 响应无效
    CodeLLMRateLimited   = "E8003"  // LLM 限流
    CodeLLMContextExceeded = "E8004"  // 上下文超限
)

// 错误码说明表
var ErrorDescriptions = map[string]string{
    // 通用错误码
    "E1001": "请求的资源不存在",
    "E1002": "输入参数不符合要求",
    "E1003": "操作在规定时间内未完成",
    "E1004": "系统内部错误，请联系管理员",
    "E1005": "无权限执行此操作",
    "E1006": "资源状态冲突，请重试",
    
    // Agent 模块
    "E2001": "指定的 Agent 不存在",
    "E2002": "Agent 执行过程中发生错误",
    "E2003": "Agent 处于暂停状态，需恢复后继续",
    "E2004": "Agent 已停止，无法继续执行",
    "E2005": "Agent 执行时间超过限制",
    
    // Reactor 模块
    "E3001": "无法生成执行计划",
    "E3002": "推理过程失败",
    "E3003": "行动执行失败",
    "E3004": "观察结果处理失败",
    "E3005": "反思过程失败",
    "E3006": "执行步数超过最大限制",
    "E3007": "检测到重复行动循环",
    "E3008": "没有可执行的下一步行动",
    
    // Memory 模块
    "E4001": "Memory 初始化加载失败",
    "E4002": "请求的节点不存在",
    "E4003": "创建节点失败",
    "E4004": "更新节点失败",
    "E4005": "查询操作失败",
    "E4006": "索引操作失败",
    
    // Tool 模块
    "E5001": "请求的工具不存在",
    "E5002": "工具不在授权白名单中",
    "E5003": "工具执行过程中发生错误",
    "E5004": "工具执行时间超过限制",
    "E5005": "工具调用被安全策略拒绝",
    
    // Skill 模块
    "E6001": "请求的技能不存在",
    "E6002": "技能编译失败",
    "E6003": "技能执行失败",
    
    // Orchestration 模块
    "E7001": "多 Agent 编排执行失败",
    "E7002": "任务分解失败",
    "E7003": "无法选择合适的 Agent",
    "E7004": "执行协调失败",
    "E7005": "结果聚合失败",
    "E7006": "任务依赖关系存在循环",
    
    // LLM 模块
    "E8001": "LLM API 请求失败",
    "E8002": "LLM 响应格式无效",
    "E8003": "LLM API 请求被限流",
    "E8004": "输入上下文超过模型限制",
}
```

## 6. 日志规范

### 6.1 日志级别使用

| 级别  | 使用场景     | 示例                                   |
| ----- | ------------ | -------------------------------------- |
| DEBUG | 开发调试信息 | "entering function", "variable value"  |
| INFO  | 正常业务流程 | "request received", "task completed"   |
| WARN  | 可恢复的异常 | "retry attempt", "fallback to default" |
| ERROR | 业务错误     | "request failed", "connection error"   |

### 6.2 结构化日志格式

```go
import "log/slog"

logger := slog.With(
    "module", "orchestration",
    "session_id", sessionID,
)

logger.Info("orchestration started",
    "task", task.Name,
    "agent_count", len(agents),
)
```

### 6.3 敏感信息处理

```go
// 禁止记录敏感信息
logger.Info("user login", "user_id", userID)      // ✅ 安全
logger.Info("user login", "password", password)  // ❌ 危险

// 使用脱敏处理
logger.Info("user data",
    "email", maskEmail(user.Email),   // ***@example.com
    "phone", maskPhone(user.Phone),   // 138****1234
)
```

## 7. 测试规范

> 注：通用测试规范见 [0.2 强制要求](#02-强制要求)，本节仅定义项目特定测试约定。

### 7.1 测试文件组织

```go
// 文件名: {module}_test.go
package orchestration

// 测试函数命名: Test{MethodName}_{Scenario}
func TestOrchestrate_Success(t *testing.T) { ... }
func TestOrchestrate_Timeout(t *testing.T) { ... }
```

### 7.2 Mock 对象模式

```go
// Mock 实现
type MockPlanner struct {
    PlanFunc func(ctx context.Context, task *Task) (*Plan, error)
}

func (m *MockPlanner) Plan(ctx context.Context, task *Task) (*Plan, error) {
    if m.PlanFunc != nil {
        return m.PlanFunc(ctx, task)
    }
    return &Plan{}, nil
}
```

## 8. 代码注释规范

> 注：通用注释规范见 Effective Go 和 Go Code Review Comments。

### 8.1 包注释模板

```go
// Package orchestration 提供多 Agent 编排功能。
//
// 主要功能:
//   - 任务分解与规划
//   - Agent 选择与协调
//   - 执行状态管理
//
// 使用示例:
//
//   orch := orchestration.New(config)
//   result, err := orch.Orchestrate(ctx, task)
package orchestration
```

### 8.2 函数注释模板

```go
// FunctionName 执行指定操作。
//
// 参数:
//   - ctx: 上下文
//   - param: 参数说明
//
// 返回:
//   - result: 返回值说明
//   - error: 错误说明
func FunctionName(ctx context.Context, param Type) (result Result, err error) {
    // ...
}
```

## 9. 并发安全规范

> 注：通用并发规范见 [0.2 强制要求](#02-强制要求) 和 Go 内存模型。

### 9.1 线程安全模式

```go
type Service struct {
    mu    sync.RWMutex
    state State
}

// 读操作使用 RLock
func (s *Service) GetState() State {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.state.Clone()  // 返回副本
}

// 写操作使用 Lock
func (s *Service) SetState(state State) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.state = state
}
```

### 9.2 Context 传递

```go
// 所有可能阻塞的操作必须支持取消
func (s *Service) Execute(ctx context.Context, req *Request) (*Response, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // 继续执行
    }
    // ...
}
```

## 10. 配置管理规范

### 10.1 配置结构

```go
type Config struct {
    // 必需字段（无默认值）
    Name string `yaml:"name" json:"name"`
    
    // 带默认值的字段
    Timeout time.Duration `yaml:"timeout" json:"timeout" default:"5m"`
    MaxRetries int       `yaml:"max_retries" json:"maxRetries" default:"3"`
    
    // 可选字段
    Optional *string `yaml:"optional" json:"optional,omitempty"`
    
    // 嵌套配置
    Retry RetryConfig `yaml:"retry" json:"retry"`
}
```

### 10.2 配置加载与验证

```go
// 默认配置
func DefaultConfig() *Config {
    return &Config{
        Timeout:    5 * time.Minute,
        MaxRetries: 3,
    }
}

// 配置验证
func (c *Config) Validate() error {
    if c.Timeout <= 0 {
        return errors.New("timeout must be positive")
    }
    return nil
}
```

## 11. 依赖管理规范

> 注：通用依赖管理规范见 [0.2 强制要求](#02-强制要求) 和 Go 模块指南。

### 11.1 项目模块路径

```
github.com/goreact/goreact
├── github.com/goreact/goreact/pkg/agent
├── github.com/goreact/goreact/pkg/reactor
├── github.com/goreact/goreact/pkg/orchestration
├── github.com/goreact/goreact/pkg/memory
└── ...
```

### 11.2 依赖原则

- **最小依赖**: 只引入必需的依赖
- **版本管理**: 使用语义化版本
- **避免循环**: 模块间不能有循环依赖
- **internal 隔离**: 内部实现使用 `internal` 包隔离

## 12. 文档规范

### 12.1 README 要求

每个模块根目录应包含 README.md：

```markdown
# 模块名称

## 概述
模块功能的简要描述。

## 主要功能
- 功能1
- 功能2

## 使用示例
\`\`\`go
// 示例代码
\`\`\`

## 配置说明

| 参数  | 类型   | 默认值 | 说明     |
| ----- | ------ | ------ | -------- |
| param | string | ""     | 参数说明 |
```

### 12.2 API 文档

```bash
# 生成文档
go doc ./pkg/module

# 启动文档服务器
godoc -http=:6060
```

## 13. 版本与发布规范

### 13.1 版本号

使用语义化版本 (SemVer)：

```
MAJOR.MINOR.PATCH
  │     │     │
  │     │     └── 补丁版本：bug 修复
  │     └──────── 次版本：新功能（向后兼容）
  └───────────── 主版本：破坏性变更
```

### 13.2 变更日志

使用 Conventional Commits 格式：

```
feat: 添加新功能
fix: 修复问题
docs: 文档变更
style: 代码格式调整
refactor: 代码重构
test: 测试变更
chore: 构建过程或辅助工具变更
```
