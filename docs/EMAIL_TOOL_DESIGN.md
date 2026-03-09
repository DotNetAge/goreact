# 邮件工具设计文档

## 概述

邮件是办公场景中最高频的工具，几乎每个办公人员每天都要处理大量邮件。我们需要提供一个 Schema-based 的邮件工具，让 LLM 能够轻松地发送、接收和管理邮件。

---

## 核心痛点

### 痛点 1：配置复杂

**问题：**
- SMTP/IMAP 配置参数众多
- 不同邮件服务商配置不同
- TLS/SSL 配置容易出错

**解决方案：**
- 预设常见邮件服务商配置
- 自动检测 TLS/SSL
- 提供配置验证

### 痛点 2：附件处理麻烦

**问题：**
- 附件编码复杂
- 大附件处理困难
- 多附件管理混乱

**解决方案：**
- 自动处理附件编码
- 支持多附件
- 提供附件大小限制

### 痛点 3：邮件搜索不便

**问题：**
- 无法快速搜索邮件
- 过滤条件有限
- 批量操作困难

**解决方案：**
- 提供强大的搜索功能
- 支持多种过滤条件
- 支持批量操作

---

## 功能设计

### 1. 发送邮件

#### 1.1 Send（发送文本邮件）

```go
email.Send(to, subject, body, options)
```

**参数：**
- `to` (string/[]string, required) - 收件人
- `subject` (string, required) - 主题
- `body` (string, required) - 邮件正文
- `cc` ([]string, optional) - 抄送
- `bcc` ([]string, optional) - 密送
- `attachments` ([]string, optional) - 附件路径列表
- `reply_to` (string, optional) - 回复地址

**示例：**
```json
{
  "operation": "send",
  "to": "user@example.com",
  "subject": "Meeting Tomorrow",
  "body": "Hi, let's meet tomorrow at 10am.",
  "cc": ["manager@example.com"],
  "attachments": ["/path/to/file.pdf"]
}
```

#### 1.2 SendHTML（发送 HTML 邮件）

```go
email.SendHTML(to, subject, html, options)
```

**参数：**
- `to` (string/[]string, required) - 收件人
- `subject` (string, required) - 主题
- `html` (string, required) - HTML 正文
- `cc` ([]string, optional) - 抄送
- `bcc` ([]string, optional) - 密送
- `attachments` ([]string, optional) - 附件路径列表

---

### 2. 接收邮件

#### 2.1 List（列出邮件）

```go
email.List(options)
```

**参数：**
- `folder` (string, optional) - 文件夹，默认 "INBOX"
- `limit` (int, optional) - 限制数量，默认 20
- `unread_only` (bool, optional) - 只显示未读
- `since` (string, optional) - 起始日期

**返回：**
```json
{
  "emails": [
    {
      "id": "123",
      "from": "sender@example.com",
      "subject": "Meeting Tomorrow",
      "date": "2026-03-08T10:00:00Z",
      "unread": true,
      "has_attachments": false
    }
  ]
}
```

#### 2.2 Read（读取邮件）

```go
email.Read(messageId, options)
```

**参数：**
- `message_id` (string, required) - 邮件 ID
- `mark_as_read` (bool, optional) - 标记为已读，默认 true

**返回：**
```json
{
  "id": "123",
  "from": "sender@example.com",
  "to": ["user@example.com"],
  "subject": "Meeting Tomorrow",
  "body": "Hi, let's meet tomorrow at 10am.",
  "date": "2026-03-08T10:00:00Z",
  "attachments": [
    {
      "filename": "agenda.pdf",
      "size": 102400
    }
  ]
}
```

#### 2.3 Search（搜索邮件）

```go
email.Search(query, options)
```

**参数：**
- `query` (string, required) - 搜索关键词
- `folder` (string, optional) - 文件夹
- `from` (string, optional) - 发件人过滤
- `subject` (string, optional) - 主题过滤
- `since` (string, optional) - 起始日期
- `until` (string, optional) - 结束日期

**示例：**
```json
{
  "operation": "search",
  "query": "meeting",
  "from": "boss@example.com",
  "since": "2026-03-01"
}
```

---

### 3. 邮件管理

#### 3.1 Delete（删除邮件）

```go
email.Delete(messageId)
```

**参数：**
- `message_id` (string, required) - 邮件 ID

#### 3.2 Move（移动邮件）

```go
email.Move(messageId, folder)
```

**参数：**
- `message_id` (string, required) - 邮件 ID
- `folder` (string, required) - 目标文件夹

#### 3.3 MarkAsRead（标记为已读）

```go
email.MarkAsRead(messageId)
```

**参数：**
- `message_id` (string, required) - 邮件 ID

#### 3.4 MarkAsUnread（标记为未读）

```go
email.MarkAsUnread(messageId)
```

**参数：**
- `message_id` (string, required) - 邮件 ID

---

### 4. 附件管理

#### 4.1 DownloadAttachment（下载附件）

```go
email.DownloadAttachment(messageId, filename, savePath)
```

**参数：**
- `message_id` (string, required) - 邮件 ID
- `filename` (string, required) - 附件文件名
- `save_path` (string, required) - 保存路径

---

