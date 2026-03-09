# LoopController Toolkit 设计方案

## 概述

LoopController 负责控制 ReAct 循环的流程，决定何时继续、何时停止。开发者的核心痛点和解决方案。

---

## 核心痛点分析

### 痛点 1：只有简单的迭代次数限制

**当前实现：**
```go
if state.Iteration >= c.maxIterations {
    return &types.LoopAction{
        ShouldContinue: false,
        Reason:         "Reached maximum iterations",
    }
}
```

**问题：**
- 只能设置固定的最大迭代次数
- 简单任务浪费迭代次数
- 复杂任务迭代次数不够
- 无法根据任务复杂度动态调整

### 痛点 2：无法检测任务停滞

**问题：**
- LLM 可能陷入无意义的循环
- 例如：连续 5 次都在"思考"，没有实际行动
- 例如：连续 3 次执行相同的失败操作
- 浪费 tokens 和时间

### 痛点 3：缺少智能停止条件

**问题：**
- 只检查 `ShouldFinish` 标志
- 无法检测任务已经部分完成
- 无法检测任务无法完成
- 无法根据成本/收益决定是否继续

### 痛点 4：无法处理超时

**问题：**
- 没有总体时间限制
- 任务可能运行很久
- 无法设置 deadline
- 无法根据优先级分配时间

### 痛点 5：缺少早停机制

**问题：**
- 即使任务已经足够好，也会继续迭代
- 无法根据质量阈值提前停止
- 无法根据成本控制提前停止
- 浪费资源

### 痛点 6：无法动态调整策略

**问题：**
- 控制策略是静态的
- 无法根据任务进展调整
- 无法根据历史表现优化
- 无法学习最佳停止时机

---

## 解决方案设计

### 1. 多维度停止条件

```go
// StopCondition 停止条件接口
type StopCondition interface {
    ShouldStop(state *LoopState, context *Context) (bool, string)
}

// CompositeStopCondition 组合停止条件
type CompositeStopCondition struct {
    conditions []StopCondition
    mode       StopMode  // AND 或 OR
}

// 内置停止条件
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

type CostLimitCondition struct {
    maxCost float64
    tracker CostTracker
}

// 使用示例
controller := NewSmartLoopController().
    WithCondition(NewMaxIterationCondition(10)).
    WithCondition(NewTimeoutCondition(5 * time.Minute)).
    WithCondition(NewStagnationCondition(3)).
    WithCondition(NewQualityThresholdCondition(0.8)).
    WithMode(StopModeAny)  // 任一条件满足就停止
```

### 2. 停滞检测器

```go
// StagnationDetector 停滞检测器
type StagnationDetector struct {
    window      int  // 检测窗口大小
    threshold   float64
}

type StagnationPattern struct {
    Detected    bool
    Type        string  // "no_action", "repeated_failure", "no_progress"
    Description string
    Suggestion  string
}

func (d *StagnationDetector) Detect(state *LoopState, history []LoopState) StagnationPattern {
    // 检测模式 1：连续多次没有执行 Action
    noActionCount := 0
    for i := len(history) - 1; i >= 0 && i >= len(history)-d.window; i-- {
        if history[i].LastThought == nil || history[i].LastThought.Action == nil {
            noActionCount++
        }
    }
    if noActionCount >= d.window {
        return StagnationPattern{
            Detected:    true,
            Type:        "no_action",
            Description: fmt.Sprintf("No actions taken in the last %d iterations", d.window),
            Suggestion:  "The agent seems stuck in thinking. Consider providing more specific instructions or examples.",
        }
    }

    // 检测模式 2：重复失败
    if d.detectRepeatedFailure(history) {
        return StagnationPattern{
            Detected:    true,
            Type:        "repeated_failure",
            Description: "Same action failed multiple times",
            Suggestion:  "Try a different approach or tool.",
        }
    }

    // 检测模式 3：无进展
    if d.detectNoProgress(history) {
        return StagnationPattern{
            Detected:    true,
            Type:        "no_progress",
            Description: "No measurable progress in recent iterations",
            Suggestion:  "The task may be too complex or unclear. Consider breaking it down.",
        }
    }

    return StagnationPattern{Detected: false}
}
```

