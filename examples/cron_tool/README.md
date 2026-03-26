# Cron Tool Example

这个示例展示了如何使用 GoReAct 的 Cron工具来解析和计算 cron 表达式。

## 运行示例

```bash
cd /Users/ray/workspaces/ai-ecosystem/goreact/examples/cron_tool
go run cron_example.go
```

## Cron工具功能

Cron工具提供以下三种操作：

### 1. Validate - 验证 Cron 表达式

验证 cron 表达式是否有效。

```go
result, err := cron.Execute(ctx, map[string]any{
    "operation":  "validate",
    "expression": "*/5 * * * *",
})
```

返回结果：
```json
{
    "valid": true,
    "fields": [[0,5,10,...], [0,1,2,...], ...]
}
```

### 2. Parse - 解析 Cron 表达式

解析 cron 表达式的各个字段。

```go
result, err := cron.Execute(ctx, map[string]any{
    "operation":  "parse",
    "expression": "0 12 * * *",
})
```

返回结果：
```json
{
    "expression": "0 12 * * *",
    "minute": "0",
    "hour": "12",
    "day": "*",
    "month": "*",
    "weekday": "*"
}
```

### 3. Next - 计算下一个执行时间

计算从指定时间开始的下一个或多个执行时间。

```go
result, err := cron.Execute(ctx, map[string]any{
    "operation":  "next",
    "expression": "0 9 * * 1-5",
    "from":       "2026-03-19T10:00:00Z",
    "count":      5.0,
})
```

返回结果：
```json
[
    "2026-03-20T09:00:00Z",
    "2026-03-23T09:00:00Z",
    "2026-03-24T09:00:00Z",
    "2026-03-25T09:00:00Z",
    "2026-03-26T09:00:00Z"
]
```

## Cron 表达式格式

Cron 表达式由 5 个字段组成：

```
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日期 (1 - 31)
│ │ │ ┌───────────── 月份 (1 - 12)
│ │ │ │ ┌───────────── 星期几 (0 - 6, 0 = 星期日)
│ │ │ │ │
* * * * *
```

## 特殊字符

- `*` - 通配符，表示"每个"值
- `,` - 分隔符，列出多个值（如 `1,3,5`）
- `-` - 范围，指定一个范围（如 `1-5`）
- `/` - 步长，指定步长（如 `*/5` 或 `0-30/5`）

## 示例表达式

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

## 在 Agent 中使用

将 Cron工具注册到工具管理器：

```go
import (
    "github.com/DotNetAge/goreact/pkg/tools"
    "github.com/DotNetAge/goreact/pkg/tools/builtin"
)

toolMgr := tools.NewSimpleManager()
toolMgr.Register(builtin.NewCron())
```

然后 Agent 可以通过自然语言调用它：

```
用户："帮我计算每天早上 9 点接下来 5 次的执行时间"
Agent: [调用 cron工具，operation="next", expression="0 9 * * *", count=5]
```
