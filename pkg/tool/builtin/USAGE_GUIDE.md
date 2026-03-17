# GoReAct 新工具集使用指南

## 📚 工具分类

### Tier 1: 文件操作三剑客

#### 1. Read - 读取文件
```go
// 基本用法
result := read.Execute(map[string]interface{}{
    "path": "/path/to/file.txt",
})

// 读取指定行范围
result := read.Execute(map[string]interface{}{
    "path":      "/path/to/file.txt",
    "start_line": 10,
    "end_line":   20,
})
```

**输出示例**：
```json
{
  "success": true,
  "path": "/path/to/file.txt",
  "size_bytes": 1024,
  "lines_read": 11,
  "total_lines": 100,
  "content": "1\tLine 1\n2\tLine 2\n..."
}
```

#### 2. Write - 写入文件
```go
// 覆盖写入
result := write.Execute(map[string]interface{}{
    "path":    "/path/to/file.txt",
    "content": "New content here",
})

// 追加模式
result := write.Execute(map[string]interface{}{
    "path":    "/path/to/file.txt",
    "content": "Appended content",
    "append":  true,
})
```

**输出示例**：
```json
{
  "success": true,
  "path": "/path/to/file.txt",
  "mode": "overwrite",
  "bytes_written": 16,
  "total_size": 1024,
  "message": "File written successfully"
}
```

#### 3. Edit - 精确编辑
```go
// 单处编辑
result := edit.Execute(map[string]interface{}{
    "path": "/path/to/file.txt",
    "edits": []interface{}{
        map[string]interface{}{
            "old_text": "old string",
            "new_text": "new string",
        },
    },
})

// 多处编辑
result := edit.Execute(map[string]interface{}{
    "path": "/path/to/file.txt",
    "edits": []interface{}{
        map[string]interface{}{
            "old_text": "func Foo()",
            "new_text": "func Bar()",
        },
        map[string]interface{}{
            "old_text": "Foo()",
            "new_text": "Bar()",
        },
    },
})
```

**输出示例**：
```json
{
  "success": true,
  "path": "/path/to/file.txt",
  "edits_applied": 2,
  "original_size": 1024,
  "new_size": 1050,
  "size_delta": 26,
  "edited_regions": [
    {"index": 0, "old_length": 10, "new_length": 10, "delta": 0},
    {"index": 1, "old_length": 5, "new_length": 5, "delta": 0}
  ],
  "message": "Successfully applied 2 edit(s)"
}
```

---

### Tier 2: 搜索铁三角

#### 4. Glob - 文件名模式匹配
```go
// 查找所有 Go 文件
result := glob.Execute(map[string]interface{}{
    "pattern": "*.go",
})

// 在指定目录查找
result := glob.Execute(map[string]interface{}{
    "pattern": "*.md",
    "path":    "/docs",
})

// 复杂模式
result := glob.Execute(map[string]interface{}{
    "pattern": "*_test.go",
    "path":    "./pkg",
})
```

**输出示例**：
```json
{
  "success": true,
  "pattern": "*.go",
  "search_path": ".",
  "matches_found": 15,
  "files": ["main.go", "utils/helper.go", ...],
  "message": "Found 15 file(s) matching '*.go'"
}
```

#### 5. Grep - 文本内容搜索
```go
// 基本搜索
result := grep.Execute(map[string]interface{}{
    "pattern": "TODO",
})

// 正则表达式搜索
result := grep.Execute(map[string]interface{}{
    "pattern": "func\\s+\\w+\\(",
})

// 限定文件类型
result := grep.Execute(map[string]interface{}{
    "pattern": "import",
    "include": "*.go",
})
```

**输出示例**：
```json
{
  "success": true,
  "pattern": "TODO",
  "search_path": ".",
  "files_searched": 50,
  "matches_found": 12,
  "matches": [
    {
      "file": "main.go",
      "line": 42,
      "content": "// TODO: implement this",
      "match": "TODO",
      "start_col": 3,
      "end_col": 7
    }
  ],
  "message": "Found 12 match(es) in 50 file(s)"
}
```

#### 6. LS - 列出目录内容
```go
// 基本列表
result := ls.Execute(map[string]interface{}{
    "path": "./pkg",
})

// 递归模式（包含子目录）
result := ls.Execute(map[string]interface{}{
    "path":      "./pkg",
    "recursive": true,
})

// 显示隐藏文件
result := ls.Execute(map[string]interface{}{
    "path":       ".",
    "show_hidden": true,
})
```

