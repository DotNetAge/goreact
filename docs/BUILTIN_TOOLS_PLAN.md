# GoReAct 内置工具实现计划

## 设计理念

> 不是实现所有系统命令，而是封装高频场景的工具，提供 Schema 验证和友好接口。

### 为什么不实现所有系统命令？

1. **已有通用方案** - `bash` 工具可以执行任意命令
2. **维护成本高** - 跨平台兼容性、API 变化
3. **安全风险大** - 需要复杂的权限控制

### 应该实现什么？

**高频场景的专用工具：**
- ✅ Schema-based 参数验证
- ✅ 友好的错误消息
- ✅ 跨平台兼容
- ✅ 安全控制

---

## Phase 1: 编程工具（最高频）

### 1.1 Git 工具 ⭐⭐⭐

**使用频率：** 极高（每天数十次）

**核心操作：**
```go
// 基础操作
git.Clone(url, path)
git.Pull(path)
git.Push(path, branch)
git.Commit(path, message)
git.Status(path)

// 分支管理
git.Branch(path, action) // list, create, delete, switch
git.Checkout(path, branch)
git.Merge(path, branch)

// 查看历史
git.Log(path, limit)
git.Diff(path, file)
git.Show(path, commit)

// 远程管理
git.Remote(path, action) // list, add, remove
git.Fetch(path, remote)
```

**Schema 示例：**
```go
schema.NewTool(
    "git",
    "Git version control operations",
    schema.Define(
        schema.Param("operation", schema.String, "Git operation").
            Enum("clone", "pull", "push", "commit", "status", "branch", "log").
            Required(),
        schema.Param("path", schema.String, "Repository path").
            Required(),
        schema.Param("message", schema.String, "Commit message").
            RequiredIf("operation", "commit"),
        schema.Param("url", schema.String, "Repository URL").
            RequiredIf("operation", "clone"),
    ),
    gitHandler,
)
```

**优先级：** P0（立即实现）

---

### 1.2 代码分析工具

**使用频率：** 高（每天数次）

**核心操作：**
```go
// Linting
lint.Run(path, linter) // golangci-lint, eslint, pylint

// Formatting
format.Run(path, formatter) // gofmt, prettier, black

// Testing
test.Run(path, pattern) // go test, npm test, pytest
```

**优先级：** P1

---

### 1.3 包管理工具

**使用频率：** 中高（每天数次）

**核心操作：**
```go
// Go
gomod.Tidy(path)
gomod.Download(path)
gomod.Vendor(path)

// Node.js
npm.Install(path, package)
npm.Build(path)
npm.Test(path)

// Python
pip.Install(package, version)
pip.Freeze(path)
```

**优先级：** P1

---

## Phase 2: Docker 工具（DevOps 必备）

### 2.1 容器管理 ⭐⭐

**使用频率：** 高（DevOps 场景）

**核心操作：**
```go
// 容器生命周期
docker.Run(image, options)
docker.Stop(container)
docker.Start(container)
docker.Restart(container)
docker.Rm(container)

// 容器信息
docker.Ps(filters)
docker.Logs(container, tail)
docker.Inspect(container)
docker.Stats(container)

// 容器交互
docker.Exec(container, command)
docker.Cp(container, src, dst)
```

**Schema 示例：**
```go
schema.NewTool(
    "docker",
    "Docker container management",
    schema.Define(
        schema.Param("operation", schema.String, "Docker operation").
            Enum("run", "stop", "ps", "logs", "exec", "rm").
            Required(),
        schema.Param("container", schema.String, "Container ID or name").
            RequiredIf("operation", "stop", "logs", "exec"),
        schema.Param("image", schema.String, "Image name").
            RequiredIf("operation", "run"),
    ),
    dockerHandler,
)
```

**优先级：** P0（Docker 是 DevOps 核心）

---

### 2.2 镜像管理

**核心操作：**
```go
docker.Images(filters)
docker.Pull(image, tag)
docker.Push(image, tag)
docker.Build(path, tag)
docker.Rmi(image)
docker.Tag(source, target)
```

**优先级：** P1

---

### 2.3 网络和卷

**核心操作：**
```go
// 网络
docker.NetworkLs()
docker.NetworkCreate(name, driver)
docker.NetworkRm(name)

// 卷
docker.VolumeLs()
docker.VolumeCreate(name)
docker.VolumeRm(name)
```

**优先级：** P2

---

## Phase 3: 办公工具（高频）

### 3.1 邮件工具 ⭐⭐⭐

**使用频率：** 极高（办公场景最高频）

**核心操作：**
```go
// 发送邮件
email.Send(to, subject, body, attachments)
email.SendHTML(to, subject, html, attachments)

// 读取邮件
email.List(folder, limit)
email.Read(messageId)
email.Search(query, folder)

// 邮件管理
email.Delete(messageId)
email.Move(messageId, folder)
email.MarkAsRead(messageId)
```

