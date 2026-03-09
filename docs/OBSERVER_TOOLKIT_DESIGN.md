# Observer Toolkit 设计方案

## 概述

Observer 负责观察工具执行结果，并提供反馈给下一轮循环。开发者的核心痛点和解决方案。

---

## 核心痛点分析

### 痛点 1：反馈消息千篇一律，LLM 无法从中学习

**当前实现：**
```go
if result.Success {
    feedback.Message = fmt.Sprintf("Tool executed successfully. Result: %v", result.Output)
} else {
    feedback.Message = fmt.Sprintf("Tool execution failed: %v. Please try a different approach.", result.Error)
}
```

**问题：**
- 成功消息太简单，没有提取关键信息
- 失败消息太泛化，LLM 不知道具体哪里错了
- 没有根据结果类型定制反馈
- 没有提供改进建议

### 痛点 2：无法检测任务是否真正完成

**问题：**
- 工具执行成功 ≠ 任务完成
- 例如：HTTP 请求成功返回 404，但任务失败
- 例如：计算器返回结果，但结果不符合预期
- 无法验证结果的正确性

### 痛点 3：无法检测循环陷阱

**问题：**
- LLM 可能重复执行相同的失败操作
- 例如：连续 3 次用相同参数调用失败的工具
- 浪费 tokens 和时间
- 无法自动跳出死循环

### 痛点 4：缺少结果分析能力

**问题：**
- 无法提取结果中的关键信息
- 无法判断结果的质量
- 无法提供下一步建议
- 无法检测异常模式

### 痛点 5：反馈不够智能

**问题：**
- 不会根据历史调整反馈
- 不会根据任务类型定制反馈
- 不会提供具体的改进建议
- 不会检测进度停滞

---

## 解决方案设计

### 1. 智能反馈生成器

```go
// FeedbackGenerator 反馈生成器接口
type FeedbackGenerator interface {
    Generate(result *ExecutionResult, context *Context) string
}

// SmartFeedbackGenerator 智能反馈生成器
type SmartFeedbackGenerator struct {
    generators map[string]FeedbackGenerator  // 按工具类型定制
    analyzer   ResultAnalyzer
}

// 使用示例
generator := NewSmartFeedbackGenerator().
    WithToolGenerator("http", NewHTTPFeedbackGenerator()).
    WithToolGenerator("calculator", NewCalculatorFeedbackGenerator()).
    WithDefaultGenerator(NewGenericFeedbackGenerator())

feedback := generator.Generate(result, context)
```

**效果对比：**

| 场景 | 当前反馈 | 智能反馈 |
|------|---------|---------|
| HTTP 404 | "Tool executed successfully. Result: 404" | "HTTP request succeeded but returned 404 Not Found. The resource at '/api/users' does not exist. Please check the URL path." |
| 计算错误 | "Tool executed successfully. Result: 15" | "Calculator returned 15. However, the expected result for '10 + 5' should be 15. ✓ Result is correct." |
| 连接失败 | "Tool execution failed: connection refused" | "HTTP request failed because the server refused the connection. This usually means: 1) The service is not running, 2) The port is incorrect, 3) Firewall is blocking. Suggestion: Check if the service is running on port 8080." |

### 2. 结果验证器

```go
// ResultValidator 结果验证器
type ResultValidator interface {
    Validate(result *ExecutionResult, context *Context) ValidationResult
}

type ValidationResult struct {
    IsValid      bool
    Confidence   float64  // 0.0-1.0
    Issues       []string
    Suggestions  []string
}

// 使用示例
validator := NewResultValidator().
    WithRule(NewHTTPStatusRule()).      // HTTP 状态码检查
    WithRule(NewDataFormatRule()).      // 数据格式检查
    WithRule(NewExpectedValueRule()).   // 预期值检查

validation := validator.Validate(result, context)
if !validation.IsValid {
    feedback.Message += "\nIssues found: " + strings.Join(validation.Issues, ", ")
    feedback.Message += "\nSuggestions: " + strings.Join(validation.Suggestions, ", ")
}
```

### 3. 循环检测器

