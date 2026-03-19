<div align="center">

# ⚛️ GoReAct

**Go 语言打造的高性能、可编程 ReAct (Reasoning + Acting) 智能体框架。**

[![Go Report Card](https://goreportcard.com/badge/github.com/ray/goreact)](https://goreportcard.com/report/github.com/ray/goreact)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[**English**](./README.md) | [**中文说明**](./README_zh-CN.md)

</div>

---

## 💡 什么是 GoReAct?

**GoReAct** 是一个轻量级、面向生产环境的 **ReAct** 模式实现框架。GoReAct 秉持 **万物皆工具 (AAAT)** 的设计哲学，通过引入 **“图灵完备的逻辑管线”**，赋予智能体类人的灵活性与高度可编程的执行精度，使其能够自如地应对推理、规划与自进化任务。

## ✨ 进化亮点 (Phase 4)

- 🧠 **三模态仿真记忆系统**：完美映射人类认知，包含 **工作记忆**（随时间衰减）、**语义记忆**（RAG）与 **肌肉记忆**（基于经验的 SOP）。
- 🚀 **万能逻辑管线 (Universal Pipeline)**：管线原生支持 **If/Loop/Break/Return** 等逻辑原语，已提升至上游 `gochat` 框架。
- 🧩 **Thinker 即架构师**：支持 **暗语驱动模式** (`/plan`, `/specs`)，内置任务编译器，可将自然语言计划转化为可执行的逻辑流。
- 🧰 **提示词协议化工具箱**：提供 Fluent API 用于协议级 Prompt 构建，支持智能 Token 计数与上下文滑动窗口压缩。
- 🛡️ **Sudo HITL 安全机制**：工具具备精细化安全定级，高危操作强制触发 **人机协作 (HITL)** 授权。

## 🚀 快速开始

构建一个具备 `/plan` 能力的可编程智能体：

```go
// 1. 初始化 Reactor，装配记忆体与工具
reactor := engine.NewReactor(
    engine.WithThinker(thinker.Default(client, 
        thinker.WithMemoryBank(myMemory),
        thinker.WithToolManager(toolMgr),
    )),
    engine.WithActor(actor.Default(actor.WithToolManager(toolMgr))),
)

// 2. 使用 /plan 指令拆解复杂任务
result, _ := reactor.Run(ctx, "sess-1", "/plan 分析这份 CSV，如果发现错误，调用脚本修复它。")
```

## 🏗️ 架构概览

```text
       [ 用户意图 ] 
              │
      ┌───────▼────────┐      ┌─────────────────────────┐
      │    Thinker     │◄─────┤   三模态记忆体 (Memory)    │
      │  (任务编译器)    │      └─────────────────────────┘
      └───────┬────────┘
              │ 逻辑计划 (If/Loop/Sequence)
      ┌───────▼────────┐      ┌─────────────────────────┐
      │   通用管线      │◄─────┤   安全拦截 (HITL Hook)    │
      │   (逻辑执行)    │      └─────────────────────────┘
      └───────┬────────┘
              │ 执行结果
      ┌───────▼────────┐
      │  结晶化器       │──────► [ 肌肉记忆 / 最佳 SOP ]
      │ (自进化驱动)    │
      └────────────────┘
```

## 📖 深度指南

- [核心特性详解](./FEATURES.md) - 探索前沿创新。
- [架构深度解析](./ARCHITECTURE.md) - 深入了解可编程管线。
- [技能系统规范](./pkg/skill/README.md) - 构建自进化技能。

## 🛠️ 内置工具库
- **系统工具**: `Bash`, `Grep`, `Read`, `Write`, `Edit`, `Glob`, `LS`
- **通用工具**: `Calculator`, `DateTime`, `Email`, `Echo`

## 🤝 参与贡献
欢迎各类贡献！本项目采用 MIT 许可证。
