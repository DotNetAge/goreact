# Actor Toolkit 设计方案

## 概述

为 Actor 开发提供完整的工具执行工具箱，解决工具开发和执行中的核心痛点。

## 设计原则

1. **声明式优于命令式**：用 Schema 声明，而非手写验证代码
2. **安全第一**：默认安全，显式授权
3. **可观测**：详细的执行追踪和调试信息
4. **易测试**：支持 Mock 和隔离测试
5. **渐进增强**：从简单到复杂，按需使用

---

## 核心痛点 → 解决方案映射

| 痛点 | 解决方案 | 优先级 |
|------|---------|--------|
| 参数验证和类型转换 | Schema Validator + Type Converter | P0 |
| 没有错误重试 | Retry Wrapper | P0 |
| 没有超时控制 | Timeout Wrapper | P0 |
| 输出格式混乱 | Result Formatter | P0 |
| 缺少工具元数据 | Schema-based Tool | P0 |
| 调试困难 | Execution Tracer | P1 |
| 安全问题 | Permission & Sandbox | P1 |
| 没有工具组合 | Tool Pipeline | P2 |
| 测试困难 | Mock Tool & Test Utils | P1 |
| 注册繁琐 | Auto Discovery | P2 |

---

## 一、参数验证和类型转换工具

### 1.1 Schema-based Tool 定义

**不要这样做：**
```go
// ❌ 手写验证逻辑
func (c *Calculator) Execute(params map[string]interface{}) (interface{}, error) {
    operation, ok := params["operation"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid 'operation' parameter")
    }

    a, ok := toFloat64(params["a"])
    if !ok {
        return nil, fmt.Errorf("missing or invalid 'a' parameter")
    }
    // ... 重复的验证代码
}
```

**应该这样做：**
```go
// ✅ 使用 Schema 声明
type Calculator struct {
    tool.SchemaTool
}

func NewCalculator() tool.Tool {
    return &Calculator{
        SchemaTool: tool.NewSchemaTool(
            "calculator",
            "Perform arithmetic operations",
            tool.Schema{
                Parameters: tool.Object{
                    Properties: map[string]tool.Property{
                        "operation": {
                            Type:        tool.String,
                            Description: "The operation to perform",
                            Enum:        []string{"add", "subtract", "multiply", "divide"},
                            Required:    true,
                        },
                        "a": {
                            Type:        tool.Number,
                            Description: "First operand",
                            Required:    true,
                        },
                        "b": {
                            Type:        tool.Number,
                            Description: "Second operand",
                            Required:    true,
                        },
                    },
                },
            },
            calculatorHandler,  // 只需要实现业务逻辑
        ),
    }
}

// 业务逻辑函数，参数已经验证和转换好了
func calculatorHandler(params tool.ValidatedParams) (interface{}, error) {
    operation := params.GetString("operation")
    a := params.GetFloat64("a")
    b := params.GetFloat64("b")

    switch operation {
    case "add":
        return a + b, nil
    case "subtract":
        return a - b, nil
    case "multiply":
        return a * b, nil
    case "divide":
        if b == 0 {
            return nil, tool.NewUserError("division by zero")
        }
        return a / b, nil
    default:
        return nil, tool.NewUserError("unknown operation: %s", operation)
    }
}
```

**效果对比：**
- 代码量：减少 60%
- 验证逻辑：自动生成
- 类型转换：自动处理
- 错误消息：自动生成，LLM 友好

### 1.2 智能类型转换器

```go
// TypeConverter 智能类型转换
type TypeConverter struct {
    strict bool  // 严格模式：不允许有损转换
}

// 支持的转换：
// - "123" → 123 (string to int)
// - "true" → true (string to bool)
// - 123 → "123" (int to string)
// - "123.45" → 123.45 (string to float)
// - []interface{}{1,2,3} → []int{1,2,3} (slice conversion)

func (c *TypeConverter) Convert(value interface{}, targetType PropertyType) (interface{}, error) {
    // 智能转换逻辑
}

// 使用示例
converter := NewTypeConverter(false)  // 非严格模式
result, err := converter.Convert("123", tool.Number)
// result = 123.0
```