**Schema 示例：**
```go
schema.NewTool(
    "email",
    "Email operations (send, read, search)",
    schema.Define(
        schema.Param("operation", schema.String, "Email operation").
            Enum("send", "list", "read", "search", "delete").
            Required(),
        schema.Param("to", schema.String, "Recipient email").
            RequiredIf("operation", "send"),
        schema.Param("subject", schema.String, "Email subject").
            RequiredIf("operation", "send"),
        schema.Param("body", schema.String, "Email body").
            RequiredIf("operation", "send"),
        schema.Param("folder", schema.String, "Email folder").
            Default("INBOX"),
    ),
    emailHandler,
)
```

**配置：**
```go
type EmailConfig struct {
    SMTP struct {
        Host     string
        Port     int
        Username string
        Password string
        TLS      bool
    }
    IMAP struct {
        Host     string
        Port     int
        Username string
        Password string
        TLS      bool
    }
}
```

**优先级：** P0（办公场景最高频）

---

### 3.2 文档处理

**使用频率：** 中高

**核心操作：**
```go
// PDF
pdf.Read(path)
pdf.Merge(files, output)
pdf.Split(path, pages)
pdf.Extract(path, page)

// Excel
excel.Read(path, sheet)
excel.Write(path, data)
excel.Query(path, query)

// Word
word.Read(path)
word.Write(path, content)
word.Template(template, data)
```

**优先级：** P1

---

### 3.3 日程管理

**使用频率：** 中

**核心操作：**
```go
// Calendar
calendar.List(start, end)
calendar.Create(event)
calendar.Update(eventId, event)
calendar.Delete(eventId)

// Todo
todo.List(filter)
todo.Create(task)
todo.Complete(taskId)
```

**优先级：** P2

---

## 实现顺序

### 第一批（立即实现）
1. **Git 工具** - 编程最高频（3 天）
2. **Docker 工具** - DevOps 必备（3 天）
3. **邮件工具** - 办公最高频（4 天）

**总计：10 天**

### 第二批（短期）
1. 代码分析工具（2 天）
2. 包管理工具（2 天）
3. Docker 镜像管理（2 天）

**总计：6 天**

### 第三批（中期）
1. 文档处理工具（5 天）
2. 日程管理工具（3 天）
3. Docker 网络和卷（2 天）

**总计：10 天**

---

## 实现规范

### 1. 使用 Schema-based Tool

```go
import "github.com/ray/goreact/pkg/actor/schema"

func NewGitTool() *schema.Tool {
    return schema.NewTool(
        "git",
        "Git version control operations",
        gitSchema,
        gitHandler,
    )
}
```

### 2. 友好的错误消息

```go
if err != nil {
    return nil, schema.NewUserError(
        "Failed to clone repository: %s\n"+
        "Suggestions:\n"+
        "1. Check if the URL is correct\n"+
        "2. Verify your Git credentials\n"+
        "3. Ensure you have network access",
        err.Error(),
    )
}
```

### 3. 跨平台兼容

```go
// 使用 Go 标准库或跨平台库
import "os/exec"

cmd := exec.Command("git", "clone", url, path)
```

### 4. 安全控制

```go
// 使用 Actor Presets 的权限控制
actor := actorPresets.NewSafeActor(toolManager,
    actorPresets.WithAllowedTools("git", "docker", "email"),
    actorPresets.WithDenyCommands("rm -rf /", "dd if=/dev/zero"),
)
```

---

## 测试策略

### 单元测试
- 每个工具都有独立的单元测试
- 覆盖率目标：80%+

### 集成测试
- 测试真实的 Git/Docker/Email 操作
- 使用 Mock 服务器（测试环境）

### 示例测试
- 每个工具都有完整的示例代码
- 示例可运行、有注释

---

## 文档规范

### 每个工具需要：
1. **设计文档** - 功能、Schema、使用场景
2. **使用指南** - 场景驱动，问题 → 解决方案
3. **示例代码** - 完整可运行的示例
4. **API 文档** - 参数说明、返回值

---

## 成功指标

### 代码质量
- [ ] 单元测试覆盖率 > 80%
- [ ] 所有示例可运行
- [ ] 文档完整

### 用户体验
- [ ] Schema 验证自动化
- [ ] 错误消息友好
- [ ] 跨平台兼容

### 安全性
- [ ] 权限控制完善
- [ ] 输入验证严格
- [ ] 敏感信息保护（密码、Token）

---

## 总结

### 核心价值
1. **高频场景优先** - Git、Docker、Email
2. **Schema-based** - 自动验证、友好错误
3. **安全可控** - 权限系统、输入验证
4. **跨平台** - 使用标准库和跨平台工具

### 不做什么
- ❌ 不实现所有系统命令（用 bash 工具）
- ❌ 不重复造轮子（用现有库）
- ❌ 不牺牲安全性（严格权限控制）

### 记住
> 我们是在做**高频场景的专用工具**，不是在做**系统命令的包装器**。
> 每个工具都应该解决真实问题，提供比直接调用命令更好的体验。
