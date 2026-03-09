# GoReAct 工具箱开发计划

## 概述

本文档是 GoReAct 框架工具箱的**完整开发计划**，涵盖 Thinker、Actor、Observer、LoopController 四个环节的所有工具。

---

## 📋 开发原则

1. **痛点驱动** - 每个工具都解决真实问题
2. **场景优先** - 先写使用指南，再写代码
3. **渐进实现** - 分 Phase 1/2/3 逐步实现
4. **文档先行** - 设计文档 → 使用指南 → 代码实现
5. **可选使用** - 工具是可选的，不是强制的

---

## 🎯 四大环节工具箱

### 1. Thinker Toolkit - Prompt 构建工具箱
**核心痛点：** Prompt 构建复杂、Token 超限、调试困难

### 2. Actor Toolkit - 工具执行工具箱
**核心痛点：** 参数验证重复、错误处理缺失、测试困难

### 3. Observer Toolkit - 反馈生成工具箱
**核心痛点：** 反馈千篇一律、无法检测循环、缺少智能分析

### 4. LoopController Toolkit - 循环控制工具箱
**核心痛点：** 控制策略简单、无法检测停滞、缺少成本控制

---

## 📦 Phase 1: 核心功能（立即实现）

### 1.1 Thinker Toolkit - Phase 1 ✅ 已完成

**已实现：**
- [x] FluentPromptBuilder - 流式 API 构建器
- [x] Token Counter (Simple, Universal, Cached)
- [x] Tool Formatter (Simple, JSON Schema, Markdown, Compact)
- [x] Compression Strategy (Truncate, SlidingWindow, Priority, Hybrid)
- [x] Prompt Debugger - 调试和追踪工具

**文档：**
- [x] PROMPT_TOOLKIT_DESIGN.md - 设计文档
- [x] PROMPT_TOOLKIT_USAGE.md - 使用指南
- [x] PROMPT_TOOLKIT_SUMMARY.md - 实现总结
- [x] examples/prompt_toolkit/ - 示例代码

**代码位置：**
```
pkg/prompt/
├── types.go
├── counter/
│   └── counter.go
├── formatter/
│   └── formatter.go
├── builder/
│   └── builder.go
├── compression/
│   └── compression.go
└── debug/
    └── debug.go
```

---

### 1.2 Actor Toolkit - Phase 1 🔜 待实现

**核心组件：**

#### 1.2.1 Schema-based Tool
```go
// 位置：pkg/actor/schema/
// 文件：
// - schema.go - Schema 定义
// - validator.go - 参数验证器
// - converter.go - 类型转换器
// - tool.go - Schema-based Tool 实现

// 核心功能：
type Schema struct {
    Parameters Object
}

type Object struct {
    Properties map[string]Property
}

type Property struct {
    Type        PropertyType  // String, Number, Boolean, Array, Object
    Description string
    Enum        []string
    Required    bool
    Default     interface{}
}

// ValidatedParams - 类型安全的参数访问
type ValidatedParams struct {
    raw    map[string]interface{}
    schema Schema
}

func (p *ValidatedParams) GetString(key string) string
func (p *ValidatedParams) GetInt(key string) int
func (p *ValidatedParams) GetFloat64(key string) float64
func (p *ValidatedParams) GetBool(key string) bool
```

**测试用例：**
- 参数验证（必需参数、类型检查、Enum 验证）
- 类型转换（"123" → 123, "true" → true）
- 错误消息（LLM 友好）

---

#### 1.2.2 Execution Wrappers
```go
// 位置：pkg/actor/wrapper/
// 文件：
// - timeout.go - 超时包装器
// - retry.go - 重试包装器
// - wrapper.go - 包装器组合

// TimeoutWrapper
type TimeoutWrapper struct {
    timeout time.Duration
}

func (w *TimeoutWrapper) Wrap(tool Tool) Tool

// RetryWrapper
type RetryWrapper struct {
    maxAttempts int
    interval    time.Duration
    retryIf     func(error) bool
}

func (w *RetryWrapper) Wrap(tool Tool) Tool

// 组合包装器
func WrapTool(tool Tool, wrappers ...Wrapper) Tool
```

