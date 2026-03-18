# GoReAct 工具集重构计划

## 🎯 目标
对标 Claude Code 的 11 个核心工具，打造**轻量、实用而强大**的工具集

## ✅ 已完成 (Tier 1 - 文件操作)

### 文件操作三剑客
- ✅ `read.go` - Read 工具（读取文件，支持行范围）
- ✅ `write.go` - Write 工具（写入文件，自动创建目录）
- ✅ `edit.go` - Edit 工具（精确编辑，支持多位置 diff 式修改）

**核心特性**：
- 安全性：特殊文件访问限制
- 错误处理：详细的错误消息和建议
- 用户体验：带行号的内容显示

## ✅ 已完成 (Tier 2 - 搜索浏览)

### 搜索铁三角
- ✅ `glob.go` - Glob 工具（文件名模式匹配，支持 glob 语法）
- ✅ `grep.go` - Grep 工具（文本内容搜索，正则支持，行列定位）
- ✅ `ls.go` - LS 工具（列出目录内容，树形结构，详细信息）

**核心特性**：
- 智能过滤：隐藏文件跳过、大文件限制
- 性能优化：逐行扫描、内存友好
- 用户友好：相对路径显示、统计信息

## 📋 待完成 (Tier 3 - 执行扩展)

### 执行与扩展
- ⏳ `bash.go` - Bash 工具（Shell 命令执行，持久 session）
- ✅ `calculator.go` - 计算器（保留，高频使用）
- ✅ `datetime.go` - 日期时间（保留，高频使用）

## ❌ 移除清单

### ✅ 已完成清理

#### 移至独立插件包（低频使用）
- `email.go` → goreact-plugins/email
- `docker.go` → goreact-plugins/docker  
- `git.go` → goreact-plugins/git

#### ✅ 已直接删除（冗余/无价值）
- ~~`http.go`~~ ✅ 已删除
- ~~`curl.go`~~ ✅ 已删除
- ~~`echo.go`~~ ✅ 已删除
- ~~`port.go`~~ ✅ 已删除
- ~~`filesystem.go`~~ ✅ 已删除

## 📊 对比分析

### 原始工具集 (13 个)
```
calculator, http, datetime, bash, filesystem, grep, 
docker, git, email, curl, echo, port, (test)
```
**问题**：职责不清、功能堆砌、缺乏重点

### 新工具集 (9 个)
```
Read, Write, Edit,        # 文件操作三剑客
Glob, Grep, LS,          # 搜索铁三角
Bash, Calculator, DateTime # 执行扩展
```
**优势**：职责单一、组合强大、覆盖全面

## 🔧 通用工具函数 (common.go)

- `validateRequired()` - 验证必需参数
- `validateRequiredString()` - 验证字符串参数
- `validateFileSafety()` - 文件安全性检查

## 📝 下一步行动

1. ✅ 完成 Tier 1 文件操作工具
2. ⏳ 创建 Tier 2 搜索工具
3. ⏳ 优化 Tier 3 执行工具
4. ⏳ 更新测试用例
5. ⏳ 更新示例代码
6. ⏳ 更新文档

## 🎨 设计原则

1. **职责单一**：每个工具只做一件事，做到极致
2. **组合强大**：通过工具组合实现复杂功能
3. **覆盖全面**：文件操作、搜索、执行三大核心场景
4. **安全可靠**：完善的权限控制和错误处理
5. **用户友好**：清晰的错误消息和使用建议