```go
// LoopDetector 循环检测器
type LoopDetector struct {
    history      []ActionRecord
    maxRepeats   int
    windowSize   int
}

type ActionRecord struct {
    ToolName   string
    Parameters map[string]interface{}
    Success    bool
    Timestamp  time.Time
}

func (d *LoopDetector) DetectLoop(action *Action) LoopPattern {
    // 检测重复模式
    pattern := d.findPattern(action)

    if pattern.RepeatCount >= d.maxRepeats {
        return LoopPattern{
            Detected:    true,
            Pattern:     pattern.Actions,
            RepeatCount: pattern.RepeatCount,
            Suggestion:  "You've tried this action 3 times with the same parameters. Consider trying a different approach.",
        }
    }

    return LoopPattern{Detected: false}
}
```

### 4. 进度追踪器

```go
// ProgressTracker 进度追踪器
type ProgressTracker struct {
    milestones []Milestone
    current    int
}

type Milestone struct {
    Name        string
    Description string
    Achieved    bool
    Timestamp   time.Time
}

func (t *ProgressTracker) Track(result *ExecutionResult, context *Context) ProgressReport {
    // 分析是否达成里程碑
    // 检测进度停滞
    // 估算完成度

    return ProgressReport{
        CompletionRate: 0.6,  // 60% 完成
        CurrentPhase:   "Data Collection",
        NextSteps:      []string{"Process data", "Generate report"},
        IsStuck:        false,
    }
}
```

### 5. 结果分析器

```go
// ResultAnalyzer 结果分析器
type ResultAnalyzer interface {
    Analyze(result *ExecutionResult) AnalysisReport
}

type AnalysisReport struct {
    Summary      string
    KeyPoints    []string
    DataQuality  float64
    Anomalies    []string
    Insights     []string
}

// 使用示例
analyzer := NewResultAnalyzer().
    WithExtractor(NewKeyInfoExtractor()).
    WithDetector(NewAnomalyDetector()).
    WithSummarizer(NewSmartSummarizer())

report := analyzer.Analyze(result)
```

---

## 实现优先级

### Phase 1: 核心功能
1. ✅ SmartFeedbackGenerator - 智能反馈生成
2. ✅ ResultValidator - 结果验证
3. ✅ LoopDetector - 循环检测

### Phase 2: 增强功能
1. ⏳ ProgressTracker - 进度追踪
2. ⏳ ResultAnalyzer - 结果分析
3. ⏳ FeedbackOptimizer - 反馈优化

### Phase 3: 高级功能
1. 🔮 LLM-based Feedback - 使用 LLM 生成反馈
2. 🔮 Adaptive Observer - 自适应观察器
3. 🔮 Multi-criteria Evaluation - 多维度评估

---

## 使用场景

### 场景 1：HTTP 请求返回错误状态码

```go
// 当前反馈
"Tool executed successfully. Result: {status: 404, body: 'Not Found'}"

// 智能反馈
"HTTP request completed but returned 404 Not Found. The endpoint '/api/users/999' does not exist.
Possible reasons:
1. The user ID 999 does not exist in the database
2. The API path is incorrect
Suggestions:
1. Try listing all users first with GET /api/users
2. Check if the user ID is correct"
```

### 场景 2：检测重复失败

```go
// 检测到循环
detector.DetectLoop(action)
// 返回：
"⚠️ Loop detected: You've tried 'http' with the same URL 3 times, all failed with 'connection refused'.
This suggests the service is not available.
Suggestion: Instead of retrying the same request, consider:
1. Checking if the service is running
2. Using a different endpoint
3. Asking the user for the correct URL"
```

### 场景 3：验证计算结果

```go
// 验证结果
validator.Validate(result, context)
// 返回：
"Calculator returned 15 for '10 + 5'. ✓ Result is mathematically correct.
However, the task asked for '10 * 5', not '10 + 5'.
⚠️ Wrong operation used. Please use 'multiply' instead of 'add'."
```

---

## 总结

Observer Toolkit 提供：
1. **智能反馈生成** - 根据工具类型和结果定制反馈
2. **结果验证** - 检查结果的正确性和质量
3. **循环检测** - 避免重复失败
4. **进度追踪** - 监控任务进度
5. **结果分析** - 提取关键信息和洞察

**核心价值：让 LLM 从反馈中真正学习，而不是收到千篇一律的消息。**