**测试用例：**
- 超时控制（命令超时自动取消）
- 重试机制（网络错误自动重试）
- 包装器组合（超时 + 重试）

---

#### 1.2.3 Result Formatter
```go
// 位置：pkg/actor/formatter/
// 文件：
// - result_formatter.go - 结果格式化器
// - error_formatter.go - 错误格式化器

// ResultFormatter
type ResultFormatter interface {
    Format(result interface{}) (string, error)
}

// 内置格式化器
type TruncateFormatter struct {
    maxLength int
}

type SummaryFormatter struct {
    maxLines int
}

type JSONFormatter struct {
    indent    bool
    maxLength int
}

// ErrorFormatter
type ErrorFormatter struct{}

func (f *ErrorFormatter) Format(err error, toolName string, params map[string]interface{}) string
```

**测试用例：**
- 截断长输出（10MB → 1KB）
- 提取摘要（文件前 10 行）
- 格式化错误消息（技术错误 → LLM 友好）

---

#### 1.2.4 Execution Tracer
```go
// 位置：pkg/actor/debug/
// 文件：
// - tracer.go - 执行追踪器
// - profiler.go - 性能分析器

// ExecutionTracer
type ExecutionTracer struct {
    enabled bool
    logger  Logger
}

type ExecutionTrace struct {
    ToolName   string
    Parameters map[string]interface{}
    StartTime  time.Time
    EndTime    time.Time
    Duration   time.Duration
    Success    bool
    Output     interface{}
    Error      error
}

// PerformanceProfiler
type PerformanceProfiler struct {
    stats map[string]*ToolStats
}

type ToolStats struct {
    TotalCalls      int
    SuccessCalls    int
    FailedCalls     int
    AverageDuration time.Duration
}
```

**测试用例：**
- 追踪工具执行（记录输入输出）
- 性能统计（平均耗时、成功率）

---

### 1.3 Observer Toolkit - Phase 1 🔜 待实现

**核心组件：**

#### 1.3.1 Smart Feedback Generator
```go
// 位置：pkg/observer/feedback/
// 文件：
// - generator.go - 反馈生成器
// - tool_generators.go - 工具特定的生成器

// FeedbackGenerator
type FeedbackGenerator interface {
    Generate(result *ExecutionResult, context *Context) string
}

// SmartFeedbackGenerator
type SmartFeedbackGenerator struct {
    generators map[string]FeedbackGenerator  // 按工具类型
    analyzer   ResultAnalyzer
}

// 工具特定的生成器
type HTTPFeedbackGenerator struct{}
type CalculatorFeedbackGenerator struct{}
type FilesystemFeedbackGenerator struct{}
```

**测试用例：**
- HTTP 404 → 友好反馈
- 计算错误 → 指出问题
- 连接失败 → 提供建议

---

#### 1.3.2 Result Validator
```go
// 位置：pkg/observer/validator/
// 文件：
// - validator.go - 结果验证器
// - rules.go - 验证规则

// ResultValidator
type ResultValidator interface {
    Validate(result *ExecutionResult, context *Context) ValidationResult
}

type ValidationResult struct {
    IsValid     bool
    Confidence  float64
    Issues      []string
    Suggestions []string
}

// 验证规则
type HTTPStatusRule struct{}
type DataFormatRule struct{}
type ExpectedValueRule struct{}
```

**测试用例：**
- HTTP 状态码验证
- 数据格式验证
- 预期值验证

---

#### 1.3.3 Loop Detector
```go
// 位置：pkg/observer/detector/
// 文件：
// - loop_detector.go - 循环检测器

// LoopDetector
type LoopDetector struct {
    history    []ActionRecord
    maxRepeats int
    windowSize int
}

type ActionRecord struct {
    ToolName   string
    Parameters map[string]interface{}
    Success    bool
    Timestamp  time.Time
}

type LoopPattern struct {
    Detected    bool
    Pattern     []ActionRecord
    RepeatCount int
    Suggestion  string
}
```

**测试用例：**
- 检测重复失败（3 次相同参数）
- 检测无行动（连续 5 次只思考）

---

### 1.4 LoopController Toolkit - Phase 1 🔜 待实现

