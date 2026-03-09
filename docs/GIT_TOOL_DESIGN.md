# Git 工具设计文档

## 概述

Git 是编程场景中最高频的工具，几乎每个开发者每天都要使用数十次。我们需要提供一个 Schema-based 的 Git 工具，让 LLM 能够轻松地执行 Git 操作。

---

## 核心痛点

### 痛点 1：参数复杂，容易出错

**问题：**
```bash
# LLM 可能生成错误的命令
git clone https://github.com/user/repo.git /wrong/path
git commit -m "message" --author="wrong format"
git push origin master --force  # 危险操作
```

**解决方案：**
- Schema 验证参数
- 自动补全默认值
- 阻止危险操作

### 痛点 2：错误消息不友好

**问题：**
```
fatal: unable to access 'https://github.com/user/repo.git/':
Could not resolve host: github.com
```

**解决方案：**
```
❌ Failed to clone repository
Reason: Network connection failed
Suggestions:
1. Check your internet connection
2. Verify the repository URL
3. Check if GitHub is accessible
```

### 痛点 3：缺少安全控制

**问题：**
- `git push --force` 可能覆盖他人代码
- `git clean -fdx` 可能删除重要文件
- `git reset --hard` 可能丢失未提交的更改

**解决方案：**
- 危险操作需要确认
- 提供安全模式
- 记录所有操作

---

## 功能设计

### 1. 基础操作

#### 1.1 Clone（克隆仓库）

```go
git.Clone(url, path, options)
```

**参数：**
- `url` (string, required) - 仓库 URL
- `path` (string, optional) - 目标路径，默认为当前目录
- `branch` (string, optional) - 指定分支
- `depth` (int, optional) - 克隆深度（浅克隆）

**示例：**
```json
{
  "operation": "clone",
  "url": "https://github.com/user/repo.git",
  "path": "./repo",
  "branch": "main",
  "depth": 1
}
```

#### 1.2 Pull（拉取更新）

