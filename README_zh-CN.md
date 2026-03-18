<div align="center">

# ⚛️ GoReAct

**Go 语言打造的高性能、高扩展性 ReAct 智能体引擎框架。**

[![Go Report Card](https://goreportcard.com/badge/github.com/ray/goreact)](https://goreportcard.com/report/github.com/ray/goreact)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

[**English**](./README.md) | [**中文说明**](./README_zh-CN.md)

</div>

---

## 💡 什么是 GoReAct?

**GoReAct** 是一个轻量级、面向生产环境的框架，完整实现了 **ReAct (Reasoning + Acting)** 核心模式。它赋予开发者构建自主 AI 智能体的能力，让大模型能够基于复杂任务进行迭代推理、制定计划并调用工具栈执行具体动作。

与那些脆弱且难以调试的分布式 P2P 多智能体网络不同，GoReAct 秉持 **Agent-as-a-Tool (AAAT) / 万物皆工具** 的设计哲学。从一个简单的计算器，到一个具备完整自主能力的子 Agent，在顶层框架看来都只是一个统一接口的“工具”，以一种优雅的分形结构无限嵌套、横向扩展。

## ✨ 核心特性

- 🏗️ **极致的整洁架构**：严格分离核心关注点，构建 **Thinker**（思考者）、**Actor**（执行者）、**Observer**（观察者）和 **Terminator**（终止器）的高效流水线。
- 🧰 **万物皆工具 (AAAT)**：将复杂的子 Agent 包装成基础 Tool 供上层调度，大幅降低大模型实时规划的 Token 成本和长上下文遗忘风险。
- 🔌 **中间件生态系统**：受 Web 框架启发，为 *Think* 阶段注入中间件（支持日志审计、多级缓存、限流熔断、RAG 上下文动态注入）。
- 🧠 **强大的 Prompt 工具箱**：高级上下文管理、智能 Token 计算（支持中/英/混合文本精确统计）、以及动态的历史对话滑动窗口压缩。
- ⚡ **高性能与安全**：专为高并发设计，原生支持严格的超时控制、Panic 恢复机制以及 LLM 故障时的优雅降级。
- 🌐 **大模型全兼容**：自带标准接口，完美集成 `gochat`，开箱即用 OpenAI, Anthropic, Qwen, Ollama 等各家大模型。

## 🚀 快速开始

### 安装

```bash
go get github.com/ray/goreact
```

### 1 分钟快速上手

我们将构建一个智能体，能够根据当前日期推算未来日期，并进行数学计算。以下是使用 Qwen/OpenAI 真实大模型的极简示例：

```go
package main

import (
	"context"
	"fmt"
	"github.com/ray/goreact/pkg/engine"
	"github.com/ray/goreact/pkg/thinker"
	"github.com/ray/goreact/pkg/actor"
	"github.com/ray/goreact/pkg/tools"
	"github.com/ray/goreact/pkg/tools/builtin"
	// 假设你已初始化了一个 gochat client
)

func main() {
	// 1. 组装工具箱
	toolMgr := tools.NewSimpleManager()
	toolMgr.Register(builtin.NewCalculator())
	toolMgr.Register(builtin.NewDateTime())

	// 2. 构建 ReAct 引擎
	reactor := engine.NewReactor(
		engine.WithThinker(thinker.Default(client, 
			thinker.WithModel("gpt-4o"),
			thinker.WithToolManager(toolMgr),
		)),
		engine.WithActor(actor.Default(actor.WithToolManager(toolMgr))),
	)

	// 3. 运行你的智能体
	ctx := context.Background()
	result, _ := reactor.Run(ctx, "session-1", "如果今天是 2026-03-18，请算一下 100 天后是几号？然后再推算那个日期的两倍天数。")

	fmt.Println("最终结果:", result.FinalResult)
}
```

## 🏗️ 架构设计图解

GoReAct 的核心是一个迭代的状态机 (State Machine)，由以下四个关键抽象构成：

1. **Thinker (思考者)**：读取对话上下文和上一步的观察结果，交由大模型进行深度推理 (Thought)，并决定下一步调用哪个工具。
2. **Actor (执行者)**：在一个隔离的环境中安全执行选定的工具 (Action)。
3. **Observer (观察者)**：验证工具执行的原始结果，将其清洗、提取并格式化为 `Observation` 反馈给引擎。
4. **Terminator (终止器)**：评估任务是否已得出 `Final Answer` 或是触发了系统终止条件（如最大迭代次数、超时断电等）。

```text
输入任务 (Task Input)
    │
 ┌──▼─────────────────────────────┐
 │ Thinker (推理并选择执行工具)      │◄──┐
 └─┬──────────────────────────────┘   │
   │ 产生思考过程 + 工具调用信息          │
 ┌─▼──────────────────────────────┐   │
 │ Actor   (安全执行底层工具)        │   │
 └─┬──────────────────────────────┘   │
   │ 返回原始结果或错误日志               │
 ┌─▼──────────────────────────────┐   │
 │ Observer(清洗提炼，格式化观察)    │   │
 └─┬──────────────────────────────┘   │
   │ 更新状态与上下文                    │
 ┌─▼──────────────────────────────┐   │
 │ Terminator(评估是否结束或终止)    ├───┘
 └────────────────────────────────┘
```

## 📖 官方文档与最佳实践

深入了解 GoReAct 的强大能力，请参阅：

- [框架架构深度解析](./ARCHITECTURE.md) - 了解底层工作原理与设计决策。
- [RAG 接入指南](./docs/RAG_INTEGRATION_GUIDE.md) - 如何在引擎中构建 Agentic RAG 系统。
- [工具开发手册](./pkg/tools/builtin/USAGE_GUIDE.md) - 学习如何编写你自己的定制化工具。

## 🛠️ 内置工具库

GoReAct 开箱即带一套久经考验的基础工具，随时为您所用：
- **系统工具组**: `Bash`, `Grep`, `Read`, `Write`, `Edit`, `Glob`, `LS` (适用于本地编码、运维类智能体)
- **效能工具组**: `Calculator` (计算器), `DateTime` (时间引擎), `Email` (邮件收发)

## 🤝 参与贡献

我们极其欢迎来自社区的各类贡献！不论是修复 Bug、完善文档还是提出振奋人心的新功能提案。

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的修改 (`git commit -m 'feat: 增加了一个不可思议的功能'`)
4. 推送至分支 (`git push origin feature/amazing-feature`)
5. 提交 Pull Request 等待 Review

## 📄 许可证

本项目采用 MIT 许可证授权 - 查看 [LICENSE](LICENSE) 文件了解详细信息。