**核心组件：**

#### 1.4.1 Composite Stop Condition
```go
// 位置：pkg/loopcontroller/condition/
// 文件：
// - condition.go - 停止条件接口
// - conditions.go - 内置条件

// StopCondition
type StopCondition interface {
    ShouldStop(state *LoopState, context *Context) (bool, string)
}

// 内置条件
type MaxIterationCondition struct {
    maxIterations int
}

type TimeoutCondition struct {
    timeout   time.Duration
    startTime time.Time
}

type StagnationCondition struct {
    maxStagnantIterations int
    detector              StagnationDetector
}

type QualityThresholdCondition struct {
    threshold float64
    evaluator QualityEvaluator
}
```

**测试用例：**
- 最大迭代次数
- 超时控制
- 停滞检测
- 质量阈值

---

#### 1.4.2 Stagnation Detector
```go
// 位置：pkg/loopcontroller/detector/
// 文件：
// - stagnation_detector.go

// StagnationDetector
type StagnationDetector struct {
    window    int
    threshold float64
}

type StagnationPattern struct {
    Detected    bool
    Type        string  // "no_action", "repeated_failure", "no_progress"
    Description string
    Suggestion  string
}
```

**测试用例：**
- 检测无行动
- 检测重复失败
- 检测无进展

---

#### 1.4.3 Quality Evaluator
```go
// 位置：pkg/loopcontroller/evaluator/
// 文件：
// - evaluator.go - 质量评估器

// QualityEvaluator
type QualityEvaluator interface {
    Evaluate(state *LoopState, context *Context) float64
}

// 组合评估器
type CompositeQualityEvaluator struct {
    evaluators []WeightedEvaluator
}

// 内置评估器
type SuccessRateEvaluator struct{}
type ProgressEvaluator struct{}
type EfficiencyEvaluator struct{}
```

**测试用例：**
- 成功率评估
- 进度评估
- 效率评估

---

## 📦 Phase 2: 增强功能（短期）

### 2.1 Thinker Toolkit - Phase 2

- [ ] FewShotManager - 示例管理和智能选择
- [ ] KeywordSelector - 关键词工具选择
- [ ] ContextWindow - 自动窗口管理
- [ ] TokenTracker 增强 - 更详细的统计

### 2.2 Actor Toolkit - Phase 2

- [ ] Permission System - 权限控制
- [ ] Input Sanitizer - 输入清理
- [ ] Mock Tool - 测试工具
- [ ] Test Utilities - 测试工具集

### 2.3 Observer Toolkit - Phase 2

- [ ] Progress Tracker - 进度追踪
- [ ] Result Analyzer - 结果分析
- [ ] Feedback Optimizer - 反馈优化

### 2.4 LoopController Toolkit - Phase 2

- [ ] Dynamic Iteration Limit - 动态迭代限制
- [ ] Cost Tracker - 成本追踪
- [ ] Early Stop - 早停机制

---

## 📦 Phase 3: 高级功能（中期）

### 3.1 Thinker Toolkit - Phase 3

- [ ] Semantic Selector - 语义工具选择
- [ ] Summarize Strategy - LLM 摘要压缩
- [ ] TikToken 集成 - OpenAI tokenizer
- [ ] SentencePiece 集成 - Llama/Qwen tokenizer

### 3.2 Actor Toolkit - Phase 3

- [ ] Tool Pipeline - 工具组合
- [ ] Auto Discovery - 自动发现
- [ ] Async Execution - 异步执行

### 3.3 Observer Toolkit - Phase 3

- [ ] LLM-based Feedback - 使用 LLM 生成反馈
- [ ] Adaptive Observer - 自适应观察器
- [ ] Multi-criteria Evaluation - 多维度评估

### 3.4 LoopController Toolkit - Phase 3

- [ ] Adaptive Controller - 自适应控制器
- [ ] Learning Controller - 学习型控制器
- [ ] Multi-objective Optimization - 多目标优化

---

## 📚 文档清单

### 已完成文档 ✅