## 配置设计

### 邮件配置

```go
type EmailConfig struct {
    // SMTP 配置（发送邮件）
    SMTP struct {
        Host     string // SMTP 服务器地址
        Port     int    // SMTP 端口
        Username string // 用户名
        Password string // 密码
        From     string // 发件人地址
        TLS      bool   // 是否使用 TLS
    }

    // IMAP 配置（接收邮件）
    IMAP struct {
        Host     string // IMAP 服务器地址
        Port     int    // IMAP 端口
        Username string // 用户名
        Password string // 密码
        TLS      bool   // 是否使用 TLS
    }
}
```

### 预设配置

```go
// Gmail
smtp.gmail.com:587 (TLS)
imap.gmail.com:993 (TLS)

// Outlook
smtp.office365.com:587 (TLS)
outlook.office365.com:993 (TLS)

// QQ Mail
smtp.qq.com:587 (TLS)
imap.qq.com:993 (TLS)

// 163 Mail
smtp.163.com:465 (SSL)
imap.163.com:993 (SSL)
```

---

## Schema 定义

```go
var emailSchema = schema.Define(
    // 操作类型
    schema.Param("operation", schema.String, "Email operation").
        Enum("send", "send_html", "list", "read", "search", "delete", "move", "mark_read", "mark_unread").
        Required(),

    // 发送邮件参数
    schema.Param("to", schema.String, "Recipient email address").
        RequiredIf("operation", "send", "send_html"),

    schema.Param("subject", schema.String, "Email subject").
        RequiredIf("operation", "send", "send_html"),

    schema.Param("body", schema.String, "Email body").
        RequiredIf("operation", "send"),

    schema.Param("html", schema.String, "HTML body").
        RequiredIf("operation", "send_html"),

    schema.Param("cc", schema.Array, "CC recipients"),
    schema.Param("bcc", schema.Array, "BCC recipients"),
    schema.Param("attachments", schema.Array, "Attachment file paths"),

    // 接收邮件参数
    schema.Param("message_id", schema.String, "Message ID").
        RequiredIf("operation", "read", "delete", "move", "mark_read", "mark_unread"),

    schema.Param("folder", schema.String, "Email folder").
        Default("INBOX"),

    schema.Param("limit", schema.Number, "Number of emails to retrieve").
        Default(20),

    // 搜索参数
    schema.Param("query", schema.String, "Search query").
        RequiredIf("operation", "search"),

    schema.Param("from", schema.String, "Filter by sender"),
    schema.Param("since", schema.String, "Filter by date (YYYY-MM-DD)"),
)
```

---

## 安全控制

### 1. 密码保护

```go
// 不在日志中显示密码
func (e *Email) sanitizeConfig(config EmailConfig) EmailConfig {
    config.SMTP.Password = "***"
    config.IMAP.Password = "***"
    return config
}
```

### 2. 附件大小限制

```go
const MaxAttachmentSize = 25 * 1024 * 1024 // 25MB
```

### 3. 速率限制

```go
type RateLimiter struct {
    maxEmailsPerHour int
    sentCount        int
    resetTime        time.Time
}
```

---

## 错误处理

### 1. 认证失败

```go
if strings.Contains(err.Error(), "authentication failed") {
    return schema.NewUserError(
        "Email authentication failed\n"+
        "Suggestions:\n"+
        "1. Check username and password\n"+
        "2. Enable 'Less secure app access' (Gmail)\n"+
        "3. Use app-specific password",
    )
}
```

### 2. 连接失败

```go
if strings.Contains(err.Error(), "connection refused") {
    return schema.NewUserError(
        "Cannot connect to email server\n"+
        "Suggestions:\n"+
        "1. Check SMTP/IMAP server address\n"+
        "2. Verify port number\n"+
        "3. Check firewall settings",
    )
}
```

### 3. 附件过大

```go
if fileSize > MaxAttachmentSize {
    return schema.NewUserError(
        "Attachment too large\n"+
        "Maximum size: 25MB\n"+
        "Suggestions:\n"+
        "1. Compress the file\n"+
        "2. Use cloud storage and share link\n"+
        "3. Split into multiple emails",
    )
}
```

---

## 实现优先级

### P0（立即实现）
- ✅ send, send_html
- ✅ list, read
- ✅ 基础配置

### P1（短期）
- search
- delete, move
- mark_read, mark_unread

### P2（中期）
- download_attachment
- 预设配置
- 速率限制

---

## 依赖库

```go
// SMTP 发送
import "net/smtp"

// IMAP 接收
import "github.com/emersion/go-imap"
import "github.com/emersion/go-imap/client"

// 邮件解析
import "github.com/emersion/go-message/mail"
```

---

## 总结

### 核心价值
1. **高频场景** - 办公场景最常用
2. **Schema 验证** - 自动验证参数
3. **友好错误** - LLM 能理解的错误消息
4. **安全可控** - 密码保护、速率限制

### 记住
> 邮件是办公场景最高频的工具，必须做到：
> - 配置简单（预设常见服务商）
> - 操作便捷（发送、接收、搜索）
> - 安全可靠（密码保护、速率限制）
> - 错误友好（清晰的错误提示）