### 1.3 ValidatedParams 辅助类

```go
// ValidatedParams 已验证的参数
type ValidatedParams struct {
    raw    map[string]interface{}
    schema Schema
}

// 类型安全的 Getter
func (p *ValidatedParams) GetString(key string) string
func (p *ValidatedParams) GetInt(key string) int
func (p *ValidatedParams) GetFloat64(key string) float64
func (p *ValidatedParams) GetBool(key string) bool
func (p *ValidatedParams) GetStringSlice(key string) []string
func (p *ValidatedParams) GetMap(key string) map[string]interface{}

// 带默认值的 Getter
func (p *ValidatedParams) GetStringOr(key, defaultValue string) string
func (p *ValidatedParams) GetIntOr(key string, defaultValue int) int

// 可选参数检查
func (p *ValidatedParams) Has(key string) bool
```

---

## 二、执行包装器（Wrappers）

### 2.1 Timeout Wrapper

```go
// TimeoutWrapper 超时包装器
type TimeoutWrapper struct {
    timeout time.Duration
}

func NewTimeoutWrapper(timeout time.Duration) *TimeoutWrapper {
    return &TimeoutWrapper{timeout: timeout}
}

func (w *TimeoutWrapper) Wrap(tool Tool) Tool {
    return &timeoutTool{
        base:    tool,
        timeout: w.timeout,
    }
}

// 使用示例
calculator := builtin.NewCalculator()
withTimeout := NewTimeoutWrapper(5 * time.Second).Wrap(calculator)

// 执行时自动超时控制
result, err := withTimeout.Execute(params)
// 如果超过 5 秒，返回 context.DeadlineExceeded
```

### 2.2 Retry Wrapper

```go
// RetryWrapper 重试包装器
type RetryWrapper struct {
    maxAttempts int
    interval    time.Duration
    retryIf     func(error) bool  // 判断是否应该重试
}

func NewRetryWrapper(maxAttempts int, interval time.Duration) *RetryWrapper {
    return &RetryWrapper{
        maxAttempts: maxAttempts,
        interval:    interval,
        retryIf:     isRetryableError,  // 默认策略
    }
}

// 默认的重试策略
func isRetryableError(err error) bool {
    // 网络错误、超时错误 → 可重试
    // 参数错误、业务逻辑错误 → 不可重试
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    if errors.Is(err, syscall.ECONNREFUSED) {
        return true
    }
    if _, ok := err.(*UserError); ok {
        return false  // 用户错误不重试
    }
    return false
}

// 使用示例
httpTool := builtin.NewHTTP()
withRetry := NewRetryWrapper(3, 1*time.Second).Wrap(httpTool)

// 执行时自动重试
result, err := withRetry.Execute(params)
// 失败时最多重试 3 次，每次间隔 1 秒
```

### 2.3 组合包装器

```go
// 组合多个包装器
tool := builtin.NewHTTP()
tool = NewTimeoutWrapper(10 * time.Second).Wrap(tool)
tool = NewRetryWrapper(3, 1 * time.Second).Wrap(tool)
tool = NewLoggingWrapper(logger).Wrap(tool)

// 或者使用 Pipeline
tool = WrapTool(builtin.NewHTTP(),
    WithTimeout(10 * time.Second),
    WithRetry(3, 1 * time.Second),
    WithLogging(logger),
)
```

---

## 三、结果格式化工具

### 3.1 Result Formatter