**Thinker Toolkit:**
- [x] PROMPT_TOOLKIT_DESIGN.md - 设计文档
- [x] PROMPT_TOOLKIT_USAGE.md - 使用指南（场景驱动）
- [x] PROMPT_TOOLKIT_SUMMARY.md - 实现总结
- [x] examples/prompt_toolkit/README.md - 示例说明

**综合文档:**
- [x] PAIN_POINTS_CATALOG.md - 痛点清单（宣传素材）

### 待完成文档 🔜

**Actor Toolkit:**
- [ ] ACTOR_TOOLKIT_DESIGN.md - 设计文档 ✅ 已创建
- [ ] ACTOR_TOOLKIT_USAGE.md - 使用指南（场景驱动）✅ 已创建
- [ ] examples/actor_toolkit/README.md - 示例说明

**Observer Toolkit:**
- [ ] OBSERVER_TOOLKIT_DESIGN.md - 设计文档 ✅ 已创建
- [ ] OBSERVER_TOOLKIT_USAGE.md - 使用指南（场景驱动）
- [ ] examples/observer_toolkit/README.md - 示例说明

**LoopController Toolkit:**
- [ ] LOOPCONTROLLER_TOOLKIT_DESIGN.md - 设计文档 ✅ 已创建
- [ ] LOOPCONTROLLER_TOOLKIT_USAGE.md - 使用指南（场景驱动）
- [ ] examples/loopcontroller_toolkit/README.md - 示例说明

**综合文档:**
- [ ] DEVELOPMENT_PLAN.md - 开发计划（本文档）✅ 正在创建
- [ ] ARCHITECTURE_OVERVIEW.md - 架构总览
- [ ] GETTING_STARTED.md - 快速开始（整合所有工具箱）

---

## 🎯 实现顺序

### 第一批：Actor Toolkit Phase 1（优先级最高）
**原因：** 解决最痛的问题（参数验证、错误处理）

1. Schema-based Tool (3 天)
2. Execution Wrappers (2 天)
3. Result Formatter (2 天)
4. Execution Tracer (1 天)
5. 示例和测试 (2 天)

**总计：10 天**

### 第二批：Observer Toolkit Phase 1
**原因：** 提升反馈质量，提高任务成功率

1. Smart Feedback Generator (3 天)
2. Result Validator (2 天)
3. Loop Detector (2 天)
4. 示例和测试 (2 天)

**总计：9 天**

### 第三批：LoopController Toolkit Phase 1
**原因：** 智能控制循环，节省资源

1. Composite Stop Condition (2 天)
2. Stagnation Detector (2 天)
3. Quality Evaluator (2 天)
4. 示例和测试 (2 天)

**总计：8 天**

### 第四批：Phase 2 功能
**原因：** 增强功能，提升用户体验

1. Actor Phase 2 (5 天)
2. Observer Phase 2 (4 天)
3. LoopController Phase 2 (4 天)
4. Thinker Phase 2 (5 天)

**总计：18 天**

---

## 🧪 测试策略

### 单元测试
- 每个组件都有独立的单元测试
- 覆盖率目标：80%+
- 测试框架：Go testing + testify

### 集成测试
- 测试工具箱之间的集成
- 测试完整的 ReAct 循环
- 使用 Mock LLM

### 性能测试
- Token 计数准确性
- 压缩效果
- 执行耗时

### 示例测试
- 所有示例代码可运行
- 输出符合预期

---

## 📊 成功指标

### 代码质量
- [ ] 单元测试覆盖率 > 80%
- [ ] 所有示例可运行
- [ ] 文档完整（设计 + 使用指南 + 示例）

### 用户体验
- [ ] 代码量减少 > 50%
- [ ] 调试时间减少 > 70%
- [ ] 学习曲线平缓（5 分钟上手）

### 性能指标
- [ ] Token 计数准确度 > 90%
- [ ] 成本节省 > 60%
- [ ] 任务成功率提升 > 20%

---

## 🚀 里程碑

### Milestone 1: Thinker Toolkit 完成 ✅
- 时间：已完成
- 交付物：
  - 完整的 Prompt 构建工具箱
  - 3 份文档 + 1 个示例
  - 单元测试