### 3. 动态迭代限制

```go
// DynamicIterationLimit 动态迭代限制
type DynamicIterationLimit struct {
    baseLimit    int
    adjuster     IterationAdjuster
}

type IterationAdjuster interface {
    Adjust(task string, context *Context) int
}

// TaskComplexityAdjuster 根据任务复杂度调整
type TaskComplexityAdjuster struct {
    analyzer ComplexityAnalyzer
}

func (a *TaskComplexityAdjuster) Adjust(task string, context *Context) int {
    complexity := a.analyzer.Analyze(task)

    switch complexity {
    case ComplexitySimple:
        return 5   // 简单任务：5 次迭代
    case ComplexityMedium:
        return 10  // 中等任务：10 次迭代
    case ComplexityComplex:
        return 20  // 复杂任务：20 次迭代
    default:
        return 10
    }
}

// HistoricalAdjuster 根据历史表现调整
type HistoricalAdjuster struct {
    history map[string]int  // task_type -> avg_iterations
}

func (a *HistoricalAdjuster) Adjust(task string, context *Context) int {
    taskType := a.classifyTask(task)
    if avgIterations, ok := a.history[taskType]; ok {
        return int(float64(avgIterations) * 1.2)  // 加 20% 余量
    }
    return 10  // 默认值
}
```

### 4. 质量评估器

```go
// QualityEvaluator 质量评估器
type QualityEvaluator interface {
    Evaluate(state *LoopState, context *Context) float64
}

// CompositeQualityEvaluator 组合质量评估器
type CompositeQualityEvaluator struct {
    evaluators []WeightedEvaluator
}

type WeightedEvaluator struct {
    Evaluator QualityEvaluator
    Weight    float64
}

// 内置评估器
type SuccessRateEvaluator struct{}  // 成功率
type ProgressEvaluator struct{}     // 进度
type EfficiencyEvaluator struct{}   // 效率
type AccuracyEvaluator struct{}     // 准确性

// 使用示例
evaluator := NewCompositeQualityEvaluator().
    WithEvaluator(NewSuccessRateEvaluator(), 0.3).
    WithEvaluator(NewProgressEvaluator(), 0.3).
    WithEvaluator(NewEfficiencyEvaluator(), 0.2).
    WithEvaluator(NewAccuracyEvaluator(), 0.2)

quality := evaluator.Evaluate(state, context)
// quality = 0.85 (85% 质量)

if quality >= 0.8 {
    // 质量足够好，可以提前停止
}
```

### 5. 成本追踪器

```go
// CostTracker 成本追踪器
type CostTracker struct {
    tokenCost   float64
    timeCost    time.Duration
    apiCalls    int
    pricing     PricingModel
}

type PricingModel struct {
    InputTokenPrice  float64  // 每 1K tokens
    OutputTokenPrice float64
    APICallPrice     float64
}

func (t *CostTracker) Track(state *LoopState) {
    // 追踪 token 使用
    t.tokenCost += t.calculateTokenCost(state)

    // 追踪时间
    t.timeCost += state.Duration

    // 追踪 API 调用
    t.apiCalls++
}

func (t *CostTracker) GetTotalCost() float64 {
    return t.tokenCost + float64(t.apiCalls)*t.pricing.APICallPrice
}

func (t *CostTracker) ShouldStop(maxCost float64) bool {
    return t.GetTotalCost() >= maxCost
}
```

### 6. 智能循环控制器

