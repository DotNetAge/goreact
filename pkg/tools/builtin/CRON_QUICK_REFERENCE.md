# Cron工具快速参考

## 🚀 快速开始

```go
import "github.com/DotNetAge/goreact/pkg/tools/builtin"

cron := builtin.NewCron()
```

## 📋 三种操作

### 1️⃣ Validate - 验证表达式
```go
result, _ := cron.Execute(ctx, map[string]any{
    "operation": "validate",
    "expression": "*/5 * * * *",
})
// → {"valid": true}
```

### 2️⃣ Parse - 解析表达式
```go
result, _ := cron.Execute(ctx, map[string]any{
    "operation": "parse",
    "expression": "0 12 * * *",
})
// → {"minute": "0", "hour": "12", "day": "*", ...}
```

### 3️⃣ Next - 计算执行时间
```go
result, _ := cron.Execute(ctx, map[string]any{
    "operation": "next",
    "expression": "0 9 * * 1-5",
    "from": "2026-03-19T10:00:00Z",
    "count": 5.0,
})
// → ["2026-03-20T09:00:00Z", ...]
```

## 🕐 Cron 格式

```
┌───── 分钟 (0-59)
│ ┌───── 小时 (0-23)
│ │ ┌───── 日期 (1-31)
│ │ │ ┌───── 月份 (1-12)
│ │ │ │ ┌───── 星期 (0-6)
│ │ │ │ │
* * * * *
```

## 🔤 特殊符号

| 符号 | 含义 | 示例 |
|------|------|------|
| `*` | 每个值 | `* * * * *` 每分钟 |
| `,` | 列举多个值 | `0,15,30,45 * * * *` |
| `-` | 范围 | `0 9-17 * * *` 朝九晚五 |
| `/` | 步长 | `*/5 * * * *` 每 5 分钟 |

## 📝 常用表达式

| 表达式 | 含义 |
|--------|------|
| `* * * * *` | 每分钟 |
| `0 * * * *` | 每小时整点 |
| `0 12 * * *` | 每天中午 12 点 |
| `0 9 * * 1-5` | 工作日早上 9 点 |
| `*/5 * * * *` | 每 5 分钟 |
| `0 0 1 * *` | 每月 1 号午夜 |
| `0 0 * * 0` | 每周日午夜 |
| `*/15 9-17 * * 1-5` | 工作日 9:00-17:00 每 15 分钟 |

## ✅ 测试运行

```bash
cd /Users/ray/workspaces/ai-ecosystem/goreact
go test ./pkg/tools/builtin -v -run TestCron
```

## 💻 示例运行

```bash
cd /Users/ray/workspaces/ai-ecosystem/goreact/examples/cron_tool
go run cron_example.go
```

## 🎯 Agent 集成

```go
toolMgr := tools.NewSimpleManager()
toolMgr.Register(builtin.NewCron())
```

Agent 可以这样使用：
- "每天早上 9 点的闹钟，接下来 5 次是几点？"
- "`*/15 * * * *` 这个表达式有效吗？"
- "解释一下 `0 12 * * 1-5` 的含义"