**输出示例**：
```json
{
  "success": true,
  "path": "./pkg",
  "total_items": 8,
  "items": [
    {
      "name": "tool",
      "type": "directory",
      "size": 4096,
      "modTime": "2024-01-01 12:00:00",
      "mode": "drwxr-xr-x",
      "children": [...]
    },
    {
      "name": "main.go",
      "type": "file",
      "size": 1024,
      "modTime": "2024-01-01 12:00:00",
      "mode": "-rw-r--r--"
    }
  ],
  "message": "Listed 8 item(s) in './pkg'"
}
```

---

### Tier 3: 执行扩展

#### 7. Bash - Shell 命令执行
```go
// 基本命令
result := bash.Execute(map[string]interface{}{
    "command": "ls -la",
})

// 管道命令
result := bash.Execute(map[string]interface{}{
    "command": "grep -r \"TODO\" . | head -n 10",
})

// 多命令组合
result := bash.Execute(map[string]interface{}{
    "command": "cd /tmp && echo 'test' > file.txt && cat file.txt",
})
```

**输出示例**：
```json
{
  "success": true,
  "command": "ls -la",
  "output": "total 48\ndrwxr-xr-x  5 user staff  160 Jan  1 12:00 .\n...",
  "exit_code": 0
}
```

#### 8. Calculator - 数学计算
```go
// 加法
result := calculator.Execute(map[string]interface{}{
    "operation": "add",
    "a":         10,
    "b":         5,
})

// 乘法
result := calculator.Execute(map[string]interface{}{
    "operation": "multiply",
    "a":         25,
    "b":         4,
})
```

**输出示例**：
```json
{
  "success": true,
  "operation": "add",
  "result": 15
}
```

#### 9. DateTime - 日期时间
```go
// 获取当前时间
result := datetime.Execute(map[string]interface{}{
    "operation": "now",
})

// 格式化时间
result := datetime.Execute(map[string]interface{}{
    "operation": "format",
    "time":      "2024-01-01 12:00:00",
    "format":    "Monday, January 2, 2006",
})
```

---

## 🎯 组合使用示例

### 示例 1: 代码重构助手
```go
// 1. 查找所有 Go 文件
globResult := glob.Execute(map[string]interface{}{
    "pattern": "*.go",
})

// 2. 搜索需要修改的函数
grepResult := grep.Execute(map[string]interface{}{
    "pattern": "func OldFunctionName",
    "include": "*.go",
})

// 3. 逐个文件编辑
for _, match := range grepResult["matches"].([]map[string]interface{}) {
    edit.Execute(map[string]interface{}{
        "path": match["file"].(string),
        "edits": []interface{}{
            map[string]interface{}{
                "old_text": "OldFunctionName",
                "new_text": "NewFunctionName",
            },
        },
    })
}
```

### 示例 2: 项目分析工具
```go
// 1. 列出项目结构
lsResult := ls.Execute(map[string]interface{}{
    "path":      ".",
    "recursive": true,
})

// 2. 统计代码行数
globResult := glob.Execute(map[string]interface{}{
    "pattern": "*.go",
})

totalLines := 0
for _, file := range globResult["files"].([]string) {
    readResult := read.Execute(map[string]interface{}{
        "path": file,
    })
    // 解析 result 计算行数
}

// 3. 搜索 TODO 注释
todoResult := grep.Execute(map[string]interface{}{
    "pattern": "TODO|FIXME",
    "include": "*.go",
})
```

### 示例 3: 自动化测试脚本
```bash
# 使用 bash 执行测试
bash.Execute(map[string]interface{}{
    "command": "go test ./... -coverprofile=coverage.out",
})

# 读取覆盖率报告
read.Execute(map[string]interface{}{
    "path": "coverage.out",
})

# 搜索低覆盖率的函数
grep.Execute(map[string]interface{}{
    "pattern": "\\d\\.\\d%.*\\d+$",
})
```

---

## 🔒 安全性说明

### 文件访问限制
- 特殊文件保护：`passwd`, `shadow`, `sudoers` 等系统文件禁止访问
- 文件大小限制：读取最大 10MB，搜索最大 5MB
- 路径安全检查：防止目录穿越攻击

### 最佳实践
1. **最小权限原则**：只授予必要的文件访问权限
2. **输入验证**：所有参数都经过严格验证
3. **错误处理**：详细的错误消息但不泄露敏感信息
4. **日志记录**：记录所有工具调用用于审计

---

## 📊 性能优化建议

### 批量操作
- 使用 `Edit` 工具一次性完成多处修改
- 避免频繁的 Read/Write 循环

### 搜索优化
- 使用 `include` 参数限定文件类型
- 使用具体的路径而非全局搜索
- 大文件自动跳过（>5MB）

### 内存友好
- `Read` 工具支持行范围读取
- `Grep` 工具逐行扫描，不加载整个文件