```go
// SmartLoopController 智能循环控制器
type SmartLoopController struct {
    conditions       []StopCondition
    stagnationDetector *StagnationDetector
    qualityEvaluator   QualityEvaluator
    costTracker        *CostTracker
    iterationAdjuster  IterationAdjuster
    earlyStopEnabled   bool
}

func (c *SmartLoopController) Control(state *LoopState) *LoopAction {
    // 1. 检查所有停止条件
    for _, condition := range c.conditions {
        if shouldStop, reason := condition.ShouldStop(state, c.context); shouldStop {
            return &LoopAction{
                ShouldContinue: false,
                Reason:         reason,
                Metadata: map[string]interface{}{
                    "stop_condition": condition.Name(),
                    "quality":        c.qualityEvaluator.Evaluate(state, c.context),
                    "cost":           c.costTracker.GetTotalCost(),
                },
            }
        }
    }

    // 2. 检测停滞
    pattern := c.stagnationDetector.Detect(state, c.history)
    if pattern.Detected {
        return &LoopAction{
            ShouldContinue: false,
            Reason:         pattern.Description,
            Metadata: map[string]interface{}{
                "stagnation_type": pattern.Type,
                "suggestion":      pattern.Suggestion,
            },
        }
    }

    // 3. 早停检查
    if c.earlyStopEnabled {
        quality := c.qualityEvaluator.Evaluate(state, c.context)
        if quality >= c.earlyStopThreshold {
            return &LoopAction{
                ShouldContinue: false,
                Reason:         fmt.Sprintf("Quality threshold reached (%.2f)", quality),
                Metadata: map[string]interface{}{
                    "quality":     quality,
                    "early_stop":  true,
                },
            }
        }
    }

    // 4. 继续循环
    return &LoopAction{
        ShouldContinue: true,
        Reason:         "Continue processing",
        Metadata: map[string]interface{}{
            "iteration":      state.Iteration,
            "quality":        c.qualityEvaluator.Evaluate(state, c.context),
            "cost":           c.costTracker.GetTotalCost(),
            "estimated_remaining": c.estimateRemainingIterations(state),
        },
    }
}
```

---

## 实现优先级

### Phase 1: 核心功能
1. ✅ CompositeStopCondition - 多维度停止条件
2. ✅ StagnationDetector - 停滞检测
3. ✅ TimeoutCondition - 超时控制
4. ✅ QualityEvaluator - 质量评估

### Phase 2: 增强功能
1. ⏳ DynamicIterationLimit - 动态迭代限制
2. ⏳ CostTracker - 成本追踪
3. ⏳ EarlyStop - 早停机制

### Phase 3: 高级功能
1. 🔮 AdaptiveController - 自适应控制器
2. 🔮 LearningController - 学习型控制器
3. 🔮 Multi-objective Optimization - 多目标优化

---

## 使用场景

### 场景 1：简单任务快速完成

```go
// 当前：固定 10 次迭代，简单任务浪费
controller := NewDefaultLoopController(10)

// 改进：动态调整
controller := NewSmartLoopController().
    WithDynamicLimit(NewTaskComplexityAdjuster()).
    WithEarlyStop(0.8)  // 质量达到 80% 就停止

// 简单任务：3 次迭代就完成
// 复杂任务：自动扩展到 20 次
```

### 场景 2：检测停滞并提前停止

```go
// 检测到停滞
detector.Detect(state, history)
// 返回：
"⚠️ Stagnation detected: No actions taken in the last 3 iterations.
The agent seems stuck in thinking without making progress.
Suggestion: The task may be unclear or too complex. Consider:
1. Providing more specific instructions
2. Breaking down the task into smaller steps
3. Providing examples"
```

### 场景 3：成本控制

```go
controller := NewSmartLoopController().
    WithCostLimit(1.0)  // 最多花费 $1

// 执行过程中追踪成本
// 当成本接近 $1 时自动停止
// 返回：
"Stopped due to cost limit: $0.98 / $1.00
Completed 8 iterations with 85% quality.
Consider increasing the budget or simplifying the task."
```

### 场景 4：质量驱动的早停

```go
controller := NewSmartLoopController().
    WithQualityEvaluator(evaluator).
    WithEarlyStop(0.9)  // 质量达到 90% 就停止

// 第 5 次迭代时质量达到 92%
// 自动停止，节省 5 次迭代
// 返回：
"Task completed with high quality (92%) after 5 iterations.
Early stop triggered to save resources.
Estimated savings: 5 iterations, ~$0.50"
```

---

## 总结

LoopController Toolkit 提供：
1. **多维度停止条件** - 迭代次数、超时、质量、成本
2. **停滞检测** - 避免无意义的循环
3. **动态调整** - 根据任务复杂度自动调整
4. **质量评估** - 多维度评估任务完成质量
5. **成本控制** - 追踪和限制执行成本
6. **早停机制** - 质量足够好时提前停止

**核心价值：智能控制循环，避免浪费，提高效率。**