### Milestone 2: Actor Toolkit Phase 1 完成
- 时间：10 天
- 交付物：
  - Schema-based Tool
  - Execution Wrappers
  - Result Formatter
  - Execution Tracer
  - 2 份文档 + 1 个示例

### Milestone 3: Observer Toolkit Phase 1 完成
- 时间：9 天
- 交付物：
  - Smart Feedback Generator
  - Result Validator
  - Loop Detector
  - 2 份文档 + 1 个示例

### Milestone 4: LoopController Toolkit Phase 1 完成
- 时间：8 天
- 交付物：
  - Composite Stop Condition
  - Stagnation Detector
  - Quality Evaluator
  - 2 份文档 + 1 个示例

### Milestone 5: Phase 1 完整发布
- 时间：Milestone 4 + 5 天（整合和优化）
- 交付物：
  - 四大工具箱 Phase 1 完成
  - 完整文档体系
  - 综合示例
  - 发布 v1.0

---

## 📝 开发规范

### 代码规范
- 遵循 Go 官方代码规范
- 使用 golangci-lint 检查
- 所有导出函数必须有注释

### 文档规范
- 设计文档：架构、接口、实现优先级
- 使用指南：场景驱动，问题 → 解决方案 → 效果
- 示例代码：可运行、有注释、有 README

### 测试规范
- 单元测试：覆盖核心逻辑
- 集成测试：覆盖关键流程
- 示例测试：确保可运行

### Git 规范
- 分支：feature/toolkit-name
- 提交：feat/fix/docs/test
- PR：包含测试和文档

---

## 🎁 Phase 1.5: 预装实现（Presets）

> 工具箱是"零件"，用户需要的是"成品"。
> 即使不看文档、不懂工具箱，也能**开箱即用**。

### 设计理念

每个环节提供 3-4 个预装实现，覆盖最常见的使用场景：

```
零配置 → 选择预装 → 自定义组合 → 完全自己实现
  ↓         ↓           ↓              ↓
engine.New  presets.*   toolkit 组合    实现接口
```

### 1.5.1 Actor Presets

```go
// 位置：pkg/core/actor/presets/

// SafeActor 安全模式
// 自带：参数验证 + 权限控制 + 输入过滤
// 适用：面向用户的场景，安全第一
actor := presets.NewSafeActor(toolManager,
    presets.WithAllowedTools("calculator", "http", "search"),
    presets.WithDenyCommands("rm", "chmod", "dd"),
)

// ResilientActor 弹性模式
// 自带：参数验证 + 超时(10s) + 重试(3次) + 结果格式化
// 适用：网络环境不稳定，需要容错
actor := presets.NewResilientActor(toolManager,
    presets.WithTimeout(10 * time.Second),
    presets.WithRetry(3, 1 * time.Second),
)

// DebugActor 调试模式
// 自带：参数验证 + 完整追踪 + 性能分析 + 详细日志
// 适用：开发调试阶段
actor := presets.NewDebugActor(toolManager, logger)

// ProductionActor 生产模式（全部最佳实践）
// 自带：参数验证 + 超时 + 重试 + 结果格式化 + 追踪 + 权限控制
// 适用：生产环境
actor := presets.NewProductionActor(toolManager)
```

### 1.5.2 Observer Presets

```go
// 位置：pkg/core/observer/presets/

// SmartObserver 智能反馈
// 自带：工具特定反馈 + 循环检测(3次) + 结果验证
observer := presets.NewSmartObserver()

// StrictObserver 严格验证
// 自带：智能反馈 + 严格结果验证 + 假阳性检测
observer := presets.NewStrictObserver()

// VerboseObserver 详细日志
// 自带：智能反馈 + 完整日志 + 进度追踪
observer := presets.NewVerboseObserver(logger)

// ProductionObserver 生产模式
// 自带：智能反馈 + 循环检测 + 结果验证 + 进度追踪
observer := presets.NewProductionObserver()
```

### 1.5.3 LoopController Presets

