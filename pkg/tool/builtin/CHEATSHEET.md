# GoReAct 内置工具速查表

## 📋 工具清单 (9 个)

### Tier 1 - 文件操作 (3 个)
| 工具 | 功能 | 参数 | 示例 |
|------|------|------|------|
| **Read** | 读取文件 | `path`, `start_line?`, `end_line?` | `read(path="main.go", start_line=10)` |
| **Write** | 写入文件 | `path`, `content`, `append?` | `write(path="out.txt", content="data")` |
| **Edit** | 精确编辑 | `path`, `edits: [{old_text, new_text}]` | `edit(path="app.js", edits=[...])` |

### Tier 2 - 搜索浏览 (3 个)
| 工具 | 功能 | 参数 | 示例 |
|------|------|------|------|
| **Glob** | 文件名匹配 | `pattern`, `path?` | `glob(pattern="*_test.go")` |
| **Grep** | 文本搜索 | `pattern`, `path?`, `include?` | `grep(pattern="TODO", include="*.go")` |
| **LS** | 目录列表 | `path?`, `recursive?`, `show_hidden?` | `ls(path=".", recursive=true)` |

### Tier 3 - 执行扩展 (3 个)
| 工具 | 功能 | 参数 | 示例 |
|------|------|------|------|
| **Bash** | Shell 执行 | `command` | `bash(command="go test ./...")` |
| **Calculator** | 数学计算 | `operation`, `a`, `b` | `calculator(operation="add", a=10, b=5)` |
| **DateTime** | 日期时间 | `operation`, `time?`, `format?` | `datetime(operation="now")` |

---

## 🎯 常用组合

### 代码重构
```
Glob("*.go") → Grep("OldFunc") → Edit(edits=[...])
```

### 项目分析
```
LS(recursive=true) → Glob("*.md") → Read(path=...)
```

### 自动化测试
```
Bash("go test -v") → Read("coverage.out") → Grep("FAIL")
```

---

## 🔒 安全限制

- ❌ 禁止访问：`passwd`, `shadow`, `sudoers`
- 📏 文件大小：读取≤10MB，搜索≤5MB
- 👁️ 隐藏文件：自动跳过（除非 `show_hidden=true`）

---

## 📊 返回值格式

所有工具返回统一的 JSON 格式：
```json
{
  "success": true/false,
  "message": "操作描述",
  // ... 工具特定字段
}
```

---

## 💡 最佳实践

✅ **推荐**：
- 使用 `Edit` 而非 `Read+Write` 组合
- 使用 `include` 参数限定 Grep 范围
- 大文件先 `LS` 查看大小再 `Read`

❌ **避免**：
- 在循环中频繁调用 Read/Write
- 全局搜索不使用路径限定
- 修改系统文件

---

## 🚀 快速开始

```go
import "github.com/ray/goreact/pkg/tool/builtin"

// 创建工具实例
read := builtin.NewRead()
write := builtin.NewWrite()
edit := builtin.NewEdit()
glob := builtin.NewGlob()
grep := builtin.NewGrep()
ls := builtin.NewLS()
bash := builtin.NewBash()
calc := builtin.NewCalculator()
dt := builtin.NewDateTime()

// 使用工具
result := read.Execute(map[string]interface{}{
    "path": "main.go",
})
```

---

**完整文档**: 参见 [USAGE_GUIDE.md](./USAGE_GUIDE.md)