```go
// ResultFormatter 结果格式化器
type ResultFormatter interface {
    Format(result interface{}) (string, error)
}

// TextFormatter 文本格式化器
type TextFormatter struct {
    maxLength int  // 最大长度（避免超长输出）
}

func (f *TextFormatter) Format(result interface{}) (string, error) {
    text := fmt.Sprintf("%v", result)
    if len(text) > f.maxLength {
        return text[:f.maxLength] + "... (truncated)", nil
    }
    return text, nil
}

// JSONFormatter JSON 格式化器
type JSONFormatter struct {
    indent    bool
    maxLength int
}

func (f *JSONFormatter) Format(result interface{}) (string, error) {
    var data []byte
    var err error
    if f.indent {
        data, err = json.MarshalIndent(result, "", "  ")
    } else {
        data, err = json.Marshal(result)
    }
    if err != nil {
        return "", err
    }

    text := string(data)
    if len(text) > f.maxLength {
        return text[:f.maxLength] + "... (truncated)", nil
    }
    return text, nil
}

// SummaryFormatter 摘要格式化器（提取关键信息）
type SummaryFormatter struct {
    maxLines int
}

func (f *SummaryFormatter) Format(result interface{}) (string, error) {
    // 根据类型提取摘要
    switch v := result.(type) {
    case string:
        return f.summarizeText(v)
    case []byte:
        return f.summarizeBytes(v)
    case map[string]interface{}:
        return f.summarizeMap(v)
    default:
        return fmt.Sprintf("%v", v), nil
    }
}
```

### 3.2 Error Formatter

```go
// ErrorFormatter 错误格式化器（生成 LLM 友好的错误消息）
type ErrorFormatter struct{}

func (f *ErrorFormatter) Format(err error, toolName string, params map[string]interface{}) string {
    // 根据错误类型生成不同的消息

    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Sprintf("Tool '%s' timed out. The operation took too long to complete. You may want to try again or use a different approach.", toolName)
    }

    if errors.Is(err, syscall.ECONNREFUSED) {
        return fmt.Sprintf("Tool '%s' failed to connect. The service may be unavailable. Please check if the service is running.", toolName)
    }

    if userErr, ok := err.(*UserError); ok {
        return fmt.Sprintf("Tool '%s' failed: %s. Please check your parameters: %v", toolName, userErr.Message, params)
    }

    // 默认消息
    return fmt.Sprintf("Tool '%s' encountered an error: %s", toolName, err.Error())
}
```

---

## 四、执行追踪和调试工具

### 4.1 Execution Tracer

```go
// ExecutionTracer 执行追踪器
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
    Metadata   map[string]interface{}
}

func (t *ExecutionTracer) Trace(toolName string, params map[string]interface{}, fn func() (interface{}, error)) (interface{}, error) {
    if !t.enabled {
        return fn()
    }

    trace := &ExecutionTrace{
        ToolName:   toolName,
        Parameters: params,
        StartTime:  time.Now(),
        Metadata:   make(map[string]interface{}),
    }

    output, err := fn()

    trace.EndTime = time.Now()
    trace.Duration = trace.EndTime.Sub(trace.StartTime)
    trace.Success = err == nil
    trace.Output = output
    trace.Error = err

    t.logTrace(trace)

    return output, err
}

func (t *ExecutionTracer) logTrace(trace *ExecutionTrace) {
    t.logger.Info("Tool Execution",
        "tool", trace.ToolName,
        "duration_ms", trace.Duration.Milliseconds(),
        "success", trace.Success,
    )

    if t.logger.IsDebug() {
        t.logger.Debug("Tool Execution Details",
            "parameters", trace.Parameters,
            "output", truncate(fmt.Sprintf("%v", trace.Output), 200),
            "error", trace.Error,
        )
    }
}
```

### 4.2 Performance Profiler