```go
// 位置：pkg/core/loopcontroller/presets/

// SmartController 智能控制
// 自带：动态迭代 + 停滞检测(3次) + 早停
ctrl := presets.NewSmartController()

// BudgetController 预算控制
// 自带：智能控制 + 成本追踪 + 成本限制
ctrl := presets.NewBudgetController(0.50)  // 最多 $0.50/任务

// TimedController 时间控制
// 自带：智能控制 + 超时控制
ctrl := presets.NewTimedController(2 * time.Minute)

// ProductionController 生产模式
// 自带：动态迭代 + 停滞检测 + 早停 + 成本控制 + 超时
ctrl := presets.NewProductionController()
```

### 1.5.4 Engine Presets（终极开箱即用）

```go
// 位置：pkg/engine/presets.go

// 一行代码，全部最佳实践
eng := engine.NewProduction(llmClient)

// 等价于：
eng := engine.New(
    engine.WithThinker(thinkerPresets.NewReActThinker(llmClient, tools)),
    engine.WithActor(actorPresets.NewProductionActor(toolManager)),
    engine.WithObserver(observerPresets.NewProductionObserver()),
    engine.WithLoopController(loopPresets.NewProductionController()),
)

// 其他预装 Engine：
eng := engine.NewDevelopment(llmClient)   // 开发模式（详细日志）
eng := engine.NewBudget(llmClient, 0.50)  // 预算模式（成本控制）
eng := engine.NewFast(llmClient)          // 快速模式（早停 + 超时）
```

---

## 📅 更新后的开发顺序

### Step 1: Actor Toolkit + Presets
1. `pkg/actor/schema/` - Schema 定义 + 验证器 + 类型转换
2. `pkg/actor/wrapper/` - Timeout + Retry + 组合包装器
3. `pkg/actor/formatter/` - Result Formatter + Error Formatter
4. `pkg/actor/debug/` - Execution Tracer
5. `pkg/core/actor/presets/` - Safe / Resilient / Debug / Production

### Step 2: Observer Toolkit + Presets
1. `pkg/observer/feedback/` - Smart Feedback Generator
2. `pkg/observer/validator/` - Result Validator + Rules
3. `pkg/observer/detector/` - Loop Detector
4. `pkg/core/observer/presets/` - Smart / Strict / Verbose / Production

### Step 3: LoopController Toolkit + Presets
1. `pkg/loopcontroller/condition/` - Stop Conditions
2. `pkg/loopcontroller/detector/` - Stagnation Detector
3. `pkg/loopcontroller/tracker/` - Cost Tracker
4. `pkg/core/loopcontroller/presets/` - Smart / Budget / Timed / Production

### Step 4: Engine Presets + 集成测试
1. `pkg/engine/presets.go` - Production / Development / Budget / Fast
2. `examples/production/` - 生产模式示例
3. 集成测试 - 端到端测试

---

## 🎉 总结

### 已完成
- ✅ Thinker Toolkit Phase 1（完整实现 + 文档）
- ✅ 四大工具箱设计文档
- ✅ 痛点清单（21 个痛点，带代码示例，宣传素材）
- ✅ 开发计划（本文档）
- ✅ 预装实现规划（Presets）

### 下一步
1. **立即开始：** Actor Toolkit + Presets 实现
2. **并行进行：** Observer / LoopController 使用指南
3. **最终目标：** `engine.NewProduction(llm)` 一行代码开箱即用

### 三层架构

```
┌─────────────────────────────────────────────┐
│  Layer 3: Presets（开箱即用）                  │
│  engine.NewProduction(llm)                   │
│  一行代码，全部最佳实践                         │
├─────────────────────────────────────────────┤
│  Layer 2: Toolkit（工具箱）                    │
│  schema.NewTool() / wrapper.Wrap()           │
│  按需组合，灵活配置                             │
├─────────────────────────────────────────────┤
│  Layer 1: Interface（接口）                    │
│  Tool / Actor / Observer / LoopController    │
│  完全自定义，实现接口即可                        │
└─────────────────────────────────────────────┘
```

**核心价值：**
- **Layer 1** - 给专家用：完全控制
- **Layer 2** - 给进阶用户用：灵活组合
- **Layer 3** - 给所有人用：开箱即用

**记住：我们不是在做框架，而是在做工具箱。每个工具都应该独立有价值，组合起来更强大。预装实现让不看文档的人也能用好。**
