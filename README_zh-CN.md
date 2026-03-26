<div align="center">

# ⚛️ GoReAct

**一个高性能、基于模式编排 (Pattern-Driven) 的 Go 语言 ReAct 智能体引擎。**

[![Go Report Card](https://goreportcard.com/badge/github.com/DotNetAge/goreact)](https://goreportcard.com/report/github.com/DotNetAge/goreact)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Documentation](https://img.shields.io/badge/docs-goreact.rayainfo.cn-6019bd.svg)](https://goreact.rayainfo.cn)

[**官方网站**](https://goreact.rayainfo.cn) | [**English**](./README.md) | [**中文说明**](./README_zh-CN.md)

</div>

---

## 💡 什么是 GoReAct?

**GoReAct** 是一个轻量级、面向生产环境的 **ReAct** 模式实现框架。与普通的 LLM 包装器不同，GoReAct 专为 **可编程认知 (Programmable Cognition)** 而生。它秉持 **编排模式驱动 (Pattern-Driven Orchestration)** 的设计哲学，通过引入 **“图灵完备的逻辑管线”**，赋予智能体类人的灵活性与高度可编程的执行精度。

通过与兄弟项目 [**GoRAG**](https://gorag.rayainfo.cn/) 深度融合，GoReAct 实现了业界领先的 **语义化匹配 (Semantic Matching)**，彻底告别了脆弱的字符串硬编码工具召回，让意图、记忆与技能的匹配如同资深专家般精准。

## ✨ 进化亮点 (Phase 4)

- 🧠 **三模态仿真记忆系统**：完美映射人类认知，包含 **工作记忆**（随时间衰减）、**语义记忆**（由 GoRAG 提供 RAG 支持）与 **肌肉记忆**（基于经验的 SOP）。
- 🚀 **万能逻辑管线 (Universal Pipeline)**：管线原生支持 **If/Loop/Break/Return** 等逻辑原语，支持在执行流中处理循环重试与熔断，已提升至上游 `gochat` 框架。
- 🧬 **自适应快径演化 (Evolution)**：独创的“三态转化”机制，自动将成功的推理轨迹编译为 **CompiledAction**（肌肉记忆）。未来面对同类任务时，直接命中**零 LLM 推理**的快径，延迟直降 99%。
- 🧩 **暗语驱动的架构师 (Thinker as Architect)**：支持通过 `/plan`, `/specs` 等前缀指令，强制改变思考模式，将自然语言意图编译为结构化的执行图谱。
- 🛡️ **Sudo HITL 安全防线**：原子级的权限管控。涉及敏感操作（如删库、发邮件）的工具强制触发“人在回路 (Human-In-The-Loop)”授权拦截。

## 🚀 极速开始

通过 `Builder`，仅需几行代码即可构建一个支持 `/plan` 复杂规划的智能体：

```go
package main

import (
    "context"
    "github.com/DotNetAge/goreact/pkg/agent"
    "github.com/DotNetAge/goreact/pkg/engine"
)

func main() {
    // 1. 使用 Builder 装配一个拥有编排能力的 Agent
    builder := agent.NewBuilder("DevOpsAgent")
    builder.WithModel("gpt-4o") // 也支持 ollama, qwen 等
    builder.WithSystemPrompt("你是一个资深运维工程师。")
    builder.WithTools(myDeploymentTools...)
    
    myAgent, _ := builder.Build()

    // 2. 使用 /plan 模式下达复杂任务
    // 智能体会自动拆解任务，并通过逻辑管线逐步执行
    ctx := context.Background()
    result, _ := myAgent.Chat(ctx, "session-1", "/plan 部署服务，如果测试失败，则回滚到 v1.0.0 版本。")
    
    println(result)
}
```

## 🏗️ 技能三态转化架构

在 GoReAct 中，经验与技能会经历三种状态的进化：

1.  **源码态 (Source)**: 人类可读的 Markdown (`SKILL.md`)，由 **Master-Sub** 主从模式解释执行。
2.  **编译态 (Compiled)**: **编译器 (Compiler)** 扫描历史成功记录，蒸馏提炼出带有执行指纹的结构化 **CompiledAction**。
3.  **执行态 (Execution)**: **自适应执行器 (Adaptive Runner)** 尝试命中快径，利用 **Observer (观察者)** 作为裁判比对指纹，实现极速执行。

## 🏛️ 系统架构总览

```text
       [ 用户意图 / User Intent ] 
              │
      ┌───────▼────────┐      ┌─────────────────────────┐
      │ 思考者 Thinker │◄─────┤  三模态记忆 MemoryBank  │
      │ (任务编译器)   │      │  (Powered by GoRAG)     │
      └───────┬────────┘      └─────────────────────────┘
              │ 逻辑计划 (If/Loop/Sequence)
      ┌───────▼────────┐      ┌─────────────────────────┐
      │ 执行管线       │◄─────┤ 安全防线 Security Hook  │
      │ Pipeline       │      └─────────────────────────┘
      └───────┬────────┘
              │ 动作执行结果
      ┌───────▼────────┐
      │ 演化器 Runner  │──────► [ 编译动作缓存库 ]
      │ (Evolution)    │        (CompiledAction Cache)
      └────────────────┘
```

## 📖 文档与资源

- [**官方文档网站**](https://goreact.rayainfo.cn) - 获取最全面的 API 参考与进阶指南。
- [详细特性说明](./FEATURES.md) - 深入探索核心创新。
- [架构设计白皮书](./ARCHITECTURE.md) - 深入了解底层引擎与管线。
- [技能演化规范](./pkg/skill/README.md) - 学习如何让智能体自我进化。

## 🛠️ 内置能力支持
- **模型接入**: OpenAI, Anthropic, Ollama, DashScope (通义千问), DeepSeek。
- **内置工具**: `Bash` (Shell 执行), `FileIO`, `HTTP`, `Calculator`, `DateTime`。
- **编排模式**: `Master-Sub` (主从拆解), `Evolution` (演化快径), `Chain-of-Thought`。

## 🤝 参与贡献

欢迎任何形式的贡献！请阅读 [Contributing Guidelines](CONTRIBUTING.md) 了解详情。

**GoReAct** 基于 [MIT License](LICENSE) 开源。
