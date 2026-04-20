<div align="center">

# GoReAct

**高性能、模式驱动的 ReAct (Reasoning + Acting) 引擎，为 Go 而生。**

[![Go Report Card](https://goreportcard.com/badge/github.com/DotNetAge/goreact)](https://goreportcard.com/report/github.com/DotNetAge/goreact)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[**Website**](https://goreact.rayainfo.cn) | [**English**](./README.md) | [**中文说明**](./README_zh-CN.md)

</div>

---

## GoReAct 是什么

GoReAct 是一个用纯 Go 实现的 AI Agent 框架，核心采用 **T-A-O (Think-Act-Observe)** 循环模式驱动 LLM 进行推理与行动。它的设计目标是帮助开发人员**专注于 Tools 与 Skills 的开发与运用**，内核机制与性能由 GoReAct 负责，保证以最少量的 Token 收获最大的价值。开发人员在垂直领域中提供各种不同的 Tools 和 Skills，才能让 AI Agent 在垂直领域中创造最大的价值。

### 核心特性

**T-A-O 推理引擎**：Think（意图分析 + 推理决策）→ Act（工具执行）→ Observe（结果评估），循环迭代直到得出最终答案。支持流式输出思维链，客户端可实时展示推理过程。

**五类型记忆系统**：Session（临时）、User（用户偏好）、LongTerm（知识库）、Refactive（工具/技能语义索引）、Experience（成功执行经验自动保存）。Memory 由外部调用者实现，简单 InMemory 到完整 RAG 均可接入。Think 阶段自动检索相关记忆注入 Prompt，抑制幻觉。

**ReNew 语义上下文重建**：当上下文窗口逼近极限时，优先通过 Memory 的 ReNew 接口进行语义重建（从记忆中检索相关上下文重新组装），而非暴力截断。优先级链：ReNew → MicroCompact → Full Compact (LLM 摘要)。

**经验自成长**：任务成功完成后自动将问题描述 + 解决方案保存为经验记忆。Memory 实现可以将经验转化为 Skill（SKILL.md），使 GoReAct 越用越强。

**三层上下文防御**：Layer 1 工具结果截断与持久化 → Layer 2 大结果溢写到磁盘 → Layer 3 上下文压缩/摘要。防止上下文爆炸。

**多 Agent 团队协作**：支持创建团队、异步派发 SubAgent、Channel 式消息通信、结果收集与综合。每个 SubAgent 可拥有独立的 SystemPrompt 和 Model。

**中断-恢复交互模式**：`ask_user` 工具支持多轮对话澄清（Think 阶段发现问题暂停 → 用户回答 → 恢复执行），`ask_permission` 工具支持高风险工具授权（执行前暂停 → 用户审批 → 继续或拒绝）。

**事件总线**：完整的 ReactEvent 事件流（ThinkingDelta、ActionStart、ActionResult、SubtaskSpawned、ExecutionSummary 等），支持客户端实时渲染。

**渐进式 Skill 加载**：YAML Frontmatter + Markdown Body 的 SKILL.md 格式，支持文件系统目录加载和 Go embed 内置加载。三级渐进披露：元数据（启动时）→ 指令（激活时）→ 资源（按需加载）。

**MCP 协议支持**：通过 `MCPClient` 接口接入外部 MCP 服务器，将 MCP 工具自动转换为标准 `FuncTool` 注册到 Reactor。

**Prompt 模板化**：所有内置 Prompt 使用 Go Template（embed.FS）实现，支持独立编辑和运行时渲染。

---

## 快速开始

### 安装

```bash
go get github.com/DotNetAge/goreact
```

### 入口说明

`goreact.Agent` 是面向用户的**一级入口**，封装了 Reactor（T-A-O 引擎）、Memory、Model 和 ContextWindow，一行代码即可创建一个完整的智能体。`reactor.Reactor` 是底层推理引擎，通常不需要直接使用，仅在需要深度定制 T-A-O 循环行为时才直接操作。

### 5 分钟 Hello World

```go
package main

import (
    "fmt"

    "github.com/DotNetAge/goreact"
)

func main() {
    // 1. 创建 Agent（只需一个 API Key）
    agent := goreact.DefaultAgent("your-api-key")

    // 2. 提问，自动管理会话上下文
    answer, err := agent.Ask("你好，请介绍一下你自己")
    if err != nil {
        panic(err)
    }

    fmt.Println("Answer:", answer)
}
```

### 带记忆的多轮对话

`DefaultAgent` 已内置 InMemory 记忆和会话管理，支持自动上下文管理和记忆检索：

```go
package main

import (
    "fmt"

    "github.com/DotNetAge/goreact"
)

func main() {
    agent := goreact.DefaultAgent("your-api-key")

    // 第一轮：告诉 Agent 一件事
    agent.Ask("记住：我最喜欢的编程语言是 Go")

    // 第二轮：Agent 会自动检索记忆来回答
    answer, _ := agent.Ask("我最喜欢什么编程语言？")
    fmt.Println(answer) // "Go"

    // 查看 Agent 信息
    fmt.Println("Name:", agent.Name())
    fmt.Println("Session:", agent.SessionID())
}
```

### 自定义 Agent

当 `DefaultAgent` 不够用时，可以通过 `NewAgent` 精确控制每个组件：

```go
package main

import (
    "fmt"

    "github.com/DotNetAge/goreact"
    "github.com/DotNetAge/goreact/core"
    "github.com/DotNetAge/goreact/reactor"
)

func main() {
    // 自定义模型配置
    model := &core.ModelConfig{
        Name:      "gpt-4o",
        APIKey:    "your-api-key",
        BaseURL:   "https://api.openai.com/v1",  // 或其他兼容 API
        MaxTokens: 16384,
    }

    // 自定义记忆（可替换为 RAG 等外部实现）
    memory := core.NewInMemoryMemory()

    // 自定义 Reactor（深度定制 T-A-O 引擎）
    r := reactor.NewReactor(
        reactor.ReactorConfig{
            APIKey:  model.APIKey,
            Model:   model.Name,
            BaseURL: model.BaseURL,
        },
        reactor.WithMemory(memory),
        reactor.WithSecurityPolicy(yourPolicy),
    )

    // 组装 Agent
    agent := goreact.NewAgentWithSession(
        &core.AgentConfig{
            Name:        "my-coder",
            Domain:      "programming",
            Description: "A coding assistant",
        },
        model,
        memory,
        r,
        "session-001", // Session ID
        16384,         // 上下文窗口 Token 上限
    )

    answer, err := agent.Ask("帮我写一个 Go 的 HTTP server")
    if err != nil {
        panic(err)
    }
    fmt.Println(answer)
}
```

---

## 进阶：直接使用 Reactor

> 通常不需要直接使用 Reactor。`goreact.Agent` 已封装了完整的智能体生命周期。以下场景才需要直接操作 Reactor：
> - 需要完全控制 T-A-O 循环的每次迭代
> - 需要在无状态模式下使用（不管理会话上下文）
> - 需要深度定制 Prompt、工具管线、压缩策略等引擎内部行为

### 无状态调用

```go
package main

import (
    "context"
    "fmt"

    "github.com/DotNetAge/goreact/reactor"
)

func main() {
    r := reactor.NewReactor(reactor.ReactorConfig{
        APIKey: "your-api-key",
        Model:  "qwen3.5-flash",
    })

    // 无状态调用：每次 Run 独立，无会话记忆
    result, err := r.Run(context.Background(), "你好", nil)
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Answer)
}
```

### 流式输出 + 事件监听

```go
package main

import (
    "context"
    "fmt"

    "github.com/DotNetAge/goreact/core"
    "github.com/DotNetAge/goreact/reactor"
)

func main() {
    bus := reactor.NewEventBus()
    ch, cancel := bus.Subscribe()
    defer cancel()

    go func() {
        for event := range ch {
            switch event.Type {
            case core.ThinkingDelta:
                fmt.Print(event.Data)
            case core.ActionStart:
                data := event.Data.(core.ActionStartData)
                fmt.Printf("\n[Tool: %s]\n", data.ToolName)
            case core.ExecutionSummary:
                data := event.Data.(core.ExecutionSummaryData)
                fmt.Printf("\n=== Summary ===\nIterations: %d\nTokens: %d\nTools: %v\n",
                    data.TotalIterations, data.TokensUsed, data.ToolsUsed)
            }
        }
    }()

    r := reactor.NewReactor(
        reactor.ReactorConfig{APIKey: "key", Model: "qwen3.5-flash"},
        reactor.WithEventBus(bus),
    )
    r.Run(context.Background(), "帮我写一个 Go 的 HTTP server", nil)
}
```

---

## 内置工具清单

| 类别 | 工具名 | 说明 |
|------|--------|------|
| **文件操作** | `read` | 读取文件内容 |
| | `write` | 写入/创建文件 |
| | `file_edit` | 编辑文件（基于搜索替换） |
| | `replace` | 全局搜索替换 |
| | `ls` | 列出目录内容 |
| | `grep` | 正则搜索文件内容 |
| | `glob` | Glob 模式搜索文件名 |
| **终端与执行** | `bash` | 执行 Shell 命令 |
| | `repl` | 交互式代码执行环境 |
| | `calculator` | 数学计算 |
| | `echo` | 输出文本 |
| **信息获取** | `web_search` | 互联网搜索 |
| | `web_fetch` | 网页内容抓取 |
| **任务编排** | `task_create` | 创建并同步执行子任务 |
| | `task_result` | 获取子任务结果 |
| | `task_list` | 列出所有子任务 |
| **多 Agent 团队** | `subagent` | 派发独立异步 Agent |
| | `subagent_result` | 获取 SubAgent 结果 |
| | `subagent_list` | 列出所有 SubAgent |
| | `team_create` | 创建团队 |
| | `send_message` | 发送团队消息 |
| | `receive_messages` | 接收团队消息 |
| | `team_status` | 查看团队状态 |
| | `team_delete` | 删除团队 |
| | `wait_team` | 等待团队全部完成 |
| **记忆系统** | `memory_save` | 保存知识到长期记忆 |
| | `memory_search` | 搜索长期记忆 |
| **任务管理** | `todo_write` | 创建/更新任务计划 |
| | `todo_read` | 读取当前任务列表 |
| | `todo_execute` | 执行任务计划 |
| **交互工具** | `ask_user` | 向用户提问（中断-恢复） |
| | `ask_permission` | 请求用户授权（中断-恢复） |
| **其他** | `cron` | 定时任务 |
| | `skill_create` | 动态创建 Skill |
| | `skill_list` | 列出可用 Skill |

---

## 架构概览

```
┌──────────────────────────────────────────────────┐
│                    Agent / CLI                    │
│  (Ask / AskWithContext / ContextWindow)           │
└────────────────────┬─────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────┐
│                  Reactor (T-A-O)                  │
│                                                    │
│  Run(ctx, input, history)                         │
│    ├─ Phase 1: classifyIntent (意图分类)           │
│    ├─ Phase 2: T-A-O Loop                         │
│    │    ├─ Think (LLM推理 + Memory检索 + Skill匹配) │
│    │    ├─ Act   (工具执行 / 权限检查)              │
│    │    ├─ Observe (结果评估)                       │
│    │    └─ maybeCompact (上下文压缩)                │
│    ├─ Phase 3: saveExperience (经验保存)           │
│    └─ Phase 4: Emit ExecutionSummary               │
│                                                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │
│  │ToolRegistry│  │SkillRegistry│ │  Memory (外部注入)│ │
│  │(含权限管线)│  │(渐进式加载) │ │(5种记忆类型)      │ │
│  └──────────┘  └──────────┘  └──────────────────┘ │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │
│  │ EventBus │  │MessageBus│  │  MCPToolRegistry  │ │
│  │(事件流)   │  │(团队通信) │ │  (外部工具协议)    │ │
│  └──────────┘  └──────────┘  └──────────────────┘ │
└──────────────────────────────────────────────────┘
```

---

## 配置选项

GoReAct 通过 `ReactorOption` 函数式选项进行配置：

```go
r := reactor.NewReactor(config,
    // 记忆系统
    reactor.WithMemory(yourMemory),

    // 上下文防御
    reactor.WithCompactor(yourCompactor),           // LLM 摘要压缩
    reactor.WithCompactorConfig(yourConfig),        // 压缩阈值
    reactor.WithResultStorage(yourStorage),          // 大结果溢写到磁盘
    reactor.WithTokenEstimator(yourEstimator),       // 自定义 Token 估算

    // 安全
    reactor.WithSecurityPolicy(yourPolicy),          // 安全策略
    reactor.WithoutTool("bash"),                     // 禁用特定工具
    reactor.WithoutBundledTools(),                   // 禁用全部内置工具

    // 事件与通信
    reactor.WithEventBus(yourBus),                   // 自定义事件总线
    reactor.WithMessageBus(yourMsgBus),              // 团队消息总线

    // 扩展
    reactor.WithExtraTools(customTool),              // 注册自定义工具
    reactor.WithSkillDir("/path/to/skills"),         // 外部 Skill 目录
    reactor.WithoutBundledSkills(),                  // 禁用内置 Skill
    reactor.WithMCPRegistry(yourMCPRegistry),         // MCP 工具注册
)
```

---

## 文档

| 文档 | 说明 |
|------|------|
| [功能描述与设计理念](./docs/记忆体的设计.md) | Memory 子系统设计文档 |
| [Memory 开发指南](./docs/memory-dev-guide.md) | 如何实现自定义 Memory |
| [集成指南](./docs/integration-guide.md) | 事件流、流式输出、Token 追踪等集成细节 |
| [工具开发指南](./docs/tool-dev-guide.md) | 如何开发自定义工具 |
| [Skill 开发指南](./docs/skill-dev-guide.md) | 如何开发 Skill |
| [MCP 开发指南](./docs/mcp-dev-guide.md) | 如何接入 MCP 服务器 |

---

## 许可证

[MIT License](./LICENSE)
