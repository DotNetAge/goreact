# Critical Issues 修复进度

**修复日期**: 2026-03-09
**状态**: 进行中 (3/8 完成)

---

## ✅ 已完成修复

### Issue #1: Panic on Error in Memory Manager
**文件**: `pkg/memory/manager.go`
**修复内容**:
- 将 `NewDefaultMemoryManager` 改为返回 `(*DefaultMemoryManager, error)`
- 移除 `panic(err)`，改为返回错误
- 添加 `fmt` 包导入用于错误格式化

**验证**: ✅ 编译通过

---

### Issue #2: Goroutine Leak in Cache Cleanup
**文件**: `pkg/cache/memory.go`
**修复内容**:
- 添加 `context.Context` 和 `context.CancelFunc` 字段
- 在 `NewMemoryCache` 中创建可取消的 context
- 添加 `Close()` 方法用于停止清理协程
- 修改 `cleanupExpired()` 使用 select 监听 context 取消信号

**验证**: ✅ 编译通过

---

### Issue #3: API Key Exposure
**文件**: `pkg/llm/secure_string.go` (新建), `pkg/llm/openai/client.go`, `pkg/llm/anthropic/client.go`
**修复内容**:
- 创建 `SecureString` 类型包装敏感字符串
- 实现 `String()` 方法返回脱敏字符串（只显示前后4个字符）
- 实现 `Value()` 方法获取实际值
- 更新 OpenAI 和 Anthropic Client 使用 `SecureString` 存储 API Key
- 在需要使用时调用 `.Value()` 获取实际值

**验证**: ✅ 编译通过

---

## 🔄 待修复

### Issue #4: Race Condition in Token Usage
**文件**: `pkg/llm/ollama/client.go:123-129`
**问题**: `LastTokenUsage()` 返回可变指针，可能被并发修改
**修复方案**:
```go
func (c *OllamaClient) LastTokenUsage() *llm.TokenUsage {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.lastTokenUsage == nil {
        return nil
    }
    // 返回副本而不是指针
    copy := *c.lastTokenUsage
    return &copy
}
```

---

### Issue #5: Unsafe Concurrent Map Access in Metrics
**文件**: `pkg/metrics/metrics.go:384-426`
**问题**: `GetMetrics()` 在持有锁时调用嵌套对象的方法，可能死锁
**修复方案**:
```go
func (m *DefaultMetrics) GetMetrics() map[string]any {
    m.mutex.Lock()

    // 先复制所有引用
    latenciesCopy := make(map[string]*Timer, len(m.latencies))
    for k, v := range m.latencies {
        latenciesCopy[k] = v
    }
    // ... 其他字段类似

    m.mutex.Unlock()

    // 在锁外处理
    latencyMetrics := make(map[string]any)
    for operation, timer := range latenciesCopy {
        latencyMetrics[operation] = timer.GetMetrics()
    }
    // ...
}
```

---

### Issue #6: Unhandled Error in Logger Init
**文件**: `pkg/log/zap_logger.go:25-26`
**问题**: `config.Build()` 的错误被忽略
**修复方案**:
```go
func NewDefaultZapLogger() (*ZapLogger, error) {
    config := zap.NewProductionConfig()
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

    logger, err := config.Build()
    if err != nil {
        return nil, fmt.Errorf("failed to build logger: %w", err)
    }
    return &ZapLogger{logger: logger}, nil
}

// 同样修复 NewDevelopmentZapLogger
```

---

### Issue #7: Fragile JSON Parsing
**文件**: `pkg/core/thinker/simple_thinker.go:146-174`
**问题**: 手动字符串解析 JSON，不可靠且有安全风险
**修复方案**:
```go
func (t *SimpleThinker) parseSimpleJSON(jsonStr string) map[string]interface{} {
    var params map[string]interface{}
    if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
        // 记录错误但返回空 map，避免中断执行
        return make(map[string]interface{})
    }
    return params
}
```

---

### Issue #8: Missing Context Propagation
**文件**: `pkg/engine/engine.go:104`
**问题**: `Execute` 方法不支持 context 取消和超时
**修复方案**:
```go
// 修改签名
func (e *Engine) Execute(ctx context.Context, task string, execCtx *core.Context) *types.Result {
    // 在循环中检查 context
    for {
        select {
        case <-ctx.Done():
            return &types.Result{
                Success: false,
                Error:   ctx.Err(),
                EndTime: time.Now(),
            }
        default:
        }

        // ... 原有逻辑
    }
}

// 更新所有调用点
```

**注意**: 这个修改会影响所有调用 `Execute` 的地方，需要：
1. 更新所有示例代码
2. 更新测试代码
3. 更新文档

---

## 修复优先级

1. **Issue #4** (Race Condition) - 高优先级，影响并发安全
2. **Issue #5** (Unsafe Map Access) - 高优先级，可能死锁
3. **Issue #6** (Logger Error) - 中优先级，影响可靠性
4. **Issue #7** (JSON Parsing) - 中优先级，影响功能和安全
5. **Issue #8** (Context) - 低优先级，但需要大量修改

---

## 下一步行动

1. 继续修复 Issue #4-#8
2. 运行完整测试套件验证修复
3. 更新所有受影响的示例代码
4. 更新文档反映 API 变更
5. 进行 Phase 2 修复（Important Issues）

---

## 测试建议

修复完成后需要进行：
1. 单元测试 - 验证每个修复点
2. 并发测试 - 使用 `go test -race` 检测竞态条件
3. 集成测试 - 验证整体功能
4. 性能测试 - 确保修复没有引入性能问题