```go
git.Pull(path, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `remote` (string, optional) - 远程名称，默认 "origin"
- `branch` (string, optional) - 分支名称，默认当前分支
- `rebase` (bool, optional) - 是否使用 rebase

**示例：**
```json
{
  "operation": "pull",
  "path": "./repo",
  "remote": "origin",
  "branch": "main"
}
```

#### 1.3 Push（推送更改）

```go
git.Push(path, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `remote` (string, optional) - 远程名称，默认 "origin"
- `branch` (string, optional) - 分支名称，默认当前分支
- `force` (bool, optional) - 是否强制推送（危险，默认 false）

**示例：**
```json
{
  "operation": "push",
  "path": "./repo",
  "remote": "origin",
  "branch": "main"
}
```

#### 1.4 Commit（提交更改）

```go
git.Commit(path, message, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `message` (string, required) - 提交消息
- `files` ([]string, optional) - 要提交的文件，默认全部
- `amend` (bool, optional) - 是否修改上次提交

**示例：**
```json
{
  "operation": "commit",
  "path": "./repo",
  "message": "feat: add new feature",
  "files": ["src/main.go", "README.md"]
}
```

#### 1.5 Status（查看状态）

```go
git.Status(path)
```

**参数：**
- `path` (string, required) - 仓库路径

**返回：**
```json
{
  "branch": "main",
  "ahead": 2,
  "behind": 0,
  "modified": ["src/main.go"],
  "untracked": ["test.txt"],
  "staged": ["README.md"]
}
```

---

### 2. 分支管理

#### 2.1 Branch（分支操作）

```go
git.Branch(path, action, options)
```

**操作类型：**
- `list` - 列出所有分支
- `create` - 创建新分支
- `delete` - 删除分支
- `rename` - 重命名分支

**示例：**
```json
{
  "operation": "branch",
  "path": "./repo",
  "action": "create",
  "name": "feature/new-feature"
}
```

#### 2.2 Checkout（切换分支）

```go
git.Checkout(path, branch, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `branch` (string, required) - 分支名称
- `create` (bool, optional) - 如果不存在则创建

**示例：**
```json
{
  "operation": "checkout",
  "path": "./repo",
  "branch": "feature/new-feature",
  "create": true
}
```

#### 2.3 Merge（合并分支）

```go
git.Merge(path, branch, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `branch` (string, required) - 要合并的分支
- `no_ff` (bool, optional) - 禁用快进合并

**示例：**
```json
{
  "operation": "merge",
  "path": "./repo",
  "branch": "feature/new-feature"
}
```

---

### 3. 历史查看

#### 3.1 Log（查看提交历史）

```go
git.Log(path, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `limit` (int, optional) - 限制数量，默认 10
- `author` (string, optional) - 按作者过滤
- `since` (string, optional) - 起始日期
- `until` (string, optional) - 结束日期

**返回：**
```json
{
  "commits": [
    {
      "hash": "abc123",
      "author": "John Doe",
      "date": "2026-03-08T10:00:00Z",
      "message": "feat: add new feature"
    }
  ]
}
```

#### 3.2 Diff（查看差异）

```go
git.Diff(path, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `file` (string, optional) - 指定文件
- `staged` (bool, optional) - 查看暂存区差异

**示例：**
```json
{
  "operation": "diff",
  "path": "./repo",
  "file": "src/main.go"
}
```

#### 3.3 Show（查看提交详情）

```go
git.Show(path, commit)
```

**参数：**
- `path` (string, required) - 仓库路径
- `commit` (string, required) - 提交哈希

---

### 4. 远程管理

#### 4.1 Remote（远程仓库操作）

```go
git.Remote(path, action, options)
```

**操作类型：**
- `list` - 列出远程仓库
- `add` - 添加远程仓库
- `remove` - 删除远程仓库
- `set-url` - 修改远程仓库 URL

**示例：**
```json
{
  "operation": "remote",
  "path": "./repo",
  "action": "add",
  "name": "upstream",
  "url": "https://github.com/original/repo.git"
}
```

#### 4.2 Fetch（获取远程更新）

```go
git.Fetch(path, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `remote` (string, optional) - 远程名称，默认 "origin"
- `prune` (bool, optional) - 删除不存在的远程分支

---

### 5. 其他操作

#### 5.1 Add（添加到暂存区）

```go
git.Add(path, files)
```

**参数：**
- `path` (string, required) - 仓库路径
- `files` ([]string, required) - 要添加的文件
- `all` (bool, optional) - 添加所有文件

#### 5.2 Reset（重置）

```go
git.Reset(path, options)
```

**参数：**
- `path` (string, required) - 仓库路径
- `mode` (string, optional) - 模式：soft, mixed, hard
- `commit` (string, optional) - 目标提交

**⚠️ 危险操作，需要确认**

#### 5.3 Stash（暂存更改）

```go
git.Stash(path, action, options)
```

**操作类型：**
- `save` - 保存当前更改
- `list` - 列出所有暂存
- `apply` - 应用暂存
- `pop` - 应用并删除暂存
- `drop` - 删除暂存

---

## Schema 定义

```go
var gitSchema = schema.Define(
    // 操作类型
    schema.Param("operation", schema.String, "Git operation").
        Enum(
            "clone", "pull", "push", "commit", "status",
            "branch", "checkout", "merge",
            "log", "diff", "show",
            "remote", "fetch",
            "add", "reset", "stash",
        ).
        Required(),

    // 仓库路径
    schema.Param("path", schema.String, "Repository path").
        RequiredExcept("operation", "clone"),

    // Clone 参数
    schema.Param("url", schema.String, "Repository URL").
        RequiredIf("operation", "clone"),

    // Commit 参数
    schema.Param("message", schema.String, "Commit message").
        RequiredIf("operation", "commit"),

    // Branch/Checkout 参数
    schema.Param("branch", schema.String, "Branch name"),

    // 通用选项
    schema.Param("force", schema.Boolean, "Force operation").
        Default(false),
)
```

---

## 安全控制

### 1. 危险操作列表

```go
var dangerousOperations = map[string]bool{
    "push --force":  true,
    "reset --hard":  true,
    "clean -fdx":    true,
    "branch -D":     true,
}
```

### 2. 安全模式

```go
type GitTool struct {
    safeMode bool  // 安全模式，阻止危险操作
    dryRun   bool  // 演习模式，不实际执行
}
```

### 3. 操作日志

```go
type GitOperation struct {
    Timestamp time.Time
    Operation string
    Path      string
    Success   bool
    Output    string
}
```

---

## 错误处理

### 1. 网络错误

```go
if strings.Contains(err.Error(), "Could not resolve host") {
    return schema.NewUserError(
        "Network connection failed\n"+
        "Suggestions:\n"+
        "1. Check your internet connection\n"+
        "2. Verify the repository URL\n"+
        "3. Check if the Git server is accessible",
    )
}
```

### 2. 认证错误

```go
if strings.Contains(err.Error(), "Authentication failed") {
    return schema.NewUserError(
        "Git authentication failed\n"+
        "Suggestions:\n"+
        "1. Check your Git credentials\n"+
        "2. Verify SSH key is configured\n"+
        "3. Use HTTPS with personal access token",
    )
}
```

### 3. 冲突错误

```go
if strings.Contains(err.Error(), "CONFLICT") {
    return schema.NewUserError(
        "Merge conflict detected\n"+
        "Suggestions:\n"+
        "1. Resolve conflicts manually\n"+
        "2. Use 'git status' to see conflicted files\n"+
        "3. After resolving, use 'git add' and 'git commit'",
    )
}
```

---

## 使用示例

### 示例 1：克隆仓库

```go
result, err := gitTool.Execute(map[string]any{
    "operation": "clone",
    "url":       "https://github.com/user/repo.git",
    "path":      "./repo",
    "depth":     1,
})
```

### 示例 2：提交更改

```go
// 1. 添加文件
gitTool.Execute(map[string]any{
    "operation": "add",
    "path":      "./repo",
    "files":     []string{"src/main.go"},
})

// 2. 提交
gitTool.Execute(map[string]any{
    "operation": "commit",
    "path":      "./repo",
    "message":   "feat: add new feature",
})

// 3. 推送
gitTool.Execute(map[string]any{
    "operation": "push",
    "path":      "./repo",
})
```

### 示例 3：创建分支并切换

```go
gitTool.Execute(map[string]any{
    "operation": "checkout",
    "path":      "./repo",
    "branch":    "feature/new-feature",
    "create":    true,
})
```

---

## 测试策略

### 1. 单元测试

```go
func TestGitClone(t *testing.T) {
    tool := NewGitTool()

    result, err := tool.Execute(map[string]any{
        "operation": "clone",
        "url":       "https://github.com/test/repo.git",
        "path":      t.TempDir(),
    })

    assert.NoError(t, err)
    assert.True(t, result.(map[string]any)["success"].(bool))
}
```

### 2. 集成测试

```go
func TestGitWorkflow(t *testing.T) {
    // 1. Clone
    // 2. Create branch
    // 3. Make changes
    // 4. Commit
    // 5. Push
}
```

---

## 实现优先级

### P0（立即实现）
- ✅ clone, pull, push, commit, status
- ✅ branch, checkout
- ✅ add

### P1（短期）
- merge, log, diff
- remote, fetch

### P2（中期）
- show, reset, stash
- 高级功能

---

## 总结

### 核心价值
1. **Schema 验证** - 自动验证参数，减少错误
2. **友好错误** - LLM 能理解的错误消息
3. **安全控制** - 阻止危险操作
4. **高频操作** - 覆盖 90% 的日常使用场景

### 记住
> Git 是编程场景最高频的工具，必须做到：
> - 参数验证严格
> - 错误消息友好
> - 安全控制完善
> - 操作日志完整