```go
// PerformanceProfiler 性能分析器
type PerformanceProfiler struct {
    stats map[string]*ToolStats
    mu    sync.RWMutex
}

type ToolStats struct {
    TotalCalls      int
    SuccessCalls    int
    FailedCalls     int
    TotalDuration   time.Duration
    AverageDuration time.Duration
    MinDuration     time.Duration
    MaxDuration     time.Duration
}

func (p *PerformanceProfiler) Record(toolName string, duration time.Duration, success bool) {
    p.mu.Lock()
    defer p.mu.Unlock()

    stats, ok := p.stats[toolName]
    if !ok {
        stats = &ToolStats{
            MinDuration: duration,
            MaxDuration: duration,
        }
        p.stats[toolName] = stats
    }

    stats.TotalCalls++
    if success {
        stats.SuccessCalls++
    } else {
        stats.FailedCalls++
    }

    stats.TotalDuration += duration
    stats.AverageDuration = stats.TotalDuration / time.Duration(stats.TotalCalls)

    if duration < stats.MinDuration {
        stats.MinDuration = duration
    }
    if duration > stats.MaxDuration {
        stats.MaxDuration = duration
    }
}

func (p *PerformanceProfiler) Report() string {
    // 生成性能报告
}
```

---

## 五、安全工具

### 5.1 Permission System

```go
// Permission 权限定义
type Permission string

const (
    PermissionFileRead    Permission = "file:read"
    PermissionFileWrite   Permission = "file:write"
    PermissionNetworkHTTP Permission = "network:http"
    PermissionShellExec   Permission = "shell:exec"
    PermissionSystemInfo  Permission = "system:info"
)

// PermissionChecker 权限检查器
type PermissionChecker struct {
    allowedPermissions map[Permission]bool
}

func NewPermissionChecker(allowed []Permission) *PermissionChecker {
    checker := &PermissionChecker{
        allowedPermissions: make(map[Permission]bool),
    }
    for _, perm := range allowed {
        checker.allowedPermissions[perm] = true
    }
    return checker
}

func (c *PermissionChecker) Check(perm Permission) error {
    if !c.allowedPermissions[perm] {
        return fmt.Errorf("permission denied: %s", perm)
    }
    return nil
}

// 在工具中使用
type Filesystem struct {
    checker *PermissionChecker
}

func (f *Filesystem) Execute(params ValidatedParams) (interface{}, error) {
    operation := params.GetString("operation")

    if operation == "write" {
        if err := f.checker.Check(PermissionFileWrite); err != nil {
            return nil, err
        }
    }

    // 执行操作
}
```

### 5.2 Input Sanitizer

```go
// InputSanitizer 输入清理器
type InputSanitizer struct{}

// SanitizeShellCommand 清理 Shell 命令
func (s *InputSanitizer) SanitizeShellCommand(command string) (string, error) {
    // 检查危险命令
    dangerous := []string{"rm -rf", "mkfs", "dd if=", "> /dev/"}
    for _, pattern := range dangerous {
        if strings.Contains(command, pattern) {
            return "", fmt.Errorf("dangerous command detected: %s", pattern)
        }
    }

    // 检查命令注入
    if strings.Contains(command, ";") || strings.Contains(command, "&&") {
        return "", fmt.Errorf("command chaining not allowed")
    }

    return command, nil
}

// SanitizePath 清理文件路径
func (s *InputSanitizer) SanitizePath(path string) (string, error) {
    // 防止路径遍历
    if strings.Contains(path, "..") {
        return "", fmt.Errorf("path traversal not allowed")
    }

    // 转换为绝对路径
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", err
    }

    return absPath, nil
}
```

---

## 六、测试工具

### 6.1 Mock Tool

