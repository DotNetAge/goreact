# Replace 工具使用指南

## 📋 功能概述

`Replace` 工具用于在文件中查找并替换文本内容，支持:
- ✅ 全局替换（所有匹配项）
- ✅ 限定次数替换（只替换前 N 个匹配项）
- ✅ 详细的执行报告和统计信息

## 🎯 工具参数

```json
{
  "path": "文件路径 (必需)",
  "search": "要查找的文本 (必需)",
  "replace": "替换为的文本 (必需)",
  "limit": "最大替换次数 (可选，-1 表示全部替换)"
}
```

### 参数说明

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `path` | string | ✅ | - | 目标文件的绝对或相对路径 |
| `search` | string | ✅ | - | 要查找和替换的原始文本 |
| `replace` | string | ✅ | - | 替换后的新文本 |
| `limit` | number | ❌ | -1 | 最大替换次数，-1 表示替换所有匹配项 |

## 💡 使用示例

### 示例 1: 全局替换

将所有出现的 "TODO" 替换为 "FIXME":

```json
{
  "path": "./main.go",
  "search": "TODO",
  "replace": "FIXME"
}
```

**返回结果**:
```json
{
  "success": true,
  "path": "./main.go",
  "search": "TODO",
  "replace": "FIXME",
  "matches_found": 5,
  "replacements": 5,
  "original_size": 1024,
  "new_size": 1034,
  "size_delta": 10,
  "message": "Successfully replaced 5 occurrence(s) of 'TODO'"
}
```

### 示例 2: 限定次数替换

只替换前 3 个匹配的 "old_function" 为 "new_function":

```json
{
  "path": "./utils.js",
  "search": "old_function",
  "replace": "new_function",
  "limit": 3
}
```

**返回结果**:
```json
{
  "success": true,
  "path": "./utils.js",
  "search": "old_function",
  "replace": "new_function",
  "matches_found": 10,
  "replacements": 3,
  "original_size": 2048,
  "new_size": 2063,
  "size_delta": 15,
  "message": "Successfully replaced 3 occurrence(s) of 'old_function'"
}
```

### 示例 3: 未找到匹配文本

```json
{
  "path": "./config.yaml",
  "search": "not_found_key",
  "replace": "new_value"
}
```

**返回结果**:
```json
{
  "success": false,
  "path": "./config.yaml",
  "replacements": 0,
  "message": "Text 'not_found_key' not found in file"
}
```

## 🔧 实际应用场景

### 场景 1: 批量更新配置项

```json
{
  "path": "./config.json",
  "search": "\"api_version\": \"v1\"",
  "replace": "\"api_version\": \"v2\""
}
```

### 场景 2: 修复拼写错误

```json
{
  "path": "./README.md",
  "search": "funtion",
  "replace": "function"
}
```

### 场景 3: 更新依赖版本

```json
{
  "path": "./go.mod",
  "search": "github.com/example/pkg v1.0.0",
  "replace": "github.com/example/pkg v2.0.0",
  "limit": 1
}
```

### 场景 4: 多语言文本替换

```json
{
  "path": "./messages.go",
  "search": "Hello, World!",
  "replace": "你好，世界!"
}
```

## 📊 返回值字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | boolean | 操作是否成功 |
| `path` | string | 处理的文件路径 |
| `search` | string | 查找的文本（截断） |
| `replace` | string | 替换的文本（截断） |
| `matches_found` | int | 找到的匹配总数 |
| `replacements` | int | 实际执行的替换数量 |
| `original_size` | int | 原始文件大小（字节） |
| `new_size` | int | 修改后的文件大小（字节） |
| `size_delta` | int | 文件大小变化（正数表示增加） |
| `message` | string | 人类可读的结果描述 |

## ⚠️ 注意事项

1. **备份重要文件**: 替换操作会直接修改文件内容，建议先备份
2. **精确匹配**: 使用足够具体的搜索文本，避免意外替换
3. **大小写敏感**: 当前实现是大小写敏感的
4. **特殊字符**: 搜索文本中的特殊字符需要正确转义
5. **大文件**: 对于非常大的文件，建议先用 `grep` 工具预览匹配结果

## 🔒 安全特性

- ✅ 文件路径安全检查
- ✅ 系统文件保护（不能修改 `/etc/passwd` 等敏感文件）
- ✅ 操作可追溯（返回详细统计信息）

## 🆚 与 Edit 工具的区别

| 特性 | Replace | Edit |
|------|---------|------|
| 替换方式 | 基于字符串匹配 | 基于原文本块 |
| 适用场景 | 简单、重复的文本替换 | 复杂、结构化的代码编辑 |
| 精确度 | 全局/限定次数 | 精确定位特定位置 |
| 安全性 | 中等（可能误匹配） | 高（精确匹配） |

## 💻 Go 代码调用示例

```go
import (
    "context"
    "github.com/DotNetAge/goreact/pkg/tools/builtin"
)

// 创建工具实例
replace := builtin.NewReplace()

// 执行替换
result, err := replace.Execute(context.Background(), map[string]any{
    "path":    "./main.go",
    "search":  "fmt.Println",
    "replace": "log.Println",
    "limit":   -1, // 全部替换
})

if err != nil {
    log.Fatalf("Replace failed: %v", err)
}

fmt.Printf("Replace result: %+v\n", result)
```

---

**相关工具**: 
- [`Edit`](./USAGE_GUIDE.md#edit) - 精确的多位置编辑器
- [`Read`](./USAGE_GUIDE.md#read) - 文件读取工具
- [`Write`](./USAGE_GUIDE.md#write) - 文件写入工具
- [`Grep`](./USAGE_GUIDE.md#grep) - 文本搜索工具