```go
// MockTool 模拟工具
type MockTool struct {
    name        string
    description string
    responses   []MockResponse
    callCount   int
}

type MockResponse struct {
    Output interface{}
    Error  error
    Delay  time.Duration  // 模拟延迟
}

func NewMockTool(name, description string) *MockTool {
    return &MockTool{
        name:        name,
        description: description,
        responses:   []MockResponse{},
    }
}

func (m *MockTool) WithResponse(output interface{}, err error) *MockTool {
    m.responses = append(m.responses, MockResponse{
        Output: output,
        Error:  err,
    })
    return m
}

func (m *MockTool) WithDelay(delay time.Duration) *MockTool {
    if len(m.responses) > 0 {
        m.responses[len(m.responses)-1].Delay = delay
    }
    return m
}

func (m *MockTool) Execute(params map[string]interface{}) (interface{}, error) {
    if m.callCount >= len(m.responses) {
        return nil, fmt.Errorf("no more mock responses")
    }

    resp := m.responses[m.callCount]
    m.callCount++

    if resp.Delay > 0 {
        time.Sleep(resp.Delay)
    }

    return resp.Output, resp.Error
}

// 使用示例
mockCalc := NewMockTool("calculator", "Mock calculator").
    WithResponse(15.0, nil).                    // 第一次调用返回 15
    WithResponse(nil, errors.New("error")).     // 第二次调用返回错误
    WithResponse(30.0, nil).WithDelay(2*time.Second)  // 第三次调用延迟 2 秒
```

### 6.2 Test Utilities

```go
// TestToolExecution 测试工具执行
func TestToolExecution(t *testing.T, tool Tool, testCases []TestCase) {
    for _, tc := range testCases {
        t.Run(tc.Name, func(t *testing.T) {
            output, err := tool.Execute(tc.Params)

            if tc.ExpectError {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tc.ErrorContains)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.ExpectedOutput, output)
            }
        })
    }
}

// 使用示例
TestToolExecution(t, calculator, []TestCase{
    {
        Name:           "add two numbers",
        Params:         map[string]interface{}{"operation": "add", "a": 10, "b": 5},
        ExpectedOutput: 15.0,
    },
    {
        Name:          "division by zero",
        Params:        map[string]interface{}{"operation": "divide", "a": 10, "b": 0},
        ExpectError:   true,
        ErrorContains: "division by zero",
    },
})
```

---

## 七、实现优先级

### Phase 1: 核心功能（立即实现）
1. ✅ Schema-based Tool
2. ✅ ValidatedParams
3. ✅ TypeConverter
4. ✅ TimeoutWrapper
5. ✅ RetryWrapper
6. ✅ ResultFormatter

### Phase 2: 增强功能（短期）
1. ⏳ ExecutionTracer
2. ⏳ PerformanceProfiler
3. ⏳ ErrorFormatter
4. ⏳ MockTool

### Phase 3: 高级功能（中期）
1. 🔮 PermissionSystem
2. 🔮 InputSanitizer
3. 🔮 Tool Pipeline
4. 🔮 Auto Discovery

---

## 八、使用示例

### 简单使用
```go
// 使用 Schema 定义工具
calculator := tool.NewSchemaTool(
    "calculator",
    "Perform arithmetic operations",
    schema,
    handler,
)
```

### 进阶使用
```go
// 添加包装器
calculator = WrapTool(calculator,
    WithTimeout(5 * time.Second),
    WithRetry(3, 1 * time.Second),
    WithTracing(tracer),
)
```

### 高级使用
```go
// 完整的工具定义
calculator := tool.NewSchemaTool(
    "calculator",
    "Perform arithmetic operations",
    schema,
    handler,
).
    WithTimeout(5 * time.Second).
    WithRetry(3, 1 * time.Second).
    WithPermissions([]Permission{PermissionSystemInfo}).
    WithFormatter(NewJSONFormatter(true, 1000)).
    WithTracing(tracer)
```

---

## 总结

Actor Toolkit 提供：
1. **声明式工具定义**：用 Schema 替代手写验证
2. **自动类型转换**：智能处理 LLM 返回的参数
3. **执行包装器**：超时、重试、追踪
4. **结果格式化**：LLM 友好的输出
5. **安全机制**：权限控制、输入清理
6. **测试支持**：Mock 工具、测试工具

**核心价值：让开发者专注于业务逻辑，而非重复的基础设施代码。**
